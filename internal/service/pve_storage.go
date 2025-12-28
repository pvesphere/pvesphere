package service

import (
	"context"
	"time"

	v1 "pvesphere/api/v1"
	"pvesphere/internal/model"
	"pvesphere/internal/repository"
	"pvesphere/pkg/log"

	"go.uber.org/zap"
)

type PveStorageService interface {
	CreateStorage(ctx context.Context, req *v1.CreateStorageRequest) error
	UpdateStorage(ctx context.Context, id int64, req *v1.UpdateStorageRequest) error
	DeleteStorage(ctx context.Context, id int64) error
	GetStorage(ctx context.Context, id int64) (*v1.StorageDetail, error)
	ListStorages(ctx context.Context, req *v1.ListStorageRequest) (*v1.ListStorageResponseData, error)
}

func NewPveStorageService(
	service *Service,
	storageRepo repository.PveStorageRepository,
	nodeRepo repository.PveNodeRepository,
	logger *log.Logger,
) PveStorageService {
	return &pveStorageService{
		storageRepo: storageRepo,
		nodeRepo:    nodeRepo,
		Service:     service,
		logger:      logger,
	}
}

type pveStorageService struct {
	storageRepo repository.PveStorageRepository
	nodeRepo    repository.PveNodeRepository
	*Service
	logger *log.Logger
}

func (s *pveStorageService) CreateStorage(ctx context.Context, req *v1.CreateStorageRequest) error {
	// 检查存储是否已存在
	existing, err := s.storageRepo.GetByStorageName(ctx, req.StorageName, req.NodeName, req.ClusterID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to check storage", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if existing != nil {
		return v1.ErrBadRequest
	}

	storage := &model.PveStorage{
		NodeName:     req.NodeName,
		ClusterID:    req.ClusterID,
		Active:       req.Active,
		Type:         req.Type,
		Avail:        req.Avail,
		StorageName:  req.StorageName,
		Content:      req.Content,
		Used:         req.Used,
		Total:        req.Total,
		Enabled:      req.Enabled,
		UsedFraction: req.UsedFraction,
		Shared:       req.Shared,
		CreateTime:   time.Now(),
		UpdateTime:   time.Now(),
	}

	if err := s.storageRepo.Create(ctx, storage); err != nil {
		s.logger.WithContext(ctx).Error("failed to create storage", zap.Error(err))
		return v1.ErrInternalServerError
	}

	return nil
}

func (s *pveStorageService) UpdateStorage(ctx context.Context, id int64, req *v1.UpdateStorageRequest) error {
	storage, err := s.storageRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get storage", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if storage == nil {
		return v1.ErrNotFound
	}

	// 更新字段
	if req.Active != nil {
		storage.Active = *req.Active
	}
	if req.Type != nil {
		storage.Type = *req.Type
	}
	if req.Avail != nil {
		storage.Avail = *req.Avail
	}
	if req.Content != nil {
		storage.Content = *req.Content
	}
	if req.Used != nil {
		storage.Used = *req.Used
	}
	if req.Total != nil {
		storage.Total = *req.Total
	}
	if req.Enabled != nil {
		storage.Enabled = *req.Enabled
	}
	if req.UsedFraction != nil {
		storage.UsedFraction = *req.UsedFraction
	}
	if req.Shared != nil {
		storage.Shared = *req.Shared
	}
	storage.UpdateTime = time.Now()

	if err := s.storageRepo.Update(ctx, storage); err != nil {
		s.logger.WithContext(ctx).Error("failed to update storage", zap.Error(err))
		return v1.ErrInternalServerError
	}

	return nil
}

func (s *pveStorageService) DeleteStorage(ctx context.Context, id int64) error {
	storage, err := s.storageRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get storage", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if storage == nil {
		return v1.ErrNotFound
	}

	if err := s.storageRepo.Delete(ctx, id); err != nil {
		s.logger.WithContext(ctx).Error("failed to delete storage", zap.Error(err))
		return v1.ErrInternalServerError
	}

	return nil
}

func (s *pveStorageService) GetStorage(ctx context.Context, id int64) (*v1.StorageDetail, error) {
	storage, err := s.storageRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get storage", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	if storage == nil {
		return nil, v1.ErrNotFound
	}

	// 查询节点 ID
	var nodeID int64
	node, err := s.nodeRepo.GetByNodeName(ctx, storage.NodeName, storage.ClusterID)
	if err != nil {
		s.logger.WithContext(ctx).Warn("failed to get node for storage",
			zap.Error(err),
			zap.String("node_name", storage.NodeName),
			zap.Int64("cluster_id", storage.ClusterID))
	} else if node != nil {
		nodeID = node.Id
	}

	return &v1.StorageDetail{
		Id:           storage.Id,
		NodeName:     storage.NodeName,
		NodeID:       nodeID,
		ClusterID:    storage.ClusterID,
		Active:       storage.Active,
		Type:         storage.Type,
		Avail:        storage.Avail,
		StorageName:  storage.StorageName,
		Content:      storage.Content,
		Used:         storage.Used,
		Total:        storage.Total,
		Enabled:      storage.Enabled,
		UsedFraction: storage.UsedFraction,
		Shared:       storage.Shared,
		CreateTime:   storage.CreateTime,
		UpdateTime:   storage.UpdateTime,
		Creator:      storage.Creator,
		Modifier:     storage.Modifier,
	}, nil
}

func (s *pveStorageService) ListStorages(ctx context.Context, req *v1.ListStorageRequest) (*v1.ListStorageResponseData, error) {
	storages, total, err := s.storageRepo.ListWithPagination(ctx, req.Page, req.PageSize, req.ClusterID, req.NodeName, req.Type, req.StorageName)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to list storages", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}

	// 构建节点名称到节点 ID 的映射（批量查询优化）
	type nodeKey struct {
		nodeName  string
		clusterID int64
	}
	nodeIDMap := make(map[nodeKey]int64)

	// 收集所有唯一的 (node_name, cluster_id) 组合
	nodeKeys := make(map[nodeKey]struct{})
	for _, storage := range storages {
		nodeKeys[nodeKey{nodeName: storage.NodeName, clusterID: storage.ClusterID}] = struct{}{}
	}

	// 批量查询节点信息
	for key := range nodeKeys {
		node, err := s.nodeRepo.GetByNodeName(ctx, key.nodeName, key.clusterID)
		if err != nil {
			s.logger.WithContext(ctx).Warn("failed to get node for storage mapping",
				zap.Error(err),
				zap.String("node_name", key.nodeName),
				zap.Int64("cluster_id", key.clusterID))
			continue
		}
		if node != nil {
			nodeIDMap[key] = node.Id
		}
	}

	items := make([]v1.StorageItem, 0, len(storages))
	for _, storage := range storages {
		nodeIDKey := nodeKey{nodeName: storage.NodeName, clusterID: storage.ClusterID}
		nodeID := nodeIDMap[nodeIDKey]

		items = append(items, v1.StorageItem{
			Id:           storage.Id,
			NodeName:     storage.NodeName,
			NodeID:       nodeID,
			ClusterID:    storage.ClusterID,
			Active:       storage.Active,
			Type:         storage.Type,
			Avail:        storage.Avail,
			StorageName:  storage.StorageName,
			Content:      storage.Content,
			Used:         storage.Used,
			Total:        storage.Total,
			Enabled:      storage.Enabled,
			UsedFraction: storage.UsedFraction,
			Shared:       storage.Shared,
		})
	}

	return &v1.ListStorageResponseData{
		Total: total,
		List:  items,
	}, nil
}
