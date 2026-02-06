package repo

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/lurus-api/internal/domain/entity"
	"github.com/QuantumNous/lurus-api/internal/pkg/common"
	"gorm.io/gorm"
)

// Type aliases pointing to entity package
type UserIdentityMapping = entity.UserIdentityMapping
type ZitadelUserClaims = entity.ZitadelUserClaims

// GetUserMappingByZitadelID retrieves user mapping by Zitadel User ID and Tenant ID
func GetUserMappingByZitadelID(zitadelUserID string, tenantID string) (*UserIdentityMapping, error) {
	var mapping UserIdentityMapping
	err := DB.Where("zitadel_user_id = ? AND tenant_id = ? AND is_active = ?", zitadelUserID, tenantID, true).
		First(&mapping).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user mapping not found")
		}
		return nil, err
	}
	return &mapping, nil
}

// GetUserMappingByLurusUserID retrieves user mapping by lurus user ID and tenant ID
func GetUserMappingByLurusUserID(lurusUserID int, tenantID string) (*UserIdentityMapping, error) {
	var mapping UserIdentityMapping
	err := DB.Where("lurus_user_id = ? AND tenant_id = ? AND is_active = ?", lurusUserID, tenantID, true).
		First(&mapping).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user mapping not found")
		}
		return nil, err
	}
	return &mapping, nil
}

// CreateUserMapping creates a new user identity mapping
func CreateUserMapping(lurusUserID int, zitadelUserID string, tenantID string, email string, displayName string, preferredUsername string) (*UserIdentityMapping, error) {
	// Check if mapping already exists
	existingMapping, _ := GetUserMappingByZitadelID(zitadelUserID, tenantID)
	if existingMapping != nil {
		// Update last sync time
		now := time.Now()
		existingMapping.LastSyncAt = &now
		existingMapping.Email = email
		existingMapping.DisplayName = displayName
		existingMapping.PreferredUsername = preferredUsername
		existingMapping.UpdatedAt = now
		err := DB.Save(existingMapping).Error
		return existingMapping, err
	}

	now := time.Now()
	mapping := &UserIdentityMapping{
		LurusUserID:       lurusUserID,
		ZitadelUserID:     zitadelUserID,
		TenantID:          tenantID,
		Email:             email,
		DisplayName:       displayName,
		PreferredUsername: preferredUsername,
		LastSyncAt:        &now,
		IsActive:          true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	err := DB.Create(mapping).Error
	if err != nil {
		return nil, err
	}

	return mapping, nil
}

// UpdateUserMapping updates user mapping metadata
func UpdateUserMapping(id int, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	return DB.Model(&UserIdentityMapping{}).Where("id = ?", id).Updates(updates).Error
}

// DeactivateUserMapping deactivates a user mapping (soft delete)
func DeactivateUserMapping(id int) error {
	return UpdateUserMapping(id, map[string]interface{}{
		"is_active": false,
	})
}

// DeleteUserMapping hard deletes a user mapping
func DeleteUserMapping(id int) error {
	return DB.Delete(&UserIdentityMapping{}, "id = ?", id).Error
}

// ListUserMappingsByTenant retrieves all user mappings for a tenant
func ListUserMappingsByTenant(tenantID string, offset int, limit int) ([]*UserIdentityMapping, int64, error) {
	var mappings []*UserIdentityMapping
	var total int64

	query := DB.Model(&UserIdentityMapping{}).Where("tenant_id = ? AND is_active = ?", tenantID, true)

	// Get total count
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err = query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&mappings).Error
	if err != nil {
		return nil, 0, err
	}

	return mappings, total, nil
}

// ListUserMappingsByZitadelUser retrieves all mappings for a Zitadel user across tenants
func ListUserMappingsByZitadelUser(zitadelUserID string) ([]*UserIdentityMapping, error) {
	var mappings []*UserIdentityMapping
	err := DB.Where("zitadel_user_id = ? AND is_active = ?", zitadelUserID, true).
		Order("created_at DESC").
		Find(&mappings).Error
	return mappings, err
}

// SyncUserDataFromZitadel syncs user data from Zitadel claims to mapping
func SyncUserDataFromZitadel(mappingID int, email string, displayName string, preferredUsername string) error {
	now := time.Now()
	return UpdateUserMapping(mappingID, map[string]interface{}{
		"email":              email,
		"display_name":       displayName,
		"preferred_username": preferredUsername,
		"last_sync_at":       &now,
	})
}

// GetUserByZitadelID retrieves lurus user by Zitadel user ID and tenant
// This is a helper function that combines mapping lookup and user retrieval
func GetUserByZitadelID(zitadelUserID string, tenantID string) (*User, *UserIdentityMapping, error) {
	// Get mapping
	mapping, err := GetUserMappingByZitadelID(zitadelUserID, tenantID)
	if err != nil {
		return nil, nil, err
	}

	// Get user
	user, err := GetUserById(mapping.LurusUserID, false)
	if err != nil {
		return nil, nil, err
	}

	return user, mapping, nil
}

// CreateUserFromZitadelClaims creates a new lurus user from Zitadel JWT claims
// and establishes the identity mapping
func CreateUserFromZitadelClaims(claims *ZitadelUserClaims, tenantID string) (*User, *UserIdentityMapping, error) {
	// Check if mapping already exists
	existingMapping, _ := GetUserMappingByZitadelID(claims.Sub, tenantID)
	if existingMapping != nil {
		// User already exists, retrieve and return
		user, err := GetUserById(existingMapping.LurusUserID, false)
		if err != nil {
			return nil, nil, err
		}
		return user, existingMapping, nil
	}

	// Get tenant config for default quota
	tenant, err := GetTenantByID(tenantID)
	if err != nil {
		return nil, nil, err
	}

	// Check if tenant can add more users
	canAdd, err := TenantCanAddUser(tenant)
	if err != nil {
		return nil, nil, err
	}
	if !canAdd {
		return nil, nil, errors.New("tenant has reached maximum user limit")
	}

	// Generate unique username (handle duplicates)
	username := claims.PreferredUsername
	if username == "" {
		username = claims.Email
	}
	username = ensureUniqueUsername(username, tenantID)

	// Get default user quota from tenant config
	defaultQuota := GetTenantConfigInt(tenantID, "quota.new_user_quota", 10000)

	// Create new lurus user
	user := &User{
		Username:    username,
		Email:       claims.Email,
		DisplayName: claims.Name,
		Role:        1, // RoleCommonUser
		Status:      1, // UserStatusEnabled
		Quota:       defaultQuota,
		UsedQuota:   0,
		Group:       "default",
		AffCode:     generateAffCode(),
		// TenantID will be set automatically by GORM plugin in context
	}

	// Note: Password is not set for Zitadel users (they authenticate via Zitadel)
	// If password is required, generate a random strong password
	user.Password = GenerateRandomPassword()

	// Use WithTenantID to inject tenant context for GORM beforeCreate hook
	tenantDB := WithTenantID(DB, tenantID)

	err = tenantDB.Create(user).Error
	if err != nil {
		return nil, nil, err
	}

	// Create identity mapping (user_identity_mapping table does not have
	// tenant isolation plugin — it manages tenant_id explicitly)
	mapping, err := CreateUserMapping(
		user.Id,
		claims.Sub,
		tenantID,
		claims.Email,
		claims.Name,
		claims.PreferredUsername,
	)
	if err != nil {
		// Rollback user creation if mapping fails
		tenantDB.Delete(user)
		return nil, nil, err
	}

	return user, mapping, nil
}

// ensureUniqueUsername ensures username is unique within tenant
func ensureUniqueUsername(baseUsername string, tenantID string) string {
	username := baseUsername
	suffix := 1
	tenantDB := WithTenantID(DB, tenantID)

	for {
		var count int64
		tenantDB.Model(&User{}).Where("username = ?", username).Count(&count)
		if count == 0 {
			return username
		}
		suffix++
		username = baseUsername + fmt.Sprintf("_%d", suffix)
	}
}

// GenerateRandomPassword generates a cryptographically secure random password for Zitadel users.
// Since they authenticate via Zitadel, this password won't be used for login.
func GenerateRandomPassword() string {
	const passwordBytes = 32
	b := make([]byte, passwordBytes)
	if _, err := rand.Read(b); err != nil {
		// Fallback: still produce a unique, non-guessable value
		return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// generateAffCode generates a unique affiliate code, consistent with user.go registration flow.
func generateAffCode() string {
	return common.GetRandomString(4)
}
