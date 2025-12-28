package repository

import (
	"context"
	"errors"
	"pvesphere/internal/model"
	"time"

	"gorm.io/gorm"
)

type PveVMRepository interface {
	Create(ctx context.Context, vm *model.PveVM) error
	Update(ctx context.Context, vm *model.PveVM) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*model.PveVM, error)
	GetByVMID(ctx context.Context, vmid uint32, nodeID int64) (*model.PveVM, error)               // 通过 VM ID 和节点 ID 查询
	GetByVMIDAndNodeName(ctx context.Context, vmid uint32, nodeName string) (*model.PveVM, error) // 通过 VM ID 和节点名称查询（向后兼容）
	GetByClusterID(ctx context.Context, clusterID int64) ([]*model.PveVM, error)                         // 通过集群 ID 查询
	GetByClusterName(ctx context.Context, clusterName string) ([]*model.PveVM, error)                    // 通过集群名称查询（向后兼容）
	ListWithPagination(ctx context.Context, page, pageSize int, clusterID int64, clusterName string, nodeID int64, nodeName string, templateID int64, status, appId string) ([]*model.PveVM, int64, error)
	Upsert(ctx context.Context, vm *model.PveVM) error
	DeleteByVMID(ctx context.Context, vmid uint32, nodeID int64) error
	GetHashByVMID(ctx context.Context, vmid uint32, nodeID int64) (string, int64, error)
	UpdateSyncTimeOnly(ctx context.Context, id int64) error
	GetTemplateVMByID(ctx context.Context, templateID, clusterID int64, nodeName string) (*model.PveVM, error) // 根据模板 ID、集群 ID 和节点名称查找模板虚拟机
	GetTemplateVM(ctx context.Context, templateName, clusterName string) (*model.PveVM, error)                 // 根据模板名称和集群名称查找模板虚拟机（向后兼容）
}

func NewPveVMRepository(r *Repository) PveVMRepository {
	return &pveVMRepository{Repository: r}
}

type pveVMRepository struct {
	*Repository
}

func (r *pveVMRepository) Create(ctx context.Context, vm *model.PveVM) error {
	return r.DB(ctx).Create(vm).Error
}

func (r *pveVMRepository) Update(ctx context.Context, vm *model.PveVM) error {
	return r.DB(ctx).Save(vm).Error
}

func (r *pveVMRepository) GetByID(ctx context.Context, id int64) (*model.PveVM, error) {
	var vm model.PveVM
	if err := r.DB(ctx).Where("id = ?", id).First(&vm).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &vm, nil
}

func (r *pveVMRepository) GetByVMID(ctx context.Context, vmid uint32, nodeID int64) (*model.PveVM, error) {
	var vm model.PveVM
	if err := r.DB(ctx).Where("vmid = ? AND node_id = ?", vmid, nodeID).First(&vm).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &vm, nil
}

func (r *pveVMRepository) GetByVMIDAndNodeName(ctx context.Context, vmid uint32, nodeName string) (*model.PveVM, error) {
	var vm model.PveVM
	// 通过 JOIN Node 表查询，使用 node_id 关联
	err := r.DB(ctx).
		Table("pve_vm").
		Select("pve_vm.*").
		Joins("JOIN pve_node ON pve_vm.node_id = pve_node.id").
		Where("pve_vm.vmid = ? AND pve_node.node_name = ?", vmid, nodeName).
		First(&vm).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &vm, nil
}

func (r *pveVMRepository) GetByClusterID(ctx context.Context, clusterID int64) ([]*model.PveVM, error) {
	var vms []*model.PveVM
	if err := r.DB(ctx).Where("cluster_id = ?", clusterID).Find(&vms).Error; err != nil {
		return nil, err
	}
	return vms, nil
}

func (r *pveVMRepository) GetByClusterName(ctx context.Context, clusterName string) ([]*model.PveVM, error) {
	var vms []*model.PveVM
	if err := r.DB(ctx).Where("cluster_name = ?", clusterName).Find(&vms).Error; err != nil {
		return nil, err
	}
	return vms, nil
}

func (r *pveVMRepository) Upsert(ctx context.Context, vm *model.PveVM) error {
	// 验证必要字段
	if vm.VMID == 0 {
		return errors.New("vmid is required")
	}
	if vm.NodeID == 0 && vm.NodeName == "" {
		return errors.New("node_id or node_name is required")
	}

	// 先查询是否存在以及 hash
	var existingHash string
	var existingID int64
	var err error

	if vm.NodeID > 0 {
		existingHash, existingID, err = r.GetHashByVMID(ctx, vm.VMID, vm.NodeID)
	} else if vm.NodeName != "" {
		// 向后兼容：如果 NodeID 为空，使用 NodeName
		existingHash, existingID, err = r.GetHashByVMIDAndNodeName(ctx, vm.VMID, vm.NodeName)
	}

	if err != nil {
		return err
	}

	// 如果不存在，创建新记录
	if existingID == 0 {
		// 使用事务确保唯一性（即使唯一索引存在，也要防止并发问题）
		return r.DB(ctx).Create(vm).Error
	}

	// 如果 hash 相同，只更新同步时间（轻量级更新）
	if existingHash != "" && existingHash == vm.ResourceHash {
		vm.Id = existingID
		return r.UpdateSyncTimeOnly(ctx, existingID)
	}

	// hash 不同，完整更新记录
	vm.Id = existingID
	return r.Update(ctx, vm)
}

func (r *pveVMRepository) GetHashByVMID(ctx context.Context, vmid uint32, nodeID int64) (string, int64, error) {
	var result struct {
		Id           int64  `gorm:"column:id"`
		ResourceHash string `gorm:"column:resource_hash"`
	}

	err := r.DB(ctx).
		Table("pve_vm").
		Select("id, resource_hash").
		Where("vmid = ? AND node_id = ?", vmid, nodeID).
		First(&result).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", 0, nil
	}
	if err != nil {
		return "", 0, err
	}

	return result.ResourceHash, result.Id, nil
}

func (r *pveVMRepository) GetHashByVMIDAndNodeName(ctx context.Context, vmid uint32, nodeName string) (string, int64, error) {
	// 先通过 NodeName 查询 Node 获取 NodeID（需要知道 ClusterID，但这里没有，所以先尝试通过 VM 表查找）
	// 由于数据库表中已经没有 node_name 字段，我们需要通过 JOIN Node 表来查询
	var result struct {
		Id           int64  `gorm:"column:id"`
		ResourceHash string `gorm:"column:resource_hash"`
	}

	// 通过 JOIN Node 表查询，使用 node_id 关联
	err := r.DB(ctx).
		Table("pve_vm").
		Select("pve_vm.id, pve_vm.resource_hash").
		Joins("JOIN pve_node ON pve_vm.node_id = pve_node.id").
		Where("pve_vm.vmid = ? AND pve_node.node_name = ?", vmid, nodeName).
		First(&result).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", 0, nil
	}
	if err != nil {
		return "", 0, err
	}

	return result.ResourceHash, result.Id, nil
}

func (r *pveVMRepository) UpdateSyncTimeOnly(ctx context.Context, id int64) error {
	return r.DB(ctx).
		Model(&model.PveVM{}).
		Where("id = ?", id).
		Update("last_sync_time", time.Now()).Error
}

func (r *pveVMRepository) DeleteByVMID(ctx context.Context, vmid uint32, nodeID int64) error {
	return r.DB(ctx).Where("vmid = ? AND node_id = ?", vmid, nodeID).Delete(&model.PveVM{}).Error
}

func (r *pveVMRepository) Delete(ctx context.Context, id int64) error {
	return r.DB(ctx).Where("id = ?", id).Delete(&model.PveVM{}).Error
}

func (r *pveVMRepository) ListWithPagination(ctx context.Context, page, pageSize int, clusterID int64, clusterName string, nodeID int64, nodeName string, templateID int64, status, appId string) ([]*model.PveVM, int64, error) {
	var vms []*model.PveVM
	var total int64

	query := r.DB(ctx).Model(&model.PveVM{})

	// 优先使用 ID 过滤，如果 ID 为空则使用名称（向后兼容）
	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	} else if clusterName != "" {
		query = query.Where("cluster_name = ?", clusterName)
	}

	if nodeID > 0 {
		query = query.Where("node_id = ?", nodeID)
	} else if nodeName != "" {
		query = query.Where("node_name = ?", nodeName)
	}

	if templateID > 0 {
		query = query.Where("template_id = ?", templateID)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if appId != "" {
		query = query.Where("appid = ?", appId)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&vms).Error; err != nil {
		return nil, 0, err
	}

	return vms, total, nil
}

// GetTemplateVMByID 根据模板 ID、集群 ID 和节点名称查找模板虚拟机
// 模板虚拟机应该通过 vm_name 匹配模板名称来查找，而不是通过 template_id
// 因为 template_id 字段表示虚拟机是从哪个模板创建的，而不是模板虚拟机本身
func (r *pveVMRepository) GetTemplateVMByID(ctx context.Context, templateID, clusterID int64, nodeName string) (*model.PveVM, error) {
	// 通过 JOIN vm_template 和 pve_node 表来查询
	var vm model.PveVM
	query := r.DB(ctx).
		Table("pve_vm").
		Select("pve_vm.*").
		Joins("JOIN vm_template ON pve_vm.vm_name = vm_template.template_name").
		Joins("JOIN pve_node ON pve_vm.node_id = pve_node.id").
		Where("vm_template.id = ? AND pve_vm.cluster_id = ?", templateID, clusterID)

	// 如果提供了节点名称，则添加节点名称过滤条件
	if nodeName != "" {
		query = query.Where("pve_node.node_name = ?", nodeName)
	}

	err := query.First(&vm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &vm, nil
}

// GetTemplateVM 根据模板名称和集群名称查找模板虚拟机（向后兼容）
// 假设模板虚拟机在 pve_vm 表中，vm_name 或 template_name 字段等于模板名称
func (r *pveVMRepository) GetTemplateVM(ctx context.Context, templateName, clusterName string) (*model.PveVM, error) {
	var vm model.PveVM
	err := r.DB(ctx).
		Where("cluster_name = ? AND (vm_name = ? OR template_name = ?)", clusterName, templateName, templateName).
		First(&vm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &vm, nil
}
