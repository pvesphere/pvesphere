package repository

import (
	"context"
	"errors"
	"pvesphere/internal/model"

	"gorm.io/gorm"
)

type VMIPAddressRepository interface {
	Create(ctx context.Context, ip *model.VMIPAddress) error
	Update(ctx context.Context, ip *model.VMIPAddress) error
	GetByID(ctx context.Context, id int64) (*model.VMIPAddress, error)
	GetByVMID(ctx context.Context, vmID int64) ([]*model.VMIPAddress, error)
	DeleteByVMID(ctx context.Context, vmID int64) error
}

func NewVMIPAddressRepository(r *Repository) VMIPAddressRepository {
	return &vmIPAddressRepository{Repository: r}
}

type vmIPAddressRepository struct {
	*Repository
}

func (r *vmIPAddressRepository) Create(ctx context.Context, ip *model.VMIPAddress) error {
	return r.DB(ctx).Create(ip).Error
}

func (r *vmIPAddressRepository) Update(ctx context.Context, ip *model.VMIPAddress) error {
	return r.DB(ctx).Save(ip).Error
}

func (r *vmIPAddressRepository) GetByID(ctx context.Context, id int64) (*model.VMIPAddress, error) {
	var ip model.VMIPAddress
	if err := r.DB(ctx).Where("id = ?", id).First(&ip).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ip, nil
}

func (r *vmIPAddressRepository) GetByVMID(ctx context.Context, vmID int64) ([]*model.VMIPAddress, error) {
	var ips []*model.VMIPAddress
	if err := r.DB(ctx).Where("vm_id = ?", vmID).Find(&ips).Error; err != nil {
		return nil, err
	}
	return ips, nil
}

func (r *vmIPAddressRepository) DeleteByVMID(ctx context.Context, vmID int64) error {
	return r.DB(ctx).Where("vm_id = ?", vmID).Delete(&model.VMIPAddress{}).Error
}
