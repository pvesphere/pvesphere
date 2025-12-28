package v1

import "time"

// PveVM 相关 API 定义

// CreateVMRequest 创建虚拟机请求（从模板克隆）
type CreateVMRequest struct {
	// CreateMode 创建模式：
	// - template：从模板克隆（默认，兼容旧前端不传 create_mode 的情况）
	// - iso：创建空虚拟机并挂载 ISO，从光驱启动安装
	// - empty：创建空虚拟机（不挂载 ISO），后续可通过 /api/v1/vms/config 再挂载 ISO
	CreateMode string `json:"create_mode,omitempty" example:"template"`

	VmName      string `json:"vm_name" binding:"required" example:"vm-001"` // 新虚拟机名称
	ClusterID   int64  `json:"cluster_id" example:"1"`                      // 集群ID（推荐使用，优先级高于 cluster_name）
	ClusterName string `json:"cluster_name,omitempty" example:"my-cluster"` // 集群名称（可选，向后兼容，如果提供了 cluster_id 则忽略此字段）
	NodeID      int64  `json:"node_id" example:"1"`                         // 节点ID（推荐使用，优先级高于 node_name）
	NodeName    string `json:"node_name,omitempty" example:"pve-node-1"`    // 目标节点名称（可选，向后兼容，如果提供了 node_id 则忽略此字段）
	VMID        uint32 `json:"vmid,omitempty" example:"100"`                // 新虚拟机的 VM ID（可选，不传则自动生成8位数）
	TemplateID  int64  `json:"template_id,omitempty" example:"1"`           // 模板ID（create_mode=template 时必填）
	CPUNum      *int   `json:"cpu_num,omitempty" example:"2"`               // CPU核心数（可选，不设置则使用模板配置）
	MemorySize  *int   `json:"memory_size,omitempty" example:"4096"`        // 内存大小MB（可选，不设置则使用模板配置）
	Storage     string `json:"storage,omitempty" example:"local"`           // 存储名称（可选）
	StorageCfg  string `json:"storage_cfg,omitempty" example:"{}"`          // 存储配置（可选）

	// ISO / 空机创建相关字段（create_mode=iso/empty）
	// ISO 卷标识（建议直接传 volume id：local-dir:iso/xxx.iso；也兼容以 / 开头：/local-dir:iso/xxx.iso）
	ISOVolume string `json:"iso_volume,omitempty" example:"local-dir:iso/ubuntu-22.04-server-amd64.iso"`
	// 系统盘大小（GB），create_mode=iso/empty 时建议提供，不传则默认 32
	DiskSizeGB *int `json:"disk_size_gb,omitempty" example:"32"`
	// 磁盘格式（仅 dir 等文件存储需要，local-lvm 可忽略），默认 qcow2
	DiskFormat string `json:"disk_format,omitempty" example:"qcow2"`
	// 网桥名称，默认 vmbr0
	Bridge string `json:"bridge,omitempty" example:"vmbr0"`
	// 网卡模型，默认 virtio
	NetModel string `json:"net_model,omitempty" example:"virtio"`
	// 操作系统类型（Proxmox ostype），默认 l26
	OSType string `json:"os_type,omitempty" example:"l26"`

	AppId       string `json:"app_id,omitempty" example:"app-001"`       // 应用ID（可选）
	VmUser      string `json:"vm_user,omitempty" example:"root"`         // 虚拟机用户名（可选）
	VmPassword  string `json:"vm_password,omitempty" example:"password"` // 虚拟机密码（可选）
	Description string `json:"description,omitempty" example:"虚拟机描述"`    // 描述（可选）
	FullClone   *int   `json:"full_clone,omitempty" example:"1"`         // 是否完整克隆（1=完整克隆，0=链接克隆，默认1）
	IPAddressID *int64 `json:"ip_address_id,omitempty" example:"1"`      // IP地址ID（从vm_ipaddress表，可选）
}

// UpdateVMRequest 更新虚拟机请求
type UpdateVMRequest struct {
	VmName       *string `json:"vm_name,omitempty"`
	CPUNum       *int    `json:"cpu_num,omitempty"`
	MemorySize   *int    `json:"memory_size,omitempty"`
	Storage      *string `json:"storage,omitempty"`
	StorageCfg   *string `json:"storage_cfg,omitempty"`
	AppId        *string `json:"app_id,omitempty"`
	Status       *string `json:"status,omitempty"`
	TemplateName *string `json:"template_name,omitempty"`
	VmUser       *string `json:"vm_user,omitempty"`
	VmPassword   *string `json:"vm_password,omitempty"`
	NodeIP       *string `json:"node_ip,omitempty"`
	Description  *string `json:"description,omitempty"`
}

// ListVMRequest 列表查询请求
type ListVMRequest struct {
	Page        int    `form:"page" example:"1"`
	PageSize    int    `form:"page_size" binding:"omitempty,max=100" example:"10"`
	ClusterID   int64  `form:"cluster_id" example:"1"`            // 集群ID（推荐使用，优先级高于 cluster_name）
	ClusterName string `form:"cluster_name" example:"my-cluster"` // 集群名称（可选，向后兼容）
	NodeID      int64  `form:"node_id" example:"1"`               // 节点ID（推荐使用，优先级高于 node_name）
	NodeName    string `form:"node_name" example:"pve-node-1"`    // 节点名称（可选，向后兼容）
	TemplateID  int64  `form:"template_id" example:"1"`           // 模板ID（可选）
	Status      string `form:"status" example:"running"`
	AppId       string `form:"app_id" example:"app-001"`
}

// ListVMResponse 列表查询响应
type ListVMResponse struct {
	Response
	Data ListVMResponseData
}

type ListVMResponseData struct {
	Total int64    `json:"total"`
	List  []VMItem `json:"list"`
}

type VMItem struct {
	Id           int64  `json:"id"`
	VmName       string `json:"vm_name"`
	ClusterID    int64  `json:"cluster_id"`    // 集群ID
	ClusterName  string `json:"cluster_name"`  // 集群名称（冗余字段，用于显示）
	NodeID       int64  `json:"node_id"`       // 节点ID
	NodeName     string `json:"node_name"`     // 节点名称（冗余字段，用于显示）
	TemplateID   int64  `json:"template_id"`   // 模板ID
	TemplateName string `json:"template_name"` // 模板名称（冗余字段，用于显示）
	IsTemplate   int8   `json:"is_template"`   // 是否为模板：0=否, 1=是
	VMID         uint32 `json:"vmid"`
	CPUNum       int    `json:"cpu_num"`
	MemorySize   int    `json:"memory_size"`
	Status       string `json:"status"`
	AppId        string `json:"app_id"`
	NodeIP       string `json:"node_ip"`
}

// GetVMResponse 详情查询响应
type GetVMResponse struct {
	Response
	Data VMDetail
}

type VMDetail struct {
	Id           int64     `json:"id"`
	VmName       string    `json:"vm_name"`
	ClusterID    int64     `json:"cluster_id"`    // 集群ID
	ClusterName  string    `json:"cluster_name"`  // 集群名称（冗余字段，用于显示）
	NodeID       int64     `json:"node_id"`       // 节点ID
	NodeName     string    `json:"node_name"`     // 节点名称（冗余字段，用于显示）
	TemplateID   int64     `json:"template_id"`   // 模板ID
	TemplateName string    `json:"template_name"` // 模板名称（冗余字段，用于显示）
	IsTemplate   int8      `json:"is_template"`   // 是否为模板：0=否, 1=是
	VMID         uint32    `json:"vmid"`
	CPUNum       int       `json:"cpu_num"`
	MemorySize   int       `json:"memory_size"`
	Storage      string    `json:"storage"`
	StorageCfg   string    `json:"storage_cfg"`
	AppId        string    `json:"app_id"`
	Status       string    `json:"status"`
	VmUser       string    `json:"vm_user"`
	NodeIP       string    `json:"node_ip"`
	Description  string    `json:"description"`
	CreateTime   time.Time `json:"create_time"` // 创建时间
	UpdateTime   time.Time `json:"update_time"` // 更新时间
	Creator      string    `json:"creator"`     // 创建者
	Modifier     string    `json:"modifier"`    // 修改者
}

// ========================
// 虚拟机备份相关 API
// ========================

// CreateBackupRequest 创建备份请求
// 参考: https://pve.proxmox.com/pve-docs/api-viewer/#/nodes/{node}/vzdump
type CreateBackupRequest struct {
	VMID            uint32 `json:"vmid" binding:"required" example:"100"`              // 虚拟机ID（必填）
	Storage         string `json:"storage,omitempty" example:"local"`                 // 存储名称（可选，默认使用配置的存储）
	Compress        string `json:"compress,omitempty" example:"zstd"`                  // 压缩格式：zstd, lzo, gzip（可选，支持 zst 作为 zstd 的别名）
	Mode            string `json:"mode,omitempty" example:"snapshot"`                 // 备份模式：snapshot, suspend, stop（可选，默认 snapshot）
	Remove          *int   `json:"remove,omitempty" example:"0"`                        // 是否删除旧备份：0=否, 1=是（可选）
	MailTo          string `json:"mailto,omitempty" example:"admin@example.com"`       // 备份完成后发送邮件到（可选）
	MailNotification string `json:"mailnotification,omitempty" example:"always"`      // 邮件通知类型：always, failure（可选）
	NotesTemplate   string `json:"notes_template,omitempty" example:"Backup of {name}"` // 备份注释模板（可选）
	Exclude         string `json:"exclude,omitempty" example:"/mnt/data"`             // 排除的挂载点（可选，逗号分隔）
	Quiesce         *int   `json:"quiesce,omitempty" example:"0"`                      // 是否使用 quiesce：0=否, 1=是（可选，需要 qemu-guest-agent）
	MaxFiles        *int   `json:"maxfiles,omitempty" example:"5"`                     // 保留的最大备份文件数（可选）
	Bwlimit         *int   `json:"bwlimit,omitempty" example:"10"`                    // 带宽限制（MB/s）（可选）
	Ionice          *int   `json:"ionice,omitempty" example:"7"`                      // IO 优先级（可选）
	Stop            *int   `json:"stop,omitempty" example:"0"`                        // 是否停止虚拟机：0=否, 1=是（可选）
	StopWait        *int   `json:"stopwait,omitempty" example:"300"`                  // 停止等待时间（秒）（可选）
	DumpDir         string `json:"dumpdir,omitempty" example:"/var/lib/vz/dump"`      // 备份目录（可选，覆盖存储配置）
	Zstd            *int   `json:"zstd,omitempty" example:"1"`                         // zstd 压缩级别 1-22（可选，仅当 compress=zst 时有效）
}

// CreateBackupResponse 创建备份响应
type CreateBackupResponse struct {
	Response
	Data CreateBackupResponseData `json:"data"`
}

type CreateBackupResponseData struct {
	UPID    string `json:"upid"`    // 任务ID
	VMID    uint32 `json:"vmid"`    // 虚拟机ID
	NodeID  int64  `json:"node_id"` // 节点ID
	NodeName string `json:"node_name"` // 节点名称
}

// DeleteBackupRequest 删除备份请求
// 参考: https://pve.proxmox.com/pve-docs/api-viewer/#/nodes/{node}/storage/{storage}/content/{volume}
type DeleteBackupRequest struct {
	NodeID    int64  `json:"node_id" binding:"required" example:"1"`              // 节点ID（必填）
	StorageID int64  `json:"storage_id" binding:"required" example:"6"`          // 存储ID（必填）
	Volume    string `json:"volume" binding:"required" example:"local:backup/vzdump-qemu-100-2024_01_01-00_00_00.vma.zst"` // 卷标识（volid），格式：storage:backup/filename（必填）
	Delay     *int   `json:"delay,omitempty" example:"5"`                        // 延迟删除时间（秒），可选，默认立即删除
}

// DeleteBackupResponse 删除备份响应
type DeleteBackupResponse struct {
	Response
}

// GetVMCloudInitRequest 获取 CloudInit 配置请求
type GetVMCloudInitRequest struct {
	VMID   uint32 `form:"vm_id" binding:"required" example:"100"`   // 虚拟机ID（必填）
	NodeID int64  `form:"node_id" binding:"required" example:"1"`   // 节点ID（必填）
}

// GetVMCloudInitResponse 获取 CloudInit 配置响应
type GetVMCloudInitResponse struct {
	Response
	Data map[string]interface{} `json:"data"` // CloudInit 配置（包含当前和待处理值）
}

// UpdateVMCloudInitRequest 更新 CloudInit 配置请求
// 参考: https://pve.proxmox.com/pve-docs/api-viewer/#/nodes/{node}/qemu/{vmid}/cloudinit
// 所有字段都是可选的，根据需要传递相应的参数
type UpdateVMCloudInitRequest struct {
	VMID      uint32  `json:"vm_id" binding:"required" example:"100"`       // 虚拟机ID（必填）
	NodeID    int64   `json:"node_id" binding:"required" example:"1"`       // 节点ID（必填）
	Cipassword *string `json:"cipassword,omitempty" example:"password123"`  // CloudInit 密码
	CIuser    *string `json:"ciuser,omitempty" example:"ubuntu"`            // CloudInit 用户名
	Citype    *string `json:"citype,omitempty" example:"nocloud"`           // CloudInit 类型：nocloud, configdrive2, opennebula
	Nameserver *string `json:"nameserver,omitempty" example:"8.8.8.8"`      // DNS 服务器
	Searchdomain *string `json:"searchdomain,omitempty" example:"example.com"` // 搜索域
	SSHkeys   *string `json:"sshkeys,omitempty" example:"ssh-rsa AAAAB3..."`   // SSH 公钥（多行用 \n 分隔）
	IPconfig0 *string `json:"ipconfig0,omitempty" example:"ip=192.168.1.100/24,gw=192.168.1.1"` // IP 配置（格式：ip=xxx,gw=xxx）
	IPconfig1 *string `json:"ipconfig1,omitempty"`                           // 第二个网络接口的 IP 配置
	IPconfig2 *string `json:"ipconfig2,omitempty"`                           // 第三个网络接口的 IP 配置
	// 更多 IP 配置字段可以根据需要添加
}

// UpdateVMCloudInitResponse 更新 CloudInit 配置响应
type UpdateVMCloudInitResponse struct {
	Response
}
