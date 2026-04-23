package tracing

import (
	"github.com/LurusTech/lurus-api/internal/pkg/common"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	// TraceIDHeader is the response header containing the trace ID
	TraceIDHeader = "X-Trace-Id"
	// TraceIDKey is the context key for trace ID
	TraceIDKey = "trace_id"
)

// Middleware returns a Gin middleware that creates spans for HTTP requests
// and adds trace ID to response headers and context
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !IsEnabled() {
			c.Next()
			return
		}

		// Start span
		ctx, span := StartSpan(c.Request.Context(), c.Request.Method+" "+c.FullPath(),
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		// Set request attributes
		span.SetAttributes(
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.url", c.Request.URL.String()),
			attribute.String("http.path", c.FullPath()),
			attribute.String("http.client_ip", c.ClientIP()),
			attribute.String("http.user_agent", c.Request.UserAgent()),
		)

		// Get trace ID and add to context/response
		traceID := GetTraceID(ctx)
		if traceID != "" {
			c.Set(TraceIDKey, traceID)
			c.Header(TraceIDHeader, traceID)
		}

		// Update request context
		c.Request = c.Request.WithContext(ctx)

		// Process request
		c.Next()

		// Record response status
		statusCode := c.Writer.Status()
		span.SetAttributes(
			attribute.Int("http.status_code", statusCode),
			attribute.Int("http.response_size", c.Writer.Size()),
		)

		// Mark span as error if status >= 400
		if statusCode >= 400 {
			span.SetStatus(codes.Error, "HTTP error")
		}

		// Record any errors from context
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				span.RecordError(e.Err)
			}
		}
	}
}

// RelaySpan creates a span for relay operations
func RelaySpan(c *gin.Context, operation string) (trace.Span, func()) {
	if !IsEnabled() {
		return nil, func() {}
	}

	ctx := c.Request.Context()
	_, span := StartSpan(ctx, "relay."+operation,
		trace.WithSpanKind(trace.SpanKindInternal),
	)

	return span, func() { span.End() }
}

// SetRelayAttributes sets common relay attributes on context
func SetRelayAttributes(c *gin.Context, provider, model string, channelID int) {
	if !IsEnabled() {
		return
	}

	ctx := c.Request.Context()
	SetSpanAttributes(ctx,
		attribute.String("relay.provider", provider),
		attribute.String("relay.model", model),
		attribute.Int("relay.channel_id", channelID),
	)
}

// StartChannelSelectSpan starts a span for channel selection
func StartChannelSelectSpan(c *gin.Context) func(selectedChannelID int, err error) {
	if !IsEnabled() {
		return func(int, error) {}
	}

	ctx := c.Request.Context()
	_, span := StartSpan(ctx, "relay.channel_select",
		trace.WithSpanKind(trace.SpanKindInternal),
	)

	return func(selectedChannelID int, err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "channel selection failed")
		} else {
			span.SetAttributes(attribute.Int("channel.selected_id", selectedChannelID))
		}
		span.End()
	}
}

// StartUpstreamSpan starts a span for upstream API calls
func StartUpstreamSpan(c *gin.Context, provider, endpoint string) func(statusCode int, err error) {
	if !IsEnabled() {
		return func(int, error) {}
	}

	ctx := c.Request.Context()
	_, span := StartSpan(ctx, "relay.upstream."+provider,
		trace.WithSpanKind(trace.SpanKindClient),
	)

	span.SetAttributes(
		attribute.String("upstream.provider", provider),
		attribute.String("upstream.endpoint", endpoint),
	)

	return func(statusCode int, err error) {
		span.SetAttributes(attribute.Int("upstream.status_code", statusCode))
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "upstream request failed")
		}
		span.End()
	}
}

// StartAuthSpan starts a span for authentication
func StartAuthSpan(c *gin.Context) func(userID int, err error) {
	if !IsEnabled() {
		return func(int, error) {}
	}

	ctx := c.Request.Context()
	_, span := StartSpan(ctx, "auth.validate",
		trace.WithSpanKind(trace.SpanKindInternal),
	)

	return func(userID int, err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "authentication failed")
		} else {
			span.SetAttributes(attribute.Int("auth.user_id", userID))
		}
		span.End()
	}
}

// GetTraceIDFromContext gets the trace ID from Gin context
func GetTraceIDFromContext(c *gin.Context) string {
	if traceID, exists := c.Get(TraceIDKey); exists {
		if id, ok := traceID.(string); ok {
			return id
		}
	}
	return ""
}

// InjectTraceIDToLogger adds trace_id to the request ID for logging
func InjectTraceIDToLogger(c *gin.Context) {
	traceID := GetTraceIDFromContext(c)
	if traceID == "" {
		return
	}

	// Append trace ID to existing request ID or set it
	existingRequestID := c.GetString(common.RequestIdKey)
	if existingRequestID != "" {
		// Don't replace, just ensure trace ID is available separately
		c.Set("trace_id", traceID)
	} else {
		c.Set(common.RequestIdKey, traceID[:16]) // Use first 16 chars as request ID
		c.Set("trace_id", traceID)
	}
}
