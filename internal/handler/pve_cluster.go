package handler

import (
	"strconv"

	"net/http"
	v1 "pvesphere/api/v1"
	"pvesphere/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PveClusterHandler struct {
	*Handler
	clusterService service.PveClusterService
}

func NewPveClusterHandler(handler *Handler, clusterService service.PveClusterService) *PveClusterHandler {
	return &PveClusterHandler{
		Handler:        handler,
		clusterService: clusterService,
	}
}

// CreateCluster godoc
// @Summary 创建集群
// @Tags PVE集群模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.CreateClusterRequest true "params"
// @Success 200 {object} v1.Response
// @Router /api/v1/clusters [post]
func (h *PveClusterHandler) CreateCluster(ctx *gin.Context) {
	req := new(v1.CreateClusterRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.clusterService.CreateCluster(ctx, req); err != nil {
		h.logger.WithContext(ctx).Error("clusterService.CreateCluster error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// UpdateCluster godoc
// @Summary 更新集群
// @Tags PVE集群模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "集群ID"
// @Param request body v1.UpdateClusterRequest true "params"
// @Success 200 {object} v1.Response
// @Router /api/v1/clusters/{id} [put]
func (h *PveClusterHandler) UpdateCluster(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	req := new(v1.UpdateClusterRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.clusterService.UpdateCluster(ctx, id, req); err != nil {
		h.logger.WithContext(ctx).Error("clusterService.UpdateCluster error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// DeleteCluster godoc
// @Summary 删除集群
// @Tags PVE集群模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "集群ID"
// @Success 200 {object} v1.Response
// @Router /api/v1/clusters/{id} [delete]
func (h *PveClusterHandler) DeleteCluster(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.clusterService.DeleteCluster(ctx, id); err != nil {
		h.logger.WithContext(ctx).Error("clusterService.DeleteCluster error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// GetCluster godoc
// @Summary 获取集群详情
// @Tags PVE集群模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "集群ID"
// @Success 200 {object} v1.GetClusterResponse
// @Router /api/v1/clusters/{id} [get]
func (h *PveClusterHandler) GetCluster(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	cluster, err := h.clusterService.GetCluster(ctx, id)
	if err != nil {
		h.logger.WithContext(ctx).Error("clusterService.GetCluster error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, cluster)
}

// ListClusters godoc
// @Summary 获取集群列表
// @Tags PVE集群模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param env query string false "环境"
// @Param region query string false "区域"
// @Success 200 {object} v1.ListClusterResponse
// @Router /api/v1/clusters [get]
func (h *PveClusterHandler) ListClusters(ctx *gin.Context) {
	req := new(v1.ListClusterRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	// 验证 PageSize 最大值
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	data, err := h.clusterService.ListClusters(ctx, req)
	if err != nil {
		h.logger.WithContext(ctx).Error("clusterService.ListClusters error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, data)
}

// GetClusterStatus godoc
// @Summary 获取集群状态
// @Tags PVE集群模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param cluster_id query int true "集群ID"
// @Success 200 {object} v1.GetClusterStatusResponse
// @Router /api/v1/clusters/status [get]
func (h *PveClusterHandler) GetClusterStatus(ctx *gin.Context) {
	req := new(v1.GetClusterStatusRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	status, err := h.clusterService.GetClusterStatus(ctx, req.ClusterID)
	if err != nil {
		h.logger.WithContext(ctx).Error("clusterService.GetClusterStatus error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, status)
}

// GetClusterResources godoc
// @Summary 获取集群资源
// @Tags PVE集群模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param cluster_id query int true "集群ID"
// @Success 200 {object} v1.GetClusterResourcesResponse
// @Router /api/v1/clusters/resources [get]
func (h *PveClusterHandler) GetClusterResources(ctx *gin.Context) {
	req := new(v1.GetClusterResourcesRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	resources, err := h.clusterService.GetClusterResources(ctx, req.ClusterID)
	if err != nil {
		h.logger.WithContext(ctx).Error("clusterService.GetClusterResources error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, resources)
}

// VerifyCluster godoc
// @Summary 验证集群连接
// @Description 通过调用 Proxmox /api2/json/version 接口验证集群连接和认证是否正常。支持两种验证方式：1. 通过 cluster_id 验证（从数据库获取集群信息）；2. 通过 api_url + user_id + user_token 直接验证（不依赖数据库）
// @Tags PVE集群模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param cluster_id query int false "集群ID（与 api_url+user_id+user_token 二选一）"
// @Param api_url query string false "API地址（与 cluster_id 二选一）"
// @Param user_id query string false "用户ID（与 cluster_id 二选一）"
// @Param user_token query string false "用户Token（与 cluster_id 二选一）"
// @Success 200 {object} v1.VerifyClusterResponse
// @Router /api/v1/clusters/verify [get]
func (h *PveClusterHandler) VerifyCluster(ctx *gin.Context) {
	req := new(v1.VerifyClusterRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	// 验证参数：必须提供 cluster_id 或者 (api_url + user_id + user_token)
	if req.ClusterID == nil && (req.ApiUrl == "" || req.UserId == "" || req.UserToken == "") {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	var data *v1.VerifyClusterData
	var err error

	if req.ClusterID != nil {
		// 方式1：通过 cluster_id 验证
		data, err = h.clusterService.VerifyCluster(ctx, req.ClusterID)
	} else {
		// 方式2：通过 api_url + user_id + user_token 直接验证
		data, err = h.clusterService.VerifyClusterWithCredentials(ctx, req.ApiUrl, req.UserId, req.UserToken)
	}

	if err != nil {
		h.logger.WithContext(ctx).Error("clusterService.VerifyCluster error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, data)
}
