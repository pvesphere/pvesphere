package service

import (
	"context"
	"fmt"
	"pvesphere/api/v1"
	"pvesphere/internal/model"
	"pvesphere/internal/repository"
	"pvesphere/pkg/proxmox"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// NodeMaintenanceService 主机维护服务
type NodeMaintenanceService struct {
	repo *repository.Repository
}

// NewNodeMaintenanceService 创建主机维护服务
func NewNodeMaintenanceService(repo *repository.Repository) *NodeMaintenanceService {
	return &NodeMaintenanceService{
		repo: repo,
	}
}

// GetMigrateCandidates 获取可迁移的目标节点列表
func (s *NodeMaintenanceService) GetMigrateCandidates(ctx context.Context, nodeID int64) ([]v1.MigrateCandidate, error) {
	// 获取节点信息
	var node model.PveNode
	if err := s.repo.GormDB.First(&node, nodeID).Error; err != nil {
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}

	// 获取同集群的其他节点
	var nodes []model.PveNode
	if err := s.repo.GormDB.Where("cluster_id = ? AND id != ? AND status = 'online' AND is_schedulable = 1", node.ClusterID, nodeID).Find(&nodes).Error; err != nil {
		return nil, fmt.Errorf("cluster not found: %w", err)
	}

	// 获取集群信息用于创建 Proxmox 客户端
	var cluster model.PveCluster
	if err := s.repo.GormDB.First(&cluster, node.ClusterID).Error; err != nil {
		return nil, fmt.Errorf("cluster not found: %w", err)
	}

	// 创建 Proxmox 客户端
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

		// 获取节点上的 VM 数量
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

		// 计算调度评分：资源使用率越低分数越高
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

// GetNodeVMs 获取节点上的 VM 列表 (从数据库查询)
func (s *NodeMaintenanceService) GetNodeVMs(ctx context.Context, nodeID int64) ([]v1.NodeVMItem, error) {
	var node model.PveNode
	if err := s.repo.GormDB.First(&node, nodeID).Error; err != nil {
		return nil, fmt.Errorf("failed to get node VMs: %w", err)
	}

	// 从数据库查询该节点上的 VM
	var vms []model.PveVM
	if err := s.repo.GormDB.Where("node_id = ?", nodeID).Find(&vms).Error; err != nil {
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

// CreateMaintenancePlan 创建维护计划（异步执行）
func (s *NodeMaintenanceService) CreateMaintenancePlan(ctx context.Context, req v1.CreateMaintenancePlanRequest) (*model.NodeMaintenancePlan, error) {
	nodeID, err := strconv.ParseInt(req.NodeID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse node ID: %w", err)
	}

	// 获取节点信息
	var node model.PveNode
	if err := s.repo.GormDB.First(&node, nodeID).Error; err != nil {
		return nil, fmt.Errorf("node not found: %w", err)
	}

	// 获取节点上的 VM 列表
	vms, err := s.GetNodeVMs(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get node VMs: %w", err)
	}

	// 创建维护计划记录
	plan := &model.NodeMaintenancePlan{
		NodeID:      nodeID,
		NodeName:    node.NodeName,
		Status:      "pending",
		Mode:        req.Mode,
		Concurrency: req.Concurrency,
		TotalVMs:    len(vms),
	}

	if err := s.repo.GormDB.Create(plan).Error; err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	// 异步执行迁移任务（使用独立的 context，不受 HTTP 请求影响）
	go func() {
		bgCtx := context.Background()
		s.executeMaintenancePlan(bgCtx, plan.Id, req)
	}()

	return plan, nil
}

// executeMaintenancePlan 执行维护计划（后台 goroutine）
func (s *NodeMaintenanceService) executeMaintenancePlan(ctx context.Context, planID int64, req v1.CreateMaintenancePlanRequest) {
	// 更新计划状态为运行中
	s.repo.GormDB.Model(&model.NodeMaintenancePlan{}).Where("id = ?", planID).Updates(map[string]interface{}{
		"status":     "running",
		"start_time": time.Now(),
	})

	nodeID, _ := strconv.ParseInt(req.NodeID, 10, 64)

	// 获取节点信息
	var node model.PveNode
	if err := s.repo.GormDB.First(&node, nodeID).Error; err != nil {
		s.repo.GormDB.Model(&model.NodeMaintenancePlan{}).Where("id = ?", planID).Updates(map[string]interface{}{
			"status":        "failed",
			"error_message": fmt.Sprintf("node not found: %v", err),
			"end_time":      time.Now(),
		})
		return
	}

	// 获取节点上的 VM 列表
	vms, err := s.GetNodeVMs(ctx, nodeID)
	if err != nil {
		s.repo.GormDB.Model(&model.NodeMaintenancePlan{}).Where("id = ?", planID).Updates(map[string]interface{}{
			"status":        "failed",
			"error_message": fmt.Sprintf("failed to get VMs: %v", err),
			"end_time":      time.Now(),
		})
		return
	}

	// 如果没有 VM，直接标记为完成
	if len(vms) == 0 {
		s.repo.GormDB.Model(&model.NodeMaintenancePlan{}).Where("id = ?", planID).Updates(map[string]interface{}{
			"status":       "completed",
			"start_time":   time.Now(),
			"end_time":     time.Now(),
			"migrated_vms": 0,
			"failed_vms":   0,
		})
		// 设置节点为不可调度
		s.repo.GormDB.Model(&model.PveNode{}).Where("id = ?", nodeID).Update("is_schedulable", 0)
		return
	}

	// 获取目标节点列表
	var targetNodes []model.PveNode
	if req.Mode == "manual" && len(req.TargetNodes) > 0 {
		s.repo.GormDB.Where("id IN ?", req.TargetNodes).Find(&targetNodes)
	} else {
		// 自动选择
		candidates, err := s.GetMigrateCandidates(ctx, nodeID)
		if err != nil {
			s.repo.GormDB.Model(&model.NodeMaintenancePlan{}).Where("id = ?", planID).Updates(map[string]interface{}{
				"status":        "failed",
				"error_message": fmt.Sprintf("failed to get candidates: %v", err),
				"end_time":      time.Now(),
			})
			return
		}
		if len(candidates) == 0 {
			s.repo.GormDB.Model(&model.NodeMaintenancePlan{}).Where("id = ?", planID).Updates(map[string]interface{}{
				"status":        "failed",
				"error_message": "no target nodes available",
				"end_time":      time.Now(),
			})
			return
		}
		targetNodeIDs := make([]int64, 0, len(candidates))
		for _, c := range candidates {
			targetNodeIDs = append(targetNodeIDs, c.NodeID)
		}
		s.repo.GormDB.Where("id IN ?", targetNodeIDs).Find(&targetNodes)
	}

	if len(targetNodes) == 0 {
		s.repo.GormDB.Model(&model.NodeMaintenancePlan{}).Where("id = ?", planID).Updates(map[string]interface{}{
			"status":        "failed",
			"error_message": "no target nodes available",
			"end_time":      time.Now(),
		})
		return
	}

	// 设置节点为不可调度
	s.repo.GormDB.Model(&model.PveNode{}).Where("id = ?", nodeID).Update("is_schedulable", 0)

	// 获取集群信息
	var cluster model.PveCluster
	if err := s.repo.GormDB.First(&cluster, node.ClusterID).Error; err != nil {
		s.repo.GormDB.Model(&model.NodeMaintenancePlan{}).Where("id = ?", planID).Updates(map[string]interface{}{
			"status":        "failed",
			"error_message": fmt.Sprintf("cluster not found: %v", err),
			"end_time":      time.Now(),
		})
		return
	}

	// 创建 Proxmox 客户端
	pClient, err := proxmox.NewProxmoxClient(cluster.ApiUrl, cluster.UserId, cluster.UserToken)
	if err != nil {
		s.repo.GormDB.Model(&model.NodeMaintenancePlan{}).Where("id = ?", planID).Updates(map[string]interface{}{
			"status":        "failed",
			"error_message": fmt.Sprintf("failed to create Proxmox client: %v", err),
			"end_time":      time.Now(),
		})
		return
	}

	// 并发控制
	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = 3
	}
	if concurrency > len(vms) {
		concurrency = len(vms)
	}
	if concurrency > 10 {
		concurrency = 10
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var failedCount int32
	var successCount int32
	var failedVMIDs []string

	// 创建任务记录
	for i, vm := range vms {
		targetNode := targetNodes[i%len(targetNodes)]
		task := &model.NodeMaintenanceTask{
			PlanID:       planID,
			VMID:         uint32(vm.VMID),
			VMName:       vm.VMName,
			SourceNodeID: nodeID,
			SourceNode:   node.NodeName,
			TargetNodeID: targetNode.Id,
			TargetNode:   targetNode.NodeName,
			Status:       "pending",
		}
		s.repo.GormDB.Create(task)
	}

	// 并发执行迁移
	wg.Add(len(vms))
	for i, vm := range vms {
		go func(idx int, vm v1.NodeVMItem) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			targetNode := targetNodes[idx%len(targetNodes)]

			// 更新任务状态为运行中
			s.repo.GormDB.Model(&model.NodeMaintenanceTask{}).Where("plan_id = ? AND vm_id = ?", planID, vm.VMID).Updates(map[string]interface{}{
				"status":     "running",
				"started_at": time.Now(),
			})

			// 执行迁移
			_, err := pClient.MigrateVM(ctx, targetNode.NodeName, uint32(vm.VMID), map[string]interface{}{
				"target": targetNode.NodeName,
				"online": true,
			})

			if err != nil {
				// 迁移失败
				s.repo.GormDB.Model(&model.NodeMaintenanceTask{}).Where("plan_id = ? AND vm_id = ?", planID, vm.VMID).Updates(map[string]interface{}{
					"status":        "failed",
					"error_message": err.Error(),
					"completed_at":  time.Now(),
				})
				atomic.AddInt32(&failedCount, 1)
				failedVMIDs = append(failedVMIDs, strconv.FormatUint(uint64(vm.VMID), 10))
			} else {
				// 迁移成功，更新数据库中的 node_id
				if err := s.repo.GormDB.Model(&model.PveVM{}).Where("vmid = ?", vm.VMID).Update("node_id", targetNode.Id).Error; err != nil {
					s.repo.GormDB.Model(&model.NodeMaintenanceTask{}).Where("plan_id = ? AND vm_id = ?", planID, vm.VMID).Updates(map[string]interface{}{
						"status":        "failed",
						"error_message": fmt.Sprintf("migration succeeded but failed to update database: %v", err),
						"completed_at":  time.Now(),
					})
					atomic.AddInt32(&failedCount, 1)
					failedVMIDs = append(failedVMIDs, strconv.FormatUint(uint64(vm.VMID), 10))
				} else {
					s.repo.GormDB.Model(&model.NodeMaintenanceTask{}).Where("plan_id = ? AND vm_id = ?", planID, vm.VMID).Updates(map[string]interface{}{
						"status":       "completed",
						"completed_at": time.Now(),
					})
					atomic.AddInt32(&successCount, 1)
				}
			}
		}(i, vm)
	}

	wg.Wait()

	// 更新计划状态
	finalStatus := "completed"
	if failedCount > 0 {
		finalStatus = "failed"
	}

	updateData := map[string]interface{}{
		"status":       finalStatus,
		"end_time":     time.Now(),
		"migrated_vms": int(atomic.LoadInt32(&successCount)),
		"failed_vms":   int(atomic.LoadInt32(&failedCount)),
	}
	if len(failedVMIDs) > 0 {
		updateData["failed_vm_ids"] = strings.Join(failedVMIDs, ",")
	}

	s.repo.GormDB.Model(&model.NodeMaintenancePlan{}).Where("id = ?", planID).Updates(updateData)
}

// GetMaintenancePlan 获取维护计划详情
func (s *NodeMaintenanceService) GetMaintenancePlan(ctx context.Context, planID int64) (*model.NodeMaintenancePlan, error) {
	var plan model.NodeMaintenancePlan
	if err := s.repo.GormDB.First(&plan, planID).Error; err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	return &plan, nil
}

// GetMaintenancePlanTasks 获取维护计划任务列表
func (s *NodeMaintenanceService) GetMaintenancePlanTasks(ctx context.Context, planID int64, page, pageSize int) ([]model.NodeMaintenanceTask, int64, error) {
	var total int64
	if err := s.repo.GormDB.Model(&model.NodeMaintenanceTask{}).Where("plan_id = ?", planID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	var tasks []model.NodeMaintenanceTask
	offset := (page - 1) * pageSize
	if err := s.repo.GormDB.Where("plan_id = ?", planID).Order("id ASC").Offset(offset).Limit(pageSize).Find(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get tasks: %w", err)
	}

	return tasks, total, nil
}

// GetMaintenancePlanProgress 获取维护计划进度
func (s *NodeMaintenanceService) GetMaintenancePlanProgress(ctx context.Context, planID int64) (*model.NodeMaintenancePlan, []model.NodeMaintenanceTask, error) {
	var plan model.NodeMaintenancePlan
	if err := s.repo.GormDB.First(&plan, planID).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to get plan: %w", err)
	}

	var tasks []model.NodeMaintenanceTask
	if err := s.repo.GormDB.Where("plan_id = ?", planID).Order("id ASC").Find(&tasks).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	return &plan, tasks, nil
}

// CancelMaintenancePlan 取消维护计划
func (s *NodeMaintenanceService) CancelMaintenancePlan(ctx context.Context, planID int64) error {
	var plan model.NodeMaintenancePlan
	if err := s.repo.GormDB.First(&plan, planID).Error; err != nil {
		return fmt.Errorf("failed to get plan: %w", err)
	}

	if plan.Status == "completed" || plan.Status == "failed" || plan.Status == "cancelled" {
		return fmt.Errorf("plan already %s", plan.Status)
	}

	// 更新计划状态为已取消
	s.repo.GormDB.Model(&model.NodeMaintenancePlan{}).Where("id = ?", planID).Updates(map[string]interface{}{
		"status":   "cancelled",
		"end_time": time.Now(),
	})

	return nil
}

// GetMaintenanceHistory 获取维护历史
func (s *NodeMaintenanceService) GetMaintenanceHistory(ctx context.Context, nodeID int64, page, pageSize int) ([]model.NodeMaintenancePlan, int64, error) {
	var total int64
	query := s.repo.GormDB.Model(&model.NodeMaintenancePlan{}).Where("status IN ?", []string{"completed", "failed", "cancelled"})
	if nodeID > 0 {
		query = query.Where("node_id = ?", nodeID)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count history: %w", err)
	}

	var plans []model.NodeMaintenancePlan
	offset := (page - 1) * pageSize
	if err := query.Order("start_time DESC").Offset(offset).Limit(pageSize).Find(&plans).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get history: %w", err)
	}

	return plans, total, nil
}

// StartMaintenance 开始主机维护（同步版本，保留向后兼容）
func (s *NodeMaintenanceService) StartMaintenance(ctx context.Context, req v1.StartMaintenanceRequest) (*v1.NodeMaintenanceResult, error) {
	nodeID, err := strconv.ParseInt(req.NodeID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse node ID: %w", err)
	}

	// 获取节点信息
	var node model.PveNode
	if err := s.repo.GormDB.First(&node, nodeID).Error; err != nil {
		return nil, fmt.Errorf("node not found: %w", err)
	}

	// 获取节点上的 VM 列表
	vms, err := s.GetNodeVMs(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get node VMs: %w", err)
	}

	// 获取目标节点列表
	var targetNodes []model.PveNode
	if req.Mode == "manual" && len(req.TargetNodes) > 0 {
		s.repo.GormDB.Where("id IN ?", req.TargetNodes).Find(&targetNodes)
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
		s.repo.GormDB.Where("id IN ?", targetNodeIDs).Find(&targetNodes)
	}

	// 边界检查：目标节点不能为空
	if len(targetNodes) == 0 {
		return nil, fmt.Errorf("no target nodes available")
	}

	// 设置节点为不可调度
	s.repo.GormDB.Model(&model.PveNode{}).Where("id = ?", nodeID).Update("is_schedulable", 0)

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
	if err := s.repo.GormDB.First(&cluster, node.ClusterID).Error; err != nil {
		return nil, fmt.Errorf("cluster not found: %w", err)
	}

	// 创建 Proxmox 客户端
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
				if err := s.repo.GormDB.Model(&model.PveVM{}).Where("vmid = ?", vm.VMID).Update("node_id", targetNode.Id).Error; err != nil {
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

// EndMaintenance 结束维护（带校验）
func (s *NodeMaintenanceService) EndMaintenance(ctx context.Context, nodeID string) error {
	id, err := strconv.ParseInt(nodeID, 10, 64)
	if err != nil {
		return err
	}

	// 检查是否有运行中的维护任务
	var runningPlans int64
	if err := s.repo.GormDB.Model(&model.NodeMaintenancePlan{}).Where("node_id = ? AND status IN ?", id, []string{"pending", "running"}).Count(&runningPlans).Error; err != nil {
		return fmt.Errorf("failed to check running plans: %w", err)
	}

	if runningPlans > 0 {
		return fmt.Errorf("cannot end maintenance: %d maintenance plan(s) still running", runningPlans)
	}

	// 恢复节点调度
	return s.repo.GormDB.Model(&model.PveNode{}).Where("id = ?", id).Update("is_schedulable", 1).Error
}
