<div align="center">

![lurus-hub](/web/public/logo.png)

# Lurus Hub

**AI Data Processing Hub & Multi-Tenant LLM Gateway**

**AI 数据处理枢纽 · 多租户大模型网关**

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25-blue?logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/License-MIT-brightgreen" alt="License">
  <img src="https://img.shields.io/badge/Meilisearch-v1.10+-orange?logo=meilisearch" alt="Meilisearch">
  <img src="https://img.shields.io/badge/Docker-Ready-blue?logo=docker" alt="Docker">
  <img src="https://img.shields.io/badge/K3s-Production-green?logo=kubernetes" alt="K3s">
</p>

</div>

---

## Overview / 项目简介

**Lurus Hub** is an AI data processing hub built on top of a multi-tenant LLM relay. Beyond unified API access to every major model provider, it adds real-time usage analytics, cost optimization, per-product routing, and platform-grade billing integration — turning a relay into a data plane.

不只是 LLM 中转站，更是数据处理枢纽 — 实时用量分析、成本优化、模型性能监控、按产品个性化路由。基于 [New API](https://github.com/QuantumNous/new-api) / [One API](https://github.com/songquanpeng/one-api) 开源基座深度定制，集成 Meilisearch 搜索、Zitadel OIDC 多租户认证、Prometheus/OpenTelemetry 可观测性，与 lurus-platform 通过 gRPC 完成计费打通。

---

## Core Features / 核心特性

### Multi-Tenant Architecture / 多租户架构
- **Zitadel OIDC** authentication with tenant isolation (shared DB + GORM plugin auto-inject `tenant_id`)
- **V2 API** (`/api/v2/:tenant_slug/*`) with full RBAC (admin, user, billing_manager)
- **V1 backward compatibility** — existing integrations continue working
- **Platform admin** API for cross-tenant management

### AI Model Gateway / AI 模型网关
- **Unified API** — one interface for OpenAI, Claude, Gemini, DeepSeek, Qwen, GLM, Moonshot, and more
- **Format auto-conversion** — OpenAI ↔ Claude ↔ Gemini transparent translation
- **Intelligent routing** — weighted load balancing, auto-retry on failure, priority-based channel selection
- **Specialized models** — embeddings, rerank, TTS, STT, image/video generation
- **Realtime API** — OpenAI Realtime (WebSocket) support

### Search & Performance / 搜索与性能
- **Meilisearch** integration — < 50ms search across logs, users, channels
- **Object pooling** — BufferPool, IntSlicePool, MapPool for hot path optimization
- **Gateway overhead** — p95 < 50ms verified by benchmark suite
- **HA deployment** — 2-replica rolling updates with PodDisruptionBudget

### Observability / 可观测性
- **Prometheus** `/metrics` endpoint with 11 metric types (request count, latency, token usage, quota)
- **OpenTelemetry** distributed tracing with Jaeger exporter and X-Trace-Id header
- **10 alerting rules** — error rate, latency, channel health, pod restarts
- **Structured logging** — JSON format with slog

### Billing & Security / 计费与安全
- Per-token, per-request, time-based billing with cache billing support
- Online top-up (Creem, Stripe integration)
- Fine-grained access control — model-level permissions, IP whitelist, token quotas
- Audit logging for all operations

---

## Tech Stack / 技术栈

| Layer | Technology |
|-------|-----------|
| **Backend** | Go 1.25, Gin, GORM, PostgreSQL/SQLite |
| **Frontend** | React 18, Vite, Semi UI, TailwindCSS |
| **Search** | Meilisearch v1.10+ |
| **Cache** | Redis |
| **Auth** | Zitadel OIDC + JWT |
| **Observability** | Prometheus, OpenTelemetry, Jaeger |
| **Deployment** | K3s, ArgoCD (GitOps), Docker |
| **CI/CD** | GitHub Actions → GHCR → ArgoCD sync |

---

## Architecture / 架构

```
                          ┌─────────────────────────┐
                          │      Lurus Hub Gateway   │
                          │    (Hexagonal / Go+Gin)  │
                          └────────────┬────────────┘
                                       │
                 ┌─────────────────────┼─────────────────────┐
                 │                     │                     │
          ┌──────▼──────┐     ┌───────▼───────┐    ┌───────▼───────┐
          │  V1 API     │     │  V2 API       │    │  Relay API    │
          │  (compat)   │     │  (multi-tenant)│    │  /v1/chat/*   │
          └─────────────┘     └───────────────┘    └───────────────┘
                                       │
          ┌────────────────────────────┼────────────────────────────┐
          │                            │                            │
   ┌──────▼──────┐           ┌────────▼────────┐          ┌───────▼───────┐
   │  Zitadel    │           │  PostgreSQL     │          │  Meilisearch  │
   │  OIDC Auth  │           │  (tenant_id)    │          │  Search       │
   └─────────────┘           └─────────────────┘          └───────────────┘

          ┌──────────────┐   ┌─────────────────┐   ┌──────────────┐
          │  Redis       │   │  Prometheus     │   │  Jaeger      │
          │  Cache       │   │  Metrics        │   │  Tracing     │
          └──────────────┘   └─────────────────┘   └──────────────┘
                                       │
                 ┌─────────────────────┼─────────────────────┐
                 │                     │                     │
          ┌──────▼──────┐     ┌───────▼───────┐    ┌───────▼───────┐
          │  OpenAI     │     │  Claude       │    │  Gemini      │
          │  Azure      │     │  DeepSeek     │    │  Qwen/GLM    │
          └─────────────┘     └───────────────┘    └───────────────┘
```

### Directory Structure / 目录结构

```
internal/
├── domain/entity/     # Domain models (no dependencies)
├── app/               # Business logic (use cases, relay handlers)
│   ├── relay/         # AI model relay orchestration
│   └── passkey/       # Passkey authentication
├── adapter/           # Infrastructure layer
│   ├── handler/       # HTTP handlers (Gin controllers)
│   │   └── router/    # Route definitions (v1, v2, v2-admin)
│   ├── middleware/     # Auth, CORS, tenant isolation, rate limiting
│   ├── repo/          # GORM repositories (data access)
│   └── provider/      # AI vendor adaptors (OpenAI, Claude, Gemini...)
├── lifecycle/         # App init, shutdown, background tasks
└── pkg/               # Shared utilities (config, logger, metrics, tracing, search)
```

---

## Quick Start / 快速开始

### Development / 开发环境

```bash
# Backend
go build -o lurus-api ./cmd/server
./lurus-api

# Frontend
cd web && bun install && bun run dev
```

### Testing / 测试

```bash
# Run all tests (recommended)
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detector
go test -race ./...

# Run only unit tests (skip integration tests)
go test -short ./...

# Run tests for specific package
go test -v ./internal/adapter/handler/
go test -v ./internal/app/

# Run specific test
go test -v ./internal/app/ -run TestCompareVersions

# Frontend tests
cd web && bun run test
cd web && bun run typecheck
cd web && bun run lint
```

**Important**: Always run tests by package or using `go test ./...`. Do NOT run individual test files directly (e.g., `go test ./path/to/file_test.go`) as it will fail with missing dependencies.

### Docker Compose

```bash
docker-compose up -d
# Access: http://localhost:3000
```

### Production (K3s + ArgoCD)

See [Deployment Runbook](./doc/runbook/deployment.md) for full instructions.

```bash
# Build
CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -trimpath -o app ./cmd/server

# Deploy via ArgoCD — push to main, ArgoCD syncs automatically
# Manual rollout:
ssh root@100.98.57.55 "kubectl rollout restart deployment/lurus-api -n lurus-system"
```

---

## API Endpoints / API 端点

### V2 Multi-Tenant API (Zitadel JWT)

| Endpoint | Description |
|----------|-------------|
| `GET /api/v2/:slug/auth/login` | OAuth login via Zitadel |
| `GET /api/v2/:slug/user/self` | Current user info |
| `CRUD /api/v2/:slug/token/` | Token management |
| `CRUD /api/v2/:slug/channel/` | Channel management |
| `GET /api/v2/:slug/log/` | Request logs |
| `POST /api/v2/admin/tenants` | Platform admin |

### Relay API (OpenAI-compatible)

| Endpoint | Description |
|----------|-------------|
| `POST /v1/chat/completions` | Chat completions (all providers) |
| `POST /v1/messages` | Claude Messages API |
| `POST /v1/embeddings` | Text embeddings |
| `POST /v1/images/generations` | Image generation |

### V1 Legacy API (backward compatible)

| Endpoint | Description |
|----------|-------------|
| `POST /api/user/login` | User login |
| `GET /api/user/self` | Current user |
| `CRUD /api/token/` | Token management |
| `GET /api/log/search` | Log search (Meilisearch) |

Full API documentation: [https://docs.lurus.cn/](https://docs.lurus.cn/) | [OpenAPI Spec](./docs/openapi/api-v2.yaml)

---

## Documentation / 文档

| Document | Description |
|----------|-------------|
| [Deployment Runbook](./doc/runbook/deployment.md) | Build, deploy, verify, rollback |
| [Database Runbook](./doc/runbook/database.md) | Backup, restore, migration |
| [Tenant Onboarding](./doc/runbook/tenant-onboarding.md) | New tenant setup |
| [Incident Response](./doc/runbook/incident-response.md) | Triage, escalation, postmortem |
| [HA Deployment](./doc/runbook/ha-deployment.md) | High availability guide |
| [Staging Environment](./doc/runbook/staging-environment.md) | Pre-production setup |
| [Zitadel Setup](./doc/zitadel-setup-guide.md) | OIDC auth configuration |
| [OpenAPI Spec](./docs/openapi/api-v2.yaml) | 45 endpoints, 30+ schemas |
| [Development Log](./doc/process.md) | Change history |

---

## Environment Variables / 环境变量

| Variable | Required | Description |
|----------|----------|-------------|
| `SQL_DSN` | Yes | PostgreSQL connection string |
| `SESSION_SECRET` | Yes | Session encryption key |
| `REDIS_CONN_STRING` | No | Redis connection (recommended) |
| `MEILISEARCH_ENABLED` | No | Enable search engine |
| `MEILISEARCH_HOST` | No | Meilisearch URL |
| `MEILISEARCH_API_KEY` | No | Meilisearch key |
| `ZITADEL_ISSUER` | No | Zitadel OIDC issuer URL |
| `ZITADEL_CLIENT_ID` | No | OIDC client ID |
| `OTEL_TRACING_ENABLED` | No | Enable OpenTelemetry tracing |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | Jaeger/OTLP endpoint |

See [.env.meilisearch.example](./.env.meilisearch.example) and [.env.zitadel.example](./.env.zitadel.example) for full configuration.

---

## License

MIT License. See [LICENSE](./LICENSE).

Based on [One API](https://github.com/songquanpeng/one-api) (MIT License).
