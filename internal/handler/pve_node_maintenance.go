package handler

import (
	"go.uber.org/zap"
	"fmt"
	"net/http"
	"pvesphere/api/v1"
	"pvesphere/internal/service"

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

// GetNodeVMs 获取节点上的VM列表
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

// StartMaintenance 开始主机维护
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

// EndMaintenance 结束主机维护
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
