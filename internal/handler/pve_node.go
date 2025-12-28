package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	v1 "pvesphere/api/v1"
	"pvesphere/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type PveNodeHandler struct {
	*Handler
	nodeService service.PveNodeService
}

func NewPveNodeHandler(handler *Handler, nodeService service.PveNodeService) *PveNodeHandler {
	return &PveNodeHandler{
		Handler:     handler,
		nodeService: nodeService,
	}
}

// CreateNode godoc
// @Summary 创建节点
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.CreateNodeRequest true "params"
// @Success 200 {object} v1.Response
// @Router /api/v1/nodes [post]
func (h *PveNodeHandler) CreateNode(ctx *gin.Context) {
	req := new(v1.CreateNodeRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.nodeService.CreateNode(ctx, req); err != nil {
		h.logger.WithContext(ctx).Error("nodeService.CreateNode error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// UpdateNode godoc
// @Summary 更新节点
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "节点ID"
// @Param request body v1.UpdateNodeRequest true "params"
// @Success 200 {object} v1.Response
// @Router /api/v1/nodes/{id} [put]
func (h *PveNodeHandler) UpdateNode(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	req := new(v1.UpdateNodeRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.nodeService.UpdateNode(ctx, id, req); err != nil {
		h.logger.WithContext(ctx).Error("nodeService.UpdateNode error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// DeleteNode godoc
// @Summary 删除节点
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "节点ID"
// @Success 200 {object} v1.Response
// @Router /api/v1/nodes/{id} [delete]
func (h *PveNodeHandler) DeleteNode(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.nodeService.DeleteNode(ctx, id); err != nil {
		h.logger.WithContext(ctx).Error("nodeService.DeleteNode error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// GetNode godoc
// @Summary 获取节点详情
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "节点ID"
// @Success 200 {object} v1.GetNodeResponse
// @Router /api/v1/nodes/{id} [get]
func (h *PveNodeHandler) GetNode(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	node, err := h.nodeService.GetNode(ctx, id)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.GetNode error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, node)
}

// ListNodes godoc
// @Summary 获取节点列表
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param cluster_id query int false "集群ID"
// @Param env query string false "环境"
// @Param status query string false "状态"
// @Success 200 {object} v1.ListNodeResponse
// @Router /api/v1/nodes [get]
func (h *PveNodeHandler) ListNodes(ctx *gin.Context) {
	req := new(v1.ListNodeRequest)
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

	data, err := h.nodeService.ListNodes(ctx, req)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.ListNodes error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, data)
}

// GetNodeStatus godoc
// @Summary 获取节点状态
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param node_id query int true "节点ID"
// @Success 200 {object} v1.GetNodeStatusResponse
// @Router /api/v1/nodes/status [get]
func (h *PveNodeHandler) GetNodeStatus(ctx *gin.Context) {
	req := new(v1.GetNodeStatusRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	status, err := h.nodeService.GetNodeStatus(ctx, req.NodeID)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.GetNodeStatus error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, status)
}

// SetNodeStatus godoc
// @Summary 设置节点状态（重启/关闭）
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.SetNodeStatusRequest true "params"
// @Success 200 {object} v1.SetNodeStatusResponse
// @Router /api/v1/nodes/status [post]
func (h *PveNodeHandler) SetNodeStatus(ctx *gin.Context) {
	req := new(v1.SetNodeStatusRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	upid, err := h.nodeService.SetNodeStatus(ctx, req.NodeID, req.Command)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.SetNodeStatus error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, map[string]interface{}{
		"upid": upid,
	})
}

// GetNodeServices godoc
// @Summary 获取节点服务列表
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param node_id query int true "节点ID"
// @Success 200 {object} v1.GetNodeServicesResponse
// @Router /api/v1/nodes/services [get]
func (h *PveNodeHandler) GetNodeServices(ctx *gin.Context) {
	req := new(v1.GetNodeServicesRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	services, err := h.nodeService.GetNodeServices(ctx, req.NodeID)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.GetNodeServices error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, services)
}

// StartNodeService godoc
// @Summary 启动节点服务
// @Description 启动指定的节点服务
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.StartNodeServiceRequest true "启动服务请求"
// @Success 200 {object} v1.StartNodeServiceResponse
// @Router /api/v1/nodes/services/start [post]
func (h *PveNodeHandler) StartNodeService(ctx *gin.Context) {
	req := new(v1.StartNodeServiceRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		h.logger.WithContext(ctx).Error("StartNodeService bind json error", zap.Error(err))
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	upid, err := h.nodeService.StartNodeService(ctx, req.NodeID, req.ServiceName)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.StartNodeService error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, map[string]interface{}{
		"upid": upid,
	})
}

// StopNodeService godoc
// @Summary 停止节点服务
// @Description 停止指定的节点服务
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.StopNodeServiceRequest true "停止服务请求"
// @Success 200 {object} v1.StopNodeServiceResponse
// @Router /api/v1/nodes/services/stop [post]
func (h *PveNodeHandler) StopNodeService(ctx *gin.Context) {
	req := new(v1.StopNodeServiceRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		h.logger.WithContext(ctx).Error("StopNodeService bind json error", zap.Error(err))
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	upid, err := h.nodeService.StopNodeService(ctx, req.NodeID, req.ServiceName)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.StopNodeService error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, map[string]interface{}{
		"upid": upid,
	})
}

// RestartNodeService godoc
// @Summary 重启节点服务
// @Description 重启指定的节点服务
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.RestartNodeServiceRequest true "重启服务请求"
// @Success 200 {object} v1.RestartNodeServiceResponse
// @Router /api/v1/nodes/services/restart [post]
func (h *PveNodeHandler) RestartNodeService(ctx *gin.Context) {
	req := new(v1.RestartNodeServiceRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		h.logger.WithContext(ctx).Error("RestartNodeService bind json error", zap.Error(err))
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	upid, err := h.nodeService.RestartNodeService(ctx, req.NodeID, req.ServiceName)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.RestartNodeService error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, map[string]interface{}{
		"upid": upid,
	})
}

// GetNodeNetworks godoc
// @Summary 获取节点网络列表
// @Description 列出节点的可用网络配置
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param node_id query int true "节点ID"
// @Success 200 {object} v1.GetNodeNetworksResponse
// @Router /api/v1/nodes/network [get]
func (h *PveNodeHandler) GetNodeNetworks(ctx *gin.Context) {
	req := new(v1.GetNodeNetworksRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		h.logger.WithContext(ctx).Error("GetNodeNetworks bind query error", zap.Error(err))
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	networks, err := h.nodeService.GetNodeNetworks(ctx, req.NodeID)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.GetNodeNetworks error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, networks)
}

// CreateNodeNetwork godoc
// @Summary 创建网络设备配置
// @Description 创建或更新节点的网络设备配置
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.CreateNodeNetworkRequest true "创建网络配置请求"
// @Success 200 {object} v1.CreateNodeNetworkResponse
// @Router /api/v1/nodes/network [post]
func (h *PveNodeHandler) CreateNodeNetwork(ctx *gin.Context) {
	req := new(v1.CreateNodeNetworkRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		h.logger.WithContext(ctx).Error("CreateNodeNetwork bind json error", zap.Error(err))
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.nodeService.CreateNodeNetwork(ctx, req); err != nil {
		h.logger.WithContext(ctx).Error("nodeService.CreateNodeNetwork error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// ReloadNodeNetwork godoc
// @Summary 重新加载网络配置
// @Description 重新加载节点的网络配置
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.ReloadNodeNetworkRequest true "重新加载网络配置请求"
// @Success 200 {object} v1.ReloadNodeNetworkResponse
// @Router /api/v1/nodes/network [put]
func (h *PveNodeHandler) ReloadNodeNetwork(ctx *gin.Context) {
	req := new(v1.ReloadNodeNetworkRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		h.logger.WithContext(ctx).Error("ReloadNodeNetwork bind json error", zap.Error(err))
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.nodeService.ReloadNodeNetwork(ctx, req.NodeID); err != nil {
		h.logger.WithContext(ctx).Error("nodeService.ReloadNodeNetwork error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// RevertNodeNetwork godoc
// @Summary 恢复网络配置更改
// @Description 恢复节点的网络配置更改
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.RevertNodeNetworkRequest true "恢复网络配置请求"
// @Success 200 {object} v1.RevertNodeNetworkResponse
// @Router /api/v1/nodes/network [delete]
func (h *PveNodeHandler) RevertNodeNetwork(ctx *gin.Context) {
	req := new(v1.RevertNodeNetworkRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		h.logger.WithContext(ctx).Error("RevertNodeNetwork bind json error", zap.Error(err))
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	if err := h.nodeService.RevertNodeNetwork(ctx, req.NodeID); err != nil {
		h.logger.WithContext(ctx).Error("nodeService.RevertNodeNetwork error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// GetNodeRRDData godoc
// @Summary 获取节点RRD监控数据
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param node_id query int true "节点ID"
// @Param timeframe query string true "时间范围" Enums(hour, day, week, month, year)
// @Param cf query string true "聚合函数" Enums(AVERAGE, MAX)
// @Success 200 {object} v1.GetNodeRRDDataResponse
// @Router /api/v1/nodes/rrd [get]
func (h *PveNodeHandler) GetNodeRRDData(ctx *gin.Context) {
	req := new(v1.GetNodeRRDDataRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	rrdData, err := h.nodeService.GetNodeRRDData(ctx, req.NodeID, req.Timeframe, req.Cf)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.GetNodeRRDData error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, rrdData)
}

// GetNodeDisksList godoc
// @Summary 获取节点磁盘列表
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param node_id query int true "节点ID"
// @Param include_partitions query bool false "是否包含分区信息" default(false)
// @Success 200 {object} v1.GetNodeDisksListResponse
// @Router /api/v1/nodes/disks/list [get]
func (h *PveNodeHandler) GetNodeDisksList(ctx *gin.Context) {
	req := new(v1.GetNodeDisksListRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	disks, err := h.nodeService.GetNodeDisksList(ctx, req.NodeID, req.IncludePartitions)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.GetNodeDisksList error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, disks)
}

// GetNodeDisksDirectory godoc
// @Summary 获取节点 Directory 存储
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param node_id query int true "节点ID"
// @Success 200 {object} v1.GetNodeDisksDirectoryResponse
// @Router /api/v1/nodes/disks/directory [get]
func (h *PveNodeHandler) GetNodeDisksDirectory(ctx *gin.Context) {
	req := new(v1.GetNodeDisksDirectoryRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	directories, err := h.nodeService.GetNodeDisksDirectory(ctx, req.NodeID)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.GetNodeDisksDirectory error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, directories)
}

// GetNodeDisksLVM godoc
// @Summary 获取节点 LVM 存储
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param node_id query int true "节点ID"
// @Success 200 {object} v1.GetNodeDisksLVMResponse
// @Router /api/v1/nodes/disks/lvm [get]
func (h *PveNodeHandler) GetNodeDisksLVM(ctx *gin.Context) {
	req := new(v1.GetNodeDisksLVMRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	lvms, err := h.nodeService.GetNodeDisksLVM(ctx, req.NodeID)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.GetNodeDisksLVM error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, lvms)
}

// GetNodeDisksLVMThin godoc
// @Summary 获取节点 LVM-Thin 存储
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param node_id query int true "节点ID"
// @Success 200 {object} v1.GetNodeDisksLVMThinResponse
// @Router /api/v1/nodes/disks/lvmthin [get]
func (h *PveNodeHandler) GetNodeDisksLVMThin(ctx *gin.Context) {
	req := new(v1.GetNodeDisksLVMThinRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	lvmthins, err := h.nodeService.GetNodeDisksLVMThin(ctx, req.NodeID)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.GetNodeDisksLVMThin error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, lvmthins)
}

// GetNodeDisksZFS godoc
// @Summary 获取节点 ZFS 存储
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param node_id query int true "节点ID"
// @Success 200 {object} v1.GetNodeDisksZFSResponse
// @Router /api/v1/nodes/disks/zfs [get]
func (h *PveNodeHandler) GetNodeDisksZFS(ctx *gin.Context) {
	req := new(v1.GetNodeDisksZFSRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	zfss, err := h.nodeService.GetNodeDisksZFS(ctx, req.NodeID)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.GetNodeDisksZFS error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, zfss)
}

// InitGPTDisk godoc
// @Summary 初始化 GPT 磁盘
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.InitGPTDiskRequest true "params"
// @Success 200 {object} v1.InitGPTDiskResponse
// @Router /api/v1/nodes/disks/initgpt [post]
func (h *PveNodeHandler) InitGPTDisk(ctx *gin.Context) {
	req := new(v1.InitGPTDiskRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	upid, err := h.nodeService.InitGPTDisk(ctx, req.NodeID, req.Disk)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.InitGPTDisk error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, upid)
}

// WipeDisk godoc
// @Summary 擦除磁盘或分区
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.WipeDiskRequest true "params"
// @Success 200 {object} v1.WipeDiskResponse
// @Router /api/v1/nodes/disks/wipedisk [put]
func (h *PveNodeHandler) WipeDisk(ctx *gin.Context) {
	req := new(v1.WipeDiskRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	upid, err := h.nodeService.WipeDisk(ctx, req.NodeID, req.Disk, req.Partition)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.WipeDisk error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, upid)
}

// GetNodeStorageStatus godoc
// @Summary 获取节点存储状态
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param node_id query int true "节点ID"
// @Param storage query string true "存储名称"
// @Success 200 {object} v1.GetStorageStatusResponse
// @Router /api/v1/nodes/storage/status [get]
func (h *PveNodeHandler) GetNodeStorageStatus(ctx *gin.Context) {
	req := new(v1.GetStorageStatusRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	data, err := h.nodeService.GetNodeStorageStatus(ctx, req.NodeID, req.Storage)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.GetNodeStorageStatus error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, data)
}

// GetNodeStorageRRDData godoc
// @Summary 获取节点存储 RRD 监控数据
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param node_id query int true "节点ID"
// @Param storage query string true "存储名称"
// @Param timeframe query string true "时间范围" Enums(hour, day, week, month, year)
// @Param cf query string true "聚合函数" Enums(AVERAGE, MAX)
// @Success 200 {object} v1.GetStorageRRDDataResponse
// @Router /api/v1/nodes/storage/rrd [get]
func (h *PveNodeHandler) GetNodeStorageRRDData(ctx *gin.Context) {
	req := new(v1.GetStorageRRDDataRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	data, err := h.nodeService.GetNodeStorageRRDData(ctx, req.NodeID, req.Storage, req.Timeframe, req.Cf)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.GetNodeStorageRRDData error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, data)
}

// GetNodeStorageContent godoc
// @Summary 获取节点存储内容列表
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param node_id query int true "节点ID"
// @Param storage query string true "存储名称"
// @Param content query string false "内容类型过滤，如 images,iso,backup"
// @Success 200 {object} v1.GetStorageContentResponse
// @Router /api/v1/nodes/storage/content [get]
func (h *PveNodeHandler) GetNodeStorageContent(ctx *gin.Context) {
	req := new(v1.GetStorageContentRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	items, err := h.nodeService.GetNodeStorageContent(ctx, req.NodeID, req.Storage, req.Content)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.GetNodeStorageContent error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, items)
}

// GetNodeStorageVolume godoc
// @Summary 获取节点存储卷属性
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param node_id query int true "节点ID"
// @Param storage query string true "存储名称"
// @Param volume query string true "卷标识（volume ID）"
// @Success 200 {object} v1.GetStorageVolumeResponse
// @Router /api/v1/nodes/storage/content/detail [get]
func (h *PveNodeHandler) GetNodeStorageVolume(ctx *gin.Context) {
	req := new(v1.GetStorageVolumeRequest)
	if err := ctx.ShouldBindQuery(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	info, err := h.nodeService.GetNodeStorageVolume(ctx, req.NodeID, req.Storage, req.Volume)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.GetNodeStorageVolume error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, info)
}

// UploadNodeStorageContent godoc
// @Summary 上传存储内容（模板 / ISO / OVA / VM 镜像）
// @Tags PVE节点模块
// @Accept multipart/form-data
// @Produce json
// @Security Bearer
// @Param node_id formData int true "节点ID"
// @Param storage formData string true "存储名称"
// @Param content formData string false "内容类型" Enums(iso,vztmpl,backup,images)
// @Param file formData file true "上传文件"
// @Success 200 {object} v1.Response
// @Router /api/v1/nodes/storage/upload [post]
func (h *PveNodeHandler) UploadNodeStorageContent(ctx *gin.Context) {
	nodeIDStr := ctx.PostForm("node_id")
	if nodeIDStr == "" {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}
	nodeID, err := strconv.ParseInt(nodeIDStr, 10, 64)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	storage := ctx.PostForm("storage")
	if storage == "" {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}
	content := ctx.PostForm("content")

	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}
	defer file.Close()

	result, err := h.nodeService.UploadNodeStorageContent(ctx, nodeID, storage, content, header.Filename, file)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.UploadNodeStorageContent error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, result)
}

// DeleteNodeStorageContent godoc
// @Summary 删除存储内容（镜像 / ISO / OVA / VM 镜像等）
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param node_id query int true "节点ID"
// @Param storage query string true "存储名称"
// @Param volume query string true "卷标识（volume ID，需要完整路径，例如：/local-dir:iso/baohe_pro_8_51_0_1619.iso）"
// @Param delay query int false "延迟删除时间（秒）"
// @Success 200 {object} v1.Response
// @Router /api/v1/nodes/storage/content [delete]
func (h *PveNodeHandler) DeleteNodeStorageContent(ctx *gin.Context) {
	var req v1.DeleteStorageContentRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	var delay *int
	if req.Delay != nil && *req.Delay > 0 {
		delay = req.Delay
	}

	if err := h.nodeService.DeleteNodeStorageContent(ctx, req.NodeID, req.Storage, req.Volume, delay); err != nil {
		h.logger.WithContext(ctx).Error("nodeService.DeleteNodeStorageContent error", zap.Error(err))
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}

	v1.HandleSuccess(ctx, nil)
}

// GetNodeConsole godoc
// @Summary 获取节点控制台信息
// @Description 获取节点控制台信息，支持 termproxy（终端）和 vncshell（VNC图形界面）两种模式
// @Tags PVE节点模块
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body v1.GetNodeConsoleRequest true "params"
// @Success 200 {object} v1.GetNodeConsoleResponse
// @Router /api/v1/nodes/console [post]
func (h *PveNodeHandler) GetNodeConsole(ctx *gin.Context) {
	req := new(v1.GetNodeConsoleRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}

	data, err := h.nodeService.GetNodeConsole(ctx, req)
	if err != nil {
		h.logger.WithContext(ctx).Error("nodeService.GetNodeConsole error", zap.Error(err))
		// 检查是否是预定义的错误类型
		if err == v1.ErrInternalServerError || err == v1.ErrNotFound || err == v1.ErrBadRequest {
			v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		} else {
			// 返回自定义错误信息（包含 Proxmox API 的详细错误）
			ctx.JSON(http.StatusInternalServerError, v1.Response{
				Code:    500,
				Message: err.Error(),
				Data:    nil,
			})
		}
		return
	}

	// 如果返回了 ws_token（无论是 vncshell 还是 termproxy），组装同域 websocket 连接地址（用于 noVNC/终端）
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

			wsURL := fmt.Sprintf("%s://%s/api/v1/nodes/console/ws?token=%s", scheme, host, url.QueryEscape(wsToken))
			data["ws_url"] = wsURL
	}

	v1.HandleSuccess(ctx, data)
}

// NodeConsoleWS godoc
// @Summary 节点 Console WebSocket（VNC WebSocket 代理）
// @Description 同域 WS 代理到 Proxmox vncwebsocket，供 noVNC 直接连接（仅 vncshell 模式）
// @Tags PVE节点模块
// @Security Bearer
// @Param token query string true "ws_token（由 /api/v1/nodes/console 返回）"
// @Router /api/v1/nodes/console/ws [get]
func (h *PveNodeHandler) NodeConsoleWS(ctx *gin.Context) {
	token := ctx.Query("token")
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	clientConn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		h.logger.WithContext(ctx).Error("NodeConsoleWS: failed to upgrade websocket", zap.Error(err))
		return
	}
	defer clientConn.Close()

	proxmoxConn, err := h.nodeService.DialNodeConsoleWebsocket(ctx, token)
	if err != nil {
		h.logger.WithContext(ctx).Error("NodeConsoleWS: failed to dial proxmox", zap.Error(err))
		_ = clientConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "invalid console token"))
		return
	}
	defer proxmoxConn.Close()

	h.logger.WithContext(ctx).Info("NodeConsoleWS: proxy established")

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
