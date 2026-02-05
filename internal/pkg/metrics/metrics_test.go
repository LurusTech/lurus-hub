package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRecordRelayRequest(t *testing.T) {
	// Record a successful request
	RecordRelayRequest("openai", "gpt-4", "success", 0.5)

	// Verify counter incremented
	count := testutil.ToFloat64(RelayRequestsTotal.WithLabelValues("openai", "gpt-4", "success"))
	if count != 1 {
		t.Errorf("expected count 1, got %f", count)
	}

	// Record an error request
	RecordRelayRequest("openai", "gpt-4", "error", 1.0)
	errorCount := testutil.ToFloat64(RelayRequestsTotal.WithLabelValues("openai", "gpt-4", "error"))
	if errorCount != 1 {
		t.Errorf("expected error count 1, got %f", errorCount)
	}
}

func TestRecordTokens(t *testing.T) {
	RecordTokens("claude", "claude-3", 100, 50)

	inputCount := testutil.ToFloat64(TokensProcessed.WithLabelValues("claude", "claude-3", "input"))
	if inputCount != 100 {
		t.Errorf("expected input tokens 100, got %f", inputCount)
	}

	outputCount := testutil.ToFloat64(TokensProcessed.WithLabelValues("claude", "claude-3", "output"))
	if outputCount != 50 {
		t.Errorf("expected output tokens 50, got %f", outputCount)
	}
}

func TestRecordQuotaConsumed(t *testing.T) {
	RecordQuotaConsumed("tenant1", "user1", 1000)

	quota := testutil.ToFloat64(QuotaConsumed.WithLabelValues("tenant1", "user1"))
	if quota != 1000 {
		t.Errorf("expected quota 1000, got %f", quota)
	}
}

func TestActiveConnections(t *testing.T) {
	initial := testutil.ToFloat64(ActiveConnections)

	ActiveConnections.Inc()
	after := testutil.ToFloat64(ActiveConnections)
	if after != initial+1 {
		t.Errorf("expected %f, got %f", initial+1, after)
	}

	ActiveConnections.Dec()
	final := testutil.ToFloat64(ActiveConnections)
	if final != initial {
		t.Errorf("expected %f, got %f", initial, final)
	}
}
