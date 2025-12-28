package model

import "time"

// PveTemplate 模板模型（映射到 vm_template 表）
// 与现有的 VmTemplate 结构字段保持一致，共享同一张表。
type PveTemplate struct {
	Id           int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	TemplateName string    `json:"template_name" gorm:"column:template_name"`
	ClusterID    int64     `json:"cluster_id" gorm:"column:cluster_id"`
	Description  string    `json:"description" gorm:"column:description"`
	CreateTime   time.Time `json:"create_time" gorm:"column:gmt_create"`   // 创建时间
	UpdateTime   time.Time `json:"update_time" gorm:"column:gmt_modified"` // 更新时间
	Creator      string    `json:"creator" gorm:"column:creator"`          // 创建者
	Modifier     string    `json:"modifier" gorm:"column:modifier"`        // 修改者
}

// TableName 复用现有的 vm_template 表
func (PveTemplate) TableName() string {
	return "vm_template"
}


