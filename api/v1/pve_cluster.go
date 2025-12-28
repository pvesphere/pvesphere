package v1

import "time"

// PveCluster 相关 API 定义

// CreateClusterRequest 创建集群请求
type CreateClusterRequest struct {
	ClusterName      string `json:"cluster_name" binding:"required" example:"my-cluster"`
	ClusterNameAlias string `json:"cluster_name_alias" example:"生产集群"`
	Env              string `json:"env" example:"prod"`
	Datacenter       string `json:"datacenter" example:"dc1"`
	ApiUrl           string `json:"api_url" binding:"required" example:"https://10.7.64.206:8006"`
	UserId           string `json:"user_id" binding:"required" example:"api-user@pve"`
	UserToken        string `json:"user_token" binding:"required" example:"your-token"`
	Dns              string `json:"dns" example:"8.8.8.8"`
	Describes        string `json:"describes" example:"集群描述"`
	Region           string `json:"region" example:"us-west-1"`
	IsSchedulable    int8   `json:"is_schedulable" example:"1"`
	IsEnabled        int8   `json:"is_enabled" example:"1"`
}

// UpdateClusterRequest 更新集群请求
type UpdateClusterRequest struct {
	ClusterNameAlias *string `json:"cluster_name_alias,omitempty"`
	Env              *string `json:"env,omitempty"`
	Datacenter       *string `json:"datacenter,omitempty"`
	ApiUrl           *string `json:"api_url,omitempty"`
	UserId           *string `json:"user_id,omitempty"`
	UserToken        *string `json:"user_token,omitempty"`
	Dns              *string `json:"dns,omitempty"`
	Describes        *string `json:"describes,omitempty"`
	Region           *string `json:"region,omitempty"`
	IsSchedulable    *int8   `json:"is_schedulable,omitempty"`
	IsEnabled        *int8   `json:"is_enabled,omitempty"`
}

// ListClusterRequest 列表查询请求
type ListClusterRequest struct {
	Page     int    `form:"page" example:"1"`
	PageSize int    `form:"page_size" binding:"omitempty,max=100" example:"10"`
	Env      string `form:"env" example:"prod"`
	Region   string `form:"region" example:"us-west-1"`
}

// ListClusterResponse 列表查询响应
type ListClusterResponse struct {
	Response
	Data ListClusterResponseData
}

type ListClusterResponseData struct {
	Total int64         `json:"total"`
	List  []ClusterItem `json:"list"`
}

type ClusterItem struct {
	Id               int64  `json:"id"`
	ClusterName      string `json:"cluster_name"`
	ClusterNameAlias string `json:"cluster_name_alias"`
	Env              string `json:"env"`
	Datacenter       string `json:"datacenter"`
	ApiUrl           string `json:"api_url"`
	Region           string `json:"region"`
	IsSchedulable    int8   `json:"is_schedulable"`
	IsEnabled        int8   `json:"is_enabled"`
}

// GetClusterResponse 详情查询响应
type GetClusterResponse struct {
	Response
	Data ClusterDetail
}

type ClusterDetail struct {
	Id               int64     `json:"id"`
	ClusterName      string    `json:"cluster_name"`
	ClusterNameAlias string    `json:"cluster_name_alias"`
	Env              string    `json:"env"`
	Datacenter       string    `json:"datacenter"`
	ApiUrl           string    `json:"api_url"`
	UserId           string    `json:"user_id"`
	Dns              string    `json:"dns"`
	Describes        string    `json:"describes"`
	Region           string    `json:"region"`
	IsSchedulable    int8      `json:"is_schedulable"`
	IsEnabled        int8      `json:"is_enabled"`
	CreateTime       time.Time `json:"create_time"` // 创建时间
	UpdateTime       time.Time `json:"update_time"` // 更新时间
	Creator          string    `json:"creator"`     // 创建者
	Modifier         string    `json:"modifier"`    // 修改者
}

// GetClusterStatusRequest 获取集群状态请求
type GetClusterStatusRequest struct {
	ClusterID int64 `form:"cluster_id" binding:"required" example:"1"` // 集群ID
}

// GetClusterStatusResponse 获取集群状态响应
type GetClusterStatusResponse struct {
	Response
	Data []map[string]interface{} `json:"data"`
}

// GetClusterResourcesRequest 获取集群资源请求
type GetClusterResourcesRequest struct {
	ClusterID int64 `form:"cluster_id" binding:"required" example:"1"` // 集群ID
}

// GetClusterResourcesResponse 获取集群资源响应
type GetClusterResourcesResponse struct {
	Response
	Data []map[string]interface{} `json:"data"`
}

// VerifyClusterRequest 验证集群连接请求
// 支持两种验证方式：
// 1. 通过 cluster_id 验证（从数据库获取集群信息）
// 2. 通过 api_url + user_id + user_token 直接验证（不依赖数据库）
type VerifyClusterRequest struct {
	ClusterID  *int64 `form:"cluster_id" example:"1"`                                    // 集群ID（可选）
	ApiUrl     string `form:"api_url" example:"https://10.7.64.206:8006"`               // API地址（可选）
	UserId     string `form:"user_id" example:"api-user@pve"`                          // 用户ID（可选）
	UserToken  string `form:"user_token" example:"your-token"`                          // 用户Token（可选）
}

// VerifyClusterResponse 验证集群连接响应
type VerifyClusterResponse struct {
	Response
	Data VerifyClusterData `json:"data"`
}

type VerifyClusterData struct {
	Version   string `json:"version" example:"8.3.0"`                           // Proxmox VE 版本
	Release   string `json:"release" example:"8.3"`                             // 发行版本
	RepoID    string `json:"repoid" example:"c1689ccb"`                         // 仓库ID
	Connected bool   `json:"connected" example:"true"`                          // 连接状态
	Message   string `json:"message,omitempty" example:"connection successful"` // 附加信息
}
