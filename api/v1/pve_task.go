package v1

// PveTask 相关 API 定义

// ListClusterTasksRequest 集群任务列表请求
type ListClusterTasksRequest struct {
	ClusterID int64 `form:"cluster_id" binding:"required" example:"1"`
}

// ListClusterTasksResponse 集群任务列表响应
type ListClusterTasksResponse struct {
	Response
	Data []ClusterTaskItem `json:"data"`
}

// ClusterTaskItem 集群任务项
type ClusterTaskItem struct {
	UPID      string                 `json:"upid"`
	Type      string                 `json:"type"`
	ID        string                 `json:"id"`
	User      string                 `json:"user"`
	Status    string                 `json:"status"`
	StartTime int64                  `json:"starttime"`
	EndTime   int64                  `json:"endtime"`
	Node      string                 `json:"node,omitempty"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// ListNodeTasksRequest 节点任务列表请求
type ListNodeTasksRequest struct {
	ClusterID int64  `form:"cluster_id" binding:"required" example:"1"`
	NodeName  string `form:"node_name" binding:"required" example:"pve-node1"`
}

// ListNodeTasksResponse 节点任务列表响应
type ListNodeTasksResponse struct {
	Response
	Data []NodeTaskItem `json:"data"`
}

// NodeTaskItem 节点任务项
type NodeTaskItem struct {
	UPID      string                 `json:"upid"`
	Type      string                 `json:"type"`
	ID        string                 `json:"id"`
	User      string                 `json:"user"`
	Status    string                 `json:"status"`
	StartTime int64                  `json:"starttime"`
	EndTime   int64                  `json:"endtime"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// GetTaskLogRequest 获取任务日志请求
type GetTaskLogRequest struct {
	ClusterID int64  `form:"cluster_id" binding:"required" example:"1"`
	NodeName  string `form:"node_name" binding:"required" example:"pve-node1"`
	UPID      string `form:"upid" binding:"required" example:"UPID:pve-node1:00001234:12345678:90ABCDEF:qmclone:root@pam:"`
	Start     int    `form:"start" example:"0"`
	Limit     int    `form:"limit" example:"50"`
}

// GetTaskLogResponse 获取任务日志响应
type GetTaskLogResponse struct {
	Response
	Data []TaskLogItem `json:"data"`
}

// TaskLogItem 任务日志项
type TaskLogItem struct {
	N int    `json:"n"` // 行号
	T string `json:"t"` // 日志内容
}

// GetTaskStatusRequest 获取任务状态请求
type GetTaskStatusRequest struct {
	ClusterID int64  `form:"cluster_id" binding:"required" example:"1"`
	NodeName  string `form:"node_name" binding:"required" example:"pve-node1"`
	UPID      string `form:"upid" binding:"required" example:"UPID:pve-node1:00001234:12345678:90ABCDEF:qmclone:root@pam:"`
}

// GetTaskStatusResponse 获取任务状态响应
type GetTaskStatusResponse struct {
	Response
	Data TaskStatusItem `json:"data"`
}

// TaskStatusItem 任务状态项
type TaskStatusItem struct {
	UPID       string                 `json:"upid"`
	Type       string                 `json:"type"`
	ID         string                 `json:"id"`
	User       string                 `json:"user"`
	Status     string                 `json:"status"`
	StartTime  int64                  `json:"starttime"`
	EndTime    int64                  `json:"endtime"`
	Pid        int                    `json:"pid,omitempty"`
	PStart     int64                  `json:"pstart,omitempty"`
	ExitStatus interface{}            `json:"exitstatus,omitempty"` // 可能是string或int
	Extra      map[string]interface{} `json:"extra,omitempty"`
}

// StopTaskRequest 终止任务请求
type StopTaskRequest struct {
	ClusterID int64  `form:"cluster_id" binding:"required" example:"1"`
	NodeName  string `form:"node_name" binding:"required" example:"pve-node1"`
	UPID      string `form:"upid" binding:"required" example:"UPID:pve-node1:00001234:12345678:90ABCDEF:qmclone:root@pam:"`
}

// StopTaskResponse 终止任务响应
type StopTaskResponse struct {
	Response
}
