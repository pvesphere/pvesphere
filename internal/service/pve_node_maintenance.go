package service

import (
	"context"
	"fmt"
	"pvesphere/api/v1"
	"pvesphere/internal/model"
	"pvesphere/pkg/proxmox"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"

	"gorm.io/gorm"
)

// NodeMaintenanceService 主机维护服务
type NodeMaintenanceService struct {
	db *gorm.DB
}

// NewNodeMaintenanceService 创建主机维护服务
func NewNodeMaintenanceService(db *gorm.DB) *NodeMaintenanceService {
	return &NodeMaintenanceService{
		db: db,
	}
}

// GetMigrateCandidates 获取可迁移的目标节点列表
func (s *NodeMaintenanceService) GetMigrateCandidates(ctx context.Context, nodeID int64) ([]v1.MigrateCandidate, error) {
	// 获取节点信息
	var node model.PveNode
	if err := s.db.First(&node, nodeID).Error; err != nil {
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}

	// 获取同集群的其他节点
	var nodes []model.PveNode
	if err := s.db.Where("cluster_id = ? AND id != ? AND status = 'online' AND is_schedulable = 1", node.ClusterID, nodeID).Find(&nodes).Error; err != nil {
		return nil, fmt.Errorf("cluster not found: %w", err)
	}

	// 获取集群信息用于创建Proxmox客户端
	var cluster model.PveCluster
	if err := s.db.First(&cluster, node.ClusterID).Error; err != nil {
		return nil, fmt.Errorf("cluster not found: %w", err)
	}

	// 创建Proxmox客户端
	pClient, err := proxmox.NewProxmoxClient(cluster.ApiUrl, cluster.UserId, cluster.UserToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Proxmox client: %w", err)
	}

	candidates := make([]v1.MigrateCandidate, 0, len(nodes))
	for _, n := range nodes {
		// 获取节点状态
		status, err := pClient.GetNodeStatus(ctx, n.NodeName)
		if err != nil {
			continue
		}

		// 提取 CPU 核心数
		cpuCores := 0
		if cpuInfo, ok := status["cpuinfo"].(map[string]interface{}); ok {
			if cores, ok := cpuInfo["cores"].(float64); ok {
				cpuCores = int(cores)
			}
		}

		// 获取节点上的VM数量
		vms, err := pClient.GetNodeVMs(ctx, n.NodeName)
		vmCount := 0
		if err == nil && vms != nil {
			vmCount = len(vms)
		}

		// 计算资源评分
		cpu := float64(0)
		memTotal := int64(0)
		memUsed := int64(0)

		if cpuInfo, ok := status["cpu"].(float64); ok {
			cpu = cpuInfo * 100
		}
		if memInfo, ok := status["memory"].(map[string]interface{}); ok {
			if total, ok := memInfo["total"].(float64); ok {
				memTotal = int64(total)
			}
			if used, ok := memInfo["used"].(float64); ok {
				memUsed = int64(used)
			}
		}

		memUsage := float64(0)
		if memTotal > 0 {
			memUsage = float64(memUsed) / float64(memTotal) * 100
		}

		// 计算调度评分: 资源使用率越低分数越高
		score := 100 - (cpu+memUsage)/2

		candidates = append(candidates, v1.MigrateCandidate{
			NodeID:    n.Id,
			NodeName:  n.NodeName,
			CPUCores:  cpuCores,
			CPUUsage:  cpu,
			MemTotal:  memTotal,
			MemUsed:   memUsed,
			MemFree:   memTotal - memUsed,
			MemUsage:  memUsage,
			VMCount:   vmCount,
			Score:     score,
		})
	}

	// 按评分排序，优先选择资源最空闲的节点
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	return candidates, nil
}

// GetNodeVMs 获取节点上的VM列表 (从数据库查询)
func (s *NodeMaintenanceService) GetNodeVMs(ctx context.Context, nodeID int64) ([]v1.NodeVMItem, error) {
	var node model.PveNode
	if err := s.db.First(&node, nodeID).Error; err != nil {
		return nil, fmt.Errorf("failed to get node VMs: %w", err)
	}

	// 从数据库查询该节点上的VM
	var vms []model.PveVM
	if err := s.db.Where("node_id = ?", nodeID).Find(&vms).Error; err != nil {
		return nil, fmt.Errorf("failed to query VMs: %w", err)
	}

	items := make([]v1.NodeVMItem, 0, len(vms))
	for _, vm := range vms {
		items = append(items, v1.NodeVMItem{
			VMID:     int64(vm.VMID),
			VMName:   vm.VmName,
			Status:   vm.Status,
			NodeID:   node.Id,
			NodeName: node.NodeName,
		})
	}

	return items, nil
}

// StartMaintenance 开始主机维护
func (s *NodeMaintenanceService) StartMaintenance(ctx context.Context, req v1.StartMaintenanceRequest) (*v1.NodeMaintenanceResult, error) {
	nodeID, err := strconv.ParseInt(req.NodeID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse node ID: %w", err)
	}

	// 获取节点信息
	var node model.PveNode
	if err := s.db.First(&node, nodeID).Error; err != nil {
		return nil, fmt.Errorf("node not found: %w", err)
	}

	// 获取节点上的VM列表
	vms, err := s.GetNodeVMs(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get node VMs: %w", err)
	}

	// 获取目标节点列表
	var targetNodes []model.PveNode
	if req.Mode == "manual" && len(req.TargetNodes) > 0 {
		s.db.Where("id IN ?", req.TargetNodes).Find(&targetNodes)
	} else {
		// 自动选择
		candidates, err := s.GetMigrateCandidates(ctx, nodeID)
		if err != nil {
			return nil, fmt.Errorf("failed to create Proxmox client: %w", err)
		}
		if len(candidates) == 0 {
			return nil, fmt.Errorf("无可用的目标节点")
		}
		// 选择最优的节点
		targetNodeIDs := make([]int64, 0, len(candidates))
		for _, c := range candidates {
			targetNodeIDs = append(targetNodeIDs, c.NodeID)
		}
		s.db.Where("id IN ?", targetNodeIDs).Find(&targetNodes)
	}

	// 边界检查：目标节点不能为空
	if len(targetNodes) == 0 {
		return nil, fmt.Errorf("no target nodes available")
	}

	// 设置节点为不可调度
	s.db.Model(&model.PveNode{}).Where("id = ?", nodeID).Update("is_schedulable", 0)

	// 并发迁移
	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = 3
	}
	// 并发数不能超过 VM 数量
	if concurrency > len(vms) {
		concurrency = len(vms)
	}

	result := &v1.NodeMaintenanceResult{
		NodeID:      req.NodeID,
		Status:      "migrating",
		TotalVMs:    len(vms),
		MigratedVMs: 0,
		FailedVMs:   0,
		Tasks:       make([]v1.MaintenanceTask, 0, len(vms)),
	}

	if len(vms) == 0 {
		result.Status = "completed"
		return result, nil
	}

	// 获取集群信息
	var cluster model.PveCluster
	if err := s.db.First(&cluster, node.ClusterID).Error; err != nil {
		return nil, fmt.Errorf("cluster not found: %w", err)
	}

	// 创建Proxmox客户端
	pClient, err := proxmox.NewProxmoxClient(cluster.ApiUrl, cluster.UserId, cluster.UserToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Proxmox client: %w", err)
	}

	// 并发控制 - 使用信号量模式
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	taskResults := make([]v1.MaintenanceTask, 0, len(vms))
	var mu sync.Mutex
	var failedCount int32
	var successCount int32

	// 使用 WaitGroup 并发处理每个 VM
	wg.Add(len(vms))

	for i, vm := range vms {
		go func(idx int, vm v1.NodeVMItem) {
			defer wg.Done()

			// 获取信号量
			sem <- struct{}{}
			defer func() { <-sem }()

			// 选择目标节点 (使用 VM 索引实现 round-robin)
			targetNode := targetNodes[idx%len(targetNodes)]

			task := v1.MaintenanceTask{
				VMID:       strconv.FormatUint(uint64(vm.VMID), 10),
				VMName:     vm.VMName,
				TargetNode: targetNode.NodeName,
				Status:     "migrating",
			}

			// 执行迁移
			vmIDUint, _ := strconv.ParseUint(task.VMID, 10, 32)
			vmID := uint32(vmIDUint)
			_, err := pClient.MigrateVM(ctx, targetNode.NodeName, vmID, map[string]interface{}{
				"target": targetNode.NodeName,
				"online": true,
			})

			if err != nil {
				task.Status = "failed"
				task.Error = err.Error()
				atomic.AddInt32(&failedCount, 1)
			} else {
				task.Status = "completed"
				// 迁移成功后更新数据库中的 node_id
				if err := s.db.Model(&model.PveVM{}).Where("vmid = ?", vm.VMID).Update("node_id", targetNode.Id).Error; err != nil {
					task.Status = "failed"
					task.Error = fmt.Sprintf("migration succeeded but failed to update database: %v", err)
					atomic.AddInt32(&failedCount, 1)
				} else {
					atomic.AddInt32(&successCount, 1)
				}
			}

			mu.Lock()
			taskResults = append(taskResults, task)
			mu.Unlock()
		}(i, vm)
	}

	wg.Wait()

	// 更新结果
	for _, t := range taskResults {
		result.Tasks = append(result.Tasks, t)
	}

	result.MigratedVMs = int(atomic.LoadInt32(&successCount))
	result.FailedVMs = int(atomic.LoadInt32(&failedCount))

	if result.MigratedVMs == result.TotalVMs {
		result.Status = "completed"
	} else if result.FailedVMs > 0 {
		result.Status = "failed"
	} else {
		result.Status = "migrating"
	}

	return result, nil
}

// EndMaintenance 结束维护
func (s *NodeMaintenanceService) EndMaintenance(ctx context.Context, nodeID string) error {
	id, err := strconv.ParseInt(nodeID, 10, 64)
	if err != nil {
		return err
	}

	// 恢复节点调度
	return s.db.Model(&model.PveNode{}).Where("id = ?", id).Update("is_schedulable", 1).Error
}
