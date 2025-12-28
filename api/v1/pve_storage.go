package v1

import "time"

// PveStorage 相关 API 定义

// CreateStorageRequest 创建存储请求
type CreateStorageRequest struct {
	NodeName     string  `json:"node_name" binding:"required" example:"pve-node-1"`
	ClusterID    int64   `json:"cluster_id" binding:"required" example:"1"`
	Active       int     `json:"active" example:"1"`
	Type         string  `json:"type" example:"dir"`
	Avail        int64   `json:"avail" example:"10737418240"`
	StorageName  string  `json:"storage_name" binding:"required" example:"local"`
	Content      string  `json:"content" example:"images,iso"`
	Used         int64   `json:"used" example:"5368709120"`
	Total        int64   `json:"total" example:"16106127360"`
	Enabled      int     `json:"enabled" example:"1"`
	UsedFraction float64 `json:"used_fraction" example:"0.333"`
	Shared       int     `json:"shared" example:"0"`
}

// UpdateStorageRequest 更新存储请求
type UpdateStorageRequest struct {
	Active       *int     `json:"active,omitempty"`
	Type         *string  `json:"type,omitempty"`
	Avail        *int64   `json:"avail,omitempty"`
	Content      *string  `json:"content,omitempty"`
	Used         *int64   `json:"used,omitempty"`
	Total        *int64   `json:"total,omitempty"`
	Enabled      *int     `json:"enabled,omitempty"`
	UsedFraction *float64 `json:"used_fraction,omitempty"`
	Shared       *int     `json:"shared,omitempty"`
}

// ListStorageRequest 列表查询请求
type ListStorageRequest struct {
	Page        int    `form:"page" example:"1"`
	PageSize    int    `form:"page_size" binding:"omitempty,max=100" example:"10"`
	ClusterID   int64  `form:"cluster_id" example:"1"`
	NodeName    string `form:"node_name" example:"pve-node-1"`
	Type        string `form:"type" example:"dir"`
	StorageName string `form:"storage_name" example:"local"`
}

// ListStorageResponse 列表查询响应
type ListStorageResponse struct {
	Response
	Data ListStorageResponseData
}

type ListStorageResponseData struct {
	Total int64         `json:"total"`
	List  []StorageItem `json:"list"`
}

type StorageItem struct {
	Id           int64   `json:"id"`
	NodeName     string  `json:"node_name"`
	NodeID       int64   `json:"node_id"` // 节点ID（关联字段）
	ClusterID    int64   `json:"cluster_id"`
	Active       int     `json:"active"`
	Type         string  `json:"type"`
	Avail        int64   `json:"avail"`
	StorageName  string  `json:"storage_name"`
	Content      string  `json:"content"`
	Used         int64   `json:"used"`
	Total        int64   `json:"total"`
	Enabled      int     `json:"enabled"`
	UsedFraction float64 `json:"used_fraction"`
	Shared       int     `json:"shared"`
}

// GetStorageResponse 详情查询响应
type GetStorageResponse struct {
	Response
	Data StorageDetail
}

type StorageDetail struct {
	Id           int64     `json:"id"`
	NodeName     string    `json:"node_name"`
	NodeID       int64     `json:"node_id"` // 节点ID（关联字段）
	ClusterID    int64     `json:"cluster_id"`
	Active       int       `json:"active"`
	Type         string    `json:"type"`
	Avail        int64     `json:"avail"`
	StorageName  string    `json:"storage_name"`
	Content      string    `json:"content"`
	Used         int64     `json:"used"`
	Total        int64     `json:"total"`
	Enabled      int       `json:"enabled"`
	UsedFraction float64   `json:"used_fraction"`
	Shared       int       `json:"shared"`
	CreateTime   time.Time `json:"create_time"` // 创建时间
	UpdateTime   time.Time `json:"update_time"` // 更新时间
	Creator      string    `json:"creator"`     // 创建者
	Modifier     string    `json:"modifier"`    // 修改者
}

// GetStorageStatusRequest 获取存储状态请求
type GetStorageStatusRequest struct {
	NodeID  int64  `form:"node_id" binding:"required" example:"1"`     // 节点ID（数据库ID）
	Storage string `form:"storage" binding:"required" example:"local"` // 存储名称
}

// GetStorageStatusResponse 获取存储状态响应
type GetStorageStatusResponse struct {
	Response
	Data map[string]interface{} `json:"data"`
}

// GetStorageRRDDataRequest 获取存储 RRD 监控数据请求
type GetStorageRRDDataRequest struct {
	NodeID    int64  `form:"node_id" binding:"required" example:"1"`     // 节点ID
	Storage   string `form:"storage" binding:"required" example:"local"` // 存储名称
	Timeframe string `form:"timeframe" binding:"required" example:"day"` // 时间范围 hour|day|week|month|year
	Cf        string `form:"cf" binding:"required" example:"AVERAGE"`    // AVERAGE|MAX
}

// GetStorageRRDDataResponse 获取存储 RRD 监控数据响应
type GetStorageRRDDataResponse struct {
	Response
	Data []map[string]interface{} `json:"data"`
}

// GetStorageContentRequest 获取存储内容列表请求
type GetStorageContentRequest struct {
	NodeID  int64  `form:"node_id" binding:"required" example:"1"`     // 节点ID
	Storage string `form:"storage" binding:"required" example:"local"` // 存储名称
	Content string `form:"content" example:"images"`                   // 内容类型过滤: images,iso,backup 等
}

// GetStorageContentResponse 获取存储内容列表响应
type GetStorageContentResponse struct {
	Response
	Data []map[string]interface{} `json:"data"`
}

// GetStorageVolumeRequest 获取卷属性请求
type GetStorageVolumeRequest struct {
	NodeID  int64  `form:"node_id" binding:"required" example:"1"`            // 节点ID
	Storage string `form:"storage" binding:"required" example:"local"`        // 存储名称
	Volume  string `form:"volume" binding:"required" example:"vm-100-disk-0"` // 卷标识（volume ID）
}

// GetStorageVolumeResponse 获取卷属性响应
type GetStorageVolumeResponse struct {
	Response
	Data map[string]interface{} `json:"data"`
}

// DeleteStorageContentRequest 删除存储内容请求
type DeleteStorageContentRequest struct {
	NodeID  int64  `form:"node_id" binding:"required" example:"1"`                                       // 节点ID
	Storage string `form:"storage" binding:"required" example:"local-dir"`                               // 存储名称
	Volume  string `form:"volume" binding:"required" example:"/local-dir:iso/baohe_pro_8_51_0_1619.iso"` // 卷标识（volume ID，需要完整路径）
	Delay   *int   `form:"delay" example:"5"`                                                            // 延迟删除时间（秒，可选）
}

// DeleteStorageContentResponse 删除存储内容响应
type DeleteStorageContentResponse struct {
	Response
}
