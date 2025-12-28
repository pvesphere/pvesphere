package repository

import (
	"context"
	"errors"
	"pvesphere/internal/model"

	"gorm.io/gorm"
)

type VmTemplateRepository interface {
	Create(ctx context.Context, template *model.VmTemplate) error
	Update(ctx context.Context, template *model.VmTemplate) error
	GetByID(ctx context.Context, id int64) (*model.VmTemplate, error)
	GetByTemplateName(ctx context.Context, templateName string, clusterID int64) (*model.VmTemplate, error)
	GetByClusterID(ctx context.Context, clusterID int64) ([]*model.VmTemplate, error)
	GetByIDs(ctx context.Context, ids []int64) (map[int64]*model.VmTemplate, error) // 批量查询模板，返回 map[id]*template
}

func NewVmTemplateRepository(r *Repository) VmTemplateRepository {
	return &vmTemplateRepository{Repository: r}
}

type vmTemplateRepository struct {
	*Repository
}

func (r *vmTemplateRepository) Create(ctx context.Context, template *model.VmTemplate) error {
	return r.DB(ctx).Create(template).Error
}

func (r *vmTemplateRepository) Update(ctx context.Context, template *model.VmTemplate) error {
	return r.DB(ctx).Save(template).Error
}

func (r *vmTemplateRepository) GetByID(ctx context.Context, id int64) (*model.VmTemplate, error) {
	var template model.VmTemplate
	if err := r.DB(ctx).Where("id = ?", id).First(&template).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &template, nil
}

func (r *vmTemplateRepository) GetByTemplateName(ctx context.Context, templateName string, clusterID int64) (*model.VmTemplate, error) {
	var template model.VmTemplate
	if err := r.DB(ctx).Where("template_name = ? AND cluster_id = ?", templateName, clusterID).First(&template).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &template, nil
}

func (r *vmTemplateRepository) GetByClusterID(ctx context.Context, clusterID int64) ([]*model.VmTemplate, error) {
	var templates []*model.VmTemplate
	if err := r.DB(ctx).Where("cluster_id = ?", clusterID).Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

// GetByIDs 批量查询模板，返回 map[id]*template，用于批量填充名称
func (r *vmTemplateRepository) GetByIDs(ctx context.Context, ids []int64) (map[int64]*model.VmTemplate, error) {
	if len(ids) == 0 {
		return make(map[int64]*model.VmTemplate), nil
	}

	var templates []*model.VmTemplate
	if err := r.DB(ctx).Where("id IN ?", ids).Find(&templates).Error; err != nil {
		return nil, err
	}

	result := make(map[int64]*model.VmTemplate, len(templates))
	for _, template := range templates {
		result[template.Id] = template
	}
	return result, nil
}
