package app

import (
	"strings"
	"testing"

	"github.com/LurusTech/lurus-hub/internal/pkg/common"
)

func TestCheckPermission_ReturnsNil_WhenActorRoleHigherThanTarget(t *testing.T) {
	tests := []struct {
		name       string
		actorRole  int
		targetRole int
	}{
		{"admin over common user", common.RoleAdminUser, common.RoleCommonUser},
		{"root over admin", common.RoleRootUser, common.RoleAdminUser},
		{"root over common user", common.RoleRootUser, common.RoleCommonUser},
		{"admin over guest", common.RoleAdminUser, common.RoleGuestUser},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckPermission(tt.actorRole, tt.targetRole)
			if err != nil {
				t.Errorf("expected nil, got %v", err)
			}
		})
	}
}

func TestCheckPermission_ReturnsNil_WhenActorIsRootUser(t *testing.T) {
	// Root user can operate on anyone, including another root user
	tests := []struct {
		name       string
		actorRole  int
		targetRole int
	}{
		{"root on root", common.RoleRootUser, common.RoleRootUser},
		{"root on admin", common.RoleRootUser, common.RoleAdminUser},
		{"root on common", common.RoleRootUser, common.RoleCommonUser},
		{"root on guest", common.RoleRootUser, common.RoleGuestUser},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckPermission(tt.actorRole, tt.targetRole)
			if err != nil {
				t.Errorf("expected nil, got %v", err)
			}
		})
	}
}

func TestCheckPermission_ReturnsError_WhenActorRoleLessOrEqualTarget(t *testing.T) {
	tests := []struct {
		name       string
		actorRole  int
		targetRole int
	}{
		{"common on common (equal)", common.RoleCommonUser, common.RoleCommonUser},
		{"common on admin (lower)", common.RoleCommonUser, common.RoleAdminUser},
		{"admin on admin (equal)", common.RoleAdminUser, common.RoleAdminUser},
		{"guest on common (lower)", common.RoleGuestUser, common.RoleCommonUser},
		{"common on root (lower)", common.RoleCommonUser, common.RoleRootUser},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckPermission(tt.actorRole, tt.targetRole)
			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

func TestCheckRolePromotion_ReturnsNil_WhenActorRoleHigherThanNewRole(t *testing.T) {
	tests := []struct {
		name      string
		actorRole int
		newRole   int
	}{
		{"admin promotes to common", common.RoleAdminUser, common.RoleCommonUser},
		{"root promotes to admin", common.RoleRootUser, common.RoleAdminUser},
		{"admin promotes to guest", common.RoleAdminUser, common.RoleGuestUser},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckRolePromotion(tt.actorRole, tt.newRole)
			if err != nil {
				t.Errorf("expected nil, got %v", err)
			}
		})
	}
}

func TestCheckRolePromotion_ReturnsNil_WhenActorIsRootUser(t *testing.T) {
	// Root user can promote to any role, including root
	tests := []struct {
		name      string
		actorRole int
		newRole   int
	}{
		{"root promotes to root", common.RoleRootUser, common.RoleRootUser},
		{"root promotes to admin", common.RoleRootUser, common.RoleAdminUser},
		{"root promotes to common", common.RoleRootUser, common.RoleCommonUser},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckRolePromotion(tt.actorRole, tt.newRole)
			if err != nil {
				t.Errorf("expected nil, got %v", err)
			}
		})
	}
}

func TestCheckRolePromotion_ReturnsError_WhenActorRoleLessOrEqualNewRole(t *testing.T) {
	tests := []struct {
		name      string
		actorRole int
		newRole   int
	}{
		{"common promotes to common (equal)", common.RoleCommonUser, common.RoleCommonUser},
		{"common promotes to admin (lower)", common.RoleCommonUser, common.RoleAdminUser},
		{"admin promotes to admin (equal)", common.RoleAdminUser, common.RoleAdminUser},
		{"admin promotes to root (lower)", common.RoleAdminUser, common.RoleRootUser},
		{"guest promotes to common (lower)", common.RoleGuestUser, common.RoleCommonUser},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckRolePromotion(tt.actorRole, tt.newRole)
			if err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

func TestValidateDisplayName_ReturnsNil_WhenWithinLimit(t *testing.T) {
	tests := []struct {
		name        string
		displayName string
	}{
		{"empty string", ""},
		{"single character", "A"},
		{"exactly 50 characters", strings.Repeat("a", 50)},
		{"normal name", "Alice Johnson"},
		{"unicode within limit", strings.Repeat("\u4e2d", 16)}, // 16 CJK chars = 48 bytes, within limit
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDisplayName(tt.displayName)
			if err != nil {
				t.Errorf("expected nil for display name len=%d, got %v", len(tt.displayName), err)
			}
		})
	}
}

func TestValidateDisplayName_ReturnsError_WhenExceedsLimit(t *testing.T) {
	tests := []struct {
		name        string
		displayName string
	}{
		{"51 characters", strings.Repeat("a", 51)},
		{"100 characters", strings.Repeat("b", 100)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDisplayName(tt.displayName)
			if err == nil {
				t.Errorf("expected error for display name len=%d, got nil", len(tt.displayName))
			}
		})
	}
}

func TestGetTenantIdFromContext_ReturnsDefault_WhenEmpty(t *testing.T) {
	result := GetTenantIdFromContext("")
	if result != "default" {
		t.Errorf("expected \"default\", got %q", result)
	}
}

func TestGetTenantIdFromContext_ReturnsTenantId_WhenProvided(t *testing.T) {
	tests := []struct {
		name     string
		tenantId string
		expected string
	}{
		{"simple tenant id", "tenant-abc", "tenant-abc"},
		{"numeric tenant id", "12345", "12345"},
		{"uuid tenant id", "550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTenantIdFromContext(tt.tenantId)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
