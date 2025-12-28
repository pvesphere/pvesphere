package handler

import (
	"strconv"
	
	"github.com/gin-gonic/gin"
	"pvesphere/api/v1"
	"pvesphere/internal/service"
	"go.uber.org/zap"
	"net/http"
)

type PveStorageHandler struct {
	*Handler
	storageService service.PveStorageService
}

func NewPveStorageHandler(handler *Handler, storageService service.PveStorageService) *PveStorageHandler {
	return &PveStorageHandler{
		Handler:        handler,
		storageService: storageService,
	}
}

// CreateStorage godoc
// @Summary 创建存储
// @Tags PVE存储模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.CreateStorageRequest true "params"
// @Success 200 {object} v1.Response
// @Router /api/v1/storages [post]
func (h *PveStorageHandler) CreateStorage(ctx *gin.Context) {
	req := new(v1.CreateStorageRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.storageService.CreateStorage(ctx, req); err != nil {
		h.logger.WithContext(ctx).Error("storageService.CreateStorage error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// UpdateStorage godoc
// @Summary 更新存储
// @Tags PVE存储模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "存储ID"
// @Param request body v1.UpdateStorageRequest true "params"
// @Success 200 {object} v1.Response
// @Router /api/v1/storages/{id} [put]
func (h *PveStorageHandler) UpdateStorage(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	req := new(v1.UpdateStorageRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.storageService.UpdateStorage(ctx, id, req); err != nil {
		h.logger.WithContext(ctx).Error("storageService.UpdateStorage error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// DeleteStorage godoc
// @Summary 删除存储
// @Tags PVE存储模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "存储ID"
// @Success 200 {object} v1.Response
// @Router /api/v1/storages/{id} [delete]
func (h *PveStorageHandler) DeleteStorage(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.storageService.DeleteStorage(ctx, id); err != nil {
		h.logger.WithContext(ctx).Error("storageService.DeleteStorage error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// GetStorage godoc
// @Summary 获取存储详情
// @Tags PVE存储模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "存储ID"
// @Success 200 {object} v1.GetStorageResponse
// @Router /api/v1/storages/{id} [get]
func (h *PveStorageHandler) GetStorage(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	storage, err := h.storageService.GetStorage(ctx, id)
	if err != nil {
		h.logger.WithContext(ctx).Error("storageService.GetStorage error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, storage)
}

// ListStorages godoc
// @Summary 获取存储列表
// @Tags PVE存储模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param cluster_id query int false "集群ID"
// @Param node_name query string false "节点名称"
// @Param type query string false "存储类型"
// @Param storage_name query string false "存储名称"
// @Success 200 {object} v1.ListStorageResponse
// @Router /api/v1/storages [get]
func (h *PveStorageHandler) ListStorages(ctx *gin.Context) {
	req := new(v1.ListStorageRequest)
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

	data, err := h.storageService.ListStorages(ctx, req)
	if err != nil {
		h.logger.WithContext(ctx).Error("storageService.ListStorages error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, data)
}

