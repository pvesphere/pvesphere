package handler

import (
	"net/http"

	v1 "pvesphere/api/v1"
	"pvesphere/internal/service"
	"pvesphere/pkg/proxmox"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PveAuthHandler struct {
	*Handler
	clusterService service.PveClusterService
}

func NewPveAuthHandler(handler *Handler, clusterService service.PveClusterService) *PveAuthHandler {
	return &PveAuthHandler{
		Handler:        handler,
		clusterService: clusterService,
	}
}

// GetAccessTicket godoc
// @Summary 获取 Proxmox 高权限票据（/access/ticket）
// @Description 按照 Proxmox 原生 /api2/json/access/ticket 接口封装，使用用户名/密码获取 ticket 和 CSRFPreventionToken。api_url 从集群表中根据 cluster_id 获取。
// @Tags PVE认证模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.GetAccessTicketRequest true "params"
// @Success 200 {object} v1.GetAccessTicketResponse
// @Router /api/v1/pve/access/ticket [post]
func (h *PveAuthHandler) GetAccessTicket(ctx *gin.Context) {
	req := new(v1.GetAccessTicketRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	// 通过 cluster_id 获取集群信息，提取 api_url
	cluster, err := h.clusterService.GetCluster(ctx, req.ClusterID)
	if err != nil {
		h.logger.WithContext(ctx).Error("failed to get cluster", zap.Error(err), zap.Int64("cluster_id", req.ClusterID))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}
	if cluster == nil {
		v1.HandleError(ctx, http.StatusNotFound, v1.ErrNotFound, nil)
		return
	}

	// 使用集群的 api_url 调用 Proxmox 接口
	result, err := proxmox.GetAccessTicket(ctx, cluster.ApiUrl, req.Username, req.Realm, req.Password)
	if err != nil {
		h.logger.WithContext(ctx).Error("failed to get proxmox access ticket", zap.Error(err),
			zap.Int64("cluster_id", req.ClusterID),
			zap.String("api_url", cluster.ApiUrl))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	data := map[string]interface{}{
		"username":            result.Username,
		"ticket":              result.Ticket,
		"CSRFPreventionToken": result.CSRFPreventionToken,
	}

	v1.HandleSuccess(ctx, data)
}


