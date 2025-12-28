package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	v1 "pvesphere/api/v1"
	"pvesphere/internal/model"
	"pvesphere/internal/repository"
	"pvesphere/pkg/log"
	"pvesphere/pkg/proxmox"

	"go.uber.org/zap"
)

type DashboardService interface {
	GetScopes(ctx context.Context) (*v1.DashboardScopesData, error)
	GetOverview(ctx context.Context, req *v1.DashboardOverviewRequest) (*v1.DashboardOverviewData, error)
	GetResources(ctx context.Context, req *v1.DashboardResourcesRequest) (*v1.DashboardResourcesData, error)
	GetHotspots(ctx context.Context, req *v1.DashboardHotspotsRequest) (*v1.DashboardHotspotsData, error)
	GetOperations(ctx context.Context, req *v1.DashboardOperationsRequest) (*v1.DashboardOperationsData, error)
}

func NewDashboardService(
	service *Service,
	clusterRepo repository.PveClusterRepository,
	nodeRepo repository.PveNodeRepository,
	vmRepo repository.PveVMRepository,
	storageRepo repository.PveStorageRepository,
	logger *log.Logger,
) DashboardService {
	return &dashboardService{
		clusterRepo: clusterRepo,
		nodeRepo:    nodeRepo,
		vmRepo:      vmRepo,
		storageRepo: storageRepo,
		Service:     service,
		logger:      logger,
	}
}

type dashboardService struct {
	clusterRepo repository.PveClusterRepository
	nodeRepo    repository.PveNodeRepository
	vmRepo      repository.PveVMRepository
	storageRepo repository.PveStorageRepository
	*Service
	logger *log.Logger
}

// GetScopes 获取可选集群列表
func (s *dashboardService) GetScopes(ctx context.Context) (*v1.DashboardScopesData, error) {
	clusters, err := s.clusterRepo.List(ctx)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to list clusters", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}

	items := make([]v1.ScopeItem, 0, len(clusters))
	for _, cluster := range clusters {
		items = append(items, v1.ScopeItem{
			ClusterID:        cluster.Id,
			ClusterName:      cluster.ClusterName,
			ClusterNameAlias: cluster.ClusterNameAlias,
		})
	}

	return &v1.DashboardScopesData{
		Items: items,
	}, nil
}

// GetOverview 获取全局概览
func (s *dashboardService) GetOverview(ctx context.Context, req *v1.DashboardOverviewRequest) (*v1.DashboardOverviewData, error) {
	var clusters []*model.PveCluster
	var err error

	// 根据 scope 获取集群列表
	if req.Scope == "cluster" && req.ClusterID != nil {
		cluster, err := s.clusterRepo.GetByID(ctx, *req.ClusterID)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to get cluster", zap.Error(err))
			return nil, v1.ErrInternalServerError
		}
		if cluster == nil {
			return nil, v1.ErrNotFound
		}
		clusters = []*model.PveCluster{cluster}
	} else {
		clusters, err = s.clusterRepo.List(ctx)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to list clusters", zap.Error(err))
			return nil, v1.ErrInternalServerError
		}
	}

	// 统计概览数据
	summary := v1.DashboardOverviewSummary{
		ClusterCount: int64(len(clusters)),
	}

	// 健康状态统计
	health := v1.DashboardOverviewHealth{
		Healthy:  0,
		Warning:  0,
		Critical: 0,
	}

	// 遍历集群,统计节点、虚拟机、存储数量
	for _, cluster := range clusters {
		// 统计节点数量
		nodes, err := s.nodeRepo.GetByClusterID(ctx, cluster.Id)
		if err != nil {
			s.logger.WithContext(ctx).Warn("failed to get nodes for cluster",
				zap.Error(err), zap.Int64("cluster_id", cluster.Id))
			continue
		}
		summary.NodeCount += int64(len(nodes))

		// 统计虚拟机数量
		vms, err := s.vmRepo.GetByClusterID(ctx, cluster.Id)
		if err != nil {
			s.logger.WithContext(ctx).Warn("failed to get vms for cluster",
				zap.Error(err), zap.Int64("cluster_id", cluster.Id))
			continue
		}
		summary.VMCount += int64(len(vms))

		// 统计存储数量
		storages, err := s.storageRepo.GetByClusterID(ctx, cluster.Id)
		if err != nil {
			s.logger.WithContext(ctx).Warn("failed to get storages for cluster",
				zap.Error(err), zap.Int64("cluster_id", cluster.Id))
			continue
		}
		summary.StorageCount += int64(len(storages))

		// 评估集群健康状态（简单实现，可以后续扩展）
		clusterHealth := s.evaluateClusterHealth(ctx, cluster, nodes)
		switch clusterHealth {
		case "healthy":
			health.Healthy++
		case "warning":
			health.Warning++
		case "critical":
			health.Critical++
		}
	}

	return &v1.DashboardOverviewData{
		Scope:     req.Scope,
		ClusterID: req.ClusterID,
		Summary:   summary,
		Health:    health,
	}, nil
}

// evaluateClusterHealth 评估集群健康状态
func (s *dashboardService) evaluateClusterHealth(ctx context.Context, cluster *model.PveCluster, nodes []*model.PveNode) string {
	// 简单的健康评估逻辑
	// 1. 检查集群是否启用
	if cluster.IsEnabled != 1 {
		return "warning"
	}

	// 2. 检查节点状态
	offlineNodes := 0
	for _, node := range nodes {
		if node.Status != "online" {
			offlineNodes++
		}
	}

	if offlineNodes == len(nodes) && len(nodes) > 0 {
		return "critical"
	} else if offlineNodes > 0 {
		return "warning"
	}

	return "healthy"
}

// GetResources 获取资源使用率
func (s *dashboardService) GetResources(ctx context.Context, req *v1.DashboardResourcesRequest) (*v1.DashboardResourcesData, error) {
	var clusters []*model.PveCluster
	var err error

	// 根据 scope 获取集群列表
	if req.Scope == "cluster" && req.ClusterID != nil {
		cluster, err := s.clusterRepo.GetByID(ctx, *req.ClusterID)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to get cluster", zap.Error(err))
			return nil, v1.ErrInternalServerError
		}
		if cluster == nil {
			return nil, v1.ErrNotFound
		}
		clusters = []*model.PveCluster{cluster}
	} else {
		clusters, err = s.clusterRepo.List(ctx)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to list clusters", zap.Error(err))
			return nil, v1.ErrInternalServerError
		}
	}

	// 统计资源使用情况
	var totalCPUCores, usedCPUCores float64
	var totalMemory, usedMemory int64
	var totalStorage, usedStorage int64

	// 遍历集群,通过 Proxmox API 获取实时资源数据
	for _, cluster := range clusters {
		// 创建 Proxmox 客户端
		client, err := proxmox.NewProxmoxClient(cluster.ApiUrl, cluster.UserId, cluster.UserToken)
		if err != nil {
			s.logger.WithContext(ctx).Warn("failed to create proxmox client",
				zap.Error(err), zap.Int64("cluster_id", cluster.Id))
			continue
		}

		// 获取集群资源
		resources, err := client.GetClusterResources(ctx)
		if err != nil {
			s.logger.WithContext(ctx).Warn("failed to get cluster resources",
				zap.Error(err), zap.Int64("cluster_id", cluster.Id))
			continue
		}

		// 聚合节点资源
		for _, resource := range resources {
			resourceType, _ := resource["type"].(string)
			if resourceType != "node" {
				continue
			}

			// CPU
			if maxcpu, ok := resource["maxcpu"].(float64); ok {
				totalCPUCores += maxcpu
			}
			if cpu, ok := resource["cpu"].(float64); ok {
				if maxcpu, ok := resource["maxcpu"].(float64); ok {
					usedCPUCores += cpu * maxcpu
				}
			}

			// 内存
			if maxmem, ok := resource["maxmem"].(float64); ok {
				totalMemory += int64(maxmem)
			}
			if mem, ok := resource["mem"].(float64); ok {
				usedMemory += int64(mem)
			}

			// 存储
			if maxdisk, ok := resource["maxdisk"].(float64); ok {
				totalStorage += int64(maxdisk)
			}
			if disk, ok := resource["disk"].(float64); ok {
				usedStorage += int64(disk)
			}
		}
	}

	// 计算使用率
	cpuUsagePercent := 0.0
	if totalCPUCores > 0 {
		cpuUsagePercent = (usedCPUCores / totalCPUCores) * 100
	}

	memoryUsagePercent := 0.0
	if totalMemory > 0 {
		memoryUsagePercent = (float64(usedMemory) / float64(totalMemory)) * 100
	}

	storageUsagePercent := 0.0
	if totalStorage > 0 {
		storageUsagePercent = (float64(usedStorage) / float64(totalStorage)) * 100
	}

	return &v1.DashboardResourcesData{
		Scope:     req.Scope,
		ClusterID: req.ClusterID,
		CPU: v1.ResourceUsage{
			UsedCores:    &usedCPUCores,
			TotalCores:   &totalCPUCores,
			UsagePercent: cpuUsagePercent,
		},
		Memory: v1.ResourceUsage{
			UsedBytes:    &usedMemory,
			TotalBytes:   &totalMemory,
			UsagePercent: memoryUsagePercent,
		},
		Storage: v1.ResourceUsage{
			UsedBytes:    &usedStorage,
			TotalBytes:   &totalStorage,
			UsagePercent: storageUsagePercent,
		},
	}, nil
}

// GetHotspots 获取压力和风险焦点
func (s *dashboardService) GetHotspots(ctx context.Context, req *v1.DashboardHotspotsRequest) (*v1.DashboardHotspotsData, error) {
	var clusters []*model.PveCluster
	var err error

	// 设置默认 limit
	if req.Limit <= 0 {
		req.Limit = 5
	}

	// 根据 scope 获取集群列表
	if req.Scope == "cluster" && req.ClusterID != nil {
		cluster, err := s.clusterRepo.GetByID(ctx, *req.ClusterID)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to get cluster", zap.Error(err))
			return nil, v1.ErrInternalServerError
		}
		if cluster == nil {
			return nil, v1.ErrNotFound
		}
		clusters = []*model.PveCluster{cluster}
	} else {
		clusters, err = s.clusterRepo.List(ctx)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to list clusters", zap.Error(err))
			return nil, v1.ErrInternalServerError
		}
	}

	// 分别收集各类资源的使用率
	type vmResource struct {
		ID          string
		Name        string
		NodeName    string
		ClusterID   int64
		ClusterName string
		MetricValue float64
		Unit        string
	}

	type nodeResource struct {
		ID          string
		Name        string
		ClusterID   int64
		ClusterName string
		MetricValue float64
		Unit        string
	}

	type storageResource struct {
		ID           string
		Name         string
		UsagePercent float64
		UsedBytes    int64
		TotalBytes   int64
		Unit         string
	}

	// 分别收集 VM 的 CPU、Memory
	var vmCPU []vmResource
	var vmMemory []vmResource

	// 分别收集 Node 的 CPU、Memory
	var nodeCPU []nodeResource
	var nodeMemory []nodeResource

	// 收集 Storage
	var storages []storageResource

	// 遍历集群,获取资源消耗数据
	for _, cluster := range clusters {
		// 创建 Proxmox 客户端
		client, err := proxmox.NewProxmoxClient(cluster.ApiUrl, cluster.UserId, cluster.UserToken)
		if err != nil {
			s.logger.WithContext(ctx).Warn("failed to create proxmox client",
				zap.Error(err), zap.Int64("cluster_id", cluster.Id))
			continue
		}

		// 1. 获取节点资源（CPU 和 Memory）
		// 使用 GetClusterResources 获取节点数据，因为 GetNodeStatus 可能不包含完整的内存信息
		nodeResources, err := client.GetClusterResources(ctx)
		if err != nil {
			s.logger.WithContext(ctx).Warn("failed to get cluster resources",
				zap.Error(err), zap.Int64("cluster_id", cluster.Id))
		} else {
			// 从数据库获取节点列表，用于匹配和填充集群信息
			nodes, err := s.nodeRepo.GetByClusterID(ctx, cluster.Id)
			if err != nil {
				s.logger.WithContext(ctx).Warn("failed to get nodes from database",
					zap.Error(err), zap.Int64("cluster_id", cluster.Id))
			}

			// 创建节点名称映射，用于快速查找
			nodeMap := make(map[string]*model.PveNode)
			for _, node := range nodes {
				nodeMap[node.NodeName] = node
			}

			// 遍历资源，查找节点
			for _, resource := range nodeResources {
				resourceType, _ := resource["type"].(string)
				if resourceType != "node" {
					continue
				}

				nodeName, _ := resource["name"].(string)
				if nodeName == "" {
					// 如果没有 name 字段，尝试从 id 字段获取（格式可能是 "node/pve01"）
					id, _ := resource["id"].(string)
					if strings.HasPrefix(id, "node/") {
						nodeName = strings.TrimPrefix(id, "node/")
					}
				}

				if nodeName == "" {
					continue
				}

				// 节点 CPU 使用率
				if cpu, ok := resource["cpu"].(float64); ok {
					cpuPercent := cpu * 100
					nodeCPU = append(nodeCPU, nodeResource{
						ID:          fmt.Sprintf("node/%s", nodeName),
						Name:        nodeName,
						ClusterID:   cluster.Id,
						ClusterName: cluster.ClusterName,
						MetricValue: cpuPercent,
						Unit:        "%",
					})
				}

				// 节点内存使用率
				// GetClusterResources 返回的字段是 mem 和 maxmem
				var mem, maxmem float64

				// 尝试多种类型转换
				if memVal, ok := resource["mem"].(float64); ok {
					mem = memVal
				} else if memVal, ok := resource["mem"].(int64); ok {
					mem = float64(memVal)
				} else if memVal, ok := resource["mem"].(int); ok {
					mem = float64(memVal)
				}

				if maxmemVal, ok := resource["maxmem"].(float64); ok {
					maxmem = maxmemVal
				} else if maxmemVal, ok := resource["maxmem"].(int64); ok {
					maxmem = float64(maxmemVal)
				} else if maxmemVal, ok := resource["maxmem"].(int); ok {
					maxmem = float64(maxmemVal)
				}

				// 只有当两个值都有效时才计算使用率
				if mem >= 0 && maxmem > 0 {
					memPercent := (mem / maxmem) * 100
					nodeMemory = append(nodeMemory, nodeResource{
						ID:          fmt.Sprintf("node/%s", nodeName),
						Name:        nodeName,
						ClusterID:   cluster.Id,
						ClusterName: cluster.ClusterName,
						MetricValue: memPercent,
						Unit:        "%",
					})
				} else {
					// 添加调试日志，帮助排查问题
					s.logger.WithContext(ctx).Debug("failed to get node memory data from cluster resources",
						zap.String("node", nodeName),
						zap.Any("resource", resource),
						zap.Float64("mem", mem),
						zap.Float64("maxmem", maxmem))
				}
			}
		}

		// 2. 获取存储资源
		dbStorages, err := s.storageRepo.GetByClusterID(ctx, cluster.Id)
		if err != nil {
			s.logger.WithContext(ctx).Warn("failed to get storages from database",
				zap.Error(err), zap.Int64("cluster_id", cluster.Id))
		} else {
			resources, err := client.GetClusterResources(ctx)
			if err == nil {
				// 创建存储映射，用于快速查找
				storageMap := make(map[string]*model.PveStorage)
				for _, storage := range dbStorages {
					key := fmt.Sprintf("%s:%s", storage.NodeName, storage.StorageName)
					storageMap[key] = storage
				}

				// 遍历资源，查找存储
				for _, resource := range resources {
					resourceType, _ := resource["type"].(string)
					if resourceType != "storage" {
						continue
					}

					storageName, _ := resource["storage"].(string)
					nodeName, _ := resource["node"].(string)
					if storageName == "" || nodeName == "" {
						continue
					}

					// 检查是否在数据库中存在
					key := fmt.Sprintf("%s:%s", nodeName, storageName)
					if _, exists := storageMap[key]; !exists {
						continue
					}

					// 存储使用率
					if disk, ok := resource["disk"].(float64); ok {
						if maxdisk, ok := resource["maxdisk"].(float64); ok && maxdisk > 0 {
							diskPercent := (disk / maxdisk) * 100
							storages = append(storages, storageResource{
								ID:           fmt.Sprintf("storage/%s/%s", nodeName, storageName),
								Name:         fmt.Sprintf("%s (%s)", storageName, nodeName),
								UsagePercent: diskPercent,
								UsedBytes:    int64(disk),
								TotalBytes:   int64(maxdisk),
								Unit:         "%",
							})
						}
					}
				}
			}
		}

		// 3. 获取 VM 资源（CPU、Memory）
		resources, err := client.GetClusterResources(ctx)
		if err != nil {
			s.logger.WithContext(ctx).Warn("failed to get cluster resources",
				zap.Error(err), zap.Int64("cluster_id", cluster.Id))
			continue
		}

		for _, resource := range resources {
			resourceType, _ := resource["type"].(string)
			if resourceType != "qemu" && resourceType != "lxc" {
				continue
			}

			name, _ := resource["name"].(string)
			id, _ := resource["id"].(string)
			nodeName, _ := resource["node"].(string)

			// VM CPU 使用率
			if cpu, ok := resource["cpu"].(float64); ok {
				cpuPercent := cpu * 100
				vmCPU = append(vmCPU, vmResource{
					ID:          id,
					Name:        name,
					NodeName:    nodeName,
					ClusterID:   cluster.Id,
					ClusterName: cluster.ClusterName,
					MetricValue: cpuPercent,
					Unit:        "%",
				})
			}

			// VM Memory 使用率
			if mem, ok := resource["mem"].(float64); ok {
				if maxmem, ok := resource["maxmem"].(float64); ok && maxmem > 0 {
					memPercent := (mem / maxmem) * 100
					vmMemory = append(vmMemory, vmResource{
						ID:          id,
						Name:        name,
						NodeName:    nodeName,
						ClusterID:   cluster.Id,
						ClusterName: cluster.ClusterName,
						MetricValue: memPercent,
						Unit:        "%",
					})
				}
			}
		}
	}

	// 分别排序并取 Top N
	topN := req.Limit

	// VM CPU Top N
	sort.Slice(vmCPU, func(i, j int) bool {
		return vmCPU[i].MetricValue > vmCPU[j].MetricValue
	})
	vmCPUTopN := make([]v1.TopResourceConsumer, 0, topN)
	for i := 0; i < topN && i < len(vmCPU); i++ {
		vmCPUTopN = append(vmCPUTopN, v1.TopResourceConsumer{
			ID:          vmCPU[i].ID,
			Name:        vmCPU[i].Name,
			MetricValue: vmCPU[i].MetricValue,
			Unit:        vmCPU[i].Unit,
			NodeName:    vmCPU[i].NodeName,
			ClusterID:   vmCPU[i].ClusterID,
			ClusterName: vmCPU[i].ClusterName,
		})
	}

	// VM Memory Top N
	sort.Slice(vmMemory, func(i, j int) bool {
		return vmMemory[i].MetricValue > vmMemory[j].MetricValue
	})
	vmMemoryTopN := make([]v1.TopResourceConsumer, 0, topN)
	for i := 0; i < topN && i < len(vmMemory); i++ {
		vmMemoryTopN = append(vmMemoryTopN, v1.TopResourceConsumer{
			ID:          vmMemory[i].ID,
			Name:        vmMemory[i].Name,
			MetricValue: vmMemory[i].MetricValue,
			Unit:        vmMemory[i].Unit,
			NodeName:    vmMemory[i].NodeName,
			ClusterID:   vmMemory[i].ClusterID,
			ClusterName: vmMemory[i].ClusterName,
		})
	}

	// Node CPU Top N
	sort.Slice(nodeCPU, func(i, j int) bool {
		return nodeCPU[i].MetricValue > nodeCPU[j].MetricValue
	})
	nodeCPUTopN := make([]v1.TopResourceConsumer, 0, topN)
	for i := 0; i < topN && i < len(nodeCPU); i++ {
		nodeCPUTopN = append(nodeCPUTopN, v1.TopResourceConsumer{
			ID:          nodeCPU[i].ID,
			Name:        nodeCPU[i].Name,
			MetricValue: nodeCPU[i].MetricValue,
			Unit:        nodeCPU[i].Unit,
			ClusterID:   nodeCPU[i].ClusterID,
			ClusterName: nodeCPU[i].ClusterName,
		})
	}

	// Node Memory Top N
	sort.Slice(nodeMemory, func(i, j int) bool {
		return nodeMemory[i].MetricValue > nodeMemory[j].MetricValue
	})
	nodeMemoryTopN := make([]v1.TopResourceConsumer, 0, topN)
	for i := 0; i < topN && i < len(nodeMemory); i++ {
		nodeMemoryTopN = append(nodeMemoryTopN, v1.TopResourceConsumer{
			ID:          nodeMemory[i].ID,
			Name:        nodeMemory[i].Name,
			MetricValue: nodeMemory[i].MetricValue,
			Unit:        nodeMemory[i].Unit,
			ClusterID:   nodeMemory[i].ClusterID,
			ClusterName: nodeMemory[i].ClusterName,
		})
	}

	// Storage Top N
	sort.Slice(storages, func(i, j int) bool {
		return storages[i].UsagePercent > storages[j].UsagePercent
	})
	storageTopN := make([]v1.StorageHotspot, 0, topN)
	for i := 0; i < topN && i < len(storages); i++ {
		storageTopN = append(storageTopN, v1.StorageHotspot{
			ID:           storages[i].ID,
			Name:         storages[i].Name,
			UsagePercent: storages[i].UsagePercent,
			UsedBytes:    storages[i].UsedBytes,
			TotalBytes:   storages[i].TotalBytes,
			Unit:         storages[i].Unit,
		})
	}

	// 获取最近的风险
	recentRisks := s.getRecentRisks(ctx, clusters)

	return &v1.DashboardHotspotsData{
		Scope:     req.Scope,
		ClusterID: req.ClusterID,
		VMHotspots: v1.VMHotspots{
			CPU:    vmCPUTopN,
			Memory: vmMemoryTopN,
		},
		NodeHotspots: v1.NodeHotspots{
			CPU:    nodeCPUTopN,
			Memory: nodeMemoryTopN,
		},
		StorageHotspots: storageTopN,
		RecentRisks:     recentRisks,
	}, nil
}

// getRecentRisks 获取最近的风险（简单实现，实际应该从监控系统获取）
func (s *dashboardService) getRecentRisks(ctx context.Context, clusters []*model.PveCluster) []v1.RecentRisk {
	risks := make([]v1.RecentRisk, 0)

	// 遍历集群，检查节点状态
	for _, cluster := range clusters {
		nodes, err := s.nodeRepo.GetByClusterID(ctx, cluster.Id)
		if err != nil {
			continue
		}

		for _, node := range nodes {
			// 检查节点是否离线
			if node.Status != "online" {
				risks = append(risks, v1.RecentRisk{
					ID:           fmt.Sprintf("risk-node-%d", node.Id),
					Level:        "warning",
					Message:      fmt.Sprintf("Node %s is %s", node.NodeName, node.Status),
					OccurredAt:   node.UpdateTime.Format(time.RFC3339),
					RelativeTime: s.getRelativeTime(node.UpdateTime),
					TargetType:   "node",
					TargetID:     fmt.Sprintf("node-%d", node.Id),
					TargetName:   node.NodeName,
				})
			}
		}

		// 检查存储空间
		_, err = s.storageRepo.GetByClusterID(ctx, cluster.Id)
		if err != nil {
			continue
		}

		// 这里需要从 Proxmox API 获取实时存储数据
		// 简化处理：如果有存储相关的字段可以检查
		// 实际应该调用 API 获取实时数据
	}

	return risks
}

// getRelativeTime 获取相对时间
func (s *dashboardService) getRelativeTime(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return fmt.Sprintf("%d sec ago", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%d min ago", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%d h ago", int(duration.Hours()))
	} else {
		return fmt.Sprintf("%d d ago", int(duration.Hours()/24))
	}
}

// GetOperations 获取运行中的操作
func (s *dashboardService) GetOperations(ctx context.Context, req *v1.DashboardOperationsRequest) (*v1.DashboardOperationsData, error) {
	var clusters []*model.PveCluster
	var err error

	// 根据 scope 获取集群列表
	if req.Scope == "cluster" && req.ClusterID != nil {
		cluster, err := s.clusterRepo.GetByID(ctx, *req.ClusterID)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to get cluster", zap.Error(err))
			return nil, v1.ErrInternalServerError
		}
		if cluster == nil {
			return nil, v1.ErrNotFound
		}
		clusters = []*model.PveCluster{cluster}
	} else {
		clusters, err = s.clusterRepo.List(ctx)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to list clusters", zap.Error(err))
			return nil, v1.ErrInternalServerError
		}
	}

	// 统计各类操作
	operationCounts := make(map[string]int64)
	var allItems []v1.OperationItem

	// 遍历集群，获取正在运行的任务
	for _, cluster := range clusters {
		// 创建 Proxmox 客户端
		client, err := proxmox.NewProxmoxClient(cluster.ApiUrl, cluster.UserId, cluster.UserToken)
		if err != nil {
			s.logger.WithContext(ctx).Warn("failed to create proxmox client",
				zap.Error(err), zap.Int64("cluster_id", cluster.Id))
			continue
		}

		// 获取集群任务
		tasks, err := client.GetClusterTasks(ctx)
		if err != nil {
			s.logger.WithContext(ctx).Warn("failed to get cluster tasks",
				zap.Error(err), zap.Int64("cluster_id", cluster.Id))
			continue
		}

		// 遍历任务
		for _, task := range tasks {
			status, _ := task["status"].(string)
			taskType, _ := task["type"].(string)
			upid, _ := task["upid"].(string)

			// 只统计运行中的任务
			if status != "running" {
				continue
			}

			// 分类统计
			operationType := s.classifyOperationType(taskType)
			operationCounts[operationType]++

			// 收集任务详情
			startTime, _ := task["starttime"].(float64)
			startTimeStr := time.Unix(int64(startTime), 0).Format(time.RFC3339)

			allItems = append(allItems, v1.OperationItem{
				ID:            upid,
				OperationType: operationType,
				Name:          fmt.Sprintf("%s on %s", taskType, task["node"]),
				Progress:      0, // Proxmox API 可能不提供进度信息
				Status:        "running",
				StartedAt:     startTimeStr,
			})
		}
	}

	// 构建摘要
	summary := make([]v1.OperationSummary, 0)
	if count, ok := operationCounts["vm_migration"]; ok && count > 0 {
		summary = append(summary, v1.OperationSummary{
			OperationType: "vm_migration",
			DisplayName:   "VM Migrations",
			Count:         count,
		})
	}
	if count, ok := operationCounts["node_maintenance"]; ok && count > 0 {
		summary = append(summary, v1.OperationSummary{
			OperationType: "node_maintenance",
			DisplayName:   "Node Maintenance",
			Count:         count,
		})
	}
	if count, ok := operationCounts["storage_rebalance"]; ok && count > 0 {
		summary = append(summary, v1.OperationSummary{
			OperationType: "storage_rebalance",
			DisplayName:   "Storage Rebalance",
			Count:         count,
		})
	}
	if count, ok := operationCounts["other"]; ok && count > 0 {
		summary = append(summary, v1.OperationSummary{
			OperationType: "other",
			DisplayName:   "Other Operations",
			Count:         count,
		})
	}

	return &v1.DashboardOperationsData{
		Scope:     req.Scope,
		ClusterID: req.ClusterID,
		Summary:   summary,
		Items:     allItems,
	}, nil
}

// classifyOperationType 分类操作类型
func (s *dashboardService) classifyOperationType(taskType string) string {
	switch taskType {
	case "qmigrate", "vzmigrate":
		return "vm_migration"
	case "startall", "stopall":
		return "node_maintenance"
	case "storage":
		return "storage_rebalance"
	default:
		return "other"
	}
}
