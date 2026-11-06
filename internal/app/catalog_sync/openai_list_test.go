package catalog_sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAICompatibleProvider_ParsesIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("auth=%q", got)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"m1"},{"id":"m2"},{"id":"m1"}]}`))
	}))
	defer srv.Close()

	p := &OpenAICompatibleProvider{
		Label:      "test",
		BaseURL:    srv.URL,
		APIKey:     "test-key",
		HTTPClient: srv.Client(),
	}
	models, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0].ID != "m1" || models[1].ID != "m2" {
		t.Fatalf("models=%#v", models)
	}
}

func TestVolcengineProvider_SkipsShutdown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/models" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"live","status":""},{"id":"dead","status":"Shutdown"},{"id":"retire","status":"Retiring"}]}`))
	}))
	defer srv.Close()

	p := &VolcengineProvider{BaseURL: srv.URL, HTTPClient: srv.Client()}
	models, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 {
		t.Fatalf("want 2 (skip Shutdown), got %#v", models)
	}
}
