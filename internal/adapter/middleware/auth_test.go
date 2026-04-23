package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/LurusTech/lurus-api/internal/pkg/common"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
	// Redis must be disabled: the Redis client is not initialized in unit tests,
	// and calling RDB.HGetAll on a nil client panics.
	common.RedisEnabled = false
}

// buildAuthRouter creates a test router that injects session values before
// running UserAuth() middleware. Using id=0 avoids DB lookups in GetUserById.
func buildAuthRouter(sessionValues map[string]interface{}) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	store := cookie.NewStore([]byte("auth-test-secret"))
	r.Use(sessions.Sessions("session", store))

	// Session injector middleware: sets values before the auth check
	r.Use(func(c *gin.Context) {
		s := sessions.Default(c)
		for k, v := range sessionValues {
			s.Set(k, v)
		}
		s.Save()
		c.Next()
	})

	r.GET("/test", UserAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	return r
}

// TestAuthHelper_LurusApiUserHeader tests the optional lurus-api-User header
// behavior introduced by P1-4 security fix.
func TestAuthHelper_LurusApiUserHeader(t *testing.T) {
	// Common session values used for authenticated user.
	// id=0 is intentional: GetUserById(0) returns early with an error (no DB panic),
	// and the lurus-api-User header uses "0" to match.
	validSession := map[string]interface{}{
		"username": "testuser",
		"role":     common.RoleCommonUser,
		"id":       0,
		"status":   common.UserStatusEnabled,
	}

	t.Run("header_absent_succeeds", func(t *testing.T) {
		r := buildAuthRouter(validSession)

		req := httptest.NewRequest("GET", "/test", nil)
		// No lurus-api-User header - must succeed normally
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d (missing header must not cause rejection)", w.Code, http.StatusOK)
		}
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["success"] != true {
			t.Errorf("response success = %v, want true", resp["success"])
		}
	})

	t.Run("header_matches_session_succeeds", func(t *testing.T) {
		r := buildAuthRouter(validSession)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("lurus-api-User", strconv.Itoa(0)) // matches session id=0
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d (matching header must pass)", w.Code, http.StatusOK)
		}
	})

	t.Run("header_mismatches_session_rejects", func(t *testing.T) {
		r := buildAuthRouter(validSession)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("lurus-api-User", "9999") // does NOT match session id=0
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d (mismatched header must be rejected)", w.Code, http.StatusUnauthorized)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["success"] != false {
			t.Errorf("response success = %v, want false", resp["success"])
		}
	})

	t.Run("header_invalid_format_rejects", func(t *testing.T) {
		r := buildAuthRouter(validSession)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("lurus-api-User", "not-a-number") // non-integer format
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d (invalid format must be rejected)", w.Code, http.StatusUnauthorized)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["success"] != false {
			t.Errorf("response success = %v, want false", resp["success"])
		}
	})
}
