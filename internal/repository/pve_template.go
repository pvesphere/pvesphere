package repository

import (
	"context"
	"errors"
	"pvesphere/internal/model"

	"gorm.io/gorm"
)

// PveTemplateRepository 模板仓储
type PveTemplateRepository interface {
	Create(ctx context.Context, tpl *model.PveTemplate) error
	Update(ctx context.Context, tpl *model.PveTemplate) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*model.PveTemplate, error)
	ListWithPagination(ctx context.Context, page, pageSize int, clusterID int64) ([]*model.PveTemplate, int64, error)
}

func NewPveTemplateRepository(r *Repository) PveTemplateRepository {
	return &pveTemplateRepository{Repository: r}
}

type pveTemplateRepository struct {
	*Repository
}

func (r *pveTemplateRepository) Create(ctx context.Context, tpl *model.PveTemplate) error {
	return r.DB(ctx).Create(tpl).Error
}

func (r *pveTemplateRepository) Update(ctx context.Context, tpl *model.PveTemplate) error {
	return r.DB(ctx).Save(tpl).Error
}

func (r *pveTemplateRepository) Delete(ctx context.Context, id int64) error {
	return r.DB(ctx).Where("id = ?", id).Delete(&model.PveTemplate{}).Error
}

func (r *pveTemplateRepository) GetByID(ctx context.Context, id int64) (*model.PveTemplate, error) {
	var tpl model.PveTemplate
	if err := r.DB(ctx).Where("id = ?", id).First(&tpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tpl, nil
}

func (r *pveTemplateRepository) ListWithPagination(ctx context.Context, page, pageSize int, clusterID int64) ([]*model.PveTemplate, int64, error) {
	var tpls []*model.PveTemplate
	var total int64

	query := r.DB(ctx).Model(&model.PveTemplate{})
	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&tpls).Error; err != nil {
		return nil, 0, err
	}

	return tpls, total, nil
}
