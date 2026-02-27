package helper

import (
	"strings"
	"testing"
)

// outcome represents the result type of parsing a single SSE line.
type outcome int

const (
	outcomeSkipped outcome = iota
	outcomeData
	outcomeDone
)

// parseStreamLine replicates the inline parsing logic from stream_scanner.go
// (lines 207-248) as a pure function so it can be unit-tested without needing
// a full gin.Context / HTTP response pipeline.
func parseStreamLine(line string) (outcome, string) {
	data := line

	if len(data) < 6 {
		return outcomeSkipped, ""
	}
	if data[:5] != "data:" && data[:6] != "[DONE]" {
		return outcomeSkipped, ""
	}
	data = data[5:]
	data = strings.TrimSpace(data)
	if data == "" {
		return outcomeSkipped, ""
	}
	if strings.HasPrefix(data, "[DONE]") {
		return outcomeDone, data
	}
	return outcomeData, data
}

func TestStreamLineParseLogic(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantOutcome outcome
		wantData    string
	}{
		{
			name:        "standard JSON data",
			line:        `data: {"key":"value"}`,
			wantOutcome: outcomeData,
			wantData:    `{"key":"value"}`,
		},
		{
			name:        "trailing CR",
			line:        "data: {\"k\":\"v\"}\r",
			wantOutcome: outcomeData,
			wantData:    `{"k":"v"}`,
		},
		{
			name:        "leading+trailing whitespace after colon",
			line:        `data:   {"k":"v"}   `,
			wantOutcome: outcomeData,
			wantData:    `{"k":"v"}`,
		},
		{
			name:        "no space after colon",
			line:        `data:{"k":"v"}`,
			wantOutcome: outcomeData,
			wantData:    `{"k":"v"}`,
		},
		{
			name:        "standard DONE",
			line:        "data: [DONE]",
			wantOutcome: outcomeDone,
			wantData:    "[DONE]",
		},
		{
			// "data:[DONE]" → data[:5]="data:" matches, data[5:]="[DONE]",
			// TrimSpace → "[DONE]", HasPrefix("[DONE]") → true → outcomeDone
			name:        "DONE no space",
			line:        "data:[DONE]",
			wantOutcome: outcomeDone,
			wantData:    "[DONE]",
		},
		{
			name:        "empty line",
			line:        "",
			wantOutcome: outcomeSkipped,
			wantData:    "",
		},
		{
			name:        "5 chars data colon only",
			line:        "data:",
			wantOutcome: outcomeSkipped,
			wantData:    "",
		},
		{
			name:        "4 chars",
			line:        "ping",
			wantOutcome: outcomeSkipped,
			wantData:    "",
		},
		{
			name:        "non-data prefix",
			line:        "event: message",
			wantOutcome: outcomeSkipped,
			wantData:    "",
		},
		{
			name:        "SSE id field",
			line:        "id: 12345678",
			wantOutcome: outcomeSkipped,
			wantData:    "",
		},
		{
			name:        "whitespace after colon",
			line:        "data:  \r\n",
			wantOutcome: outcomeSkipped,
			wantData:    "",
		},
		{
			name:        "only CR after colon",
			line:        "data:\r",
			wantOutcome: outcomeSkipped,
			wantData:    "",
		},
		{
			name:        "only spaces after colon",
			line:        "data:     ",
			wantOutcome: outcomeSkipped,
			wantData:    "",
		},
		{
			// Known quirk: "[DONE]" → len=6, [0:5]="[DONE" != "data:", [0:6]="[DONE]" == "[DONE]"
			// → data = [5:] = "]", TrimSpace → "]", not empty, not HasPrefix "[DONE]" → outcomeData
			name:        "raw DONE line quirk",
			line:        "[DONE]",
			wantOutcome: outcomeData,
			wantData:    "]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOutcome, gotData := parseStreamLine(tt.line)
			if gotOutcome != tt.wantOutcome {
				t.Errorf("parseStreamLine(%q) outcome = %d, want %d", tt.line, gotOutcome, tt.wantOutcome)
			}
			if gotData != tt.wantData {
				t.Errorf("parseStreamLine(%q) data = %q, want %q", tt.line, gotData, tt.wantData)
			}
		})
	}
}
