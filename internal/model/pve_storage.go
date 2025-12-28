package model

import (
	"time"
)

type PveStorage struct {
	Id           int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	NodeName     string    `json:"node_name" gorm:"column:node_name"`
	ClusterID    int64     `json:"cluster_id" gorm:"column:cluster_id"`
	Active       int       `json:"active" gorm:"column:active"`
	Type         string    `json:"type" gorm:"column:type"`
	Avail        int64     `json:"avail" gorm:"column:avail"`
	StorageName  string    `json:"storage_name" gorm:"column:storage_name"`
	Content      string    `json:"content" gorm:"column:content"`
	Used         int64     `json:"used" gorm:"column:used"`
	Total        int64     `json:"total" gorm:"column:total"`
	Enabled      int       `json:"enabled" gorm:"column:enabled"`
	UsedFraction float64   `json:"used_fraction" gorm:"column:used_fraction"`
	Shared       int       `json:"shared" gorm:"column:shared"`
	CreateTime   time.Time `json:"create_time" gorm:"column:gmt_create"`
	UpdateTime   time.Time `json:"update_time" gorm:"column:gmt_modified"`
	Creator      string    `json:"creator" gorm:"column:creator"`
	Modifier     string    `json:"modifier" gorm:"column:modifier"`
	ResourceHash string    `json:"resource_hash" gorm:"column:resource_hash;index"`
	LastSyncTime time.Time `json:"last_sync_time" gorm:"column:last_sync_time"`
}

func (PveStorage) TableName() string {
	return "pve_storage"
}
