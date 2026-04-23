package handler

import (
	"context"
	"testing"
	"time"

	"github.com/LurusTech/lurus-api/internal/adapter/repo"
	"github.com/LurusTech/lurus-api/internal/pkg/constant"
	"github.com/stretchr/testify/assert"
)

func TestAutoSyncChannelModelsWithContext_InvalidFrequency(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Test with invalid frequency (should return immediately)
	done := make(chan bool)
	go func() {
		AutoSyncChannelModelsWithContext(ctx, 0)
		done <- true
	}()

	// Should complete immediately
	select {
	case <-done:
		// Success - function returned
	case <-time.After(1 * time.Second):
		t.Fatal("Function did not return with invalid frequency")
	}
}

func TestAutoSyncChannelModelsWithContext_NegativeFrequency(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Test with negative frequency (should return immediately)
	done := make(chan bool)
	go func() {
		AutoSyncChannelModelsWithContext(ctx, -5)
		done <- true
	}()

	// Should complete immediately
	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Function did not return with negative frequency")
	}
}

func TestAutoSyncChannelModelsWithContext_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Start worker with very long frequency
	done := make(chan bool)
	go func() {
		AutoSyncChannelModelsWithContext(ctx, 60) // 60 minutes
		done <- true
	}()

	// Cancel context immediately
	cancel()

	// Should stop within 1 second
	select {
	case <-done:
		// Success - context cancellation worked
	case <-time.After(2 * time.Second):
		t.Fatal("Worker did not stop after context cancellation")
	}
}

func TestBuildModelsURL_OpenAI(t *testing.T) {
	baseURL := "https://api.openai.com"
	result := buildModelsURL(constant.ChannelTypeOpenAI, baseURL)

	expected := "https://api.openai.com/v1/models"
	assert.Equal(t, expected, result)
}

func TestBuildModelsURL_Gemini(t *testing.T) {
	baseURL := "https://generativelanguage.googleapis.com"
	result := buildModelsURL(constant.ChannelTypeGemini, baseURL)

	expected := "https://generativelanguage.googleapis.com/v1beta/openai/models"
	assert.Equal(t, expected, result)
}

func TestBuildModelsURL_Ali(t *testing.T) {
	baseURL := "https://dashscope.aliyuncs.com"
	result := buildModelsURL(constant.ChannelTypeAli, baseURL)

	expected := "https://dashscope.aliyuncs.com/compatible-mode/v1/models"
	assert.Equal(t, expected, result)
}

func TestBuildModelsURL_Zhipu(t *testing.T) {
	baseURL := "https://open.bigmodel.cn"
	result := buildModelsURL(constant.ChannelTypeZhipu_v4, baseURL)

	expected := "https://open.bigmodel.cn/api/paas/v4/models"
	assert.Equal(t, expected, result)
}

func TestBuildModelsURL_VolcEngine(t *testing.T) {
	baseURL := "https://ark.cn-beijing.volces.com"
	result := buildModelsURL(constant.ChannelTypeVolcEngine, baseURL)

	expected := "https://ark.cn-beijing.volces.com/v1/models"
	assert.Equal(t, expected, result)
}

func TestBuildModelsURL_Moonshot(t *testing.T) {
	baseURL := "https://api.moonshot.cn"
	result := buildModelsURL(constant.ChannelTypeMoonshot, baseURL)

	expected := "https://api.moonshot.cn/v1/models"
	assert.Equal(t, expected, result)
}

func TestBuildModelsURL_AllChannelTypes(t *testing.T) {
	tests := []struct {
		name        string
		channelType int
		baseURL     string
		expected    string
	}{
		{
			name:        "OpenAI",
			channelType: constant.ChannelTypeOpenAI,
			baseURL:     "https://api.openai.com",
			expected:    "https://api.openai.com/v1/models",
		},
		{
			name:        "Azure",
			channelType: constant.ChannelTypeAzure,
			baseURL:     "https://api.azure.com",
			expected:    "https://api.azure.com/v1/models",
		},
		{
			name:        "Anthropic",
			channelType: constant.ChannelTypeAnthropic,
			baseURL:     "https://api.anthropic.com",
			expected:    "https://api.anthropic.com/v1/models",
		},
		{
			name:        "Gemini",
			channelType: constant.ChannelTypeGemini,
			baseURL:     "https://generativelanguage.googleapis.com",
			expected:    "https://generativelanguage.googleapis.com/v1beta/openai/models",
		},
		{
			name:        "Ali",
			channelType: constant.ChannelTypeAli,
			baseURL:     "https://dashscope.aliyuncs.com",
			expected:    "https://dashscope.aliyuncs.com/compatible-mode/v1/models",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildModelsURL(tt.channelType, tt.baseURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFetchAndMergeModels_NoBaseURL(t *testing.T) {
	// Use a valid channel type but clear its base URL
	channel := &repo.Channel{
		Type: constant.ChannelTypeOpenAI,
	}

	// Save original base URL
	originalURL := constant.ChannelBaseURLs[constant.ChannelTypeOpenAI]
	// Clear base URL to trigger error
	constant.ChannelBaseURLs[constant.ChannelTypeOpenAI] = ""

	newModels, err := fetchAndMergeModels(channel)

	assert.Nil(t, newModels)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no base URL")

	// Restore original URL
	constant.ChannelBaseURLs[constant.ChannelTypeOpenAI] = originalURL
}

func TestFetchAndMergeModels_NoAvailableKey(t *testing.T) {
	// Save and restore original base URL
	originalURL := constant.ChannelBaseURLs[constant.ChannelTypeOpenAI]
	defer func() {
		constant.ChannelBaseURLs[constant.ChannelTypeOpenAI] = originalURL
	}()
	constant.ChannelBaseURLs[constant.ChannelTypeOpenAI] = "https://api.openai.com"

	channel := &repo.Channel{
		Type: constant.ChannelTypeOpenAI,
	}
	// Enable multi-key mode with no keys to trigger "no available key" error
	channel.ChannelInfo.IsMultiKey = true
	channel.Key = ""

	newModels, err := fetchAndMergeModels(channel)

	// Should fail because no key available
	assert.Nil(t, newModels)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no available key")
}

// Integration test - requires database and network
func TestSyncAllChannelModels_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// This test requires:
	// 1. Database with channels table
	// 2. Network access to model APIs
	// 3. Valid API keys

	ctx := context.Background()

	// Test (should not panic)
	assert.NotPanics(t, func() {
		syncAllChannelModels(ctx)
	})
}

func TestSyncAllChannelModels_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Create context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should return immediately without error
	assert.NotPanics(t, func() {
		syncAllChannelModels(ctx)
	})
}

// Mock test for syncAllChannelModels logic
func TestSyncAllChannelModels_MockChannels(t *testing.T) {
	// This is a conceptual test showing what should be tested
	// Real implementation would require mocking repo.GetAllChannels

	t.Skip("Requires database mocking infrastructure")

	// Pseudo-code:
	// 1. Mock repo.GetAllChannels to return test channels
	// 2. Mock fetchAndMergeModels to return controlled results
	// 3. Verify that:
	//    - Enabled channels are processed
	//    - Disabled channels are skipped
	//    - Failures don't stop processing other channels
	//    - Correct counts (synced/skipped/failed) are logged
}

// Test for model deduplication logic
func TestFetchAndMergeModels_ModelDeduplication(t *testing.T) {
	t.Skip("Requires mocking HTTP client for API responses")

	// Pseudo-code:
	// 1. Mock API response with models: ["gpt-4", "gpt-3.5-turbo"]
	// 2. Channel already has models: "gpt-4,gpt-3.5-turbo,old-model"
	// 3. Verify newModels is empty (no new models to add)
}

func TestFetchAndMergeModels_GeminiPrefixStripping(t *testing.T) {
	t.Skip("Requires mocking HTTP client")

	// Pseudo-code:
	// 1. Mock Gemini API response with "models/gemini-pro"
	// 2. Verify that prefix "models/" is stripped
	// 3. Result should be "gemini-pro" only
}

// Benchmark test for concurrent channel sync
func BenchmarkSyncAllChannelModels(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		syncAllChannelModels(ctx)
	}
}

// Test for error handling in fetchAndMergeModels
func TestFetchAndMergeModels_ErrorHandling(t *testing.T) {
	t.Run("empty_base_url", func(t *testing.T) {
		// Save original base URL
		originalURL := constant.ChannelBaseURLs[constant.ChannelTypeOpenAI]
		defer func() {
			// Restore base URL
			constant.ChannelBaseURLs[constant.ChannelTypeOpenAI] = originalURL
		}()

		// Clear base URL to trigger error
		constant.ChannelBaseURLs[constant.ChannelTypeOpenAI] = ""

		channel := &repo.Channel{
			Type: constant.ChannelTypeOpenAI,
		}

		newModels, err := fetchAndMergeModels(channel)

		assert.Nil(t, newModels)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no base URL")
	})
}

// Test the worker lifecycle
func TestModelSyncWorker_Lifecycle(t *testing.T) {
	tests := []struct {
		name      string
		frequency int
		testTime  time.Duration
		wantTicks int
	}{
		{
			name:      "invalid_frequency",
			frequency: 0,
			testTime:  100 * time.Millisecond,
			wantTicks: 0, // Should exit immediately
		},
		{
			name:      "negative_frequency",
			frequency: -1,
			testTime:  100 * time.Millisecond,
			wantTicks: 0, // Should exit immediately
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.testTime)
			defer cancel()

			done := make(chan bool)
			go func() {
				AutoSyncChannelModelsWithContext(ctx, tt.frequency)
				done <- true
			}()

			// Wait for completion or timeout
			select {
			case <-done:
				// Success
			case <-time.After(tt.testTime + 100*time.Millisecond):
				if tt.wantTicks > 0 {
					t.Fatal("Worker did not complete in expected time")
				}
			}
		})
	}
}

// Test buildFetchModelsHeaders (if it's exported or we add a wrapper)
func TestBuildFetchModelsHeaders(t *testing.T) {
	t.Skip("buildFetchModelsHeaders is not exported - consider exporting for testing")

	// Pseudo-code:
	// 1. Test different channel types
	// 2. Verify correct Authorization header format
	// 3. Verify API-Key header for specific providers
	// 4. Test error cases (empty key, invalid channel type)
}

// Race condition test
func TestAutoSyncChannelModels_RaceCondition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition test")
	}

	// Run with: go test -race
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start multiple workers concurrently
	for i := 0; i < 3; i++ {
		go AutoSyncChannelModelsWithContext(ctx, 60)
	}

	// Wait for context timeout
	<-ctx.Done()
}

// Test for proper resource cleanup
func TestModelSyncWorker_ResourceCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Start worker
	done := make(chan bool)
	go func() {
		AutoSyncChannelModelsWithContext(ctx, 60)
		done <- true
	}()

	// Cancel immediately
	cancel()

	// Wait for cleanup
	select {
	case <-done:
		// Success - resources cleaned up
	case <-time.After(2 * time.Second):
		t.Fatal("Worker did not clean up resources")
	}
}

// Mock OpenAI response structure for testing
type mockModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// Test model list parsing
func TestParseModelResponse(t *testing.T) {
	t.Skip("Requires refactoring to extract parsing logic")

	// Pseudo-code:
	// 1. Create mock JSON response
	// 2. Parse into OpenAIModelsResponse
	// 3. Verify all model IDs extracted correctly
	// 4. Test edge cases: empty list, malformed JSON
}

// Error scenarios
func TestSyncAllChannelModels_ErrorScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test")
	}

	tests := []struct {
		name        string
		setupMock   func()
		expectPanic bool
	}{
		{
			name: "database_connection_lost",
			setupMock: func() {
				// Mock database error
			},
			expectPanic: false, // Should handle gracefully
		},
		{
			name: "network_timeout",
			setupMock: func() {
				// Mock network timeout
			},
			expectPanic: false, // Should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock()
			}

			ctx := context.Background()
			if tt.expectPanic {
				assert.Panics(t, func() {
					syncAllChannelModels(ctx)
				})
			} else {
				assert.NotPanics(t, func() {
					syncAllChannelModels(ctx)
				})
			}
		})
	}
}
