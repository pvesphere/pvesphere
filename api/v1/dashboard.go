package v1

// Dashboard 相关 API 定义

// ==================== Scopes ====================

// DashboardScopesResponse 获取可选集群列表响应
type DashboardScopesResponse struct {
	Response
	Data DashboardScopesData `json:"data"`
}

type DashboardScopesData struct {
	Items []ScopeItem `json:"items"`
}

type ScopeItem struct {
	ClusterID        int64  `json:"cluster_id" example:"1"`
	ClusterName      string `json:"cluster_name" example:"pve-prod-01"`
	ClusterNameAlias string `json:"cluster_name_alias" example:"生产集群一"`
}

// ==================== Overview ====================

// DashboardOverviewRequest 全局概览请求
type DashboardOverviewRequest struct {
	Scope     string `form:"scope" example:"all"`    // all 或 cluster
	ClusterID *int64 `form:"cluster_id" example:"1"` // 当 scope 为 cluster 时使用
}

// DashboardOverviewResponse 全局概览响应
type DashboardOverviewResponse struct {
	Response
	Data DashboardOverviewData `json:"data"`
}

type DashboardOverviewData struct {
	Scope     string                   `json:"scope" example:"all"`  // all 或 cluster
	ClusterID *int64                   `json:"cluster_id,omitempty"` // 集群ID（当 scope 为 cluster 时）
	Summary   DashboardOverviewSummary `json:"summary"`              // 概览统计
	Health    DashboardOverviewHealth  `json:"health"`               // 健康状态
}

type DashboardOverviewSummary struct {
	ClusterCount int64 `json:"cluster_count" example:"5"`  // 集群数量
	NodeCount    int64 `json:"node_count" example:"42"`    // 节点数量
	VMCount      int64 `json:"vm_count" example:"1328"`    // 虚拟机数量
	StorageCount int64 `json:"storage_count" example:"18"` // 存储数量
}

type DashboardOverviewHealth struct {
	Healthy  int64 `json:"healthy" example:"4"`  // 健康数量
	Warning  int64 `json:"warning" example:"1"`  // 警告数量
	Critical int64 `json:"critical" example:"0"` // 严重数量
}

// ==================== Resources ====================

// DashboardResourcesRequest 资源使用率请求
type DashboardResourcesRequest struct {
	Scope     string `form:"scope" example:"all"`    // all 或 cluster
	ClusterID *int64 `form:"cluster_id" example:"1"` // 当 scope 为 cluster 时使用
}

// DashboardResourcesResponse 资源使用率响应
type DashboardResourcesResponse struct {
	Response
	Data DashboardResourcesData `json:"data"`
}

type DashboardResourcesData struct {
	Scope     string        `json:"scope" example:"all"`  // all 或 cluster
	ClusterID *int64        `json:"cluster_id,omitempty"` // 集群ID（当 scope 为 cluster 时）
	CPU       ResourceUsage `json:"cpu"`                  // CPU 使用率
	Memory    ResourceUsage `json:"memory"`               // 内存使用率
	Storage   ResourceUsage `json:"storage"`              // 存储使用率
}

type ResourceUsage struct {
	UsedCores    *float64 `json:"used_cores,omitempty" example:"180"`           // CPU 已使用核心数（仅 CPU）
	TotalCores   *float64 `json:"total_cores,omitempty" example:"250"`          // CPU 总核心数（仅 CPU）
	UsedBytes    *int64   `json:"used_bytes,omitempty" example:"180000000000"`  // 已使用字节数（内存/存储）
	TotalBytes   *int64   `json:"total_bytes,omitempty" example:"260000000000"` // 总字节数（内存/存储）
	UsagePercent float64  `json:"usage_percent" example:"72.0"`                 // 使用率百分比
}

// ==================== Hotspots ====================

// DashboardHotspotsRequest 压力和风险焦点请求
type DashboardHotspotsRequest struct {
	Scope     string `form:"scope" example:"all"`    // all 或 cluster
	ClusterID *int64 `form:"cluster_id" example:"1"` // 当 scope 为 cluster 时使用
	Limit     int    `form:"limit" example:"5"`      // Top N 数量，默认 5
}

// DashboardHotspotsResponse 压力和风险焦点响应
type DashboardHotspotsResponse struct {
	Response
	Data DashboardHotspotsData `json:"data"`
}

type DashboardHotspotsData struct {
	Scope      string              `json:"scope" example:"all"`    // all 或 cluster
	ClusterID  *int64              `json:"cluster_id,omitempty"`  // 集群ID（当 scope 为 cluster 时）
	VMHotspots VMHotspots          `json:"vm_hotspots"`           // 虚拟机热点
	NodeHotspots NodeHotspots      `json:"node_hotspots"`         // 节点热点
	StorageHotspots []StorageHotspot `json:"storage_hotspots"`   // 存储热点
	RecentRisks []RecentRisk       `json:"recent_risks"`         // 最近风险（24h）
}

// VMHotspots 虚拟机热点（按指标类型分组）
type VMHotspots struct {
	CPU    []TopResourceConsumer `json:"cpu"`    // CPU Top N
	Memory []TopResourceConsumer `json:"memory"` // Memory Top N
}

// NodeHotspots 节点热点（按指标类型分组）
type NodeHotspots struct {
	CPU    []TopResourceConsumer `json:"cpu"`    // CPU Top N
	Memory []TopResourceConsumer `json:"memory"` // Memory Top N
}

// StorageHotspot 存储热点
type StorageHotspot struct {
	ID          string  `json:"id" example:"storage/pve01/local"`         // 存储ID
	Name        string  `json:"name" example:"local (pve01)"`             // 存储名称
	UsagePercent float64 `json:"usage_percent" example:"85.5"`            // 使用率百分比
	UsedBytes   int64   `json:"used_bytes" example:"85899345920"`        // 已使用字节
	TotalBytes  int64   `json:"total_bytes" example:"100663296000"`       // 总字节
	Unit        string  `json:"unit" example:"%"`                        // 单位
}

// TopResourceConsumer 资源消耗者
type TopResourceConsumer struct {
	ID          string  `json:"id" example:"qemu/101"`         // 资源ID
	Name        string  `json:"name" example:"web-prod-01"`    // 资源名称
	MetricValue float64 `json:"metric_value" example:"92.0"`  // 指标值
	Unit        string  `json:"unit" example:"%"`              // 单位
	NodeName    string  `json:"node_name,omitempty" example:"pve01"` // 节点名称（VM 使用）
	ClusterID   int64   `json:"cluster_id,omitempty" example:"1"`    // 集群ID
	ClusterName string  `json:"cluster_name,omitempty" example:"cluster1"` // 集群名称
}

type RecentRisk struct {
	ID           string `json:"id" example:"risk-1"`                        // 风险ID
	Level        string `json:"level" example:"warning"`                    // info / warning / critical
	Message      string `json:"message" example:"Node pve-07 unreachable"`  // 风险信息
	OccurredAt   string `json:"occurred_at" example:"2025-12-23T08:01:00Z"` // 发生时间
	RelativeTime string `json:"relative_time" example:"5 min ago"`          // 相对时间
	TargetType   string `json:"target_type" example:"node"`                 // vm / node / storage / cluster
	TargetID     string `json:"target_id" example:"node-7"`                 // 目标ID
	TargetName   string `json:"target_name" example:"pve-07"`               // 目标名称
}

// ==================== Operations ====================

// DashboardOperationsRequest 运行中操作请求
type DashboardOperationsRequest struct {
	Scope     string `form:"scope" example:"all"`    // all 或 cluster
	ClusterID *int64 `form:"cluster_id" example:"1"` // 当 scope 为 cluster 时使用
}

// DashboardOperationsResponse 运行中操作响应
type DashboardOperationsResponse struct {
	Response
	Data DashboardOperationsData `json:"data"`
}

type DashboardOperationsData struct {
	Scope     string             `json:"scope" example:"all"`  // all 或 cluster
	ClusterID *int64             `json:"cluster_id,omitempty"` // 集群ID（当 scope 为 cluster 时）
	Summary   []OperationSummary `json:"summary"`              // 操作摘要（按类型聚合）
	Items     []OperationItem    `json:"items,omitempty"`      // 操作明细（可选）
}

type OperationSummary struct {
	OperationType string `json:"operation_type" example:"vm_migration"` // vm_migration / node_maintenance / storage_rebalance
	DisplayName   string `json:"display_name" example:"VM Migrations"`  // 显示名称
	Count         int64  `json:"count" example:"3"`                     // 数量
}

type OperationItem struct {
	ID            string `json:"id" example:"op-1"`                         // 操作ID
	OperationType string `json:"operation_type" example:"vm_migration"`     // 操作类型
	Name          string `json:"name" example:"Migrate vm-101 to node-3"`   // 操作名称
	Progress      int    `json:"progress" example:"45"`                     // 进度百分比
	Status        string `json:"status" example:"running"`                  // running / pending / failed / completed
	StartedAt     string `json:"started_at" example:"2025-12-23T07:50:00Z"` // 开始时间
}
