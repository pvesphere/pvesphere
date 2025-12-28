package repository

import (
	"context"
	"errors"
	"pvesphere/internal/model"
	"time"

	"gorm.io/gorm"
)

type PveStorageRepository interface {
	Create(ctx context.Context, storage *model.PveStorage) error
	Update(ctx context.Context, storage *model.PveStorage) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*model.PveStorage, error)
	GetByStorageName(ctx context.Context, storageName string, nodeName string, clusterID int64) (*model.PveStorage, error)
	ListByStorageName(ctx context.Context, clusterID int64, storageName string) ([]*model.PveStorage, error)
	GetByClusterID(ctx context.Context, clusterID int64) ([]*model.PveStorage, error)
	ListWithPagination(ctx context.Context, page, pageSize int, clusterID int64, nodeName, storageType, storageName string) ([]*model.PveStorage, int64, error)
	Upsert(ctx context.Context, storage *model.PveStorage) error
	DeleteByStorageName(ctx context.Context, storageName string, nodeName string, clusterID int64) error
	GetHashByStorageName(ctx context.Context, storageName string, nodeName string, clusterID int64) (string, int64, error)
	UpdateSyncTimeOnly(ctx context.Context, id int64) error
}

func NewPveStorageRepository(r *Repository) PveStorageRepository {
	return &pveStorageRepository{Repository: r}
}

type pveStorageRepository struct {
	*Repository
}

func (r *pveStorageRepository) Create(ctx context.Context, storage *model.PveStorage) error {
	return r.DB(ctx).Create(storage).Error
}

func (r *pveStorageRepository) Update(ctx context.Context, storage *model.PveStorage) error {
	return r.DB(ctx).Save(storage).Error
}

func (r *pveStorageRepository) GetByID(ctx context.Context, id int64) (*model.PveStorage, error) {
	var storage model.PveStorage
	if err := r.DB(ctx).Where("id = ?", id).First(&storage).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &storage, nil
}

func (r *pveStorageRepository) GetByStorageName(ctx context.Context, storageName string, nodeName string, clusterID int64) (*model.PveStorage, error) {
	var storage model.PveStorage
	if err := r.DB(ctx).Where("storage_name = ? AND node_name = ? AND cluster_id = ?", storageName, nodeName, clusterID).First(&storage).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &storage, nil
}

func (r *pveStorageRepository) GetByClusterID(ctx context.Context, clusterID int64) ([]*model.PveStorage, error) {
	var storages []*model.PveStorage
	if err := r.DB(ctx).Where("cluster_id = ?", clusterID).Find(&storages).Error; err != nil {
		return nil, err
	}
	return storages, nil
}

func (r *pveStorageRepository) ListByStorageName(ctx context.Context, clusterID int64, storageName string) ([]*model.PveStorage, error) {
	var storages []*model.PveStorage
	if err := r.DB(ctx).Where("cluster_id = ? AND storage_name = ?", clusterID, storageName).Find(&storages).Error; err != nil {
		return nil, err
	}
	return storages, nil
}

func (r *pveStorageRepository) Upsert(ctx context.Context, storage *model.PveStorage) error {
	// 先查询是否存在以及 hash
	existingHash, existingID, err := r.GetHashByStorageName(ctx, storage.StorageName, storage.NodeName, storage.ClusterID)
	if err != nil {
		return err
	}

	// 如果不存在，创建新记录
	if existingID == 0 {
		return r.Create(ctx, storage)
	}

	// 如果 hash 相同，只更新同步时间（轻量级更新）
	if existingHash != "" && existingHash == storage.ResourceHash {
		storage.Id = existingID
		return r.UpdateSyncTimeOnly(ctx, existingID)
	}

	// hash 不同，完整更新记录
	storage.Id = existingID
	return r.Update(ctx, storage)
}

func (r *pveStorageRepository) GetHashByStorageName(ctx context.Context, storageName string, nodeName string, clusterID int64) (string, int64, error) {
	var result struct {
		Id           int64  `gorm:"column:id"`
		ResourceHash string `gorm:"column:resource_hash"`
	}

	err := r.DB(ctx).
		Table("pve_storage").
		Select("id, resource_hash").
		Where("storage_name = ? AND node_name = ? AND cluster_id = ?", storageName, nodeName, clusterID).
		First(&result).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", 0, nil
	}
	if err != nil {
		return "", 0, err
	}

	return result.ResourceHash, result.Id, nil
}

func (r *pveStorageRepository) UpdateSyncTimeOnly(ctx context.Context, id int64) error {
	return r.DB(ctx).
		Model(&model.PveStorage{}).
		Where("id = ?", id).
		Update("last_sync_time", time.Now()).Error
}

func (r *pveStorageRepository) DeleteByStorageName(ctx context.Context, storageName string, nodeName string, clusterID int64) error {
	return r.DB(ctx).Where("storage_name = ? AND node_name = ? AND cluster_id = ?", storageName, nodeName, clusterID).Delete(&model.PveStorage{}).Error
}

func (r *pveStorageRepository) Delete(ctx context.Context, id int64) error {
	return r.DB(ctx).Where("id = ?", id).Delete(&model.PveStorage{}).Error
}

func (r *pveStorageRepository) ListWithPagination(ctx context.Context, page, pageSize int, clusterID int64, nodeName, storageType, storageName string) ([]*model.PveStorage, int64, error) {
	var storages []*model.PveStorage
	var total int64

	query := r.DB(ctx).Model(&model.PveStorage{})

	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}
	if nodeName != "" {
		query = query.Where("node_name = ?", nodeName)
	}
	if storageType != "" {
		query = query.Where("type = ?", storageType)
	}
	if storageName != "" {
		query = query.Where("storage_name = ?", storageName)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&storages).Error; err != nil {
		return nil, 0, err
	}

	return storages, total, nil
}
