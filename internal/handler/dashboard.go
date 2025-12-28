package handler

import (
	"net/http"

	v1 "pvesphere/api/v1"
	"pvesphere/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type DashboardHandler struct {
	*Handler
	dashboardService service.DashboardService
}

func NewDashboardHandler(handler *Handler, dashboardService service.DashboardService) *DashboardHandler {
	return &DashboardHandler{
		Handler:          handler,
		dashboardService: dashboardService,
	}
}

// GetScopes godoc
// @Summary 获取可选集群列表
// @Tags Dashboard模块
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} v1.DashboardScopesResponse
// @Router /api/v1/dashboard/scopes [get]
func (h *DashboardHandler) GetScopes(ctx *gin.Context) {
	data, err := h.dashboardService.GetScopes(ctx)
	if err != nil {
		h.logger.WithContext(ctx).Error("dashboardService.GetScopes error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, data)
}

// GetOverview godoc
// @Summary 获取全局概览
// @Tags Dashboard模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param scope query string false "范围: all 或 cluster" default(all)
// @Param cluster_id query int false "集群ID（当 scope 为 cluster 时使用）"
// @Success 200 {object} v1.DashboardOverviewResponse
// @Router /api/v1/dashboard/overview [get]
func (h *DashboardHandler) GetOverview(ctx *gin.Context) {
	req := new(v1.DashboardOverviewRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	// 设置默认值
	if req.Scope == "" {
		req.Scope = "all"
	}

	data, err := h.dashboardService.GetOverview(ctx, req)
	if err != nil {
		h.logger.WithContext(ctx).Error("dashboardService.GetOverview error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, data)
}

// GetResources godoc
// @Summary 获取资源使用率
// @Tags Dashboard模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param scope query string false "范围: all 或 cluster" default(all)
// @Param cluster_id query int false "集群ID（当 scope 为 cluster 时使用）"
// @Success 200 {object} v1.DashboardResourcesResponse
// @Router /api/v1/dashboard/resources [get]
func (h *DashboardHandler) GetResources(ctx *gin.Context) {
	req := new(v1.DashboardResourcesRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	// 设置默认值
	if req.Scope == "" {
		req.Scope = "all"
	}

	data, err := h.dashboardService.GetResources(ctx, req)
	if err != nil {
		h.logger.WithContext(ctx).Error("dashboardService.GetResources error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, data)
}

// GetHotspots godoc
// @Summary 获取压力和风险焦点
// @Tags Dashboard模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param scope query string false "范围: all 或 cluster" default(all)
// @Param cluster_id query int false "集群ID（当 scope 为 cluster 时使用）"
// @Param limit query int false "Top N 数量" default(5)
// @Success 200 {object} v1.DashboardHotspotsResponse
// @Router /api/v1/dashboard/hotspots [get]
func (h *DashboardHandler) GetHotspots(ctx *gin.Context) {
	req := new(v1.DashboardHotspotsRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	// 设置默认值
	if req.Scope == "" {
		req.Scope = "all"
	}
	if req.Limit <= 0 {
		req.Limit = 5
	}

	data, err := h.dashboardService.GetHotspots(ctx, req)
	if err != nil {
		h.logger.WithContext(ctx).Error("dashboardService.GetHotspots error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, data)
}

// GetOperations godoc
// @Summary 获取运行中的操作
// @Tags Dashboard模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param scope query string false "范围: all 或 cluster" default(all)
// @Param cluster_id query int false "集群ID（当 scope 为 cluster 时使用）"
// @Success 200 {object} v1.DashboardOperationsResponse
// @Router /api/v1/dashboard/operations [get]
func (h *DashboardHandler) GetOperations(ctx *gin.Context) {
	req := new(v1.DashboardOperationsRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	// 设置默认值
	if req.Scope == "" {
		req.Scope = "all"
	}

	data, err := h.dashboardService.GetOperations(ctx, req)
	if err != nil {
		h.logger.WithContext(ctx).Error("dashboardService.GetOperations error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, data)
}

