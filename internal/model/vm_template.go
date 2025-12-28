package model

import (
	"time"
)

type VmTemplate struct {
	Id           int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	TemplateName string    `json:"template_name" gorm:"column:template_name"`
	ClusterID    int64     `json:"cluster_id" gorm:"column:cluster_id"`
	Description  string    `json:"description" gorm:"column:description"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:gmt_create"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"column:gmt_modified"`
	Creator      string    `json:"creator" gorm:"column:creator"`
	Modifier     string    `json:"modifier" gorm:"column:modifier"`
}

func (VmTemplate) TableName() string {
	return "vm_template"
}
