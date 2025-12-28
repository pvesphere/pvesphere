package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	v1 "pvesphere/api/v1"
	"pvesphere/internal/model"
	"pvesphere/internal/repository"
	"pvesphere/pkg/log"
	"pvesphere/pkg/proxmox"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type PveNodeService interface {
	CreateNode(ctx context.Context, req *v1.CreateNodeRequest) error
	UpdateNode(ctx context.Context, id int64, req *v1.UpdateNodeRequest) error
	DeleteNode(ctx context.Context, id int64) error
	GetNode(ctx context.Context, id int64) (*v1.NodeDetail, error)
	ListNodes(ctx context.Context, req *v1.ListNodeRequest) (*v1.ListNodeResponseData, error)
	GetNodeStatus(ctx context.Context, nodeID int64) (map[string]interface{}, error)
	SetNodeStatus(ctx context.Context, nodeID int64, command string) (string, error)
	GetNodeServices(ctx context.Context, nodeID int64) ([]map[string]interface{}, error)
	StartNodeService(ctx context.Context, nodeID int64, serviceName string) (string, error)
	StopNodeService(ctx context.Context, nodeID int64, serviceName string) (string, error)
	RestartNodeService(ctx context.Context, nodeID int64, serviceName string) (string, error)
	GetNodeNetworks(ctx context.Context, nodeID int64) ([]map[string]interface{}, error)
	CreateNodeNetwork(ctx context.Context, req *v1.CreateNodeNetworkRequest) error
	ReloadNodeNetwork(ctx context.Context, nodeID int64) error
	RevertNodeNetwork(ctx context.Context, nodeID int64) error
	GetNodeRRDData(ctx context.Context, nodeID int64, timeframe, cf string) ([]map[string]interface{}, error)
	GetNodeDisksList(ctx context.Context, nodeID int64, includePartitions bool) ([]map[string]interface{}, error)
	GetNodeDisksDirectory(ctx context.Context, nodeID int64) ([]map[string]interface{}, error)
	GetNodeDisksLVM(ctx context.Context, nodeID int64) ([]map[string]interface{}, error)
	GetNodeDisksLVMThin(ctx context.Context, nodeID int64) ([]map[string]interface{}, error)
	GetNodeDisksZFS(ctx context.Context, nodeID int64) ([]map[string]interface{}, error)
	InitGPTDisk(ctx context.Context, nodeID int64, disk string) (string, error)
	WipeDisk(ctx context.Context, nodeID int64, disk string, partition *int) (string, error)
	GetNodeStorageStatus(ctx context.Context, nodeID int64, storage string) (map[string]interface{}, error)
	GetNodeStorageRRDData(ctx context.Context, nodeID int64, storage, timeframe, cf string) ([]map[string]interface{}, error)
	GetNodeStorageContent(ctx context.Context, nodeID int64, storage, content string) ([]map[string]interface{}, error)
	GetNodeStorageVolume(ctx context.Context, nodeID int64, storage, volume string) (map[string]interface{}, error)
	UploadNodeStorageContent(ctx context.Context, nodeID int64, storage, content, filename string, file multipart.File) (interface{}, error)
	DeleteNodeStorageContent(ctx context.Context, nodeID int64, storage, volume string, delay *int) error
	GetNodeConsole(ctx context.Context, req *v1.GetNodeConsoleRequest) (map[string]interface{}, error)
	DialNodeConsoleWebsocket(ctx context.Context, token string) (*websocket.Conn, error)
}

func NewPveNodeService(
	service *Service,
	nodeRepo repository.PveNodeRepository,
	clusterRepo repository.PveClusterRepository,
	logger *log.Logger,
) PveNodeService {
	return &pveNodeService{
		nodeRepo:    nodeRepo,
		clusterRepo: clusterRepo,
		Service:     service,
		logger:      logger,
	}
}

type pveNodeService struct {
	nodeRepo    repository.PveNodeRepository
	clusterRepo repository.PveClusterRepository
	*Service
	logger *log.Logger

	consoleSessions sync.Map // token -> nodeConsoleSession
}

type nodeConsoleSession struct {
	NodeID    int64
	NodeName  string
	Port      int
	Ticket    string // VNC ticket（用于 vncwebsocket 连接）
	ExpiresAt time.Time
	// 高权限认证信息（可选）：如果原始请求使用了 ticket + csrf_token，保存这些信息用于 WebSocket 连接
	AuthTicket    string // Proxmox 高权限认证 ticket
	AuthCSRFToken string // CSRF 防护令牌
	ClusterApiURL string // 集群 API URL（用于创建 ProxmoxClient）
}

func newNodeConsoleToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

func (s *pveNodeService) CreateNode(ctx context.Context, req *v1.CreateNodeRequest) error {
	// 检查节点名称是否已存在
	existing, err := s.nodeRepo.GetByNodeName(ctx, req.NodeName, req.ClusterID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to check node name", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if existing != nil {
		return v1.ErrBadRequest
	}

	node := &model.PveNode{
		NodeName:      req.NodeName,
		IPAddress:     req.IPAddress,
		ClusterID:     req.ClusterID,
		IsSchedulable: req.IsSchedulable,
		Env:           req.Env,
		Status:        req.Status,
		Annotations:   req.Annotations,
		VMLimit:       req.VMLimit,
		CreateTime:    time.Now(),
		UpdateTime:    time.Now(),
	}

	if err := s.nodeRepo.Create(ctx, node); err != nil {
		s.logger.WithContext(ctx).Error("failed to create node", zap.Error(err))
		return v1.ErrInternalServerError
	}

	return nil
}

func (s *pveNodeService) UpdateNode(ctx context.Context, id int64, req *v1.UpdateNodeRequest) error {
	node, err := s.nodeRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if node == nil {
		return v1.ErrNotFound
	}

	// 更新字段
	if req.IPAddress != nil {
		node.IPAddress = *req.IPAddress
	}
	if req.IsSchedulable != nil {
		node.IsSchedulable = *req.IsSchedulable
	}
	if req.Env != nil {
		node.Env = *req.Env
	}
	if req.Status != nil {
		node.Status = *req.Status
	}
	if req.Annotations != nil {
		node.Annotations = *req.Annotations
	}
	if req.VMLimit != nil {
		node.VMLimit = *req.VMLimit
	}
	node.UpdateTime = time.Now()

	if err := s.nodeRepo.Update(ctx, node); err != nil {
		s.logger.WithContext(ctx).Error("failed to update node", zap.Error(err))
		return v1.ErrInternalServerError
	}

	return nil
}

func (s *pveNodeService) DeleteNode(ctx context.Context, id int64) error {
	node, err := s.nodeRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if node == nil {
		return v1.ErrNotFound
	}

	if err := s.nodeRepo.Delete(ctx, id); err != nil {
		s.logger.WithContext(ctx).Error("failed to delete node", zap.Error(err))
		return v1.ErrInternalServerError
	}

	return nil
}

func (s *pveNodeService) GetNode(ctx context.Context, id int64) (*v1.NodeDetail, error) {
	node, err := s.nodeRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	if node == nil {
		return nil, v1.ErrNotFound
	}

	// 查询集群信息填充 cluster_name
	var clusterName string
	if node.ClusterID > 0 {
		cluster, err := s.clusterRepo.GetByID(ctx, node.ClusterID)
		if err != nil {
			s.logger.WithContext(ctx).Warn("failed to get cluster",
				zap.Error(err), zap.Int64("cluster_id", node.ClusterID))
			// 不阻塞主流程，cluster_name 为空
		} else if cluster != nil {
			clusterName = cluster.ClusterName
		}
	}

	return &v1.NodeDetail{
		Id:            node.Id,
		NodeName:      node.NodeName,
		IPAddress:     node.IPAddress,
		ClusterID:     node.ClusterID,
		ClusterName:   clusterName,
		IsSchedulable: node.IsSchedulable,
		Env:           node.Env,
		Status:        node.Status,
		Annotations:   node.Annotations,
		VMLimit:       node.VMLimit,
		CreateTime:    node.CreateTime,
		UpdateTime:    node.UpdateTime,
		Creator:       node.Creator,
		Modifier:      node.Modifier,
	}, nil
}

func (s *pveNodeService) ListNodes(ctx context.Context, req *v1.ListNodeRequest) (*v1.ListNodeResponseData, error) {
	nodes, total, err := s.nodeRepo.ListWithPagination(ctx, req.Page, req.PageSize, req.ClusterID, req.Env, req.Status)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to list nodes", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}

	// 1. 提取所有唯一的 cluster_id
	clusterIDs := make([]int64, 0)
	clusterIDSet := make(map[int64]struct{})
	for _, node := range nodes {
		if node.ClusterID > 0 {
			if _, exists := clusterIDSet[node.ClusterID]; !exists {
				clusterIDs = append(clusterIDs, node.ClusterID)
				clusterIDSet[node.ClusterID] = struct{}{}
			}
		}
	}

	// 2. 批量查询集群信息（通常只有 1-5 个集群）
	clusterMap := make(map[int64]*model.PveCluster)
	if len(clusterIDs) > 0 {
		clusters, err := s.clusterRepo.GetByIDs(ctx, clusterIDs)
		if err != nil {
			s.logger.WithContext(ctx).Warn("failed to get clusters", zap.Error(err))
			// 不阻塞主流程，cluster_name 为空
		} else {
			clusterMap = clusters
		}
	}

	// 3. 构建响应，填充 cluster_name
	items := make([]v1.NodeItem, 0, len(nodes))
	for _, node := range nodes {
		item := v1.NodeItem{
			Id:            node.Id,
			NodeName:      node.NodeName,
			IPAddress:     node.IPAddress,
			ClusterID:     node.ClusterID,
			ClusterName:   "", // 默认为空
			IsSchedulable: node.IsSchedulable,
			Env:           node.Env,
			Status:        node.Status,
			VMLimit:       node.VMLimit,
		}

		// 填充 cluster_name
		if cluster, ok := clusterMap[node.ClusterID]; ok {
			item.ClusterName = cluster.ClusterName
		}

		items = append(items, item)
	}

	return &v1.ListNodeResponseData{
		Total: total,
		List:  items,
	}, nil
}

// getProxmoxClientForNode 根据节点ID获取ProxmoxClient
func (s *pveNodeService) getProxmoxClientForNode(ctx context.Context, nodeID int64) (*proxmox.ProxmoxClient, *model.PveNode, error) {
	// 1. 获取节点信息
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node", zap.Error(err))
		return nil, nil, v1.ErrInternalServerError
	}
	if node == nil {
		return nil, nil, v1.ErrNotFound
	}

	// 2. 获取集群信息
	if node.ClusterID <= 0 {
		return nil, nil, fmt.Errorf("节点的集群 ID 无效")
	}
	cluster, err := s.clusterRepo.GetByID(ctx, node.ClusterID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get cluster", zap.Error(err))
		return nil, nil, v1.ErrInternalServerError
	}
	if cluster == nil {
		return nil, nil, fmt.Errorf("集群 ID %d 不存在", node.ClusterID)
	}

	// 3. 创建 Proxmox 客户端
	proxmoxClient, err := proxmox.NewProxmoxClient(cluster.ApiUrl, cluster.UserId, cluster.UserToken)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to create proxmox client", zap.Error(err))
		return nil, nil, v1.ErrInternalServerError
	}

	return proxmoxClient, node, nil
}

func (s *pveNodeService) GetNodeStatus(ctx context.Context, nodeID int64) (map[string]interface{}, error) {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	status, err := client.GetNodeStatus(ctx, node.NodeName)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node status", zap.Error(err),
			zap.String("node", node.NodeName), zap.Int64("node_id", nodeID))
		return nil, v1.ErrInternalServerError
	}

	return status, nil
}

func (s *pveNodeService) SetNodeStatus(ctx context.Context, nodeID int64, command string) (string, error) {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return "", err
	}

	// 验证 command 参数
	command = strings.ToLower(strings.TrimSpace(command))
	if command != "reboot" && command != "shutdown" {
		return "", fmt.Errorf("invalid command: %s (must be 'reboot' or 'shutdown')", command)
	}

	upid, err := client.SetNodeStatus(ctx, node.NodeName, command)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to set node status", zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Int64("node_id", nodeID),
			zap.String("command", command))
		return "", v1.ErrInternalServerError
	}

	s.logger.WithContext(ctx).Info("node status set", zap.String("node", node.NodeName),
		zap.Int64("node_id", nodeID),
		zap.String("command", command),
		zap.String("upid", upid))

	return upid, nil
}

func (s *pveNodeService) GetNodeServices(ctx context.Context, nodeID int64) ([]map[string]interface{}, error) {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	services, err := client.GetNodeServices(ctx, node.NodeName)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node services", zap.Error(err),
			zap.String("node", node.NodeName), zap.Int64("node_id", nodeID))
		return nil, v1.ErrInternalServerError
	}

	return services, nil
}

// StartNodeService 启动节点服务
func (s *pveNodeService) StartNodeService(ctx context.Context, nodeID int64, serviceName string) (string, error) {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return "", err
	}

	upid, err := client.StartNodeService(ctx, node.NodeName, serviceName)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to start node service",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.String("service", serviceName),
			zap.Int64("node_id", nodeID))
		return "", fmt.Errorf("启动节点服务失败: %w", err)
	}

	s.logger.WithContext(ctx).Info("node service started successfully",
		zap.String("node", node.NodeName),
		zap.String("service", serviceName),
		zap.String("upid", upid))

	return upid, nil
}

// StopNodeService 停止节点服务
func (s *pveNodeService) StopNodeService(ctx context.Context, nodeID int64, serviceName string) (string, error) {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return "", err
	}

	upid, err := client.StopNodeService(ctx, node.NodeName, serviceName)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to stop node service",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.String("service", serviceName),
			zap.Int64("node_id", nodeID))
		return "", fmt.Errorf("停止节点服务失败: %w", err)
	}

	s.logger.WithContext(ctx).Info("node service stopped successfully",
		zap.String("node", node.NodeName),
		zap.String("service", serviceName),
		zap.String("upid", upid))

	return upid, nil
}

// RestartNodeService 重启节点服务
func (s *pveNodeService) RestartNodeService(ctx context.Context, nodeID int64, serviceName string) (string, error) {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return "", err
	}

	upid, err := client.RestartNodeService(ctx, node.NodeName, serviceName)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to restart node service",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.String("service", serviceName),
			zap.Int64("node_id", nodeID))
		return "", fmt.Errorf("重启节点服务失败: %w", err)
	}

	s.logger.WithContext(ctx).Info("node service restarted successfully",
		zap.String("node", node.NodeName),
		zap.String("service", serviceName),
		zap.String("upid", upid))

	return upid, nil
}

// GetNodeNetworks 获取节点网络列表
func (s *pveNodeService) GetNodeNetworks(ctx context.Context, nodeID int64) ([]map[string]interface{}, error) {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	networks, err := client.GetNodeNetworks(ctx, node.NodeName)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node networks",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Int64("node_id", nodeID))
		return nil, v1.ErrInternalServerError
	}

	return networks, nil
}

// CreateNodeNetwork 创建网络设备配置
func (s *pveNodeService) CreateNodeNetwork(ctx context.Context, req *v1.CreateNodeNetworkRequest) error {
	client, node, err := s.getProxmoxClientForNode(ctx, req.NodeID)
	if err != nil {
		return err
	}

	// 构建 form 参数
	params := url.Values{}
	if req.Iface != nil && *req.Iface != "" {
		params.Set("iface", *req.Iface)
	}
	if req.Type != nil && *req.Type != "" {
		params.Set("type", *req.Type)
	}
	if req.Autostart != nil {
		params.Set("autostart", fmt.Sprintf("%d", *req.Autostart))
	}
	if req.Comments != nil && *req.Comments != "" {
		params.Set("comments", *req.Comments)
	}
	if req.BridgePorts != nil && *req.BridgePorts != "" {
		params.Set("bridge_ports", *req.BridgePorts)
	}
	if req.BridgeVlanAware != nil {
		params.Set("bridge_vlan_aware", fmt.Sprintf("%d", *req.BridgeVlanAware))
	}
	if req.Gateway != nil && *req.Gateway != "" {
		params.Set("gateway", *req.Gateway)
	}
	if req.Address != nil && *req.Address != "" {
		params.Set("address", *req.Address)
	}
	if req.Netmask != nil && *req.Netmask != "" {
		params.Set("netmask", *req.Netmask)
	}
	if req.BondMode != nil {
		params.Set("bond_mode", fmt.Sprintf("%d", *req.BondMode))
	}
	if req.BondSlaves != nil && *req.BondSlaves != "" {
		params.Set("bond_slaves", *req.BondSlaves)
	}
	if req.MTU != nil {
		params.Set("mtu", fmt.Sprintf("%d", *req.MTU))
	}

	// 调用 Proxmox API 创建网络配置
	err = client.CreateNodeNetwork(ctx, node.NodeName, params)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to create node network",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Int64("node_id", req.NodeID))
		return fmt.Errorf("创建网络配置失败: %w", err)
	}

	s.logger.WithContext(ctx).Info("node network created successfully",
		zap.String("node", node.NodeName),
		zap.Int64("node_id", req.NodeID))

	return nil
}

// ReloadNodeNetwork 重新加载网络配置
func (s *pveNodeService) ReloadNodeNetwork(ctx context.Context, nodeID int64) error {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return err
	}

	err = client.ReloadNodeNetwork(ctx, node.NodeName)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to reload node network",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Int64("node_id", nodeID))
		return fmt.Errorf("重新加载网络配置失败: %w", err)
	}

	s.logger.WithContext(ctx).Info("node network reloaded successfully",
		zap.String("node", node.NodeName),
		zap.Int64("node_id", nodeID))

	return nil
}

// RevertNodeNetwork 恢复网络配置更改
func (s *pveNodeService) RevertNodeNetwork(ctx context.Context, nodeID int64) error {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return err
	}

	err = client.RevertNodeNetwork(ctx, node.NodeName)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to revert node network",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Int64("node_id", nodeID))
		return fmt.Errorf("恢复网络配置失败: %w", err)
	}

	s.logger.WithContext(ctx).Info("node network reverted successfully",
		zap.String("node", node.NodeName),
		zap.Int64("node_id", nodeID))

	return nil
}

func (s *pveNodeService) GetNodeRRDData(ctx context.Context, nodeID int64, timeframe, cf string) ([]map[string]interface{}, error) {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	rrdData, err := client.GetNodeRRDData(ctx, node.NodeName, timeframe, cf)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node rrd data", zap.Error(err),
			zap.String("node", node.NodeName), zap.Int64("node_id", nodeID),
			zap.String("timeframe", timeframe), zap.String("cf", cf))
		return nil, v1.ErrInternalServerError
	}

	return rrdData, nil
}

func (s *pveNodeService) GetNodeDisksList(ctx context.Context, nodeID int64, includePartitions bool) ([]map[string]interface{}, error) {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	disks, err := client.GetNodeDisksList(ctx, node.NodeName, includePartitions)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node disks list",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Int64("node_id", nodeID),
			zap.Bool("include_partitions", includePartitions))
		return nil, v1.ErrInternalServerError
	}

	return disks, nil
}

func (s *pveNodeService) GetNodeDisksDirectory(ctx context.Context, nodeID int64) ([]map[string]interface{}, error) {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	directories, err := client.GetNodeDisksDirectory(ctx, node.NodeName)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node disks directory",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Int64("node_id", nodeID))
		return nil, v1.ErrInternalServerError
	}

	return directories, nil
}

func (s *pveNodeService) GetNodeDisksLVM(ctx context.Context, nodeID int64) ([]map[string]interface{}, error) {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	lvms, err := client.GetNodeDisksLVM(ctx, node.NodeName)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node disks lvm",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Int64("node_id", nodeID))
		return nil, v1.ErrInternalServerError
	}

	return lvms, nil
}

func (s *pveNodeService) GetNodeDisksLVMThin(ctx context.Context, nodeID int64) ([]map[string]interface{}, error) {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	lvmthins, err := client.GetNodeDisksLVMThin(ctx, node.NodeName)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node disks lvmthin",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Int64("node_id", nodeID))
		return nil, v1.ErrInternalServerError
	}

	return lvmthins, nil
}

func (s *pveNodeService) GetNodeDisksZFS(ctx context.Context, nodeID int64) ([]map[string]interface{}, error) {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	zfss, err := client.GetNodeDisksZFS(ctx, node.NodeName)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node disks zfs",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Int64("node_id", nodeID))
		return nil, v1.ErrInternalServerError
	}

	return zfss, nil
}

func (s *pveNodeService) InitGPTDisk(ctx context.Context, nodeID int64, disk string) (string, error) {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return "", err
	}

	upid, err := client.InitGPTDisk(ctx, node.NodeName, disk)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to init gpt disk",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Int64("node_id", nodeID),
			zap.String("disk", disk))
		return "", v1.ErrInternalServerError
	}

	return upid, nil
}

func (s *pveNodeService) WipeDisk(ctx context.Context, nodeID int64, disk string, partition *int) (string, error) {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return "", err
	}

	upid, err := client.WipeDisk(ctx, node.NodeName, disk, partition)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to wipe disk",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Int64("node_id", nodeID),
			zap.String("disk", disk),
			zap.Any("partition", partition))
		return "", v1.ErrInternalServerError
	}

	return upid, nil
}

// GetNodeStorageStatus 获取节点存储状态
func (s *pveNodeService) GetNodeStorageStatus(ctx context.Context, nodeID int64, storage string) (map[string]interface{}, error) {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	status, err := client.GetStorageStatus(ctx, node.NodeName, storage)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get storage status",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Int64("node_id", nodeID),
			zap.String("storage", storage))
		return nil, v1.ErrInternalServerError
	}
	return status, nil
}

// GetNodeStorageRRDData 获取节点存储 RRD 监控数据
func (s *pveNodeService) GetNodeStorageRRDData(ctx context.Context, nodeID int64, storage, timeframe, cf string) ([]map[string]interface{}, error) {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	data, err := client.GetStorageRRDData(ctx, node.NodeName, storage, timeframe, cf)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get storage rrd data",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Int64("node_id", nodeID),
			zap.String("storage", storage),
			zap.String("timeframe", timeframe),
			zap.String("cf", cf))
		return nil, v1.ErrInternalServerError
	}
	return data, nil
}

// GetNodeStorageContent 获取节点存储内容列表
func (s *pveNodeService) GetNodeStorageContent(ctx context.Context, nodeID int64, storage, content string) ([]map[string]interface{}, error) {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	items, err := client.GetStorageContent(ctx, node.NodeName, storage, content)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get storage content",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Int64("node_id", nodeID),
			zap.String("storage", storage),
			zap.String("content", content))
		return nil, v1.ErrInternalServerError
	}
	return items, nil
}

// GetNodeStorageVolume 获取节点存储卷属性
func (s *pveNodeService) GetNodeStorageVolume(ctx context.Context, nodeID int64, storage, volume string) (map[string]interface{}, error) {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	info, err := client.GetStorageVolume(ctx, node.NodeName, storage, volume)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get storage volume",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Int64("node_id", nodeID),
			zap.String("storage", storage),
			zap.String("volume", volume))
		return nil, v1.ErrInternalServerError
	}
	return info, nil
}

// UploadNodeStorageContent 上传存储内容（模板 / ISO / OVA / VM 镜像）
func (s *pveNodeService) UploadNodeStorageContent(
	ctx context.Context,
	nodeID int64,
	storage, content, filename string,
	file multipart.File,
) (interface{}, error) {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	result, err := client.UploadStorageContent(ctx, node.NodeName, storage, content, filename, file)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to upload storage content",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Int64("node_id", nodeID),
			zap.String("storage", storage),
			zap.String("filename", filename),
			zap.String("content", content))
		return nil, v1.ErrInternalServerError
	}
	return result, nil
}

// DeleteNodeStorageContent 删除存储内容（镜像 / ISO / OVA / VM 镜像等）
func (s *pveNodeService) DeleteNodeStorageContent(
	ctx context.Context,
	nodeID int64,
	storage, volume string,
	delay *int,
) error {
	client, node, err := s.getProxmoxClientForNode(ctx, nodeID)
	if err != nil {
		return err
	}

	if err := client.DeleteStorageContent(ctx, node.NodeName, storage, volume, delay); err != nil {
		s.logger.WithContext(ctx).Error("failed to delete storage content",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Int64("node_id", nodeID),
			zap.String("storage", storage),
			zap.String("volume", volume),
			zap.Any("delay", delay))
		return v1.ErrInternalServerError
	}
	return nil
}

// GetNodeConsole 获取节点控制台信息
func (s *pveNodeService) GetNodeConsole(ctx context.Context, req *v1.GetNodeConsoleRequest) (map[string]interface{}, error) {
	// 验证控制台类型
	req.ConsoleType = strings.ToLower(strings.TrimSpace(req.ConsoleType))
	if req.ConsoleType != "termproxy" && req.ConsoleType != "vncshell" {
		return nil, fmt.Errorf("invalid console_type: %s (must be 'termproxy' or 'vncshell')", req.ConsoleType)
	}

	// 获取节点信息
	node, err := s.nodeRepo.GetByID(ctx, req.NodeID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	if node == nil {
		return nil, v1.ErrNotFound
	}

	// 获取集群信息（用于获取 api_url）
	if node.ClusterID <= 0 {
		return nil, fmt.Errorf("节点的集群 ID 无效")
	}
	cluster, err := s.clusterRepo.GetByID(ctx, node.ClusterID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get cluster", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	if cluster == nil {
		return nil, fmt.Errorf("集群 ID %d 不存在", node.ClusterID)
	}

	// 如果提供了 ticket 和 csrf_token，使用高权限认证方式；否则使用集群配置的 API Token
	var client *proxmox.ProxmoxClient
	if strings.TrimSpace(req.Ticket) != "" && strings.TrimSpace(req.CSRFToken) != "" {
		// 使用高权限 ticket 和 CSRF token 创建客户端
		client, err = proxmox.NewProxmoxClientWithTicket(cluster.ApiUrl, req.Ticket, req.CSRFToken)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to create proxmox client with ticket", zap.Error(err))
			return nil, v1.ErrInternalServerError
		}
		s.logger.WithContext(ctx).Info("using high-privilege ticket authentication",
			zap.Int64("node_id", req.NodeID),
			zap.String("node_name", node.NodeName))
	} else {
		// 使用集群配置的 API Token
		client, err = proxmox.NewProxmoxClient(cluster.ApiUrl, cluster.UserId, cluster.UserToken)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to create proxmox client", zap.Error(err))
			return nil, v1.ErrInternalServerError
		}
	}

	// 验证节点名称不为空
	if strings.TrimSpace(node.NodeName) == "" {
		s.logger.WithContext(ctx).Error("node name is empty",
			zap.Int64("node_id", req.NodeID))
		return nil, fmt.Errorf("node name is empty for node_id=%d", req.NodeID)
	}

	// 验证节点是否存在于 Proxmox 集群中
	_, err = client.GetNodeStatus(ctx, node.NodeName)
	if err != nil {
		s.logger.WithContext(ctx).Error("node not found in proxmox cluster",
			zap.Error(err),
			zap.Int64("node_id", req.NodeID),
			zap.String("node_name", node.NodeName))
		return nil, fmt.Errorf("node '%s' not found in proxmox cluster or access denied: %w", node.NodeName, err)
	}

	s.logger.WithContext(ctx).Info("getting node console",
		zap.Int64("node_id", req.NodeID),
		zap.String("node_name", node.NodeName),
		zap.String("console_type", req.ConsoleType))

	var result map[string]interface{}

	if req.ConsoleType == "termproxy" {
		// 终端代理模式
		// 注意：termproxy 返回的数据结构与 vncshell 相同（包含 port、ticket、user、upid 等）
		result, err = client.NodeTermProxy(ctx, node.NodeName)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to get node termproxy",
				zap.Error(err),
				zap.String("node", node.NodeName),
				zap.Int64("node_id", req.NodeID),
				zap.String("error_detail", err.Error()))
			// 返回原始错误，让调用方能看到具体错误信息
			return nil, fmt.Errorf("failed to get node termproxy: %w", err)
		}
	} else {
		// VNC Shell 模式
		// 默认启用 websocket，确保返回 port/ticket
		websocket := req.Websocket
		if !websocket {
			websocket = true
		}
		s.logger.WithContext(ctx).Debug("calling NodeVncShell",
			zap.String("node_name", node.NodeName),
			zap.Bool("websocket", websocket),
			zap.Bool("generate_password", req.GeneratePassword))
		result, err = client.NodeVncShell(ctx, node.NodeName, websocket, req.GeneratePassword)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to get node vncshell",
				zap.Error(err),
				zap.String("node", node.NodeName),
				zap.Int64("node_id", req.NodeID),
				zap.Bool("websocket", websocket),
				zap.Bool("generate_password", req.GeneratePassword),
				zap.String("error_detail", err.Error()))
			// 返回原始错误，让调用方能看到具体错误信息
			return nil, fmt.Errorf("failed to get node vncshell: %w", err)
		}
	}

	// 无论是 termproxy 还是 vncshell，只要返回了 port/ticket，就生成短期 ws_token，
	// 统一通过 /api/v1/nodes/console/ws 代理到 Proxmox vncwebsocket。
	var port int
	switch v := result["port"].(type) {
	case float64:
		port = int(v)
	case int:
		port = v
	case int64:
		port = int(v)
	case json.Number:
		if p, err := v.Int64(); err == nil {
			port = int(p)
		}
	case string:
		if p, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			port = p
		}
	}
	ticket, _ := result["ticket"].(string)
	if port <= 0 || strings.TrimSpace(ticket) == "" {
		s.logger.WithContext(ctx).Error("node console response missing port/ticket",
			zap.String("node", node.NodeName),
			zap.Int64("node_id", req.NodeID),
			zap.String("console_type", req.ConsoleType),
			zap.Int("port", port),
			zap.String("ticket", ticket),
			zap.Any("response_data", result))
		return nil, fmt.Errorf("node console response missing required fields: port=%d, ticket=%s", port, ticket)
	}

	// 生成短期 token 用于 WebSocket 连接
	token, err := newNodeConsoleToken()
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to generate console token", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	exp := time.Now().Add(2 * time.Minute)
	session := nodeConsoleSession{
		NodeID:        req.NodeID,
		NodeName:      node.NodeName,
		Port:          port,
		Ticket:        ticket, // VNC ticket（用于 vncwebsocket）
		ExpiresAt:     exp,
		ClusterApiURL: cluster.ApiUrl,
	}
	// 如果原始请求使用了高权限认证，保存认证信息用于后续 WebSocket 连接
	if strings.TrimSpace(req.Ticket) != "" && strings.TrimSpace(req.CSRFToken) != "" {
		session.AuthTicket = req.Ticket
		session.AuthCSRFToken = req.CSRFToken
	}
	s.consoleSessions.Store(token, session)
	result["ws_token"] = token
	result["ws_expires_at"] = exp.Unix()

	return result, nil
}

// DialNodeConsoleWebsocket 通过 ws_token 建立到 Proxmox vncwebsocket 的连接（单次使用/短期有效）
func (s *pveNodeService) DialNodeConsoleWebsocket(ctx context.Context, token string) (*websocket.Conn, error) {
	if strings.TrimSpace(token) == "" {
		return nil, v1.ErrBadRequest
	}

	val, ok := s.consoleSessions.LoadAndDelete(token)
	if !ok {
		return nil, v1.ErrNotFound
	}
	session, ok := val.(nodeConsoleSession)
	if !ok {
		return nil, v1.ErrInternalServerError
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, v1.ErrUnauthorized
	}

	// 如果 session 中保存了高权限认证信息，使用这些信息创建客户端；否则使用集群配置的 API Token
	var client *proxmox.ProxmoxClient
	var err error
	if session.AuthTicket != "" && session.AuthCSRFToken != "" {
		// 使用高权限 ticket 和 CSRF token
		client, err = proxmox.NewProxmoxClientWithTicket(session.ClusterApiURL, session.AuthTicket, session.AuthCSRFToken)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to create proxmox client with ticket for websocket", zap.Error(err))
			return nil, v1.ErrInternalServerError
		}
	} else {
		// 使用集群配置的 API Token
		client, _, err = s.getProxmoxClientForNode(ctx, session.NodeID)
		if err != nil {
			return nil, err
		}
	}

	params := url.Values{}
	params.Set("port", fmt.Sprintf("%d", session.Port))
	params.Set("vncticket", session.Ticket)

	path := fmt.Sprintf("/nodes/%s/vncwebsocket", session.NodeName)
	conn, resp, err := client.WebSocket(path, params.Encode())
	if err != nil {
		var statusCode int
		if resp != nil {
			statusCode = resp.StatusCode
		}
		s.logger.WithContext(ctx).Error("failed to dial proxmox vncwebsocket", zap.Error(err),
			zap.String("node", session.NodeName),
			zap.Int64("node_id", session.NodeID),
			zap.Int("response_status", statusCode))
		return nil, v1.ErrInternalServerError
	}
	return conn, nil
}
