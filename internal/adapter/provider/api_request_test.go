package provider

import (
	"testing"

	"github.com/LurusTech/lurus-hub/internal/adapter/provider/common"
)

func TestProcessHeaderOverride(t *testing.T) {
	tests := []struct {
		name            string
		headersOverride map[string]interface{}
		apiKey          string
		wantHeaders     map[string]string
		wantErr         bool
	}{
		{
			name:            "nil map",
			headersOverride: nil,
			apiKey:          "sk-123",
			wantHeaders:     map[string]string{},
			wantErr:         false,
		},
		{
			name:            "empty map",
			headersOverride: map[string]interface{}{},
			apiKey:          "sk-123",
			wantHeaders:     map[string]string{},
			wantErr:         false,
		},
		{
			name:            "single header",
			headersOverride: map[string]interface{}{"X-Custom": "val"},
			apiKey:          "sk-123",
			wantHeaders:     map[string]string{"X-Custom": "val"},
			wantErr:         false,
		},
		{
			name: "multiple headers",
			headersOverride: map[string]interface{}{
				"X-One":   "1",
				"X-Two":   "2",
				"X-Three": "3",
			},
			apiKey: "sk-123",
			wantHeaders: map[string]string{
				"X-One":   "1",
				"X-Two":   "2",
				"X-Three": "3",
			},
			wantErr: false,
		},
		{
			name:            "Accept-Encoding lowercase skipped",
			headersOverride: map[string]interface{}{"accept-encoding": "gzip"},
			apiKey:          "sk-123",
			wantHeaders:     map[string]string{},
			wantErr:         false,
		},
		{
			name:            "Accept-Encoding canonical skipped",
			headersOverride: map[string]interface{}{"Accept-Encoding": "br"},
			apiKey:          "sk-123",
			wantHeaders:     map[string]string{},
			wantErr:         false,
		},
		{
			name:            "ACCEPT-ENCODING uppercase skipped",
			headersOverride: map[string]interface{}{"ACCEPT-ENCODING": "deflate"},
			apiKey:          "sk-123",
			wantHeaders:     map[string]string{},
			wantErr:         false,
		},
		{
			name:            "api_key replacement",
			headersOverride: map[string]interface{}{"Auth": "Bearer {api_key}"},
			apiKey:          "sk-123",
			wantHeaders:     map[string]string{"Auth": "Bearer sk-123"},
			wantErr:         false,
		},
		{
			name:            "double api_key replacement",
			headersOverride: map[string]interface{}{"X-Keys": "{api_key}:{api_key}"},
			apiKey:          "mykey",
			wantHeaders:     map[string]string{"X-Keys": "mykey:mykey"},
			wantErr:         false,
		},
		{
			name:            "empty apiKey",
			headersOverride: map[string]interface{}{"Auth": "Bearer {api_key}"},
			apiKey:          "",
			wantHeaders:     map[string]string{"Auth": "Bearer "},
			wantErr:         false,
		},
		{
			name:            "no variable",
			headersOverride: map[string]interface{}{"X-Static": "val"},
			apiKey:          "sk-456",
			wantHeaders:     map[string]string{"X-Static": "val"},
			wantErr:         false,
		},
		{
			name:            "non-string value int",
			headersOverride: map[string]interface{}{"X-Num": 42},
			apiKey:          "sk-123",
			wantErr:         true,
		},
		{
			name:            "non-string value bool",
			headersOverride: map[string]interface{}{"X-Bool": true},
			apiKey:          "sk-123",
			wantErr:         true,
		},
		{
			name:            "nil value",
			headersOverride: map[string]interface{}{"X-Nil": nil},
			apiKey:          "sk-123",
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &common.RelayInfo{
				ChannelMeta: &common.ChannelMeta{
					HeadersOverride: tt.headersOverride,
					ApiKey:          tt.apiKey,
				},
			}

			got, err := processHeaderOverride(info)
			if (err != nil) != tt.wantErr {
				t.Fatalf("processHeaderOverride() err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if len(got) != len(tt.wantHeaders) {
				t.Fatalf("processHeaderOverride() returned %d headers, want %d", len(got), len(tt.wantHeaders))
			}
			for k, wantV := range tt.wantHeaders {
				gotV, ok := got[k]
				if !ok {
					t.Errorf("missing header %q", k)
					continue
				}
				if gotV != wantV {
					t.Errorf("header %q = %q, want %q", k, gotV, wantV)
				}
			}
		})
	}
}
