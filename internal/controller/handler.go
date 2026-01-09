package controller

import (
	"context"
	"strings"
	"time"

	"pvesphere/internal/model"
	"pvesphere/internal/repository"
	"pvesphere/pkg/hash"
	"pvesphere/pkg/log"

	"go.uber.org/zap"
)

// isStableVMStatus 判断虚拟机状态是否为稳定状态（可以安全上报）
// 过滤掉临时状态：locked, migrating, cloning, creating 等
func isStableVMStatus(status string) bool {
	// Proxmox VM 状态：
	// - running: 运行中（稳定）
	// - stopped: 已停止（稳定）
	// - paused: 已暂停（稳定）
	// - locked: 锁定中（临时，如克隆、迁移、备份等操作中）
	// - migrating: 迁移中（临时）
	// - creating: 创建中（临时）
	// - unknown: 未知状态（可能临时）

	// 过滤掉临时状态
	unstableStatuses := map[string]bool{
		"locked":    true, // 锁定状态（克隆、迁移、备份等操作中）
		"migrating": true, // 迁移中
		"creating":  true, // 创建中
		"unknown":   true, // 未知状态
	}

	// 如果状态为空，也认为是临时状态
	if status == "" {
		return false
	}

	// 检查是否为不稳定状态
	if unstableStatuses[status] {
		return false
	}

	// 其他状态（running, stopped, paused 等）认为是稳定状态
	return true
}

// NodeEventHandler 节点事件处理器
type NodeEventHandler struct {
	repo      repository.PveNodeRepository
	logger    *log.Logger
	clusterID int64
	env       string
}

func NewNodeEventHandler(repo repository.PveNodeRepository, logger *log.Logger, clusterID int64, env string) *NodeEventHandler {
	return &NodeEventHandler{
		repo:      repo,
		logger:    logger,
		clusterID: clusterID,
		env:       env,
	}
}

func (h *NodeEventHandler) OnAdd(obj interface{}) error {
	node, ok := obj.(*model.PveNode)
	if !ok {
		return nil
	}

	node.ClusterID = h.clusterID
	node.Env = h.env
	node.CreateTime = time.Now()
	node.UpdateTime = time.Now()
	// 控制器上报时，Creator 和 Modifier 设置为空字符串（系统自动同步）
	node.Creator = ""
	node.Modifier = ""

	// 计算资源 hash
	resourceHash, err := hash.CalculateResourceHash(node)
	if err != nil {
		h.logger.Error("failed to calculate resource hash", zap.Error(err), zap.String("node", node.NodeName))
		return err
	}
	node.ResourceHash = resourceHash
	node.LastSyncTime = time.Now()

	ctx := context.Background()
	if err := h.repo.Upsert(ctx, node); err != nil {
		h.logger.Error("failed to upsert node", zap.Error(err), zap.String("node", node.NodeName))
		return err
	}

	h.logger.Info("node added", zap.String("node", node.NodeName), zap.Int64("cluster_id", h.clusterID))
	return nil
}

func (h *NodeEventHandler) OnUpdate(oldObj, newObj interface{}) error {
	node, ok := newObj.(*model.PveNode)
	if !ok {
		return nil
	}

	node.ClusterID = h.clusterID
	node.Env = h.env
	node.UpdateTime = time.Now()
	// 控制器上报时，保留已有的 Creator（如果存在），Modifier 设置为空字符串（系统自动同步）
	ctx := context.Background()
	existingNode, err := h.repo.GetByNodeName(ctx, node.NodeName, h.clusterID)
	if err == nil && existingNode != nil {
		node.Creator = existingNode.Creator // 保留已有的 Creator
	}
	node.Modifier = ""

	// 计算资源 hash
	resourceHash, err := hash.CalculateResourceHash(node)
	if err != nil {
		h.logger.Error("failed to calculate resource hash", zap.Error(err), zap.String("node", node.NodeName))
		return err
	}
	node.ResourceHash = resourceHash
	node.LastSyncTime = time.Now()

	if err := h.repo.Upsert(ctx, node); err != nil {
		h.logger.Error("failed to update node", zap.Error(err), zap.String("node", node.NodeName))
		return err
	}

	h.logger.Info("node updated", zap.String("node", node.NodeName), zap.Int64("cluster_id", h.clusterID))
	return nil
}

func (h *NodeEventHandler) OnDelete(obj interface{}) error {
	node, ok := obj.(*model.PveNode)
	if !ok {
		return nil
	}

	ctx := context.Background()
	if err := h.repo.DeleteByNodeName(ctx, node.NodeName, h.clusterID); err != nil {
		h.logger.Error("failed to delete node", zap.Error(err), zap.String("node", node.NodeName))
		return err
	}

	h.logger.Info("node deleted", zap.String("node", node.NodeName), zap.Int64("cluster_id", h.clusterID))
	return nil
}

// VMEventHandler VM 事件处理器
type VMEventHandler struct {
	repo        repository.PveVMRepository
	nodeRepo    repository.PveNodeRepository
	logger      *log.Logger
	clusterID   int64
	clusterName string
}

func NewVMEventHandler(repo repository.PveVMRepository, nodeRepo repository.PveNodeRepository, logger *log.Logger, clusterID int64, clusterName string) *VMEventHandler {
	return &VMEventHandler{
		repo:        repo,
		nodeRepo:    nodeRepo,
		logger:      logger,
		clusterID:   clusterID,
		clusterName: clusterName,
	}
}

func (h *VMEventHandler) OnAdd(obj interface{}) error {
	vm, ok := obj.(*model.PveVM)
	if !ok {
		return nil
	}

	// 状态验证：只处理稳定状态的虚拟机
	if !isStableVMStatus(vm.Status) {
		h.logger.Debug("skipping vm with unstable status",
			zap.Uint32("vmid", vm.VMID),
			zap.String("status", vm.Status),
			zap.String("node", vm.NodeName))
		return nil
	}

	// 验证虚拟机基本信息的完整性
	if vm.VMID == 0 {
		h.logger.Warn("skipping vm with invalid vmid", zap.String("node", vm.NodeName))
		return nil
	}

	// 过滤掉模板同步过程中的临时虚拟机（名称以 "sync-" 开头且不是模板）
	// 说明：
	// - 在克隆和迁移阶段，临时虚拟机是普通虚拟机（IsTemplate=0），不应该上报
	// - 转换完成后，它变成模板（IsTemplate=1），应该上报以便管理
	// - 如果数据库中的记录 IsTemplate=0 但实际已经是模板，会在后续同步时更新
	// 因此只过滤 sync- 开头且 IsTemplate=0 的情况
	if strings.HasPrefix(vm.VmName, "sync-") && vm.IsTemplate == 0 {
		h.logger.Debug("skipping temporary sync vm (not yet converted to template)",
			zap.Uint32("vmid", vm.VMID),
			zap.String("vm_name", vm.VmName),
			zap.Int8("is_template", vm.IsTemplate),
			zap.String("node", vm.NodeName))
		return nil
	}

	vm.ClusterID = h.clusterID
	vm.ClusterName = h.clusterName
	vm.CreateTime = time.Now()
	vm.UpdateTime = time.Now()
	// 控制器上报时，Creator 和 Modifier 设置为空字符串（系统自动同步）
	vm.Creator = ""
	vm.Modifier = ""

	// 如果只有 NodeName 而没有 NodeID，先查询 Node 获取 NodeID 和 NodeIP
	if vm.NodeID == 0 && vm.NodeName != "" {
		ctx := context.Background()
		node, err := h.nodeRepo.GetByNodeName(ctx, vm.NodeName, h.clusterID)
		if err != nil {
			h.logger.Error("failed to get node by name", zap.Error(err), zap.String("node", vm.NodeName))
			return err
		}
		if node != nil {
			vm.NodeID = node.Id
			vm.NodeIP = node.IPAddress
		} else {
			h.logger.Warn("node not found, skipping vm",
				zap.String("node", vm.NodeName),
				zap.Uint32("vmid", vm.VMID))
			return nil
		}
	}

	// 验证 NodeID 是否有效
	if vm.NodeID == 0 {
		h.logger.Warn("skipping vm with invalid node_id",
			zap.Uint32("vmid", vm.VMID),
			zap.String("node", vm.NodeName))
		return nil
	}

	// 计算资源 hash
	resourceHash, err := hash.CalculateResourceHash(vm)
	if err != nil {
		h.logger.Error("failed to calculate resource hash", zap.Error(err), zap.Uint32("vmid", vm.VMID))
		return err
	}
	vm.ResourceHash = resourceHash
	vm.LastSyncTime = time.Now()

	ctx := context.Background()
	if err := h.repo.Upsert(ctx, vm); err != nil {
		h.logger.Error("failed to upsert vm", zap.Error(err), zap.Uint32("vmid", vm.VMID))
		return err
	}

	h.logger.Info("vm added", zap.String("vm", vm.VmName), zap.Uint32("vmid", vm.VMID), zap.String("status", vm.Status))
	return nil
}

func (h *VMEventHandler) OnUpdate(oldObj, newObj interface{}) error {
	vm, ok := newObj.(*model.PveVM)
	if !ok {
		return nil
	}

	// 状态验证：只处理稳定状态的虚拟机
	if !isStableVMStatus(vm.Status) {
		h.logger.Debug("skipping vm update with unstable status",
			zap.Uint32("vmid", vm.VMID),
			zap.String("status", vm.Status),
			zap.String("node", vm.NodeName))
		return nil
	}

	// 验证虚拟机基本信息的完整性
	if vm.VMID == 0 {
		h.logger.Warn("skipping vm update with invalid vmid", zap.String("node", vm.NodeName))
		return nil
	}

	// 过滤掉模板同步过程中的临时虚拟机（名称以 "sync-" 开头且不是模板）
	// 说明：
	// - 在克隆和迁移阶段，临时虚拟机是普通虚拟机（IsTemplate=0），不应该上报
	// - 转换完成后，它变成模板（IsTemplate=1），应该上报以便管理
	// - 如果数据库中的记录 IsTemplate=0 但实际已经是模板，会在后续同步时更新
	// 因此只过滤 sync- 开头且 IsTemplate=0 的情况
	if strings.HasPrefix(vm.VmName, "sync-") && vm.IsTemplate == 0 {
		h.logger.Debug("skipping temporary sync vm update (not yet converted to template)",
			zap.Uint32("vmid", vm.VMID),
			zap.String("vm_name", vm.VmName),
			zap.Int8("is_template", vm.IsTemplate),
			zap.String("node", vm.NodeName))
		return nil
	}

	vm.ClusterID = h.clusterID
	vm.ClusterName = h.clusterName
	vm.UpdateTime = time.Now()

	// 如果只有 NodeName 而没有 NodeID，先查询 Node 获取 NodeID 和 NodeIP
	if vm.NodeID == 0 && vm.NodeName != "" {
		ctx := context.Background()
		node, err := h.nodeRepo.GetByNodeName(ctx, vm.NodeName, h.clusterID)
		if err != nil {
			h.logger.Error("failed to get node by name", zap.Error(err), zap.String("node", vm.NodeName))
			return err
		}
		if node != nil {
			vm.NodeID = node.Id
			vm.NodeIP = node.IPAddress
		} else {
			h.logger.Warn("node not found, skipping vm update",
				zap.String("node", vm.NodeName),
				zap.Uint32("vmid", vm.VMID))
			return nil
		}
	}

	// 验证 NodeID 是否有效
	if vm.NodeID == 0 {
		h.logger.Warn("skipping vm update with invalid node_id",
			zap.Uint32("vmid", vm.VMID),
			zap.String("node", vm.NodeName))
		return nil
	}

	// 控制器上报时，保留已有的 Creator（如果存在），Modifier 设置为空字符串（系统自动同步）
	ctx := context.Background()
	existingVM, err := h.repo.GetByVMID(ctx, vm.VMID, vm.NodeID)
	if err == nil && existingVM != nil {
		vm.Creator = existingVM.Creator // 保留已有的 Creator
	}
	vm.Modifier = ""

	// 计算资源 hash
	resourceHash, err := hash.CalculateResourceHash(vm)
	if err != nil {
		h.logger.Error("failed to calculate resource hash", zap.Error(err), zap.Uint32("vmid", vm.VMID))
		return err
	}
	vm.ResourceHash = resourceHash
	vm.LastSyncTime = time.Now()

	if err := h.repo.Upsert(ctx, vm); err != nil {
		h.logger.Error("failed to update vm", zap.Error(err), zap.Uint32("vmid", vm.VMID))
		return err
	}

	h.logger.Info("vm updated", zap.String("vm", vm.VmName), zap.Uint32("vmid", vm.VMID), zap.String("status", vm.Status))
	return nil
}

func (h *VMEventHandler) OnDelete(obj interface{}) error {
	vm, ok := obj.(*model.PveVM)
	if !ok {
		return nil
	}

	ctx := context.Background()
	// 方案 1：在 controller 中先查询 VM，获取 ID 后删除
	// 先通过 NodeName 查询 VM（向后兼容，如果数据库表中还有 node_name 字段）
	existingVM, err := h.repo.GetByVMIDAndNodeName(ctx, vm.VMID, vm.NodeName)
	if err != nil {
		h.logger.Error("failed to get vm for deletion", zap.Error(err), zap.Uint32("vmid", vm.VMID), zap.String("node", vm.NodeName))
		return err
	}

	if existingVM == nil {
		// VM 不存在，直接返回成功（幂等性）
		h.logger.Info("vm not found, already deleted", zap.Uint32("vmid", vm.VMID), zap.String("node", vm.NodeName))
		return nil
	}

	// 通过 ID 删除
	if err := h.repo.Delete(ctx, existingVM.Id); err != nil {
		h.logger.Error("failed to delete vm", zap.Error(err), zap.Uint32("vmid", vm.VMID), zap.Int64("id", existingVM.Id))
		return err
	}

	h.logger.Info("vm deleted", zap.String("vm", vm.VmName), zap.Uint32("vmid", vm.VMID), zap.String("node", vm.NodeName), zap.Int64("id", existingVM.Id))
	return nil
}

// StorageEventHandler 存储事件处理器
type StorageEventHandler struct {
	repo      repository.PveStorageRepository
	logger    *log.Logger
	clusterID int64
}

func NewStorageEventHandler(repo repository.PveStorageRepository, logger *log.Logger, clusterID int64) *StorageEventHandler {
	return &StorageEventHandler{
		repo:      repo,
		logger:    logger,
		clusterID: clusterID,
	}
}

func (h *StorageEventHandler) OnAdd(obj interface{}) error {
	storage, ok := obj.(*model.PveStorage)
	if !ok {
		return nil
	}

	storage.ClusterID = h.clusterID
	storage.CreateTime = time.Now()
	storage.UpdateTime = time.Now()
	// 控制器上报时，Creator 和 Modifier 设置为空字符串（系统自动同步）
	storage.Creator = ""
	storage.Modifier = ""

	// 计算资源 hash
	resourceHash, err := hash.CalculateResourceHash(storage)
	if err != nil {
		h.logger.Error("failed to calculate resource hash", zap.Error(err), zap.String("storage", storage.StorageName))
		return err
	}
	storage.ResourceHash = resourceHash
	storage.LastSyncTime = time.Now()

	ctx := context.Background()
	if err := h.repo.Upsert(ctx, storage); err != nil {
		h.logger.Error("failed to upsert storage", zap.Error(err), zap.String("storage", storage.StorageName))
		return err
	}

	h.logger.Info("storage added", zap.String("storage", storage.StorageName), zap.String("node", storage.NodeName))
	return nil
}

func (h *StorageEventHandler) OnUpdate(oldObj, newObj interface{}) error {
	storage, ok := newObj.(*model.PveStorage)
	if !ok {
		return nil
	}

	storage.ClusterID = h.clusterID
	storage.UpdateTime = time.Now()
	// 控制器上报时，保留已有的 Creator（如果存在），Modifier 设置为空字符串（系统自动同步）
	ctx := context.Background()
	existingStorage, err := h.repo.GetByStorageName(ctx, storage.StorageName, storage.NodeName, h.clusterID)
	if err == nil && existingStorage != nil {
		storage.Creator = existingStorage.Creator // 保留已有的 Creator
	}
	storage.Modifier = ""

	// 计算资源 hash
	resourceHash, err := hash.CalculateResourceHash(storage)
	if err != nil {
		h.logger.Error("failed to calculate resource hash", zap.Error(err), zap.String("storage", storage.StorageName))
		return err
	}
	storage.ResourceHash = resourceHash
	storage.LastSyncTime = time.Now()

	if err := h.repo.Upsert(ctx, storage); err != nil {
		h.logger.Error("failed to update storage", zap.Error(err), zap.String("storage", storage.StorageName))
		return err
	}

	h.logger.Info("storage updated", zap.String("storage", storage.StorageName), zap.String("node", storage.NodeName))
	return nil
}

func (h *StorageEventHandler) OnDelete(obj interface{}) error {
	storage, ok := obj.(*model.PveStorage)
	if !ok {
		return nil
	}

	ctx := context.Background()
	if err := h.repo.DeleteByStorageName(ctx, storage.StorageName, storage.NodeName, h.clusterID); err != nil {
		h.logger.Error("failed to delete storage", zap.Error(err), zap.String("storage", storage.StorageName))
		return err
	}

	h.logger.Info("storage deleted", zap.String("storage", storage.StorageName), zap.String("node", storage.NodeName))
	return nil
}
