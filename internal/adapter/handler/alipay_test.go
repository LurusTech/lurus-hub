package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/QuantumNous/lurus-api/internal/pkg/common"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupAlipayTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Setup session middleware
	store := cookie.NewStore([]byte("test-secret-key"))
	router.Use(sessions.Sessions("test-session", store))

	return router
}

func TestGetAlipayClient_MissingConfig(t *testing.T) {
	// Save original values
	originalAppId := common.AlipayAppId
	originalPrivateKey := os.Getenv("ALIPAY_PRIVATE_KEY")

	// Clear config
	common.AlipayAppId = ""
	os.Setenv("ALIPAY_PRIVATE_KEY", "")

	// Reset client to force re-initialization
	resetAlipayClient()

	// Test
	client, err := getAlipayClient()

	// Assertions
	assert.Nil(t, client)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")

	// Restore
	common.AlipayAppId = originalAppId
	os.Setenv("ALIPAY_PRIVATE_KEY", originalPrivateKey)
	resetAlipayClient()
}

func TestGetAlipayClient_ValidConfig(t *testing.T) {
	// Skip if no real credentials
	if os.Getenv("ALIPAY_PRIVATE_KEY") == "" {
		t.Skip("ALIPAY_PRIVATE_KEY not set, skipping")
	}

	// Save original
	originalAppId := common.AlipayAppId

	// Set test config
	common.AlipayAppId = "test-app-id"
	resetAlipayClient()

	// Test
	client, err := getAlipayClient()

	// Assertions
	assert.NotNil(t, client)
	assert.NoError(t, err)

	// Restore
	common.AlipayAppId = originalAppId
	resetAlipayClient()
}

func TestAlipayOAuth_MissingState(t *testing.T) {
	router := setupAlipayTestRouter()
	router.GET("/oauth/alipay", AlipayOAuth)

	// Test request without state
	req := httptest.NewRequest(http.MethodGet, "/oauth/alipay?code=test-code", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "state is empty")
}

func TestAlipayOAuth_InvalidState(t *testing.T) {
	router := setupAlipayTestRouter()
	router.GET("/oauth/alipay", func(c *gin.Context) {
		// Set different state in session
		session := sessions.Default(c)
		session.Set("oauth_state", "valid-state")
		session.Save()

		AlipayOAuth(c)
	})

	// Test request with wrong state
	req := httptest.NewRequest(http.MethodGet, "/oauth/alipay?state=wrong-state&code=test-code", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "not same")
}

func TestAlipayOAuth_DisabledByAdmin(t *testing.T) {
	router := setupAlipayTestRouter()
	router.GET("/oauth/alipay", func(c *gin.Context) {
		// Set valid state
		session := sessions.Default(c)
		session.Set("oauth_state", "test-state")
		session.Save()

		AlipayOAuth(c)
	})

	// Save original value
	originalEnabled := common.AlipayOAuthEnabled
	common.AlipayOAuthEnabled = false

	// Test
	req := httptest.NewRequest(http.MethodGet, "/oauth/alipay?state=test-state&code=test-code", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "未开启通过支付宝登录")

	// Restore
	common.AlipayOAuthEnabled = originalEnabled
}

func TestAlipayOAuth_EmptyAuthCode(t *testing.T) {
	router := setupAlipayTestRouter()
	router.GET("/oauth/alipay", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("oauth_state", "test-state")
		session.Save()

		AlipayOAuth(c)
	})

	// Enable OAuth
	originalEnabled := common.AlipayOAuthEnabled
	common.AlipayOAuthEnabled = true

	// Test without code parameter
	req := httptest.NewRequest(http.MethodGet, "/oauth/alipay?state=test-state", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"success":false`)

	// Restore
	common.AlipayOAuthEnabled = originalEnabled
}

func TestAlipayBind_NotLoggedIn(t *testing.T) {
	router := setupAlipayTestRouter()
	router.GET("/bind/alipay", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("oauth_state", "test-state")
		session.Save()

		AlipayBind(c)
	})

	// Enable OAuth
	originalEnabled := common.AlipayOAuthEnabled
	common.AlipayOAuthEnabled = true

	// Test without user session (no id in session)
	req := httptest.NewRequest(http.MethodGet, "/bind/alipay?state=test-state&code=test-code", nil)
	w := httptest.NewRecorder()

	// With P1 fix: should return 401 Unauthorized instead of panic
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Should return 401 when not logged in")
	assert.Contains(t, w.Body.String(), "未登录", "Error message should mention not logged in")

	// Restore
	common.AlipayOAuthEnabled = originalEnabled
}

func TestAlipayBind_DisabledByAdmin(t *testing.T) {
	router := setupAlipayTestRouter()
	router.GET("/bind/alipay", AlipayBind)

	// Disable OAuth
	originalEnabled := common.AlipayOAuthEnabled
	common.AlipayOAuthEnabled = false

	// Test
	req := httptest.NewRequest(http.MethodGet, "/bind/alipay?code=test-code", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "未开启通过支付宝登录")

	// Restore
	common.AlipayOAuthEnabled = originalEnabled
}

func TestGetAlipayUserInfoByCode_EmptyCode(t *testing.T) {
	userId, nickname, err := getAlipayUserInfoByCode(context.Background(), "")

	assert.Empty(t, userId)
	assert.Empty(t, nickname)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth_code is empty")
}

func TestGetAlipayUserInfoByCode_ClientNotConfigured(t *testing.T) {
	// Clear config
	originalAppId := common.AlipayAppId
	common.AlipayAppId = ""
	os.Setenv("ALIPAY_PRIVATE_KEY", "")
	resetAlipayClient()

	// Test
	userId, nickname, err := getAlipayUserInfoByCode(context.Background(), "test-code")

	// Assertions
	assert.Empty(t, userId)
	assert.Empty(t, nickname)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")

	// Restore
	common.AlipayAppId = originalAppId
	resetAlipayClient()
}

func TestResetAlipayClient(t *testing.T) {
	// Setup
	common.AlipayAppId = "test-app-id"
	os.Setenv("ALIPAY_PRIVATE_KEY", "test-key")

	// Initialize client (will fail but that's ok)
	_, _ = getAlipayClient()

	// Verify client was initialized
	assert.NotNil(t, alipayClientOnce)

	// Reset
	resetAlipayClient()

	// Verify reset worked
	assert.Nil(t, alipayClient)
	assert.Nil(t, alipayClientErr)

	// Cleanup
	os.Unsetenv("ALIPAY_PRIVATE_KEY")
}

// Integration test (requires database)
func TestAlipayOAuth_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// This test requires:
	// 1. Database connection
	// 2. Mock Alipay API server
	// 3. Valid Alipay credentials

	// TODO: Implement when test infrastructure is ready
	t.Skip("Integration test not implemented yet")
}

// Table-driven test for AlipayOAuth scenarios
func TestAlipayOAuth_Scenarios(t *testing.T) {
	tests := []struct {
		name           string
		setupSession   func(sessions.Session)
		queryParams    string
		alipayEnabled  bool
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "missing_state",
			setupSession: func(s sessions.Session) {
				// No state in session
			},
			queryParams:    "?code=test",
			alipayEnabled:  true,
			expectedStatus: http.StatusForbidden,
			expectedBody:   "state is empty",
		},
		{
			name: "oauth_disabled",
			setupSession: func(s sessions.Session) {
				s.Set("oauth_state", "test-state")
			},
			queryParams:    "?state=test-state&code=test",
			alipayEnabled:  false,
			expectedStatus: http.StatusOK,
			expectedBody:   "未开启通过支付宝登录",
		},
		{
			name: "empty_code",
			setupSession: func(s sessions.Session) {
				s.Set("oauth_state", "test-state")
			},
			queryParams:    "?state=test-state",
			alipayEnabled:  true,
			expectedStatus: http.StatusOK,
			expectedBody:   `"success":false`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupAlipayTestRouter()
			router.GET("/oauth/alipay", func(c *gin.Context) {
				session := sessions.Default(c)
				tt.setupSession(session)
				session.Save()

				AlipayOAuth(c)
			})

			// Set OAuth enabled state
			originalEnabled := common.AlipayOAuthEnabled
			common.AlipayOAuthEnabled = tt.alipayEnabled
			defer func() { common.AlipayOAuthEnabled = originalEnabled }()

			// Test
			req := httptest.NewRequest(http.MethodGet, "/oauth/alipay"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code, "Status code mismatch")
			assert.Contains(t, w.Body.String(), tt.expectedBody, "Response body mismatch")
		})
	}
}
