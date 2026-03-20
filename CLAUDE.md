# lurus-api

LLM 统一网关（Multi-tenant AI Gateway）。OpenAI 兼容接口代理 30+ 供应商，内置计费、配额、渠道管理、多租户、全文搜索。

- **Module**: `github.com/QuantumNous/lurus-api`
- **Namespace / Port**: `lurus-system` / pod:3000, svc:8850
- **Image**: `ghcr.io/hanmahong5-arch/lurus-api:main`
- **DB**: PostgreSQL (`lurus_api` schema, GORM auto-migrate) + SQLite fallback (dev only), Redis DB 0, Meilisearch（可选）
- **Auth**: Zitadel OIDC (`auth.lurus.cn`)，Passkey，session cookie/Redis

## Tech Stack

| Layer | Tech |
|-------|------|
| Backend | Go 1.25.1, Gin, GORM |
| Frontend | React 18, Vite, Semi UI (`web/`), Bun |
| DB | PostgreSQL / MySQL / SQLite |
| Cache | Redis DB 0 (session + channel cache + quota sync) |
| Search | Meilisearch (log full-text, 可选) |
| Observability | Prometheus `/metrics`, OpenTelemetry traces (OTLP), pprof :8005, Pyroscope |
| Providers | 30+ LLM vendors（详见 AI Providers 章节） |

## Directory Structure

```
cmd/server/main.go           # 启动、路由注册、graceful shutdown
internal/
├── domain/entity/           # 领域实体 (channel, user, log, token, tenant, task, midjourney…)
├── app/                     # 业务编排
│   ├── relay/               # 各模态请求分发 (audio/claude/compatible/embedding/gemini/image/mjproxy/rerank/responses/websocket)
│   │   └── helper/          # 公共 relay 工具 (price, model_mapped, stream_scanner, valid_request)
│   ├── passkey/             # WebAuthn passkey (service/session/user)
│   ├── billing_service.go   # 配额显示换算 (USD/CNY/Tokens)
│   ├── quota.go             # 配额扣减与预扣
│   ├── pre_consume_quota.go # relay 前置配额检查
│   ├── channel.go           # 渠道选择与测试
│   ├── channel_select.go    # 渠道负载均衡
│   ├── token_service.go     # API Token 管理
│   ├── log_service.go       # 日志写入
│   ├── midjourney.go        # MJ 任务代理
│   ├── user_service.go      # 用户业务逻辑
│   └── sensitive.go         # 敏感词过滤
├── adapter/
│   ├── handler/             # HTTP controllers
│   │   ├── router/          # 路由注册 (api-router / api-v2-router / relay-router / internal-api-router / web-router / video-router / dashboard)
│   │   ├── relay.go         # Relay 入口
│   │   ├── internal_api.go  # /internal/* handlers
│   │   ├── internal_api_ext.go  # /internal/* 扩展 (quota/balance)
│   │   ├── billing.go       # 订阅/用量 handlers
│   │   ├── v2_*.go          # v2 多租户 handlers
│   │   └── *.go             # 其余业务 handlers
│   ├── middleware/          # Gin middleware
│   │   ├── auth.go          # UserAuth / AdminAuth / RootAuth
│   │   ├── zitadel_auth.go  # Zitadel JWT 验证 (v2 API)
│   │   ├── internal_api_auth.go  # InternalApiAuth + RequireScope
│   │   ├── distributor.go   # Distribute (渠道分配)
│   │   ├── rate-limit.go / model-rate-limit.go / email-verification-rate-limit.go
│   │   ├── cors.go          # CORS
│   │   ├── stats.go         # 请求统计
│   │   └── *.go             # secure_verification, sensitive_action, turnstile…
│   ├── repo/                # GORM repositories
│   │   ├── channel.go / channel_cache.go
│   │   ├── user.go / user_cache.go / user_mapping.go
│   │   ├── token.go / token_cache.go
│   │   ├── log.go
│   │   ├── tenant.go / tenant_config.go / tenant_context.go / tenant_plugin.go
│   │   ├── internal_api_key.go  # Scoped API keys
│   │   ├── daily_quota_cron.go
│   │   └── *.go             # ability, checkin, midjourney, pricing, task…
│   └── provider/            # AI 供应商适配器
│       ├── common/          # relay_info, relay_utils
│       ├── constant/        # relay_mode
│       └── <vendor>/        # openai, claude, gemini, aws, baidu, cloudflare, cohere, coze, dify,
│                            # jina, minimax, mokaai, ollama, palm, perplexity, siliconflow,
│                            # tencent, vertex, xunfei, zhipu, zhipu_4v
├── lifecycle/lifecycle.go   # Task interface + Manager + TickerTask (background task lifecycle)
└── pkg/
    ├── common/              # 全局变量, Redis client, identity_client.go (HTTP), identity_grpc_client.go (gRPC→HTTP fallback)
    ├── config/config.go     # 集中式配置 (从 env 加载, 启动时 fast-fail)
    ├── constant/            # api_type, azure, cache_key, channel, context_key, endpoint_type, env, finish_reason, midjourney, multi_key_mode, setup, task
    ├── dto/                 # 请求/响应 DTOs (audio, claude, embedding, gemini, openai_*, rerank, video…)
    ├── types/               # 共享类型 (channel_error, relay_format, request_meta, rw_map, set, file_data, price_data)
    ├── logger/logger.go     # 结构化日志
    ├── metrics/             # Prometheus metrics + Gin middleware
    ├── pool/pool.go         # goroutine pool
    ├── search/              # Meilisearch (client, logs_index, channels_index, users_index, sync)
    ├── setting/             # 运行时热更新设置
    │   ├── ratio_setting/   # model_ratio, group_ratio, cache_ratio, expose_ratio, model_family
    │   ├── model_setting/   # global, claude, gemini
    │   ├── operation_setting/ # general, quota, monitor, checkin, tools
    │   ├── system_setting/  # oidc, passkey, discord, legal, fetch_setting
    │   ├── console_setting/ # config, validation
    │   ├── reasoning/       # suffix
    │   └── *.go             # auto_group, chat, midjourney, rate_limit, sensitive, user_usable_group
    └── tracing/             # OpenTelemetry tracing + Gin middleware
web/                         # React frontend (Bun)
migrations/                  # PostgreSQL SQL migrations (001-004)
deploy/k8s/                  # K8s manifests (deployment, service, ingress, hpa, pdb, servicemonitor, kustomization)
deploy/k8s/staging/          # Staging overlay
pkg/ionet/                   # io.net 客户端 (client, container, deployment, hardware)
doc/runbook/                 # 运维 runbooks (database, deployment, ha-deployment, incident-response, staging, tenant-onboarding)
doc/decisions/               # 架构决策记录 (ha-deployment, observability, v1-deprecation)
doc/process.md               # 变更日志
```

## Commands

```bash
# --- Local Dev ---
cp .env.example .env                        # 复制并填写 SQL_DSN, REDIS_CONN_STRING, SESSION_SECRET
go run ./cmd/server                         # 后端 port 3000
cd web && bun install && bun run dev        # 前端 port 5173 (代理到 3000)

# --- Build (production) ---
CGO_ENABLED=0 go build -ldflags "-s -w -X 'github.com/QuantumNous/lurus-api/internal/pkg/common.Version=$(cat VERSION)'" -o lurus-api ./cmd/server

# --- Frontend ---
cd web && bun run typecheck
cd web && bun run lint
cd web && bun run build

# --- Test ---
go test -short ./...                        # 单元测试（跳过集成测试）
go test -v -race ./...                      # 全量 + 竞态检测（merge 前必跑）
go test -v ./internal/adapter/handler/...  # 指定包
go test -run Integration ./...             # 仅集成测试
go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

# --- K8s ---
ssh root@100.98.57.55 "kubectl get pods -n lurus-system"
ssh root@100.98.57.55 "kubectl rollout restart deployment/lurus-api -n lurus-system"
ssh root@100.98.57.55 "kubectl logs -n lurus-system -l app=lurus-api --tail=100"
ssh root@100.98.57.55 "kubectl describe pod -n lurus-system <pod>"
```

## K8s Deployment Facts

| Key | Value |
|-----|-------|
| nodeSelector | `lurus.cn/vpn: "true"` |
| Resources | req: 256Mi/100m  lim: 1Gi/500m |
| Security | runAsUser:65534, readOnlyRootFilesystem, drop ALL caps |
| Volumes | `data: emptyDir`, `tmp: emptyDir` (no persistent disk) |
| Redis | `redis://redis:6379` (in-cluster) |
| Meilisearch | `http://meilisearch:7700` (in-cluster) |
| Outbound proxy | `http://10.42.1.1:10808` (for Gemini/OpenAI/外网 LLM) |
| NO_PROXY | `*.svc,*.svc.cluster.local,*.lurus.cn,10.0.0.0/8,…` |
| ALLOWED_ORIGINS | `https://www.lurus.cn,https://gushen.lurus.cn,https://webmail.lurus.cn` |
| MODEL_SYNC_FREQUENCY | `60` (分钟) |
| Secret | `lurus-api-secrets` (SESSION_SECRET, SQL_DSN, ZITADEL_CLIENT_ID, IDENTITY_SESSION_SECRET, IDENTITY_SERVICE_INTERNAL_KEY, ALIPAY_*) |

## Environment Variables

### Required

| Variable | Description |
|----------|-------------|
| `SQL_DSN` | PostgreSQL: `postgresql://user:pass@host/db`；MySQL: `user:pass@tcp(host:3306)/db` |
| `SESSION_SECRET` | Session 签名密钥（多节点必须一致） |
| `REDIS_CONN_STRING` | `redis://redis:6379`，缺失则退化为 cookie session |

### lurus-platform Integration

| Variable | Default | Description |
|----------|---------|-------------|
| `IDENTITY_SERVICE_URL` | `http://platform-core.lurus-platform.svc.cluster.local:18104` | HTTP 地址 |
| `IDENTITY_GRPC_ADDR` | `platform-core.lurus-platform.svc.cluster.local:18105` | gRPC 地址（自动 HTTP fallback） |
| `IDENTITY_SERVICE_INTERNAL_KEY` | — | platform `/internal/v1/*` bearer token |
| `IDENTITY_SESSION_SECRET` | — | 与 lurus-platform 共享的 session token 验签密钥 |
| `IDENTITY_AUTH_REDIRECT` | `false` | `true` → register/login/topup 重定向到 identity |
| `IDENTITY_PUBLIC_URL` | `https://identity.lurus.cn` | 用于 redirect URL 构造 |

### Zitadel OIDC (v2 API)

| Variable | Default | Description |
|----------|---------|-------------|
| `ZITADEL_ENABLED` | `false` | 启用 Zitadel OIDC |
| `ZITADEL_ISSUER` | — | `https://auth.lurus.cn` |
| `ZITADEL_JWKS_URI` | — | `https://auth.lurus.cn/oauth/v2/keys` |
| `ZITADEL_CLIENT_ID` | — | Zitadel app client ID |
| `ZITADEL_REDIRECT_URI` | — | 生产: `https://api.lurus.cn/api/v2/oauth/callback` |
| `ZITADEL_POST_LOGOUT_REDIRECT_URI` | — | 登出后跳转 URL |
| `ZITADEL_ALLOWED_REDIRECT_DOMAINS` | — | `lurus.cn,api.lurus.cn` |
| `ZITADEL_ENABLE_PKCE` | `false` | 启用 PKCE |
| `ZITADEL_AUTO_CREATE_USER` | `false` | OIDC 登录自动建用户 |
| `ZITADEL_AUTO_CREATE_TENANT` | `false` | OIDC 登录自动建租户 |

### Meilisearch (Optional)

| Variable | Default | Description |
|----------|---------|-------------|
| `MEILISEARCH_ENABLED` | `false` | 启用 Meilisearch |
| `MEILISEARCH_HOST` | — | `http://meilisearch:7700` |
| `MEILISEARCH_API_KEY` | — | Master key |
| `MEILISEARCH_SYNC_ENABLED` | `false` | 启用日志同步 |
| `MEILISEARCH_SYNC_WORKERS` | `32` | 同步并发数 |
| `MEILISEARCH_SYNC_BATCH_SIZE` | `1000` | 批次大小 |
| `MEILISEARCH_WORKER_COUNT` | `2` | 生产用 worker 数 |

### Runtime Tuning

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3000` | HTTP 监听端口 |
| `GIN_MODE` | `debug` | `release` 关闭 debug 输出 |
| `DEBUG` | `false` | 启用 debug 日志 |
| `NODE_TYPE` | `master` | `slave` 禁用 master-only 任务 |
| `SYNC_FREQUENCY` | `60` | 缓存同步周期（秒） |
| `RELAY_TIMEOUT` | `0` | Relay 超时（秒，0=不限） |
| `RELAY_MAX_IDLE_CONNS` | `500` | HTTP 连接池最大空闲连接 |
| `BATCH_UPDATE_ENABLED` | `false` | 启用批量数据库写入 |
| `BATCH_UPDATE_INTERVAL` | `5` | 批量写入间隔（秒） |
| `DAILY_QUOTA_ENABLED` | `true` | `false` 禁用每日配额重置 |
| `CHANNEL_UPDATE_FREQUENCY` | — | 渠道自动更新频率（分钟） |
| `MODEL_SYNC_FREQUENCY` | — | 模型自动同步频率（分钟） |
| `MEMORY_CACHE_ENABLED` | `false` | 启用内存缓存（Redis 存在时自动开启） |
| `STREAMING_TIMEOUT` | `300` | 流式请求无响应超时（秒） |
| `MAX_REQUEST_BODY_MB` | `64` | 请求体最大大小（MB） |
| `GRACEFUL_SHUTDOWN_TIMEOUT` | `30s` | 优雅停机等待时间 |
| `ALLOWED_ORIGINS` | 见 config.go | CORS 允许域名（逗号分隔） |
| `FRONTEND_BASE_URL` | — | slave 节点将前端路由重定向到此 URL |
| `MINIO_RELEASES_BUCKET` | `lurus-releases` | Release 文件的 MinIO bucket |

### Observability (Optional)

| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_TRACING_ENABLED` | `false` | 启用 OpenTelemetry traces |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | `jaeger-collector.observability.svc:4318` |
| `OTEL_EXPORTER_OTLP_INSECURE` | — | `true` |
| `OTEL_TRACE_SAMPLE_RATE` | `0.1` | 采样率（0.0~1.0） |
| `OTEL_ENVIRONMENT` | — | `production` |
| `ENABLE_PPROF` | `false` | 启用 pprof（port 8005） |
| `LOG_FORMAT` | — | `json` 启用结构化日志 |
| `LOG_LEVEL` | — | 日志级别 |

### Proxy (For External LLM APIs)

| Variable | Value (production) | Description |
|----------|--------------------|-------------|
| `HTTP_PROXY` / `http_proxy` | `http://10.42.1.1:10808` | 出站代理（访问 OpenAI/Gemini 等） |
| `HTTPS_PROXY` / `https_proxy` | `http://10.42.1.1:10808` | — |
| `NO_PROXY` / `no_proxy` | `localhost,127.0.0.1,10.0.0.0/8,*.svc,*.lurus.cn…` | 内网绕过代理 |

### OAuth Providers (Optional)

`GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`, `ALIPAY_PRIVATE_KEY`, `ALIPAY_PUBLIC_KEY`,
`WECHAT_SERVER_ADDRESS`, `WECHAT_SERVER_TOKEN`, `TELEGRAM_BOT_TOKEN`,
`UMAMI_WEBSITE_ID`, `GOOGLE_ANALYTICS_ID`

## Route Groups

| Group | Auth | Description |
|-------|------|-------------|
| `GET /api/status` | public | Healthcheck → `{"success": true}` |
| `/api/*` | session (v1) | 用户/管理/渠道/Token 等 v1 API |
| `/api/v2/:tenant_slug/*` | Zitadel JWT | 多租户 v2 API (渠道/Token/日志/配置/兑换码) |
| `/api/v2/oauth/*` | public | Zitadel OAuth callback/logout/refresh |
| `/api/v2/switch/*` | public | lurus-switch 版本查询 + 预设库 |
| `/api/v2/user/identity-overview` | Zitadel JWT | VIP/钱包/订阅信息（来自 platform） |
| `/api/v2/admin/*` | v1 session + RootAuth | 平台管理员：租户管理/用户映射/统计 |
| `/v1/*` | Token auth | Relay: chat/completions, messages(Claude), responses, images, audio, embeddings, rerank, realtime(WS) |
| `/v1beta/models/*` | Token auth | Gemini relay |
| `/mj/*`, `/:mode/mj/*` | Token auth | Midjourney relay |
| `/suno/*` | Token auth | Suno task relay |
| `/pg/chat/completions` | User session | Playground |
| `/internal/*` | API Key + Scope | 服务内通信（见 Internal API Scopes） |
| `GET /metrics` | public | Prometheus scrape |

## Internal API Scopes

路径前缀 `/internal`，需 `Authorization: Bearer <key>` + scope 匹配（`repo.ScopeXxx`）。

| Scope | Endpoints |
|-------|-----------|
| `ScopeUserRead` | `GET /internal/user/:id`, `/user/by-email/:email`, `/user/by-phone/:phone` |
| `ScopeUserWrite` | `PUT /internal/user/:id` |
| `ScopeQuotaRead` | `GET /internal/quota/user/:id` |
| `ScopeQuotaWrite` | `POST /internal/quota/adjust` |
| `ScopeBalanceRead` | `GET /internal/balance/user/:id` |
| `ScopeBalanceWrite` | `POST /internal/balance/topup` |

## lurus-platform gRPC Integration

`internal/pkg/common/identity_grpc_client.go` — singleton gRPC client，连接失败自动 fallback 到 HTTP。

| Function | gRPC Method | Description |
|----------|-------------|-------------|
| `GetAccountByZitadelSubGRPC` | `GetAccountByZitadelSub` | 通过 Zitadel sub 查账户 |
| `UpsertAccountGRPC` | `UpsertAccount` | 创建或更新账户（OIDC 首次登录） |
| `GetEntitlementsGRPC` | `GetEntitlements` | 查权益（产品功能开关） |
| `GetAccountOverviewGRPC` | `GetAccountOverview` | 聚合：账户 + VIP + 钱包 + 订阅 |
| `ReportLLMUsageGRPC` | `ReportUsage` | 上报 LLM 用量 (amountCNY) |
| `DebitWalletGRPC` | `WalletDebit` | 钱包扣款（消费 LLM 时） |
| `CreditWalletGRPC` | `WalletCredit` | 钱包充值 |

gRPC auth: Bearer token in metadata `authorization` header (同 `IDENTITY_SERVICE_INTERNAL_KEY`)。

## Relay Formats

`internal/pkg/types/relay_format.go` 定义，`handler.Relay(c, types.RelayFormatXxx)` 分发。

| RelayFormat | Endpoint | Notes |
|-------------|----------|-------|
| `RelayFormatOpenAI` | `/v1/chat/completions`, `/v1/completions` | 主流 chat |
| `RelayFormatClaude` | `/v1/messages` | Anthropic 原生格式 |
| `RelayFormatGemini` | `/v1beta/models/*`, `/v1/models/*path` | Gemini 原生格式 |
| `RelayFormatOpenAIResponses` | `/v1/responses` | OpenAI Responses API |
| `RelayFormatOpenAIImage` | `/v1/images/generations`, `/v1/images/edits`, `/v1/edits` | 图像生成 |
| `RelayFormatEmbedding` | `/v1/embeddings`, `/v1/engines/:model/embeddings` | Embeddings |
| `RelayFormatOpenAIAudio` | `/v1/audio/*` | TTS/ASR |
| `RelayFormatRerank` | `/v1/rerank` | Rerank |
| `RelayFormatOpenAIRealtime` | `GET /v1/realtime` (WebSocket) | Realtime API |

Midjourney/Suno 通过独立 handler 处理（`handler.RelayMidjourney`, `handler.RelayTask`）。

## AI Providers

`internal/adapter/provider/<vendor>/` 目录：

`openai`, `claude` (Anthropic), `gemini`, `aws` (Bedrock), `baidu`, `cloudflare`,
`cohere`, `coze`, `dify`, `jina`, `minimax`, `mokaai`, `ollama`, `palm`,
`perplexity`, `siliconflow`, `tencent`, `vertex`, `xunfei`, `zhipu`, `zhipu_4v`

## Key Runtime Notes

- **DB 自动迁移**: 启动时 GORM 自动建表（Go 结构体驱动）；PostgreSQL 手动 SQL migration 在 `migrations/` (001-004)
- **SQLite fallback**: 未设置 `SQL_DSN` 时使用 SQLite (`one-api.db`)，仅用于开发
- **渠道缓存**: Redis 存在时自动启用内存缓存，`SYNC_FREQUENCY` 控制同步周期
- **Background tasks**: `lifecycle.Manager` 管理，`TickerTask` 封装定时任务
- **ProtoImport**: 通过独立模块 `lurus-proto-go` 引用 identity gRPC 契约类型（`github.com/hanmahong5-arch/lurus-proto-go/identity/v1`）
- **go.mod replace**: `github.com/hanmahong5-arch/lurus-proto-go => ../shared/lurus-proto-go`（本地开发；发布到 GitHub 后移除）

## BMAD

| Resource | Path |
|----------|------|
| PRD | `./_bmad-output/planning-artifacts/prd.md` |
| Epics | `./_bmad-output/planning-artifacts/epics.md` |
| Architecture | `./_bmad-output/planning-artifacts/architecture.md` |
| Sprint Status | `./_bmad-output/planning-artifacts/sprint-status.yaml` |
| Project Context | `./_bmad-output/planning-artifacts/project-context.md` |

**Story 文档规则（Epic 6+ 严格执行）**: 实现前建 story 文档 → 通过 `dev-story/checklist.md` → 含验证证据才可标 done。违反 = 工作无效。
