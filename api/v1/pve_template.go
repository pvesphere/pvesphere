package v1

import "time"

// PveTemplate 相关 API 定义

// CreateTemplateRequest 创建模板请求
type CreateTemplateRequest struct {
	TemplateName string `json:"template_name" binding:"required" example:"centos7.9-x86-64-temp"`
	ClusterID    int64  `json:"cluster_id" binding:"required" example:"1"`
	Description  string `json:"description" example:"模板描述"`
}

// UpdateTemplateRequest 更新模板请求
type UpdateTemplateRequest struct {
	TemplateName *string `json:"template_name,omitempty"`
	Description  *string `json:"description,omitempty"`
}

// ListTemplateRequest 列表查询请求
type ListTemplateRequest struct {
	Page      int   `form:"page" example:"1"`
	PageSize  int   `form:"page_size" binding:"omitempty,max=100" example:"10"`
	ClusterID int64 `form:"cluster_id" example:"1"`
}

// ListTemplateResponse 列表查询响应
type ListTemplateResponse struct {
	Response
	Data ListTemplateResponseData
}

type ListTemplateResponseData struct {
	Total int64          `json:"total"`
	List  []TemplateItem `json:"list"`
}

type TemplateItem struct {
	Id           int64  `json:"id"`
	TemplateName string `json:"template_name"`
	ClusterID    int64  `json:"cluster_id"`
	ClusterName  string `json:"cluster_name"` // 从关联表查询填充
	Description  string `json:"description"`
}

// GetTemplateResponse 详情查询响应
type GetTemplateResponse struct {
	Response
	Data TemplateDetail
}

type TemplateDetail struct {
	Id           int64     `json:"id"`
	TemplateName string    `json:"template_name"`
	ClusterID    int64     `json:"cluster_id"`
	ClusterName  string    `json:"cluster_name"` // 从关联表查询填充
	Description  string    `json:"description"`
	CreateTime   time.Time `json:"create_time"`  // 创建时间
	UpdateTime   time.Time `json:"update_time"`  // 更新时间
	Creator      string    `json:"creator"`      // 创建者
	Modifier     string    `json:"modifier"`     // 修改者
}

// ========================
// 模板导入相关 API
// ========================

// ImportTemplateRequest 导入模板请求（基于已有备份文件）
type ImportTemplateRequest struct {
	TemplateName    string  `json:"template_name" binding:"required" example:"centos7-template"`
	ClusterID       int64   `json:"cluster_id" binding:"required" example:"1"`
	NodeID          int64   `json:"node_id" binding:"required" example:"1"`                                               // 导入节点ID
	BackupStorageID int64   `json:"backup_storage_id" binding:"required" example:"6"`                                     // 备份文件所在的存储ID（通常是local）
	BackupFile      string  `json:"backup_file" binding:"required" example:"vzdump-qemu-100-2024_01_01-00_00_00.vma.zst"` // 备份文件名
	TargetStorageID int64   `json:"target_storage_id" binding:"required" example:"7"`                                     // VM磁盘要创建的目标存储ID（必须支持images，如local-lvm）
	Description     string  `json:"description" example:"CentOS 7 模板"`
	AutoSync        bool    `json:"auto_sync" example:"false"`   // local存储时是否自动同步到所有节点
	SyncNodeIDs     []int64 `json:"sync_node_ids" example:"2,3"` // local存储时，指定要同步的节点ID列表
}

// ImportTemplateResponse 导入模板响应
type ImportTemplateResponse struct {
	Response
	Data ImportTemplateResponseData `json:"data"`
}

type ImportTemplateResponseData struct {
	TemplateID  int64                  `json:"template_id"`
	ImportID    int64                  `json:"import_id"`
	StorageType string                 `json:"storage_type"`
	IsShared    bool                   `json:"is_shared"`
	ImportNode  TemplateImportNode     `json:"import_node"`
	SyncTasks   []TemplateSyncTaskInfo `json:"sync_tasks,omitempty"`
}

type TemplateImportNode struct {
	NodeID   int64  `json:"node_id"`
	NodeName string `json:"node_name"`
}

type TemplateSyncTaskInfo struct {
	TaskID         int64  `json:"task_id"`
	TargetNodeID   int64  `json:"target_node_id"`
	TargetNodeName string `json:"target_node_name"`
	Status         string `json:"status"`
}

// GetTemplateDetailRequest 查询模板详情（包含实例）
type GetTemplateDetailRequest struct {
	IncludeInstances bool `form:"include_instances" example:"true"` // 是否包含实例信息
}

// GetTemplateDetailResponse 模板详情响应
type GetTemplateDetailResponse struct {
	Response
	Data TemplateDetailWithInstances `json:"data"`
}

type TemplateDetailWithInstances struct {
	Id           int64                  `json:"id"`
	TemplateName string                 `json:"template_name"`
	ClusterID    int64                  `json:"cluster_id"`
	ClusterName  string                 `json:"cluster_name"`
	Description  string                 `json:"description"`
	UploadInfo   *TemplateUploadInfo    `json:"upload_info,omitempty"`
	Instances    []TemplateInstanceInfo `json:"instances,omitempty"`
}

type TemplateUploadInfo struct {
	UploadID    int64  `json:"upload_id"`
	StorageName string `json:"storage_name"`
	IsShared    bool   `json:"is_shared"`
	FileName    string `json:"file_name"`
	FileSize    int64  `json:"file_size"`
	Status      string `json:"status"`
}

type TemplateInstanceInfo struct {
	InstanceID   int64  `json:"instance_id"`
	NodeID       int64  `json:"node_id"`
	NodeName     string `json:"node_name"`
	VMID         uint32 `json:"vmid"`
	StorageName  string `json:"storage_name"`
	Status       string `json:"status"`
	IsPrimary    bool   `json:"is_primary"`
	SyncProgress *int   `json:"sync_progress,omitempty"`
}

// ========================
// 模板同步相关 API
// ========================

// SyncTemplateRequest 同步模板请求
type SyncTemplateRequest struct {
	TargetNodeIDs []int64 `json:"target_node_ids" binding:"required" example:"3,4"`
}

// SyncTemplateResponse 同步模板响应
type SyncTemplateResponse struct {
	Response
	Data SyncTemplateResponseData `json:"data"`
}

type SyncTemplateResponseData struct {
	SyncTasks []TemplateSyncTaskInfo `json:"sync_tasks"`
}

// GetSyncTaskResponse 查询同步任务响应
type GetSyncTaskResponse struct {
	Response
	Data SyncTaskDetail `json:"data"`
}

type SyncTaskDetail struct {
	TaskID        int64      `json:"task_id"`
	TemplateID    int64      `json:"template_id"`
	TemplateName  string     `json:"template_name"`
	SourceNode    NodeInfo   `json:"source_node"`
	TargetNode    NodeInfo   `json:"target_node"`
	StorageName   string     `json:"storage_name"`
	Status        string     `json:"status"`
	Progress      int        `json:"progress"`
	SyncStartTime *time.Time `json:"sync_start_time,omitempty"`
	SyncEndTime   *time.Time `json:"sync_end_time,omitempty"`
	ErrorMessage  string     `json:"error_message,omitempty"`
}

type NodeInfo struct {
	NodeID   int64  `json:"node_id"`
	NodeName string `json:"node_name"`
}

// ListSyncTasksRequest 列出同步任务请求
type ListSyncTasksRequest struct {
	Page       int    `form:"page" example:"1"`
	PageSize   int    `form:"page_size" binding:"omitempty,max=100" example:"10"`
	TemplateID *int64 `form:"template_id" example:"1"`
	Status     string `form:"status" example:"pending"`
}

// ListSyncTasksResponse 列出同步任务响应
type ListSyncTasksResponse struct {
	Response
	Data ListSyncTasksResponseData `json:"data"`
}

type ListSyncTasksResponseData struct {
	Total int64            `json:"total"`
	List  []SyncTaskDetail `json:"list"`
}

// RetrySyncTaskResponse 重试同步任务响应
type RetrySyncTaskResponse struct {
	Response
	Data RetrySyncTaskResponseData `json:"data"`
}

type RetrySyncTaskResponseData struct {
	TaskID int64  `json:"task_id"`
	Status string `json:"status"`
}

// ========================
// 模板实例相关 API
// ========================

// ListTemplateInstancesRequest 列出模板实例请求
type ListTemplateInstancesRequest struct {
	TemplateID int64 `form:"template_id" binding:"required" example:"1"`
}

// ListTemplateInstancesResponse 列出模板实例响应
type ListTemplateInstancesResponse struct {
	Response
	Data ListTemplateInstancesResponseData `json:"data"`
}

type ListTemplateInstancesResponseData struct {
	TemplateID   int64                  `json:"template_id"`
	TemplateName string                 `json:"template_name"`
	Total        int64                  `json:"total"`
	Instances    []TemplateInstanceInfo `json:"instances"`
}
