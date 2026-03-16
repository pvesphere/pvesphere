package v1

// NodeMaintenance 相关 API 定义

// StartMaintenanceRequest 开始主机维护请求
type StartMaintenanceRequest struct {
	NodeID      string   `json:"node_id" binding:"required" example:"1"`       // 节点ID
	Mode        string   `json:"mode" example:"auto"`                           // 迁移模式: manual(手动指定) / auto(自动选择)
	TargetNodes []int64  `json:"target_nodes,omitempty" example:"[1,2,3]"`     // 目标节点ID列表 (manual模式)
	Concurrency int      `json:"concurrency" example:"3"`                      // 并发迁移数，默认3
}

// GetMigrateCandidatesRequest 获取可迁移目标节点请求
type GetMigrateCandidatesRequest struct {
	NodeID string `json:"node_id" binding:"required" example:"1"` // 节点ID
}

// MigrateCandidatesResponse 可迁移目标节点响应
type MigrateCandidatesResponse struct {
	Response
	Data []MigrateCandidate `json:"data"`
}

// MigrateCandidate 可迁移目标节点
type MigrateCandidate struct {
	NodeID     int64   `json:"node_id" example:"1"`
	NodeName   string  `json:"node_name" example:"pve-node-1"`
	CPUCores  int     `json:"cpu_cores" example:"32"`
	CPUUsage  float64 `json:"cpu_usage" example:"45.5"`              // CPU使用率%
	MemTotal  int64   `json:"mem_total" example:"128849018880"`      // 内存总量 (bytes)
	MemUsed   int64   `json:"mem_used" example:"64424509440"`        // 内存使用 (bytes)
	MemFree   int64   `json:"mem_free" example:"64424509440"`        // 内存可用 (bytes)
	MemUsage  float64 `json:"mem_usage" example:"50.0"`              // 内存使用率%
	VMCount   int     `json:"vm_count" example:"10"`                 // 当前VM数量
	Score     float64 `json:"score" example:"85.0"`                 // 调度评分 (越高越好)
}

// NodeMaintenanceResponse 主机维护响应
type NodeMaintenanceResponse struct {
	Response
	Data NodeMaintenanceResult `json:"data"`
}

// NodeMaintenanceResult 维护结果
type NodeMaintenanceResult struct {
	NodeID       string           `json:"node_id" example:"1"`
	Status       string           `json:"status" example:"maintenance"` // maintenance / migrating / completed / failed
	TotalVMs     int              `json:"total_vms" example:"5"`
	MigratedVMs  int              `json:"migrated_vms" example:"3"`
	FailedVMs    int              `json:"failed_vms" example:"0"`
	Tasks        []MaintenanceTask `json:"tasks"`
}

// MaintenanceTask 迁移任务
type MaintenanceTask struct {
	VMID       string `json:"vm_id" example:"101"`
	VMName     string `json:"vm_name" example:"vm-nginx"`
	TargetNode string `json:"target_node" example:"pve-node-2"`
	Status     string `json:"status" example:"completed"` // pending / migrating / completed / failed
	Error      string `json:"error,omitempty"`
}

// EndMaintenanceRequest 结束维护请求
type EndMaintenanceRequest struct {
	NodeID string `json:"node_id" binding:"required" example:"1"` // 节点ID
}

// GetNodeVMsRequest 获取节点VM列表请求
type GetNodeVMsRequest struct {
	NodeID int64 `json:"node_id" binding:"required" example:"1"`
}

// NodeVMsResponse 节点VM列表响应
type NodeVMsResponse struct {
	Response
	Data []NodeVMItem `json:"data"`
}

// NodeVMItem 节点上的虚拟机
type NodeVMItem struct {
	VMID     int64  `json:"vm_id" example:"101"`
	VMName   string `json:"vm_name" example:"vm-nginx"`
	Status   string `json:"status" example:"running"`
	CPU      int    `json:"cpu" example:"2"`
	Mem      int64  `json:"mem" example:"4294967296"`
	NodeID   int64  `json:"node_id" example:"1"`
	NodeName string `json:"node_name" example:"pve-node-1"`
}
