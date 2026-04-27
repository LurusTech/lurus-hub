package app

import (
	"testing"

	"github.com/LurusTech/lurus-hub/internal/pkg/setting"
	"github.com/LurusTech/lurus-hub/internal/pkg/setting/ratio_setting"
)

// saveAndRestoreUserUsableGroups saves the current user usable groups
// and restores them after the test completes.
func saveAndRestoreUserUsableGroups(t *testing.T) {
	t.Helper()
	orig := setting.UserUsableGroups2JSONString()
	t.Cleanup(func() {
		_ = setting.UpdateUserUsableGroupsByJSONString(orig)
	})
}

// saveAndRestoreAutoGroups saves the current auto groups
// and restores them after the test completes.
func saveAndRestoreAutoGroups(t *testing.T) {
	t.Helper()
	orig := setting.AutoGroups2JsonString()
	t.Cleanup(func() {
		_ = setting.UpdateAutoGroupsByJsonString(orig)
	})
}

// saveAndRestoreGroupRatio saves the current group ratio
// and restores it after the test completes.
func saveAndRestoreGroupRatio(t *testing.T) {
	t.Helper()
	orig := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		_ = ratio_setting.UpdateGroupRatioByJSONString(orig)
	})
}

// saveAndRestoreGroupGroupRatio saves the current group-group ratio
// and restores it after the test completes.
func saveAndRestoreGroupGroupRatio(t *testing.T) {
	t.Helper()
	orig := ratio_setting.GroupGroupRatio2JSONString()
	t.Cleanup(func() {
		_ = ratio_setting.UpdateGroupGroupRatioByJSONString(orig)
	})
}

func TestGetUserUsableGroups_DefaultGroups(t *testing.T) {
	saveAndRestoreUserUsableGroups(t)
	_ = setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP"}`)

	groups := GetUserUsableGroups("")
	if _, ok := groups["default"]; !ok {
		t.Error("expected 'default' group to be present")
	}
	if _, ok := groups["vip"]; !ok {
		t.Error("expected 'vip' group to be present")
	}
}

func TestGetUserUsableGroups_UserGroupNotInUsable_IsAdded(t *testing.T) {
	saveAndRestoreUserUsableGroups(t)
	_ = setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default"}`)

	groups := GetUserUsableGroups("premium")
	if _, ok := groups["premium"]; !ok {
		t.Error("expected user's own group 'premium' to be added when not in usable groups")
	}
	if _, ok := groups["default"]; !ok {
		t.Error("expected 'default' group to remain")
	}
}

func TestGetUserUsableGroups_EmptyUserGroup(t *testing.T) {
	saveAndRestoreUserUsableGroups(t)
	_ = setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP"}`)

	groups := GetUserUsableGroups("")
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
}

func TestGetUserUsableGroups_SpecialAddGroup(t *testing.T) {
	saveAndRestoreUserUsableGroups(t)
	_ = setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default"}`)

	grs := ratio_setting.GetGroupRatioSetting()
	origSpecial := grs.GroupSpecialUsableGroup.ReadAll()
	t.Cleanup(func() {
		grs.GroupSpecialUsableGroup.Clear()
		grs.GroupSpecialUsableGroup.AddAll(origSpecial)
	})

	// Set up special usable group with +: prefix to add a group
	grs.GroupSpecialUsableGroup.Set("premium", map[string]string{
		"+:extra": "Extra Group",
	})

	groups := GetUserUsableGroups("premium")
	if _, ok := groups["extra"]; !ok {
		t.Error("expected 'extra' group to be added via +: prefix")
	}
	if _, ok := groups["default"]; !ok {
		t.Error("expected 'default' group to remain")
	}
}

func TestGetUserUsableGroups_SpecialRemoveGroup(t *testing.T) {
	saveAndRestoreUserUsableGroups(t)
	_ = setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP"}`)

	grs := ratio_setting.GetGroupRatioSetting()
	origSpecial := grs.GroupSpecialUsableGroup.ReadAll()
	t.Cleanup(func() {
		grs.GroupSpecialUsableGroup.Clear()
		grs.GroupSpecialUsableGroup.AddAll(origSpecial)
	})

	grs.GroupSpecialUsableGroup.Set("premium", map[string]string{
		"-:vip": "removed",
	})

	groups := GetUserUsableGroups("premium")
	if _, ok := groups["vip"]; ok {
		t.Error("expected 'vip' group to be removed via -: prefix")
	}
	if _, ok := groups["default"]; !ok {
		t.Error("expected 'default' group to remain")
	}
}

func TestGetUserUsableGroups_SpecialDirectAddGroup(t *testing.T) {
	saveAndRestoreUserUsableGroups(t)
	_ = setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default"}`)

	grs := ratio_setting.GetGroupRatioSetting()
	origSpecial := grs.GroupSpecialUsableGroup.ReadAll()
	t.Cleanup(func() {
		grs.GroupSpecialUsableGroup.Clear()
		grs.GroupSpecialUsableGroup.AddAll(origSpecial)
	})

	grs.GroupSpecialUsableGroup.Set("premium", map[string]string{
		"direct_group": "Directly Added",
	})

	groups := GetUserUsableGroups("premium")
	if _, ok := groups["direct_group"]; !ok {
		t.Error("expected 'direct_group' to be added directly without prefix")
	}
}

func TestGroupInUserUsableGroups_GroupExists(t *testing.T) {
	saveAndRestoreUserUsableGroups(t)
	_ = setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP"}`)

	if !GroupInUserUsableGroups("", "default") {
		t.Error("expected 'default' to be in usable groups")
	}
	if !GroupInUserUsableGroups("", "vip") {
		t.Error("expected 'vip' to be in usable groups")
	}
}

func TestGroupInUserUsableGroups_GroupMissing(t *testing.T) {
	saveAndRestoreUserUsableGroups(t)
	_ = setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default"}`)

	if GroupInUserUsableGroups("", "nonexistent") {
		t.Error("expected 'nonexistent' to NOT be in usable groups")
	}
}

func TestGroupInUserUsableGroups_UserOwnGroupAlwaysUsable(t *testing.T) {
	saveAndRestoreUserUsableGroups(t)
	_ = setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default"}`)

	// User's own group should always be usable
	if !GroupInUserUsableGroups("premium", "premium") {
		t.Error("expected user's own group 'premium' to be usable")
	}
}

func TestGetUserAutoGroup_FiltersToUsableGroups(t *testing.T) {
	saveAndRestoreUserUsableGroups(t)
	saveAndRestoreAutoGroups(t)

	_ = setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP"}`)
	_ = setting.UpdateAutoGroupsByJsonString(`["default","vip","svip"]`)

	autoGroups := GetUserAutoGroup("")
	found := map[string]bool{}
	for _, g := range autoGroups {
		found[g] = true
	}
	if !found["default"] {
		t.Error("expected 'default' in auto groups")
	}
	if !found["vip"] {
		t.Error("expected 'vip' in auto groups")
	}
	if found["svip"] {
		t.Error("expected 'svip' to be filtered out (not in usable groups)")
	}
}

func TestGetUserAutoGroup_EmptyAutoGroups(t *testing.T) {
	saveAndRestoreAutoGroups(t)
	_ = setting.UpdateAutoGroupsByJsonString(`[]`)

	autoGroups := GetUserAutoGroup("")
	if len(autoGroups) != 0 {
		t.Errorf("expected 0 auto groups, got %d", len(autoGroups))
	}
}

func TestGetUserAutoGroup_EmptyUsableGroups(t *testing.T) {
	saveAndRestoreUserUsableGroups(t)
	saveAndRestoreAutoGroups(t)

	_ = setting.UpdateUserUsableGroupsByJSONString(`{}`)
	_ = setting.UpdateAutoGroupsByJsonString(`["default"]`)

	autoGroups := GetUserAutoGroup("")
	if len(autoGroups) != 0 {
		t.Errorf("expected 0 auto groups when usable groups empty, got %d", len(autoGroups))
	}
}

func TestGetUserGroupRatio_SpecificGroupGroupRatio(t *testing.T) {
	saveAndRestoreGroupGroupRatio(t)
	_ = ratio_setting.UpdateGroupGroupRatioByJSONString(`{"vip":{"default":0.8}}`)

	ratio := GetUserGroupRatio("vip", "default")
	if ratio != 0.8 {
		t.Errorf("expected ratio 0.8, got %f", ratio)
	}
}

func TestGetUserGroupRatio_FallbackToGroupRatio(t *testing.T) {
	saveAndRestoreGroupGroupRatio(t)
	saveAndRestoreGroupRatio(t)

	// No group-group ratio for this combination
	_ = ratio_setting.UpdateGroupGroupRatioByJSONString(`{}`)
	_ = ratio_setting.UpdateGroupRatioByJSONString(`{"default":1.5}`)

	ratio := GetUserGroupRatio("regular", "default")
	if ratio != 1.5 {
		t.Errorf("expected fallback ratio 1.5, got %f", ratio)
	}
}

func TestGetUserGroupRatio_MissingGroupReturnsDefault(t *testing.T) {
	saveAndRestoreGroupGroupRatio(t)
	saveAndRestoreGroupRatio(t)

	_ = ratio_setting.UpdateGroupGroupRatioByJSONString(`{}`)
	_ = ratio_setting.UpdateGroupRatioByJSONString(`{"default":1}`)

	// "nonexistent" group not found in groupRatio => returns 1 (with syslog warning)
	ratio := GetUserGroupRatio("regular", "nonexistent")
	if ratio != 1 {
		t.Errorf("expected default ratio 1 for missing group, got %f", ratio)
	}
}
