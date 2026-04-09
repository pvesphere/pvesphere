package model

import (
	"time"
)

// NodeMaintenancePlan 主机维护计划
type NodeMaintenancePlan struct {
	Id             int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	NodeID         int64     `json:"node_id" gorm:"column:node_id;index"`                            // 节点 ID
	NodeName       string    `json:"node_name" gorm:"column:node_name"`                              // 节点名称（冗余）
	Status         string    `json:"status" gorm:"column:status;index;default:'pending'"`            // pending/running/completed/failed/cancelled
	Mode           string    `json:"mode" gorm:"column:mode;default:'auto'"`                         // auto/manual
	Concurrency    int       `json:"concurrency" gorm:"column:concurrency;default:3"`                // 并发数
	TotalVMs       int       `json:"total_vms" gorm:"column:total_vms;default:0"`                    // 计划迁移 VM 总数
	MigratedVMs    int       `json:"migrated_vms" gorm:"column:migrated_vms;default:0"`              // 已迁移 VM 数
	FailedVMs      int       `json:"failed_vms" gorm:"column:failed_vms;default:0"`                  // 失败 VM 数
	StartTime      time.Time `json:"start_time" gorm:"column:start_time;index"`                      // 开始时间
	EndTime        time.Time `json:"end_time" gorm:"column:end_time"`                                // 结束时间
	FailedVMIDs    string    `json:"failed_vm_ids,omitempty" gorm:"column:failed_vm_ids;size:1024"`  // 失败 VM ID 列表（逗号分隔）
	ErrorMessage   string    `json:"error_message,omitempty" gorm:"column:error_message;size:2048"`  // 错误信息
	Creator        string    `json:"creator" gorm:"column:creator"`
	CreateTime     time.Time `json:"create_time" gorm:"column:gmt_create;autoCreateTime"`
	UpdateTime     time.Time `json:"update_time" gorm:"column:gmt_modified;autoUpdateTime"`
}

func (NodeMaintenancePlan) TableName() string {
	return "node_maintenance_plan"
}

// NodeMaintenanceTask 主机维护任务
type NodeMaintenanceTask struct {
	Id           int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	PlanID       int64     `json:"plan_id" gorm:"column:plan_id;index"`                            // 关联维护计划 ID
	VMID         uint32    `json:"vm_id" gorm:"column:vm_id;index"`                                // VM ID
	VMName       string    `json:"vm_name" gorm:"column:vm_name"`                                  // VM 名称
	SourceNodeID int64     `json:"source_node_id" gorm:"column:source_node_id"`                    // 源节点 ID
	SourceNode   string    `json:"source_node" gorm:"column:source_node"`                          // 源节点名称
	TargetNodeID int64     `json:"target_node_id" gorm:"column:target_node_id"`                    // 目标节点 ID
	TargetNode   string    `json:"target_node" gorm:"column:target_node"`                          // 目标节点名称
	Status       string    `json:"status" gorm:"column:status;index;default:'pending'"`            // pending/running/completed/failed
	ErrorMessage string    `json:"error_message,omitempty" gorm:"column:error_message;size:2048"`  // 错误信息
	StartedAt    time.Time `json:"started_at" gorm:"column:started_at"`                            // 开始时间
	CompletedAt  time.Time `json:"completed_at" gorm:"column:completed_at"`                        // 完成时间
	CreateTime   time.Time `json:"create_time" gorm:"column:gmt_create;autoCreateTime"`
	UpdateTime   time.Time `json:"update_time" gorm:"column:gmt_modified;autoUpdateTime"`
}

func (NodeMaintenanceTask) TableName() string {
	return "node_maintenance_task"
}
