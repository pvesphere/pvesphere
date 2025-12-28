package model

import "time"

// TemplateUpload 模板导入记录（复用 template_upload 表，但语义改为"导入"）
type TemplateUpload struct {
	Id           int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	TemplateID   int64     `json:"template_id" gorm:"column:template_id;not null;index"`
	ClusterID    int64     `json:"cluster_id" gorm:"column:cluster_id;not null;index"`
	StorageID    int64     `json:"storage_id" gorm:"column:storage_id;not null;index"`
	StorageName  string    `json:"storage_name" gorm:"column:storage_name;size:100;not null"`
	StorageType  string    `json:"storage_type" gorm:"column:storage_type;size:50;not null"`
	IsShared     int8      `json:"is_shared" gorm:"column:is_shared;not null;default:0"`
	
	UploadNodeID   int64  `json:"upload_node_id" gorm:"column:upload_node_id;not null"`      // 导入节点ID（语义改为导入节点）
	UploadNodeName string `json:"upload_node_name" gorm:"column:upload_node_name;size:100;not null"` // 导入节点名称
	
	FileName   string `json:"file_name" gorm:"column:file_name;size:255;not null"`      // 备份文件名
	FilePath   string `json:"file_path" gorm:"column:file_path;size:500;not null"`      // 备份文件路径
	FileSize   int64  `json:"file_size" gorm:"column:file_size;not null;default:0"`     // 备份文件大小
	FileFormat string `json:"file_format" gorm:"column:file_format;size:50;not null"`   // 备份文件格式（vma, vma.zst, vma.lzo等）
	
	Status         string `json:"status" gorm:"column:status;size:50;not null;default:'importing';index"` // 状态：importing, imported, failed
	ImportProgress int    `json:"import_progress" gorm:"column:import_progress;default:0"`
	ErrorMessage   string `json:"error_message" gorm:"column:error_message;type:text"`
	
	Creator    string    `json:"creator" gorm:"column:creator;size:100"`
	Modifier   string    `json:"modifier" gorm:"column:modifier;size:100"`
	CreateTime time.Time `json:"create_time" gorm:"column:gmt_create;autoCreateTime"`
	UpdateTime time.Time `json:"update_time" gorm:"column:gmt_modified;autoUpdateTime"`
}

func (TemplateUpload) TableName() string {
	return "template_upload"
}

// TemplateUploadStatus 导入状态常量
const (
	TemplateUploadStatusImporting = "importing"  // 导入中
	TemplateUploadStatusImported  = "imported"   // 导入完成
	TemplateUploadStatusFailed    = "failed"     // 导入失败
)

