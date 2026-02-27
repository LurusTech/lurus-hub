package repo

import (
	"testing"
)

func TestBuildTestDSN(t *testing.T) {
	tests := []struct {
		name     string
		baseDSN  string
		dbName   string
		wantSub  string // substring that must appear in the result
		wantFull string // exact match (empty = skip exact check)
	}{
		{
			name:     "kv/replaces dbname",
			baseDSN:  "host=localhost dbname=postgres sslmode=disable",
			dbName:   "test_repo_123",
			wantFull: "host=localhost dbname=test_repo_123 sslmode=disable",
		},
		{
			name:    "kv/appends when missing",
			baseDSN: "host=localhost sslmode=disable",
			dbName:  "test_repo_456",
			wantSub: "dbname=test_repo_456",
		},
		{
			name:    "kv/case insensitive DBNAME",
			baseDSN: "host=localhost DBNAME=postgres",
			dbName:  "test_repo_789",
			wantSub: "DBNAME=test_repo_789",
		},
		{
			name:    "kv/case insensitive DbName",
			baseDSN: "host=localhost DbName=mydb",
			dbName:  "test_repo_abc",
			wantSub: "test_repo_abc",
		},
		{
			name:    "kv/multiple dbname",
			baseDSN: "dbname=postgres dbname=extra",
			dbName:  "test_repo_multi",
			wantSub: "test_repo_multi",
		},
		{
			name:     "kv/empty baseDSN",
			baseDSN:  "",
			dbName:   "test_repo_empty",
			wantFull: " dbname=test_repo_empty",
		},
		{
			name:    "url/standard postgres",
			baseDSN: "postgres://user:pass@localhost:5432/postgres?sslmode=disable",
			dbName:  "test_repo_url1",
			wantSub: "/test_repo_url1",
		},
		{
			name:    "url/postgresql scheme",
			baseDSN: "postgresql://user:pass@host/mydb?sslmode=disable",
			dbName:  "test_repo_url2",
			wantSub: "/test_repo_url2",
		},
		{
			name:    "url/no query params",
			baseDSN: "postgres://user:pass@host/postgres",
			dbName:  "test_repo_url3",
			wantSub: "/test_repo_url3",
		},
		{
			name:    "url/encoded password",
			baseDSN: "postgres://user:p%40ss@host/postgres?sslmode=disable",
			dbName:  "test_repo_url4",
			wantSub: "p%40ss",
		},
		{
			name:    "url/no port",
			baseDSN: "postgres://user:pass@host/postgres",
			dbName:  "test_repo_url5",
			wantSub: "/test_repo_url5",
		},
		{
			name:    "kv/capture group preserves case",
			baseDSN: "DBNAME=postgres",
			dbName:  "testdb",
			wantSub: "DBNAME=testdb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildTestDSN(tt.baseDSN, tt.dbName)

			if tt.wantFull != "" && got != tt.wantFull {
				t.Errorf("buildTestDSN(%q, %q) = %q, want %q", tt.baseDSN, tt.dbName, got, tt.wantFull)
			}

			if tt.wantSub != "" {
				if !contains(got, tt.wantSub) {
					t.Errorf("buildTestDSN(%q, %q) = %q, want substring %q", tt.baseDSN, tt.dbName, got, tt.wantSub)
				}
			}

			// URL-format results must not contain the original db name in path
			if tt.baseDSN != "" && (len(tt.baseDSN) > 11 && (tt.baseDSN[:11] == "postgres://" || tt.baseDSN[:13] == "postgresql://")) {
				if contains(got, "/postgres?") || (got != tt.baseDSN && hasSuffix(got, "/postgres")) {
					// Original db name should be replaced
					if !contains(got, tt.dbName) {
						t.Errorf("buildTestDSN(%q, %q) = %q, expected dbName in result", tt.baseDSN, tt.dbName, got)
					}
				}
			}
		})
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchSubstring(s, sub)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
