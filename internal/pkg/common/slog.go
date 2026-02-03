package common

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Global slog logger instance
var (
	slogLogger     *slog.Logger
	slogOnce       sync.Once
	slogLevel      = new(slog.LevelVar) // Dynamic log level
	slogWriter     io.Writer
	slogErrWriter  io.Writer
	slogMu         sync.RWMutex
)

// SlogConfig holds configuration for the slog logger
type SlogConfig struct {
	// Level is the minimum log level (default: Info)
	Level slog.Level
	// Writer is the output writer for info/debug logs (default: os.Stdout)
	Writer io.Writer
	// ErrWriter is the output writer for warn/error logs (default: os.Stderr)
	ErrWriter io.Writer
	// AddSource adds source file information to logs
	AddSource bool
	// JSONFormat uses JSON output format instead of text
	JSONFormat bool
	// TimeFormat is the time format string (default: "2006/01/02 - 15:04:05")
	TimeFormat string
}

// DefaultSlogConfig returns the default configuration
func DefaultSlogConfig() *SlogConfig {
	return &SlogConfig{
		Level:      slog.LevelInfo,
		Writer:     os.Stdout,
		ErrWriter:  os.Stderr,
		AddSource:  false,
		JSONFormat: false,
		TimeFormat: "2006/01/02 - 15:04:05",
	}
}

// SlogConfigFromEnv creates a SlogConfig from environment variables.
// Supported env vars:
//   - LOG_FORMAT: "json" or "text" (default: "text"; auto-selects "json" when GIN_MODE=release)
//   - LOG_LEVEL: "debug", "info", "warn", "error" (default: "info")
func SlogConfigFromEnv() *SlogConfig {
	cfg := DefaultSlogConfig()

	// Determine log format
	format := strings.ToLower(os.Getenv("LOG_FORMAT"))
	switch format {
	case "json":
		cfg.JSONFormat = true
	case "text":
		cfg.JSONFormat = false
	default:
		// Auto-select JSON in release mode
		if os.Getenv("GIN_MODE") == "release" {
			cfg.JSONFormat = true
		}
	}

	// Determine log level
	level := strings.ToLower(os.Getenv("LOG_LEVEL"))
	switch level {
	case "debug":
		cfg.Level = slog.LevelDebug
	case "warn", "warning":
		cfg.Level = slog.LevelWarn
	case "error":
		cfg.Level = slog.LevelError
	default:
		cfg.Level = slog.LevelInfo
	}

	return cfg
}

// customHandler wraps slog.Handler to provide custom formatting
type customHandler struct {
	slog.Handler
	writer     io.Writer
	errWriter  io.Writer
	timeFormat string
	mu         *sync.Mutex
}

func (h *customHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Determine output writer based on level
	w := h.writer
	if r.Level >= slog.LevelWarn {
		w = h.errWriter
	}

	// Format level tag
	var levelTag string
	switch r.Level {
	case slog.LevelDebug:
		levelTag = "DEBUG"
	case slog.LevelInfo:
		levelTag = "INFO"
	case slog.LevelWarn:
		levelTag = "WARN"
	case slog.LevelError:
		levelTag = "ERR"
	default:
		levelTag = r.Level.String()
	}

	// Build the log line
	timeStr := r.Time.Format(h.timeFormat)

	// Get request ID from context if present
	requestID := "SYSTEM"
	if ctx != nil {
		if id := ctx.Value(RequestIdKey); id != nil {
			requestID = fmt.Sprintf("%v", id)
		}
	}

	// Collect attributes
	var attrs []string
	r.Attrs(func(a slog.Attr) bool {
		if a.Key != "" && a.Key != "msg" {
			attrs = append(attrs, fmt.Sprintf("%s=%v", a.Key, a.Value.Any()))
		}
		return true
	})

	// Format message
	msg := r.Message
	if len(attrs) > 0 {
		msg = fmt.Sprintf("%s | %s", msg, strings.Join(attrs, " "))
	}

	// Write formatted log line
	_, err := fmt.Fprintf(w, "[%s] %s | %s | %s\n", levelTag, timeStr, requestID, msg)
	return err
}

func (h *customHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &customHandler{
		Handler:    h.Handler.WithAttrs(attrs),
		writer:     h.writer,
		errWriter:  h.errWriter,
		timeFormat: h.timeFormat,
		mu:         h.mu,
	}
}

func (h *customHandler) WithGroup(name string) slog.Handler {
	return &customHandler{
		Handler:    h.Handler.WithGroup(name),
		writer:     h.writer,
		errWriter:  h.errWriter,
		timeFormat: h.timeFormat,
		mu:         h.mu,
	}
}

// InitSlog initializes the global slog logger with the given config
func InitSlog(cfg *SlogConfig) {
	if cfg == nil {
		cfg = DefaultSlogConfig()
	}

	slogMu.Lock()
	defer slogMu.Unlock()

	slogLevel.Set(cfg.Level)
	slogWriter = cfg.Writer
	slogErrWriter = cfg.ErrWriter

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     slogLevel,
		AddSource: cfg.AddSource,
	}

	if cfg.JSONFormat {
		handler = slog.NewJSONHandler(cfg.Writer, opts)
	} else {
		// Use custom handler for text format
		baseHandler := slog.NewTextHandler(cfg.Writer, opts)
		handler = &customHandler{
			Handler:    baseHandler,
			writer:     cfg.Writer,
			errWriter:  cfg.ErrWriter,
			timeFormat: cfg.TimeFormat,
			mu:         &sync.Mutex{},
		}
	}

	slogLogger = slog.New(handler)
	slog.SetDefault(slogLogger)
}

// ensureSlogInit ensures the slog logger is initialized
func ensureSlogInit() {
	slogOnce.Do(func() {
		if slogLogger == nil {
			InitSlog(nil)
		}
	})
}

// GetSlogLogger returns the global slog logger
func GetSlogLogger() *slog.Logger {
	ensureSlogInit()
	return slogLogger
}

// SetSlogLevel dynamically sets the log level
func SetSlogLevel(level slog.Level) {
	ensureSlogInit()
	slogLevel.Set(level)
}

// SetSlogWriter sets the output writer for logs
func SetSlogWriter(w io.Writer) {
	slogMu.Lock()
	defer slogMu.Unlock()
	slogWriter = w
}

// SetSlogErrWriter sets the output writer for error logs
func SetSlogErrWriter(w io.Writer) {
	slogMu.Lock()
	defer slogMu.Unlock()
	slogErrWriter = w
}

// Structured logging functions that work with context

// LogInfo logs an info message with optional key-value pairs
func LogInfo(ctx context.Context, msg string, args ...any) {
	ensureSlogInit()
	slogLogger.InfoContext(ctx, msg, args...)
}

// LogWarn logs a warning message with optional key-value pairs
func LogWarn(ctx context.Context, msg string, args ...any) {
	ensureSlogInit()
	slogLogger.WarnContext(ctx, msg, args...)
}

// LogError logs an error message with optional key-value pairs
func LogError(ctx context.Context, msg string, args ...any) {
	ensureSlogInit()
	slogLogger.ErrorContext(ctx, msg, args...)
}

// LogDebug logs a debug message with optional key-value pairs
func LogDebug(ctx context.Context, msg string, args ...any) {
	if !DebugEnabled {
		return
	}
	ensureSlogInit()
	slogLogger.DebugContext(ctx, msg, args...)
}

// LogWithSource logs a message with source file information
func LogWithSource(ctx context.Context, level slog.Level, msg string, args ...any) {
	ensureSlogInit()
	// Get caller information
	_, file, line, ok := runtime.Caller(1)
	if ok {
		args = append(args, "source", fmt.Sprintf("%s:%d", file, line))
	}
	slogLogger.Log(ctx, level, msg, args...)
}

// Helper functions for common patterns

// LogErrorf logs a formatted error message
func LogErrorf(ctx context.Context, format string, args ...any) {
	LogError(ctx, fmt.Sprintf(format, args...))
}

// LogInfof logs a formatted info message
func LogInfof(ctx context.Context, format string, args ...any) {
	LogInfo(ctx, fmt.Sprintf(format, args...))
}

// LogWarnf logs a formatted warning message
func LogWarnf(ctx context.Context, format string, args ...any) {
	LogWarn(ctx, fmt.Sprintf(format, args...))
}

// LogDebugf logs a formatted debug message
func LogDebugf(ctx context.Context, format string, args ...any) {
	if DebugEnabled {
		LogDebug(ctx, fmt.Sprintf(format, args...))
	}
}

// Background returns a background context with optional request ID
func Background() context.Context {
	return context.Background()
}

// WithRequestID returns a context with request ID attached
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIdKey, requestID)
}

// Timer returns a function that logs the elapsed time when called
func Timer(ctx context.Context, name string) func() {
	start := time.Now()
	return func() {
		elapsed := time.Since(start)
		LogDebug(ctx, fmt.Sprintf("%s took %v", name, elapsed))
	}
}
