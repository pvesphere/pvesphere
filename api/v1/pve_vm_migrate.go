package v1

// PveVMMigrate 相关 API 定义

// MigrateVMRequest 同集群迁移虚拟机请求
type MigrateVMRequest struct {
	VMID             int64  `json:"vm_id" binding:"required" example:"1"`              // 虚拟机ID（数据库ID）
	TargetNodeID     int64  `json:"target_node_id" binding:"required" example:"2"`     // 目标节点ID（数据库ID）
	Online           *bool  `json:"online,omitempty" example:"true"`                   // 是否在线迁移
	Bwlimit          *int   `json:"bwlimit,omitempty" example:"1000"`                  // 带宽限制（KiB/s）
	WithLocalDisks   *bool  `json:"with_local_disks,omitempty" example:"true"`         // 是否迁移本地磁盘
	MigrationType    string `json:"migration_type,omitempty" example:"secure"`         // 迁移类型：secure（默认）或 insecure
	MigrationNetwork string `json:"migration_network,omitempty" example:"10.0.0.0/24"` // 迁移网络CIDR
	MapStorage       string `json:"map_storage,omitempty" example:"local:shared"`      // 存储映射，格式：FROM:TO
}

// MigrateVMResponse 同集群迁移虚拟机响应
type MigrateVMResponse struct {
	Response
	Data string `json:"data"` // UPID (任务ID)
}

// RemoteMigrateVMRequest 跨集群迁移虚拟机请求
type RemoteMigrateVMRequest struct {
	VMID            int64  `json:"vm_id" binding:"required" example:"1"`                  // 虚拟机ID（数据库ID）
	TargetClusterID int64  `json:"target_cluster_id" binding:"required" example:"2"`      // 目标集群ID（数据库ID）
	TargetNodeID    int64  `json:"target_node_id" binding:"required" example:"3"`         // 目标节点ID（数据库ID）
	TargetBridge    string `json:"target_bridge" binding:"required" example:"vmbr0"`      // 目标网桥（必填）
	TargetStorage   string `json:"target_storage" binding:"required" example:"local-lvm"` // 目标存储（必填）
	TargetVMID      *int64 `json:"target_vmid,omitempty" example:"200"`                   // 目标虚拟机ID（可选，不指定则使用源VMID）
	Online          *bool  `json:"online,omitempty" example:"true"`                       // 是否在线迁移
	Bwlimit         *int   `json:"bwlimit,omitempty" example:"1000"`                      // 带宽限制（KiB/s）
	Delete          *bool  `json:"delete,omitempty" example:"false"`                      // 迁移成功后是否删除源VM（默认false）
}

// RemoteMigrateVMResponse 跨集群迁移虚拟机响应
type RemoteMigrateVMResponse struct {
	Response
	Data string `json:"data"` // UPID (任务ID)
}
