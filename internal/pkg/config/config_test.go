package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

// --- Tests for Get() singleton and default values ---

func TestGet_ReturnsNonNilConfig(t *testing.T) {
	cfg := Get()
	if cfg == nil {
		t.Fatal("Get() returned nil, expected a valid *Config")
	}
}

func TestGet_ReturnsSameInstance(t *testing.T) {
	a := Get()
	b := Get()
	if a != b {
		t.Fatalf("Get() returned different pointers: %p vs %p", a, b)
	}
}

// --- Tests for Relay default values ---

func TestGet_RelayDefaults_StreamScannerInitialBuffer(t *testing.T) {
	cfg := Get()
	want := 64 * 1024 // 64KB
	if cfg.Relay.StreamScannerInitialBuffer != want {
		t.Errorf("StreamScannerInitialBuffer = %d, want %d", cfg.Relay.StreamScannerInitialBuffer, want)
	}
}

func TestGet_RelayDefaults_StreamScannerMaxBuffer(t *testing.T) {
	cfg := Get()
	want := 64 * 1024 * 1024 // 64MB
	if cfg.Relay.StreamScannerMaxBuffer != want {
		t.Errorf("StreamScannerMaxBuffer = %d, want %d", cfg.Relay.StreamScannerMaxBuffer, want)
	}
}

func TestGet_RelayDefaults_PingInterval(t *testing.T) {
	cfg := Get()
	want := 10 * time.Second
	if cfg.Relay.PingInterval != want {
		t.Errorf("PingInterval = %s, want %s", cfg.Relay.PingInterval, want)
	}
}

func TestGet_RelayDefaults_WriteTimeout(t *testing.T) {
	cfg := Get()
	want := 10 * time.Second
	if cfg.Relay.WriteTimeout != want {
		t.Errorf("WriteTimeout = %s, want %s", cfg.Relay.WriteTimeout, want)
	}
}

func TestGet_RelayDefaults_MaxPingDuration(t *testing.T) {
	cfg := Get()
	want := 30 * time.Minute
	if cfg.Relay.MaxPingDuration != want {
		t.Errorf("MaxPingDuration = %s, want %s", cfg.Relay.MaxPingDuration, want)
	}
}

func TestGet_RelayDefaults_GoroutineShutdownTimeout(t *testing.T) {
	cfg := Get()
	want := 5 * time.Second
	if cfg.Relay.GoroutineShutdownTimeout != want {
		t.Errorf("GoroutineShutdownTimeout = %s, want %s", cfg.Relay.GoroutineShutdownTimeout, want)
	}
}

func TestGet_RelayDefaults_StopChannelBuffer(t *testing.T) {
	cfg := Get()
	want := 3
	if cfg.Relay.StopChannelBuffer != want {
		t.Errorf("StopChannelBuffer = %d, want %d", cfg.Relay.StopChannelBuffer, want)
	}
}

// --- Tests for Server default values ---

func TestGet_ServerDefaults_GracefulShutdownTimeout(t *testing.T) {
	cfg := Get()
	want := 30 * time.Second
	if cfg.Server.GracefulShutdownTimeout != want {
		t.Errorf("GracefulShutdownTimeout = %s, want %s", cfg.Server.GracefulShutdownTimeout, want)
	}
}

// --- Tests for Security default values ---

func TestGet_SecurityDefaults_SMSCodeExpiration(t *testing.T) {
	cfg := Get()
	want := 5 * time.Minute
	if cfg.Security.SMSCodeExpiration != want {
		t.Errorf("SMSCodeExpiration = %s, want %s", cfg.Security.SMSCodeExpiration, want)
	}
}

// --- Tests for Search default values ---

func TestGet_SearchDefaults_SyncWorkerCount(t *testing.T) {
	cfg := Get()
	want := 32
	if cfg.Search.SyncWorkerCount != want {
		t.Errorf("SyncWorkerCount = %d, want %d", cfg.Search.SyncWorkerCount, want)
	}
}

// --- Tests for Cron default values ---

func TestGet_CronDefaults_QuotaResetCheckInterval(t *testing.T) {
	cfg := Get()
	want := 1 * time.Minute
	if cfg.Cron.QuotaResetCheckInterval != want {
		t.Errorf("QuotaResetCheckInterval = %s, want %s", cfg.Cron.QuotaResetCheckInterval, want)
	}
}

// --- Tests for PrintEffective ---

func TestConfig_PrintEffective_ReturnsNonEmpty(t *testing.T) {
	cfg := Get()
	result := cfg.PrintEffective()
	if result == "" {
		t.Fatal("PrintEffective() returned empty string")
	}
}

func TestConfig_PrintEffective_ContainsExpectedFields(t *testing.T) {
	cfg := Get()
	result := cfg.PrintEffective()

	expectedSubstrings := []string{
		"Config:",
		"relay.ping_interval=",
		"relay.write_timeout=",
		"relay.max_ping_duration=",
		"relay.scanner_buffer=",
	}
	for _, sub := range expectedSubstrings {
		if !strings.Contains(result, sub) {
			t.Errorf("PrintEffective() output missing %q\ngot: %s", sub, result)
		}
	}
}

func TestConfig_PrintEffective_ContainsCorrectValues(t *testing.T) {
	cfg := Get()
	result := cfg.PrintEffective()

	// Verify the formatted values match defaults: 64KB / 64MB
	if !strings.Contains(result, "64KB") {
		t.Errorf("PrintEffective() should contain '64KB', got: %s", result)
	}
	if !strings.Contains(result, "64MB") {
		t.Errorf("PrintEffective() should contain '64MB', got: %s", result)
	}
}

// --- Tests for envInt helper ---

func TestEnvInt_ReturnsDefault_WhenEnvUnset(t *testing.T) {
	const key = "CONFIG_TEST_ENV_INT_UNSET"
	os.Unsetenv(key)

	got := envInt(key, 42)
	if got != 42 {
		t.Errorf("envInt(%q, 42) = %d, want 42", key, got)
	}
}

func TestEnvInt_ReturnsEnvValue_WhenSet(t *testing.T) {
	const key = "CONFIG_TEST_ENV_INT_SET"
	t.Setenv(key, "128")

	got := envInt(key, 42)
	if got != 128 {
		t.Errorf("envInt(%q, 42) = %d, want 128", key, got)
	}
}

func TestEnvInt_ReturnsDefault_WhenEnvInvalid(t *testing.T) {
	const key = "CONFIG_TEST_ENV_INT_INVALID"
	t.Setenv(key, "not-a-number")

	got := envInt(key, 99)
	if got != 99 {
		t.Errorf("envInt(%q, 99) = %d, want 99 (fallback on invalid)", key, got)
	}
}

func TestEnvInt_ReturnsDefault_WhenEnvEmpty(t *testing.T) {
	const key = "CONFIG_TEST_ENV_INT_EMPTY"
	t.Setenv(key, "")

	got := envInt(key, 50)
	if got != 50 {
		t.Errorf("envInt(%q, 50) = %d, want 50 (fallback on empty)", key, got)
	}
}

func TestEnvInt_HandlesZeroValue(t *testing.T) {
	const key = "CONFIG_TEST_ENV_INT_ZERO"
	t.Setenv(key, "0")

	got := envInt(key, 77)
	if got != 0 {
		t.Errorf("envInt(%q, 77) = %d, want 0", key, got)
	}
}

func TestEnvInt_HandlesNegativeValue(t *testing.T) {
	const key = "CONFIG_TEST_ENV_INT_NEGATIVE"
	t.Setenv(key, "-5")

	got := envInt(key, 10)
	if got != -5 {
		t.Errorf("envInt(%q, 10) = %d, want -5", key, got)
	}
}

// --- Tests for envDuration helper ---

func TestEnvDuration_ReturnsDefault_WhenEnvUnset(t *testing.T) {
	const key = "CONFIG_TEST_ENV_DUR_UNSET"
	os.Unsetenv(key)

	got := envDuration(key, 15*time.Second)
	if got != 15*time.Second {
		t.Errorf("envDuration(%q, 15s) = %s, want 15s", key, got)
	}
}

func TestEnvDuration_ParsesDurationString(t *testing.T) {
	const key = "CONFIG_TEST_ENV_DUR_STRING"
	t.Setenv(key, "2m30s")

	got := envDuration(key, time.Second)
	want := 2*time.Minute + 30*time.Second
	if got != want {
		t.Errorf("envDuration(%q, 1s) = %s, want %s", key, got, want)
	}
}

func TestEnvDuration_ParsesIntegerAsSeconds(t *testing.T) {
	const key = "CONFIG_TEST_ENV_DUR_INT"
	t.Setenv(key, "45")

	got := envDuration(key, time.Second)
	want := 45 * time.Second
	if got != want {
		t.Errorf("envDuration(%q, 1s) = %s, want %s", key, got, want)
	}
}

func TestEnvDuration_ReturnsDefault_WhenEnvInvalid(t *testing.T) {
	const key = "CONFIG_TEST_ENV_DUR_INVALID"
	t.Setenv(key, "not-a-duration")

	got := envDuration(key, 20*time.Second)
	if got != 20*time.Second {
		t.Errorf("envDuration(%q, 20s) = %s, want 20s (fallback on invalid)", key, got)
	}
}

func TestEnvDuration_ReturnsDefault_WhenEnvEmpty(t *testing.T) {
	const key = "CONFIG_TEST_ENV_DUR_EMPTY"
	t.Setenv(key, "")

	got := envDuration(key, 10*time.Minute)
	if got != 10*time.Minute {
		t.Errorf("envDuration(%q, 10m) = %s, want 10m (fallback on empty)", key, got)
	}
}

func TestEnvDuration_HandlesMilliseconds(t *testing.T) {
	const key = "CONFIG_TEST_ENV_DUR_MS"
	t.Setenv(key, "500ms")

	got := envDuration(key, time.Second)
	want := 500 * time.Millisecond
	if got != want {
		t.Errorf("envDuration(%q, 1s) = %s, want %s", key, got, want)
	}
}

func TestEnvDuration_HandlesHours(t *testing.T) {
	const key = "CONFIG_TEST_ENV_DUR_HOURS"
	t.Setenv(key, "2h")

	got := envDuration(key, time.Second)
	want := 2 * time.Hour
	if got != want {
		t.Errorf("envDuration(%q, 1s) = %s, want %s", key, got, want)
	}
}

// --- Tests for loadFromEnv with environment overrides ---

func TestLoadFromEnv_OverridesIntField(t *testing.T) {
	t.Setenv("STREAM_SCANNER_INITIAL_BUFFER", "131072") // 128KB
	t.Setenv("RELAY_STOP_CHANNEL_BUFFER", "10")

	cfg := loadFromEnv()

	if cfg.Relay.StreamScannerInitialBuffer != 131072 {
		t.Errorf("StreamScannerInitialBuffer = %d, want 131072", cfg.Relay.StreamScannerInitialBuffer)
	}
	if cfg.Relay.StopChannelBuffer != 10 {
		t.Errorf("StopChannelBuffer = %d, want 10", cfg.Relay.StopChannelBuffer)
	}
}

func TestLoadFromEnv_OverridesDurationField(t *testing.T) {
	t.Setenv("RELAY_PING_INTERVAL", "30s")
	t.Setenv("GRACEFUL_SHUTDOWN_TIMEOUT", "1m")
	t.Setenv("SMS_CODE_EXPIRATION", "10m")

	cfg := loadFromEnv()

	if cfg.Relay.PingInterval != 30*time.Second {
		t.Errorf("PingInterval = %s, want 30s", cfg.Relay.PingInterval)
	}
	if cfg.Server.GracefulShutdownTimeout != 1*time.Minute {
		t.Errorf("GracefulShutdownTimeout = %s, want 1m", cfg.Server.GracefulShutdownTimeout)
	}
	if cfg.Security.SMSCodeExpiration != 10*time.Minute {
		t.Errorf("SMSCodeExpiration = %s, want 10m", cfg.Security.SMSCodeExpiration)
	}
}

func TestLoadFromEnv_OverridesDurationField_IntegerAsSeconds(t *testing.T) {
	t.Setenv("RELAY_WRITE_TIMEOUT", "60")

	cfg := loadFromEnv()

	if cfg.Relay.WriteTimeout != 60*time.Second {
		t.Errorf("WriteTimeout = %s, want 1m0s", cfg.Relay.WriteTimeout)
	}
}

func TestLoadFromEnv_FallsBackOnInvalidValues(t *testing.T) {
	t.Setenv("STREAM_SCANNER_INITIAL_BUFFER", "garbage")
	t.Setenv("RELAY_PING_INTERVAL", "not-a-duration")

	cfg := loadFromEnv()

	if cfg.Relay.StreamScannerInitialBuffer != 64*1024 {
		t.Errorf("StreamScannerInitialBuffer = %d, want %d (default on invalid)",
			cfg.Relay.StreamScannerInitialBuffer, 64*1024)
	}
	if cfg.Relay.PingInterval != 10*time.Second {
		t.Errorf("PingInterval = %s, want 10s (default on invalid)", cfg.Relay.PingInterval)
	}
}

func TestLoadFromEnv_AllFieldsPopulated(t *testing.T) {
	cfg := loadFromEnv()

	// Verify every field has a non-zero value (all defaults are positive).
	checks := []struct {
		name string
		ok   bool
	}{
		{"Server.GracefulShutdownTimeout", cfg.Server.GracefulShutdownTimeout > 0},
		{"Relay.StreamScannerInitialBuffer", cfg.Relay.StreamScannerInitialBuffer > 0},
		{"Relay.StreamScannerMaxBuffer", cfg.Relay.StreamScannerMaxBuffer > 0},
		{"Relay.PingInterval", cfg.Relay.PingInterval > 0},
		{"Relay.WriteTimeout", cfg.Relay.WriteTimeout > 0},
		{"Relay.MaxPingDuration", cfg.Relay.MaxPingDuration > 0},
		{"Relay.GoroutineShutdownTimeout", cfg.Relay.GoroutineShutdownTimeout > 0},
		{"Relay.StopChannelBuffer", cfg.Relay.StopChannelBuffer > 0},
		{"Search.SyncWorkerCount", cfg.Search.SyncWorkerCount > 0},
		{"Cron.QuotaResetCheckInterval", cfg.Cron.QuotaResetCheckInterval > 0},
		{"Security.SMSCodeExpiration", cfg.Security.SMSCodeExpiration > 0},
	}
	for _, c := range checks {
		if !c.ok {
			t.Errorf("%s has zero value, expected positive default", c.name)
		}
	}
}
