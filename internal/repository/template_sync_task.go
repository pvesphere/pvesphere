package repository

import (
	"context"
	"time"

	"pvesphere/internal/model"

	"gorm.io/gorm"
)

type TemplateSyncTaskRepository interface {
	Create(ctx context.Context, task *model.TemplateSyncTask) error
	Update(ctx context.Context, task *model.TemplateSyncTask) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*model.TemplateSyncTask, error)
	ListByTemplateID(ctx context.Context, templateID int64) ([]*model.TemplateSyncTask, error)
	ListByStatus(ctx context.Context, status string) ([]*model.TemplateSyncTask, error)
	ListWithPagination(ctx context.Context, page, pageSize int, templateID *int64, status string) ([]*model.TemplateSyncTask, int64, error)
	UpdateStatus(ctx context.Context, id int64, status string, progress int, errorMsg string) error
	UpdateSyncTime(ctx context.Context, id int64, startTime, endTime *time.Time) error
	GetPendingTasks(ctx context.Context, limit int) ([]*model.TemplateSyncTask, error)
}

func NewTemplateSyncTaskRepository(
	repository *Repository,
) TemplateSyncTaskRepository {
	return &templateSyncTaskRepository{
		Repository: repository,
	}
}

type templateSyncTaskRepository struct {
	*Repository
}

func (r *templateSyncTaskRepository) Create(ctx context.Context, task *model.TemplateSyncTask) error {
	return r.DB(ctx).Create(task).Error
}

func (r *templateSyncTaskRepository) Update(ctx context.Context, task *model.TemplateSyncTask) error {
	return r.DB(ctx).Save(task).Error
}

func (r *templateSyncTaskRepository) Delete(ctx context.Context, id int64) error {
	return r.DB(ctx).Delete(&model.TemplateSyncTask{}, id).Error
}

func (r *templateSyncTaskRepository) GetByID(ctx context.Context, id int64) (*model.TemplateSyncTask, error) {
	var task model.TemplateSyncTask
	err := r.DB(ctx).Where("id = ?", id).First(&task).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *templateSyncTaskRepository) ListByTemplateID(ctx context.Context, templateID int64) ([]*model.TemplateSyncTask, error) {
	var tasks []*model.TemplateSyncTask
	err := r.DB(ctx).Where("template_id = ?", templateID).
		Order("gmt_create DESC").
		Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *templateSyncTaskRepository) ListByStatus(ctx context.Context, status string) ([]*model.TemplateSyncTask, error) {
	var tasks []*model.TemplateSyncTask
	err := r.DB(ctx).Where("status = ?", status).
		Order("gmt_create ASC").
		Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *templateSyncTaskRepository) ListWithPagination(ctx context.Context, page, pageSize int, templateID *int64, status string) ([]*model.TemplateSyncTask, int64, error) {
	var tasks []*model.TemplateSyncTask
	var total int64

	query := r.DB(ctx).Model(&model.TemplateSyncTask{})

	// 条件过滤
	if templateID != nil && *templateID > 0 {
		query = query.Where("template_id = ?", *templateID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Order("gmt_create DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

func (r *templateSyncTaskRepository) UpdateStatus(ctx context.Context, id int64, status string, progress int, errorMsg string) error {
	updates := map[string]interface{}{
		"status":   status,
		"progress": progress,
	}
	if errorMsg != "" {
		updates["error_message"] = errorMsg
	}
	return r.DB(ctx).Model(&model.TemplateSyncTask{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *templateSyncTaskRepository) UpdateSyncTime(ctx context.Context, id int64, startTime, endTime *time.Time) error {
	updates := map[string]interface{}{}
	if startTime != nil {
		updates["sync_start_time"] = startTime
	}
	if endTime != nil {
		updates["sync_end_time"] = endTime
	}
	if len(updates) == 0 {
		return nil
	}
	return r.DB(ctx).Model(&model.TemplateSyncTask{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *templateSyncTaskRepository) GetPendingTasks(ctx context.Context, limit int) ([]*model.TemplateSyncTask, error) {
	var tasks []*model.TemplateSyncTask
	err := r.DB(ctx).Where("status = ?", model.TemplateSyncTaskStatusPending).
		Order("gmt_create ASC").
		Limit(limit).
		Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

