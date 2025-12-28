package handler

import (
	"net/http"
	"strconv"

	v1 "pvesphere/api/v1"
	"pvesphere/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PveTemplateHandler struct {
	*Handler
	templateService service.PveTemplateService
}

func NewPveTemplateHandler(handler *Handler, templateService service.PveTemplateService) *PveTemplateHandler {
	return &PveTemplateHandler{
		Handler:         handler,
		templateService: templateService,
	}
}

// CreateTemplate godoc
// @Summary 创建模板
// @Tags 模板管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.CreateTemplateRequest true "params"
// @Success 200 {object} v1.Response
// @Router /api/v1/templates [post]
func (h *PveTemplateHandler) CreateTemplate(ctx *gin.Context) {
	req := new(v1.CreateTemplateRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.templateService.CreateTemplate(ctx, req); err != nil {
		h.logger.WithContext(ctx).Error("templateService.CreateTemplate error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// UpdateTemplate godoc
// @Summary 更新模板
// @Tags 模板管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "模板ID"
// @Param request body v1.UpdateTemplateRequest true "params"
// @Success 200 {object} v1.Response
// @Router /api/v1/templates/{id} [put]
func (h *PveTemplateHandler) UpdateTemplate(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	req := new(v1.UpdateTemplateRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.templateService.UpdateTemplate(ctx, id, req); err != nil {
		h.logger.WithContext(ctx).Error("templateService.UpdateTemplate error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// DeleteTemplate godoc
// @Summary 删除模板
// @Tags 模板管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "模板ID"
// @Success 200 {object} v1.Response
// @Router /api/v1/templates/{id} [delete]
func (h *PveTemplateHandler) DeleteTemplate(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.templateService.DeleteTemplate(ctx, id); err != nil {
		h.logger.WithContext(ctx).Error("templateService.DeleteTemplate error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// GetTemplate godoc
// @Summary 获取模板详情
// @Tags 模板管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "模板ID"
// @Success 200 {object} v1.GetTemplateResponse
// @Router /api/v1/templates/{id} [get]
func (h *PveTemplateHandler) GetTemplate(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	data, err := h.templateService.GetTemplate(ctx, id)
	if err != nil {
		h.logger.WithContext(ctx).Error("templateService.GetTemplate error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, data)
}

// ListTemplates godoc
// @Summary 获取模板列表
// @Tags 模板管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param cluster_id query int false "集群ID"
// @Success 200 {object} v1.ListTemplateResponse
// @Router /api/v1/templates [get]
func (h *PveTemplateHandler) ListTemplates(ctx *gin.Context) {
	req := new(v1.ListTemplateRequest)
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

	data, err := h.templateService.ListTemplates(ctx, req)
	if err != nil {
		h.logger.WithContext(ctx).Error("templateService.ListTemplates error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, data)
}
