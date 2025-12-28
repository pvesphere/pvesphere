package model

import "time"

// TemplateSyncTask 模板同步任务（用于 local 存储的跨节点同步）
type TemplateSyncTask struct {
	Id         int64  `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	TemplateID int64  `json:"template_id" gorm:"column:template_id;not null;index"`
	UploadID   int64  `json:"upload_id" gorm:"column:upload_id;not null;index"`
	ClusterID  int64  `json:"cluster_id" gorm:"column:cluster_id;not null"`
	
	SourceNodeID   int64  `json:"source_node_id" gorm:"column:source_node_id;not null;index"`
	SourceNodeName string `json:"source_node_name" gorm:"column:source_node_name;size:100;not null"`
	TargetNodeID   int64  `json:"target_node_id" gorm:"column:target_node_id;not null;index"`
	TargetNodeName string `json:"target_node_name" gorm:"column:target_node_name;size:100;not null"`
	
	StorageName string `json:"storage_name" gorm:"column:storage_name;size:100;not null"`
	FilePath    string `json:"file_path" gorm:"column:file_path;size:500;not null"`
	FileSize    int64  `json:"file_size" gorm:"column:file_size;not null;default:0"`
	
	Status   string `json:"status" gorm:"column:status;size:50;not null;default:'pending';index"`
	Progress int    `json:"progress" gorm:"column:progress;default:0"`
	
	SyncStartTime *time.Time `json:"sync_start_time" gorm:"column:sync_start_time"`
	SyncEndTime   *time.Time `json:"sync_end_time" gorm:"column:sync_end_time"`
	
	ErrorMessage string `json:"error_message" gorm:"column:error_message;type:text"`
	
	Creator    string    `json:"creator" gorm:"column:creator;size:100"`
	CreateTime time.Time `json:"create_time" gorm:"column:gmt_create;autoCreateTime"`
	UpdateTime time.Time `json:"update_time" gorm:"column:gmt_modified;autoUpdateTime"`
}

func (TemplateSyncTask) TableName() string {
	return "template_sync_task"
}

// TemplateSyncTaskStatus 同步任务状态常量
const (
	TemplateSyncTaskStatusPending   = "pending"
	TemplateSyncTaskStatusSyncing   = "syncing"
	TemplateSyncTaskStatusImporting = "importing"
	TemplateSyncTaskStatusCompleted = "completed"
	TemplateSyncTaskStatusFailed    = "failed"
)

