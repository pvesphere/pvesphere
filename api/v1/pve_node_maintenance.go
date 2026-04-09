package v1

// NodeMaintenance 相关 API 定义

// StartMaintenanceRequest 开始主机维护请求
type StartMaintenanceRequest struct {
	NodeID      string   `json:"node_id" binding:"required" example:"1"`                            // 节点 ID
	Mode        string   `json:"mode" example:"auto"`                                                 // 迁移模式：manual(手动指定) / auto(自动选择)
	TargetNodes []int64  `json:"target_nodes,omitempty" collectionFormat:"csv" example:"1,2,3"`       // 目标节点 ID 列表 (manual 模式)
	Concurrency int      `json:"concurrency" example:"3"`                                             // 并发迁移数，默认 3
}

// GetMigrateCandidatesRequest 获取可迁移目标节点请求
type GetMigrateCandidatesRequest struct {
	NodeID string `json:"node_id" binding:"required" example:"1"` // 节点 ID
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
	CPUCores   int     `json:"cpu_cores" example:"32"`
	CPUUsage   float64 `json:"cpu_usage" example:"45.5"`   // CPU 使用率%
	MemTotal   int64   `json:"mem_total" example:"128849018880"`  // 内存总量 (bytes)
	MemUsed    int64   `json:"mem_used" example:"64424509440"`    // 内存使用 (bytes)
	MemFree    int64   `json:"mem_free" example:"64424509440"`    // 内存可用 (bytes)
	MemUsage   float64 `json:"mem_usage" example:"50.0"`   // 内存使用率%
	VMCount    int     `json:"vm_count" example:"10"`      // 当前 VM 数量
	Score      float64 `json:"score" example:"85.0"`       // 调度评分 (越高越好)
}

// NodeMaintenanceResponse 主机维护响应
type NodeMaintenanceResponse struct {
	Response
	Data NodeMaintenanceResult `json:"data"`
}

// NodeMaintenanceResult 维护结果
type NodeMaintenanceResult struct {
	NodeID       string            `json:"node_id" example:"1"`
	Status       string            `json:"status" example:"maintenance"` // maintenance / migrating / completed / failed
	TotalVMs     int               `json:"total_vms" example:"5"`
	MigratedVMs  int               `json:"migrated_vms" example:"3"`
	FailedVMs    int               `json:"failed_vms" example:"0"`
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
	NodeID string `json:"node_id" binding:"required" example:"1"` // 节点 ID
}

// GetNodeVMsRequest 获取节点 VM 列表请求
type GetNodeVMsRequest struct {
	NodeID int64 `json:"node_id" binding:"required" example:"1"`
}

// NodeVMsResponse 节点 VM 列表响应
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

// ============================================================
// 维护计划管理相关 API
// ============================================================

// CreateMaintenancePlanRequest 创建维护计划请求
type CreateMaintenancePlanRequest struct {
	NodeID      string  `json:"node_id" binding:"required" example:"1"`                    // 节点 ID
	Mode        string  `json:"mode" example:"auto"`                                         // 迁移模式：auto/manual
	TargetNodes []int64 `json:"target_nodes,omitempty" example:"1,2,3"`                        // 目标节点 ID 列表 (manual 模式)
	Concurrency int     `json:"concurrency" example:"3"`                                     // 并发迁移数，默认 3
}

// CreateMaintenancePlanResponse 创建维护计划响应
type CreateMaintenancePlanResponse struct {
	Response
	Data MaintenancePlanDetail `json:"data"`
}

// MaintenancePlanDetail 维护计划详情
type MaintenancePlanDetail struct {
	PlanID       int64  `json:"plan_id" example:"1"`
	NodeID       int64  `json:"node_id" example:"1"`
	NodeName     string `json:"node_name" example:"pve-node-1"`
	Status       string `json:"status" example:"pending"`  // pending/running/completed/failed/cancelled
	Mode         string `json:"mode" example:"auto"`
	Concurrency  int    `json:"concurrency" example:"3"`
	TotalVMs     int    `json:"total_vms" example:"5"`
	MigratedVMs  int    `json:"migrated_vms" example:"0"`
	FailedVMs    int    `json:"failed_vms" example:"0"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
	CreateTime   string `json:"create_time"`
}

// MaintenancePlanResponse 维护计划响应
type MaintenancePlanResponse struct {
	Response
	Data MaintenancePlanDetail `json:"data"`
}

// MaintenancePlanListResponse 维护计划列表响应
type MaintenancePlanListResponse struct {
	Response
	Data struct {
		Total int64                 `json:"total"`
		Plans []MaintenancePlanItem `json:"plans"`
	} `json:"data"`
}

// MaintenancePlanItem 维护计划列表项
type MaintenancePlanItem struct {
	PlanID       int64  `json:"plan_id" example:"1"`
	NodeID       int64  `json:"node_id" example:"1"`
	NodeName     string `json:"node_name" example:"pve-node-1"`
	Status       string `json:"status" example:"running"`
	TotalVMs     int    `json:"total_vms" example:"5"`
	MigratedVMs  int    `json:"migrated_vms" example:"3"`
	FailedVMs    int    `json:"failed_vms" example:"0"`
	StartTime    string `json:"start_time"`
	CreateTime   string `json:"create_time"`
}

// MaintenanceTaskDetail 维护任务详情
type MaintenanceTaskDetail struct {
	TaskID      int64  `json:"task_id" example:"1"`
	PlanID      int64  `json:"plan_id" example:"1"`
	VMID        uint32 `json:"vm_id" example:"101"`
	VMName      string `json:"vm_name" example:"vm-nginx"`
	SourceNode  string `json:"source_node" example:"pve-node-1"`
	TargetNode  string `json:"target_node" example:"pve-node-2"`
	Status      string `json:"status" example:"completed"`  // pending/running/completed/failed
	ErrorMessage string `json:"error_message,omitempty"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at"`
}

// MaintenanceTaskListResponse 维护任务列表响应
type MaintenanceTaskListResponse struct {
	Response
	Data struct {
		Total int64                 `json:"total"`
		Tasks []MaintenanceTaskItem `json:"tasks"`
	} `json:"data"`
}

// MaintenanceTaskItem 维护任务列表项
type MaintenanceTaskItem struct {
	TaskID      int64  `json:"task_id" example:"1"`
	VMID        uint32 `json:"vm_id" example:"101"`
	VMName      string `json:"vm_name" example:"vm-nginx"`
	SourceNode  string `json:"source_node" example:"pve-node-1"`
	TargetNode  string `json:"target_node" example:"pve-node-2"`
	Status      string `json:"status" example:"completed"`
	ErrorMessage string `json:"error_message,omitempty"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at"`
}

// MaintenancePlanProgressResponse 维护计划进度响应
type MaintenancePlanProgressResponse struct {
	Response
	Data MaintenancePlanProgress `json:"data"`
}

// MaintenancePlanProgress 维护计划进度
type MaintenancePlanProgress struct {
	PlanID        int64                 `json:"plan_id" example:"1"`
	NodeName      string                `json:"node_name" example:"pve-node-1"`
	Status        string                `json:"status" example:"running"`
	TotalVMs      int                   `json:"total_vms" example:"5"`
	MigratedVMs   int                   `json:"migrated_vms" example:"3"`
	FailedVMs     int                   `json:"failed_vms" example:"0"`
	Progress      float64               `json:"progress" example:"60.0"`  // 进度百分比
	Tasks         []MaintenanceTaskItem `json:"tasks"`
}

// CancelMaintenancePlanRequest 取消维护计划请求
type CancelMaintenancePlanRequest struct {
	PlanID int64 `json:"plan_id" binding:"required" example:"1"`
}

// CancelMaintenancePlanResponse 取消维护计划响应
type CancelMaintenancePlanResponse struct {
	Response
}

// MaintenanceHistoryResponse 维护历史响应
type MaintenanceHistoryResponse struct {
	Response
	Data struct {
		Total   int64            `json:"total"`
		Records []HistoryRecord `json:"records"`
	} `json:"data"`
}

// HistoryRecord 历史记录
type HistoryRecord struct {
	PlanID       int64  `json:"plan_id" example:"1"`
	NodeID       int64  `json:"node_id" example:"1"`
	NodeName     string `json:"node_name" example:"pve-node-1"`
	Status       string `json:"status" example:"completed"`  // completed/failed/cancelled
	TotalVMs     int    `json:"total_vms" example:"5"`
	MigratedVMs  int    `json:"migrated_vms" example:"5"`
	FailedVMs    int    `json:"failed_vms" example:"0"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
	Duration     string `json:"duration"`  // 持续时间
}
