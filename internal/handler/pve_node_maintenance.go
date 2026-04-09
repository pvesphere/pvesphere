package handler

import (
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"pvesphere/api/v1"
	"pvesphere/internal/service"
	"time"

	"github.com/gin-gonic/gin"
)

// NodeMaintenanceHandler 主机维护处理器
type NodeMaintenanceHandler struct {
	*Handler
	nodeMaintenanceService *service.NodeMaintenanceService
}

// NewNodeMaintenanceHandler 创建主机维护处理器
func NewNodeMaintenanceHandler(
	handler *Handler,
	nodeMaintenanceService *service.NodeMaintenanceService,
) *NodeMaintenanceHandler {
	return &NodeMaintenanceHandler{
		Handler:                handler,
		nodeMaintenanceService: nodeMaintenanceService,
	}
}

// GetMigrateCandidates 获取可迁移的目标节点列表
// @Summary 获取可迁移的目标节点列表
// @Description 根据指定节点的资源使用情况，推荐可用于 VM 迁移的目标节点
// @Tags 主机维护模块
// @Accept json
// @Produce json
// @Param node_id query int true "源节点 ID"
// @Success 200 {object} v1.MigrateCandidatesResponse
// @Router /api/v1/nodes/maintenance/candidates [get]
func (h *NodeMaintenanceHandler) GetMigrateCandidates(c *gin.Context) {
	nodeIDStr := c.Query("node_id")
	if nodeIDStr == "" {
		c.JSON(http.StatusBadRequest, v1.Response{Code: 400, Message: "node_id is required"})
		return
	}

	nodeID, err := parseIntParam(nodeIDStr)
	if err != nil || nodeID <= 0 {
		c.JSON(http.StatusBadRequest, v1.Response{Code: 400, Message: "invalid node_id"})
		return
	}

	data, err := h.nodeMaintenanceService.GetMigrateCandidates(c.Request.Context(), nodeID)
	if err != nil {
		h.logger.WithContext(c).Error("GetMigrateCandidates error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, v1.Response{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, v1.MigrateCandidatesResponse{Response: v1.Response{Code: 200, Message: "success"}, Data: data})
}

// GetNodeVMs 获取节点上的 VM 列表
// @Summary 获取节点上的 VM 列表
// @Description 获取指定节点上的所有虚拟机列表
// @Tags 主机维护模块
// @Accept json
// @Produce json
// @Param node_id query int true "节点 ID"
// @Success 200 {object} v1.NodeVMsResponse
// @Router /api/v1/nodes/maintenance/vms [get]
func (h *NodeMaintenanceHandler) GetNodeVMs(c *gin.Context) {
	nodeIDStr := c.Query("node_id")
	if nodeIDStr == "" {
		c.JSON(http.StatusBadRequest, v1.Response{Code: 400, Message: "node_id is required"})
		return
	}

	nodeID, err := parseIntParam(nodeIDStr)
	if err != nil || nodeID <= 0 {
		c.JSON(http.StatusBadRequest, v1.Response{Code: 400, Message: "invalid node_id"})
		return
	}

	data, err := h.nodeMaintenanceService.GetNodeVMs(c.Request.Context(), nodeID)
	if err != nil {
		h.logger.WithContext(c).Error("GetNodeVMs error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, v1.Response{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, v1.NodeVMsResponse{Response: v1.Response{Code: 200, Message: "success"}, Data: data})
}

// StartMaintenance 开始主机维护（同步版本，保留向后兼容）
// @Summary 开始主机维护
// @Description 启动节点维护流程，支持自动和手动两种模式迁移 VM（同步阻塞模式）
// @Tags 主机维护模块
// @Accept json
// @Produce json
// @Param request body v1.StartMaintenanceRequest true "开始维护请求"
// @Success 200 {object} v1.NodeMaintenanceResponse
// @Router /api/v1/nodes/maintenance/start [post]
func (h *NodeMaintenanceHandler) StartMaintenance(c *gin.Context) {
	var req v1.StartMaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, v1.Response{Code: 400, Message: err.Error()})
		return
	}

	// 参数验证
	if req.NodeID == "" {
		c.JSON(http.StatusBadRequest, v1.Response{Code: 400, Message: "node_id is required"})
		return
	}
	if req.Mode != "auto" && req.Mode != "manual" {
		c.JSON(http.StatusBadRequest, v1.Response{Code: 400, Message: "mode must be 'auto' or 'manual'"})
		return
	}
	if req.Concurrency <= 0 {
		req.Concurrency = 3
	}
	if req.Concurrency > 10 {
		req.Concurrency = 10 // 最大并发数限制
	}

	data, err := h.nodeMaintenanceService.StartMaintenance(c.Request.Context(), req)
	if err != nil {
		h.logger.WithContext(c).Error("StartMaintenance error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, v1.Response{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, v1.NodeMaintenanceResponse{Response: v1.Response{Code: 200, Message: "success"}, Data: *data})
}

// CreateMaintenancePlan 创建维护计划（异步版本）
// @Summary 创建维护计划
// @Description 创建节点维护计划，异步执行 VM 迁移任务
// @Tags 主机维护计划
// @Accept json
// @Produce json
// @Param request body v1.CreateMaintenancePlanRequest true "创建维护计划请求"
// @Success 200 {object} v1.CreateMaintenancePlanResponse
// @Router /api/v1/nodes/maintenance/plans [post]
func (h *NodeMaintenanceHandler) CreateMaintenancePlan(c *gin.Context) {
	var req v1.CreateMaintenancePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, v1.Response{Code: 400, Message: err.Error()})
		return
	}

	// 参数验证
	if req.NodeID == "" {
		c.JSON(http.StatusBadRequest, v1.Response{Code: 400, Message: "node_id is required"})
		return
	}
	if req.Mode != "auto" && req.Mode != "manual" {
		c.JSON(http.StatusBadRequest, v1.Response{Code: 400, Message: "mode must be 'auto' or 'manual'"})
		return
	}
	if req.Concurrency <= 0 {
		req.Concurrency = 3
	}
	if req.Concurrency > 10 {
		req.Concurrency = 10
	}

	plan, err := h.nodeMaintenanceService.CreateMaintenancePlan(c.Request.Context(), req)
	if err != nil {
		h.logger.WithContext(c).Error("CreateMaintenancePlan error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, v1.Response{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, v1.CreateMaintenancePlanResponse{
		Response: v1.Response{Code: 200, Message: "success"},
		Data: v1.MaintenancePlanDetail{
			PlanID:      plan.Id,
			NodeID:      plan.NodeID,
			NodeName:    plan.NodeName,
			Status:      plan.Status,
			Mode:        plan.Mode,
			Concurrency: plan.Concurrency,
			TotalVMs:    plan.TotalVMs,
			MigratedVMs: plan.MigratedVMs,
			FailedVMs:   plan.FailedVMs,
			StartTime:   formatTime(plan.StartTime),
			EndTime:     formatTime(plan.EndTime),
			CreateTime:  formatTime(plan.CreateTime),
		},
	})
}

// GetMaintenancePlan 获取维护计划详情
// @Summary 获取维护计划详情
// @Description 根据计划 ID 查询维护计划详细信息
// @Tags 主机维护计划
// @Accept json
// @Produce json
// @Param plan_id path int true "计划 ID"
// @Success 200 {object} v1.MaintenancePlanResponse
// @Router /api/v1/nodes/maintenance/plans/:plan_id [get]
func (h *NodeMaintenanceHandler) GetMaintenancePlan(c *gin.Context) {
	planIDStr := c.Param("plan_id")
	if planIDStr == "" {
		c.JSON(http.StatusBadRequest, v1.Response{Code: 400, Message: "plan_id is required"})
		return
	}

	planID, err := parseIntParam(planIDStr)
	if err != nil || planID <= 0 {
		c.JSON(http.StatusBadRequest, v1.Response{Code: 400, Message: "invalid plan_id"})
		return
	}

	plan, err := h.nodeMaintenanceService.GetMaintenancePlan(c.Request.Context(), planID)
	if err != nil {
		h.logger.WithContext(c).Error("GetMaintenancePlan error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, v1.Response{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, v1.MaintenancePlanResponse{
		Response: v1.Response{Code: 200, Message: "success"},
		Data: v1.MaintenancePlanDetail{
			PlanID:      plan.Id,
			NodeID:      plan.NodeID,
			NodeName:    plan.NodeName,
			Status:      plan.Status,
			Mode:        plan.Mode,
			Concurrency: plan.Concurrency,
			TotalVMs:    plan.TotalVMs,
			MigratedVMs: plan.MigratedVMs,
			FailedVMs:   plan.FailedVMs,
			StartTime:   formatTime(plan.StartTime),
			EndTime:     formatTime(plan.EndTime),
			CreateTime:  formatTime(plan.CreateTime),
		},
	})
}

// GetMaintenancePlanTasks 获取维护计划任务列表
// @Summary 获取维护计划任务列表
// @Description 查询指定维护计划的所有任务及进度
// @Tags 主机维护计划
// @Accept json
// @Produce json
// @Param plan_id path int true "计划 ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} v1.MaintenanceTaskListResponse
// @Router /api/v1/nodes/maintenance/plans/:plan_id/tasks [get]
func (h *NodeMaintenanceHandler) GetMaintenancePlanTasks(c *gin.Context) {
	planIDStr := c.Param("plan_id")
	if planIDStr == "" {
		c.JSON(http.StatusBadRequest, v1.Response{Code: 400, Message: "plan_id is required"})
		return
	}

	planID, err := parseIntParam(planIDStr)
	if err != nil || planID <= 0 {
		c.JSON(http.StatusBadRequest, v1.Response{Code: 400, Message: "invalid plan_id"})
		return
	}

	page, _ := parseIntParam(c.DefaultQuery("page", "1"))
	pageSize, _ := parseIntParam(c.DefaultQuery("page_size", "20"))

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	tasks, total, err := h.nodeMaintenanceService.GetMaintenancePlanTasks(c.Request.Context(), planID, int(page), int(pageSize))
	if err != nil {
		h.logger.WithContext(c).Error("GetMaintenancePlanTasks error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, v1.Response{Code: 500, Message: err.Error()})
		return
	}

	taskItems := make([]v1.MaintenanceTaskItem, 0, len(tasks))
	for _, task := range tasks {
		taskItems = append(taskItems, v1.MaintenanceTaskItem{
			TaskID:      task.Id,
			VMID:        task.VMID,
			VMName:      task.VMName,
			SourceNode:  task.SourceNode,
			TargetNode:  task.TargetNode,
			Status:      task.Status,
			ErrorMessage: task.ErrorMessage,
			StartedAt:   formatTime(task.StartedAt),
			CompletedAt: formatTime(task.CompletedAt),
		})
	}

	c.JSON(http.StatusOK, v1.MaintenanceTaskListResponse{
		Response: v1.Response{Code: 200, Message: "success"},
		Data: struct {
			Total int64                 `json:"total"`
			Tasks []v1.MaintenanceTaskItem `json:"tasks"`
		}{
			Total: total,
			Tasks: taskItems,
		},
	})
}

// GetMaintenancePlanProgress 获取维护计划进度
// @Summary 获取维护计划进度
// @Description 查询维护计划的实时进度
// @Tags 主机维护计划
// @Accept json
// @Produce json
// @Param plan_id path int true "计划 ID"
// @Success 200 {object} v1.MaintenancePlanProgressResponse
// @Router /api/v1/nodes/maintenance/plans/:plan_id/progress [get]
func (h *NodeMaintenanceHandler) GetMaintenancePlanProgress(c *gin.Context) {
	planIDStr := c.Param("plan_id")
	if planIDStr == "" {
		c.JSON(http.StatusBadRequest, v1.Response{Code: 400, Message: "plan_id is required"})
		return
	}

	planID, err := parseIntParam(planIDStr)
	if err != nil || planID <= 0 {
		c.JSON(http.StatusBadRequest, v1.Response{Code: 400, Message: "invalid plan_id"})
		return
	}

	plan, tasks, err := h.nodeMaintenanceService.GetMaintenancePlanProgress(c.Request.Context(), planID)
	if err != nil {
		h.logger.WithContext(c).Error("GetMaintenancePlanProgress error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, v1.Response{Code: 500, Message: err.Error()})
		return
	}

	progress := float64(0)
	if plan.TotalVMs > 0 {
		progress = float64(plan.MigratedVMs) / float64(plan.TotalVMs) * 100
	}

	taskItems := make([]v1.MaintenanceTaskItem, 0, len(tasks))
	for _, task := range tasks {
		taskItems = append(taskItems, v1.MaintenanceTaskItem{
			TaskID:      task.Id,
			VMID:        task.VMID,
			VMName:      task.VMName,
			SourceNode:  task.SourceNode,
			TargetNode:  task.TargetNode,
			Status:      task.Status,
			ErrorMessage: task.ErrorMessage,
			StartedAt:   formatTime(task.StartedAt),
			CompletedAt: formatTime(task.CompletedAt),
		})
	}

	c.JSON(http.StatusOK, v1.MaintenancePlanProgressResponse{
		Response: v1.Response{Code: 200, Message: "success"},
		Data: v1.MaintenancePlanProgress{
			PlanID:      plan.Id,
			NodeName:    plan.NodeName,
			Status:      plan.Status,
			TotalVMs:    plan.TotalVMs,
			MigratedVMs: plan.MigratedVMs,
			FailedVMs:   plan.FailedVMs,
			Progress:    progress,
			Tasks:       taskItems,
		},
	})
}

// CancelMaintenancePlan 取消维护计划
// @Summary 取消维护计划
// @Description 取消正在执行的维护计划
// @Tags 主机维护计划
// @Accept json
// @Produce json
// @Param plan_id path int true "计划 ID"
// @Success 200 {object} v1.CancelMaintenancePlanResponse
// @Router /api/v1/nodes/maintenance/plans/:plan_id/cancel [post]
func (h *NodeMaintenanceHandler) CancelMaintenancePlan(c *gin.Context) {
	planIDStr := c.Param("plan_id")
	if planIDStr == "" {
		c.JSON(http.StatusBadRequest, v1.Response{Code: 400, Message: "plan_id is required"})
		return
	}

	planID, err := parseIntParam(planIDStr)
	if err != nil || planID <= 0 {
		c.JSON(http.StatusBadRequest, v1.Response{Code: 400, Message: "invalid plan_id"})
		return
	}

	err = h.nodeMaintenanceService.CancelMaintenancePlan(c.Request.Context(), planID)
	if err != nil {
		h.logger.WithContext(c).Error("CancelMaintenancePlan error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, v1.Response{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, v1.CancelMaintenancePlanResponse{
		Response: v1.Response{Code: 200, Message: "success"},
	})
}

// GetMaintenanceHistory 获取维护历史
// @Summary 获取维护历史
// @Description 查询已完成的维护计划历史记录
// @Tags 主机维护计划
// @Accept json
// @Produce json
// @Param node_id query int false "节点 ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} v1.MaintenanceHistoryResponse
// @Router /api/v1/nodes/maintenance/history [get]
func (h *NodeMaintenanceHandler) GetMaintenanceHistory(c *gin.Context) {
	nodeIDStr := c.Query("node_id")
	var nodeID int64 = 0
	if nodeIDStr != "" {
		nodeID, _ = parseIntParam(nodeIDStr)
	}

	page, _ := parseIntParam(c.DefaultQuery("page", "1"))
	pageSize, _ := parseIntParam(c.DefaultQuery("page_size", "20"))

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	plans, total, err := h.nodeMaintenanceService.GetMaintenanceHistory(c.Request.Context(), nodeID, int(page), int(pageSize))
	if err != nil {
		h.logger.WithContext(c).Error("GetMaintenanceHistory error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, v1.Response{Code: 500, Message: err.Error()})
		return
	}

	records := make([]v1.HistoryRecord, 0, len(plans))
	for _, plan := range plans {
		duration := ""
		if !plan.StartTime.IsZero() && !plan.EndTime.IsZero() {
			d := plan.EndTime.Sub(plan.StartTime)
			duration = d.String()
		}
		records = append(records, v1.HistoryRecord{
			PlanID:      plan.Id,
			NodeID:      plan.NodeID,
			NodeName:    plan.NodeName,
			Status:      plan.Status,
			TotalVMs:    plan.TotalVMs,
			MigratedVMs: plan.MigratedVMs,
			FailedVMs:   plan.FailedVMs,
			StartTime:   formatTime(plan.StartTime),
			EndTime:     formatTime(plan.EndTime),
			Duration:    duration,
		})
	}

	c.JSON(http.StatusOK, v1.MaintenanceHistoryResponse{
		Response: v1.Response{Code: 200, Message: "success"},
		Data: struct {
			Total   int64                `json:"total"`
			Records []v1.HistoryRecord   `json:"records"`
		}{
			Total:   total,
			Records: records,
		},
	})
}

// EndMaintenance 结束主机维护
// @Summary 结束主机维护
// @Description 结束节点维护模式，恢复正常运行状态（有任务运行时禁止结束）
// @Tags 主机维护模块
// @Accept json
// @Produce json
// @Param request body v1.EndMaintenanceRequest true "结束维护请求"
// @Success 200 {object} v1.Response
// @Router /api/v1/nodes/maintenance/end [post]
func (h *NodeMaintenanceHandler) EndMaintenance(c *gin.Context) {
	var req v1.EndMaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, v1.Response{Code: 400, Message: err.Error()})
		return
	}

	// 参数验证
	if req.NodeID == "" {
		c.JSON(http.StatusBadRequest, v1.Response{Code: 400, Message: "node_id is required"})
		return
	}

	err := h.nodeMaintenanceService.EndMaintenance(c.Request.Context(), req.NodeID)
	if err != nil {
		h.logger.WithContext(c).Error("EndMaintenance error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, v1.Response{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, v1.Response{Code: 200, Message: "success"})
}

// parseIntParam 解析整数参数
func parseIntParam(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	var v int64
	_, err := fmt.Sscanf(s, "%d", &v)
	return v, err
}

// formatTime 格式化时间
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}
