package repository

import (
	"context"

	"pvesphere/internal/model"

	"gorm.io/gorm"
)

type TemplateInstanceRepository interface {
	Create(ctx context.Context, instance *model.TemplateInstance) error
	Update(ctx context.Context, instance *model.TemplateInstance) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*model.TemplateInstance, error)
	GetByTemplateAndNode(ctx context.Context, templateID, nodeID int64) (*model.TemplateInstance, error)
	ListByTemplateID(ctx context.Context, templateID int64) ([]*model.TemplateInstance, error)
	ListByNodeID(ctx context.Context, nodeID int64) ([]*model.TemplateInstance, error)
	GetPrimaryInstance(ctx context.Context, templateID int64) (*model.TemplateInstance, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	UpdateSyncTask(ctx context.Context, id int64, syncTaskID int64) error
	DeleteByTemplateID(ctx context.Context, templateID int64) error
}

func NewTemplateInstanceRepository(
	repository *Repository,
) TemplateInstanceRepository {
	return &templateInstanceRepository{
		Repository: repository,
	}
}

type templateInstanceRepository struct {
	*Repository
}

func (r *templateInstanceRepository) Create(ctx context.Context, instance *model.TemplateInstance) error {
	return r.DB(ctx).Create(instance).Error
}

func (r *templateInstanceRepository) Update(ctx context.Context, instance *model.TemplateInstance) error {
	return r.DB(ctx).Save(instance).Error
}

func (r *templateInstanceRepository) Delete(ctx context.Context, id int64) error {
	return r.DB(ctx).Delete(&model.TemplateInstance{}, id).Error
}

func (r *templateInstanceRepository) GetByID(ctx context.Context, id int64) (*model.TemplateInstance, error) {
	var instance model.TemplateInstance
	err := r.DB(ctx).Where("id = ?", id).First(&instance).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (r *templateInstanceRepository) GetByTemplateAndNode(ctx context.Context, templateID, nodeID int64) (*model.TemplateInstance, error) {
	var instance model.TemplateInstance
	err := r.DB(ctx).Where("template_id = ? AND node_id = ?", templateID, nodeID).
		First(&instance).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (r *templateInstanceRepository) ListByTemplateID(ctx context.Context, templateID int64) ([]*model.TemplateInstance, error) {
	var instances []*model.TemplateInstance
	err := r.DB(ctx).Where("template_id = ?", templateID).
		Order("is_primary DESC, gmt_create ASC").
		Find(&instances).Error
	if err != nil {
		return nil, err
	}
	return instances, nil
}

func (r *templateInstanceRepository) ListByNodeID(ctx context.Context, nodeID int64) ([]*model.TemplateInstance, error) {
	var instances []*model.TemplateInstance
	err := r.DB(ctx).Where("node_id = ?", nodeID).
		Order("gmt_create DESC").
		Find(&instances).Error
	if err != nil {
		return nil, err
	}
	return instances, nil
}

func (r *templateInstanceRepository) GetPrimaryInstance(ctx context.Context, templateID int64) (*model.TemplateInstance, error) {
	var instance model.TemplateInstance
	err := r.DB(ctx).Where("template_id = ? AND is_primary = 1", templateID).
		First(&instance).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (r *templateInstanceRepository) UpdateStatus(ctx context.Context, id int64, status string) error {
	return r.DB(ctx).Model(&model.TemplateInstance{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *templateInstanceRepository) UpdateSyncTask(ctx context.Context, id int64, syncTaskID int64) error {
	return r.DB(ctx).Model(&model.TemplateInstance{}).
		Where("id = ?", id).
		Update("sync_task_id", syncTaskID).Error
}

func (r *templateInstanceRepository) DeleteByTemplateID(ctx context.Context, templateID int64) error {
	return r.DB(ctx).Where("template_id = ?", templateID).
		Delete(&model.TemplateInstance{}).Error
}

