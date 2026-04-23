package repo

import (
	"github.com/LurusTech/lurus-api/internal/domain/entity"
	"github.com/LurusTech/lurus-api/internal/pkg/common"
)

type Vendor = entity.Vendor

// VendorInsert 创建新的供应商记录
func VendorInsert(v *Vendor) error {
	now := common.GetTimestamp()
	v.CreatedTime = now
	v.UpdatedTime = now
	return DB.Create(v).Error
}

// IsVendorNameDuplicated 检查供应商名称是否重复（排除自身 ID）
func IsVendorNameDuplicated(id int, name string) (bool, error) {
	if name == "" {
		return false, nil
	}
	var cnt int64
	err := DB.Model(&Vendor{}).Where("name = ? AND id <> ?", name, id).Count(&cnt).Error
	return cnt > 0, err
}

// VendorUpdate 更新供应商记录
func VendorUpdate(v *Vendor) error {
	v.UpdatedTime = common.GetTimestamp()
	return DB.Save(v).Error
}

// VendorDelete 软删除供应商
func VendorDelete(v *Vendor) error {
	return DB.Delete(v).Error
}

// GetVendorByID 根据 ID 获取供应商
func GetVendorByID(id int) (*Vendor, error) {
	var v Vendor
	err := DB.First(&v, id).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// GetAllVendors 获取全部供应商（分页）
func GetAllVendors(offset int, limit int) ([]*Vendor, error) {
	var vendors []*Vendor
	err := DB.Offset(offset).Limit(limit).Find(&vendors).Error
	return vendors, err
}

// GetOrCreateVendorByName finds a vendor by name, creating it if it doesn't exist.
func GetOrCreateVendorByName(name string) (int, error) {
	var existing Vendor
	if err := DB.Where("name = ?", name).First(&existing).Error; err == nil {
		return existing.Id, nil
	}
	v := &Vendor{
		Name:   name,
		Status: 1,
	}
	if err := VendorInsert(v); err != nil {
		// Race condition: another goroutine may have created it
		if err2 := DB.Where("name = ?", name).First(&existing).Error; err2 == nil {
			return existing.Id, nil
		}
		return 0, err
	}
	return v.Id, nil
}

// SearchVendors 按关键字搜索供应商
func SearchVendors(keyword string, offset int, limit int) ([]*Vendor, int64, error) {
	db := DB.Model(&Vendor{})
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR description LIKE ?", like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var vendors []*Vendor
	if err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&vendors).Error; err != nil {
		return nil, 0, err
	}
	return vendors, total, nil
}
