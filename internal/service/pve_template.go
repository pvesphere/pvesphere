package service

import (
	"context"
	"time"

	v1 "pvesphere/api/v1"
	"pvesphere/internal/model"
	"pvesphere/internal/repository"
	"pvesphere/pkg/log"
	"pvesphere/pkg/proxmox"

	"go.uber.org/zap"
)

type PveTemplateService interface {
	CreateTemplate(ctx context.Context, req *v1.CreateTemplateRequest) error
	UpdateTemplate(ctx context.Context, id int64, req *v1.UpdateTemplateRequest) error
	DeleteTemplate(ctx context.Context, id int64) error
	GetTemplate(ctx context.Context, id int64) (*v1.TemplateDetail, error)
	ListTemplates(ctx context.Context, req *v1.ListTemplateRequest) (*v1.ListTemplateResponseData, error)
}

func NewPveTemplateService(
	service *Service,
	tplRepo repository.PveTemplateRepository,
	instanceRepo repository.TemplateInstanceRepository,
	syncTaskRepo repository.TemplateSyncTaskRepository,
	uploadRepo repository.TemplateUploadRepository,
	nodeRepo repository.PveNodeRepository,
	clusterRepo repository.PveClusterRepository,
	logger *log.Logger,
) PveTemplateService {
	return &pveTemplateService{
		tplRepo:      tplRepo,
		instanceRepo: instanceRepo,
		syncTaskRepo: syncTaskRepo,
		uploadRepo:   uploadRepo,
		nodeRepo:     nodeRepo,
		clusterRepo:  clusterRepo,
		Service:      service,
		logger:       logger,
	}
}

type pveTemplateService struct {
	tplRepo      repository.PveTemplateRepository
	instanceRepo repository.TemplateInstanceRepository
	syncTaskRepo repository.TemplateSyncTaskRepository
	uploadRepo   repository.TemplateUploadRepository
	nodeRepo     repository.PveNodeRepository
	clusterRepo  repository.PveClusterRepository
	*Service
	logger *log.Logger
}

func (s *pveTemplateService) CreateTemplate(ctx context.Context, req *v1.CreateTemplateRequest) error {
	tpl := &model.PveTemplate{
		TemplateName: req.TemplateName,
		ClusterID:    req.ClusterID,
		Description:  req.Description,
		CreateTime:   time.Now(),
		UpdateTime:   time.Now(),
	}

	if err := s.tplRepo.Create(ctx, tpl); err != nil {
		s.logger.WithContext(ctx).Error("failed to create template", zap.Error(err))
		return v1.ErrInternalServerError
	}
	return nil
}

func (s *pveTemplateService) UpdateTemplate(ctx context.Context, id int64, req *v1.UpdateTemplateRequest) error {
	tpl, err := s.tplRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get template", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if tpl == nil {
		return v1.ErrNotFound
	}

	if req.TemplateName != nil {
		tpl.TemplateName = *req.TemplateName
	}
	if req.Description != nil {
		tpl.Description = *req.Description
	}
	tpl.UpdateTime = time.Now()

	if err := s.tplRepo.Update(ctx, tpl); err != nil {
		s.logger.WithContext(ctx).Error("failed to update template", zap.Error(err))
		return v1.ErrInternalServerError
	}
	return nil
}

func (s *pveTemplateService) DeleteTemplate(ctx context.Context, id int64) error {
	tpl, err := s.tplRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get template", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if tpl == nil {
		return v1.ErrNotFound
	}

	// 1. 查询所有模板实例
	instances, err := s.instanceRepo.ListByTemplateID(ctx, id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to list template instances", zap.Error(err))
		return v1.ErrInternalServerError
	}

	// 2. 删除所有节点上的 Proxmox 模板
	for _, instance := range instances {
		if instance.VMID == 0 {
			s.logger.WithContext(ctx).Warn("instance has no VMID, skip deletion",
				zap.Int64("instance_id", instance.Id),
				zap.String("node_name", instance.NodeName))
			continue
		}

		// 获取节点信息
		node, err := s.nodeRepo.GetByID(ctx, instance.NodeID)
		if err != nil || node == nil {
			s.logger.WithContext(ctx).Error("failed to get node",
				zap.Error(err),
				zap.Int64("node_id", instance.NodeID))
			continue
		}

		// 获取集群信息
		cluster, err := s.clusterRepo.GetByID(ctx, node.ClusterID)
		if err != nil || cluster == nil {
			s.logger.WithContext(ctx).Error("failed to get cluster",
				zap.Error(err),
				zap.Int64("cluster_id", node.ClusterID))
			continue
		}

		// 创建 Proxmox 客户端
		client, err := proxmox.NewProxmoxClient(cluster.ApiUrl, cluster.UserId, cluster.UserToken)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to create proxmox client",
				zap.Error(err),
				zap.String("node_name", node.NodeName))
			continue
		}

		// 删除 Proxmox 模板（purge=true 表示同时删除磁盘）
		err = client.DeleteVM(ctx, node.NodeName, instance.VMID, true)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to delete proxmox template",
				zap.Error(err),
				zap.String("node_name", node.NodeName),
				zap.Uint32("vmid", instance.VMID))
			// 继续删除其他实例，不中断流程
		} else {
			s.logger.WithContext(ctx).Info("deleted proxmox template",
				zap.String("node_name", node.NodeName),
				zap.Uint32("vmid", instance.VMID))
		}
	}

	// 3. 删除所有同步任务
	syncTasks, err := s.syncTaskRepo.ListByTemplateID(ctx, id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to list sync tasks", zap.Error(err))
		// 继续执行，不中断流程
	} else {
		for _, task := range syncTasks {
			if err := s.syncTaskRepo.Delete(ctx, task.Id); err != nil {
				s.logger.WithContext(ctx).Error("failed to delete sync task",
					zap.Error(err),
					zap.Int64("task_id", task.Id))
			}
		}
	}

	// 4. 删除所有模板实例
	if err := s.instanceRepo.DeleteByTemplateID(ctx, id); err != nil {
		s.logger.WithContext(ctx).Error("failed to delete template instances", zap.Error(err))
		return v1.ErrInternalServerError
	}

	// 5. 删除上传记录（可选，保留历史记录）
	// 如果需要保留历史记录，可以注释掉这部分
	upload, err := s.uploadRepo.GetByTemplateID(ctx, id)
	if err == nil && upload != nil {
		// 可以选择删除或保留上传记录
		// 这里选择保留，只删除关联关系
		s.logger.WithContext(ctx).Info("keeping upload record for history",
			zap.Int64("upload_id", upload.Id))
	}

	// 6. 最后删除模板记录
	if err := s.tplRepo.Delete(ctx, id); err != nil {
		s.logger.WithContext(ctx).Error("failed to delete template", zap.Error(err))
		return v1.ErrInternalServerError
	}

	s.logger.WithContext(ctx).Info("template deleted successfully",
		zap.Int64("template_id", id),
		zap.String("template_name", tpl.TemplateName))

	return nil
}

func (s *pveTemplateService) GetTemplate(ctx context.Context, id int64) (*v1.TemplateDetail, error) {
	tpl, err := s.tplRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get template", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	if tpl == nil {
		return nil, v1.ErrNotFound
	}

	// 查询集群信息填充 cluster_name
	var clusterName string
	if tpl.ClusterID > 0 {
		cluster, err := s.clusterRepo.GetByID(ctx, tpl.ClusterID)
		if err != nil {
			s.logger.WithContext(ctx).Warn("failed to get cluster",
				zap.Error(err), zap.Int64("cluster_id", tpl.ClusterID))
			// 不阻塞主流程，cluster_name 为空
		} else if cluster != nil {
			clusterName = cluster.ClusterName
		}
	}

	return &v1.TemplateDetail{
		Id:           tpl.Id,
		TemplateName: tpl.TemplateName,
		ClusterID:    tpl.ClusterID,
		ClusterName:  clusterName,
		Description:  tpl.Description,
		CreateTime:   tpl.CreateTime,
		UpdateTime:   tpl.UpdateTime,
		Creator:      tpl.Creator,
		Modifier:     tpl.Modifier,
	}, nil
}

func (s *pveTemplateService) ListTemplates(ctx context.Context, req *v1.ListTemplateRequest) (*v1.ListTemplateResponseData, error) {
	tpls, total, err := s.tplRepo.ListWithPagination(ctx, req.Page, req.PageSize, req.ClusterID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to list templates", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}

	items := make([]v1.TemplateItem, 0, len(tpls))
	for _, tpl := range tpls {
		items = append(items, v1.TemplateItem{
			Id:           tpl.Id,
			TemplateName: tpl.TemplateName,
			ClusterID:    tpl.ClusterID,
			Description:  tpl.Description,
		})
	}

	return &v1.ListTemplateResponseData{
		Total: total,
		List:  items,
	}, nil
}
