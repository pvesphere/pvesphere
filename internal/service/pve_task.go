package service

import (
	"context"

	v1 "pvesphere/api/v1"
	"pvesphere/internal/repository"
	"pvesphere/pkg/log"
	"pvesphere/pkg/proxmox"

	"go.uber.org/zap"
)

type PveTaskService interface {
	ListClusterTasks(ctx context.Context, req *v1.ListClusterTasksRequest) ([]v1.ClusterTaskItem, error)
	ListNodeTasks(ctx context.Context, req *v1.ListNodeTasksRequest) ([]v1.NodeTaskItem, error)
	GetTaskLog(ctx context.Context, req *v1.GetTaskLogRequest) ([]v1.TaskLogItem, error)
	GetTaskStatus(ctx context.Context, req *v1.GetTaskStatusRequest) (*v1.TaskStatusItem, error)
	StopTask(ctx context.Context, req *v1.StopTaskRequest) error
}

func NewPveTaskService(
	service *Service,
	clusterRepo repository.PveClusterRepository,
	logger *log.Logger,
) PveTaskService {
	return &pveTaskService{
		clusterRepo: clusterRepo,
		Service:     service,
		logger:      logger,
	}
}

type pveTaskService struct {
	clusterRepo repository.PveClusterRepository
	*Service
	logger *log.Logger
}

// getProxmoxClient 根据集群ID获取ProxmoxClient
func (s *pveTaskService) getProxmoxClient(ctx context.Context, clusterID int64) (*proxmox.ProxmoxClient, error) {
	cluster, err := s.clusterRepo.GetByID(ctx, clusterID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get cluster", zap.Error(err), zap.Int64("cluster_id", clusterID))
		return nil, v1.ErrInternalServerError
	}
	if cluster == nil {
		return nil, v1.ErrNotFound
	}

	client, err := proxmox.NewProxmoxClient(cluster.ApiUrl, cluster.UserId, cluster.UserToken)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to create proxmox client", zap.Error(err), zap.Int64("cluster_id", clusterID))
		return nil, v1.ErrInternalServerError
	}

	return client, nil
}

func (s *pveTaskService) ListClusterTasks(ctx context.Context, req *v1.ListClusterTasksRequest) ([]v1.ClusterTaskItem, error) {
	client, err := s.getProxmoxClient(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}

	tasks, err := client.GetClusterTasks(ctx)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get cluster tasks", zap.Error(err), zap.Int64("cluster_id", req.ClusterID))
		return nil, v1.ErrInternalServerError
	}

	result := make([]v1.ClusterTaskItem, 0, len(tasks))
	for _, task := range tasks {
		item := v1.ClusterTaskItem{
			Extra: make(map[string]interface{}),
		}

		if upid, ok := task["upid"].(string); ok {
			item.UPID = upid
		}
		if taskType, ok := task["type"].(string); ok {
			item.Type = taskType
		}
		if id, ok := task["id"].(string); ok {
			item.ID = id
		}
		if user, ok := task["user"].(string); ok {
			item.User = user
		}
		if status, ok := task["status"].(string); ok {
			item.Status = status
		}
		if startTime, ok := task["starttime"].(float64); ok {
			item.StartTime = int64(startTime)
		}
		if endTime, ok := task["endtime"].(float64); ok {
			item.EndTime = int64(endTime)
		}
		if node, ok := task["node"].(string); ok {
			item.Node = node
		}

		// 保存其他字段到Extra
		for k, v := range task {
			if k != "upid" && k != "type" && k != "id" && k != "user" && k != "status" && k != "starttime" && k != "endtime" && k != "node" {
				item.Extra[k] = v
			}
		}

		result = append(result, item)
	}

	return result, nil
}

func (s *pveTaskService) ListNodeTasks(ctx context.Context, req *v1.ListNodeTasksRequest) ([]v1.NodeTaskItem, error) {
	client, err := s.getProxmoxClient(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}

	tasks, err := client.GetNodeTasks(ctx, req.NodeName)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node tasks", zap.Error(err),
			zap.Int64("cluster_id", req.ClusterID), zap.String("node_name", req.NodeName))
		return nil, v1.ErrInternalServerError
	}

	result := make([]v1.NodeTaskItem, 0, len(tasks))
	for _, task := range tasks {
		item := v1.NodeTaskItem{
			Extra: make(map[string]interface{}),
		}

		if upid, ok := task["upid"].(string); ok {
			item.UPID = upid
		}
		if taskType, ok := task["type"].(string); ok {
			item.Type = taskType
		}
		if id, ok := task["id"].(string); ok {
			item.ID = id
		}
		if user, ok := task["user"].(string); ok {
			item.User = user
		}
		if status, ok := task["status"].(string); ok {
			item.Status = status
		}
		if startTime, ok := task["starttime"].(float64); ok {
			item.StartTime = int64(startTime)
		}
		if endTime, ok := task["endtime"].(float64); ok {
			item.EndTime = int64(endTime)
		}

		// 保存其他字段到Extra
		for k, v := range task {
			if k != "upid" && k != "type" && k != "id" && k != "user" && k != "status" && k != "starttime" && k != "endtime" {
				item.Extra[k] = v
			}
		}

		result = append(result, item)
	}

	return result, nil
}

func (s *pveTaskService) GetTaskLog(ctx context.Context, req *v1.GetTaskLogRequest) ([]v1.TaskLogItem, error) {
	client, err := s.getProxmoxClient(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}

	logs, err := client.GetTaskLog(ctx, req.NodeName, req.UPID, req.Start, req.Limit)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get task log", zap.Error(err),
			zap.Int64("cluster_id", req.ClusterID), zap.String("node_name", req.NodeName), zap.String("upid", req.UPID))
		return nil, v1.ErrInternalServerError
	}

	result := make([]v1.TaskLogItem, 0, len(logs))
	for _, log := range logs {
		item := v1.TaskLogItem{}

		if n, ok := log["n"].(float64); ok {
			item.N = int(n)
		}
		if t, ok := log["t"].(string); ok {
			item.T = t
		}

		result = append(result, item)
	}

	return result, nil
}

func (s *pveTaskService) GetTaskStatus(ctx context.Context, req *v1.GetTaskStatusRequest) (*v1.TaskStatusItem, error) {
	client, err := s.getProxmoxClient(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}

	status, err := client.GetTaskStatus(ctx, req.NodeName, req.UPID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get task status", zap.Error(err),
			zap.Int64("cluster_id", req.ClusterID), zap.String("node_name", req.NodeName), zap.String("upid", req.UPID))
		return nil, v1.ErrInternalServerError
	}

	item := &v1.TaskStatusItem{
		Extra: make(map[string]interface{}),
	}

	if upid, ok := status["upid"].(string); ok {
		item.UPID = upid
	}
	if taskType, ok := status["type"].(string); ok {
		item.Type = taskType
	}
	if id, ok := status["id"].(string); ok {
		item.ID = id
	}
	if user, ok := status["user"].(string); ok {
		item.User = user
	}
	if statusStr, ok := status["status"].(string); ok {
		item.Status = statusStr
	}
	if startTime, ok := status["starttime"].(float64); ok {
		item.StartTime = int64(startTime)
	}
	if endTime, ok := status["endtime"].(float64); ok {
		item.EndTime = int64(endTime)
	}
	if pid, ok := status["pid"].(float64); ok {
		item.Pid = int(pid)
	}
	if pStart, ok := status["pstart"].(float64); ok {
		item.PStart = int64(pStart)
	}
	if exitStatus, ok := status["exitstatus"]; ok {
		item.ExitStatus = exitStatus
	}

	// 保存其他字段到Extra
	for k, v := range status {
		if k != "upid" && k != "type" && k != "id" && k != "user" && k != "status" && k != "starttime" && k != "endtime" && k != "pid" && k != "pstart" && k != "exitstatus" {
			item.Extra[k] = v
		}
	}

	return item, nil
}

func (s *pveTaskService) StopTask(ctx context.Context, req *v1.StopTaskRequest) error {
	client, err := s.getProxmoxClient(ctx, req.ClusterID)
	if err != nil {
		return err
	}

	if err := client.StopTask(ctx, req.NodeName, req.UPID); err != nil {
		s.logger.WithContext(ctx).Error("failed to stop task", zap.Error(err),
			zap.Int64("cluster_id", req.ClusterID), zap.String("node_name", req.NodeName), zap.String("upid", req.UPID))
		return v1.ErrInternalServerError
	}

	return nil
}
