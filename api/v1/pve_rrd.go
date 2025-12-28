package v1

// PveRRD 相关 API 定义

// GetNodeRRDDataRequest 获取节点RRD监控数据请求
type GetNodeRRDDataRequest struct {
	NodeID    int64  `form:"node_id" binding:"required" example:"1"`                                     // 节点ID（数据库ID）
	Timeframe string `form:"timeframe" binding:"required,oneof=hour day week month year" example:"hour"` // 时间范围
	Cf        string `form:"cf" binding:"required,oneof=AVERAGE MAX" example:"AVERAGE"`                  // 聚合函数
}

// GetNodeRRDDataResponse 获取节点RRD监控数据响应
type GetNodeRRDDataResponse struct {
	Response
	Data []map[string]interface{} `json:"data"`
}

// GetVMRRDDataRequest 获取虚拟机RRD监控数据请求
type GetVMRRDDataRequest struct {
	VMID      int64  `form:"vm_id" binding:"required" example:"1"`                                       // 虚拟机ID（数据库ID）
	Timeframe string `form:"timeframe" binding:"required,oneof=hour day week month year" example:"hour"` // 时间范围
	Cf        string `form:"cf" binding:"required,oneof=AVERAGE MAX" example:"AVERAGE"`                  // 聚合函数
}

// GetVMRRDDataResponse 获取虚拟机RRD监控数据响应
type GetVMRRDDataResponse struct {
	Response
	Data []map[string]interface{} `json:"data"`
}
