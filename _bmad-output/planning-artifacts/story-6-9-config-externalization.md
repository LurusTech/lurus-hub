# Story 6-9: P1 Config Externalization

## Meta
- **Story ID**: 6-9
- **Epic**: 6 (Code Review & Security Hardening)
- **Priority**: P1
- **Status**: in-progress
- **Date**: 2026-02-13

## Problem

Code review (2026-02-11, 2026-02-13) identified 3 hardcoded values violating the zero-hardcoding principle:

| # | File | Hardcoded Value | Env Var |
|---|------|----------------|---------|
| 1 | `internal/app/release_service.go:28` | `"lurus-releases"` (MinIO bucket) | `MINIO_RELEASES_BUCKET` |
| 2 | `internal/adapter/middleware/cors.go:12-17` | CORS origin list | `ALLOWED_ORIGINS` |
| 3 | `internal/adapter/handler/alipay.go:150` | `"alipay_"` prefix | Constant extraction |

## Technical Design

### 1. Config Package Extension (`internal/pkg/config/config.go`)

Add `envString` helper and new config sections:

```go
type StorageConfig struct {
    MinIOBucket string
}

type CORSConfig struct {
    AllowedOrigins []string
}
```

### 2. MinIO Bucket (`release_service.go`)

- Read from `config.Get().Storage.MinIOBucket`
- Default: `"lurus-releases"`
- Env: `MINIO_RELEASES_BUCKET`

### 3. CORS Origins (`cors.go`)

- Read from `config.Get().CORS.AllowedOrigins`
- Default: current 5 origins
- Env: `ALLOWED_ORIGINS` (comma-separated)

### 4. Alipay Username Prefix (`alipay.go`)

- Extract to constant in `common/constants.go`
- `AlipayUsernamePrefix = "alipay_"`

### 5. K8s Deployment

- Add `MINIO_RELEASES_BUCKET` and `ALLOWED_ORIGINS` to `deploy/k8s/deployment.yaml`

## Files to Modify

1. `internal/pkg/config/config.go` - Add StorageConfig, CORSConfig, envString helper
2. `internal/pkg/config/config_test.go` - Add tests for new configs
3. `internal/app/release_service.go` - Use config for MinIO bucket
4. `internal/adapter/middleware/cors.go` - Use config for CORS origins
5. `internal/adapter/handler/alipay.go` - Use constant for prefix
6. `internal/pkg/common/constants.go` - Add AlipayUsernamePrefix constant
7. `deploy/k8s/deployment.yaml` - Add new env vars

## Test Plan

- Unit tests for new config fields (default + env override)
- Unit test for envString helper
- Verify `go build ./...` passes
- Verify `go test ./...` passes
- Verify `go vet ./...` passes

## Definition of Done

- [x] All 3 hardcoded values externalized
- [x] Config defaults match current behavior (zero regression)
- [x] envString + envStringSlice helpers added with 17 new tests
- [x] K8s deployment updated with ALLOWED_ORIGINS + MINIO_RELEASES_BUCKET
- [x] `go build ./...` passes
- [x] `go test ./internal/pkg/config/...` passes (47/47)
- [x] `go test ./internal/pkg/... ./internal/app/... ./internal/adapter/middleware/...` all PASS
- [x] `go vet ./...` — pre-existing warnings only, no new issues
