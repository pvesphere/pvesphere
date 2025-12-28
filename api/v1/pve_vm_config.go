package v1

// PveVMConfig 相关 API 定义

// GetVMCurrentConfigRequest 获取虚拟机当前配置请求
type GetVMCurrentConfigRequest struct {
	VMID int64 `form:"vm_id" binding:"required" example:"1"` // 虚拟机ID（数据库ID）
}

// GetVMCurrentConfigResponse 获取虚拟机当前配置响应
type GetVMCurrentConfigResponse struct {
	Response
	Data map[string]interface{} `json:"data"`
}

// GetVMPendingConfigRequest 获取虚拟机pending配置请求
type GetVMPendingConfigRequest struct {
	VMID int64 `form:"vm_id" binding:"required" example:"1"` // 虚拟机ID（数据库ID）
}

// GetVMPendingConfigResponse 获取虚拟机pending配置响应
type GetVMPendingConfigResponse struct {
	Response
	Data []map[string]interface{} `json:"data"`
}

// UpdateVMConfigRequest 更新虚拟机配置请求
type UpdateVMConfigRequest struct {
	VMID   int64                  `json:"vm_id" binding:"required" example:"1"` // 虚拟机ID（数据库ID）
	Config map[string]interface{} `json:"config" binding:"required"`            // 配置项，key-value格式
}

// UpdateVMConfigResponse 更新虚拟机配置响应
type UpdateVMConfigResponse struct {
	Response
}

// GetVMStatusRequest 获取虚拟机状态请求
type GetVMStatusRequest struct {
	VMID int64 `form:"vm_id" binding:"required" example:"1"` // 虚拟机ID（数据库ID）
}

// GetVMStatusResponse 获取虚拟机状态响应
type GetVMStatusResponse struct {
	Response
	Data map[string]interface{} `json:"data"`
}

// GetVMConsoleRequest 获取虚拟机 Console（VNCProxy）请求
type GetVMConsoleRequest struct {
	VMID             int64 `json:"vm_id" binding:"required" example:"1"` // 虚拟机ID（数据库ID）
	Websocket        bool  `json:"websocket,omitempty" example:"false"`  // 是否启用 websocket（可选）
	GeneratePassword bool  `json:"generate_password,omitempty" example:"false"`
}

// GetVMConsoleResponse 获取虚拟机 Console（VNCProxy）响应
type GetVMConsoleResponse struct {
	Response
	Data map[string]interface{} `json:"data"`
}
