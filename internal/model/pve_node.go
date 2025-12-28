package model

import (
	"time"
)

type PveNode struct {
	Id            int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	NodeName      string    `json:"node_name" gorm:"column:node_name"`
	IPAddress     string    `json:"ip_address" gorm:"column:ip_address"`
	ClusterID     int64     `json:"cluster_id" gorm:"column:cluster_id"`
	IsSchedulable int8      `json:"is_schedulable" gorm:"column:is_schedulable"`
	Env           string    `json:"env" gorm:"column:env"`
	Status        string    `json:"status" gorm:"column:status"`
	CreateTime    time.Time `json:"create_time" gorm:"column:gmt_create"`
	UpdateTime    time.Time `json:"update_time" gorm:"column:gmt_modified"`
	Creator       string    `json:"creator" gorm:"column:creator"`
	Modifier      string    `json:"modifier" gorm:"column:modifier"`
	Annotations   string    `json:"annotations" gorm:"column:annotations"`
	VMLimit       int64     `json:"vm_limit" gorm:"column:vm_limit"`
	ResourceHash  string    `json:"resource_hash" gorm:"column:resource_hash;index"`
	LastSyncTime  time.Time `json:"last_sync_time" gorm:"column:last_sync_time"`
}

func (PveNode) TableName() string {
	return "pve_node"
}
