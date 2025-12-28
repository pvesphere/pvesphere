package service

import (
	"context"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	mrand "math/rand"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	v1 "pvesphere/api/v1"
	"pvesphere/internal/model"
	"pvesphere/internal/repository"
	"pvesphere/pkg/log"
	"pvesphere/pkg/proxmox"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type PveVMService interface {
	CreateVM(ctx context.Context, req *v1.CreateVMRequest) error
	CreateVMInProxmox(ctx context.Context, req *v1.CreateVMRequest) error
	UpdateVM(ctx context.Context, id int64, req *v1.UpdateVMRequest) error
	DeleteVM(ctx context.Context, id int64) error
	GetVM(ctx context.Context, id int64) (*v1.VMDetail, error)
	ListVMs(ctx context.Context, req *v1.ListVMRequest) (*v1.ListVMResponseData, error)
	StartVM(ctx context.Context, id int64) error
	StopVM(ctx context.Context, id int64) error
	GetVMCurrentConfig(ctx context.Context, vmID int64) (map[string]interface{}, error)
	GetVMPendingConfig(ctx context.Context, vmID int64) ([]map[string]interface{}, error)
	UpdateVMConfig(ctx context.Context, req *v1.UpdateVMConfigRequest) error
	GetVMStatus(ctx context.Context, vmID int64) (map[string]interface{}, error)
	GetVMConsole(ctx context.Context, req *v1.GetVMConsoleRequest) (map[string]interface{}, error)
	DialVMConsoleWebsocket(ctx context.Context, token string) (*websocket.Conn, error)
	GetVMRRDData(ctx context.Context, vmID int64, timeframe, cf string) ([]map[string]interface{}, error)
	MigrateVM(ctx context.Context, req *v1.MigrateVMRequest) (string, error)
	RemoteMigrateVM(ctx context.Context, req *v1.RemoteMigrateVMRequest) (string, error)
	CreateBackup(ctx context.Context, req *v1.CreateBackupRequest) (*v1.CreateBackupResponseData, error)
	DeleteBackup(ctx context.Context, req *v1.DeleteBackupRequest) error
	GetVMCloudInit(ctx context.Context, req *v1.GetVMCloudInitRequest) (map[string]interface{}, error)
	UpdateVMCloudInit(ctx context.Context, req *v1.UpdateVMCloudInitRequest) error
}

func NewPveVMService(
	service *Service,
	vmRepo repository.PveVMRepository,
	templateRepo repository.VmTemplateRepository,
	templateInstanceRepo repository.TemplateInstanceRepository,
	storageRepo repository.PveStorageRepository,
	ipRepo repository.VMIPAddressRepository,
	clusterRepo repository.PveClusterRepository,
	nodeRepo repository.PveNodeRepository,
	logger *log.Logger,
) PveVMService {
	return &pveVMService{
		vmRepo:               vmRepo,
		templateRepo:         templateRepo,
		templateInstanceRepo: templateInstanceRepo,
		storageRepo:          storageRepo,
		ipRepo:               ipRepo,
		clusterRepo:          clusterRepo,
		nodeRepo:             nodeRepo,
		Service:              service,
		logger:               logger,
	}
}

type pveVMService struct {
	vmRepo               repository.PveVMRepository
	templateRepo         repository.VmTemplateRepository
	templateInstanceRepo repository.TemplateInstanceRepository
	storageRepo          repository.PveStorageRepository
	ipRepo               repository.VMIPAddressRepository
	clusterRepo          repository.PveClusterRepository
	nodeRepo             repository.PveNodeRepository
	*Service
	logger *log.Logger

	consoleSessions sync.Map // token -> vmConsoleSession
}

type vmConsoleSession struct {
	VMID      int64
	Port      int
	Ticket    string
	ExpiresAt time.Time
}

func newConsoleToken() (string, error) {
	b := make([]byte, 24)
	if _, err := crand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// generateProxmoxVMID 根据传入的 ID 或时间纳秒 + 随机数生成 8 位数 VM ID。
// - 如果 providedID 不为 0，则直接返回 providedID
// - 如果为 0，则自动生成一个 8 位的随机数字（范围 10000000 ~ 99999999）
func generateProxmoxVMID(providedID uint32) uint32 {
	if providedID != 0 {
		return providedID
	}

	// 以当前时间纳秒作为随机种子，叠加一次随机数，生成 8 位数字
	now := time.Now().UnixNano()
	r := mrand.New(mrand.NewSource(now))
	n := now + int64(r.Intn(1_000_000))

	// 取模 1e8，保证不超过 8 位；再保证至少是 8 位（>= 10^7）
	v := n % 100_000_000
	if v < 10_000_000 {
		v += 10_000_000
	}
	return uint32(v)
}

// CreateVM 仅创建数据库记录，用于手动同步或导入场景
func (s *pveVMService) CreateVM(ctx context.Context, req *v1.CreateVMRequest) error {
	// 0. 如果未显式传入 VMID，则自动生成一个 8 位数的 VM ID
	vmID := generateProxmoxVMID(req.VMID)
	req.VMID = vmID

	// 1. 获取模板信息
	template, err := s.templateRepo.GetByID(ctx, req.TemplateID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get template", zap.Error(err), zap.Int64("template_id", req.TemplateID))
		return v1.ErrInternalServerError
	}
	if template == nil {
		return fmt.Errorf("模板 %d 不存在", req.TemplateID)
	}

	// 2. 获取集群信息（优先使用 ID，如果没有则使用名称）
	var cluster *model.PveCluster
	if req.ClusterID > 0 {
		cluster, err = s.clusterRepo.GetByID(ctx, req.ClusterID)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to get cluster by id", zap.Error(err), zap.Int64("cluster_id", req.ClusterID))
			return v1.ErrInternalServerError
		}
	} else if req.ClusterName != "" {
		cluster, err = s.clusterRepo.GetByClusterName(ctx, req.ClusterName)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to get cluster by name", zap.Error(err))
			return v1.ErrInternalServerError
		}
	} else {
		return fmt.Errorf("必须提供 cluster_id 或 cluster_name")
	}

	if cluster == nil {
		if req.ClusterID > 0 {
			return fmt.Errorf("集群 ID %d 不存在", req.ClusterID)
		}
		return fmt.Errorf("集群 %s 不存在", req.ClusterName)
	}

	// 3. 获取节点信息（优先使用 ID，如果没有则使用名称）
	var node *model.PveNode
	if req.NodeID > 0 {
		node, err = s.nodeRepo.GetByID(ctx, req.NodeID)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to get node by id", zap.Error(err), zap.Int64("node_id", req.NodeID))
			return v1.ErrInternalServerError
		}
	} else if req.NodeName != "" {
		node, err = s.nodeRepo.GetByNodeName(ctx, req.NodeName, cluster.Id)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to get node by name", zap.Error(err))
			return v1.ErrInternalServerError
		}
	} else {
		return fmt.Errorf("必须提供 node_id 或 node_name")
	}

	if node == nil {
		if req.NodeID > 0 {
			return fmt.Errorf("节点 ID %d 不存在", req.NodeID)
		}
		return fmt.Errorf("节点 %s 在集群 %s 中不存在", req.NodeName, cluster.ClusterName)
	}

	// 4. 检查新虚拟机是否已存在（使用 NodeID）
	existing, err := s.vmRepo.GetByVMID(ctx, vmID, node.Id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to check vm", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if existing != nil {
		s.logger.WithContext(ctx).Warn("vm already exists", zap.Uint32("vmid", vmID), zap.Int64("node_id", node.Id))
		return fmt.Errorf("虚拟机 %d 在节点 %s 上已存在", vmID, node.NodeName)
	}

	// 5. 查找模板实例（优先查找目标节点上的实例，如果没有则查找主实例或其他可用实例）
	var templateInstance *model.TemplateInstance

	// 优先查找目标节点上的模板实例
	instance, err := s.templateInstanceRepo.GetByTemplateAndNode(ctx, template.Id, node.Id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get template instance", zap.Error(err))
		return v1.ErrInternalServerError
	}

	if instance != nil && instance.Status == model.TemplateInstanceStatusAvailable {
		// 目标节点上有可用的模板实例
		templateInstance = instance
	} else {
		// 目标节点上没有可用实例，查找主实例或其他可用实例
		primaryInstance, err := s.templateInstanceRepo.GetPrimaryInstance(ctx, template.Id)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to get primary template instance", zap.Error(err))
			return v1.ErrInternalServerError
		}
		if primaryInstance != nil && primaryInstance.Status == model.TemplateInstanceStatusAvailable {
			templateInstance = primaryInstance
		} else {
			// 查找任何可用的实例
			allInstances, err := s.templateInstanceRepo.ListByTemplateID(ctx, template.Id)
			if err != nil {
				s.logger.WithContext(ctx).Error("failed to list template instances", zap.Error(err))
				return v1.ErrInternalServerError
			}
			for _, inst := range allInstances {
				if inst.Status == model.TemplateInstanceStatusAvailable {
					templateInstance = inst
					break
				}
			}
		}
	}

	if templateInstance == nil {
		return fmt.Errorf("模板 ID %d 没有可用的模板实例", template.Id)
	}

	// 6. 创建数据库记录（仅数据库操作，不调用 Proxmox API）
	vm := &model.PveVM{
		VmName:     req.VmName,
		ClusterID:  cluster.Id,
		NodeID:     node.Id,
		TemplateID: template.Id,
		VMID:       vmID,
		Status:     "stopped", // 默认停止状态
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}

	// 如果提供了配置，则使用提供的配置，否则使用默认值
	if req.CPUNum != nil {
		vm.CPUNum = *req.CPUNum
	} else {
		vm.CPUNum = 2 // 默认 2 核
	}
	if req.MemorySize != nil {
		vm.MemorySize = *req.MemorySize
	} else {
		vm.MemorySize = 4096 // 默认 4GB
	}
	if req.Storage != "" {
		vm.Storage = req.Storage
	} else {
		vm.Storage = templateInstance.StorageName // 使用模板实例的存储
	}
	if req.StorageCfg != "" {
		vm.StorageCfg = req.StorageCfg
	} else {
		vm.StorageCfg = "{}" // 默认空配置
	}
	if req.AppId != "" {
		vm.AppId = req.AppId
	}
	if req.VmUser != "" {
		vm.VmUser = req.VmUser
	}
	if req.VmPassword != "" {
		vm.VmPassword = req.VmPassword
	}
	if node.IPAddress != "" {
		vm.NodeIP = node.IPAddress
	}
	if req.Description != "" {
		vm.Description = req.Description
	}

	if err := s.vmRepo.Create(ctx, vm); err != nil {
		s.logger.WithContext(ctx).Error("failed to create vm record", zap.Error(err))
		return v1.ErrInternalServerError
	}

	// 7. 如果提供了 IP 地址 ID，创建 IP 地址记录
	if req.IPAddressID != nil {
		ipAddr, err := s.ipRepo.GetByID(ctx, *req.IPAddressID)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to get ip address", zap.Error(err))
			// IP 地址获取失败不影响虚拟机创建，只记录日志
		} else if ipAddr != nil {
			// 更新 IP 地址记录的 VMId 和 ClusterID
			ipAddr.VMId = vm.Id
			ipAddr.ClusterID = cluster.Id
			if err := s.ipRepo.Update(ctx, ipAddr); err != nil {
				s.logger.WithContext(ctx).Error("failed to update ip address", zap.Error(err))
				// IP 地址更新失败不影响虚拟机创建，只记录日志
			}
		}
	}

	return nil
}

// CreateVMInProxmox 完整创建流程：调用 Proxmox API 创建虚拟机 + 自动创建数据库记录
func (s *pveVMService) CreateVMInProxmox(ctx context.Context, req *v1.CreateVMRequest) error {
	// 0. 如果未显式传入 VMID，则自动生成一个 8 位数的 VM ID
	vmID := generateProxmoxVMID(req.VMID)
	req.VMID = vmID

	// 0.1 创建模式（默认 template，兼容旧前端不传 create_mode 的情况）
	createMode := strings.ToLower(strings.TrimSpace(req.CreateMode))
	if createMode == "" {
		createMode = "template"
	}

	// 1. 获取集群信息（优先使用 ID，如果没有则使用名称）
	var cluster *model.PveCluster
	var err error
	if req.ClusterID > 0 {
		cluster, err = s.clusterRepo.GetByID(ctx, req.ClusterID)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to get cluster by id", zap.Error(err), zap.Int64("cluster_id", req.ClusterID))
			return v1.ErrInternalServerError
		}
	} else if req.ClusterName != "" {
		cluster, err = s.clusterRepo.GetByClusterName(ctx, req.ClusterName)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to get cluster by name", zap.Error(err))
			return v1.ErrInternalServerError
		}
	} else {
		return fmt.Errorf("必须提供 cluster_id 或 cluster_name")
	}

	if cluster == nil {
		if req.ClusterID > 0 {
			return fmt.Errorf("集群 ID %d 不存在", req.ClusterID)
		}
		return fmt.Errorf("集群 %s 不存在", req.ClusterName)
	}

	// 检查集群是否可调度
	if cluster.IsSchedulable != 1 {
		return fmt.Errorf("集群 %s 不可调度", cluster.ClusterName)
	}

	// 2. 获取节点信息（优先使用 ID，如果没有则使用名称）
	var node *model.PveNode
	if req.NodeID > 0 {
		node, err = s.nodeRepo.GetByID(ctx, req.NodeID)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to get node by id", zap.Error(err), zap.Int64("node_id", req.NodeID))
			return v1.ErrInternalServerError
		}
	} else if req.NodeName != "" {
		node, err = s.nodeRepo.GetByNodeName(ctx, req.NodeName, cluster.Id)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to get node by name", zap.Error(err))
			return v1.ErrInternalServerError
		}
	} else {
		return fmt.Errorf("必须提供 node_id 或 node_name")
	}

	if node == nil {
		if req.NodeID > 0 {
			return fmt.Errorf("节点 ID %d 不存在", req.NodeID)
		}
		return fmt.Errorf("节点 %s 在集群 %s 中不存在", req.NodeName, cluster.ClusterName)
	}

	// 3. 检查新虚拟机是否已存在（使用 NodeID）
	existing, err := s.vmRepo.GetByVMID(ctx, vmID, node.Id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to check vm", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if existing != nil {
		s.logger.WithContext(ctx).Warn("vm already exists", zap.Uint32("vmid", vmID), zap.Int64("node_id", node.Id))
		return fmt.Errorf("虚拟机 %d 在节点 %s 上已存在", vmID, node.NodeName)
	}

	// 4. 创建 Proxmox 客户端
	proxmoxClient, err := proxmox.NewProxmoxClient(cluster.ApiUrl, cluster.UserId, cluster.UserToken)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to create proxmox client", zap.Error(err))
		return v1.ErrInternalServerError
	}

	switch createMode {
	case "template":
		// 5.template 分支：从模板克隆
		if req.TemplateID <= 0 {
			return fmt.Errorf("create_mode=template 时必须提供 template_id")
		}

		// 5.1 获取模板信息
		template, err := s.templateRepo.GetByID(ctx, req.TemplateID)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to get template", zap.Error(err), zap.Int64("template_id", req.TemplateID))
			return v1.ErrInternalServerError
		}
		if template == nil {
			return fmt.Errorf("模板 %d 不存在", req.TemplateID)
		}

		// 5.2 查找模板实例（优先查找目标节点上的实例，如果没有则查找主实例或其他可用实例）
		var templateInstance *model.TemplateInstance

		// 优先查找目标节点上的模板实例
		instance, err := s.templateInstanceRepo.GetByTemplateAndNode(ctx, template.Id, node.Id)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to get template instance", zap.Error(err))
			return v1.ErrInternalServerError
		}

		if instance != nil && instance.Status == model.TemplateInstanceStatusAvailable && instance.VMID > 0 {
			// 目标节点上有可用的模板实例
			templateInstance = instance
			s.logger.WithContext(ctx).Info("using template instance on target node",
				zap.Int64("template_id", template.Id),
				zap.Int64("node_id", node.Id),
				zap.Uint32("vmid", instance.VMID))
		} else {
			// 目标节点上没有可用实例，查找主实例或其他可用实例
			primaryInstance, err := s.templateInstanceRepo.GetPrimaryInstance(ctx, template.Id)
			if err != nil {
				s.logger.WithContext(ctx).Error("failed to get primary template instance", zap.Error(err))
				return v1.ErrInternalServerError
			}
			if primaryInstance != nil && primaryInstance.Status == model.TemplateInstanceStatusAvailable && primaryInstance.VMID > 0 {
				templateInstance = primaryInstance
				s.logger.WithContext(ctx).Info("using primary template instance",
					zap.Int64("template_id", template.Id),
					zap.Int64("node_id", primaryInstance.NodeID),
					zap.Uint32("vmid", primaryInstance.VMID))
			} else {
				// 查找任何可用的实例
				allInstances, err := s.templateInstanceRepo.ListByTemplateID(ctx, template.Id)
				if err != nil {
					s.logger.WithContext(ctx).Error("failed to list template instances", zap.Error(err))
					return v1.ErrInternalServerError
				}
				for _, inst := range allInstances {
					if inst.Status == model.TemplateInstanceStatusAvailable && inst.VMID > 0 {
						templateInstance = inst
						s.logger.WithContext(ctx).Info("using available template instance",
							zap.Int64("template_id", template.Id),
							zap.Int64("node_id", inst.NodeID),
							zap.Uint32("vmid", inst.VMID))
						break
					}
				}
			}
		}

		if templateInstance == nil || templateInstance.VMID == 0 {
			return fmt.Errorf("模板 ID %d 没有可用的模板实例", template.Id)
		}

		// 获取模板实例所在的节点
		sourceNode, err := s.nodeRepo.GetByID(ctx, templateInstance.NodeID)
		if err != nil || sourceNode == nil {
			return fmt.Errorf("无法获取模板实例的节点信息")
		}
		sourceNodeName := sourceNode.NodeName

		// 5.3 准备克隆请求参数
		fullClone := 1 // 默认完整克隆
		if req.FullClone != nil {
			fullClone = *req.FullClone
		}

		// 如果目标节点和源节点相同，则不设置 target 参数
		var targetNode string
		if node.NodeName != sourceNodeName && node.NodeName != "" {
			targetNode = node.NodeName
		}

		cloneReq := &proxmox.CloneVMRequest{
			NewID:       vmID,
			Name:        req.VmName,
			Target:      targetNode, // 目标节点（如果不同才设置）
			Full:        fullClone,
			Storage:     req.Storage,
			Description: req.Description,
		}

		// 5.4 调用 Proxmox API 克隆虚拟机
		upid, err := proxmoxClient.CloneVM(ctx, sourceNodeName, templateInstance.VMID, cloneReq)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to clone vm", zap.Error(err),
				zap.String("template_node", sourceNodeName),
				zap.Uint32("template_vmid", templateInstance.VMID))
			return fmt.Errorf("克隆虚拟机失败: %v", err)
		}
		s.logger.WithContext(ctx).Info("vm cloned", zap.String("upid", upid), zap.Uint32("vmid", vmID))

		// 5.5 创建数据库记录
		vm := &model.PveVM{
			VmName:     req.VmName,
			ClusterID:  cluster.Id,
			NodeID:     node.Id,
			TemplateID: template.Id,
			VMID:       vmID,
			Status:     "stopped", // 克隆后默认停止状态
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		}

		// 如果提供了配置，则使用提供的配置，否则使用默认值
		if req.CPUNum != nil {
			vm.CPUNum = *req.CPUNum
		} else {
			vm.CPUNum = 2 // 默认 2 核
		}
		if req.MemorySize != nil {
			vm.MemorySize = *req.MemorySize
		} else {
			vm.MemorySize = 4096 // 默认 4GB
		}
		if req.Storage != "" {
			vm.Storage = req.Storage
		} else {
			vm.Storage = templateInstance.StorageName // 使用模板实例的存储
		}
		if req.StorageCfg != "" {
			vm.StorageCfg = req.StorageCfg
		} else {
			vm.StorageCfg = "{}" // 默认空配置
		}
		if req.AppId != "" {
			vm.AppId = req.AppId
		}
		if req.VmUser != "" {
			vm.VmUser = req.VmUser
		}
		if req.VmPassword != "" {
			vm.VmPassword = req.VmPassword
		}
		if node.IPAddress != "" {
			vm.NodeIP = node.IPAddress
		}
		if req.Description != "" {
			vm.Description = req.Description
		}

		if err := s.vmRepo.Create(ctx, vm); err != nil {
			s.logger.WithContext(ctx).Error("failed to create vm record", zap.Error(err))
			// 注意：如果数据库创建失败，可以考虑回滚 Proxmox 的克隆操作
			return v1.ErrInternalServerError
		}

		// 10. 如果提供了 IP 地址 ID，创建 IP 地址记录
		if req.IPAddressID != nil {
			ipAddr, err := s.ipRepo.GetByID(ctx, *req.IPAddressID)
			if err != nil {
				s.logger.WithContext(ctx).Error("failed to get ip address", zap.Error(err))
				// IP 地址获取失败不影响虚拟机创建，只记录日志
			} else if ipAddr != nil {
				// 更新 IP 地址记录的 VMId 和 ClusterID
				ipAddr.VMId = vm.Id
				ipAddr.ClusterID = cluster.Id
				if err := s.ipRepo.Update(ctx, ipAddr); err != nil {
					s.logger.WithContext(ctx).Error("failed to update ip address", zap.Error(err))
					// IP 地址更新失败不影响虚拟机创建，只记录日志
				}
			}
		}

		return nil

	case "iso", "empty":
		// 5.iso/empty 分支：创建空机（iso 会额外挂载 ISO 并从光驱启动）
		if req.Storage == "" {
			return fmt.Errorf("create_mode=%s 时必须提供 storage", createMode)
		}

		// 默认值
		cpu := 2
		if req.CPUNum != nil && *req.CPUNum > 0 {
			cpu = *req.CPUNum
		}
		mem := 2048
		if req.MemorySize != nil && *req.MemorySize > 0 {
			mem = *req.MemorySize
		}
		diskGB := 32
		if req.DiskSizeGB != nil && *req.DiskSizeGB > 0 {
			diskGB = *req.DiskSizeGB
		}
		bridge := "vmbr0"
		if strings.TrimSpace(req.Bridge) != "" {
			bridge = strings.TrimSpace(req.Bridge)
		}
		netModel := "virtio"
		if strings.TrimSpace(req.NetModel) != "" {
			netModel = strings.TrimSpace(req.NetModel)
		}
		ostype := "l26"
		if strings.TrimSpace(req.OSType) != "" {
			ostype = strings.TrimSpace(req.OSType)
		}

		// iso 模式必须提供 iso_volume
		isoVol := strings.TrimSpace(req.ISOVolume)
		if createMode == "iso" {
			if isoVol == "" {
				return fmt.Errorf("create_mode=iso 时必须提供 iso_volume")
			}
			// 兼容前端可能传入以 / 开头的 volume
			isoVol = strings.TrimPrefix(isoVol, "/")
		}

		// 组装 Proxmox 创建参数（form）
		params := url.Values{}
		params.Set("vmid", fmt.Sprintf("%d", vmID))
		params.Set("name", req.VmName)
		params.Set("cores", fmt.Sprintf("%d", cpu))
		params.Set("memory", fmt.Sprintf("%d", mem))
		params.Set("sockets", "1")
		params.Set("ostype", ostype)
		params.Set("scsihw", "virtio-scsi-pci")
		params.Set("agent", "1")

		// 系统盘：scsi0=<storage>:<sizeGB>[,format=xxx]
		disk := fmt.Sprintf("%s:%d", req.Storage, diskGB)
		if strings.TrimSpace(req.DiskFormat) != "" {
			// 注意：某些后端存储（如 lvmthin/zfs）不支持 qcow2，若报错请前端不传 disk_format
			disk += fmt.Sprintf(",format=%s", strings.TrimSpace(req.DiskFormat))
		}
		params.Set("scsi0", disk)

		// 网卡：net0=<model>,bridge=<bridge>
		params.Set("net0", fmt.Sprintf("%s,bridge=%s", netModel, bridge))

		// ISO 挂载与启动顺序
		if createMode == "iso" {
			params.Set("ide2", fmt.Sprintf("%s,media=cdrom", isoVol))
			params.Set("boot", "order=ide2;scsi0;net0")
		} else {
			params.Set("boot", "order=scsi0;net0")
		}

		if strings.TrimSpace(req.Description) != "" {
			params.Set("description", strings.TrimSpace(req.Description))
		}

		upid, err := proxmoxClient.CreateQemuVM(ctx, node.NodeName, params)
		if err != nil {
			s.logger.WithContext(ctx).Error("failed to create qemu vm", zap.Error(err),
				zap.String("node", node.NodeName),
				zap.Uint32("vmid", vmID),
				zap.String("create_mode", createMode))
			return fmt.Errorf("创建虚拟机失败: %v", err)
		}
		s.logger.WithContext(ctx).Info("vm created", zap.String("upid", upid), zap.Uint32("vmid", vmID), zap.String("create_mode", createMode))

		// 记录创建信息到 storage_cfg（若前端未显式传入）
		storageCfg := req.StorageCfg
		if strings.TrimSpace(storageCfg) == "" {
			cfg := map[string]interface{}{
				"create_mode":  createMode,
				"disk_size_gb": diskGB,
				"storage":      req.Storage,
				"bridge":       bridge,
				"net_model":    netModel,
				"os_type":      ostype,
			}
			if createMode == "iso" {
				cfg["iso_volume"] = isoVol
			}
			if b, err := json.Marshal(cfg); err == nil {
				storageCfg = string(b)
			}
		}

		// 创建数据库记录（TemplateID=0 表示非模板克隆创建）
		vm := &model.PveVM{
			VmName:     req.VmName,
			ClusterID:  cluster.Id,
			NodeID:     node.Id,
			TemplateID: 0,
			VMID:       vmID,
			CPUNum:     cpu,
			MemorySize: mem,
			Storage:    req.Storage,
			StorageCfg: storageCfg,
			Status:     "stopped",
			AppId:      req.AppId,
			VmUser:     req.VmUser,
			VmPassword: req.VmPassword,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		}
		if node.IPAddress != "" {
			vm.NodeIP = node.IPAddress
		}
		if req.Description != "" {
			vm.Description = req.Description
		}

		if err := s.vmRepo.Create(ctx, vm); err != nil {
			s.logger.WithContext(ctx).Error("failed to create vm record", zap.Error(err))
			return v1.ErrInternalServerError
		}

		// IP 地址绑定（可选）
		if req.IPAddressID != nil {
			ipAddr, err := s.ipRepo.GetByID(ctx, *req.IPAddressID)
			if err != nil {
				s.logger.WithContext(ctx).Error("failed to get ip address", zap.Error(err))
			} else if ipAddr != nil {
				ipAddr.VMId = vm.Id
				ipAddr.ClusterID = cluster.Id
				if err := s.ipRepo.Update(ctx, ipAddr); err != nil {
					s.logger.WithContext(ctx).Error("failed to update ip address", zap.Error(err))
				}
			}
		}

		return nil

	default:
		s.logger.WithContext(ctx).Warn("invalid create mode", zap.String("create_mode", createMode))
		return v1.ErrInvalidCreateMode //nolint:stylecheck,staticcheck // false-positive in editor diagnostics
	}
}

func (s *pveVMService) UpdateVM(ctx context.Context, id int64, req *v1.UpdateVMRequest) error {
	vm, err := s.vmRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get vm", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if vm == nil {
		return v1.ErrNotFound
	}

	// 更新字段
	if req.VmName != nil {
		vm.VmName = *req.VmName
	}
	if req.CPUNum != nil {
		vm.CPUNum = *req.CPUNum
	}
	if req.MemorySize != nil {
		vm.MemorySize = *req.MemorySize
	}
	if req.Storage != nil {
		vm.Storage = *req.Storage
	}
	if req.StorageCfg != nil {
		vm.StorageCfg = *req.StorageCfg
	}
	if req.AppId != nil {
		vm.AppId = *req.AppId
	}
	if req.Status != nil {
		vm.Status = *req.Status
	}
	if req.TemplateName != nil {
		vm.TemplateName = *req.TemplateName
	}
	if req.VmUser != nil {
		vm.VmUser = *req.VmUser
	}
	if req.VmPassword != nil {
		vm.VmPassword = *req.VmPassword
	}
	if req.NodeIP != nil {
		vm.NodeIP = *req.NodeIP
	}
	if req.Description != nil {
		vm.Description = *req.Description
	}
	vm.UpdateTime = time.Now()

	if err := s.vmRepo.Update(ctx, vm); err != nil {
		s.logger.WithContext(ctx).Error("failed to update vm", zap.Error(err))
		return v1.ErrInternalServerError
	}

	return nil
}

func (s *pveVMService) DeleteVM(ctx context.Context, id int64) error {
	// 1. 获取虚拟机信息
	vm, err := s.vmRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get vm", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if vm == nil {
		return v1.ErrNotFound
	}

	// 2. 基于数据库状态进行前置校验，要求先在上层手动停止
	if vm.Status == "offline" || vm.Status == "orphan" {
		return fmt.Errorf("虚拟机已经销毁，请勿重复销毁")
	}
	if vm.Status != "stopped" {
		return fmt.Errorf("虚拟机未停止，请先停止虚拟机")
	}

	// 3. 获取集群信息（通过 ID）
	if vm.ClusterID <= 0 {
		return fmt.Errorf("虚拟机的集群 ID 无效，无法删除 Proxmox 虚拟机")
	}
	cluster, err := s.clusterRepo.GetByID(ctx, vm.ClusterID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get cluster", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if cluster == nil {
		return fmt.Errorf("集群 ID %d 不存在，无法删除 Proxmox 虚拟机", vm.ClusterID)
	}

	// 4. 获取节点信息（通过 ID）
	if vm.NodeID <= 0 {
		return fmt.Errorf("虚拟机的节点 ID 无效，无法删除 Proxmox 虚拟机")
	}
	node, err := s.nodeRepo.GetByID(ctx, vm.NodeID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if node == nil {
		return fmt.Errorf("节点 ID %d 不存在，无法删除 Proxmox 虚拟机", vm.NodeID)
	}

	// 5. 创建 Proxmox 客户端
	proxmoxClient, err := proxmox.NewProxmoxClient(cluster.ApiUrl, cluster.UserId, cluster.UserToken)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to create proxmox client", zap.Error(err))
		return fmt.Errorf("创建 Proxmox 客户端失败: %v", err)
	}

	// 6. 先获取虚拟机配置，检查虚拟机是否存在
	vmExistsInProxmox := true
	_, err = proxmoxClient.GetVMConfig(ctx, node.NodeName, vm.VMID)
	if err != nil {
		// 检查错误是否是 404/500（虚拟机不存在或已删除）
		errStr := err.Error()
		if strings.Contains(errStr, "status 404") || strings.Contains(errStr, "status 500") {
			// 如果虚拟机已经不存在（404）或返回 500（可能是已删除），直接删除数据库记录
			s.logger.WithContext(ctx).Warn("vm not found in proxmox or already deleted, will delete database record only", zap.Error(err),
				zap.String("node", node.NodeName),
				zap.Uint32("vmid", vm.VMID))
			vmExistsInProxmox = false
		} else {
			s.logger.WithContext(ctx).Error("failed to get vm config from proxmox", zap.Error(err),
				zap.String("node", node.NodeName),
				zap.Uint32("vmid", vm.VMID))
			return fmt.Errorf("从 Proxmox 获取虚拟机配置失败: %v", err)
		}
	}

	var vmStatus string
	if vmExistsInProxmox {
		// 7. 从 Proxmox 获取虚拟机的实际状态，并最多重试 3 次确认为 stopped
		for i := 0; i < 3; i++ {
			statusData, err := proxmoxClient.GetVMStatus(ctx, node.NodeName, vm.VMID)
			if err != nil {
				// 检查错误是否是 404/500（虚拟机不存在或已删除）
				errStr := err.Error()
				if strings.Contains(errStr, "status 404") || strings.Contains(errStr, "status 500") {
					s.logger.WithContext(ctx).Warn("vm not found in proxmox when getting status, will delete database record only", zap.Error(err),
						zap.String("node", node.NodeName),
						zap.Uint32("vmid", vm.VMID))
					vmExistsInProxmox = false
					break
				}
				s.logger.WithContext(ctx).Error("failed to get vm status from proxmox", zap.Error(err),
					zap.String("node", node.NodeName),
					zap.Uint32("vmid", vm.VMID))
				return fmt.Errorf("从 Proxmox 获取虚拟机状态失败: %v", err)
			}

			// 从返回的 map 中提取 status 字段
			if status, ok := statusData["status"].(string); ok {
				vmStatus = status
				s.logger.WithContext(ctx).Info("get proxmox vm status before delete", zap.Uint32("vmid", vm.VMID), zap.String("status", vmStatus), zap.Any("statusData", statusData))
				if vmStatus == "stopped" {
					break
				}
			} else {
				s.logger.WithContext(ctx).Warn("vm status field not found in response", zap.Any("statusData", statusData))
			}

			if i < 2 {
				time.Sleep(2 * time.Second)
			}
		}

		if vmExistsInProxmox && vmStatus != "stopped" {
			return fmt.Errorf("proxmox 虚拟机当前状态为 %s，无法执行销毁操作，请先在 PVE 中停止虚拟机", vmStatus)
		}

		// 8. 调用 Proxmox API 删除虚拟机（purge=true 表示完全删除，包括磁盘）
		if vmExistsInProxmox {
			s.logger.WithContext(ctx).Info("deleting vm from proxmox", zap.Uint32("vmid", vm.VMID), zap.String("node", node.NodeName))
			if err := proxmoxClient.DeleteVM(ctx, node.NodeName, vm.VMID, true); err != nil {
				s.logger.WithContext(ctx).Error("failed to delete vm from proxmox", zap.Error(err),
					zap.String("node", node.NodeName),
					zap.Uint32("vmid", vm.VMID),
					zap.String("vm_name", vm.VmName),
					zap.String("vm_status", vmStatus))
				return fmt.Errorf("从 Proxmox 删除虚拟机失败: %v", err)
			}
			s.logger.WithContext(ctx).Info("vm deleted from proxmox", zap.Uint32("vmid", vm.VMID))
		}
	}

	// 删除 IP 地址记录
	if err := s.ipRepo.DeleteByVMID(ctx, id); err != nil {
		s.logger.WithContext(ctx).Error("failed to delete ip addresses", zap.Error(err))
		// IP 地址删除失败不影响虚拟机删除，只记录日志
	}

	// 8. 删除数据库记录
	if err := s.vmRepo.Delete(ctx, id); err != nil {
		s.logger.WithContext(ctx).Error("failed to delete vm record", zap.Error(err))
		return v1.ErrInternalServerError
	}

	return nil
}

func (s *pveVMService) GetVM(ctx context.Context, id int64) (*v1.VMDetail, error) {
	vm, err := s.vmRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get vm", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	if vm == nil {
		return nil, v1.ErrNotFound
	}

	detail := &v1.VMDetail{
		Id:          vm.Id,
		VmName:      vm.VmName,
		ClusterID:   vm.ClusterID,
		NodeID:      vm.NodeID,
		TemplateID:  vm.TemplateID,
		IsTemplate:  vm.IsTemplate,
		VMID:        vm.VMID,
		CPUNum:      vm.CPUNum,
		MemorySize:  vm.MemorySize,
		Storage:     vm.Storage,
		StorageCfg:  vm.StorageCfg,
		AppId:       vm.AppId,
		Status:      vm.Status,
		VmUser:      vm.VmUser,
		NodeIP:      vm.NodeIP,
		Description: vm.Description,
		CreateTime:  vm.CreateTime,
		UpdateTime:  vm.UpdateTime,
		Creator:     vm.Creator,
		Modifier:    vm.Modifier,
	}

	// 填充名称字段
	if vm.ClusterID > 0 {
		cluster, _ := s.clusterRepo.GetByID(ctx, vm.ClusterID)
		if cluster != nil {
			detail.ClusterName = cluster.ClusterName
		}
	}
	if vm.NodeID > 0 {
		node, _ := s.nodeRepo.GetByID(ctx, vm.NodeID)
		if node != nil {
			detail.NodeName = node.NodeName
		}
	}
	if vm.TemplateID > 0 {
		template, _ := s.templateRepo.GetByID(ctx, vm.TemplateID)
		if template != nil {
			detail.TemplateName = template.TemplateName
		}
	}

	return detail, nil
}

func (s *pveVMService) ListVMs(ctx context.Context, req *v1.ListVMRequest) (*v1.ListVMResponseData, error) {
	vms, total, err := s.vmRepo.ListWithPagination(ctx, req.Page, req.PageSize, req.ClusterID, req.ClusterName, req.NodeID, req.NodeName, req.TemplateID, req.Status, req.AppId)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to list vms", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}

	// 提取所有需要查询的 ID 用于批量填充名称
	clusterIDs := make([]int64, 0)
	nodeIDs := make([]int64, 0)
	templateIDs := make([]int64, 0)

	for _, vm := range vms {
		if vm.ClusterID > 0 {
			clusterIDs = append(clusterIDs, vm.ClusterID)
		}
		if vm.NodeID > 0 {
			nodeIDs = append(nodeIDs, vm.NodeID)
		}
		if vm.TemplateID > 0 {
			templateIDs = append(templateIDs, vm.TemplateID)
		}
	}

	// 批量查询关联数据
	clusterMap, _ := s.clusterRepo.GetByIDs(ctx, clusterIDs)
	nodeMap, _ := s.nodeRepo.GetByIDs(ctx, nodeIDs)
	templateMap, _ := s.templateRepo.GetByIDs(ctx, templateIDs)

	items := make([]v1.VMItem, 0, len(vms))
	for _, vm := range vms {
		item := v1.VMItem{
			Id:         vm.Id,
			VmName:     vm.VmName,
			ClusterID:  vm.ClusterID,
			NodeID:     vm.NodeID,
			TemplateID: vm.TemplateID,
			IsTemplate: vm.IsTemplate,
			VMID:       vm.VMID,
			CPUNum:     vm.CPUNum,
			MemorySize: vm.MemorySize,
			Status:     vm.Status,
			AppId:      vm.AppId,
			NodeIP:     vm.NodeIP,
		}

		// 从 map 中填充名称
		if cluster, ok := clusterMap[vm.ClusterID]; ok {
			item.ClusterName = cluster.ClusterName
		}
		if node, ok := nodeMap[vm.NodeID]; ok {
			item.NodeName = node.NodeName
		}
		if template, ok := templateMap[vm.TemplateID]; ok {
			item.TemplateName = template.TemplateName
		}

		items = append(items, item)
	}

	return &v1.ListVMResponseData{
		Total: total,
		List:  items,
	}, nil
}

func (s *pveVMService) StartVM(ctx context.Context, id int64) error {
	// 1. 获取虚拟机信息
	vm, err := s.vmRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get vm", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if vm == nil {
		return v1.ErrNotFound
	}

	// 2. 检查虚拟机当前状态
	if vm.Status == "running" {
		return fmt.Errorf("虚拟机已在运行中，无需启动")
	}

	// 3. 获取集群信息（通过 ID）
	if vm.ClusterID <= 0 {
		return fmt.Errorf("虚拟机的集群 ID 无效")
	}
	cluster, err := s.clusterRepo.GetByID(ctx, vm.ClusterID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get cluster", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if cluster == nil {
		return fmt.Errorf("集群 ID %d 不存在", vm.ClusterID)
	}

	// 4. 获取节点信息（通过 ID）
	if vm.NodeID <= 0 {
		return fmt.Errorf("虚拟机的节点 ID 无效")
	}
	node, err := s.nodeRepo.GetByID(ctx, vm.NodeID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if node == nil {
		return fmt.Errorf("节点 ID %d 不存在", vm.NodeID)
	}

	// 5. 创建 Proxmox 客户端
	proxmoxClient, err := proxmox.NewProxmoxClient(cluster.ApiUrl, cluster.UserId, cluster.UserToken)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to create proxmox client", zap.Error(err))
		return fmt.Errorf("创建 Proxmox 客户端失败: %v", err)
	}

	// 6. 调用 Proxmox API 启动虚拟机
	s.logger.WithContext(ctx).Info("starting vm from proxmox", zap.Uint32("vmid", vm.VMID), zap.String("node", node.NodeName))
	upid, err := proxmoxClient.StartVM(ctx, node.NodeName, vm.VMID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to start vm from proxmox", zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Uint32("vmid", vm.VMID),
			zap.String("vm_name", vm.VmName))
		return fmt.Errorf("从 Proxmox 启动虚拟机失败: %v", err)
	}
	s.logger.WithContext(ctx).Info("vm started from proxmox", zap.Uint32("vmid", vm.VMID), zap.String("upid", upid))

	return nil
}

func (s *pveVMService) StopVM(ctx context.Context, id int64) error {
	// 1. 获取虚拟机信息
	vm, err := s.vmRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get vm", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if vm == nil {
		return v1.ErrNotFound
	}

	// 2. 检查虚拟机当前状态
	if vm.Status == "stopped" {
		return fmt.Errorf("虚拟机已停止，无需关机")
	}

	// 3. 获取集群信息（通过 ID）
	if vm.ClusterID <= 0 {
		return fmt.Errorf("虚拟机的集群 ID 无效")
	}
	cluster, err := s.clusterRepo.GetByID(ctx, vm.ClusterID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get cluster", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if cluster == nil {
		return fmt.Errorf("集群 ID %d 不存在", vm.ClusterID)
	}

	// 4. 获取节点信息（通过 ID）
	if vm.NodeID <= 0 {
		return fmt.Errorf("虚拟机的节点 ID 无效")
	}
	node, err := s.nodeRepo.GetByID(ctx, vm.NodeID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if node == nil {
		return fmt.Errorf("节点 ID %d 不存在", vm.NodeID)
	}

	// 5. 创建 Proxmox 客户端
	proxmoxClient, err := proxmox.NewProxmoxClient(cluster.ApiUrl, cluster.UserId, cluster.UserToken)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to create proxmox client", zap.Error(err))
		return fmt.Errorf("创建 Proxmox 客户端失败: %v", err)
	}

	// 6. 调用 Proxmox API 停止虚拟机
	s.logger.WithContext(ctx).Info("stopping vm from proxmox", zap.Uint32("vmid", vm.VMID), zap.String("node", node.NodeName))
	upid, err := proxmoxClient.StopVM(ctx, node.NodeName, vm.VMID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to stop vm from proxmox", zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Uint32("vmid", vm.VMID),
			zap.String("vm_name", vm.VmName))
		return fmt.Errorf("从 Proxmox 停止虚拟机失败: %v", err)
	}
	s.logger.WithContext(ctx).Info("vm stopped from proxmox", zap.Uint32("vmid", vm.VMID), zap.String("upid", upid))

	return nil
}

// getProxmoxClientForVM 根据虚拟机ID获取ProxmoxClient和节点信息
func (s *pveVMService) getProxmoxClientForVM(ctx context.Context, vmID int64) (*proxmox.ProxmoxClient, *model.PveNode, error) {
	// 1. 获取虚拟机信息
	vm, err := s.vmRepo.GetByID(ctx, vmID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get vm", zap.Error(err))
		return nil, nil, v1.ErrInternalServerError
	}
	if vm == nil {
		return nil, nil, v1.ErrNotFound
	}

	// 2. 获取集群信息
	if vm.ClusterID <= 0 {
		return nil, nil, fmt.Errorf("虚拟机的集群 ID 无效")
	}
	cluster, err := s.clusterRepo.GetByID(ctx, vm.ClusterID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get cluster", zap.Error(err))
		return nil, nil, v1.ErrInternalServerError
	}
	if cluster == nil {
		return nil, nil, fmt.Errorf("集群 ID %d 不存在", vm.ClusterID)
	}

	// 3. 获取节点信息
	if vm.NodeID <= 0 {
		return nil, nil, fmt.Errorf("虚拟机的节点 ID 无效")
	}
	node, err := s.nodeRepo.GetByID(ctx, vm.NodeID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node", zap.Error(err))
		return nil, nil, v1.ErrInternalServerError
	}
	if node == nil {
		return nil, nil, fmt.Errorf("节点 ID %d 不存在", vm.NodeID)
	}

	// 4. 创建 Proxmox 客户端
	proxmoxClient, err := proxmox.NewProxmoxClient(cluster.ApiUrl, cluster.UserId, cluster.UserToken)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to create proxmox client", zap.Error(err))
		return nil, nil, v1.ErrInternalServerError
	}

	return proxmoxClient, node, nil
}

// getProxmoxClientForNode 根据节点ID获取ProxmoxClient和节点信息
func (s *pveVMService) getProxmoxClientForNode(ctx context.Context, nodeID int64) (*proxmox.ProxmoxClient, error) {
	// 1. 获取节点信息
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	if node == nil {
		return nil, fmt.Errorf("节点 ID %d 不存在", nodeID)
	}

	// 2. 获取集群信息
	if node.ClusterID <= 0 {
		return nil, fmt.Errorf("节点的集群 ID 无效")
	}
	cluster, err := s.clusterRepo.GetByID(ctx, node.ClusterID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get cluster", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	if cluster == nil {
		return nil, fmt.Errorf("集群 ID %d 不存在", node.ClusterID)
	}

	// 3. 创建 Proxmox 客户端
	proxmoxClient, err := proxmox.NewProxmoxClient(cluster.ApiUrl, cluster.UserId, cluster.UserToken)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to create proxmox client", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}

	return proxmoxClient, nil
}

func (s *pveVMService) GetVMCurrentConfig(ctx context.Context, vmID int64) (map[string]interface{}, error) {
	client, node, err := s.getProxmoxClientForVM(ctx, vmID)
	if err != nil {
		return nil, err
	}

	vm, err := s.vmRepo.GetByID(ctx, vmID)
	if err != nil {
		return nil, v1.ErrInternalServerError
	}

	config, err := client.GetVMCurrentConfig(ctx, node.NodeName, vm.VMID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get vm current config", zap.Error(err),
			zap.String("node", node.NodeName), zap.Uint32("vmid", vm.VMID))
		return nil, v1.ErrInternalServerError
	}

	return config, nil
}

func (s *pveVMService) GetVMPendingConfig(ctx context.Context, vmID int64) ([]map[string]interface{}, error) {
	client, node, err := s.getProxmoxClientForVM(ctx, vmID)
	if err != nil {
		return nil, err
	}

	vm, err := s.vmRepo.GetByID(ctx, vmID)
	if err != nil {
		return nil, v1.ErrInternalServerError
	}

	config, err := client.GetVMPendingConfig(ctx, node.NodeName, vm.VMID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get vm pending config", zap.Error(err),
			zap.String("node", node.NodeName), zap.Uint32("vmid", vm.VMID))
		return nil, v1.ErrInternalServerError
	}

	return config, nil
}

func (s *pveVMService) UpdateVMConfig(ctx context.Context, req *v1.UpdateVMConfigRequest) error {
	client, node, err := s.getProxmoxClientForVM(ctx, req.VMID)
	if err != nil {
		return err
	}

	vm, err := s.vmRepo.GetByID(ctx, req.VMID)
	if err != nil {
		return v1.ErrInternalServerError
	}

	if err := client.UpdateVMConfig(ctx, node.NodeName, vm.VMID, req.Config); err != nil {
		s.logger.WithContext(ctx).Error("failed to update vm config", zap.Error(err),
			zap.String("node", node.NodeName), zap.Uint32("vmid", vm.VMID))
		return v1.ErrInternalServerError
	}

	s.logger.WithContext(ctx).Info("vm config updated", zap.Uint32("vmid", vm.VMID), zap.String("node", node.NodeName))
	return nil
}

func (s *pveVMService) GetVMStatus(ctx context.Context, vmID int64) (map[string]interface{}, error) {
	client, node, err := s.getProxmoxClientForVM(ctx, vmID)
	if err != nil {
		return nil, err
	}

	vm, err := s.vmRepo.GetByID(ctx, vmID)
	if err != nil {
		return nil, v1.ErrInternalServerError
	}

	status, err := client.GetVMStatus(ctx, node.NodeName, vm.VMID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get vm status", zap.Error(err),
			zap.String("node", node.NodeName), zap.Uint32("vmid", vm.VMID))
		return nil, v1.ErrInternalServerError
	}

	return status, nil
}

func (s *pveVMService) GetVMConsole(ctx context.Context, req *v1.GetVMConsoleRequest) (map[string]interface{}, error) {
	client, node, err := s.getProxmoxClientForVM(ctx, req.VMID)
	if err != nil {
		return nil, err
	}

	vm, err := s.vmRepo.GetByID(ctx, req.VMID)
	if err != nil {
		return nil, v1.ErrInternalServerError
	}
	if vm == nil {
		return nil, v1.ErrNotFound
	}

	// noVNC 需要 vncproxy 的 port/ticket 再去连 vncwebsocket；这里默认强制开启 websocket=1
	// 避免前端未传 websocket 导致返回字段不全（port/ticket 缺失）
	result, err := client.QemuVNCProxy(ctx, node.NodeName, vm.VMID, true, req.GeneratePassword)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get vm vncproxy", zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Uint32("vmid", vm.VMID))
		return nil, v1.ErrInternalServerError
	}

	// vncproxy 返回 data 通常包含 port(int) 和 ticket(string)
	var port int
	switch v := result["port"].(type) {
	case float64:
		port = int(v)
	case int:
		port = v
	case int64:
		port = int(v)
	case json.Number:
		if p, err := v.Int64(); err == nil {
			port = int(p)
		}
	case string:
		if p, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			port = p
		}
	}
	ticket, _ := result["ticket"].(string)
	if port <= 0 || strings.TrimSpace(ticket) == "" {
		s.logger.WithContext(ctx).Warn("vncproxy response missing port/ticket", zap.Any("data", result))
		return nil, v1.ErrInternalServerError
	}

	token, err := newConsoleToken()
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to generate console token", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	exp := time.Now().Add(2 * time.Minute)
	s.consoleSessions.Store(token, vmConsoleSession{
		VMID:      req.VMID,
		Port:      port,
		Ticket:    ticket,
		ExpiresAt: exp,
	})
	result["ws_token"] = token
	result["ws_expires_at"] = exp.Unix()

	return result, nil
}

// DialVMConsoleWebsocket 通过 ws_token 建立到 Proxmox vncwebsocket 的连接（单次使用/短期有效）
func (s *pveVMService) DialVMConsoleWebsocket(ctx context.Context, token string) (*websocket.Conn, error) {
	if strings.TrimSpace(token) == "" {
		return nil, v1.ErrBadRequest
	}

	val, ok := s.consoleSessions.LoadAndDelete(token)
	if !ok {
		return nil, v1.ErrNotFound
	}
	session, ok := val.(vmConsoleSession)
	if !ok {
		return nil, v1.ErrInternalServerError
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, v1.ErrUnauthorized
	}

	client, node, err := s.getProxmoxClientForVM(ctx, session.VMID)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("port", fmt.Sprintf("%d", session.Port))
	params.Set("vncticket", session.Ticket)

	// 获取虚拟机信息以得到实际的 VMID
	vm, err := s.vmRepo.GetByID(ctx, session.VMID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get vm for websocket", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	if vm == nil {
		return nil, v1.ErrNotFound
	}

	path := fmt.Sprintf("/nodes/%s/qemu/%d/vncwebsocket", node.NodeName, vm.VMID)
	conn, _, err := client.WebSocket(path, params.Encode())
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to dial proxmox vncwebsocket", zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Uint32("vmid", vm.VMID))
		return nil, v1.ErrInternalServerError
	}
	return conn, nil
}

func (s *pveVMService) GetVMRRDData(ctx context.Context, vmID int64, timeframe, cf string) ([]map[string]interface{}, error) {
	client, node, err := s.getProxmoxClientForVM(ctx, vmID)
	if err != nil {
		return nil, err
	}

	vm, err := s.vmRepo.GetByID(ctx, vmID)
	if err != nil {
		return nil, v1.ErrInternalServerError
	}

	rrdData, err := client.GetVMRRDData(ctx, node.NodeName, vm.VMID, timeframe, cf)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get vm rrd data", zap.Error(err),
			zap.String("node", node.NodeName), zap.Uint32("vmid", vm.VMID),
			zap.String("timeframe", timeframe), zap.String("cf", cf))
		return nil, v1.ErrInternalServerError
	}

	return rrdData, nil
}

func (s *pveVMService) MigrateVM(ctx context.Context, req *v1.MigrateVMRequest) (string, error) {
	// 1. 获取源虚拟机信息
	vm, err := s.vmRepo.GetByID(ctx, req.VMID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get vm", zap.Error(err))
		return "", v1.ErrInternalServerError
	}
	if vm == nil {
		return "", v1.ErrNotFound
	}

	// 2. 获取源集群和节点信息
	client, sourceNode, err := s.getProxmoxClientForVM(ctx, req.VMID)
	if err != nil {
		return "", err
	}

	// 3. 获取目标节点信息（必须在同一集群内）
	targetNode, err := s.nodeRepo.GetByID(ctx, req.TargetNodeID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get target node", zap.Error(err))
		return "", v1.ErrInternalServerError
	}
	if targetNode == nil {
		return "", fmt.Errorf("目标节点 ID %d 不存在", req.TargetNodeID)
	}

	// 验证目标节点是否在同一集群
	if targetNode.ClusterID != vm.ClusterID {
		return "", fmt.Errorf("目标节点不在同一集群内，请使用远程迁移接口")
	}

	// 4. 构建迁移参数
	params := make(map[string]interface{})
	params["target"] = targetNode.NodeName

	if req.Online != nil {
		params["online"] = *req.Online
	}
	if req.Bwlimit != nil {
		params["bwlimit"] = *req.Bwlimit
	}
	if req.WithLocalDisks != nil {
		params["with-local-disks"] = *req.WithLocalDisks
	}
	// 注意：migration-type 参数在某些 Proxmox 版本中可能不支持，暂时移除
	// if req.MigrationType != "" {
	// 	params["migration-type"] = req.MigrationType
	// }
	if req.MigrationNetwork != "" {
		params["migration-network"] = req.MigrationNetwork
	}
	if req.MapStorage != "" {
		params["map-storage"] = req.MapStorage
	}

	// 5. 调用 Proxmox API 执行迁移
	upid, err := client.MigrateVM(ctx, sourceNode.NodeName, vm.VMID, params)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to migrate vm", zap.Error(err),
			zap.String("source_node", sourceNode.NodeName),
			zap.String("target_node", targetNode.NodeName),
			zap.Uint32("vmid", vm.VMID))
		return "", v1.ErrInternalServerError
	}

	s.logger.WithContext(ctx).Info("vm migration started", zap.Uint32("vmid", vm.VMID),
		zap.String("source_node", sourceNode.NodeName),
		zap.String("target_node", targetNode.NodeName),
		zap.String("upid", upid))

	return upid, nil
}

func (s *pveVMService) RemoteMigrateVM(ctx context.Context, req *v1.RemoteMigrateVMRequest) (string, error) {
	// 1. 获取源虚拟机信息
	vm, err := s.vmRepo.GetByID(ctx, req.VMID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get vm", zap.Error(err))
		return "", v1.ErrInternalServerError
	}
	if vm == nil {
		return "", v1.ErrNotFound
	}

	// 2. 获取源集群和节点信息
	client, sourceNode, err := s.getProxmoxClientForVM(ctx, req.VMID)
	if err != nil {
		return "", err
	}

	// 3. 获取目标集群信息
	targetCluster, err := s.clusterRepo.GetByID(ctx, req.TargetClusterID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get target cluster", zap.Error(err))
		return "", v1.ErrInternalServerError
	}
	if targetCluster == nil {
		return "", fmt.Errorf("目标集群 ID %d 不存在", req.TargetClusterID)
	}

	// 4. 获取目标节点信息
	targetNode, err := s.nodeRepo.GetByID(ctx, req.TargetNodeID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get target node", zap.Error(err))
		return "", v1.ErrInternalServerError
	}
	if targetNode == nil {
		return "", fmt.Errorf("目标节点 ID %d 不存在", req.TargetNodeID)
	}

	// 验证目标节点是否在目标集群内
	if targetNode.ClusterID != req.TargetClusterID {
		return "", fmt.Errorf("目标节点不在指定的目标集群内")
	}

	// 5. 获取目标集群的 fingerprint
	// 创建目标集群的客户端来获取证书信息
	targetClient, err := proxmox.NewProxmoxClient(targetCluster.ApiUrl, targetCluster.UserId, targetCluster.UserToken)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to create target cluster client", zap.Error(err))
		return "", v1.ErrInternalServerError
	}

	var fingerprint string
	certificates, err := targetClient.GetNodeCertificatesInfo(ctx, targetNode.NodeName)
	if err != nil {
		s.logger.WithContext(ctx).Warn("failed to get target node certificates info, will proceed without fingerprint", zap.Error(err))
		// 如果获取失败，继续执行但不使用 fingerprint
	} else {
		// 查找 filename == "pve-ssl.pem" 的证书的 fingerprint（不要使用 pve-root-ca.pem）
		for _, cert := range certificates {
			if filename, ok := cert["filename"].(string); ok && filename == "pve-ssl.pem" {
				if fp, ok := cert["fingerprint"].(string); ok && fp != "" {
					fingerprint = fp
					s.logger.WithContext(ctx).Info("found pve-ssl.pem fingerprint",
						zap.String("fingerprint", fingerprint))
					break
				}
			}
		}
		if fingerprint == "" {
			s.logger.WithContext(ctx).Warn("pve-ssl.pem certificate not found, will proceed without fingerprint",
				zap.Int("cert_count", len(certificates)))
		}
	}

	// 6. 构建 target-endpoint
	// 格式：host=<TARGET_IP>,apitoken=<API_TOKEN>[,port=<PORT>][,fingerprint=<FINGERPRINT>]
	// API_TOKEN 格式：PVEAPIToken=<UserId>=<UserToken>
	apiToken := fmt.Sprintf("PVEAPIToken=%s=%s", targetCluster.UserId, targetCluster.UserToken)

	// 从 ApiUrl 中提取 host 和 port
	targetURL, err := url.Parse(targetCluster.ApiUrl)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to parse target cluster api url", zap.Error(err))
		return "", fmt.Errorf("目标集群 API URL 格式错误")
	}

	targetHost := targetURL.Hostname()
	targetPort := targetURL.Port()
	if targetPort == "" {
		// 默认端口
		if targetURL.Scheme == "https" {
			targetPort = "8006"
		} else {
			targetPort = "8006"
		}
	}

	// 构建 target-endpoint，格式：host=<HOST>,apitoken=<TOKEN>[,fingerprint=<FINGERPRINT>],port=<PORT>
	// 注意：参数顺序可能重要，按照 Proxmox 文档格式
	targetEndpoint := fmt.Sprintf("host=%s,apitoken=%s", targetHost, apiToken)
	if fingerprint != "" {
		targetEndpoint += fmt.Sprintf(",fingerprint=%s", fingerprint)
	}
	targetEndpoint += fmt.Sprintf(",port=%s", targetPort)

	// 7. 构建迁移参数
	params := make(map[string]interface{})
	params["target-endpoint"] = targetEndpoint
	params["target-bridge"] = req.TargetBridge
	params["target-storage"] = req.TargetStorage

	// 可选参数
	if req.TargetVMID != nil {
		params["target-vmid"] = *req.TargetVMID
	}
	if req.Online != nil {
		params["online"] = *req.Online
	}
	if req.Bwlimit != nil {
		params["bwlimit"] = *req.Bwlimit
	}
	if req.Delete != nil {
		params["delete"] = *req.Delete
	}

	// 记录迁移参数（不记录敏感信息）
	s.logger.WithContext(ctx).Info("remote migrate vm params",
		zap.String("source_node", sourceNode.NodeName),
		zap.String("target_node", targetNode.NodeName),
		zap.String("target_cluster", targetCluster.ClusterName),
		zap.Uint32("vmid", vm.VMID),
		zap.String("target_bridge", req.TargetBridge),
		zap.String("target_storage", req.TargetStorage),
		zap.String("target_host", targetHost),
		zap.String("target_port", targetPort),
		zap.String("fingerprint", fingerprint),
		zap.String("target_endpoint", targetEndpoint))

	// 8. 调用 Proxmox API 执行跨集群迁移
	upid, err := client.RemoteMigrateVM(ctx, sourceNode.NodeName, vm.VMID, params)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to remote migrate vm", zap.Error(err),
			zap.String("source_node", sourceNode.NodeName),
			zap.String("target_node", targetNode.NodeName),
			zap.String("target_cluster", targetCluster.ClusterName),
			zap.Uint32("vmid", vm.VMID),
			zap.String("target_endpoint", targetEndpoint)) // 记录完整的 endpoint 用于调试
		return "", v1.ErrInternalServerError
	}

	s.logger.WithContext(ctx).Info("vm remote migration started", zap.Uint32("vmid", vm.VMID),
		zap.String("source_node", sourceNode.NodeName),
		zap.String("target_node", targetNode.NodeName),
		zap.String("target_cluster", targetCluster.ClusterName),
		zap.String("upid", upid))

	return upid, nil
}

// CreateBackup 创建虚拟机备份
// 参考: https://pve.proxmox.com/pve-docs/api-viewer/#/nodes/{node}/vzdump
func (s *pveVMService) CreateBackup(ctx context.Context, req *v1.CreateBackupRequest) (*v1.CreateBackupResponseData, error) {
	// 1. 通过 VMID 查询虚拟机（需要先获取节点信息）
	// 由于 GetByVMID 需要 nodeID，我们需要先通过其他方式查询
	// 或者直接通过 Proxmox API 验证 VM 是否存在
	// 这里我们通过查询所有集群的 VM 来找到匹配的

	// 先尝试通过 Proxmox API 直接创建备份（如果 VMID 存在）
	// 但为了获取节点信息，我们需要查询数据库

	// 方案：查询所有 VM，找到匹配的 VMID
	var vm *model.PveVM
	allVMs, _, err := s.vmRepo.ListWithPagination(ctx, 1, 1000, 0, "", 0, "", 0, "", "")
	if err == nil {
		for _, v := range allVMs {
			if v.VMID == req.VMID {
				vm = v
				break
			}
		}
	}

	if vm == nil {
		return nil, fmt.Errorf("虚拟机 VMID %d 不存在", req.VMID)
	}

	// 2. 获取节点信息
	node, err := s.nodeRepo.GetByID(ctx, vm.NodeID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	if node == nil {
		return nil, fmt.Errorf("节点不存在")
	}

	// 3. 获取 Proxmox 客户端
	client, _, err := s.getProxmoxClientForVM(ctx, vm.Id)
	if err != nil {
		return nil, err
	}

	// 4. 构建备份请求参数
	backupReq := &proxmox.CreateBackupRequest{
		VMID: req.VMID,
	}

	if req.Storage != "" {
		backupReq.Storage = req.Storage
	}
	if req.Compress != "" {
		// Proxmox API 压缩格式转换：支持常用别名
		compress := req.Compress
		switch compress {
		case "zst":
			compress = "zstd" // zst -> zstd
		case "gz":
			compress = "gzip" // gz -> gzip
		}
		backupReq.Compress = compress
	}
	if req.Mode != "" {
		backupReq.Mode = req.Mode
	} else {
		backupReq.Mode = "snapshot" // 默认使用快照模式
	}
	if req.Remove != nil {
		backupReq.Remove = *req.Remove
	}
	if req.MailTo != "" {
		backupReq.MailTo = req.MailTo
	}
	if req.MailNotification != "" {
		backupReq.MailNotification = req.MailNotification
	}
	if req.NotesTemplate != "" {
		backupReq.NotesTemplate = req.NotesTemplate
	}
	if req.Exclude != "" {
		backupReq.Exclude = req.Exclude
	}
	if req.Quiesce != nil {
		backupReq.Quiesce = *req.Quiesce
	}
	if req.MaxFiles != nil {
		backupReq.MaxFiles = *req.MaxFiles
	}
	if req.Bwlimit != nil {
		backupReq.Bwlimit = *req.Bwlimit
	}
	if req.Ionice != nil {
		backupReq.Ionice = *req.Ionice
	}
	if req.Stop != nil {
		backupReq.Stop = *req.Stop
	}
	if req.StopWait != nil {
		backupReq.StopWait = *req.StopWait
	}
	if req.DumpDir != "" {
		backupReq.DumpDir = req.DumpDir
	}
	if req.Zstd != nil {
		backupReq.Zstd = *req.Zstd
	}

	// 5. 调用 Proxmox API 创建备份
	upid, err := client.CreateBackup(ctx, node.NodeName, backupReq)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to create backup",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.Uint32("vmid", req.VMID))
		return nil, fmt.Errorf("创建备份失败: %w", err)
	}

	s.logger.WithContext(ctx).Info("vm backup started",
		zap.String("node", node.NodeName),
		zap.Uint32("vmid", req.VMID),
		zap.String("upid", upid),
		zap.String("mode", backupReq.Mode),
		zap.String("storage", backupReq.Storage))

	return &v1.CreateBackupResponseData{
		UPID:     upid,
		VMID:     req.VMID,
		NodeID:   node.Id,
		NodeName: node.NodeName,
	}, nil
}

// DeleteBackup 删除虚拟机备份
// 参考: https://pve.proxmox.com/pve-docs/api-viewer/#/nodes/{node}/storage/{storage}/content/{volume}
func (s *pveVMService) DeleteBackup(ctx context.Context, req *v1.DeleteBackupRequest) error {
	// 1. 获取节点信息
	node, err := s.nodeRepo.GetByID(ctx, req.NodeID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if node == nil {
		return fmt.Errorf("节点 ID %d 不存在", req.NodeID)
	}

	// 2. 获取存储信息
	storage, err := s.storageRepo.GetByID(ctx, req.StorageID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get storage", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if storage == nil {
		return fmt.Errorf("存储 ID %d 不存在", req.StorageID)
	}

	// 3. 获取 Proxmox 客户端
	client, err := s.getProxmoxClientForNode(ctx, node.Id)
	if err != nil {
		return err
	}

	// 4. 调用 Proxmox API 删除备份
	// volume 格式：storage:backup/filename，例如：local:backup/vzdump-qemu-100-2024_01_01-00_00_00.vma.zst
	err = client.DeleteStorageContent(ctx, node.NodeName, storage.StorageName, req.Volume, req.Delay)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to delete backup",
			zap.Error(err),
			zap.String("node", node.NodeName),
			zap.String("storage", storage.StorageName),
			zap.String("volume", req.Volume))
		return fmt.Errorf("删除备份失败: %w", err)
	}

	s.logger.WithContext(ctx).Info("backup deleted successfully",
		zap.String("node", node.NodeName),
		zap.String("storage", storage.StorageName),
		zap.String("volume", req.Volume))

	return nil
}

// GetVMCloudInit 获取虚拟机 CloudInit 配置
func (s *pveVMService) GetVMCloudInit(ctx context.Context, req *v1.GetVMCloudInitRequest) (map[string]interface{}, error) {
	// 1. 获取节点信息
	node, err := s.nodeRepo.GetByID(ctx, req.NodeID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node", zap.Error(err))
		return nil, v1.ErrInternalServerError
	}
	if node == nil {
		return nil, fmt.Errorf("节点 ID %d 不存在", req.NodeID)
	}

	// 2. 获取 Proxmox 客户端
	client, err := s.getProxmoxClientForNode(ctx, node.Id)
	if err != nil {
		return nil, err
	}

	// 3. 调用 Proxmox API 获取 CloudInit 配置
	config, err := client.GetVMCloudInitConfig(ctx, node.NodeName, req.VMID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get vm cloudinit config",
			zap.Error(err),
			zap.Uint32("vmid", req.VMID),
			zap.String("node", node.NodeName))
		return nil, fmt.Errorf("获取 CloudInit 配置失败: %w", err)
	}

	s.logger.WithContext(ctx).Info("vm cloudinit config retrieved successfully",
		zap.Uint32("vmid", req.VMID),
		zap.String("node", node.NodeName))

	return config, nil
}

// UpdateVMCloudInit 更新虚拟机 CloudInit 配置
func (s *pveVMService) UpdateVMCloudInit(ctx context.Context, req *v1.UpdateVMCloudInitRequest) error {
	// 1. 获取节点信息
	node, err := s.nodeRepo.GetByID(ctx, req.NodeID)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to get node", zap.Error(err))
		return v1.ErrInternalServerError
	}
	if node == nil {
		return fmt.Errorf("节点 ID %d 不存在", req.NodeID)
	}

	// 2. 获取 Proxmox 客户端
	client, err := s.getProxmoxClientForNode(ctx, node.Id)
	if err != nil {
		return err
	}

	// 3. 构建 form 参数
	params := url.Values{}
	if req.Cipassword != nil && *req.Cipassword != "" {
		params.Set("cipassword", *req.Cipassword)
	}
	if req.CIuser != nil && *req.CIuser != "" {
		params.Set("ciuser", *req.CIuser)
	}
	if req.Citype != nil && *req.Citype != "" {
		params.Set("citype", *req.Citype)
	}
	if req.Nameserver != nil && *req.Nameserver != "" {
		params.Set("nameserver", *req.Nameserver)
	}
	if req.Searchdomain != nil && *req.Searchdomain != "" {
		params.Set("searchdomain", *req.Searchdomain)
	}
	if req.SSHkeys != nil && *req.SSHkeys != "" {
		params.Set("sshkeys", *req.SSHkeys)
	}
	if req.IPconfig0 != nil && *req.IPconfig0 != "" {
		params.Set("ipconfig0", *req.IPconfig0)
	}
	if req.IPconfig1 != nil && *req.IPconfig1 != "" {
		params.Set("ipconfig1", *req.IPconfig1)
	}
	if req.IPconfig2 != nil && *req.IPconfig2 != "" {
		params.Set("ipconfig2", *req.IPconfig2)
	}

	// 4. 调用 Proxmox API 更新 CloudInit 配置
	err = client.UpdateVMCloudInitConfig(ctx, node.NodeName, req.VMID, params)
	if err != nil {
		s.logger.WithContext(ctx).Error("failed to update vm cloudinit config",
			zap.Error(err),
			zap.Uint32("vmid", req.VMID),
			zap.String("node", node.NodeName))
		return fmt.Errorf("更新 CloudInit 配置失败: %w", err)
	}

	s.logger.WithContext(ctx).Info("vm cloudinit config updated successfully",
		zap.Uint32("vmid", req.VMID),
		zap.String("node", node.NodeName))

	return nil
}
