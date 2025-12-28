package model

import (
	"time"
)

type PveVM struct {
	Id           int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	VmName       string    `json:"vm_name" gorm:"column:vm_name"`
	NodeID       int64     `json:"node_id" gorm:"column:node_id;index"`      // 节点ID（关联字段）
	VMID         uint32    `json:"vmid" gorm:"column:vmid"`
	CPUNum       int       `json:"cpu_num" gorm:"column:cpu_num"`
	MemorySize   int       `json:"memory_size" gorm:"column:memory_size"`
	Storage      string    `json:"storage" gorm:"column:storages"`
	StorageCfg   string    `json:"storage_cfg" gorm:"column:storage_cfg"`
	AppId        string    `json:"app_id" gorm:"column:appid"`
	ClusterID    int64     `json:"cluster_id" gorm:"column:cluster_id;index"`   // 集群ID（关联字段）
	Status       string    `json:"status" gorm:"column:status"`
	IsTemplate   int8      `json:"is_template" gorm:"column:is_template;default:0"` // 是否为模板：0=否, 1=是
	TemplateID   int64     `json:"template_id" gorm:"column:template_id;index"` // 模板ID（关联字段）
	VmUser       string    `json:"vm_user" gorm:"column:vm_user"`
	VmPassword   string    `json:"-" gorm:"column:vm_password"`
	NodeIP       string    `json:"node_ip" gorm:"column:node_ip"`               // 节点IP（冗余，用于快速访问，IP 很少变化）
	Creator      string    `json:"creator" gorm:"column:creator"`
	Modifier     string    `json:"modifier" gorm:"column:modifier"`
	Description  string    `json:"descriptions" gorm:"column:descriptions"`
	CreateTime   time.Time `json:"create_time" gorm:"column:gmt_create"`
	UpdateTime   time.Time `json:"update_time" gorm:"column:gmt_modified"`
	ResourceHash string    `json:"resource_hash" gorm:"column:resource_hash;index"`
	LastSyncTime time.Time `json:"last_sync_time" gorm:"column:last_sync_time"`
	
	// 以下字段仅用于查询时的 JOIN 填充，不存储在数据库中
	ClusterName  string `json:"cluster_name,omitempty" gorm:"-"`   // 从关联表查询填充
	NodeName     string `json:"node_name,omitempty" gorm:"-"`      // 从关联表查询填充
	TemplateName string `json:"template_name,omitempty" gorm:"-"`  // 从关联表查询填充
}

func (PveVM) TableName() string {
	return "pve_vm"
}
