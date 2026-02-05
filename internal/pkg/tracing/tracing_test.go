package tracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestLoadConfigFromEnv_Defaults(t *testing.T) {
	cfg := LoadConfigFromEnv()

	if cfg.Enabled {
		t.Error("expected tracing to be disabled by default")
	}
	if cfg.Endpoint != "localhost:4318" {
		t.Errorf("expected default endpoint 'localhost:4318', got %s", cfg.Endpoint)
	}
	if !cfg.Insecure {
		t.Error("expected insecure to be true by default")
	}
	if cfg.SampleRate != 1.0 {
		t.Errorf("expected sample rate 1.0, got %f", cfg.SampleRate)
	}
}

func TestInit_Disabled(t *testing.T) {
	cfg := Config{
		Enabled: false,
	}

	err := Init(context.Background(), cfg)
	if err != nil {
		t.Errorf("Init with disabled config should not return error: %v", err)
	}

	if IsEnabled() {
		t.Error("expected IsEnabled to return false when tracing is disabled")
	}
}

func TestGetTraceID_NoSpan(t *testing.T) {
	ctx := context.Background()
	traceID := GetTraceID(ctx)

	if traceID != "" {
		t.Errorf("expected empty trace ID for context without span, got %s", traceID)
	}
}

func TestMiddleware_Disabled(t *testing.T) {
	// Ensure tracing is disabled
	enabled = false

	router := gin.New()
	router.Use(Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// No trace ID header when disabled
	if w.Header().Get(TraceIDHeader) != "" {
		t.Error("expected no X-Trace-Id header when tracing is disabled")
	}
}

func TestGetTraceIDFromContext_Empty(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		traceID := GetTraceIDFromContext(c)
		if traceID != "" {
			c.String(http.StatusBadRequest, "unexpected trace ID")
			return
		}
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestRelaySpan_Disabled(t *testing.T) {
	enabled = false

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		span, end := RelaySpan(c, "test")
		defer end()

		if span != nil {
			c.String(http.StatusBadRequest, "expected nil span when disabled")
			return
		}
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestStartChannelSelectSpan_Disabled(t *testing.T) {
	enabled = false

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		endSpan := StartChannelSelectSpan(c)
		endSpan(1, nil) // Should not panic

		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestStartUpstreamSpan_Disabled(t *testing.T) {
	enabled = false

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		endSpan := StartUpstreamSpan(c, "openai", "https://api.openai.com")
		endSpan(200, nil) // Should not panic

		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestStartAuthSpan_Disabled(t *testing.T) {
	enabled = false

	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		endSpan := StartAuthSpan(c)
		endSpan(1, nil) // Should not panic

		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}
