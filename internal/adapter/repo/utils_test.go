package repo

import (
	"errors"
	"testing"

	"github.com/LurusTech/lurus-hub/internal/pkg/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ---------------------------------------------------------------------------
// quotaDeductSafe
// ---------------------------------------------------------------------------

func TestQuotaDeductSafe(t *testing.T) {
	tests := []struct {
		name     string
		cost     int
		wantSQL  string
		wantVars []interface{}
	}{
		{"normal cost", 1000, "GREATEST(quota - ?, 0)", []interface{}{1000}},
		{"zero cost", 0, "GREATEST(quota - ?, 0)", []interface{}{0}},
		{"negative cost", -500, "GREATEST(quota - ?, 0)", []interface{}{-500}},
		{"large number", 2_000_000_000, "GREATEST(quota - ?, 0)", []interface{}{2_000_000_000}},
		{"max int32", 2147483647, "GREATEST(quota - ?, 0)", []interface{}{2147483647}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quotaDeductSafe(tt.cost)
			expr, ok := result.(clause.Expr)
			if !ok {
				t.Fatalf("quotaDeductSafe(%d) returned %T, want clause.Expr", tt.cost, result)
			}
			if expr.SQL != tt.wantSQL {
				t.Errorf("SQL = %q, want %q", expr.SQL, tt.wantSQL)
			}
			if len(expr.Vars) != len(tt.wantVars) {
				t.Fatalf("Vars length = %d, want %d", len(expr.Vars), len(tt.wantVars))
			}
			for i, v := range expr.Vars {
				if v != tt.wantVars[i] {
					t.Errorf("Vars[%d] = %v, want %v", i, v, tt.wantVars[i])
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RecordExist
// ---------------------------------------------------------------------------

func TestRecordExist(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantExist bool
		wantErr   bool
	}{
		{"nil error", nil, true, false},
		{"record not found", gorm.ErrRecordNotFound, false, false},
		{"other error", errors.New("connection refused"), false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exist, err := RecordExist(tt.err)
			if exist != tt.wantExist {
				t.Errorf("RecordExist() exist = %v, want %v", exist, tt.wantExist)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("RecordExist() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// shouldUpdateRedis
// ---------------------------------------------------------------------------

func TestShouldUpdateRedis(t *testing.T) {
	tests := []struct {
		name         string
		redisEnabled bool
		fromDB       bool
		err          error
		want         bool
	}{
		{"all conditions met", true, true, nil, true},
		{"redis disabled", false, true, nil, false},
		{"not from DB", true, false, nil, false},
		{"error present", true, true, errors.New("some error"), false},
		{"all false", false, false, errors.New("err"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prev := common.RedisEnabled
			common.RedisEnabled = tt.redisEnabled
			defer func() { common.RedisEnabled = prev }()

			got := shouldUpdateRedis(tt.fromDB, tt.err)
			if got != tt.want {
				t.Errorf("shouldUpdateRedis(fromDB=%v, err=%v) = %v, want %v", tt.fromDB, tt.err, got, tt.want)
			}
		})
	}
}
