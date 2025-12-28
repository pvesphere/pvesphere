package model

import "time"

// TemplateInstance 模板实例（模板在特定节点上的状态）
type TemplateInstance struct {
	Id         int64  `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	TemplateID int64  `json:"template_id" gorm:"column:template_id;not null;index"`
	UploadID   int64  `json:"upload_id" gorm:"column:upload_id;not null;index"`
	ClusterID  int64  `json:"cluster_id" gorm:"column:cluster_id;not null;index"`
	
	NodeID   int64  `json:"node_id" gorm:"column:node_id;not null;index"`
	NodeName string `json:"node_name" gorm:"column:node_name;size:100;not null"`
	
	StorageID   int64  `json:"storage_id" gorm:"column:storage_id;not null;index"`
	StorageName string `json:"storage_name" gorm:"column:storage_name;size:100;not null"`
	IsShared    int8   `json:"is_shared" gorm:"column:is_shared;not null;default:0"`
	
	VMID     uint32 `json:"vmid" gorm:"column:vmid;not null"`
	VolumeID string `json:"volume_id" gorm:"column:volume_id;size:255"`
	
	Status     string `json:"status" gorm:"column:status;size:50;not null;default:'pending';index"`
	SyncTaskID *int64 `json:"sync_task_id" gorm:"column:sync_task_id;index"`
	
	IsPrimary int8 `json:"is_primary" gorm:"column:is_primary;default:0"`
	
	Creator    string    `json:"creator" gorm:"column:creator;size:100"`
	Modifier   string    `json:"modifier" gorm:"column:modifier;size:100"`
	CreateTime time.Time `json:"create_time" gorm:"column:gmt_create;autoCreateTime"`
	UpdateTime time.Time `json:"update_time" gorm:"column:gmt_modified;autoUpdateTime"`
}

func (TemplateInstance) TableName() string {
	return "template_instance"
}

// TemplateInstanceStatus 实例状态常量
const (
	TemplateInstanceStatusPending   = "pending"
	TemplateInstanceStatusSyncing   = "syncing"
	TemplateInstanceStatusAvailable = "available"
	TemplateInstanceStatusFailed    = "failed"
	TemplateInstanceStatusDeleted   = "deleted"
)

