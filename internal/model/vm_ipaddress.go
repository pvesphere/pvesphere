package model

type VMIPAddress struct {
	Id          int64  `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	IPAddress   string `json:"ip_address" gorm:"column:ip_address"`
	NetworkID   int64  `json:"network_id" gorm:"column:network_id"`
	NicName     string `json:"nic_name" gorm:"column:nic_name"`
	VMId        int64  `json:"vm_id" gorm:"column:vm_id;index"`
	MacAddress  string `json:"mac_address" gorm:"column:mac_address"`
	ClusterID   int64  `json:"cluster_id" gorm:"column:cluster_id;index"`   // 集群ID（关联字段）
	Creator     string `json:"creator" gorm:"column:creator"`
	Modifier    string `json:"modifier" gorm:"column:modifier"`
	
	// 以下字段仅用于查询时的 JOIN 填充，不存储在数据库中
	ClusterName string `json:"cluster_name,omitempty" gorm:"-"` // 从关联表查询填充
}

func (VMIPAddress) TableName() string {
	return "vm_ipaddress"
}
