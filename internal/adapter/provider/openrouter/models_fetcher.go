package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ModelArchitecture mirrors the architecture object in OpenRouter's /v1/models response.
type ModelArchitecture struct {
	Modality         string   `json:"modality"`          // e.g. "text+image->text"
	InputModalities  []string `json:"input_modalities"`  // e.g. ["text","image"]
	OutputModalities []string `json:"output_modalities"` // e.g. ["text"]
	Tokenizer        string   `json:"tokenizer"`
}

// ModelPricing mirrors the pricing object. Values are decimal strings.
// All zero ⇒ free model.
type ModelPricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
	Image      string `json:"image"`
	Request    string `json:"request"`
}

// Model represents a single model entry from OpenRouter's /v1/models endpoint.
type Model struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Created       int64             `json:"created"` // Unix seconds; used as cold-start ranking fallback
	Description   string            `json:"description"`
	ContextLength int               `json:"context_length"`
	Architecture  ModelArchitecture `json:"architecture"`
	Pricing       ModelPricing      `json:"pricing"`
}

// ModelsResponse is the wrapper around the data array.
type ModelsResponse struct {
	Data []Model `json:"data"`
}

// IsFree returns true iff every priced field on the model is zero.
// OpenRouter convention: free models have all prices = "0" and an ID suffix ":free".
// We require both signals to avoid false positives.
func (m *Model) IsFree() bool {
	return isZeroPrice(m.Pricing.Prompt) &&
		isZeroPrice(m.Pricing.Completion) &&
		isZeroPrice(m.Pricing.Image) &&
		isZeroPrice(m.Pricing.Request) &&
		strings.HasSuffix(m.ID, ":free")
}

func isZeroPrice(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	// Accept "0", "0.0", "0.00", etc. — anything that parses to numeric zero.
	for _, ch := range s {
		if ch != '0' && ch != '.' && ch != '-' && ch != '+' {
			return false
		}
	}
	return true
}

// FetchModels calls OpenRouter's /v1/models endpoint and returns the parsed list.
// baseURL should be like "https://openrouter.ai/api" (without trailing slash);
// the caller may pass a custom client (e.g. with a proxy or a test transport).
func FetchModels(ctx context.Context, baseURL string, client *http.Client) ([]Model, error) {
	if client == nil {
		client = http.DefaultClient
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openrouter fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter fetch: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	var parsed ModelsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode body: %w", err)
	}
	return parsed.Data, nil
}
