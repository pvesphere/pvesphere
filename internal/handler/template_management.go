package handler

import (
	"net/http"
	"strconv"

	v1 "pvesphere/api/v1"
	"pvesphere/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TemplateManagementHandler struct {
	*Handler
	templateManagementService service.TemplateManagementService
}

func NewTemplateManagementHandler(
	handler *Handler,
	templateManagementService service.TemplateManagementService,
) *TemplateManagementHandler {
	return &TemplateManagementHandler{
		Handler:                   handler,
		templateManagementService: templateManagementService,
	}
}

// ImportTemplate 从备份文件导入模板
// @Summary 从备份文件导入模板
// @Description 基于已有的虚拟机备份文件创建模板，支持共享存储和本地存储
// @Tags 模板管理
// @Accept json
// @Produce json
// @Param request body v1.ImportTemplateRequest true "导入模板请求"
// @Success 200 {object} v1.ImportTemplateResponse
// @Router /api/v1/templates/import [post]
func (h *TemplateManagementHandler) ImportTemplate(ctx *gin.Context) {
	// 解析请求体
	var req v1.ImportTemplateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		h.logger.WithContext(ctx).Error("ImportTemplate bind json error", zap.Error(err))
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	// 调用服务层
	data, err := h.templateManagementService.ImportTemplateFromBackup(ctx.Request.Context(), &req)
	if err != nil {
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, data)
}

// GetTemplateDetail 查询模板详情（包含实例）
// @Summary 查询模板详情
// @Description 查询模板详细信息，包括上传信息和实例列表
// @Tags 模板管理
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Param include_instances query boolean false "是否包含实例信息"
// @Success 200 {object} v1.GetTemplateDetailResponse
// @Router /api/v1/templates/{id}/detail [get]
func (h *TemplateManagementHandler) GetTemplateDetail(ctx *gin.Context) {
	// 获取模板ID
	idStr := ctx.Param("id")
	if idStr == "" {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	templateID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	// 解析查询参数
	includeInstancesStr := ctx.Query("include_instances")
	includeInstances := includeInstancesStr == "true"

	// 调用服务层
	data, err := h.templateManagementService.GetTemplateDetailWithInstances(ctx.Request.Context(), templateID, includeInstances)
	if err != nil {
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, data)
}

// SyncTemplate 同步模板到其他节点
// @Summary 同步模板到其他节点
// @Description 将本地存储的模板同步到其他节点（仅支持local存储）
// @Tags 模板管理
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Param request body v1.SyncTemplateRequest true "同步请求"
// @Success 200 {object} v1.SyncTemplateResponse
// @Router /api/v1/templates/{id}/sync [post]
func (h *TemplateManagementHandler) SyncTemplate(ctx *gin.Context) {
	// 获取模板ID
	idStr := ctx.Param("id")
	if idStr == "" {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	templateID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	// 解析请求体
	var req v1.SyncTemplateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	// 调用服务层
	data, err := h.templateManagementService.SyncTemplateToNodes(ctx.Request.Context(), templateID, req.TargetNodeIDs)
	if err != nil {
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, data)
}

// GetSyncTask 查询同步任务
// @Summary 查询同步任务
// @Description 查询指定同步任务的详细信息
// @Tags 模板管理
// @Accept json
// @Produce json
// @Param task_id path int true "任务ID"
// @Success 200 {object} v1.GetSyncTaskResponse
// @Router /api/v1/templates/sync-tasks/{task_id} [get]
func (h *TemplateManagementHandler) GetSyncTask(ctx *gin.Context) {
	// 获取任务ID
	taskIDStr := ctx.Param("task_id")
	if taskIDStr == "" {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	// 调用服务层
	data, err := h.templateManagementService.GetSyncTask(ctx.Request.Context(), taskID)
	if err != nil {
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, data)
}

// ListSyncTasks 列出同步任务
// @Summary 列出同步任务
// @Description 列出同步任务列表，支持分页和过滤
// @Tags 模板管理
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param template_id query int false "模板ID"
// @Param status query string false "任务状态"
// @Success 200 {object} v1.ListSyncTasksResponse
// @Router /api/v1/templates/sync-tasks [get]
func (h *TemplateManagementHandler) ListSyncTasks(ctx *gin.Context) {
	// 解析查询参数
	var req v1.ListSyncTasksRequest

	pageStr := ctx.Query("page")
	if pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err == nil {
			req.Page = page
		}
	}
	if req.Page <= 0 {
		req.Page = 1
	}

	pageSizeStr := ctx.Query("page_size")
	if pageSizeStr != "" {
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err == nil {
			req.PageSize = pageSize
		}
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	templateIDStr := ctx.Query("template_id")
	if templateIDStr != "" {
		templateID, err := strconv.ParseInt(templateIDStr, 10, 64)
		if err == nil {
			req.TemplateID = &templateID
		}
	}

	req.Status = ctx.Query("status")

	// 调用服务层
	data, err := h.templateManagementService.ListSyncTasks(ctx.Request.Context(), &req)
	if err != nil {
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, data)
}

// RetrySyncTask 重试同步任务
// @Summary 重试同步任务
// @Description 重试失败的同步任务
// @Tags 模板管理
// @Accept json
// @Produce json
// @Param task_id path int true "任务ID"
// @Success 200 {object} v1.RetrySyncTaskResponse
// @Router /api/v1/templates/sync-tasks/{task_id}/retry [post]
func (h *TemplateManagementHandler) RetrySyncTask(ctx *gin.Context) {
	// 获取任务ID
	taskIDStr := ctx.Param("task_id")
	if taskIDStr == "" {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	// 调用服务层
	err = h.templateManagementService.RetrySyncTask(ctx.Request.Context(), taskID)
	if err != nil {
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, map[string]interface{}{
		"task_id": taskID,
		"status":  "pending",
	})
}

// ListTemplateInstances 列出模板实例
// @Summary 列出模板实例
// @Description 列出指定模板的所有实例
// @Tags 模板管理
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} v1.ListTemplateInstancesResponse
// @Router /api/v1/templates/{id}/instances [get]
func (h *TemplateManagementHandler) ListTemplateInstances(ctx *gin.Context) {
	// 获取模板ID
	idStr := ctx.Param("id")
	if idStr == "" {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	templateID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	// 调用服务层
	data, err := h.templateManagementService.ListTemplateInstances(ctx.Request.Context(), templateID)
	if err != nil {
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, data)
}
