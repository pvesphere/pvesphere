package handler

import (
	"net/http"

	v1 "pvesphere/api/v1"
	"pvesphere/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PveTaskHandler struct {
	*Handler
	taskService service.PveTaskService
}

func NewPveTaskHandler(handler *Handler, taskService service.PveTaskService) *PveTaskHandler {
	return &PveTaskHandler{
		Handler:     handler,
		taskService: taskService,
	}
}

// ListClusterTasks godoc
// @Summary 获取集群任务列表
// @Tags PVE任务模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param cluster_id query int true "集群ID"
// @Success 200 {object} v1.ListClusterTasksResponse
// @Router /api/v1/tasks/cluster [get]
func (h *PveTaskHandler) ListClusterTasks(ctx *gin.Context) {
	req := new(v1.ListClusterTasksRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	tasks, err := h.taskService.ListClusterTasks(ctx, req)
	if err != nil {
		h.logger.WithContext(ctx).Error("taskService.ListClusterTasks error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, tasks)
}

// ListNodeTasks godoc
// @Summary 获取节点任务列表
// @Tags PVE任务模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param cluster_id query int true "集群ID"
// @Param node_name query string true "节点名称"
// @Success 200 {object} v1.ListNodeTasksResponse
// @Router /api/v1/tasks/node [get]
func (h *PveTaskHandler) ListNodeTasks(ctx *gin.Context) {
	req := new(v1.ListNodeTasksRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	tasks, err := h.taskService.ListNodeTasks(ctx, req)
	if err != nil {
		h.logger.WithContext(ctx).Error("taskService.ListNodeTasks error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, tasks)
}

// GetTaskLog godoc
// @Summary 获取任务日志
// @Tags PVE任务模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param cluster_id query int true "集群ID"
// @Param node_name query string true "节点名称"
// @Param upid query string true "任务UPID"
// @Param start query int false "起始行号" default(0)
// @Param limit query int false "返回行数" default(50)
// @Success 200 {object} v1.GetTaskLogResponse
// @Router /api/v1/tasks/log [get]
func (h *PveTaskHandler) GetTaskLog(ctx *gin.Context) {
	req := new(v1.GetTaskLogRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	// 设置默认值
	if req.Start < 0 {
		req.Start = 0
	}
	if req.Limit <= 0 {
		req.Limit = 50
	}
	// 限制最大返回行数
	if req.Limit > 1000 {
		req.Limit = 1000
	}

	logs, err := h.taskService.GetTaskLog(ctx, req)
	if err != nil {
		h.logger.WithContext(ctx).Error("taskService.GetTaskLog error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, logs)
}

// GetTaskStatus godoc
// @Summary 获取任务状态
// @Tags PVE任务模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param cluster_id query int true "集群ID"
// @Param node_name query string true "节点名称"
// @Param upid query string true "任务UPID"
// @Success 200 {object} v1.GetTaskStatusResponse
// @Router /api/v1/tasks/status [get]
func (h *PveTaskHandler) GetTaskStatus(ctx *gin.Context) {
	req := new(v1.GetTaskStatusRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	status, err := h.taskService.GetTaskStatus(ctx, req)
	if err != nil {
		h.logger.WithContext(ctx).Error("taskService.GetTaskStatus error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, status)
}

// StopTask godoc
// @Summary 终止任务
// @Tags PVE任务模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param cluster_id query int true "集群ID"
// @Param node_name query string true "节点名称"
// @Param upid query string true "任务UPID"
// @Success 200 {object} v1.StopTaskResponse
// @Router /api/v1/tasks/stop [delete]
func (h *PveTaskHandler) StopTask(ctx *gin.Context) {
	req := new(v1.StopTaskRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.taskService.StopTask(ctx, req); err != nil {
		h.logger.WithContext(ctx).Error("taskService.StopTask error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}
