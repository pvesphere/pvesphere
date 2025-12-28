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

type PveClusterService interface {
	CreateCluster(ctx context.Context, req *v1.CreateClusterRequest) error
	UpdateCluster(ctx context.Context, id int64, req *v1.UpdateClusterRequest) error
	DeleteCluster(ctx context.Context, id int64) error
	GetCluster(ctx context.Context, id int64) (*v1.ClusterDetail, error)
	ListClusters(ctx context.Context, req *v1.ListClusterRequest) (*v1.ListClusterResponseData, error)
	GetClusterStatus(ctx context.Context, clusterID int64) ([]map[string]interface{}, error)
	GetClusterResources(ctx context.Context, clusterID int64) ([]map[string]interface{}, error)
	VerifyCluster(ctx context.Context, clusterID *int64) (*v1.VerifyClusterData, error)
	VerifyClusterWithCredentials(ctx context.Context, apiUrl, userId, userToken string) (*v1.VerifyClusterData, error)
}

func NewPveClusterService(
	service *Service,
	clusterRepo repository.PveClusterRepository,
	repo *repository.Repository,
	logger *log.Logger,
) PveClusterService {
	return &pveClusterService{
		clusterRepo: clusterRepo,
		repo:        repo,
		Service:     service,
		logger:      logger,
	}
}

type pveClusterService struct {
	clusterRepo repository.PveClusterRepository
	repo        *repository.Repository
	*Service
	logger *log.Logger
}

func (s *pveClusterService) CreateCluster(ctx context.Context, req *v1.CreateClusterRequest) error {
	// 检查集群名称是否已存在
	existing, err := s.clusterRepo.GetByClusterName(ctx, req.ClusterName)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to check cluster name", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if existing != nil {
		return v1.ErrBadRequest
	}

	cluster := &model.PveCluster{
		ClusterName:      req.ClusterName,
		ClusterNameAlias: req.ClusterNameAlias,
		Env:              req.Env,
		Datacenter:       req.Datacenter,
		ApiUrl:           req.ApiUrl,
		UserId:           req.UserId,
		UserToken:        req.UserToken,
		Dns:              req.Dns,
		Describes:        req.Describes,
		Region:           req.Region,
		IsSchedulable:    req.IsSchedulable,
		IsEnabled:        req.IsEnabled,
		CreateTime:       time.Now(),
		UpdateTime:       time.Now(),
	}

	if err := s.clusterRepo.Create(ctx, cluster); err != nil {
		s.logger.WithContext(ctx).Error("failed to create cluster", zap.Error(err))
		return v1.ErrInternalServerError
	}

	return nil
}

func (s *pveClusterService) UpdateCluster(ctx context.Context, id int64, req *v1.UpdateClusterRequest) error {
	cluster, err := s.clusterRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get cluster", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if cluster == nil {
		return v1.ErrNotFound
	}

	// 更新字段
	if req.ClusterNameAlias != nil {
		cluster.ClusterNameAlias = *req.ClusterNameAlias
	}
	if req.Env != nil {
		cluster.Env = *req.Env
	}
	if req.Datacenter != nil {
		cluster.Datacenter = *req.Datacenter
	}
	if req.ApiUrl != nil {
		cluster.ApiUrl = *req.ApiUrl
	}
	if req.UserId != nil {
		cluster.UserId = *req.UserId
	}
	if req.UserToken != nil {
		cluster.UserToken = *req.UserToken
	}
	if req.Dns != nil {
		cluster.Dns = *req.Dns
	}
	if req.Describes != nil {
		cluster.Describes = *req.Describes
	}
	if req.Region != nil {
		cluster.Region = *req.Region
	}
	if req.IsSchedulable != nil {
		cluster.IsSchedulable = *req.IsSchedulable
	}
	if req.IsEnabled != nil {
		cluster.IsEnabled = *req.IsEnabled
	}
	cluster.UpdateTime = time.Now()

	if err := s.clusterRepo.Update(ctx, cluster); err != nil {
		s.logger.WithContext(ctx).Error("failed to update cluster", zap.Error(err))
		return v1.ErrInternalServerError
	}

	return nil
}

func (s *pveClusterService) DeleteCluster(ctx context.Context, id int64) error {
	cluster, err := s.clusterRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get cluster", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if cluster == nil {
		return v1.ErrNotFound
	}

	// 步骤 1：先禁用集群（设置 IsEnabled=0），阻止控制器继续同步数据
	if cluster.IsEnabled != 0 {
		cluster.IsEnabled = 0
		cluster.UpdateTime = time.Now()
		if err := s.clusterRepo.Update(ctx, cluster); err != nil {
			s.logger.WithContext(ctx).Error("failed to disable cluster before deletion", zap.Error(err))
			return v1.ErrInternalServerError
		}
		s.logger.WithContext(ctx).Info("cluster disabled before deletion",
			zap.Int64("cluster_id", id),
			zap.String("cluster_name", cluster.ClusterName))
	}

	// 步骤 2：使用事务级联删除所有关联数据
	// 删除顺序：按照依赖关系，先删除子表，再删除父表
	err = s.tm.Transaction(ctx, func(ctx context.Context) error {
		db := s.repo.DB(ctx)

		// 1. 删除虚拟机 IP 地址（依赖 vm）
		result := db.Table("vm_ipaddress").Where("cluster_id = ?", id).Delete(&model.VMIPAddress{})
		if result.Error != nil {
			s.logger.WithContext(ctx).Error("failed to delete vm ip addresses", zap.Error(result.Error))
			return result.Error
		}
		s.logger.WithContext(ctx).Debug("deleted vm ip addresses", zap.Int64("rows_affected", result.RowsAffected))

		// 2. 删除虚拟机（依赖 node）
		result = db.Table("pve_vm").Where("cluster_id = ?", id).Delete(&model.PveVM{})
		if result.Error != nil {
			s.logger.WithContext(ctx).Error("failed to delete vms", zap.Error(result.Error))
			return result.Error
		}
		s.logger.WithContext(ctx).Debug("deleted vms", zap.Int64("rows_affected", result.RowsAffected))

		// 3. 删除存储（依赖 node）
		// 使用 Table() 明确指定表名，确保删除操作正确执行
		result = db.Table("pve_storage").Where("cluster_id = ?", id).Delete(&model.PveStorage{})
		if result.Error != nil {
			s.logger.WithContext(ctx).Error("failed to delete storages", zap.Error(result.Error))
			return result.Error
		}
		s.logger.WithContext(ctx).Debug("deleted storages", zap.Int64("rows_affected", result.RowsAffected))

		// 4. 删除模板同步任务（依赖 template_instance）
		result = db.Table("template_sync_task").Where("cluster_id = ?", id).Delete(&model.TemplateSyncTask{})
		if result.Error != nil {
			s.logger.WithContext(ctx).Error("failed to delete template sync tasks", zap.Error(result.Error))
			return result.Error
		}
		s.logger.WithContext(ctx).Debug("deleted template sync tasks", zap.Int64("rows_affected", result.RowsAffected))

		// 5. 删除模板实例（依赖 template_upload）
		result = db.Table("template_instance").Where("cluster_id = ?", id).Delete(&model.TemplateInstance{})
		if result.Error != nil {
			s.logger.WithContext(ctx).Error("failed to delete template instances", zap.Error(result.Error))
			return result.Error
		}
		s.logger.WithContext(ctx).Debug("deleted template instances", zap.Int64("rows_affected", result.RowsAffected))

		// 6. 删除模板上传记录（依赖 template）
		result = db.Table("template_upload").Where("cluster_id = ?", id).Delete(&model.TemplateUpload{})
		if result.Error != nil {
			s.logger.WithContext(ctx).Error("failed to delete template uploads", zap.Error(result.Error))
			return result.Error
		}
		s.logger.WithContext(ctx).Debug("deleted template uploads", zap.Int64("rows_affected", result.RowsAffected))

		// 7. 删除模板（vm_template 表）
		result = db.Table("vm_template").Where("cluster_id = ?", id).Delete(&model.VmTemplate{})
		if result.Error != nil {
			s.logger.WithContext(ctx).Error("failed to delete templates", zap.Error(result.Error))
			return result.Error
		}
		s.logger.WithContext(ctx).Debug("deleted templates", zap.Int64("rows_affected", result.RowsAffected))

		// 8. 删除节点（依赖 cluster）
		result = db.Table("pve_node").Where("cluster_id = ?", id).Delete(&model.PveNode{})
		if result.Error != nil {
			s.logger.WithContext(ctx).Error("failed to delete nodes", zap.Error(result.Error))
			return result.Error
		}
		s.logger.WithContext(ctx).Debug("deleted nodes", zap.Int64("rows_affected", result.RowsAffected))

		// 9. 最后删除集群本身
		if err := s.clusterRepo.Delete(ctx, id); err != nil {
			s.logger.WithContext(ctx).Error("failed to delete cluster", zap.Error(err))
			return err
		}

		s.logger.WithContext(ctx).Info("cluster and all related data deleted successfully",
			zap.Int64("cluster_id", id),
			zap.String("cluster_name", cluster.ClusterName))
		return nil
	})

	if err != nil {
		s.logger.WithContext(ctx).Error("failed to delete cluster with cascade", zap.Error(err), zap.Int64("cluster_id", id))
		return v1.ErrInternalServerError
	}

	return nil
}

func (s *pveClusterService) GetCluster(ctx context.Context, id int64) (*v1.ClusterDetail, error) {
	cluster, err := s.clusterRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get cluster", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	if cluster == nil {
		return nil, v1.ErrNotFound
	}

	return &v1.ClusterDetail{
		Id:               cluster.Id,
		ClusterName:      cluster.ClusterName,
		ClusterNameAlias: cluster.ClusterNameAlias,
		Env:              cluster.Env,
		Datacenter:       cluster.Datacenter,
		ApiUrl:           cluster.ApiUrl,
		UserId:           cluster.UserId,
		Dns:              cluster.Dns,
		Describes:        cluster.Describes,
		Region:           cluster.Region,
		IsSchedulable:    cluster.IsSchedulable,
		IsEnabled:        cluster.IsEnabled,
		CreateTime:       cluster.CreateTime,
		UpdateTime:       cluster.UpdateTime,
		Creator:          cluster.Creator,
		Modifier:         cluster.Modifier,
	}, nil
}

func (s *pveClusterService) ListClusters(ctx context.Context, req *v1.ListClusterRequest) (*v1.ListClusterResponseData, error) {
	clusters, total, err := s.clusterRepo.ListWithPagination(ctx, req.Page, req.PageSize, req.Env, req.Region)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to list clusters", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}

	items := make([]v1.ClusterItem, 0, len(clusters))
	for _, cluster := range clusters {
		items = append(items, v1.ClusterItem{
			Id:               cluster.Id,
			ClusterName:      cluster.ClusterName,
			ClusterNameAlias: cluster.ClusterNameAlias,
			Env:              cluster.Env,
			Datacenter:       cluster.Datacenter,
			ApiUrl:           cluster.ApiUrl,
			Region:           cluster.Region,
			IsSchedulable:    cluster.IsSchedulable,
			IsEnabled:        cluster.IsEnabled,
		})
	}

	return &v1.ListClusterResponseData{
		Total: total,
		List:  items,
	}, nil
}

func (s *pveClusterService) GetClusterStatus(ctx context.Context, clusterID int64) ([]map[string]interface{}, error) {
	// 1. 获取集群信息
	cluster, err := s.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get cluster", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	if cluster == nil {
		return nil, v1.ErrNotFound
	}

	// 2. 创建 Proxmox 客户端
	proxmoxClient, err := proxmox.NewProxmoxClient(cluster.ApiUrl, cluster.UserId, cluster.UserToken)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to create proxmox client", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}

	// 3. 获取集群状态
	status, err := proxmoxClient.GetClusterStatus(ctx)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get cluster status", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}

	return status, nil
}

func (s *pveClusterService) GetClusterResources(ctx context.Context, clusterID int64) ([]map[string]interface{}, error) {
	// 1. 获取集群信息
	cluster, err := s.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get cluster", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	if cluster == nil {
		return nil, v1.ErrNotFound
	}

	// 2. 创建 Proxmox 客户端
	proxmoxClient, err := proxmox.NewProxmoxClient(cluster.ApiUrl, cluster.UserId, cluster.UserToken)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to create proxmox client", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}

	// 3. 获取集群资源
	resources, err := proxmoxClient.GetClusterResources(ctx)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get cluster resources", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}

	return resources, nil
}

func (s *pveClusterService) VerifyCluster(ctx context.Context, clusterID *int64) (*v1.VerifyClusterData, error) {
	// 1. 获取集群信息
	cluster, err := s.clusterRepo.GetByID(ctx, *clusterID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get cluster", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	if cluster == nil {
		return nil, v1.ErrNotFound
	}

	// 2. 使用集群信息验证连接
	return s.VerifyClusterWithCredentials(ctx, cluster.ApiUrl, cluster.UserId, cluster.UserToken)
}

func (s *pveClusterService) VerifyClusterWithCredentials(ctx context.Context, apiUrl, userId, userToken string) (*v1.VerifyClusterData, error) {
	// 1. 创建 Proxmox 客户端
	proxmoxClient, err := proxmox.NewProxmoxClient(apiUrl, userId, userToken)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to create proxmox client", zap.Error(err))
		return &v1.VerifyClusterData{
			Connected: false,
			Message:   "failed to create proxmox client: " + err.Error(),
		}, nil
	}

	// 2. 调用 /version 接口验证连接
	versionInfo, err := proxmoxClient.GetVersion(ctx)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get proxmox version", zap.Error(err),
			zap.String("api_url", apiUrl))
		return &v1.VerifyClusterData{
			Connected: false,
			Message:   "connection failed: " + err.Error(),
		}, nil
	}

	// 3. 解析版本信息
	version, _ := versionInfo["version"].(string)
	release, _ := versionInfo["release"].(string)
	repoid, _ := versionInfo["repoid"].(string)

	return &v1.VerifyClusterData{
		Version:   version,
		Release:   release,
		RepoID:    repoid,
		Connected: true,
		Message:   "connection successful",
	}, nil
}
