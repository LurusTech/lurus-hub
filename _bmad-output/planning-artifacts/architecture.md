---
stepsCompleted: [1, 2, 3, 4, 5, 6, 7, 8]
inputDocuments: ['prd-api.md', 'epics-api.md', 'product-brief.md', 'project-context.md', 'architecture.md', 'doc/decisions/v1-deprecation.md', 'doc/decisions/observability.md', 'doc/decisions/ha-deployment.md', 'doc/meilisearch-integration.md', 'doc/zitadel-setup-guide.md']
workflowType: 'architecture'
service: 'lurus-api'
project_name: 'lurus-api'
user_name: 'Anita'
date: '2026-02-03'
---

# Architecture Decision Document: Lurus API
# 架构决策文档：Lurus API

---

## 1. System Context / 系统上下文

### 1.1 System Purpose / 系统定位

Lurus API 是一个**多租户 LLM API 网关**，提供 OpenAI 兼容接口访问 30+ AI 模型供应商。核心职责：认证、计费、配额管理、请求中继。

### 1.2 System Boundary / 系统边界

```
                        ┌─────────────────────────────────────────┐
                        │              Lurus API                   │
                        │           (api.lurus.cn:8850)            │
                        │                                         │
 Clients                │  ┌─────────┐  ┌──────────┐  ┌────────┐│         Upstream
 ─────────────────────► │  │  Gin    │  │ Relay    │  │Adaptor ││ ──────► LLM Providers
 - OpenAI SDK           │  │ Router  │→ │ Engine   │→ │Factory ││         (30+)
 - curl / HTTP          │  │         │  │          │  │        ││
 - React SPA (embedded) │  └─────────┘  └──────────┘  └────────┘│
                        │       │                                 │
                        │  ┌────┴────────────────────────────────┐│
                        │  │     Middleware Stack                 ││
                        │  │  Auth │ RateLimit │ Tenant │ Stats  ││
                        │  └─────────────────────────────────────┘│
                        │       │              │                   │
                        │  ┌────┴─────┐  ┌────┴──────┐           │
                        │  │ App      │  │ Handler   │           │
                        │  │ Layer    │  │ Layer     │           │
                        │  └────┬─────┘  └───────────┘           │
                        │       │                                 │
                        └───────┼─────────────────────────────────┘
                                │
                 ┌──────────────┼──────────────┐
                 │              │              │
           ┌─────┴─────┐ ┌────┴────┐  ┌──────┴──────┐
           │ PostgreSQL │ │  Redis  │  │ Meilisearch │
           │ (CNPG)     │ │  db:0   │  │ (optional)  │
           │ lurus_api  │ │         │  │             │
           └───────────┘ └─────────┘  └─────────────┘
```

### 1.3 External Dependencies / 外部依赖

| System | Protocol | Purpose | Failure Mode |
|--------|----------|---------|-------------|
| OpenAI, Anthropic, Google, AWS Bedrock, DeepSeek, etc. (30+) | HTTPS | LLM inference | Cross-channel retry / failover |
| Zitadel (auth.lurus.cn) | OIDC/HTTPS | v2 JWT authentication | v1 session auth continues |
| PostgreSQL (CNPG) | TCP:5432 | Primary data store | Service unavailable |
| Redis (db:0) | TCP:6379 | Session, rate limiting, cache | Degraded (no rate limit, no cache) |
| Meilisearch | HTTP:7700 | Full-text search | Fallback to DB queries |
| Stripe / Epay / Creem | HTTPS webhook | Payment processing | Payment temporarily unavailable |

---

## 2. Key Architecture Decisions / 关键架构决策

### ADR-API-001: Adaptor Pattern for LLM Providers

**Status**: Accepted

**Context**: 30+ LLM 供应商各有不同 API 格式，需统一为 OpenAI 兼容接口。

**Decision**: 使用接口 + 工厂模式。`Adaptor` 接口定义 14 个方法，`BaseAdaptor` 提供默认实现，各供应商覆写差异方法。

```
Adaptor Interface (14 methods)
  ├── BaseAdaptor (defaults for all 14)
  ├── OpenAI Adaptor (minimal overrides)
  ├── Claude Adaptor (request/response transformation)
  ├── Gemini Adaptor (native protocol support)
  └── ... 30+ more
```

**Key Methods**: `GetModelList`, `ConvertRequest`, `DoRequest`, `DoResponse`, `ConvertSTTRequest`, `ConvertTTSRequest`, ...

**Consequences**:
- (+) 新增供应商只需实现差异方法，不触碰核心代码
- (+) BaseAdaptor 消除重复代码
- (-) 14 个方法的接口较大，但 BaseAdaptor 默认实现降低了负担

---

### ADR-API-002: Shared DB + tenant_id Column Isolation

**Status**: Accepted

**Context**: 多租户 SaaS 转型，2 人团队需要最低运维开销的隔离方案。

**Decision**: 所有业务表添加 `tenant_id` 字段，GORM Plugin 自动注入 `WHERE tenant_id = ?`。v1 API 默认使用 `tenant_id = 'default'`。

**Implementation**:
- `TenantPlugin` 注册 GORM Callback（Query/Create/Update/Delete）
- `DefaultTenantMiddleware` 为 v1 路由注入默认租户
- `ZitadelAuth` 中间件从 JWT 提取 `org_id` 映射到 `tenant_id`
- Platform Admin 可跳过隔离（`GetSystemDB()`）

**Consequences**:
- (+) 无需多 schema/多 DB，运维简单
- (+) v1 API 无感知兼容
- (-) 高流量租户可能影响其他租户（可通过连接池和资源限制缓解）

---

### ADR-API-003: Dual Auth (Session v1 + JWT v2)

**Status**: Accepted

**Context**: 现有 v1 API 使用 Session-based auth，多租户 v2 API 需要 Zitadel OIDC JWT。

**Decision**: 双轨运行。v1 保留 Session（cookie.NewStore），v2 使用 Zitadel JWT（JWKS 验证）。共享同一 Service 层和 GORM 租户插件。

**Auth Flow**:
```
v1: Session Cookie → SessionAuth middleware → tenant_id="default"
v2: Zitadel JWT   → ZitadelAuth middleware → tenant_id from org_id
```

**JWKS Management**:
- 内存缓存公钥（`map[kid]*rsa.PublicKey`）
- 每小时自动刷新 + key miss 时即时刷新
- `sync.RWMutex` 保护并发访问
- 30 秒最小刷新间隔防止 thundering herd

**Consequences**:
- (+) 渐进迁移，现有用户无感知
- (-) 双重认证路径增加安全审计面（待 v1 废弃后消除，见 ADR-API-007）

---

### ADR-API-004: Channel Routing with Priority + Weight

**Status**: Accepted

**Context**: 同一模型可能有多个供应商渠道（channel），需要智能路由和故障转移。

**Decision**: 三级路由策略：Group → Priority → Weight。

```
Request(model="gpt-4o", group="vip")
  ↓
1. Filter: channels supporting gpt-4o in group "vip"
2. Sort: by priority (descending)
3. Select: weighted random among highest-priority channels
4. Fail → retry next channel (same group)
5. All fail → cross-group retry (if token allows)
```

**Channel Cache**: 内存缓存所有 channel 数据，周期性从 DB 同步（默认 60s），避免热路径 DB 查询。

**Consequences**:
- (+) 自动故障转移，单个供应商宕机不影响服务
- (+) 权重分配可控制流量比例
- (-) Channel cache 有最多 60s 延迟（可手动触发刷新）

---

### ADR-API-005: Quota-Based Billing with Model Ratios

**Status**: Accepted

**Context**: 不同模型 token 成本差异巨大（GPT-4o vs GPT-3.5），需要统一配额单位。

**Decision**: 所有消费折算为统一 quota 单位。每个模型配置 `input_ratio` 和 `output_ratio`，按实际 token 数乘以比率扣减。

```
quota_consumed = prompt_tokens * model_input_ratio
              + completion_tokens * model_output_ratio
```

**Quota Layers**:
- Token 级配额（单个 API key 限制）
- User 级配额（用户总余额）
- Pre-consume: 请求前预扣，响应后精确结算
- 订阅计划通过 Daily Quota Cron 每日重置

**Payment Gateways**: Stripe, Epay, Creem — Webhook 验证包含租户归属校验。

**Consequences**:
- (+) 统一计费单位，用户易理解
- (+) 灵活调整模型定价
- (-) Model ratio 需要人工维护和更新

---

### ADR-API-006: Embedded React SPA

**Status**: Accepted

**Decision**: React 18 + Vite + Semi UI 前端编译为 `web/dist`，通过 `go:embed` 嵌入 Go binary。

**Consequences**:
- (+) 单 binary 部署，无额外 Nginx/CDN
- (+) 版本一致性（前后端同步发布）
- (-) 前端更新需要重新编译 Go binary

---

### ADR-API-007: v1 API Deprecation Plan

**Status**: Proposed（待多租户 production 验证后启动）

**Decision**: 4 阶段废弃 v1 API：Announce → Migration → Monitor → Sunset。

| Phase | Action | Timeline |
|-------|--------|----------|
| 1. Announce | 注入 `Deprecation: true` + `Sunset` HTTP headers | T+0 |
| 2. Migration | 前端/SDK 切换到 v2 端点，发布迁移指南 | T+2~8w |
| 3. Monitor | v1 使用量追踪中间件，周报跟踪迁移进度 | T+6~14w |
| 4. Sunset | v1 返回 410 Gone → 移除代码 | T+16w+ |

**Scope**: OpenAI 兼容 relay 路由（`/v1/chat/completions` 等）**不在**废弃范围，因其遵循行业标准且通过独立 Token 认证。

**Prerequisites**: v2 功能完备、Zitadel 生产就绪、DB 迁移完成、E2E 测试通过。

---

## 3. Data Architecture / 数据架构

### 3.1 Database Schema

```
PostgreSQL (CNPG) — lurus_api schema
├── tenants                    # 租户主表 (id, zitadel_org_id, slug, status, plan_type)
├── user_identity_mapping      # Zitadel ↔ lurus 用户映射
├── tenant_configs             # 租户级 KV 配置 (config_key, config_value, config_type)
│
├── users                      # 用户 (+tenant_id)
├── tokens                     # API keys (+tenant_id, quota, model limits, group)
├── channels                   # LLM 供应商渠道 (+tenant_id, priority, weight, models)
├── logs                       # API 调用日志 (model, tokens, latency, cost)
│
├── topups                     # 充值记录 (+tenant_id)
├── subscriptions              # 订阅计划 (+tenant_id, auto-renew, daily quota)
├── daily_quota_crons          # 每日配额重置记录
├── redemptions                # 兑换码 (+tenant_id)
└── passkeys / twofa           # 认证凭据 (+tenant_id)
```

**Tenant Isolation**: 所有带 `+tenant_id` 的表通过 GORM TenantPlugin 自动隔离。唯一索引已改为 `(tenant_id, field)` 组合约束。

### 3.2 Redis (db:0)

| Key Pattern | Purpose | TTL |
|-------------|---------|-----|
| `session:*` | User sessions (v1) | 90 days |
| `ratelimit:*` | API rate limiting (token bucket) | auto |
| `cache:model:*` | Model availability cache | 60s |
| `channel:cache` | All channels in-memory mirror | 60s sync |
| `tenant:{id}:user:{uid}` | Tenant-scoped user cache | varies |

### 3.3 Meilisearch (Optional)

| Index | Searchable | Filterable | Fallback |
|-------|-----------|------------|----------|
| logs | content, username, model_name, token_name | type, created_at, user_id, channel_id | DB LIKE query |
| users | username, email, display_name | group, role, status | DB query |
| channels | name, base_url, models, tag | type, status, group | DB query |

初始化失败不阻塞服务启动。`IsEnabled()` + `IsHealthy()` 提供运行时降级判断。

---

## 4. Component Architecture / 组件架构

### 4.1 Request Processing Pipeline

```
Client Request
  ↓
[Gin Router] → route matching (v1 /api/*, v2 /api/v2/:tenant_slug/*, relay /v1/*)
  ↓
[Middleware Stack]
  ├── StatsMiddleware (active connections counter)
  ├── RateLimitMiddleware (global API, critical endpoints, model-level)
  ├── AuthMiddleware
  │   ├── v1: SessionAuth → DefaultTenantMiddleware
  │   ├── v2: ZitadelAuth (JWKS JWT verification → tenant context)
  │   └── relay: TokenAuth (Bearer sk-xxx → user + token context)
  └── ModelRateLimitMiddleware (per-model rate limiting)
  ↓
[Handler (adapter/handler/)] → input validation, context extraction
  ↓
[App Layer (app/)] → business logic (quota, channel select, billing)
  ↓
[Relay Engine (app/relay/)] → creates RelayInfo, selects channel
  ↓
[Provider Factory (adapter/provider/)] → GetAdaptor(channelType) → provider-specific adaptor
  ↓
[Adaptor.DoRequest] → transform to provider format, HTTP call to upstream
  ↓
[Adaptor.DoResponse] → transform response back to OpenAI format
  ↓
[Stream/Buffer] → SSE streaming or JSON buffered response
  ↓
[Post-processing (adapter/repo/)] → RecordConsumeLog, deduct quota, update cache
```

### 4.2 Project Structure (Hexagonal Architecture)

```
lurus-api/
├── cmd/server/main.go                # Entry: errgroup, graceful shutdown (30s timeout)
├── internal/
│   ├── domain/
│   │   └── entity/                   # Domain entities (struct definitions, value objects)
│   │       ├── channel.go            # Channel, ChannelInfo
│   │       ├── user.go               # User
│   │       ├── token.go              # Token
│   │       ├── tenant.go             # Tenant
│   │       ├── log.go                # Log + constants
│   │       └── ...                   # 15+ entity files (GORM tags preserved)
│   ├── app/                          # Use case orchestration (business logic)
│   │   ├── billing.go               # Billing operations
│   │   ├── quota.go                 # Quota calculation
│   │   ├── channel_select.go        # Channel routing logic
│   │   ├── log.go                   # Log queries
│   │   ├── notify-limit.go          # Usage alerts
│   │   ├── token_service.go         # Token CRUD
│   │   ├── user_service.go          # User CRUD
│   │   ├── passkey/                 # Passkey authentication service
│   │   └── relay/                   # LLM relay engine
│   │       ├── text.go              # TextHelper, request processing
│   │       ├── claude.go            # Claude-specific handler
│   │       ├── gemini.go            # Gemini-specific handler
│   │       ├── task.go              # Task relay
│   │       └── helper/              # StreamScanner, model mapping
│   ├── adapter/
│   │   ├── handler/                 # HTTP handlers (controllers, v1 + v2)
│   │   │   ├── v2_*.go              # v2 multi-tenant controllers
│   │   │   ├── oauth.go             # Zitadel OAuth flow
│   │   │   ├── tenant.go            # Platform admin tenant mgmt
│   │   │   └── router/              # Route definitions
│   │   │       ├── api-router.go    # v1 routes
│   │   │       └── api-v2-router.go # v2 multi-tenant routes
│   │   ├── middleware/              # Auth, rate-limit, tenant, stats
│   │   │   ├── auth.go              # v1 session auth
│   │   │   ├── zitadel_auth.go      # v2 JWT + JWKS manager
│   │   │   ├── rate-limit.go        # Global + endpoint rate limiting
│   │   │   └── model-rate-limit.go  # Per-model rate limiting
│   │   ├── repo/                    # GORM repositories (data access)
│   │   │   ├── tenant.go, tenant_plugin.go, tenant_context.go
│   │   │   ├── channel.go, channel_cache.go
│   │   │   ├── user.go, user_cache.go
│   │   │   ├── token.go, log.go
│   │   │   └── topup.go, subscription.go, redemption.go
│   │   └── provider/               # AI vendor adaptors (30+)
│   │       ├── adaptor.go           # Adaptor interface
│   │       ├── base_adaptor.go      # BaseAdaptor (14 default methods)
│   │       ├── factory.go           # Adaptor factory (GetAdaptor)
│   │       ├── common/              # Shared relay utilities
│   │       ├── openai/              # OpenAI adaptor
│   │       ├── claude/              # Anthropic adaptor
│   │       ├── gemini/              # Google Gemini
│   │       └── ...                  # 30+ more
│   ├── lifecycle/                   # errgroup lifecycle, ticker tasks
│   └── pkg/
│       ├── common/                  # SysLog, Redis, SafeGo, slog, pprof
│       ├── config/                  # Centralized env config
│       ├── constant/                # API types, channel types
│       ├── logger/                  # slog setup (JSON prod / text dev)
│       ├── search/                  # Meilisearch client + sync
│       ├── setting/                 # Model ratio, operation settings
│       └── types/                   # Error types
├── web/                             # React 18 + Vite + Semi UI (embedded via go:embed)
├── deploy/k8s/                      # K8s manifests (deployment, service, ingress, secrets)
└── doc/                             # Documentation + ADRs
```

**Dependency Direction / 依赖方向**:
```
adapter/handler/ → app/         → adapter/repo/ → domain/entity/
                 → app/relay/   → adapter/provider/ → domain/entity/
adapter/middleware/ → app/                            ↑
                                                 pkg/ (shared by all layers)
```

### 4.3 Lifecycle Management

```go
// cmd/server/main.go — errgroup orchestration
g, ctx := errgroup.WithContext(context.Background())

g.Go(httpServer.ListenAndServe)         // HTTP server
g.Go(gracefulShutdownHandler)           // SIGTERM → 30s drain
g.Go(channelCacheSyncTicker)            // Channel cache periodic sync
g.Go(dailyQuotaCronTicker)             // Subscription daily quota reset
g.Go(meilisearchSyncTicker)            // Search index sync (if enabled)
```

- SafeGo / SafeGoWithContext: goroutine 级 panic recovery
- 所有后台 goroutine 通过 `ctx.Done()` 退出

---

## 5. API Architecture / API 接口架构

### 5.1 v1 API (Session Auth, Default Tenant)

**Public**: `/api/setup`, `/api/status`, `/api/notice`, `/api/pricing`

**Auth**: `/api/user/register`, `/api/user/login`, `/api/oauth/{provider}`, `/api/user/login/2fa`

**User Self-Service** (SessionAuth): Profile CRUD, token management, topup, subscription, 2FA, passkey, checkin

**Admin** (AdminAuth): User/channel/token/redemption/log CRUD

**Root** (RootAuth): System options, login config

**Relay** (TokenAuth — Bearer sk-xxx):
- `POST /v1/chat/completions` — Chat (streaming SSE supported)
- `POST /v1/embeddings` — Embeddings
- `POST /v1/images/generations` — Image gen
- `POST /v1/audio/speech|transcriptions` — TTS/STT
- `GET /v1/models` — List models
- Midjourney, Suno, Video generation (task-based)

### 5.2 v2 API (Zitadel JWT, Multi-Tenant)

**OAuth** (no auth):
- `GET /api/v2/:tenant_slug/auth/login` — Zitadel login redirect
- `GET /api/v2/oauth/callback` — OAuth callback (code exchange)
- `POST /api/v2/oauth/logout` — OIDC end_session
- `POST /api/v2/oauth/refresh` — Token refresh

**Tenant Routes** (ZitadelAuth middleware):
- User: `GET|PUT /api/v2/:ts/user/me`
- Tokens: CRUD `/api/v2/:ts/tokens/*`
- Channels: CRUD `/api/v2/:ts/channels/*` (admin role)
- Billing: TopUp, Subscribe `/api/v2/:ts/billing/*`
- Logs: `/api/v2/:ts/logs/*`
- Redemptions: `/api/v2/:ts/redemptions/*`

**Platform Admin** (RootAuth):
- Tenant CRUD: `/api/v2/admin/tenants/*`
- User mappings: `/api/v2/admin/mappings/*`
- System stats: `GET /api/v2/admin/stats`

### 5.3 Auth Model Matrix

| Route | Auth Method | Tenant Source |
|-------|-----------|--------------|
| v1 `/api/*` | Session cookie | Fixed: "default" |
| v2 `/api/v2/:ts/*` | Zitadel JWT (JWKS) | JWT org_id → tenant_id |
| v2 `/api/v2/admin/*` | v1 Session + Root role | Cross-tenant (no filter) |
| Relay `/v1/*` | Bearer Token (sk-xxx) | Token's owner tenant |

---

## 6. Security Architecture / 安全架构

### 6.1 Authentication

- **v1 Session**: `gin-contrib/sessions` cookie store, `SESSION_SECURE` env for HTTPS
- **v2 JWT**: JWKS public key verification, issuer + expiration + audience validation
- **PKCE**: OAuth authorization code flow with `SHA-256` code_challenge
- **ID Token Nonce**: 防止 token replay attack
- **JWKS Race Protection**: `sync.RWMutex` + 30s minimum refresh interval

### 6.2 Tenant Data Isolation

- GORM TenantPlugin: 自动注入 `WHERE tenant_id = ?`（Query/Create/Update/Delete）
- Webhook Tenant Verification: Stripe/Creem/Epay webhook 校验租户归属
- 25+ 租户隔离测试用例

### 6.3 Rate Limiting

| Level | Scope | Implementation |
|-------|-------|---------------|
| Global API | All endpoints | Redis token bucket (configurable) |
| Critical endpoints | Register, login, password reset | Stricter per-IP limits |
| Model-level | Per model per token | Configurable per-model rate |

### 6.4 Data Protection

- Credentials in K8s Secrets (not in code/Git)
- API keys masked in list endpoints
- Password: bcrypt hash
- Token key: SHA-256 hash for lookup, stored hashed
- Turnstile captcha on registration/login

---

## 7. Deployment Architecture / 部署架构

### 7.1 Current Topology

```
K3s Cluster → lurus-system namespace
├── lurus-api Deployment (replicas: 1 → planned: 2)
│   ├── Container: scratch base, CGO_ENABLED=0
│   ├── Port: 8850
│   ├── Resources: CPU 250-500m, Memory 256Mi-1Gi
│   └── Node: cloud-ubuntu-1 (master, 16C/32G)
├── Service: ClusterIP
└── IngressRoute: api.lurus.cn → Traefik → TLS termination
```

### 7.2 GitOps Pipeline

```
Code Push → GitHub Actions
  ├── go test ./...
  ├── CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath
  ├── docker build (multi-stage, scratch base)
  └── docker push ghcr.io/LurusTech/lurus-api:latest
        ↓
ArgoCD Sync (watches deploy/k8s/)
        ↓
K3s Rolling Update (lurus-system namespace)
```

### 7.3 HA Plan (ADR, Proposed)

| Component | Current | Target |
|-----------|---------|--------|
| API Pods | 1 | 2 (min) + HPA to 5 |
| Session Store | Cookie | Redis-backed |
| PostgreSQL | CNPG (1 primary) | CNPG (1 primary + 1 replica) |
| Redis | Single instance | Sentinel (1 primary + 1 replica + 3 sentinels) |
| Health Probes | `/api/status` | `/healthz` (liveness) + `/readyz` (readiness) |
| Shutdown | 30s graceful | + 5s preStop hook |

Key: `maxUnavailable: 0` + `maxSurge: 1` 确保零宕机滚动更新。SSE 长连接通过禁用 Traefik 响应缓冲正确处理。

---

## 8. Observability Plan / 可观测性规划

**Status**: Proposed（ADR 已编写，待实施）

### 8.1 Current State

| Capability | Status |
|-----------|--------|
| Structured logging (slog) | Implemented (JSON prod / text dev) |
| pprof profiling | Implemented (ENABLE_PPROF=true) |
| Active connections counter | Implemented (StatsMiddleware) |
| Distributed tracing | None |
| Prometheus metrics | None |
| Alerting | None |

### 8.2 Target Architecture (OpenTelemetry)

```
Application (OTel SDK)
  ├── Traces → OTLP Exporter → Grafana Tempo / Jaeger
  ├── Metrics → Prometheus Exporter → /metrics → Prometheus → Grafana
  └── Logs → slog + trace_id injection → Loki
```

**Key Metrics**: `lurus_http_requests_total`, `lurus_relay_request_duration_seconds`, `lurus_relay_first_token_latency_seconds`, `lurus_billing_quota_consumed_total`, `lurus_relay_active_streams`

**Sampling**: 10% normal, 100% errors + P99 latency, 0% health checks

**Implementation**: 3 phases — Metrics Foundation → Distributed Tracing → Dashboards & Alerting. 所有功能通过 `OTEL_METRICS_ENABLED` / `OTEL_TRACING_ENABLED` env var 可完全关闭。

---

## 9. Scalability Analysis / 可扩展性分析

### 9.1 Extension Points

| Aspect | Mechanism |
|--------|----------|
| Add LLM provider | Implement Adaptor interface (14 methods, BaseAdaptor defaults) |
| Add task platform | Implement TaskAdaptor interface |
| Add tenant | Zitadel Organization → auto-create in lurus-api |
| Add payment gateway | Implement webhook handler + controller |
| Horizontal scaling | Stateless app → increase replicas |

### 9.2 Performance Targets (from PRD)

| Metric | Current | Target |
|--------|---------|--------|
| Gateway overhead p95 | ~80ms | < 50ms |
| Streaming first-byte overhead | unknown | < 20ms |
| Concurrent connections | ~500 | 1000+ |
| Monthly uptime | ~98% | >= 99.5% |

**Optimization targets**: Channel cache hot path, token validation caching, model ratio precomputation, StreamScanner buffer allocation.

---

## 10. Technology Radar / 技术雷达

| Technology | Ring | Notes |
|-----------|------|-------|
| Go 1.25 + Gin | **Adopt** | Core runtime |
| GORM + PostgreSQL | **Adopt** | ORM + primary DB |
| Redis (go-redis/v9) | **Adopt** | Cache + rate limiting |
| golang-jwt/v5 | **Adopt** | JWT verification |
| React 18 + Vite + Semi UI | **Adopt** | Embedded SPA |
| Meilisearch | **Adopt** | Optional full-text search |
| Zitadel OIDC | **Trial** | Multi-tenant auth (pending production verification) |
| OpenTelemetry | **Assess** | Planned observability stack |
| Prometheus + Grafana | **Assess** | Planned metrics + dashboards |
| Redis Sentinel | **Assess** | Planned HA for Redis |

---

## 11. Glossary / 术语表

| Term | Definition |
|------|-----------|
| **Channel** | LLM 供应商配置实例 (API key, endpoint, models, priority, weight) |
| **Adaptor** | 供应商特定的请求/响应转换实现 |
| **Relay** | 客户端请求转发到上游 LLM 供应商的过程 |
| **Token** | API 访问密钥 (sk-xxx)，含配额、模型限制、分组 |
| **Quota** | 统一消费单位，按 model ratio 折算实际 token 消耗 |
| **Group** | 渠道选择和优先级路由的逻辑分组 |
| **Tenant** | 隔离的业务实体，映射到 Zitadel Organization |
| **Tenant Slug** | URL 友好的租户标识 (e.g., "lurus", "customer-a") |
| **BaseAdaptor** | Adaptor 接口的默认实现 (14 methods) |
| **Model Ratio** | 模型成本乘数 (input_ratio + output_ratio) |
| **JWKS** | JSON Web Key Set — JWT 签名公钥集 |
| **PKCE** | Proof Key for Code Exchange — OAuth 安全增强 |
| **TenantPlugin** | GORM 插件，自动注入 tenant_id WHERE 条件 |
