package service

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	v1 "pvesphere/api/v1"
	"pvesphere/internal/model"
	"pvesphere/internal/repository"
	"pvesphere/pkg/log"
	"pvesphere/pkg/proxmox"

	"go.uber.org/zap"
)

type TemplateManagementService interface {
	// 模板导入（基于已有备份文件）
	ImportTemplateFromBackup(ctx context.Context, req *v1.ImportTemplateRequest) (*v1.ImportTemplateResponseData, error)

	// 查询模板详情（包含实例）
	GetTemplateDetailWithInstances(ctx context.Context, templateID int64, includeInstances bool) (*v1.TemplateDetailWithInstances, error)

	// 模板同步
	SyncTemplateToNodes(ctx context.Context, templateID int64, targetNodeIDs []int64) (*v1.SyncTemplateResponseData, error)

	// 同步任务管理
	GetSyncTask(ctx context.Context, taskID int64) (*v1.SyncTaskDetail, error)
	ListSyncTasks(ctx context.Context, req *v1.ListSyncTasksRequest) (*v1.ListSyncTasksResponseData, error)
	RetrySyncTask(ctx context.Context, taskID int64) error

	// 实例管理
	ListTemplateInstances(ctx context.Context, templateID int64) (*v1.ListTemplateInstancesResponseData, error)
}

func NewTemplateManagementService(
	service *Service,
	templateRepo repository.PveTemplateRepository,
	uploadRepo repository.TemplateUploadRepository,
	instanceRepo repository.TemplateInstanceRepository,
	syncTaskRepo repository.TemplateSyncTaskRepository,
	vmRepo repository.PveVMRepository,
	storageRepo repository.PveStorageRepository,
	nodeRepo repository.PveNodeRepository,
	clusterRepo repository.PveClusterRepository,
	logger *log.Logger,
) TemplateManagementService {
	s := &templateManagementService{
		Service:       service,
		templateRepo:  templateRepo,
		uploadRepo:    uploadRepo,
		instanceRepo:  instanceRepo,
		syncTaskRepo:  syncTaskRepo,
		vmRepo:        vmRepo,
		storageRepo:   storageRepo,
		nodeRepo:      nodeRepo,
		clusterRepo:   clusterRepo,
		logger:        logger,
		syncTaskQueue: make(chan int64, 100), // 缓冲队列，最多100个任务
	}

	// 启动任务队列处理器（串行执行）
	go s.processSyncTaskQueue()

	return s
}

// processSyncTaskQueue 处理同步任务队列（串行执行）
func (s *templateManagementService) processSyncTaskQueue() {
	for taskID := range s.syncTaskQueue {
		s.executeSyncTask(context.Background(), taskID)
	}
}

type templateManagementService struct {
	*Service
	templateRepo repository.PveTemplateRepository
	uploadRepo   repository.TemplateUploadRepository
	instanceRepo repository.TemplateInstanceRepository
	syncTaskRepo repository.TemplateSyncTaskRepository
	vmRepo       repository.PveVMRepository
	storageRepo  repository.PveStorageRepository
	nodeRepo     repository.PveNodeRepository
	clusterRepo  repository.PveClusterRepository
	logger       *log.Logger

	// 同步任务队列：用于串行化执行，避免并发克隆冲突
	syncTaskQueue chan int64
	// 模板级别的锁：确保同一模板的同步任务串行执行
	templateLocks sync.Map // map[int64]*sync.Mutex
}

// ImportTemplateFromBackup 从已有备份文件导入模板
func (s *templateManagementService) ImportTemplateFromBackup(
	ctx context.Context,
	req *v1.ImportTemplateRequest,
) (*v1.ImportTemplateResponseData, error) {
	// 1. 验证备份存储是否存在（存放备份文件的存储，通常是 local）
	backupStorage, err := s.storageRepo.GetByID(ctx, req.BackupStorageID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get backup storage", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	if backupStorage == nil {
		return nil, v1.ErrStorageNotFound
	}

	// 2. 验证目标存储是否存在（创建 VM 磁盘的存储，必须支持 images）
	targetStorage, err := s.storageRepo.GetByID(ctx, req.TargetStorageID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get target storage", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	if targetStorage == nil {
		return nil, fmt.Errorf("目标存储不存在")
	}

	// 3. 验证目标存储是否支持 images 内容类型
	if !strings.Contains(targetStorage.Content, "images") {
		s.logger.WithContext(ctx).Error("target storage does not support images",
			zap.String("storage_name", targetStorage.StorageName),
			zap.String("content", targetStorage.Content))
		return nil, fmt.Errorf("目标存储 '%s' 不支持 VM 磁盘镜像(images)，当前支持的内容类型：%s。请选择支持 'images' 的存储（如 local-lvm）",
			targetStorage.StorageName, targetStorage.Content)
	}

	// 4. 防止使用 local 存储作为目标存储
	if targetStorage.Type == "dir" && targetStorage.StorageName == "local" {
		s.logger.WithContext(ctx).Error("cannot use local storage as target",
			zap.String("storage_name", targetStorage.StorageName))
		return nil, fmt.Errorf("不能使用 'local' 存储作为目标存储，请选择支持 VM 磁盘的存储（如 local-lvm）")
	}

	// 5. 验证导入节点是否存在
	importNode, err := s.nodeRepo.GetByID(ctx, req.NodeID)
	if err != nil || importNode == nil {
		s.logger.WithContext(ctx).Error("failed to get import node", zap.Error(err))
		return nil, v1.ErrNodeNotFound
	}

	// 6. 判断目标存储类型（用于后续同步逻辑）
	isShared := targetStorage.Shared == 1

	// 7. 创建模板记录
	template := &model.PveTemplate{
		TemplateName: req.TemplateName,
		ClusterID:    req.ClusterID,
		Description:  req.Description,
		CreateTime:   time.Now(),
		UpdateTime:   time.Now(),
	}
	if err := s.templateRepo.Create(ctx, template); err != nil {
		s.logger.WithContext(ctx).Error("failed to create template", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}

	// 8. 解析备份文件信息
	fileName := req.BackupFile
	fileFormat := s.getBackupFileFormat(fileName)

	// 构建备份文件完整路径（从备份存储读取）
	filePath := s.buildBackupFilePath(backupStorage.StorageName, fileName)

	// 9. 查询备份文件大小
	fileSize, err := s.getBackupFileSize(ctx, importNode, backupStorage.StorageName, fileName)
	if err != nil {
		s.logger.WithContext(ctx).Warn("failed to get backup file size, using 0",
			zap.Error(err),
			zap.String("backup_file", fileName))
		fileSize = 0 // 如果查询失败，使用 0
	}

	// 10. 创建导入记录（记录目标存储信息）
	upload := &model.TemplateUpload{
		TemplateID:     template.Id,
		ClusterID:      req.ClusterID,
		StorageID:      targetStorage.Id, // 使用目标存储
		StorageName:    targetStorage.StorageName,
		StorageType:    targetStorage.Type,
		IsShared:       int8(targetStorage.Shared),
		UploadNodeID:   importNode.Id,
		UploadNodeName: importNode.NodeName,
		FileName:       fileName,
		FilePath:       filePath, // 备份文件路径（从备份存储）
		FileSize:       fileSize, // 备份文件大小（字节）
		FileFormat:     fileFormat,
		Status:         model.TemplateUploadStatusImporting,
		ImportProgress: 0,
		CreateTime:     time.Now(),
		UpdateTime:     time.Now(),
	}
	if err := s.uploadRepo.Create(ctx, upload); err != nil {
		s.logger.WithContext(ctx).Error("failed to create import record", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}

	// 10. 从备份文件导入模板到 PVE（备份文件从 backupStorage 读取，VM 磁盘创建在 targetStorage）
	vmid, err := s.importTemplateFromBackup(ctx, importNode, backupStorage, targetStorage, filePath, fileName, template.TemplateName)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to import template from backup", zap.Error(err))
		_ = s.uploadRepo.UpdateStatus(ctx, upload.Id, model.TemplateUploadStatusFailed, 0, err.Error())
		return nil, v1.ErrTemplateImportFailed
	}

	// 8. 更新导入状态
	upload.Status = model.TemplateUploadStatusImported
	upload.ImportProgress = 100
	if err := s.uploadRepo.Update(ctx, upload); err != nil {
		s.logger.WithContext(ctx).Error("failed to update import status", zap.Error(err))
	}

	// 9. 根据存储类型创建实例
	var syncTasks []v1.TemplateSyncTaskInfo

	if isShared {
		// 共享存储：为所有可见节点创建逻辑实例
		visibleNodes, err := s.getStorageVisibleNodes(ctx, req.ClusterID, targetStorage.StorageName)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to get visible nodes", zap.Error(err))
			return nil, v1.ErrInternalServerError
		}

		for _, node := range visibleNodes {
			isPrimary := int8(0)
			if node.Id == importNode.Id {
				isPrimary = 1
			}

			instance := &model.TemplateInstance{
				TemplateID:  template.Id,
				UploadID:    upload.Id,
				ClusterID:   req.ClusterID,
				NodeID:      node.Id,
				NodeName:    node.NodeName,
				StorageID:   targetStorage.Id,
				StorageName: targetStorage.StorageName,
				IsShared:    1,
				VMID:        vmid,
				Status:      model.TemplateInstanceStatusAvailable,
				IsPrimary:   isPrimary,
				CreateTime:  time.Now(),
				UpdateTime:  time.Now(),
			}
			if err := s.instanceRepo.Create(ctx, instance); err != nil {
				s.logger.WithContext(ctx).Error("failed to create instance",
					zap.Error(err),
					zap.Int64("node_id", node.Id))
			}
		}
	} else {
		// 本地存储：仅为导入节点创建实例
		instance := &model.TemplateInstance{
			TemplateID:  template.Id,
			UploadID:    upload.Id,
			ClusterID:   req.ClusterID,
			NodeID:      importNode.Id,
			NodeName:    importNode.NodeName,
			StorageID:   targetStorage.Id,
			StorageName: targetStorage.StorageName,
			IsShared:    0,
			VMID:        vmid,
			Status:      model.TemplateInstanceStatusAvailable,
			IsPrimary:   1,
			CreateTime:  time.Now(),
			UpdateTime:  time.Now(),
		}
		if err := s.instanceRepo.Create(ctx, instance); err != nil {
			s.logger.WithContext(ctx).Error("failed to create primary instance", zap.Error(err))
			return nil, v1.ErrInternalServerError
		}

		// 如果指定了同步节点，创建同步任务
		if len(req.SyncNodeIDs) > 0 {
			syncTasks, err = s.createSyncTasks(ctx, template, upload, importNode, req.SyncNodeIDs)
			if err != nil {
				s.logger.WithContext(ctx).Error("failed to create sync tasks", zap.Error(err))
				// 不返回错误，允许后续手动同步
			}
		}
	}

	// 10. 返回响应
	return &v1.ImportTemplateResponseData{
		TemplateID:  template.Id,
		ImportID:    upload.Id,
		StorageType: targetStorage.Type,
		IsShared:    isShared,
		ImportNode: v1.TemplateImportNode{
			NodeID:   importNode.Id,
			NodeName: importNode.NodeName,
		},
		SyncTasks: syncTasks,
	}, nil
}

// getBackupFileFormat 获取备份文件格式
func (s *templateManagementService) getBackupFileFormat(fileName string) string {
	// 支持的备份格式：
	// - vzdump-qemu-100-2024_01_01-00_00_00.vma
	// - vzdump-qemu-100-2024_01_01-00_00_00.vma.zst
	// - vzdump-qemu-100-2024_01_01-00_00_00.vma.lzo
	// - vzdump-qemu-100-2024_01_01-00_00_00.vma.gz
	if len(fileName) > 4 {
		// 检查双重扩展名
		if len(fileName) > 8 {
			ext := fileName[len(fileName)-8:]
			if ext == ".vma.zst" || ext == ".vma.lzo" {
				return ext[1:] // 去掉前导点
			}
		}
		if len(fileName) > 7 {
			ext := fileName[len(fileName)-7:]
			if ext == ".vma.gz" {
				return ext[1:]
			}
		}
		// 检查单扩展名
		ext := fileName[len(fileName)-4:]
		if ext == ".vma" {
			return ext[1:]
		}
	}
	return "vma"
}

// buildBackupFilePath 构建备份文件完整路径
func (s *templateManagementService) buildBackupFilePath(storageName, fileName string) string {
	// PVE 备份文件路径通常为：
	// 本地存储：/var/lib/vz/dump/文件名
	// 共享存储：/mnt/pve/{storage_name}/dump/文件名
	// 这里返回相对路径，实际路径由 PVE API 处理
	return fmt.Sprintf("%s:backup/%s", storageName, fileName)
}

// getBackupFileSize 获取备份文件大小
func (s *templateManagementService) getBackupFileSize(
	ctx context.Context,
	node *model.PveNode,
	storageName string,
	fileName string,
) (int64, error) {
	// 1. 获取 Proxmox 客户端
	client, _, err := s.getProxmoxClientForNode(ctx, node.Id)
	if err != nil {
		return 0, fmt.Errorf("failed to get proxmox client: %w", err)
	}

	// 2. 查询存储内容（备份文件）
	contentList, err := client.GetStorageContent(ctx, node.NodeName, storageName, "backup")
	if err != nil {
		return 0, fmt.Errorf("failed to get storage content: %w", err)
	}

	// 3. 查找匹配的备份文件
	// volid 格式通常是：storage:backup/filename
	// 例如：local:backup/vzdump-qemu-100-2024_01_01-00_00_00.vma.zst
	expectedVolid := fmt.Sprintf("%s:backup/%s", storageName, fileName)

	for _, item := range contentList {
		volid, ok := item["volid"].(string)
		if !ok {
			continue
		}

		// 精确匹配 volid，或者文件名匹配
		if volid == expectedVolid || strings.HasSuffix(volid, "/"+fileName) {
			// 提取文件大小
			if size, ok := item["size"].(float64); ok {
				return int64(size), nil
			}
			// 如果 size 是字符串，尝试转换
			if sizeStr, ok := item["size"].(string); ok {
				size, err := strconv.ParseInt(sizeStr, 10, 64)
				if err == nil {
					return size, nil
				}
			}
			// 如果 size 是 int64，直接返回
			if size, ok := item["size"].(int64); ok {
				return size, nil
			}
		}
	}

	return 0, fmt.Errorf("backup file not found: %s", fileName)
}

// getStorageVisibleNodes 获取存储可见的所有节点
func (s *templateManagementService) getStorageVisibleNodes(ctx context.Context, clusterID int64, storageName string) ([]*model.PveNode, error) {
	// 1. 查询该存储的所有记录
	storages, err := s.storageRepo.ListByStorageName(ctx, clusterID, storageName)
	if err != nil {
		return nil, err
	}

	// 2. 提取节点名称
	nodeNameMap := make(map[string]bool)
	for _, s := range storages {
		nodeNameMap[s.NodeName] = true
	}

	// 3. 查询对应的节点信息
	var nodes []*model.PveNode
	for nodeName := range nodeNameMap {
		node, err := s.nodeRepo.GetByNodeName(ctx, nodeName, clusterID)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to get node",
				zap.Error(err),
				zap.String("node_name", nodeName))
			continue
		}
		if node != nil {
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

// importTemplateFromBackup 从备份文件导入模板到 PVE
// backupStorage: 备份文件所在的存储
// targetStorage: VM 磁盘要创建的目标存储
func (s *templateManagementService) importTemplateFromBackup(
	ctx context.Context,
	node *model.PveNode,
	backupStorage *model.PveStorage,
	targetStorage *model.PveStorage,
	filePath string,
	fileName string,
	templateName string,
) (uint32, error) {
	// TODO: 实现从备份导入模板的逻辑
	//
	// 根据 Proxmox VE API 文档：https://pve.proxmox.com/pve-docs/api-viewer
	// 从备份恢复虚拟机应使用：POST /nodes/{node}/qemu
	//
	// 实现步骤：
	// 1. 获取 Proxmox 客户端
	// 2. 分配新的 VMID
	// 3. 调用 CreateQemuVM API，传递 archive 参数恢复备份
	// 4. 等待恢复任务完成
	// 5. 重命名 VM（可选）
	// 6. 转换为模板

	// 1. 获取 Proxmox 客户端
	client, _, err := s.getProxmoxClientForNode(ctx, node.Id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get proxmox client", zap.Error(err))
		return 0, fmt.Errorf("failed to get proxmox client: %w", err)
	}

	// 2. 获取下一个可用的 VMID
	vmid, err := client.GetNextFreeVMID(ctx)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get next free vmid", zap.Error(err))
		return 0, fmt.Errorf("failed to get next free vmid: %w", err)
	}

	// 3. 从备份恢复虚拟机
	// archive 格式：storage:backup/filename（从备份存储读取）
	archivePath := filePath
	if !strings.Contains(filePath, ":") {
		// 如果 filePath 不包含存储前缀，自动添加
		archivePath = fmt.Sprintf("%s:backup/%s", backupStorage.StorageName, fileName)
	}

	params := url.Values{}
	params.Set("vmid", fmt.Sprintf("%d", vmid))
	params.Set("archive", archivePath) // 从备份存储读取备份文件
	if targetStorage.StorageName != "" {
		params.Set("storage", targetStorage.StorageName) // VM 磁盘创建在目标存储
	}
	if templateName != "" {
		params.Set("name", templateName)
	}

	s.logger.WithContext(ctx).Info("restoring vm from backup",
		zap.String("node", node.NodeName),
		zap.String("archive", archivePath),
		zap.String("backup_storage", backupStorage.StorageName),
		zap.String("target_storage", targetStorage.StorageName),
		zap.Uint32("vmid", vmid))

	upid, err := client.CreateQemuVM(ctx, node.NodeName, params)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to restore from backup", zap.Error(err))
		return 0, fmt.Errorf("failed to restore from backup: %w", err)
	}

	s.logger.WithContext(ctx).Info("restore task started", zap.String("upid", upid), zap.Uint32("vmid", vmid))

	// 4. 等待恢复任务完成
	err = s.waitForTask(ctx, client, node.NodeName, upid, 30*time.Minute, nil)
	if err != nil {
		s.logger.WithContext(ctx).Error("restore task failed", zap.Error(err), zap.String("upid", upid))
		return 0, fmt.Errorf("restore task failed: %w", err)
	}

	s.logger.WithContext(ctx).Info("restore task completed", zap.Uint32("vmid", vmid))

	// 5. 转换为模板
	err = client.ConvertToTemplate(ctx, node.NodeName, vmid, "")
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to convert to template", zap.Error(err), zap.Uint32("vmid", vmid))
		return 0, fmt.Errorf("failed to convert to template: %w", err)
	}

	s.logger.WithContext(ctx).Info("template imported from backup successfully",
		zap.String("node", node.NodeName),
		zap.Uint32("vmid", vmid),
		zap.String("template_name", templateName))

	return vmid, nil
}

// createSyncTasks 创建同步任务
func (s *templateManagementService) createSyncTasks(
	ctx context.Context,
	template *model.PveTemplate,
	upload *model.TemplateUpload,
	sourceNode *model.PveNode,
	targetNodeIDs []int64,
) ([]v1.TemplateSyncTaskInfo, error) {
	var tasks []v1.TemplateSyncTaskInfo

	for _, targetNodeID := range targetNodeIDs {
		// 检查是否已存在可用的实例
		existing, err := s.instanceRepo.GetByTemplateAndNode(ctx, template.Id, targetNodeID)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to check existing instance",
				zap.Error(err),
				zap.Int64("node_id", targetNodeID))
			continue
		}
		if existing != nil {
			// 如果实例状态是 available，跳过
			if existing.Status == model.TemplateInstanceStatusAvailable {
				s.logger.WithContext(ctx).Info("instance already available, skip sync",
					zap.Int64("template_id", template.Id),
					zap.Int64("node_id", targetNodeID))
				continue
			}
			// 如果实例状态是 pending 或 failed，清理旧的任务和实例
			if existing.Status == model.TemplateInstanceStatusPending || existing.Status == model.TemplateInstanceStatusFailed {
				s.logger.WithContext(ctx).Info("cleaning up failed/pending instance",
					zap.Int64("template_id", template.Id),
					zap.Int64("node_id", targetNodeID),
					zap.String("old_status", existing.Status))
				// 如果有同步任务，删除失败的任务
				if existing.SyncTaskID != nil {
					oldTask, _ := s.syncTaskRepo.GetByID(ctx, *existing.SyncTaskID)
					if oldTask != nil && oldTask.Status == model.TemplateSyncTaskStatusFailed {
						_ = s.syncTaskRepo.Delete(ctx, *existing.SyncTaskID)
					}
				}
				// 删除旧的实例记录
				_ = s.instanceRepo.Delete(ctx, existing.Id)
			}
		}

		// 获取目标节点信息
		targetNode, err := s.nodeRepo.GetByID(ctx, targetNodeID)
		if err != nil || targetNode == nil {
			s.logger.WithContext(ctx).Error("failed to get target node",
				zap.Error(err),
				zap.Int64("node_id", targetNodeID))
			continue
		}

		// 创建同步任务
		syncTask := &model.TemplateSyncTask{
			TemplateID:     template.Id,
			UploadID:       upload.Id,
			ClusterID:      template.ClusterID,
			SourceNodeID:   sourceNode.Id,
			SourceNodeName: sourceNode.NodeName,
			TargetNodeID:   targetNode.Id,
			TargetNodeName: targetNode.NodeName,
			StorageName:    upload.StorageName,
			FilePath:       upload.FilePath,
			FileSize:       upload.FileSize,
			Status:         model.TemplateSyncTaskStatusPending,
			Progress:       0,
			CreateTime:     time.Now(),
			UpdateTime:     time.Now(),
		}
		if err := s.syncTaskRepo.Create(ctx, syncTask); err != nil {
			s.logger.WithContext(ctx).Error("failed to create sync task",
				zap.Error(err),
				zap.Int64("target_node_id", targetNodeID))
			continue
		}

		// 创建待同步的实例记录
		instance := &model.TemplateInstance{
			TemplateID:  template.Id,
			UploadID:    upload.Id,
			ClusterID:   template.ClusterID,
			NodeID:      targetNode.Id,
			NodeName:    targetNode.NodeName,
			StorageID:   upload.StorageID,
			StorageName: upload.StorageName,
			IsShared:    0,
			VMID:        0, // 将在同步后分配
			Status:      model.TemplateInstanceStatusPending,
			SyncTaskID:  &syncTask.Id,
			IsPrimary:   0,
			CreateTime:  time.Now(),
			UpdateTime:  time.Now(),
		}
		if err := s.instanceRepo.Create(ctx, instance); err != nil {
			s.logger.WithContext(ctx).Error("failed to create instance",
				zap.Error(err),
				zap.Int64("node_id", targetNodeID))
		}

		// 添加到响应列表
		tasks = append(tasks, v1.TemplateSyncTaskInfo{
			TaskID:         syncTask.Id,
			TargetNodeID:   targetNode.Id,
			TargetNodeName: targetNode.NodeName,
			Status:         syncTask.Status,
		})

		// 将任务加入队列（串行执行，避免并发克隆冲突）
		select {
		case s.syncTaskQueue <- syncTask.Id:
			s.logger.WithContext(ctx).Info("sync task queued",
				zap.Int64("task_id", syncTask.Id),
				zap.Int64("template_id", template.Id),
				zap.Int64("target_node_id", targetNodeID))
		default:
			s.logger.WithContext(ctx).Error("sync task queue is full, task may be delayed",
				zap.Int64("task_id", syncTask.Id))
			// 队列满了，仍然尝试加入（可能会阻塞，但不会丢失任务）
			go func() {
				s.syncTaskQueue <- syncTask.Id
			}()
		}
	}

	return tasks, nil
}

// GetTemplateDetailWithInstances 查询模板详情（包含实例）
func (s *templateManagementService) GetTemplateDetailWithInstances(
	ctx context.Context,
	templateID int64,
	includeInstances bool,
) (*v1.TemplateDetailWithInstances, error) {
	// 1. 查询模板基本信息
	template, err := s.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get template", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	if template == nil {
		return nil, v1.ErrNotFound
	}

	// 2. 查询集群名称
	cluster, err := s.clusterRepo.GetByID(ctx, template.ClusterID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get cluster", zap.Error(err))
	}
	clusterName := ""
	if cluster != nil {
		clusterName = cluster.ClusterName
	}

	// 3. 查询上传信息
	upload, err := s.uploadRepo.GetByTemplateID(ctx, templateID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get upload", zap.Error(err))
	}

	var uploadInfo *v1.TemplateUploadInfo
	if upload != nil {
		uploadInfo = &v1.TemplateUploadInfo{
			UploadID:    upload.Id,
			StorageName: upload.StorageName,
			IsShared:    upload.IsShared == 1,
			FileName:    upload.FileName,
			FileSize:    upload.FileSize,
			Status:      upload.Status,
		}
	}

	// 4. 查询实例信息
	var instances []v1.TemplateInstanceInfo
	if includeInstances {
		instanceList, err := s.instanceRepo.ListByTemplateID(ctx, templateID)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to list instances", zap.Error(err))
		} else {
			for _, inst := range instanceList {
				instInfo := v1.TemplateInstanceInfo{
					InstanceID:  inst.Id,
					NodeID:      inst.NodeID,
					NodeName:    inst.NodeName,
					VMID:        inst.VMID,
					StorageName: inst.StorageName,
					Status:      inst.Status,
					IsPrimary:   inst.IsPrimary == 1,
				}

				// 如果有同步任务，查询进度
				if inst.SyncTaskID != nil {
					task, err := s.syncTaskRepo.GetByID(ctx, *inst.SyncTaskID)
					if err == nil && task != nil {
						instInfo.SyncProgress = &task.Progress
					}
				}

				instances = append(instances, instInfo)
			}
		}
	}

	return &v1.TemplateDetailWithInstances{
		Id:           template.Id,
		TemplateName: template.TemplateName,
		ClusterID:    template.ClusterID,
		ClusterName:  clusterName,
		Description:  template.Description,
		UploadInfo:   uploadInfo,
		Instances:    instances,
	}, nil
}

// SyncTemplateToNodes 同步模板到其他节点
func (s *templateManagementService) SyncTemplateToNodes(
	ctx context.Context,
	templateID int64,
	targetNodeIDs []int64,
) (*v1.SyncTemplateResponseData, error) {
	// 1. 获取模板信息
	template, err := s.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get template", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	if template == nil {
		return nil, v1.ErrNotFound
	}

	// 2. 获取上传信息
	upload, err := s.uploadRepo.GetByTemplateID(ctx, templateID)
	if err != nil || upload == nil {
		s.logger.WithContext(ctx).Error("failed to get upload", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}

	// 3. 验证是否为本地存储
	if upload.IsShared == 1 {
		return nil, v1.ErrSharedStorageNoSync
	}

	// 4. 获取主实例（源节点）
	primaryInstance, err := s.instanceRepo.GetPrimaryInstance(ctx, templateID)
	if err != nil || primaryInstance == nil {
		s.logger.WithContext(ctx).Error("failed to get primary instance", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}

	sourceNode, err := s.nodeRepo.GetByID(ctx, primaryInstance.NodeID)
	if err != nil || sourceNode == nil {
		s.logger.WithContext(ctx).Error("failed to get source node", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}

	// 5. 创建同步任务
	tasks, err := s.createSyncTasks(ctx, template, upload, sourceNode, targetNodeIDs)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to create sync tasks", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}

	return &v1.SyncTemplateResponseData{
		SyncTasks: tasks,
	}, nil
}

// GetSyncTask 查询同步任务
func (s *templateManagementService) GetSyncTask(ctx context.Context, taskID int64) (*v1.SyncTaskDetail, error) {
	task, err := s.syncTaskRepo.GetByID(ctx, taskID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get sync task", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	if task == nil {
		return nil, v1.ErrNotFound
	}

	// 查询模板名称
	template, err := s.templateRepo.GetByID(ctx, task.TemplateID)
	templateName := ""
	if err == nil && template != nil {
		templateName = template.TemplateName
	}

	return &v1.SyncTaskDetail{
		TaskID:       task.Id,
		TemplateID:   task.TemplateID,
		TemplateName: templateName,
		SourceNode: v1.NodeInfo{
			NodeID:   task.SourceNodeID,
			NodeName: task.SourceNodeName,
		},
		TargetNode: v1.NodeInfo{
			NodeID:   task.TargetNodeID,
			NodeName: task.TargetNodeName,
		},
		StorageName:   task.StorageName,
		Status:        task.Status,
		Progress:      task.Progress,
		SyncStartTime: task.SyncStartTime,
		SyncEndTime:   task.SyncEndTime,
		ErrorMessage:  task.ErrorMessage,
	}, nil
}

// ListSyncTasks 列出同步任务
func (s *templateManagementService) ListSyncTasks(
	ctx context.Context,
	req *v1.ListSyncTasksRequest,
) (*v1.ListSyncTasksResponseData, error) {
	tasks, total, err := s.syncTaskRepo.ListWithPagination(
		ctx,
		req.Page,
		req.PageSize,
		req.TemplateID,
		req.Status,
	)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to list sync tasks", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}

	var list []v1.SyncTaskDetail
	for _, task := range tasks {
		// 查询模板名称
		template, err := s.templateRepo.GetByID(ctx, task.TemplateID)
		templateName := ""
		if err == nil && template != nil {
			templateName = template.TemplateName
		}

		list = append(list, v1.SyncTaskDetail{
			TaskID:       task.Id,
			TemplateID:   task.TemplateID,
			TemplateName: templateName,
			SourceNode: v1.NodeInfo{
				NodeID:   task.SourceNodeID,
				NodeName: task.SourceNodeName,
			},
			TargetNode: v1.NodeInfo{
				NodeID:   task.TargetNodeID,
				NodeName: task.TargetNodeName,
			},
			StorageName:   task.StorageName,
			Status:        task.Status,
			Progress:      task.Progress,
			SyncStartTime: task.SyncStartTime,
			SyncEndTime:   task.SyncEndTime,
			ErrorMessage:  task.ErrorMessage,
		})
	}

	return &v1.ListSyncTasksResponseData{
		Total: total,
		List:  list,
	}, nil
}

// RetrySyncTask 重试同步任务
func (s *templateManagementService) RetrySyncTask(ctx context.Context, taskID int64) error {
	task, err := s.syncTaskRepo.GetByID(ctx, taskID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get sync task", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if task == nil {
		return v1.ErrNotFound
	}

	// 只有失败的任务才能重试
	if task.Status != model.TemplateSyncTaskStatusFailed {
		return v1.ErrInvalidOperation
	}

	// 重置任务状态
	err = s.syncTaskRepo.UpdateStatus(ctx, taskID, model.TemplateSyncTaskStatusPending, 0, "")
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to update task status", zap.Error(err))
		return v1.ErrInternalServerError
	}

	// 将任务加入队列（串行执行）
	select {
	case s.syncTaskQueue <- taskID:
		s.logger.WithContext(ctx).Info("retry task queued",
			zap.Int64("task_id", taskID))
	default:
		s.logger.WithContext(ctx).Error("sync task queue is full, task may be delayed",
			zap.Int64("task_id", taskID))
		// 队列满了，仍然尝试加入
		go func() {
			s.syncTaskQueue <- taskID
		}()
	}

	return nil
}

// ListTemplateInstances 列出模板实例
func (s *templateManagementService) ListTemplateInstances(
	ctx context.Context,
	templateID int64,
) (*v1.ListTemplateInstancesResponseData, error) {
	// 查询模板名称
	template, err := s.templateRepo.GetByID(ctx, templateID)
	if err != nil || template == nil {
		s.logger.WithContext(ctx).Error("failed to get template", zap.Error(err))
		return nil, v1.ErrNotFound
	}

	// 查询实例列表
	instances, err := s.instanceRepo.ListByTemplateID(ctx, templateID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to list instances", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}

	var list []v1.TemplateInstanceInfo
	for _, inst := range instances {
		instInfo := v1.TemplateInstanceInfo{
			InstanceID:  inst.Id,
			NodeID:      inst.NodeID,
			NodeName:    inst.NodeName,
			VMID:        inst.VMID,
			StorageName: inst.StorageName,
			Status:      inst.Status,
			IsPrimary:   inst.IsPrimary == 1,
		}

		// 如果有同步任务，查询进度
		if inst.SyncTaskID != nil {
			task, err := s.syncTaskRepo.GetByID(ctx, *inst.SyncTaskID)
			if err == nil && task != nil {
				instInfo.SyncProgress = &task.Progress
			}
		}

		list = append(list, instInfo)
	}

	return &v1.ListTemplateInstancesResponseData{
		TemplateID:   template.Id,
		TemplateName: template.TemplateName,
		Total:        int64(len(list)),
		Instances:    list,
	}, nil
}

// ==================== 私有辅助方法 ====================

// waitForTask 等待 Proxmox 任务完成，支持进度回调
func (s *templateManagementService) waitForTask(
	ctx context.Context,
	client *proxmox.ProxmoxClient,
	nodeName string,
	upid string,
	timeout time.Duration,
	progressCallback func(progress int),
) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastProgress := 0
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("task timeout: %w", ctx.Err())
		case <-ticker.C:
			status, err := client.GetTaskStatus(ctx, nodeName, upid)
			if err != nil {
				s.logger.Warn("failed to get task status", zap.Error(err))
				continue
			}

			// 尝试获取进度（部分任务可能不支持）
			if progress, ok := status["progress"].(float64); ok && progressCallback != nil {
				currentProgress := int(progress * 100)
				if currentProgress != lastProgress {
					progressCallback(currentProgress)
					lastProgress = currentProgress
				}
			}

			// 检查任务状态
			statusStr, _ := status["status"].(string)
			if statusStr == "stopped" {
				exitStatus, _ := status["exitstatus"].(string)
				if exitStatus == "OK" {
					return nil // 成功
				}
				return fmt.Errorf("task failed with status: %s", exitStatus)
			}
		}
	}
}

// getProxmoxClientForNode 获取指定节点的 Proxmox 客户端
func (s *templateManagementService) getProxmoxClientForNode(
	ctx context.Context,
	nodeID int64,
) (*proxmox.ProxmoxClient, *model.PveNode, error) {
	// 1. 获取节点信息
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node", zap.Error(err))
		return nil, nil, v1.ErrInternalServerError
	}
	if node == nil {
		return nil, nil, fmt.Errorf("节点不存在")
	}

	// 2. 获取集群信息
	if node.ClusterID <= 0 {
		return nil, nil, fmt.Errorf("节点的集群 ID 无效")
	}
	cluster, err := s.clusterRepo.GetByID(ctx, node.ClusterID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get cluster", zap.Error(err))
		return nil, nil, v1.ErrInternalServerError
	}
	if cluster == nil {
		return nil, nil, fmt.Errorf("集群 ID %d 不存在", node.ClusterID)
	}

	// 3. 创建 Proxmox 客户端
	client, err := proxmox.NewProxmoxClient(cluster.ApiUrl, cluster.UserId, cluster.UserToken)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to create proxmox client", zap.Error(err))
		return nil, nil, v1.ErrInternalServerError
	}

	return client, node, nil
}

// executeSyncTask 执行模板同步任务
// 流程：1. 在源节点克隆模板（存储保持一致） 2. 迁移到目标节点 3. 转换为模板
func (s *templateManagementService) executeSyncTask(ctx context.Context, taskID int64) {
	// 使用新的 context，避免请求 context 关闭
	ctx = context.Background()

	// 1. 获取同步任务信息
	task, err := s.syncTaskRepo.GetByID(ctx, taskID)
	if err != nil || task == nil {
		s.logger.WithContext(ctx).Error("failed to get sync task",
			zap.Error(err),
			zap.Int64("task_id", taskID))
		return
	}

	// 获取模板级别的锁，确保同一模板的同步任务串行执行
	lockInterface, _ := s.templateLocks.LoadOrStore(task.TemplateID, &sync.Mutex{})
	templateLock := lockInterface.(*sync.Mutex)
	templateLock.Lock()
	defer templateLock.Unlock()

	// 再次检查任务状态（可能在等待锁期间被取消或完成）
	task2, err2 := s.syncTaskRepo.GetByID(ctx, taskID)
	if err2 != nil {
		s.logger.WithContext(ctx).Error("failed to get sync task after lock",
			zap.Error(err2),
			zap.Int64("task_id", taskID))
		return
	}
	if task2 == nil {
		s.logger.WithContext(ctx).Error("sync task not found after lock",
			zap.Int64("task_id", taskID))
		return
	}
	if task2.Status != model.TemplateSyncTaskStatusPending {
		s.logger.WithContext(ctx).Info("task status changed, skip execution",
			zap.Int64("task_id", taskID),
			zap.String("status", task2.Status))
		return
	}
	task = task2 // 使用最新的任务信息

	// 更新任务状态为同步中
	now := time.Now()
	task.Status = model.TemplateSyncTaskStatusSyncing
	task.SyncStartTime = &now
	task.Progress = 0
	if err := s.syncTaskRepo.Update(ctx, task); err != nil {
		s.logger.WithContext(ctx).Error("failed to update task status to syncing", zap.Error(err))
	}

	// 2. 获取模板信息（用于生成同步后的虚拟机名称）
	template, err := s.templateRepo.GetByID(ctx, task.TemplateID)
	if err != nil || template == nil {
		errorMsg := "failed to get template"
		if err != nil {
			errorMsg = err.Error()
		}
		s.logger.WithContext(ctx).Error("failed to get template",
			zap.Error(err),
			zap.Int64("template_id", task.TemplateID))
		task.Status = model.TemplateSyncTaskStatusFailed
		task.ErrorMessage = errorMsg
		_ = s.syncTaskRepo.Update(ctx, task)
		return
	}

	// 生成同步后的虚拟机名称：sync-{template_name}-{task_id}
	syncVMName := fmt.Sprintf("sync-%s-%d", template.TemplateName, taskID)

	// 3. 获取源节点的主实例（primary instance）的 VMID
	primaryInstance, err := s.instanceRepo.GetPrimaryInstance(ctx, task.TemplateID)
	if err != nil || primaryInstance == nil {
		errorMsg := "failed to get primary instance"
		if err != nil {
			errorMsg = err.Error()
		}
		s.logger.WithContext(ctx).Error("failed to get primary instance",
			zap.Error(err),
			zap.Int64("template_id", task.TemplateID))
		task.Status = model.TemplateSyncTaskStatusFailed
		task.ErrorMessage = errorMsg
		_ = s.syncTaskRepo.Update(ctx, task)
		return
	}

	if primaryInstance.VMID == 0 {
		errorMsg := "primary instance VMID is 0"
		s.logger.WithContext(ctx).Error(errorMsg, zap.Int64("instance_id", primaryInstance.Id))
		task.Status = model.TemplateSyncTaskStatusFailed
		task.ErrorMessage = errorMsg
		_ = s.syncTaskRepo.Update(ctx, task)
		return
	}

	// 4. 获取源节点和目标节点的 Proxmox 客户端
	sourceClient, sourceNode, err := s.getProxmoxClientForNode(ctx, task.SourceNodeID)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to get source node client: %v", err)
		s.logger.WithContext(ctx).Error(errorMsg, zap.Error(err))
		task.Status = model.TemplateSyncTaskStatusFailed
		task.ErrorMessage = errorMsg
		_ = s.syncTaskRepo.Update(ctx, task)
		return
	}

	targetClient, targetNode, err := s.getProxmoxClientForNode(ctx, task.TargetNodeID)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to get target node client: %v", err)
		s.logger.WithContext(ctx).Error(errorMsg, zap.Error(err))
		task.Status = model.TemplateSyncTaskStatusFailed
		task.ErrorMessage = errorMsg
		_ = s.syncTaskRepo.Update(ctx, task)
		return
	}

	// 5. 分配新的 VMID（用于克隆的临时 VM）
	newVMID, err := sourceClient.GetNextFreeVMID(ctx)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to get next free vmid: %v", err)
		s.logger.WithContext(ctx).Error(errorMsg, zap.Error(err))
		task.Status = model.TemplateSyncTaskStatusFailed
		task.ErrorMessage = errorMsg
		_ = s.syncTaskRepo.Update(ctx, task)
		return
	}

	s.logger.WithContext(ctx).Info("starting template sync",
		zap.Int64("task_id", taskID),
		zap.Uint32("source_vmid", primaryInstance.VMID),
		zap.Uint32("new_vmid", newVMID),
		zap.String("source_node", sourceNode.NodeName),
		zap.String("target_node", targetNode.NodeName))

	// 6. 在源节点克隆模板（不指定 target，存储保持一致）
	// 注意：不指定 storage 参数，保持与原模板相同的存储
	cloneReq := &proxmox.CloneVMRequest{
		NewID:       newVMID,
		Name:        syncVMName, // 使用模板名称生成：sync-{template_name}
		Target:      "",         // 不指定 target，在同一节点克隆
		Full:        1,          // 完整克隆
		Storage:     "",         // 不指定 storage，保持原存储
		Description: fmt.Sprintf("Template sync VM: %s", template.TemplateName),
	}

	cloneUPID, err := sourceClient.CloneVM(ctx, sourceNode.NodeName, primaryInstance.VMID, cloneReq)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to clone template: %v", err)
		s.logger.WithContext(ctx).Error(errorMsg, zap.Error(err))
		task.Status = model.TemplateSyncTaskStatusFailed
		task.ErrorMessage = errorMsg
		_ = s.syncTaskRepo.Update(ctx, task)
		return
	}

	// 等待克隆任务完成
	task.Progress = 20
	_ = s.syncTaskRepo.Update(ctx, task)

	err = s.waitForTask(ctx, sourceClient, sourceNode.NodeName, cloneUPID, 30*time.Minute, func(progress int) {
		// 克隆进度：0-50%
		overallProgress := 10 + (progress * 40 / 100)
		task.Progress = overallProgress
		_ = s.syncTaskRepo.Update(ctx, task)
	})
	if err != nil {
		errorMsg := fmt.Sprintf("clone task failed: %v", err)
		s.logger.WithContext(ctx).Error(errorMsg, zap.Error(err))
		task.Status = model.TemplateSyncTaskStatusFailed
		task.ErrorMessage = errorMsg
		_ = s.syncTaskRepo.Update(ctx, task)
		// 清理：删除克隆失败的临时 VM
		_ = sourceClient.DeleteVM(ctx, sourceNode.NodeName, newVMID, true)
		return
	}

	s.logger.WithContext(ctx).Info("clone completed",
		zap.Uint32("new_vmid", newVMID),
		zap.String("source_node", sourceNode.NodeName))

	// 6. 迁移克隆的 VM 到目标节点
	task.Progress = 50
	task.Status = model.TemplateSyncTaskStatusImporting
	_ = s.syncTaskRepo.Update(ctx, task)

	migrateParams := map[string]interface{}{
		"target":  targetNode.NodeName,
		"online":  false, // 离线迁移（模板通常是停止状态）
		"storage": "",    // 不指定 storage，使用目标节点的默认存储
	}

	migrateUPID, err := sourceClient.MigrateVM(ctx, sourceNode.NodeName, newVMID, migrateParams)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to migrate VM: %v", err)
		s.logger.WithContext(ctx).Error(errorMsg, zap.Error(err))
		task.Status = model.TemplateSyncTaskStatusFailed
		task.ErrorMessage = errorMsg
		_ = s.syncTaskRepo.Update(ctx, task)
		// 清理：删除克隆的临时 VM
		_ = sourceClient.DeleteVM(ctx, sourceNode.NodeName, newVMID, true)
		return
	}

	// 等待迁移任务完成
	err = s.waitForTask(ctx, sourceClient, sourceNode.NodeName, migrateUPID, 60*time.Minute, func(progress int) {
		// 迁移进度：50-90%
		overallProgress := 50 + (progress * 40 / 100)
		task.Progress = overallProgress
		_ = s.syncTaskRepo.Update(ctx, task)
	})
	if err != nil {
		errorMsg := fmt.Sprintf("migrate task failed: %v", err)
		s.logger.WithContext(ctx).Error(errorMsg, zap.Error(err))
		task.Status = model.TemplateSyncTaskStatusFailed
		task.ErrorMessage = errorMsg
		_ = s.syncTaskRepo.Update(ctx, task)
		// 清理：尝试删除目标节点上的 VM（如果迁移部分成功）
		_ = targetClient.DeleteVM(ctx, targetNode.NodeName, newVMID, true)
		return
	}

	s.logger.WithContext(ctx).Info("migration completed",
		zap.Uint32("vmid", newVMID),
		zap.String("target_node", targetNode.NodeName))

	// 7. 在目标节点转换为模板
	task.Progress = 90
	_ = s.syncTaskRepo.Update(ctx, task)

	err = targetClient.ConvertToTemplate(ctx, targetNode.NodeName, newVMID, "")
	if err != nil {
		errorMsg := fmt.Sprintf("failed to convert to template: %v", err)
		s.logger.WithContext(ctx).Error(errorMsg, zap.Error(err))
		task.Status = model.TemplateSyncTaskStatusFailed
		task.ErrorMessage = errorMsg
		_ = s.syncTaskRepo.Update(ctx, task)
		// 清理：删除目标节点上的 VM
		_ = targetClient.DeleteVM(ctx, targetNode.NodeName, newVMID, true)
		return
	}

	s.logger.WithContext(ctx).Info("template conversion completed",
		zap.Uint32("vmid", newVMID),
		zap.String("target_node", targetNode.NodeName))

	// 7.1 清理数据库中可能存在的临时虚拟机记录（脏数据清理），并确保模板记录正确
	// 因为临时虚拟机在迁移和转换过程中可能被上报系统捕获，导致数据库中有普通虚拟机记录
	// 但实际在 Proxmox 中已经是模板了，需要清理这些脏数据
	// 同时，如果数据库中还没有记录，可以通过 GetVMConfig 确认后创建正确的模板记录
	if err := s.cleanupAndEnsureTemplateRecord(ctx, targetNode, newVMID, syncVMName); err != nil {
		s.logger.WithContext(ctx).Warn("failed to cleanup/ensure template record",
			zap.Error(err),
			zap.Uint32("vmid", newVMID),
			zap.Int64("node_id", targetNode.Id))
		// 不中断流程，只记录警告
	}

	// 8. 更新实例状态
	instance, err := s.instanceRepo.GetByTemplateAndNode(ctx, task.TemplateID, task.TargetNodeID)
	if err != nil || instance == nil {
		s.logger.WithContext(ctx).Error("failed to get target instance",
			zap.Error(err),
			zap.Int64("template_id", task.TemplateID),
			zap.Int64("target_node_id", task.TargetNodeID))
	} else {
		instance.VMID = newVMID
		instance.Status = model.TemplateInstanceStatusAvailable
		if err := s.instanceRepo.Update(ctx, instance); err != nil {
			s.logger.WithContext(ctx).Error("failed to update instance",
				zap.Error(err),
				zap.Int64("instance_id", instance.Id))
		}
	}

	// 9. 更新同步任务状态为完成
	task.Progress = 100
	task.Status = model.TemplateSyncTaskStatusCompleted
	endTime := time.Now()
	task.SyncEndTime = &endTime
	task.ErrorMessage = ""
	if err := s.syncTaskRepo.Update(ctx, task); err != nil {
		s.logger.WithContext(ctx).Error("failed to update task status to completed", zap.Error(err))
	}

	s.logger.WithContext(ctx).Info("template sync task completed successfully",
		zap.Int64("task_id", taskID),
		zap.Uint32("vmid", newVMID),
		zap.String("target_node", targetNode.NodeName))
}

// cleanupAndEnsureTemplateRecord 清理临时虚拟机的脏数据并确保模板记录正确
// 1. 如果数据库中存在 isTemplate=0 的记录，删除它（脏数据清理）
// 2. 通过 GetVMConfig 确认 Proxmox 中确实是模板后，创建正确的模板记录
// 这样确保数据库记录与实际状态一致，即使上报系统还没上报
func (s *templateManagementService) cleanupAndEnsureTemplateRecord(
	ctx context.Context,
	node *model.PveNode,
	vmid uint32,
	vmName string,
) error {
	// 1. 查找数据库中是否存在该 VMID 和 NodeID 的记录
	vm, err := s.vmRepo.GetByVMID(ctx, vmid, node.Id)
	if err != nil {
		return fmt.Errorf("failed to get vm record: %w", err)
	}

	// 2. 获取 Proxmox 客户端，用于查询 VM 配置确认是否为模板
	client, _, err := s.getProxmoxClientForNode(ctx, node.Id)
	if err != nil {
		return fmt.Errorf("failed to get proxmox client: %w", err)
	}

	// 3. 通过 GetVMConfig 确认 Proxmox 中的真实状态
	vmConfig, err := client.GetVMCurrentConfig(ctx, node.NodeName, vmid)
	if err != nil {
		// 如果获取配置失败，只清理脏数据，不创建新记录
		s.logger.WithContext(ctx).Warn("failed to get vm config for template verification",
			zap.Error(err),
			zap.Uint32("vmid", vmid),
			zap.String("node", node.NodeName))

		// 只执行清理逻辑
		if vm != nil && vm.IsTemplate == 0 {
			s.logger.WithContext(ctx).Info("cleaning up temp vm record (cannot verify template status)",
				zap.Uint32("vmid", vmid),
				zap.Int64("node_id", node.Id))
			return s.vmRepo.Delete(ctx, vm.Id)
		}
		return nil
	}

	// 4. 检查配置中的 template 字段，确认是否为模板
	isTemplateInProxmox := false
	if templateVal, ok := vmConfig["template"]; ok {
		// template 字段可能是 1, "1", 或者 boolean true
		switch v := templateVal.(type) {
		case int:
			isTemplateInProxmox = v == 1
		case int64:
			isTemplateInProxmox = v == 1
		case float64:
			isTemplateInProxmox = v == 1
		case string:
			isTemplateInProxmox = v == "1" || v == "true"
		case bool:
			isTemplateInProxmox = v
		}
	}

	// 5. 如果 Proxmox 中确实是模板
	if isTemplateInProxmox {
		// 5.1 如果数据库中有 isTemplate=0 的记录，删除它（脏数据）
		if vm != nil && vm.IsTemplate == 0 {
			s.logger.WithContext(ctx).Info("cleaning up temp vm record that was converted to template",
				zap.Uint32("vmid", vmid),
				zap.Int64("node_id", node.Id),
				zap.String("vm_name", vm.VmName),
				zap.Int64("vm_db_id", vm.Id))

			if err := s.vmRepo.Delete(ctx, vm.Id); err != nil {
				return fmt.Errorf("failed to delete temp vm record: %w", err)
			}

			s.logger.WithContext(ctx).Info("temp vm record cleaned up successfully",
				zap.Uint32("vmid", vmid),
				zap.Int64("node_id", node.Id))
		}

		// 5.2 如果数据库中没有记录或记录不正确，创建正确的模板记录
		// 这样即使上报系统还没上报，也能确保数据库中有正确的记录
		if vm == nil || vm.IsTemplate == 0 {
			// 获取集群信息（用于创建记录）
			cluster, err := s.clusterRepo.GetByID(ctx, node.ClusterID)
			if err != nil || cluster == nil {
				s.logger.WithContext(ctx).Warn("failed to get cluster for template record creation",
					zap.Error(err),
					zap.Int64("cluster_id", node.ClusterID))
				return nil // 不中断，让上报系统后续处理
			}

			// 从配置中提取其他信息
			cpuNum := 0
			memorySize := 0
			if cpu, ok := vmConfig["cores"]; ok {
				if cpuInt, ok := cpu.(float64); ok {
					cpuNum = int(cpuInt)
				}
			}
			if memory, ok := vmConfig["memory"]; ok {
				if memInt, ok := memory.(float64); ok {
					memorySize = int(memInt) // MB
				}
			}

			// 创建模板记录
			templateVM := &model.PveVM{
				VMID:         vmid,
				VmName:       vmName,
				NodeID:       node.Id,
				ClusterID:    node.ClusterID,
				IsTemplate:   1,         // 标记为模板
				Status:       "stopped", // 模板通常是停止状态
				CPUNum:       cpuNum,
				MemorySize:   memorySize,
				CreateTime:   time.Now(),
				UpdateTime:   time.Now(),
				LastSyncTime: time.Now(),
			}

			// 计算资源 hash
			resourceHash, err := s.calculateVMResourceHash(templateVM)
			if err == nil {
				templateVM.ResourceHash = resourceHash
			}

			// 使用 Upsert 创建或更新记录
			if err := s.vmRepo.Upsert(ctx, templateVM); err != nil {
				s.logger.WithContext(ctx).Warn("failed to create template record",
					zap.Error(err),
					zap.Uint32("vmid", vmid))
				return nil // 不中断，让上报系统后续处理
			}

			s.logger.WithContext(ctx).Info("template record created/updated successfully",
				zap.Uint32("vmid", vmid),
				zap.String("vm_name", vmName),
				zap.Int64("node_id", node.Id))
		}
	}

	return nil
}

// calculateVMResourceHash 计算虚拟机的资源 hash（简化版本）
func (s *templateManagementService) calculateVMResourceHash(vm *model.PveVM) (string, error) {
	// 使用简单的 hash 计算（实际应该使用 pkg/hash 包，但这里为了简化直接使用 md5）
	hashStr := fmt.Sprintf("%d-%d-%d-%d-%s-%d",
		vm.VMID, vm.NodeID, vm.CPUNum, vm.MemorySize, vm.Status, vm.IsTemplate)

	hashBytes := md5.Sum([]byte(hashStr))
	return fmt.Sprintf("%x", hashBytes), nil
}
