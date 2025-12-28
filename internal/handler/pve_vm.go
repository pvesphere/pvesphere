package handler

import (
	"fmt"
	"net/url"
	"strconv"

	"net/http"
	v1 "pvesphere/api/v1"
	"pvesphere/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type PveVMHandler struct {
	*Handler
	vmService service.PveVMService
}

func NewPveVMHandler(handler *Handler, vmService service.PveVMService) *PveVMHandler {
	return &PveVMHandler{
		Handler:   handler,
		vmService: vmService,
	}
}

// CreateVM godoc
// @Summary 创建虚拟机（仅数据库记录）
// @Description 仅创建数据库记录，用于手动同步或导入场景，不调用 Proxmox API
// @Tags PVE虚拟机模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.CreateVMRequest true "params"
// @Success 200 {object} v1.Response
// @Router /api/v1/vms [post]
func (h *PveVMHandler) CreateVM(ctx *gin.Context) {
	req := new(v1.CreateVMRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.vmService.CreateVM(ctx, req); err != nil {
		h.logger.WithContext(ctx).Error("vmService.CreateVM error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// CreateVMInProxmox godoc
// @Summary 创建虚拟机（完整流程）
// @Description 调用 Proxmox API 创建虚拟机并自动创建数据库记录，这是最常用的场景
// @Tags PVE虚拟机模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.CreateVMRequest true "params"
// @Success 200 {object} v1.Response
// @Router /api/v1/vms/create [post]
func (h *PveVMHandler) CreateVMInProxmox(ctx *gin.Context) {
	req := new(v1.CreateVMRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.vmService.CreateVMInProxmox(ctx, req); err != nil {
		h.logger.WithContext(ctx).Error("vmService.CreateVMInProxmox error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// UpdateVM godoc
// @Summary 更新虚拟机
// @Tags PVE虚拟机模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "虚拟机ID"
// @Param request body v1.UpdateVMRequest true "params"
// @Success 200 {object} v1.Response
// @Router /api/v1/vms/{id} [put]
func (h *PveVMHandler) UpdateVM(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	req := new(v1.UpdateVMRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.vmService.UpdateVM(ctx, id, req); err != nil {
		h.logger.WithContext(ctx).Error("vmService.UpdateVM error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// DeleteVM godoc
// @Summary 删除虚拟机
// @Tags PVE虚拟机模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "虚拟机ID"
// @Success 200 {object} v1.Response
// @Router /api/v1/vms/{id} [delete]
func (h *PveVMHandler) DeleteVM(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.vmService.DeleteVM(ctx, id); err != nil {
		h.logger.WithContext(ctx).Error("vmService.DeleteVM error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// GetVM godoc
// @Summary 获取虚拟机详情
// @Tags PVE虚拟机模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "虚拟机ID"
// @Success 200 {object} v1.GetVMResponse
// @Router /api/v1/vms/{id} [get]
func (h *PveVMHandler) GetVM(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	vm, err := h.vmService.GetVM(ctx, id)
	if err != nil {
		h.logger.WithContext(ctx).Error("vmService.GetVM error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, vm)
}

// ListVMs godoc
// @Summary 获取虚拟机列表
// @Tags PVE虚拟机模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param cluster_name query string false "集群名称"
// @Param node_name query string false "节点名称"
// @Param status query string false "状态"
// @Param app_id query string false "应用ID"
// @Success 200 {object} v1.ListVMResponse
// @Router /api/v1/vms [get]
func (h *PveVMHandler) ListVMs(ctx *gin.Context) {
	req := new(v1.ListVMRequest)
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

	data, err := h.vmService.ListVMs(ctx, req)
	if err != nil {
		h.logger.WithContext(ctx).Error("vmService.ListVMs error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, data)
}

// StartVM godoc
// @Summary 启动虚拟机
// @Tags PVE虚拟机模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "虚拟机ID"
// @Success 200 {object} v1.Response
// @Router /api/v1/vms/{id}/start [post]
func (h *PveVMHandler) StartVM(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.vmService.StartVM(ctx, id); err != nil {
		h.logger.WithContext(ctx).Error("vmService.StartVM error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// StopVM godoc
// @Summary 停止虚拟机
// @Tags PVE虚拟机模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "虚拟机ID"
// @Success 200 {object} v1.Response
// @Router /api/v1/vms/{id}/stop [post]
func (h *PveVMHandler) StopVM(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.vmService.StopVM(ctx, id); err != nil {
		h.logger.WithContext(ctx).Error("vmService.StopVM error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// GetVMCurrentConfig godoc
// @Summary 获取虚拟机当前配置
// @Tags PVE虚拟机模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param vm_id query int true "虚拟机ID"
// @Success 200 {object} v1.GetVMCurrentConfigResponse
// @Router /api/v1/vms/config [get]
func (h *PveVMHandler) GetVMCurrentConfig(ctx *gin.Context) {
	req := new(v1.GetVMCurrentConfigRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	config, err := h.vmService.GetVMCurrentConfig(ctx, req.VMID)
	if err != nil {
		h.logger.WithContext(ctx).Error("vmService.GetVMCurrentConfig error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, config)
}

// GetVMPendingConfig godoc
// @Summary 获取虚拟机pending配置
// @Tags PVE虚拟机模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param vm_id query int true "虚拟机ID"
// @Success 200 {object} v1.GetVMPendingConfigResponse
// @Router /api/v1/vms/config/pending [get]
func (h *PveVMHandler) GetVMPendingConfig(ctx *gin.Context) {
	req := new(v1.GetVMPendingConfigRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	config, err := h.vmService.GetVMPendingConfig(ctx, req.VMID)
	if err != nil {
		h.logger.WithContext(ctx).Error("vmService.GetVMPendingConfig error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, config)
}

// UpdateVMConfig godoc
// @Summary 更新虚拟机配置
// @Tags PVE虚拟机模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.UpdateVMConfigRequest true "params"
// @Success 200 {object} v1.UpdateVMConfigResponse
// @Router /api/v1/vms/config [put]
func (h *PveVMHandler) UpdateVMConfig(ctx *gin.Context) {
	req := new(v1.UpdateVMConfigRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.vmService.UpdateVMConfig(ctx, req); err != nil {
		h.logger.WithContext(ctx).Error("vmService.UpdateVMConfig error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// GetVMStatus godoc
// @Summary 获取虚拟机状态
// @Tags PVE虚拟机模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param vm_id query int true "虚拟机ID"
// @Success 200 {object} v1.GetVMStatusResponse
// @Router /api/v1/vms/status [get]
func (h *PveVMHandler) GetVMStatus(ctx *gin.Context) {
	req := new(v1.GetVMStatusRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	status, err := h.vmService.GetVMStatus(ctx, req.VMID)
	if err != nil {
		h.logger.WithContext(ctx).Error("vmService.GetVMStatus error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, status)
}

// GetVMRRDData godoc
// @Summary 获取虚拟机RRD监控数据
// @Tags PVE虚拟机模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param vm_id query int true "虚拟机ID"
// @Param timeframe query string true "时间范围" Enums(hour, day, week, month, year)
// @Param cf query string true "聚合函数" Enums(AVERAGE, MAX)
// @Success 200 {object} v1.GetVMRRDDataResponse
// @Router /api/v1/vms/rrd [get]
func (h *PveVMHandler) GetVMRRDData(ctx *gin.Context) {
	req := new(v1.GetVMRRDDataRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	rrdData, err := h.vmService.GetVMRRDData(ctx, req.VMID, req.Timeframe, req.Cf)
	if err != nil {
		h.logger.WithContext(ctx).Error("vmService.GetVMRRDData error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, rrdData)
}

// GetVMConsole godoc
// @Summary 获取虚拟机 Console（VNCProxy）
// @Description 调用 Proxmox vncproxy 获取 port/ticket 等信息，用于 noVNC 连接
// @Tags PVE虚拟机模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.GetVMConsoleRequest true "params"
// @Success 200 {object} v1.GetVMConsoleResponse
// @Router /api/v1/vms/console [post]
func (h *PveVMHandler) GetVMConsole(ctx *gin.Context) {
	req := new(v1.GetVMConsoleRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	data, err := h.vmService.GetVMConsole(ctx, req)
	if err != nil {
		h.logger.WithContext(ctx).Error("vmService.GetVMConsole error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	// 组装同域 websocket 连接地址（用于 noVNC）
	// 这里返回我们后端的 ws 代理地址，避免跨域/证书/鉴权问题
	wsToken, _ := data["ws_token"].(string)
	if wsToken != "" {
		// 兼容前端可能使用 token 字段名
		data["token"] = wsToken

		scheme := "ws"
		proto := ctx.Request.Header.Get("X-Forwarded-Proto")
		if proto == "https" || proto == "wss" {
			scheme = "wss"
		} else if ctx.Request.TLS != nil {
			scheme = "wss"
		}

		host := ctx.Request.Host
		if xfHost := ctx.Request.Header.Get("X-Forwarded-Host"); xfHost != "" {
			host = xfHost
		}

		wsURL := fmt.Sprintf("%s://%s/api/v1/vms/console/ws?token=%s", scheme, host, url.QueryEscape(wsToken))
		data["ws_url"] = wsURL
	}

	v1.HandleSuccess(ctx, data)
}

// VMConsoleWS godoc
// @Summary 虚拟机 Console WebSocket（VNC WebSocket 代理）
// @Description 同域 WS 代理到 Proxmox vncwebsocket，供 noVNC 直接连接
// @Tags PVE虚拟机模块
// @Security Bearer
// @Param token query string true "ws_token（由 /api/v1/vms/console 返回）"
// @Router /api/v1/vms/console/ws [get]
func (h *PveVMHandler) VMConsoleWS(ctx *gin.Context) {
	token := ctx.Query("token")
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	clientConn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return
	}
	defer clientConn.Close()

	proxmoxConn, err := h.vmService.DialVMConsoleWebsocket(ctx, token)
	if err != nil {
		_ = clientConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "invalid console token"))
		return
	}
	defer proxmoxConn.Close()

	errCh := make(chan error, 2)
	proxy := func(src, dst *websocket.Conn) {
		for {
			mt, msg, err := src.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			if err := dst.WriteMessage(mt, msg); err != nil {
				errCh <- err
				return
			}
		}
	}

	go proxy(clientConn, proxmoxConn)
	go proxy(proxmoxConn, clientConn)

	<-errCh
}

// MigrateVM godoc
// @Summary 迁移虚拟机（同集群）
// @Tags PVE虚拟机模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.MigrateVMRequest true "迁移请求"
// @Success 200 {object} v1.MigrateVMResponse
// @Router /api/v1/vms/migrate [post]
func (h *PveVMHandler) MigrateVM(ctx *gin.Context) {
	req := new(v1.MigrateVMRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	result, err := h.vmService.MigrateVM(ctx, req)
	if err != nil {
		h.logger.WithContext(ctx).Error("vmService.MigrateVM error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, result)
}

// RemoteMigrateVM godoc
// @Summary 远程迁移虚拟机（跨集群）
// @Tags PVE虚拟机模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.RemoteMigrateVMRequest true "远程迁移请求"
// @Success 200 {object} v1.RemoteMigrateVMResponse
// @Router /api/v1/vms/remote-migrate [post]
func (h *PveVMHandler) RemoteMigrateVM(ctx *gin.Context) {
	req := new(v1.RemoteMigrateVMRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	result, err := h.vmService.RemoteMigrateVM(ctx, req)
	if err != nil {
		h.logger.WithContext(ctx).Error("vmService.RemoteMigrateVM error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, result)
}

// CreateBackup godoc
// @Summary 创建虚拟机备份
// @Description 使用 Proxmox vzdump API 创建虚拟机备份
// @Tags PVE虚拟机模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.CreateBackupRequest true "创建备份请求"
// @Success 200 {object} v1.CreateBackupResponse
// @Router /api/v1/vms/backup [post]
func (h *PveVMHandler) CreateBackup(ctx *gin.Context) {
	req := new(v1.CreateBackupRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		h.logger.WithContext(ctx).Error("CreateBackup bind json error", zap.Error(err))
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	data, err := h.vmService.CreateBackup(ctx, req)
	if err != nil {
		h.logger.WithContext(ctx).Error("vmService.CreateBackup error", zap.Error(err))
		if err == v1.ErrNotFound {
			v1.HandleError(ctx, http.StatusNotFound, err, nil)
			return
		}
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, data)
}

// DeleteBackup godoc
// @Summary 删除虚拟机备份
// @Description 删除指定存储中的备份文件
// @Tags PVE虚拟机模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.DeleteBackupRequest true "删除备份请求"
// @Success 200 {object} v1.DeleteBackupResponse
// @Router /api/v1/vms/backup [delete]
func (h *PveVMHandler) DeleteBackup(ctx *gin.Context) {
	req := new(v1.DeleteBackupRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		h.logger.WithContext(ctx).Error("DeleteBackup bind json error", zap.Error(err))
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.vmService.DeleteBackup(ctx, req); err != nil {
		h.logger.WithContext(ctx).Error("vmService.DeleteBackup error", zap.Error(err))
		if err == v1.ErrNotFound {
			v1.HandleError(ctx, http.StatusNotFound, err, nil)
			return
		}
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// GetVMCloudInit godoc
// @Summary 获取虚拟机 CloudInit 配置
// @Description 获取虚拟机的 CloudInit 配置，包含当前和待处理的值
// @Tags PVE虚拟机模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param vm_id query int true "虚拟机ID"
// @Param node_id query int true "节点ID"
// @Success 200 {object} v1.GetVMCloudInitResponse
// @Router /api/v1/vms/cloudinit [get]
func (h *PveVMHandler) GetVMCloudInit(ctx *gin.Context) {
	req := new(v1.GetVMCloudInitRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		h.logger.WithContext(ctx).Error("GetVMCloudInit bind query error", zap.Error(err))
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	config, err := h.vmService.GetVMCloudInit(ctx, req)
	if err != nil {
		h.logger.WithContext(ctx).Error("vmService.GetVMCloudInit error", zap.Error(err))
		if err == v1.ErrNotFound {
			v1.HandleError(ctx, http.StatusNotFound, err, nil)
			return
		}
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, config)
}

// UpdateVMCloudInit godoc
// @Summary 更新虚拟机 CloudInit 配置
// @Description 重新生成和更改 cloudinit 配置驱动器
// @Tags PVE虚拟机模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.UpdateVMCloudInitRequest true "更新 CloudInit 配置请求"
// @Success 200 {object} v1.UpdateVMCloudInitResponse
// @Router /api/v1/vms/cloudinit [put]
func (h *PveVMHandler) UpdateVMCloudInit(ctx *gin.Context) {
	req := new(v1.UpdateVMCloudInitRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		h.logger.WithContext(ctx).Error("UpdateVMCloudInit bind json error", zap.Error(err))
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.vmService.UpdateVMCloudInit(ctx, req); err != nil {
		h.logger.WithContext(ctx).Error("vmService.UpdateVMCloudInit error", zap.Error(err))
		if err == v1.ErrNotFound {
			v1.HandleError(ctx, http.StatusNotFound, err, nil)
			return
		}
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}
