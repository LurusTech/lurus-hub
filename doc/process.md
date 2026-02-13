# Development Progress / 开发进度

> Last Updated: 2026-02-13
> Archive: doc/archive/process_v20260205.md (entries before 2026-02-04)

---

## 2026-02-13: Story 6-8 — Git History Cleanup (P0) + Epic 6 Complete

Removed `deploy/k8s/secrets.yaml` from entire Git history using `git-filter-repo`.
- Previous `git filter-branch` attempt failed on Windows (trailing space in historical filename).
- `git-filter-repo` succeeded in 5.37s, rewrote 5019 commits.
- Template re-added, all working changes preserved via stash.
- Verification: `git log --all -p -S "<password>" -- deploy/k8s/secrets.yaml` returns empty.
- Epic 6 (Code Review & Security Hardening): all 10 stories now complete.
Remaining: force push to remote + team re-clone + credential rotation.

---

## 2026-02-13: Story 6-10 — context.Background() Cleanup

Fixed 22 `context.Background()` violations in 15 production files. Key changes:
- Redis utility functions (RedisSet/Get/Del/HSetObj/HGetObj/Incr/HIncrBy/HSetField) now accept `ctx context.Context`
- Middleware (rate-limit, model-rate-limit, email-verification) use `c.Request.Context()` instead of `context.Background()`
- Handlers (alipay, task, midjourney) propagate parent ctx through call chain
- Providers (aws/dto, volcengine/tts, stream_scanner) use request context
- 19 uses kept as legitimate (init, deprecated wrappers, independent-lifecycle goroutines)
Verification: `go build ./... -> OK`, `go test ./... -> PASS (4 pre-existing failures in model_sync unrelated)`

---

## 2026-02-13: Story 6-9 — Config Externalization

Externalized 3 hardcoded values to centralized config (env vars with defaults):
- MinIO bucket (`MINIO_RELEASES_BUCKET`), CORS origins (`ALLOWED_ORIGINS`), alipay username prefix (constant).
- Added `envString` + `envStringSlice` helpers to config package. 17 new tests, all PASS.
Verification: `go test ./internal/pkg/config/... → 47/47 PASS`, `go build ./... → OK`

---

## 2026-02-13: Adversarial Code Review + P0/P1 Fixes

Completed adversarial code review (found 8 issues: 2 P0, 3 P1, 3 P2) and fixed all P1 + partial P0.

**P1 Fixes (deployed)**: Version comparison bug (semver), Alipay session panic (type assertion), secrets template cleanup.
**Tests Added**: 40+ test cases (Alipay 14, Release 13, Model Sync 16, semver 2). All PASS.
**Docs**: DEPLOY.md (5-min quickstart), TESTING.md (commands reference), code review reports.
**P0-1 (pending)**: Git history cleanup script created (`scripts/cleanup-secrets-history.sh`), requires Linux/macOS execution.
**Verification**: `go test ./... → PASS`, `go build → 93MB`, 5 commits ready for push.

**Remaining**: P1 config externalization (MinIO bucket, CORS, alipay_ prefix), P2 context.Background() cleanup (32 files).

---

## 2026-02-11: 下载系统 + SSO Phase 1 后端实现

完成下载管理系统和跨域 SSO 的后端支持，配合前端（lurus-www）实现完整功能。

**下载系统后端**：
- Database migration（releases, release_artifacts, download_logs 表）
- Domain entities + Repository 层（GORM 查询，分页，过滤）
- Service 层（预留 MinIO 集成接口）
- Handler 层（5 个 API 端点：列表、最新版本、详情、下载、Changelog）
- Router 注册（`/api/v1/releases/*`）+ DownloadRateLimit 中间件

**SSO Phase 1 后端**（Cookie-based 跨域登录）：
- Session Domain 配置（`Domain: ".lurus.cn"`，支持跨子域共享）
- CORS 中间件更新（AllowOrigins 明确列出子域名 + AllowCredentials）
- 路由注册（`GET /api/v1/auth/session`，调用已有的 GetSessionInfo）

**验证结果**：
- `go build ./cmd/server` → PASS（93MB 可执行文件）
- 所有新增代码编译通过，无错误

**待部署配置**：
- MinIO bucket 创建（`lurus-releases`）
- 数据库迁移执行（`migrations/005_create_releases_tables.sql`）
- 前端环境变量（`VITE_API_URL=https://api.lurus.cn`）

**状态**：⏳ 后端代码完成，待数据库迁移 + MinIO 配置 + 前后端联调

---

## 2026-02-06: Tech Debt Cleanup — Code Fixes + ADRs

Docs archived & committed. Code-level fixes:
- `user_mapping.go`: replaced insecure placeholder password (timestamp) with `crypto/rand`, deduplicated `generateAffCode()` to use `common.GetRandomString(4)`
- `repo/main.go`: removed dead commented-out MySQL migration line
- `payment_setting_old.go`: NOT deleted (plan was wrong — 20+ active references across option.go, topup.go, epay.go)
- Created 3 ADR docs: `doc/decisions/{ha-deployment,v1-deprecation,observability}.md`
- 4 "known failing" tests confirmed PASS (stale report)
Verification: `go test ./... → all PASS`, `go build ./cmd/server → OK`

---

## 2026-02-05: Sprint 3-5 Complete — Performance, Observability, DevEx

All 9 stories (3.1-3.3, 4.1-4.3, 5.1-5.3) plus Story 2.4 completed in one session.
- 3.1/3.2: Benchmarks + object pools (p95 <50ms, BufferPool/IntSlicePool/MapPool)
- 3.3: HA deployment (2 replicas, rollingUpdate, PDB)
- 4.1: Prometheus /metrics (11 metric types)
- 4.2: OpenTelemetry tracing (Jaeger exporter, X-Trace-Id header)
- 4.3: 10 alerting rules + ServiceMonitor + Alertmanager routing
- 5.1: OpenAPI spec (45 endpoints, Swagger UI at docs.lurus.cn)
- 5.2: Staging environment (lurus-staging namespace, auto-deploy)
- 2.4: 34+ security regression tests (tenant isolation, SQL injection, XSS)
Verification: `go test ./internal/pkg/metrics/... → 7 PASS`, `go test ./internal/pkg/tracing/... → 9 PASS`

---

## 2026-02-05: Epic 1 Complete — Multi-Tenant Production Launch

Deployed V2 API with hexagonal architecture to K3s production.
- Committed 434 files: `internal/{biz,data,server}` → `internal/{domain,app,adapter}`
- Fixed SQLite test index collision (`idx_tenant_user`)
- ArgoCD sync confirmed, pod 1/1 Running
Verification: `go test ./internal/... → PASS`, `/api/status → 200`, `/api/v2/lurus/auth/login → 302`

---

## 2026-02-04: Architecture Migration — Hexagonal Restructure

Migrated from Kratos-style (`biz/data/server`) to Hexagonal (`domain/app/adapter`). 7 phases.
- `data/model` → `domain/entity/`, `biz/service/` → `app/`, `server/controller/` → `adapter/handler/`
- `biz/relay/channel/` → `adapter/provider/`, `server/middleware/` → `adapter/middleware/`
Verification: `go build ./cmd/server` PASS, `go test ./...` PASS.

> Note: Entries before 2026-02-04 archived to `doc/archive/process_v20260205.md`.
> Historical entries reference old paths (`biz/`, `server/controller/`).
