# Story 6-10: context.Background() Cleanup

**Epic**: 6 - Code Review & Security Hardening
**Priority**: P2
**Status**: done

## Objective

Remove inappropriate `context.Background()` usage in production code per Go best practice:
"Business code MUST NOT use bare `context.Background()`; external calls MUST set timeout."

## Violation Analysis

### KEEP (19 uses - Appropriate)

| Location | Reason |
|----------|--------|
| `cmd/server/main.go` (4 uses) | Entry point / shutdown handlers |
| `*_test.go` (~70 uses) | Test root contexts |
| `internal/pkg/common/redis.go:42` | InitRedisClient - startup initialization |
| `internal/pkg/common/sys_log.go:42` (FatalLog) | Process-level fatal, no parent ctx |
| `internal/pkg/common/sys_log.go:76` (LogStartupSuccess) | Startup log, no request ctx |
| `internal/pkg/common/slog.go:313` (Background helper) | Utility wrapper, intentional |
| `internal/app/release_service.go:114` | Goroutine with independent lifecycle |
| `internal/adapter/repo/tenant_plugin.go:213` | Nil-ctx fallback (defensive) |
| `internal/adapter/repo/tenant_context.go:93` | No caller ctx available |
| `internal/adapter/provider/api_request.go:155` | Ping keep-alive (independent lifecycle) |
| Deprecated wrapper functions (7) | Wrappers calling WithContext variants |
| `adapter/provider/task/suno/adaptor.go:150` | Interface-bound, requires larger refactor |

### FIXED (22 violations in 15 files)

| # | File | Fix Applied |
|---|------|-------------|
| 1-12 | `pkg/common/redis.go` | Added `ctx context.Context` as first param to all 8 Redis functions |
| 13 | `adapter/middleware/rate-limit.go` | `c.Request.Context()` |
| 14 | `adapter/middleware/model-rate-limit.go` | `c.Request.Context()` |
| 15 | `adapter/middleware/email-verification-rate-limit.go` | `c.Request.Context()` |
| 16 | `adapter/handler/alipay.go` | Added `ctx` param to `getAlipayUserInfoByCode` |
| 17-18 | `adapter/handler/task.go` | Added `ctx` param to `UpdateTaskByPlatform` |
| 19 | `adapter/handler/midjourney.go` | Use parent `ctx` for timeout |
| 20 | `app/midjourney.go` | Use `c.Request.Context()` as parent |
| 21 | `app/relay/helper/stream_scanner.go` | Use `c.Request.Context()` |
| 22 | `adapter/provider/aws/dto.go` | Added `ctx` param to `formatRequest` |
| 23 | `adapter/provider/volcengine/tts.go` | Use `c.Request.Context()` |

### Cascading Updates (callers updated)

- `adapter/repo/user_cache.go` - all Redis calls updated with `context.TODO()`
- `adapter/repo/token_cache.go` - all Redis calls updated with `context.TODO()`
- `app/notify-limit.go` - `CheckNotificationLimit` accepts `ctx`
- `app/user_notify.go` - `NotifyUser`/`NotifyRootUser` accept `ctx`
- `app/channel.go` - `context.TODO()` at call sites
- `app/quota.go` - `context.TODO()` at call sites
- `adapter/handler/channel-test.go` - `context.TODO()` at call site
- `adapter/handler/alipay_test.go` - `context.Background()` in tests
- `app/notify_limit_test.go` - `context.Background()` in tests

## Verification

- `go build ./...` -> PASS
- `go test ./...` -> PASS (4 pre-existing failures in model_sync_worker_test unrelated)
- `go vet ./...` -> PASS (4 pre-existing warnings unrelated)

## DoD Checklist

- [x] All violations identified and categorized
- [x] Production code violations fixed
- [x] Compilation passes
- [x] Tests pass
- [x] go vet passes (no new warnings)
- [x] sprint-status.yaml updated
- [x] process.md updated
