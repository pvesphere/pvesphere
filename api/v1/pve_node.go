package v1

import "time"

// PveNode 相关 API 定义

// CreateNodeRequest 创建节点请求
type CreateNodeRequest struct {
	NodeName      string `json:"node_name" binding:"required" example:"pve-node-1"`
	IPAddress     string `json:"ip_address" example:"10.7.64.206"`
	ClusterID     int64  `json:"cluster_id" binding:"required" example:"1"`
	IsSchedulable int8   `json:"is_schedulable" example:"1"`
	Env           string `json:"env" example:"prod"`
	Status        string `json:"status" example:"online"`
	Annotations   string `json:"annotations" example:"节点备注"`
	VMLimit       int64  `json:"vm_limit" example:"100"`
}

// UpdateNodeRequest 更新节点请求
type UpdateNodeRequest struct {
	IPAddress     *string `json:"ip_address,omitempty"`
	IsSchedulable *int8   `json:"is_schedulable,omitempty"`
	Env           *string `json:"env,omitempty"`
	Status        *string `json:"status,omitempty"`
	Annotations   *string `json:"annotations,omitempty"`
	VMLimit       *int64  `json:"vm_limit,omitempty"`
}

// ListNodeRequest 列表查询请求
type ListNodeRequest struct {
	Page      int    `form:"page" example:"1"`
	PageSize  int    `form:"page_size" binding:"omitempty,max=100" example:"10"`
	ClusterID int64  `form:"cluster_id" example:"1"`
	Env       string `form:"env" example:"prod"`
	Status    string `form:"status" example:"online"`
}

// ListNodeResponse 列表查询响应
type ListNodeResponse struct {
	Response
	Data ListNodeResponseData
}

type ListNodeResponseData struct {
	Total int64      `json:"total"`
	List  []NodeItem `json:"list"`
}

type NodeItem struct {
	Id            int64  `json:"id"`
	NodeName      string `json:"node_name"`
	IPAddress     string `json:"ip_address"`
	ClusterID     int64  `json:"cluster_id"`
	ClusterName   string `json:"cluster_name"` // 从关联表查询填充
	IsSchedulable int8   `json:"is_schedulable"`
	Env           string `json:"env"`
	Status        string `json:"status"`
	VMLimit       int64  `json:"vm_limit"`
}

// GetNodeResponse 详情查询响应
type GetNodeResponse struct {
	Response
	Data NodeDetail
}

type NodeDetail struct {
	Id            int64     `json:"id"`
	NodeName      string    `json:"node_name"`
	IPAddress     string    `json:"ip_address"`
	ClusterID     int64     `json:"cluster_id"`
	ClusterName   string    `json:"cluster_name"` // 从关联表查询填充
	IsSchedulable int8      `json:"is_schedulable"`
	Env           string    `json:"env"`
	Status        string    `json:"status"`
	Annotations   string    `json:"annotations"`
	VMLimit       int64     `json:"vm_limit"`
	CreateTime    time.Time `json:"create_time"` // 创建时间
	UpdateTime    time.Time `json:"update_time"` // 更新时间
	Creator       string    `json:"creator"`     // 创建者
	Modifier      string    `json:"modifier"`    // 修改者
}

// GetNodeStatusRequest 获取节点状态请求
type GetNodeStatusRequest struct {
	NodeID int64 `form:"node_id" binding:"required" example:"1"` // 节点ID（数据库ID）
}

// GetNodeStatusResponse 获取节点状态响应
type GetNodeStatusResponse struct {
	Response
	Data map[string]interface{} `json:"data"`
}

// GetNodeServicesRequest 获取节点服务列表请求
type GetNodeServicesRequest struct {
	NodeID int64 `form:"node_id" binding:"required" example:"1"` // 节点ID（数据库ID）
}

// GetNodeServicesResponse 获取节点服务列表响应
type GetNodeServicesResponse struct {
	Response
	Data []map[string]interface{} `json:"data"`
}

// StartNodeServiceRequest 启动节点服务请求
type StartNodeServiceRequest struct {
	NodeID      int64  `json:"node_id" binding:"required" example:"1"`        // 节点ID（数据库ID）
	ServiceName string `json:"service_name" binding:"required" example:"pveproxy"` // 服务名称（如：pveproxy, pvedaemon, corosync 等）
}

// StartNodeServiceResponse 启动节点服务响应
type StartNodeServiceResponse struct {
	Response
	Data StartNodeServiceResponseData `json:"data"`
}

type StartNodeServiceResponseData struct {
	UPID string `json:"upid"` // 任务ID
}

// StopNodeServiceRequest 停止节点服务请求
type StopNodeServiceRequest struct {
	NodeID      int64  `json:"node_id" binding:"required" example:"1"`        // 节点ID（数据库ID）
	ServiceName string `json:"service_name" binding:"required" example:"pveproxy"` // 服务名称（如：pveproxy, pvedaemon, corosync 等）
}

// StopNodeServiceResponse 停止节点服务响应
type StopNodeServiceResponse struct {
	Response
	Data StopNodeServiceResponseData `json:"data"`
}

type StopNodeServiceResponseData struct {
	UPID string `json:"upid"` // 任务ID
}

// RestartNodeServiceRequest 重启节点服务请求
type RestartNodeServiceRequest struct {
	NodeID      int64  `json:"node_id" binding:"required" example:"1"`        // 节点ID（数据库ID）
	ServiceName string `json:"service_name" binding:"required" example:"pveproxy"` // 服务名称（如：pveproxy, pvedaemon, corosync 等）
}

// RestartNodeServiceResponse 重启节点服务响应
type RestartNodeServiceResponse struct {
	Response
	Data RestartNodeServiceResponseData `json:"data"`
}

type RestartNodeServiceResponseData struct {
	UPID string `json:"upid"` // 任务ID
}

// GetNodeNetworksRequest 获取节点网络列表请求
type GetNodeNetworksRequest struct {
	NodeID int64 `form:"node_id" binding:"required" example:"1"` // 节点ID（数据库ID）
}

// GetNodeNetworksResponse 获取节点网络列表响应
type GetNodeNetworksResponse struct {
	Response
	Data []map[string]interface{} `json:"data"`
}

// CreateNodeNetworkRequest 创建网络设备配置请求
// 参考: https://pve.proxmox.com/pve-docs/api-viewer/#/nodes/{node}/network
// 所有字段都是可选的，根据需要传递相应的参数
type CreateNodeNetworkRequest struct {
	NodeID         int64   `json:"node_id" binding:"required" example:"1"`        // 节点ID（数据库ID）
	Iface          *string `json:"iface,omitempty" example:"eth0"`                // 网络接口名称（如：eth0, vmbr0 等）
	Type           *string `json:"type,omitempty" example:"bridge"`                // 网络类型：bridge, bond, eth, alias, vlan, ovs_bridge, ovs_bond, ovs_port, ovs_int_port
	Autostart      *int    `json:"autostart,omitempty" example:"1"`                // 是否自动启动：0=否, 1=是
	Comments       *string `json:"comments,omitempty" example:"Main network bridge"` // 注释
	BridgePorts    *string `json:"bridge_ports,omitempty" example:"eth0"`        // 桥接端口（bridge 类型）
	BridgeVlanAware *int   `json:"bridge_vlan_aware,omitempty" example:"0"`      // 桥接 VLAN 感知：0=否, 1=是
	Gateway        *string `json:"gateway,omitempty" example:"192.168.1.1"`       // 网关地址
	Address        *string `json:"address,omitempty" example:"192.168.1.100/24"`  // IP 地址和子网掩码
	Netmask        *string `json:"netmask,omitempty" example:"255.255.255.0"`    // 子网掩码（如果 address 未包含）
	BondMode       *int    `json:"bond_mode,omitempty" example:"0"`                // Bond 模式（bond 类型）：0=balance-rr, 1=active-backup, 2=balance-xor, 3=broadcast, 4=802.3ad, 5=balance-tlb, 6=balance-alb
	BondSlaves     *string `json:"bond_slaves,omitempty" example:"eth0 eth1"`     // Bond 从接口（bond 类型）
	MTU            *int    `json:"mtu,omitempty" example:"1500"`                   // 最大传输单元
	// 更多字段可以根据需要添加
}

// CreateNodeNetworkResponse 创建网络设备配置响应
type CreateNodeNetworkResponse struct {
	Response
}

// ReloadNodeNetworkRequest 重新加载网络配置请求
type ReloadNodeNetworkRequest struct {
	NodeID int64 `json:"node_id" binding:"required" example:"1"` // 节点ID（数据库ID）
}

// ReloadNodeNetworkResponse 重新加载网络配置响应
type ReloadNodeNetworkResponse struct {
	Response
}

// RevertNodeNetworkRequest 恢复网络配置更改请求
type RevertNodeNetworkRequest struct {
	NodeID int64 `json:"node_id" binding:"required" example:"1"` // 节点ID（数据库ID）
}

// RevertNodeNetworkResponse 恢复网络配置更改响应
type RevertNodeNetworkResponse struct {
	Response
}

// SetNodeStatusRequest 设置节点状态请求（重启/关闭）
type SetNodeStatusRequest struct {
	NodeID  int64  `json:"node_id" binding:"required" example:"1"`      // 节点ID（数据库ID）
	Command string `json:"command" binding:"required" example:"reboot"` // 操作命令：reboot（重启）或 shutdown（关闭）
}

// SetNodeStatusResponse 设置节点状态响应
type SetNodeStatusResponse struct {
	Response
	Data map[string]interface{} `json:"data"`
}

// GetNodeConsoleRequest 获取节点控制台请求
type GetNodeConsoleRequest struct {
	NodeID           int64  `json:"node_id" binding:"required" example:"1"`             // 节点ID（数据库ID）
	ConsoleType      string `json:"console_type" binding:"required" example:"vncshell"` // 控制台类型：termproxy（终端）或 vncshell（VNC图形界面）
	Websocket        bool   `json:"websocket,omitempty" example:"true"`                 // 是否启用 websocket（仅 vncshell 有效）
	GeneratePassword bool   `json:"generate_password,omitempty" example:"false"`        // 是否生成密码（仅 vncshell 有效）
	// 高权限认证（可选）：如果提供了 ticket 和 csrf_token，将使用这些凭证而不是集群配置的 API Token
	Ticket    string `json:"ticket,omitempty" example:"PVE:root@pam:..."`              // Proxmox 高权限票据（从 /api/v1/pve/access/ticket 获取）
	CSRFToken string `json:"csrf_token,omitempty" example:"6948C80E:..."`              // CSRF 防护令牌（从 /api/v1/pve/access/ticket 获取）
}

// GetNodeConsoleResponse 获取节点控制台响应
type GetNodeConsoleResponse struct {
	Response
	Data map[string]interface{} `json:"data"`
}

// GetAccessTicketRequest 获取 Proxmox 高权限票据请求
type GetAccessTicketRequest struct {
	ClusterID int64  `json:"cluster_id" binding:"required" example:"1"` // 集群ID（从集群表获取 api_url）
	Username  string `json:"username" binding:"required" example:"root"`
	Realm     string `json:"realm" binding:"required" example:"pam"`
	Password  string `json:"password" binding:"required" example:"your-password"`
}

// GetAccessTicketResponse 获取 Proxmox 高权限票据响应
type GetAccessTicketResponse struct {
	Response
	Data map[string]interface{} `json:"data"`
}
