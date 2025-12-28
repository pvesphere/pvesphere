package v1

// GetNodeDisksListRequest 获取节点磁盘列表请求
type GetNodeDisksListRequest struct {
	NodeID            int64 `form:"node_id" binding:"required" example:"1"` // 节点ID
	IncludePartitions bool  `form:"include_partitions" example:"true"`      // 是否包含分区信息
}

// GetNodeDisksListResponse 获取节点磁盘列表响应
type GetNodeDisksListResponse struct {
	Response
	Data []map[string]interface{} `json:"data"`
}

// GetNodeDisksDirectoryRequest 获取节点 Directory 存储请求
type GetNodeDisksDirectoryRequest struct {
	NodeID int64 `form:"node_id" binding:"required" example:"1"` // 节点ID
}

// GetNodeDisksDirectoryResponse 获取节点 Directory 存储响应
type GetNodeDisksDirectoryResponse struct {
	Response
	Data []map[string]interface{} `json:"data"`
}

// GetNodeDisksLVMRequest 获取节点 LVM 存储请求
type GetNodeDisksLVMRequest struct {
	NodeID int64 `form:"node_id" binding:"required" example:"1"` // 节点ID
}

// GetNodeDisksLVMResponse 获取节点 LVM 存储响应
type GetNodeDisksLVMResponse struct {
	Response
	Data []map[string]interface{} `json:"data"`
}

// GetNodeDisksLVMThinRequest 获取节点 LVM-Thin 存储请求
type GetNodeDisksLVMThinRequest struct {
	NodeID int64 `form:"node_id" binding:"required" example:"1"` // 节点ID
}

// GetNodeDisksLVMThinResponse 获取节点 LVM-Thin 存储响应
type GetNodeDisksLVMThinResponse struct {
	Response
	Data []map[string]interface{} `json:"data"`
}

// GetNodeDisksZFSRequest 获取节点 ZFS 存储请求
type GetNodeDisksZFSRequest struct {
	NodeID int64 `form:"node_id" binding:"required" example:"1"` // 节点ID
}

// GetNodeDisksZFSResponse 获取节点 ZFS 存储响应
type GetNodeDisksZFSResponse struct {
	Response
	Data []map[string]interface{} `json:"data"`
}

// InitGPTDiskRequest 初始化 GPT 磁盘请求
type InitGPTDiskRequest struct {
	NodeID int64  `json:"node_id" binding:"required" example:"1"`     // 节点ID
	Disk   string `json:"disk" binding:"required" example:"/dev/sdb"` // 磁盘设备名
}

// InitGPTDiskResponse 初始化 GPT 磁盘响应
type InitGPTDiskResponse struct {
	Response
	Data string `json:"data"` // 任务ID (UPID)
}

// WipeDiskRequest 擦除磁盘或分区请求
type WipeDiskRequest struct {
	NodeID    int64  `json:"node_id" binding:"required" example:"1"`     // 节点ID
	Disk      string `json:"disk" binding:"required" example:"/dev/sdb"` // 磁盘设备名
	Partition *int   `json:"partition,omitempty" example:"1"`            // 分区号（可选，如果为空则擦除整个磁盘）
}

// WipeDiskResponse 擦除磁盘或分区响应
type WipeDiskResponse struct {
	Response
	Data string `json:"data"` // 任务ID (UPID)
}
