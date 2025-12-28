package repository

import (
	"context"
	"errors"
	"pvesphere/internal/model"
	"time"

	"gorm.io/gorm"
)

type PveNodeRepository interface {
	Create(ctx context.Context, node *model.PveNode) error
	Update(ctx context.Context, node *model.PveNode) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*model.PveNode, error)
	GetByNodeName(ctx context.Context, nodeName string, clusterID int64) (*model.PveNode, error)
	GetByClusterID(ctx context.Context, clusterID int64) ([]*model.PveNode, error)
	ListWithPagination(ctx context.Context, page, pageSize int, clusterID int64, env, status string) ([]*model.PveNode, int64, error)
	Upsert(ctx context.Context, node *model.PveNode) error
	DeleteByNodeName(ctx context.Context, nodeName string, clusterID int64) error
	GetHashByNodeName(ctx context.Context, nodeName string, clusterID int64) (string, int64, error) // 返回 hash 和 id
	UpdateSyncTimeOnly(ctx context.Context, id int64) error
	GetByIDs(ctx context.Context, ids []int64) (map[int64]*model.PveNode, error) // 批量查询节点，返回 map[id]*node
}

func NewPveNodeRepository(r *Repository) PveNodeRepository {
	return &pveNodeRepository{Repository: r}
}

type pveNodeRepository struct {
	*Repository
}

func (r *pveNodeRepository) Create(ctx context.Context, node *model.PveNode) error {
	return r.DB(ctx).Create(node).Error
}

func (r *pveNodeRepository) Update(ctx context.Context, node *model.PveNode) error {
	return r.DB(ctx).Save(node).Error
}

func (r *pveNodeRepository) GetByID(ctx context.Context, id int64) (*model.PveNode, error) {
	var node model.PveNode
	if err := r.DB(ctx).Where("id = ?", id).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &node, nil
}

func (r *pveNodeRepository) GetByNodeName(ctx context.Context, nodeName string, clusterID int64) (*model.PveNode, error) {
	var node model.PveNode
	if err := r.DB(ctx).Where("node_name = ? AND cluster_id = ?", nodeName, clusterID).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &node, nil
}

func (r *pveNodeRepository) GetByClusterID(ctx context.Context, clusterID int64) ([]*model.PveNode, error) {
	var nodes []*model.PveNode
	if err := r.DB(ctx).Where("cluster_id = ?", clusterID).Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

func (r *pveNodeRepository) Upsert(ctx context.Context, node *model.PveNode) error {
	// 先查询是否存在以及 hash
	existingHash, existingID, err := r.GetHashByNodeName(ctx, node.NodeName, node.ClusterID)
	if err != nil {
		return err
	}

	// 计算新资源的 hash
	if node.ResourceHash == "" {
		// 如果没有提供 hash，需要从外部计算（由调用者提供）
		// 这里假设 hash 已经在 Handler 中计算好了
	}

	// 如果不存在，创建新记录
	if existingID == 0 {
		return r.Create(ctx, node)
	}

	// 如果 hash 相同，只更新同步时间（轻量级更新）
	if existingHash != "" && existingHash == node.ResourceHash {
		node.Id = existingID
		return r.UpdateSyncTimeOnly(ctx, existingID)
	}

	// hash 不同，完整更新记录
	node.Id = existingID
	return r.Update(ctx, node)
}

func (r *pveNodeRepository) GetHashByNodeName(ctx context.Context, nodeName string, clusterID int64) (string, int64, error) {
	var result struct {
		Id           int64  `gorm:"column:id"`
		ResourceHash string `gorm:"column:resource_hash"`
	}

	err := r.DB(ctx).
		Table("pve_node").
		Select("id, resource_hash").
		Where("node_name = ? AND cluster_id = ?", nodeName, clusterID).
		First(&result).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", 0, nil
	}
	if err != nil {
		return "", 0, err
	}

	return result.ResourceHash, result.Id, nil
}

func (r *pveNodeRepository) UpdateSyncTimeOnly(ctx context.Context, id int64) error {
	return r.DB(ctx).
		Model(&model.PveNode{}).
		Where("id = ?", id).
		Update("last_sync_time", time.Now()).Error
}

func (r *pveNodeRepository) DeleteByNodeName(ctx context.Context, nodeName string, clusterID int64) error {
	return r.DB(ctx).Where("node_name = ? AND cluster_id = ?", nodeName, clusterID).Delete(&model.PveNode{}).Error
}

func (r *pveNodeRepository) Delete(ctx context.Context, id int64) error {
	return r.DB(ctx).Where("id = ?", id).Delete(&model.PveNode{}).Error
}

func (r *pveNodeRepository) ListWithPagination(ctx context.Context, page, pageSize int, clusterID int64, env, status string) ([]*model.PveNode, int64, error) {
	var nodes []*model.PveNode
	var total int64

	query := r.DB(ctx).Model(&model.PveNode{})

	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}
	if env != "" {
		query = query.Where("env = ?", env)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&nodes).Error; err != nil {
		return nil, 0, err
	}

	return nodes, total, nil
}

// GetByIDs 批量查询节点，返回 map[id]*node，用于批量填充名称
func (r *pveNodeRepository) GetByIDs(ctx context.Context, ids []int64) (map[int64]*model.PveNode, error) {
	if len(ids) == 0 {
		return make(map[int64]*model.PveNode), nil
	}

	var nodes []*model.PveNode
	if err := r.DB(ctx).Where("id IN ?", ids).Find(&nodes).Error; err != nil {
		return nil, err
	}

	result := make(map[int64]*model.PveNode, len(nodes))
	for _, node := range nodes {
		result[node.Id] = node
	}
	return result, nil
}
