package repository

import (
	"context"

	"pvesphere/internal/model"

	"gorm.io/gorm"
)

type TemplateUploadRepository interface {
	Create(ctx context.Context, upload *model.TemplateUpload) error
	Update(ctx context.Context, upload *model.TemplateUpload) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*model.TemplateUpload, error)
	GetByTemplateID(ctx context.Context, templateID int64) (*model.TemplateUpload, error)
	ListByClusterID(ctx context.Context, clusterID int64) ([]*model.TemplateUpload, error)
	UpdateStatus(ctx context.Context, id int64, status string, progress int, errorMsg string) error
}

func NewTemplateUploadRepository(
	repository *Repository,
) TemplateUploadRepository {
	return &templateUploadRepository{
		Repository: repository,
	}
}

type templateUploadRepository struct {
	*Repository
}

func (r *templateUploadRepository) Create(ctx context.Context, upload *model.TemplateUpload) error {
	return r.DB(ctx).Create(upload).Error
}

func (r *templateUploadRepository) Update(ctx context.Context, upload *model.TemplateUpload) error {
	return r.DB(ctx).Save(upload).Error
}

func (r *templateUploadRepository) Delete(ctx context.Context, id int64) error {
	return r.DB(ctx).Delete(&model.TemplateUpload{}, id).Error
}

func (r *templateUploadRepository) GetByID(ctx context.Context, id int64) (*model.TemplateUpload, error) {
	var upload model.TemplateUpload
	err := r.DB(ctx).Where("id = ?", id).First(&upload).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &upload, nil
}

func (r *templateUploadRepository) GetByTemplateID(ctx context.Context, templateID int64) (*model.TemplateUpload, error) {
	var upload model.TemplateUpload
	err := r.DB(ctx).Where("template_id = ?", templateID).
		Order("gmt_create DESC").
		First(&upload).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &upload, nil
}

func (r *templateUploadRepository) ListByClusterID(ctx context.Context, clusterID int64) ([]*model.TemplateUpload, error) {
	var uploads []*model.TemplateUpload
	err := r.DB(ctx).Where("cluster_id = ?", clusterID).
		Order("gmt_create DESC").
		Find(&uploads).Error
	if err != nil {
		return nil, err
	}
	return uploads, nil
}

func (r *templateUploadRepository) UpdateStatus(ctx context.Context, id int64, status string, progress int, errorMsg string) error {
	updates := map[string]interface{}{
		"status":          status,
		"import_progress": progress,
	}
	if errorMsg != "" {
		updates["error_message"] = errorMsg
	}
	return r.DB(ctx).Model(&model.TemplateUpload{}).
		Where("id = ?", id).
		Updates(updates).Error
}

