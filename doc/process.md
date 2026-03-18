# Development Progress / 开发进度

> Last Updated: 2026-02-25
> Archive: doc/archive/process_v20260205.md (entries before 2026-02-04)
> **New Rule**: 每条目 ≤ 15 行（HARD LIMIT），只记录已完成工作的极简摘要

---

## 2026-02-25: 计费系统安全威胁模型修复（P0+P1）

修复 10 个安全漏洞，覆盖无限充值、签名绕过、竞争条件、幂等等问题。
- P0-1 `subscription_cron.go`: netDelta 分三步（扣费→补额→reset daily），TotalQuota=0 不补 quota
- P0-2 `topup_creem.go/subscription_payment.go`: 删除 TestMode 签名绕过，secret 空时直接拒绝
- P0-3 `topup.go`: 删除跨 Pod 无效的 LockOrder/UnlockOrder，EpayNotify 改走 ManualCompleteTopUp（DB FOR UPDATE）
- P0-4 `subscription.go`: ActivateSubscription 加 FOR UPDATE 幂等检查，状态非 Pending 则拒绝
- P0-5 `internal_api.go`: InternalGrantSubscription 补充 RecordLog 审计
- P0-6 `subscription_payment.go`: 金额不足改为返回 error 拒绝激活，容差改固定 50 cents
- P1-1 `rate-limit.go/api-v2-router.go`: 兑换码接口加 5次/分钟 IP 限速
- P1-2 `topup.go`: AdminCompleteTopUp 补充管理员 ID 审计日志
- P1-3 `user.go`: ResetDailyQuota 加 last_daily_reset < todayStart 幂等条件
- P1-4 `auth.go`: lurus-api-User header 改可选（仅验证不作为 ID 来源）

Verification: `go build ./... → OK`; `go vet ./internal/adapter/middleware/... → OK`
Remaining: P2 系列（int64 溢出、CSRF、JWT aud 验证）待排期。

## 2026-02-25: 计费系统评估 + 自动续费实现

完成多产品计费能力评估，修复自动续费 TODO。
- `subscription_cron.go: processOneAutoRenewal()` — 实现余额扣费自动续费，原子事务（双重检查锁 + gorm FOR UPDATE）
- 修正逻辑：扣 `plan.Price * QuotaPerUnit` + 补 `plan.TotalQuota`，net delta 一次写库
- `doc/billing-system-guide.md` — 修正支付宝/微信状态（仅 OAuth，非支付），更新自动续费描述（24h 触发），新增多产品能力评估章节
- 结论：AI 网关计费 ✅ 可撑，多产品中台 ❌ 需独立服务（推荐路径 B）

Verification: `go build ./... → OK`
Remaining: 自动续费余额不足时邮件通知（TODO 标注），退款/发票系统在 lurus-billing 规划中。

---

## 2026-02-13: Epic 6 Complete — Code Review & Security Hardening

**Completed Stories**: 6-1 ~ 6-10 (对抗性审查 + P0/P1/P2 修复)

**Key Deliverables**:
- P0-1: Git history cleanup (git-filter-repo, 5019 commits, secrets.yaml removed)
- P1: Config externalization (MinIO bucket, CORS, alipay prefix → env/constant)
- P2: context.Background() cleanup (22 fixes in 15 files)
- Tests: 40+ new tests (Alipay 14, Release 13, Model Sync 16)
- Docs: DEPLOY.md, TESTING.md, code review reports

**Verification**: `go test ./... → PASS`, `go build → OK`, Git history clean

**Remaining**: Force push cleaned history + credential rotation (out-of-sprint)

---

## 2026-02-11: Download System + SSO Phase 1

Backend API for release downloads + cross-domain SSO (cookie-based).

**Files**: migrations/005, domain/entity/release*, app/release_service, handler/release

**Verification**: `go build → 93MB`, 编译通过

**Status**: ⏳ 待数据库迁移 + MinIO 配置 + 前后端联调

---

## 2026-02-06: Tech Debt Cleanup

Fixed user_mapping insecure password, removed dead code, created 3 ADRs (HA/v1-deprecation/observability).

**Verification**: `go test ./... → PASS`, `go build → OK`

---

## 2026-02-05: Epic 2-5 Complete — Tests, Performance, Observability, DevEx

**Epic 2**: 187 service tests, 100+ adaptor tests, 50 controller tests, 34 security tests
**Epic 3**: Benchmarks (p95 <50ms), object pools, HA deployment (2 replicas + PDB)
**Epic 4**: Prometheus /metrics (11 types), OpenTelemetry tracing, 10 alerting rules
**Epic 5**: OpenAPI spec (45 endpoints), staging env, 6 runbooks

**Verification**: `go test ./internal/pkg/metrics → 7 PASS`, `go test ./internal/pkg/tracing → 9 PASS`

---

## 2026-02-05: Epic 1 Complete — Multi-Tenant Production Launch

Deployed V2 API to K3s with Zitadel OIDC auth.

**Key Commits**: 80323446b, 6232258ad, d85e5d422, d74a16a65

**Verification**: ArgoCD sync, pod 1/1 Running, /api/status→200, V2 login→302 to auth.lurus.cn

---

## 2026-02-04: Architecture Migration — Hexagonal Restructure

Migrated `biz/data/server` → `domain/app/adapter` (hexagonal architecture).

**Verification**: `go build ./cmd/server → PASS`, `go test ./... → PASS`

> **Archive Note**: Entries before 2026-02-04 in `doc/archive/process_v20260205.md`

---

## 2026-02-25: 修复注册流程邮件发送失败

排查 `SendEmailVerification` 全链路，发现三个根因并逐一修复：

1. `SMTPServer=localhost` → 改为 `stalwart.mail.svc`（K8s 集群内服务名）
2. Stalwart brute-force 封锁 Pod CIDR → 在 `stalwart-config` ConfigMap 加 `[server.allowed-ip] "10.42.0.0/16"=true "10.43.0.0/16"=true`，滚动重启
3. `SMTPAccount=noreply@lurus.cn` → Stalwart 内部目录只支持短账户名，改为 `noreply`；`SMTPFrom` 保持完整地址

Verification: `curl https://api.lurus.cn/api/verification?email=tpy@lurus.cn → {"success":true}`；Stalwart 日志确认 `queue.queue-message-authenticated` + `delivery.completed`

---

## 2026-02-25: 配置 DKIM 签名

Stalwart `auth.dkim.sign = 'rsa-lurus.cn'`（selector=`default`）即实际出站签名配置，`session.data.sign` 为入站处理。
关键修复：`%{file:...}%` 宏在 RocksDB 中不展开，需内联 PEM；将 PKCS#8 转为 PKCS#1 写入 config.toml。

**DNS**: 添加 `default._domainkey.lurus.cn TXT "v=DKIM1; k=rsa; p=MIIBIjAN..."`

**Verification**:
- `dig TXT default._domainkey.lurus.cn +short` → 返回完整公钥记录
- 私钥提取公钥与 DNS 公钥 base64 完全一致（openssl 确认 2048-bit RSA）
- 实测：noreply→QQ Mail 投递成功，`DKIM-Signature: v=1; a=rsa-sha256; s=default; d=lurus.cn`

## 2026-02-25: Security Fix Supplement Tests (P0-1/P0-2/P0-4/P0-6/P1-3/P1-4)
Fixed 1 broken test (empty_secret_test_mode now expects false). Added 6 new/extended test files covering processOneAutoRenewal (5 subtests), verifyCreemSubscriptionSignature (4 subtests), amount tolerance validation (4 subtests), ActivateSubscription idempotency, ResetDailyQuota idempotency, lurus-api-User header (4 subtests).
Also fixed GREATEST() SQLite incompatibility in subscription_cron.go/subscription.go via quotaDeductSafe() helper.
Verification: `go test ./internal/adapter/handler/... -run "TestVerifyCreemSignature|TestVerifyCreemSubscription|TestAmountValidation"` → PASS; `go test ./internal/adapter/repo/... -run "TestProcessOneAutoRenewal|TestSubscription_ActivateSubscription_Idempotent|TestResetDailyQuota_Idempotent"` → PASS; `go test ./internal/adapter/middleware/... -run "TestAuthHelper"` → PASS; `go build ./...` → OK.

## 2026-02-25: 测试DB迁移PG + new-api增量融合
Part1: testutil_test.go 重写，移除 glebarez/sqlite，改用 TEST_POSTGRES_DSN；SetupTestDB 创建独立 test_repo_<nano> 数据库，cleanup 时 DROP；quotaDeductSafe 移除 SQLite 分支直接用 GREATEST。
Part2: (2a) stream_scanner.go TrimSuffix("\r")→TrimSpace+空串跳过；(2b) processHeaderOverride 跳过 Accept-Encoding；(2c) GeminiUsageMetadata 加 ToolUsePromptTokenCount，提取 buildUsageFromGeminiMetadata 消除两处重复；(2d) MiniMax 添加 MiniMax-Text-01/MiniMax-01/minimax-text-01；(2e) Gemini 添加 gemini-2.0-flash-lite/2.5-flash-preview-04-17/2.5-pro-preview-05-06/2.0-flash-thinking-exp-01-21。
Verification: `go build ./...` → OK (0 errors). PostgreSQL 集成测试待 TEST_POSTGRES_DSN 注入后验证。

## 2026-03-17: lurus-api 瘦身 — 删除 v1 auth + 前端清理 + P0 幂等修复

**Phase 1-3 (Go backend)**: 删除 30 个文件（checkin/invitation/OAuth/2FA/Passkey/SMS/admin_config），精简 User entity（移除 Password/OAuth IDs/aff 字段），清理 router 路由 ~80 条。认证统一委托 Zitadel OIDC。
**Phase 4 (Frontend)**: 删除 16 个 React 文件（LoginForm/RegisterForm/2FA/Passkey/OAuth/checkin/affiliate），修改 12 个文件移除 v1 auth 调用。净删 ~6,438 行。
**P0 fix**: `InternalTopupBalance` 加 order_id 幂等检查（查 LOG_DB 已有记录），防止 platform 支付重试导致重复充值。

Verification: `go test ./... → 20/20 PASS`; `cd web && bun run build → OK`; `CGO_ENABLED=0 GOOS=linux go build ./... → OK`
Commit: `24477bcd9` pushed to `origin/main`。
Remaining: AuthSettingPage.jsx 仍有 Passkey 管理员配置（死设置，不影响功能）；P1 async wallet bridge 无重试机制。

## 2026-03-18: Zitadel OIDC 登录修复 (3 项)

1. **Hairpin NAT**: Pod 内 `auth.lurus.cn` 解析到公网 IP 43.226.46.164，回连被拒。修复: deployment 加 `hostAliases` 指向 Traefik ClusterIP `10.43.175.138` + NetworkPolicy 增 kube-system 443/8443 出站规则。
2. **Email fallback**: `CreateUserFromZitadelClaims` 增加 email 回退匹配（跨租户），老用户首次 OIDC 登录自动建 mapping。
3. **租户数据迁移**: root/marvin 的 `tenant_id` 从 `default` 迁移至 `356204220778610952`；tokens/channels/logs/redemptions 同步迁移；重复用户 (id=6,7) 已软删除。

Verification: `kubectl exec ... wget auth.lurus.cn/.well-known/openid-configuration → OK`; Pod OIDC token exchange 畅通。
Commits: `6600e6c52` (ZitadelRedirect), `bd6fe89dc` (email fallback), deploy `5bbb940` (NetworkPolicy).
