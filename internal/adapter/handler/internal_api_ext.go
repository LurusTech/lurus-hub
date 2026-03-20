package handler

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/lurus-api/internal/adapter/repo"
	"github.com/QuantumNous/lurus-api/internal/app"
	"github.com/QuantumNous/lurus-api/internal/pkg/common"
	"github.com/gin-gonic/gin"
)

var usernameRegexp = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// InternalLogin is no longer supported — auth is delegated to Zitadel.
// POST /internal/auth/login
func InternalLogin(c *gin.Context) {
	c.JSON(http.StatusGone, gin.H{
		"success":    false,
		"message":    "Password-based login is no longer supported. Use Zitadel OIDC.",
		"error_code": "DEPRECATED",
	})
}

// ===== User CRUD =====

// InternalCreateUser creates a new user via the internal API.
// POST /internal/user
func InternalCreateUser(c *gin.Context) {
	var req struct {
		Username    string `json:"username" binding:"required"`
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		Group       string `json:"group"`
		Quota       int    `json:"quota"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	username := strings.TrimSpace(req.Username)
	if len(username) < 3 || len(username) > 20 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":    false,
			"message":    "Username must be 3-20 characters",
			"error_code": "VALIDATION_FAILED",
		})
		return
	}
	if !usernameRegexp.MatchString(username) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":    false,
			"message":    "Username contains invalid characters",
			"error_code": "VALIDATION_FAILED",
		})
		return
	}

	if req.Email != "" && !strings.Contains(req.Email, "@") {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":    false,
			"message":    "Invalid email format",
			"error_code": "VALIDATION_FAILED",
		})
		return
	}

	// Idempotency check
	idempotencyKey := c.GetHeader("X-Idempotency-Key")
	if idempotencyKey != "" {
		existing := &repo.User{Username: username}
		if err := repo.DB.Where("username = ?", username).First(existing).Error; err == nil && existing.Id > 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data": gin.H{
					"id":           existing.Id,
					"username":     existing.Username,
					"display_name": existing.DisplayName,
					"email":        existing.Email,
					"group":        existing.Group,
					"quota":        existing.Quota,
					"is_duplicate": true,
				},
			})
			return
		}
	}

	var existingCount int64
	repo.DB.Model(&repo.User{}).Where("username = ?", username).Count(&existingCount)
	if existingCount > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"success":    false,
			"message":    "Username already exists",
			"error_code": "USER_EXISTS",
		})
		return
	}

	if req.Email != "" {
		var emailCount int64
		repo.DB.Model(&repo.User{}).Where("email = ?", req.Email).Count(&emailCount)
		if emailCount > 0 {
			c.JSON(http.StatusConflict, gin.H{
				"success":    false,
				"message":    "Email already exists",
				"error_code": "USER_EXISTS",
			})
			return
		}
	}

	group := req.Group
	if group == "" {
		group = "default"
	}

	displayName := req.DisplayName
	if displayName == "" {
		displayName = username
	}

	user := &repo.User{
		Username:    username,
		Email:       req.Email,
		DisplayName: displayName,
		Group:       group,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Quota:       req.Quota,
	}

	if err := repo.DB.Create(user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to create user: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"id":           user.Id,
			"username":     user.Username,
			"display_name": user.DisplayName,
			"email":        user.Email,
			"group":        user.Group,
			"quota":        user.Quota,
		},
	})
}

// InternalDeleteUser deletes a user by ID.
// DELETE /internal/user/:id
func InternalDeleteUser(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil || userId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid user ID",
		})
		return
	}

	user, err := repo.GetUserById(userId, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success":    false,
			"message":    "User not found",
			"error_code": "USER_NOT_FOUND",
		})
		return
	}

	if user.Role >= common.RoleRootUser {
		c.JSON(http.StatusForbidden, gin.H{
			"success":    false,
			"message":    "Cannot delete admin/root user",
			"error_code": "FORBIDDEN",
		})
		return
	}

	if err = repo.DeleteUserById(userId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to delete user: " + err.Error(),
		})
		return
	}

	keyName := c.GetString("internal_api_key_name")
	common.SysLog("Internal API deleted user " + strconv.Itoa(userId) + " via key: " + keyName)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User deleted successfully",
	})
}

// InternalGetUserByZitadelSub returns user by Zitadel subject ID.
// GET /internal/user/by-zitadel-sub/:sub
func InternalGetUserByZitadelSub(c *gin.Context) {
	sub := c.Param("sub")
	if sub == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Zitadel subject ID is required",
		})
		return
	}

	tenantId := c.DefaultQuery("tenant_id", "default")

	user, mapping, err := repo.GetUserByZitadelID(sub, tenantId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success":    false,
			"message":    "User not found for Zitadel sub: " + sub,
			"error_code": "USER_NOT_FOUND",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":           user.Id,
			"username":     user.Username,
			"display_name": user.DisplayName,
			"email":        user.Email,
			"role":         user.Role,
			"status":       user.Status,
			"group":        user.Group,
			"tenant_id":    user.TenantId,
			"mapping": gin.H{
				"id":                 mapping.Id,
				"zitadel_user_id":    mapping.ZitadelUserID,
				"tenant_id":          mapping.TenantID,
				"preferred_username": mapping.PreferredUsername,
			},
		},
	})
}

// ===== User Provisioning =====

// InternalProvisionUser atomically creates a user, identity mapping, and optional initial API token.
// Idempotent: if the Zitadel sub already maps to a user, returns the existing user.
// POST /internal/user/provision
func InternalProvisionUser(c *gin.Context) {
	var req struct {
		ZitadelSub        string `json:"zitadel_sub" binding:"required"`
		Email             string `json:"email" binding:"required"`
		DisplayName       string `json:"display_name"`
		TenantID          string `json:"tenant_id"`
		Group             string `json:"group"`
		InitialQuota      int    `json:"initial_quota"`
		CreateInitialToken bool  `json:"create_initial_token"`
		InitialTokenName  string `json:"initial_token_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	if !strings.Contains(req.Email, "@") {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":    false,
			"message":    "Invalid email format",
			"error_code": "VALIDATION_FAILED",
		})
		return
	}

	tenantId := req.TenantID
	if tenantId == "" {
		tenantId = "default"
	}

	// Idempotency: check if mapping already exists
	existingUser, existingMapping, err := repo.GetUserByZitadelID(req.ZitadelSub, tenantId)
	if err == nil && existingUser != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"user_id":     existingUser.Id,
				"username":    existingUser.Username,
				"email":       existingUser.Email,
				"group":       existingUser.Group,
				"tenant_id":   tenantId,
				"is_existing": true,
				"mapping_id":  existingMapping.Id,
			},
		})
		return
	}

	// Derive a safe username from Zitadel sub
	username := "u_" + req.ZitadelSub
	if len(username) > 20 {
		username = username[:20]
	}
	// Sanitize: only allow alphanumeric + underscore
	sanitized := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, username)
	if sanitized == "" {
		sanitized = "u_provisioned"
	}

	group := req.Group
	if group == "" {
		group = "default"
	}

	displayName := req.DisplayName
	if displayName == "" {
		displayName = strings.Split(req.Email, "@")[0]
	}

	// Begin transaction for atomicity
	tx := repo.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to begin transaction: " + tx.Error.Error(),
		})
		return
	}

	// Ensure unique username within tenant
	finalUsername := sanitized
	suffix := 1
	for {
		var count int64
		tx.Model(&repo.User{}).Where("username = ? AND tenant_id = ?", finalUsername, tenantId).Count(&count)
		if count == 0 {
			break
		}
		suffix++
		base := sanitized
		candidate := fmt.Sprintf("%s_%d", base, suffix)
		if len(candidate) > 20 {
			base = base[:20-len(fmt.Sprintf("_%d", suffix))]
			candidate = fmt.Sprintf("%s_%d", base, suffix)
		}
		finalUsername = candidate
	}

	// Step 1: Create user
	user := &repo.User{
		Username:    finalUsername,
		TenantId:    tenantId,
		Email:       req.Email,
		DisplayName: displayName,
		Group:       group,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Quota:       req.InitialQuota,
	}

	if err := tx.Create(user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to create user: " + err.Error(),
		})
		return
	}

	// Step 2: Create identity mapping
	now := time.Now()
	mapping := &repo.UserIdentityMapping{
		LurusUserID:      user.Id,
		ZitadelUserID:    req.ZitadelSub,
		TenantID:         tenantId,
		Email:            req.Email,
		DisplayName:      displayName,
		PreferredUsername: finalUsername,
		LastSyncAt:       &now,
		IsActive:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := tx.Create(mapping).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to create identity mapping: " + err.Error(),
		})
		return
	}

	// Step 3 (optional): Create initial API token
	var tokenData gin.H
	if req.CreateInitialToken {
		tokenName := req.InitialTokenName
		if tokenName == "" {
			tokenName = "Default Key"
		}

		tokenKey, err := app.GenerateTokenKey()
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Failed to generate token key: " + err.Error(),
			})
			return
		}

		token := &repo.Token{
			UserId:         user.Id,
			TenantId:       tenantId,
			Name:           tokenName,
			Key:            tokenKey,
			CreatedTime:    now.Unix(),
			AccessedTime:   now.Unix(),
			Status:         common.TokenStatusEnabled,
			ExpiredTime:    -1,
			RemainQuota:    req.InitialQuota,
			UnlimitedQuota: req.InitialQuota == 0,
			Group:          group,
		}

		if err := tx.Create(token).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Failed to create initial token: " + err.Error(),
			})
			return
		}

		tokenData = gin.H{
			"id":      token.Id,
			"key":     tokenKey,
			"name":    tokenName,
			"warning": "Please save this key - it will not be shown again.",
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to commit transaction: " + err.Error(),
		})
		return
	}

	keyName := c.GetString("internal_api_key_name")
	common.SysLog(fmt.Sprintf("Internal API provisioned user %d (zitadel_sub=%s) via key: %s", user.Id, req.ZitadelSub, keyName))

	resp := gin.H{
		"user_id":     user.Id,
		"username":    finalUsername,
		"email":       req.Email,
		"group":       group,
		"tenant_id":   tenantId,
		"is_existing": false,
		"mapping_id":  mapping.Id,
	}
	if tokenData != nil {
		resp["token"] = tokenData
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    resp,
	})
}

// ===== Token CRUD =====

// InternalGetUserTokens returns paginated tokens for a user.
// GET /internal/token/user/:id
func InternalGetUserTokens(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil || userId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid user ID",
		})
		return
	}

	if _, err = repo.GetUserById(userId, false); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success":    false,
			"message":    "User not found",
			"error_code": "USER_NOT_FOUND",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	tokens, err := repo.GetAllUserTokens(userId, offset, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get tokens: " + err.Error(),
		})
		return
	}

	total, _ := repo.CountUserTokens(userId)

	for _, t := range tokens {
		t.Clean()
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"tokens":    tokens,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// InternalCreateToken creates a new API token for a user.
// POST /internal/token
func InternalCreateToken(c *gin.Context) {
	var req struct {
		UserId         int    `json:"user_id" binding:"required"`
		Name           string `json:"name" binding:"required"`
		UnlimitedQuota bool   `json:"unlimited_quota"`
		RemainQuota    int    `json:"remain_quota"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	user, err := repo.GetUserById(req.UserId, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success":    false,
			"message":    "User not found",
			"error_code": "USER_NOT_FOUND",
		})
		return
	}

	if user.Status == common.UserStatusDisabled {
		c.JSON(http.StatusForbidden, gin.H{
			"success":    false,
			"message":    "User is disabled",
			"error_code": "USER_DISABLED",
		})
		return
	}

	idempotencyKey := c.GetHeader("X-Idempotency-Key")
	if idempotencyKey != "" {
		var existing repo.Token
		if err := repo.DB.Where("user_id = ? AND name = ?", req.UserId, req.Name).First(&existing).Error; err == nil && existing.Id > 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data": gin.H{
					"id":           existing.Id,
					"name":         existing.Name,
					"is_duplicate": true,
				},
			})
			return
		}
	}

	// Generate a unique token key (BUG FIX: was missing before)
	tokenKey, err := app.GenerateTokenKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to generate token key: " + err.Error(),
		})
		return
	}

	token := &repo.Token{
		UserId:         req.UserId,
		TenantId:       user.TenantId,
		Name:           req.Name,
		Key:            tokenKey,
		UnlimitedQuota: req.UnlimitedQuota,
		RemainQuota:    req.RemainQuota,
		CreatedTime:    time.Now().Unix(),
		AccessedTime:   time.Now().Unix(),
		Status:         common.TokenStatusEnabled,
		ExpiredTime:    -1,
		Group:          user.Group,
	}

	if err = token.Insert(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to create token: " + err.Error(),
		})
		return
	}

	keyName := c.GetString("internal_api_key_name")
	common.SysLog("Internal API created token for user " + strconv.Itoa(req.UserId) + " via key: " + keyName)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"id":      token.Id,
			"key":     tokenKey,
			"name":    token.Name,
			"warning": "Please save this key - it will not be shown again.",
		},
	})
}

// InternalGetToken returns a single token by ID (key field is redacted).
// GET /internal/token/:id
func InternalGetToken(c *gin.Context) {
	tokenId, err := strconv.Atoi(c.Param("id"))
	if err != nil || tokenId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid token ID",
		})
		return
	}

	token, err := repo.GetTokenById(tokenId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success":    false,
			"message":    "Token not found",
			"error_code": "TOKEN_NOT_FOUND",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":                   token.Id,
			"user_id":              token.UserId,
			"tenant_id":            token.TenantId,
			"name":                 token.Name,
			"status":               token.Status,
			"created_time":         token.CreatedTime,
			"accessed_time":        token.AccessedTime,
			"expired_time":         token.ExpiredTime,
			"remain_quota":         token.RemainQuota,
			"used_quota":           token.UsedQuota,
			"unlimited_quota":      token.UnlimitedQuota,
			"model_limits_enabled": token.ModelLimitsEnabled,
			"model_limits":         token.GetModelLimits(),
			"allow_ips":            token.AllowIps,
			"group":                token.Group,
			"cross_group_retry":    token.CrossGroupRetry,
		},
	})
}

// InternalGetTokenUsage returns usage statistics for a token.
// GET /internal/token/:id/usage
func InternalGetTokenUsage(c *gin.Context) {
	tokenId, err := strconv.Atoi(c.Param("id"))
	if err != nil || tokenId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid token ID",
		})
		return
	}

	token, err := repo.GetTokenById(tokenId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success":    false,
			"message":    "Token not found",
			"error_code": "TOKEN_NOT_FOUND",
		})
		return
	}

	expiredAt := token.ExpiredTime
	if expiredAt == -1 {
		expiredAt = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"token_id":        token.Id,
			"name":            token.Name,
			"status":          token.Status,
			"total_granted":   token.RemainQuota + token.UsedQuota,
			"total_used":      token.UsedQuota,
			"total_available": token.RemainQuota,
			"unlimited_quota": token.UnlimitedQuota,
			"expires_at":      expiredAt,
			"last_accessed":   token.AccessedTime,
		},
	})
}

// InternalUpdateToken updates a token's properties.
// PUT /internal/token/:id
func InternalUpdateToken(c *gin.Context) {
	tokenId, err := strconv.Atoi(c.Param("id"))
	if err != nil || tokenId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid token ID",
		})
		return
	}

	token, err := repo.GetTokenById(tokenId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success":    false,
			"message":    "Token not found",
			"error_code": "TOKEN_NOT_FOUND",
		})
		return
	}

	var req struct {
		Name               *string `json:"name"`
		Status             *int    `json:"status"`
		ExpiredTime        *int64  `json:"expired_time"`
		RemainQuota        *int    `json:"remain_quota"`
		UnlimitedQuota     *bool   `json:"unlimited_quota"`
		ModelLimitsEnabled *bool   `json:"model_limits_enabled"`
		ModelLimits        *string `json:"model_limits"`
		AllowIps           *string `json:"allow_ips"`
		Group              *string `json:"group"`
		CrossGroupRetry    *bool   `json:"cross_group_retry"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	if req.Name != nil {
		if err := app.ValidateTokenName(*req.Name); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success":    false,
				"message":    err.Error(),
				"error_code": "VALIDATION_FAILED",
			})
			return
		}
		token.Name = *req.Name
	}
	if req.Status != nil {
		token.Status = *req.Status
	}
	if req.ExpiredTime != nil {
		token.ExpiredTime = *req.ExpiredTime
	}
	if req.RemainQuota != nil {
		token.RemainQuota = *req.RemainQuota
	}
	if req.UnlimitedQuota != nil {
		token.UnlimitedQuota = *req.UnlimitedQuota
	}
	if req.ModelLimitsEnabled != nil {
		token.ModelLimitsEnabled = *req.ModelLimitsEnabled
	}
	if req.ModelLimits != nil {
		token.ModelLimits = *req.ModelLimits
	}
	if req.AllowIps != nil {
		token.AllowIps = req.AllowIps
	}
	if req.Group != nil {
		token.Group = *req.Group
	}
	if req.CrossGroupRetry != nil {
		token.CrossGroupRetry = *req.CrossGroupRetry
	}

	if err := token.Update(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to update token: " + err.Error(),
		})
		return
	}

	keyName := c.GetString("internal_api_key_name")
	common.SysLog(fmt.Sprintf("Internal API updated token %d via key: %s", tokenId, keyName))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Token updated successfully",
		"data": gin.H{
			"id":     token.Id,
			"name":   token.Name,
			"status": token.Status,
		},
	})
}

// InternalDeleteToken deletes (revokes) a token by ID.
// DELETE /internal/token/:id
func InternalDeleteToken(c *gin.Context) {
	tokenId, err := strconv.Atoi(c.Param("id"))
	if err != nil || tokenId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid token ID",
		})
		return
	}

	token, err := repo.GetTokenById(tokenId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success":    false,
			"message":    "Token not found",
			"error_code": "TOKEN_NOT_FOUND",
		})
		return
	}

	if err := token.Delete(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to delete token: " + err.Error(),
		})
		return
	}

	keyName := c.GetString("internal_api_key_name")
	common.SysLog(fmt.Sprintf("Internal API deleted token %d (user=%d) via key: %s", tokenId, token.UserId, keyName))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Token deleted successfully",
	})
}
