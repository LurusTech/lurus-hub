# lurus-api

LLM 统一网关（Multi-tenant AI Gateway）。OpenAI 兼容接口代理 30+ 供应商，内置计费、配额、渠道管理、全文搜索。

- **Module**: `github.com/QuantumNous/lurus-api`
- **Namespace / Port**: `lurus-system` / pod:3000, svc:8850
- **Image**: `ghcr.io/hanmahong5-arch/lurus-api:main`
- **DB**: PostgreSQL (`lurus_api` schema) + SQLite fallback, Redis DB 默认共享, Meilisearch（可选）
- **Auth**: Zitadel OIDC (`auth.lurus.cn`)，Passkey，session cookie/Redis

## Tech Stack

| Layer | Tech |
|-------|------|
| Backend | Go 1.25, Gin, GORM |
| Frontend | React 18, Vite, Semi UI (`web/`), Bun |
| DB | PostgreSQL / MySQL / SQLite (GORM auto-migrate) |
| Cache | Redis (session store + channel cache + quota sync) |
| Search | Meilisearch (log full-text search，可选) |
| Observability | Prometheus metrics, OpenTelemetry traces (OTLP), pprof, Pyroscope |
| Providers | 30+ LLM vendors: OpenAI, Claude, Gemini, DeepSeek, Zhipu, Qwen, Coze, Dify, AWS Bedrock, Vertex, Volcengine… |

## Directory Structure

```
cmd/server/              # main.go — 启动、路由注册、graceful shutdown
internal/
├── domain/entity/       # 领域实体 (struct definitions, value objects)
├── app/                 # 业务编排 (relay handlers, passkey, billing services)
│   ├── relay/           # 各模态请求分发 (chat/image/audio/embedding/rerank/responses)
│   └── passkey/         # WebAuthn passkey service
├── adapter/
│   ├── handler/         # HTTP controllers (channel/token/user/relay/log/admin…)
│   │   └── router/      # 路由注册 (api-router, v2-router, relay-router, internal-api-router)
│   ├── middleware/       # Gin middleware (auth, Zitadel OIDC, rate-limit, CORS, gzip…)
│   ├── repo/            # GORM repositories (channel, user, log, option, daily-quota-cron…)
│   └── provider/        # AI 供应商适配器 (openai/claude/gemini/deepseek/…)
├── lifecycle/           # App lifecycle management
└── pkg/
    ├── common/          # 全局变量、Redis、env 解析、identity_client
    ├── config/          # 集中式配置 (config.Get()，从 env 加载)
    ├── setting/         # 运行时热更新设置 (ratio, model, rate-limit, system…)
    ├── logger/          # 结构化日志
    ├── search/          # Meilisearch 封装
    ├── metrics/         # Prometheus metrics
    └── tracing/         # OpenTelemetry tracing
web/                     # React frontend (bun)
migrations/              # SQL migrations (PostgreSQL)
deploy/k8s/              # K8s manifests (deployment, service, ingress, hpa, pdb)
doc/runbook/             # Operational runbooks
```

## Commands

```bash
# --- Local Dev ---
cp .env.example .env                          # 复制并填写 SQL_DSN, REDIS_CONN_STRING, SESSION_SECRET
go run ./cmd/server                           # 启动后端 (port 3000)
cd web && bun install && bun run dev          # 启动前端 (port 5173, 代理到 3000)
make all                                      # 构建前端 + 启动后端

# --- Build ---
CGO_ENABLED=0 go build -ldflags "-s -w -X 'github.com/QuantumNous/lurus-api/internal/pkg/common.Version=$(cat VERSION)'" -o lurus-api ./cmd/server

# --- Frontend ---
cd web && bun run typecheck
cd web && bun run lint
cd web && bun run build

# --- Test ---
go test -short ./...                          # 单元测试（跳过集成测试）
go test ./...                                 # 全部（需要 DB/Redis/Meilisearch）
go test -v ./internal/adapter/handler/        # 指定包
go test -race ./...                           # 竞态检测（merge 前必跑）
go test -run Integration ./...               # 仅集成测试
go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

# --- Docker Compose (local infra) ---
docker-compose up -d                          # 启动 PG + Redis（端口 3000）
docker-compose -f docker-compose.meilisearch.yml up -d

# --- K8s ---
ssh root@100.98.57.55 "kubectl get pods -n lurus-system"
ssh root@100.98.57.55 "kubectl rollout restart deployment/lurus-api -n lurus-system"
ssh root@100.98.57.55 "kubectl logs -n lurus-system -l app=lurus-api --tail=100"
```

## Environment Variables

### Required

| Variable | Description |
|----------|-------------|
| `SQL_DSN` | DB 连接串。PostgreSQL: `postgresql://user:pass@host/db`；MySQL: `user:pass@tcp(host:3306)/db` |
| `SESSION_SECRET` | Session 签名密钥（禁止使用 `random_string`，多节点必须一致） |
| `REDIS_CONN_STRING` | Redis 连接串，如 `redis://redis:6379`。缺失则用 cookie session |

### Auth (Zitadel OIDC)

| Variable | Default | Description |
|----------|---------|-------------|
| `ZITADEL_ENABLED` | `false` | 启用 Zitadel OIDC |
| `ZITADEL_ISSUER` | — | `https://auth.lurus.cn` |
| `ZITADEL_JWKS_URI` | — | `https://auth.lurus.cn/oauth/v2/keys` |
| `ZITADEL_CLIENT_ID` | — | Zitadel app client ID |
| `ZITADEL_REDIRECT_URI` | — | OAuth callback URL |
| `ZITADEL_ENABLE_PKCE` | `false` | 启用 PKCE |
| `ZITADEL_AUTO_CREATE_USER` | `false` | OIDC 登录自动建用户 |

### Identity Service

| Variable | Default | Description |
|----------|---------|-------------|
| `IDENTITY_SERVICE_URL` | `http://identity-service.lurus-identity.svc.cluster.local:18104` | lurus-identity 地址 |
| `IDENTITY_SERVICE_INTERNAL_KEY` | — | `/internal/v1/*` bearer token |

### Meilisearch (Optional)

| Variable | Default | Description |
|----------|---------|-------------|
| `MEILISEARCH_ENABLED` | `false` | 启用 Meilisearch |
| `MEILISEARCH_HOST` | — | `http://meilisearch:7700` |
| `MEILISEARCH_API_KEY` | — | Master key |
| `MEILISEARCH_SYNC_ENABLED` | `false` | 启用日志同步 |
| `MEILISEARCH_SYNC_WORKERS` | `32` | 同步并发数 |

### Runtime Tuning

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3000` | HTTP 监听端口 |
| `GIN_MODE` | `debug` | 设为 `release` 关闭 debug 输出 |
| `DEBUG` | `false` | 启用 debug 日志 |
| `NODE_TYPE` | `master` | `slave` 禁用 master-only 任务 |
| `SYNC_FREQUENCY` | `60` | 缓存同步周期（秒） |
| `RELAY_TIMEOUT` | `0` | Relay 超时（秒，0=不限） |
| `RELAY_MAX_IDLE_CONNS` | `500` | HTTP 连接池最大空闲连接 |
| `BATCH_UPDATE_ENABLED` | `false` | 启用批量数据库写入 |
| `BATCH_UPDATE_INTERVAL` | `5` | 批量写入间隔（秒） |
| `DAILY_QUOTA_ENABLED` | `true` | 设为 `false` 禁用每日配额重置 |
| `CHANNEL_UPDATE_FREQUENCY` | — | 渠道自动更新频率（分钟） |
| `MODEL_SYNC_FREQUENCY` | — | 模型自动同步频率（分钟） |
| `MEMORY_CACHE_ENABLED` | `false` | 启用内存缓存（Redis 存在时自动开启） |
| `ALLOWED_ORIGINS` | 见 config.go | 逗号分隔的 CORS 允许域名 |
| `STREAMING_TIMEOUT` | `300` | 流式请求无响应超时（秒） |
| `MAX_REQUEST_BODY_MB` | `64` | 请求体最大大小（MB） |
| `GRACEFUL_SHUTDOWN_TIMEOUT` | `30s` | 优雅停机等待时间 |

### Observability (Optional)

| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_TRACING_ENABLED` | `false` | 启用 OpenTelemetry traces |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | OTLP endpoint（如 `jaeger:4318`） |
| `OTEL_TRACE_SAMPLE_RATE` | `0.1` | 采样率（0.0~1.0） |
| `ENABLE_PPROF` | `false` | 启用 pprof（port 8005） |
| `LOG_FORMAT` | — | `json` 启用结构化日志 |
| `LOG_LEVEL` | — | 日志级别 |

### OAuth Providers (Optional)

`GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`, `ALIPAY_PRIVATE_KEY`, `ALIPAY_PUBLIC_KEY`, `WECHAT_SERVER_ADDRESS`, `WECHAT_SERVER_TOKEN`, `TELEGRAM_BOT_TOKEN`, `UMAMI_WEBSITE_ID`, `GOOGLE_ANALYTICS_ID`

## Key Runtime Notes

- **DB 自动迁移**: 启动时 GORM 自动建表；PostgreSQL 手动 SQL migration 在 `migrations/`
- **SQLite fallback**: 未设置 `SQL_DSN` 时使用 SQLite（`one-api.db`），仅用于开发
- **healthcheck**: `GET /api/status` → `{"success": true}`
- **internal API**: `POST /internal/v1/...` 需 `Authorization: Bearer $INTERNAL_API_KEY`（见 `adapter/middleware/internal_api_auth.go`）
- **渠道缓存**: Redis 存在时自动启用内存缓存，`SYNC_FREQUENCY` 控制同步周期
- **Proxy**: 生产环境通过 `HTTP_PROXY=http://10.42.1.1:10808` 访问外部 LLM API

## BMAD

| Resource | Path |
|----------|------|
| PRD | `./_bmad-output/planning-artifacts/prd.md` |
| Epics | `./_bmad-output/planning-artifacts/epics.md` |
| Architecture | `./_bmad-output/planning-artifacts/architecture.md` |
| Sprint Status | `./_bmad-output/planning-artifacts/sprint-status.yaml` |
| Project Context | `./_bmad-output/planning-artifacts/project-context.md` |

**Story 文档规则（Epic 6+ 严格执行）**: 实现前建 story 文档 → 通过 `dev-story/checklist.md` → 含验证证据才可标 done。违反 = 工作无效。
