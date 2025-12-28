package repository

import (
	"context"
	"errors"
	"pvesphere/internal/model"

	"gorm.io/gorm"
)

type PveClusterRepository interface {
	Create(ctx context.Context, cluster *model.PveCluster) error
	Update(ctx context.Context, cluster *model.PveCluster) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*model.PveCluster, error)
	GetByClusterName(ctx context.Context, clusterName string) (*model.PveCluster, error)
	List(ctx context.Context) ([]*model.PveCluster, error)
	ListWithPagination(ctx context.Context, page, pageSize int, env, region string) ([]*model.PveCluster, int64, error)
	GetAllSchedulable(ctx context.Context) ([]*model.PveCluster, error)
	GetAllEnabled(ctx context.Context) ([]*model.PveCluster, error) // 获取所有启用的集群（用于数据自动上报）
	GetByIDs(ctx context.Context, ids []int64) (map[int64]*model.PveCluster, error) // 批量查询集群，返回 map[id]*cluster
}

func NewPveClusterRepository(r *Repository) PveClusterRepository {
	return &pveClusterRepository{Repository: r}
}

type pveClusterRepository struct {
	*Repository
}

func (r *pveClusterRepository) Create(ctx context.Context, cluster *model.PveCluster) error {
	return r.DB(ctx).Create(cluster).Error
}

func (r *pveClusterRepository) Update(ctx context.Context, cluster *model.PveCluster) error {
	return r.DB(ctx).Save(cluster).Error
}

func (r *pveClusterRepository) GetByID(ctx context.Context, id int64) (*model.PveCluster, error) {
	var cluster model.PveCluster
	if err := r.DB(ctx).Where("id = ?", id).First(&cluster).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cluster, nil
}

func (r *pveClusterRepository) GetByClusterName(ctx context.Context, clusterName string) (*model.PveCluster, error) {
	var cluster model.PveCluster
	if err := r.DB(ctx).Where("cluster_name = ?", clusterName).First(&cluster).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cluster, nil
}

func (r *pveClusterRepository) List(ctx context.Context) ([]*model.PveCluster, error) {
	var clusters []*model.PveCluster
	if err := r.DB(ctx).Find(&clusters).Error; err != nil {
		return nil, err
	}
	return clusters, nil
}

func (r *pveClusterRepository) GetAllSchedulable(ctx context.Context) ([]*model.PveCluster, error) {
	var clusters []*model.PveCluster
	if err := r.DB(ctx).Where("is_schedulable = ?", 1).Find(&clusters).Error; err != nil {
		return nil, err
	}
	return clusters, nil
}

func (r *pveClusterRepository) GetAllEnabled(ctx context.Context) ([]*model.PveCluster, error) {
	var clusters []*model.PveCluster
	if err := r.DB(ctx).Where("is_enabled = ?", 1).Find(&clusters).Error; err != nil {
		return nil, err
	}
	return clusters, nil
}

func (r *pveClusterRepository) Delete(ctx context.Context, id int64) error {
	return r.DB(ctx).Where("id = ?", id).Delete(&model.PveCluster{}).Error
}

func (r *pveClusterRepository) ListWithPagination(ctx context.Context, page, pageSize int, env, region string) ([]*model.PveCluster, int64, error) {
	var clusters []*model.PveCluster
	var total int64

	query := r.DB(ctx).Model(&model.PveCluster{})

	if env != "" {
		query = query.Where("env = ?", env)
	}
	if region != "" {
		query = query.Where("region = ?", region)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&clusters).Error; err != nil {
		return nil, 0, err
	}

	return clusters, total, nil
}

// GetByIDs 批量查询集群，返回 map[id]*cluster，用于批量填充名称
func (r *pveClusterRepository) GetByIDs(ctx context.Context, ids []int64) (map[int64]*model.PveCluster, error) {
	if len(ids) == 0 {
		return make(map[int64]*model.PveCluster), nil
	}

	var clusters []*model.PveCluster
	if err := r.DB(ctx).Where("id IN ?", ids).Find(&clusters).Error; err != nil {
		return nil, err
	}

	result := make(map[int64]*model.PveCluster, len(clusters))
	for _, cluster := range clusters {
		result[cluster.Id] = cluster
	}
	return result, nil
}
