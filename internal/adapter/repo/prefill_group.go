package repo

import (
	"github.com/LurusTech/lurus-hub/internal/domain/entity"
	"github.com/LurusTech/lurus-hub/internal/pkg/common"
)

type JSONValue = entity.JSONValue
type PrefillGroup = entity.PrefillGroup

// PrefillGroupInsert 新建组
func PrefillGroupInsert(g *PrefillGroup) error {
	now := common.GetTimestamp()
	g.CreatedTime = now
	g.UpdatedTime = now
	return DB.Create(g).Error
}

// IsPrefillGroupNameDuplicated 检查组名称是否重复（排除自身 ID）
func IsPrefillGroupNameDuplicated(id int, name string) (bool, error) {
	if name == "" {
		return false, nil
	}
	var cnt int64
	err := DB.Model(&PrefillGroup{}).Where("name = ? AND id <> ?", name, id).Count(&cnt).Error
	return cnt > 0, err
}

// PrefillGroupUpdate 更新组
func PrefillGroupUpdate(g *PrefillGroup) error {
	g.UpdatedTime = common.GetTimestamp()
	return DB.Save(g).Error
}

// DeleteByID 根据 ID 删除组
func DeletePrefillGroupByID(id int) error {
	return DB.Delete(&PrefillGroup{}, id).Error
}

// GetAllPrefillGroups 获取全部组，可按类型过滤（为空则返回全部）
func GetAllPrefillGroups(groupType string) ([]*PrefillGroup, error) {
	var groups []*PrefillGroup
	query := DB.Model(&PrefillGroup{})
	if groupType != "" {
		query = query.Where("type = ?", groupType)
	}
	if err := query.Order("updated_time DESC").Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}
