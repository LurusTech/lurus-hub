package openrouter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsFree(t *testing.T) {
	tests := []struct {
		name string
		m    Model
		want bool
	}{
		{
			name: "free model with :free suffix and all zero prices",
			m: Model{
				ID:      "meta-llama/llama-3.3-70b-instruct:free",
				Pricing: ModelPricing{Prompt: "0", Completion: "0", Image: "0", Request: "0"},
			},
			want: true,
		},
		{
			name: "free pricing but no :free suffix — reject",
			m: Model{
				ID:      "openai/gpt-4o-mini",
				Pricing: ModelPricing{Prompt: "0", Completion: "0"},
			},
			want: false,
		},
		{
			name: ":free suffix but non-zero completion — reject",
			m: Model{
				ID:      "weird/model:free",
				Pricing: ModelPricing{Prompt: "0", Completion: "0.0001"},
			},
			want: false,
		},
		{
			name: "decimal zeros and empty fields are zero",
			m: Model{
				ID:      "x/y:free",
				Pricing: ModelPricing{Prompt: "0.00", Completion: "0", Image: "", Request: ""},
			},
			want: true,
		},
		{
			name: "non-numeric value is not zero",
			m: Model{
				ID:      "x/y:free",
				Pricing: ModelPricing{Prompt: "free", Completion: "0"},
			},
			want: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.m.IsFree(); got != tc.want {
				t.Fatalf("IsFree() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFetchModels(t *testing.T) {
	const fixture = `{"data":[
		{"id":"a/b:free","name":"A","created":1700000000,
		 "architecture":{"input_modalities":["text"],"output_modalities":["text"]},
		 "pricing":{"prompt":"0","completion":"0","image":"0","request":"0"}},
		{"id":"c/d","name":"C","created":1700000001,
		 "architecture":{"input_modalities":["text","image"],"output_modalities":["text"]},
		 "pricing":{"prompt":"0.00001","completion":"0.00003"}}
	]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("unexpected path %s", r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	models, err := FetchModels(context.Background(), server.URL, server.Client())
	if err != nil {
		t.Fatalf("FetchModels failed: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("want 2 models, got %d", len(models))
	}
	if !models[0].IsFree() {
		t.Errorf("expected models[0] (%s) to be free", models[0].ID)
	}
	if models[1].IsFree() {
		t.Errorf("expected models[1] (%s) to NOT be free", models[1].ID)
	}
}

func TestFetchModelsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer server.Close()

	_, err := FetchModels(context.Background(), server.URL, server.Client())
	if err == nil {
		t.Fatal("expected error on HTTP 502")
	}
}
