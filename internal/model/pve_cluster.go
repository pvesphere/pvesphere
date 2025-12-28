package model

import (
	"time"
)

type PveCluster struct {
	Id               int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	ClusterName      string    `json:"cluster_name" gorm:"column:cluster_name"`
	ClusterNameAlias string    `json:"cluster_name_alias" gorm:"column:cluster_name_alias"`
	Env              string    `json:"env" gorm:"column:env"`
	Datacenter       string    `json:"datacenter" gorm:"column:datacenter"`
	ApiUrl           string    `json:"api_url" gorm:"column:api_url"`
	UserId           string    `json:"user_id" gorm:"column:user_id"`
	UserToken        string    `json:"user_token" gorm:"column:user_token"`
	Dns              string    `json:"dns" gorm:"column:dns"`
	Describes        string    `json:"describes" gorm:"column:describes"`
	Region           string    `json:"region" gorm:"column:region"`
	IsSchedulable    int8      `json:"is_schedulable" gorm:"column:is_schedulable"` // 是否可调度（用于虚拟机创建）
	IsEnabled        int8      `json:"is_enabled" gorm:"column:is_enabled"`         // 是否启用数据自动上报，1-启用，0-禁用
	CreateTime       time.Time `json:"create_time" gorm:"column:gmt_create"`        // 创建时间
	UpdateTime       time.Time `json:"update_time" gorm:"column:gmt_modified"`      // 更新时间
	Creator          string    `json:"creator" gorm:"column:creator"`               // 创建者
	Modifier         string    `json:"modifier" gorm:"column:modifier"`             // 修改者
}

func (PveCluster) TableName() string {
	return "pve_cluster"
}
