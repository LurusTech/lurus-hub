---
stepsCompleted: [1, 2, 3, 4]
inputDocuments: ['prd-api.md', 'architecture.md', 'bmad-gap-analysis.md', 'lurus-api/doc/process.md']
date: '2026-02-02'
author: 'Anita (via BMAD Sprint Planning)'
service: 'lurus-api'
---

# Lurus API - Epic Breakdown
# Lurus API - 史诗分解

## Overview

This document provides the complete epic and story breakdown for lurus-api, decomposing the requirements from the PRD, architecture, and gap analysis into implementable stories. Epics are organized by **user value**, not technical layers.

本文档提供 lurus-api 的完整 Epic 和 Story 分解，将 PRD、架构和差距分析中的需求分解为可实现的用户故事。Epic 按**用户价值**组织，而非技术层。

---

## Requirements Inventory

### Functional Requirements

| ID | Requirement | Source |
|----|-------------|--------|
| FR-1 | LLM API Relay (10 sub-items) | prd-api.md §5 |
| FR-2 | Multi-Tenant Isolation (6 sub-items) | prd-api.md §5 |
| FR-3 | Authentication & Authorization (6 sub-items) | prd-api.md §5 |
| FR-4 | Billing (6 sub-items) | prd-api.md §5 |

### Non-Functional Requirements

| ID | Requirement | Source |
|----|-------------|--------|
| NFR-1 | Performance (p95 < 50ms overhead) | prd-api.md §6 |
| NFR-2 | Reliability (≥ 99.5% uptime) | prd-api.md §6 |
| NFR-3 | Security (7 sub-items) | prd-api.md §6 |
| NFR-4 | Scalability (4 sub-items) | prd-api.md §6 |
| NFR-5 | Observability (Prometheus + tracing) | prd-api.md §6 |
| NFR-6 | Testing (app/ ≥80%, adapter/handler/ ≥50%) | prd-api.md §6 |

### Additional Requirements (from Gap Analysis)

| ID | Requirement | Source |
|----|-------------|--------|
| GAP-R2 | Staging environment | bmad-gap-analysis.md |
| GAP-R4 | Planning documents populated | bmad-gap-analysis.md |
| GAP-R8 | API documentation (OpenAPI) | bmad-gap-analysis.md |

### FR Coverage Map

| FR | Epic 1 | Epic 2 | Epic 3 | Epic 4 | Epic 5 |
|----|--------|--------|--------|--------|--------|
| FR-1 (Relay) | | | 3.2 | | |
| FR-2 (Multi-Tenant) | 1.1-1.4 | | | | |
| FR-3 (Auth) | 1.2-1.3 | | | | |
| FR-4 (Billing) | 1.3 | | | | |
| NFR-1 (Performance) | | | 3.1-3.2 | | |
| NFR-2 (Reliability) | | | 3.3 | | |
| NFR-3 (Security) | 1.3 | 2.4 | | | |
| NFR-5 (Observability) | | | | 4.1-4.3 | |
| NFR-6 (Testing) | 1.4 | 2.1-2.4 | | | |
| GAP-R2 (Staging) | | | | | 5.2 |
| GAP-R8 (API docs) | | | | | 5.1 |

---

## Epic List

| # | Epic | Goal | Priority | Dependencies |
|---|------|------|----------|-------------|
| 1 | Multi-Tenant Production Launch | Deploy multi-tenant SaaS to production with real Zitadel integration | P0 | Zitadel instance |
| 2 | Test Coverage & Quality Gate | Close test coverage gaps to meet PRD targets (biz ≥80%) | P1 | None |
| 3 | Gateway Performance & Reliability | Reduce gateway overhead and improve uptime toward 99.5% | P1 | Epic 1 |
| 4 | Observability Stack | Add Prometheus metrics and structured tracing | P2 | Epic 3 |
| 5 | Developer Experience & Documentation | API docs, staging environment, operational runbooks | P2 | None |

---

## Epic 1: Multi-Tenant Production Launch / 多租户生产上线

**Goal**: Deploy the code-complete multi-tenant system to production, enabling real tenants to use the v2 API with Zitadel authentication and full data isolation.

**Rationale**: All 6 phases of multi-tenant code are complete (26 v2 endpoints, tenant isolation plugin, OAuth flow, billing isolation). The blocker is Zitadel configuration and production deployment. This is the highest-value work because it unlocks SC-4 (multi-tenant production) and SC-8 (5+ tenants).

### Story 1.1: Configure Zitadel Instance / 配置 Zitadel 实例

As a **platform admin**,
I want **Zitadel configured with Organization, Project, and OIDC Application**,
So that **the v2 API can authenticate users via Zitadel OIDC**.

**Acceptance Criteria:**

**Given** Zitadel is running at auth.lurus.cn
**When** admin creates Organization "Lurus Platform" with Project "lurus-api"
**Then** an OIDC Application is created with correct redirect URIs
**And** Client ID and Client Secret are stored in K8s secrets

**Given** Zitadel OIDC Application is configured
**When** admin configures Project Roles (admin, user, billing_manager)
**Then** roles are available for JWT claims

**Given** Zitadel is configured
**When** admin configures SMTP using Stalwart Mail
**Then** Zitadel can send verification and password reset emails

**Tasks:**
- [ ] Login to Zitadel console (https://auth.lurus.cn)
- [ ] Create Organization "Lurus Platform", record org_id
- [ ] Create Project "lurus-api"
- [ ] Create OIDC Application "lurus-api-backend" (JWT auth, Authorization Code + Refresh Token)
- [ ] Configure redirect URIs (https://api.lurus.cn/api/v2/oauth/callback, localhost dev)
- [ ] Configure Project Roles (admin, user, billing_manager)
- [ ] Configure SMTP (mail.lurus.cn:587, noreply@lurus.cn)
- [ ] Update K8s secrets with Client ID, Client Secret, Org ID
- [ ] Verify OIDC Discovery endpoint returns correct metadata

**Note**: This story requires **manual human action** on the Zitadel console.

---

### Story 1.2: Run Production Database Migrations / 执行生产数据库迁移

As a **platform admin**,
I want **the production PostgreSQL schema updated with tenant tables and tenant_id columns**,
So that **the multi-tenant data model is active in production**.

**Acceptance Criteria:**

**Given** the 4 migration SQL files exist
**When** migrations are executed on production PostgreSQL
**Then** `tenants`, `user_identity_mapping`, `tenant_configs` tables are created
**And** all existing tables have `tenant_id` column with default 'default'
**And** composite indexes on `(tenant_id, ...)` are created
**And** existing data is assigned to the 'default' tenant

**Given** migrations are applied
**When** the application starts
**Then** GORM AutoMigrate succeeds without errors
**And** all existing v1 API queries continue to work

**Tasks:**
- [ ] Backup production database before migration
- [ ] Execute `migrations/001_create_tenants.sql`
- [ ] Execute `migrations/002_create_user_mapping.sql`
- [ ] Execute `migrations/003_create_tenant_configs.sql`
- [ ] Execute `migrations/004_add_tenant_id.sql`
- [ ] Verify data integrity (all existing records have tenant_id='default')
- [ ] Verify v1 API still works after migration
- [ ] Document rollback procedure

---

### Story 1.3: Deploy v2 API to K3s / 部署 v2 API 到 K3s

As a **platform admin**,
I want **the v2 API routes deployed and accessible in production**,
So that **tenant users can authenticate via Zitadel and use the multi-tenant API**.

**Acceptance Criteria:**

**Given** Zitadel is configured and DB is migrated
**When** the updated lurus-api image is deployed to K3s
**Then** v2 OAuth routes respond (GET /api/v2/:tenant_slug/auth/login)
**And** v2 tenant routes require valid Zitadel JWT
**And** v2 platform admin routes require root role
**And** v1 API routes continue to work unchanged

**Given** v2 API is deployed
**When** a user navigates to the Zitadel login flow
**Then** the OAuth redirect → callback → session creation flow completes successfully
**And** the user is mapped to a lurus user with correct tenant_id

**Tasks:**
- [ ] Update K8s deployment.yaml with Zitadel env vars (OIDC_ENABLED, ZITADEL_*)
- [ ] Update K8s secrets.yaml with Zitadel credentials
- [ ] Build and push new Docker image
- [ ] Deploy via ArgoCD
- [ ] Verify v1 API health check passes
- [ ] Verify v2 OAuth login redirect works
- [ ] Verify v2 JWT-protected routes reject unauthorized requests
- [ ] Test creating a second tenant

---

### Story 1.4: End-to-End Multi-Tenant Testing / 端到端多租户测试

As a **platform admin**,
I want **verified end-to-end multi-tenant isolation in production**,
So that **I can confidently onboard new tenants with guaranteed data isolation**.

**Acceptance Criteria:**

**Given** v2 API is deployed with Zitadel
**When** two separate tenants are created with different users
**Then** Tenant A users cannot see Tenant B's tokens, channels, logs, or billing data
**And** Webhook payments correctly verify tenant ownership
**And** Platform admin can see all tenants' data

**Given** multi-tenant is working
**When** a user makes an LLM API request via a v2 tenant token
**Then** the request is correctly logged with the tenant_id
**And** quota is deducted from the correct tenant user

**Tasks:**
- [ ] Create test Tenant A and Tenant B in Zitadel
- [ ] Login as Tenant A user, create tokens, make API calls
- [ ] Login as Tenant B user, verify no Tenant A data visible
- [ ] Test payment webhook tenant verification (Stripe/Epay/Creem)
- [ ] Test platform admin cross-tenant queries
- [ ] Test v1 API backward compatibility (default tenant)
- [ ] Document tenant onboarding procedure
- [ ] Write E2E test script for regression

---

## Epic 2: Test Coverage & Quality Gate / 测试覆盖率与质量关卡

**Goal**: Close the test coverage gap from current ~70% (app/) to the PRD target of ≥80%, and increase adapter/handler/ coverage to ≥50%.

**Rationale**: The gap analysis identified insufficient test coverage as a P0 risk. The BMAD improvement sprint added 97 tests, but the app/ layer needs more coverage especially for quota management, channel selection, and notification services. Meeting NFR-6 targets is a prerequisite for production confidence.

### Story 2.1: Service Layer Test Coverage / 服务层测试覆盖

As a **developer**,
I want **comprehensive tests for all service layer functions**,
So that **business logic changes can be made with confidence (app/ ≥80%)**.

**Acceptance Criteria:**

**Given** existing tests cover token_service, user_service, billing_service
**When** new tests are added for quota, channel_select, notify, pre_consume services
**Then** `go test -cover ./internal/app/...` reports ≥80% coverage

**Tasks:**
- [ ] Add tests for `quota.go` (PreConsumeTokenQuota, PostConsumeQuota, PostClaudeConsumeQuota)
- [ ] Add tests for `channel_select.go` (CacheGetRandomSatisfiedChannel, cross-group retry)
- [ ] Add tests for `notify-limit.go` (CheckNotificationLimit, Redis + in-memory backends)
- [ ] Add tests for `pre_consume_quota.go` (PreConsumeQuota, ReturnPreConsumedQuota)
- [ ] Add tests for `channel.go` (ShouldDisableChannel, EnableChannel, DisableChannel)
- [ ] Add tests for `group.go` (GetUserUsableGroups, GetUserGroupRatio)
- [ ] Run coverage report and verify ≥80%

---

### Story 2.2: Relay Adaptor Test Coverage / 中继适配器测试覆盖

As a **developer**,
I want **tests for the BaseAdaptor and key provider adaptors**,
So that **adaptor changes don't break request/response transformation**.

**Acceptance Criteria:**

**Given** BaseAdaptor provides default implementations for 14 methods
**When** tests exercise each default method
**Then** all default behaviors are verified
**And** at least 3 provider adaptors (OpenAI, Claude, Gemini) have request conversion tests

**Tasks:**
- [ ] Add tests for `base_adaptor.go` (all 14 default methods)
- [ ] Add tests for OpenAI adaptor (ConvertOpenAIRequest, DoResponse)
- [ ] Add tests for Claude adaptor (ConvertClaudeRequest, streaming)
- [ ] Add tests for Gemini adaptor (ConvertGeminiRequest)
- [ ] Add tests for `stream_scanner.go` (SSE parsing edge cases)

---

### Story 2.3: Controller Layer Test Coverage / 控制器层测试覆盖

As a **developer**,
I want **controller tests covering input validation and error paths**,
So that **API endpoints handle edge cases correctly (adapter/handler/ ≥50%)**.

**Acceptance Criteria:**

**Given** v2 controllers exist for User, Token, Log, Channel, Billing, Redemption, Admin
**When** tests exercise input validation, auth checks, and error responses
**Then** `go test -cover ./internal/adapter/handler/...` reports ≥50% coverage

**Tasks:**
- [ ] Add tests for v2 user controller (GetSelfV2, UpdateSelfV2 validation)
- [ ] Add tests for v2 token controller (quota limits, expiration checks)
- [ ] Add tests for v2 billing controller (payment method validation, subscription lifecycle)
- [ ] Add tests for v2 redemption controller (tenant verification, batch create)
- [ ] Add tests for v2 admin controller (role checks, system stats)

---

### Story 2.4: Security Regression Test Suite / 安全回归测试套件

As a **platform admin**,
I want **automated security tests that run in CI**,
So that **security regressions are caught before deployment**.

**Acceptance Criteria:**

**Given** security tests exist for tenant isolation (25 tests)
**When** new tests are added for auth, rate limiting, and input sanitization
**Then** a security-focused test suite can be run independently
**And** it covers: tenant isolation, JWKS validation, rate limiting, XSS/injection prevention

**Tasks:**
- [ ] Consolidate existing 25 tenant isolation tests into `security_test.go`
- [ ] Add JWKS key rotation tests (key miss → refresh → retry)
- [ ] Add rate limiting tests (global, model-level, critical endpoint)
- [ ] Add input sanitization tests (SQL injection, XSS in user input)
- [ ] Add auth bypass tests (expired JWT, invalid signature, missing claims)
- [ ] Create CI workflow step for security tests

---

## Epic 3: Gateway Performance & Reliability / 网关性能与可靠性

**Goal**: Reduce gateway processing overhead from ~80ms (p95) to <50ms and improve monthly uptime from ~98% to ≥99.5%.

**Rationale**: SC-1 (uptime) and SC-2 (latency) are the North Star and key success metrics. Gateway overhead directly impacts developer experience. Reliability improvements prevent revenue loss from downtime.

### Story 3.1: Performance Benchmark & Profiling / 性能基准与分析

As a **platform operator**,
I want **a performance benchmark measuring gateway overhead**,
So that **I can identify bottlenecks and track improvement over time**.

**Acceptance Criteria:**

**Given** the pprof and CPU monitoring infrastructure exists
**When** a benchmark suite is run against the relay endpoint
**Then** p50, p95, p99 latency measurements are captured for gateway processing
**And** bottleneck functions are identified via pprof profiles
**And** results are documented with baseline numbers

**Tasks:**
- [ ] Create benchmark test for chat completions relay path (mock upstream)
- [ ] Create benchmark test for streaming relay path
- [ ] Profile with pprof to identify hot functions
- [ ] Document baseline: p50, p95, p99 for gateway overhead
- [ ] Identify top 3 optimization targets

---

### Story 3.2: Optimize Gateway Hot Paths / 优化网关热路径

As an **API consumer**,
I want **lower latency on API requests**,
So that **my application response times improve**.

**Acceptance Criteria:**

**Given** benchmark baseline is established
**When** optimizations are applied to identified hot paths
**Then** p95 gateway overhead is <50ms (measured by benchmark)
**And** no functionality is broken (all tests pass)

**Tasks:**
- [ ] Optimize channel cache lookup (avoid DB queries on hot path)
- [ ] Optimize token validation (cache recently validated tokens)
- [ ] Optimize model ratio lookup (precompute and cache)
- [ ] Review and optimize stream_scanner buffer allocation
- [ ] Re-run benchmarks and compare against baseline

---

### Story 3.3: HA Deployment Preparation / 高可用部署准备

As a **platform operator**,
I want **the application ready for multi-replica deployment**,
So that **uptime improves toward 99.5% with zero-downtime deployments**.

**Acceptance Criteria:**

**Given** lurus-api is stateless (shared PostgreSQL + Redis)
**When** replicas are increased to 2 in K8s deployment
**Then** both replicas serve requests correctly
**And** rolling updates cause zero downtime
**And** session stickiness is handled by Redis-backed sessions

**Tasks:**
- [ ] Verify all state is in PostgreSQL/Redis (no in-memory-only state)
- [ ] Test with replicas: 2 in staging
- [ ] Verify rolling update with zero downtime
- [ ] Update deployment.yaml with PodDisruptionBudget
- [ ] Document HA deployment procedure (reference doc/decisions/ha-deployment.md)

---

## Epic 4: Observability Stack / 可观测性

**Goal**: Add Prometheus metrics endpoint and structured request tracing to enable proactive monitoring and faster incident response.

**Rationale**: NFR-5.4 (Prometheus) and NFR-5.5 (tracing) are planned but not implemented. Observability is critical for meeting the 99.5% uptime target and for identifying performance regressions.

### Story 4.1: Prometheus Metrics Endpoint / Prometheus 指标端点

As a **platform operator**,
I want **a /metrics endpoint exposing key gateway metrics**,
So that **Grafana dashboards can visualize system health in real-time**.

**Acceptance Criteria:**

**Given** the application is running
**When** Prometheus scrapes the /metrics endpoint
**Then** metrics include: request_count, request_duration_seconds, active_connections, quota_consumed, channel_errors
**And** metrics are labeled by: model, channel_type, tenant_id, status_code

**Tasks:**
- [ ] Add `prometheus/client_golang` dependency
- [ ] Create metrics middleware for Gin (request count, duration histogram)
- [ ] Add custom metrics: channel_error_total, quota_consumed_total, active_relay_connections
- [ ] Expose /metrics endpoint (internal only, not via Ingress)
- [ ] Create Grafana dashboard for lurus-api
- [ ] Add ServiceMonitor CR for Prometheus operator

---

### Story 4.2: Request Tracing / 请求追踪

As a **developer**,
I want **end-to-end request tracing from client to upstream LLM provider**,
So that **I can diagnose slow requests and identify which component adds latency**.

**Acceptance Criteria:**

**Given** a request arrives at the gateway
**When** it is processed through middleware → controller → relay → upstream
**Then** a trace is created with spans for each stage
**And** trace_id is propagated in logs and response headers

**Tasks:**
- [ ] Add OpenTelemetry SDK dependency
- [ ] Create tracing middleware (generate trace_id, create root span)
- [ ] Add spans for: auth, channel_select, adaptor_convert, upstream_request, response_process
- [ ] Propagate trace_id in slog context
- [ ] Add X-Trace-Id response header
- [ ] Configure Jaeger exporter (jaeger.lurus.cn)

---

### Story 4.3: Alerting Rules / 告警规则

As a **platform operator**,
I want **automated alerts for critical system conditions**,
So that **issues are detected before they impact users**.

**Acceptance Criteria:**

**Given** Prometheus metrics are being collected
**When** error rate exceeds 5% or p95 latency exceeds 200ms
**Then** an alert is fired to the configured notification channel

**Tasks:**
- [ ] Define PrometheusRule CRs for critical alerts
- [ ] Alert: error_rate > 5% for 5 minutes
- [ ] Alert: p95_latency > 200ms for 10 minutes
- [ ] Alert: channel_consecutive_errors > 10
- [ ] Alert: pod_restart_count > 3 in 1 hour
- [ ] Configure alert routing (webhook → Bark/Gotify)

---

## Epic 5: Developer Experience & Documentation / 开发者体验与文档

**Goal**: Improve developer onboarding, API discoverability, and operational confidence with documentation, staging environment, and operational runbooks.

**Rationale**: Gap analysis identified missing API documentation (R8), no staging environment (R2), and empty planning documents (R4) as significant gaps. These affect both external API consumers and internal development velocity.

### Story 5.1: OpenAPI Specification & Documentation / OpenAPI 规范与文档

As an **API consumer**,
I want **up-to-date, accurate API documentation**,
So that **I can integrate with the API without guessing endpoint behavior**.

**Acceptance Criteria:**

**Given** the OpenAPI spec at docs/openapi/api.json exists but is incomplete
**When** the spec is updated to cover all v1 and v2 endpoints
**Then** the spec validates with OpenAPI 3.0 tooling
**And** it is accessible at docs.lurus.cn
**And** v2 multi-tenant endpoints are documented with auth requirements

**Tasks:**
- [ ] Audit current api.json against actual route definitions
- [ ] Add missing v2 endpoints to OpenAPI spec
- [ ] Add request/response schemas for all endpoints
- [ ] Add authentication descriptions (Bearer token, JWT, session)
- [ ] Validate with openapi-generator or swagger-cli
- [ ] Deploy updated docs to docs.lurus.cn

---

### Story 5.2: Staging Environment / 预发布环境

As a **developer**,
I want **a staging namespace in K3s for pre-production testing**,
So that **changes can be verified before affecting production**.

**Acceptance Criteria:**

**Given** K3s cluster has capacity
**When** a staging namespace is created with lurus-api deployment
**Then** staging uses a separate database schema (lurus_api_staging)
**And** staging is accessible at staging-api.lurus.cn
**And** CI pipeline can deploy to staging on PR merge

**Tasks:**
- [ ] Create `lurus-staging` namespace in K3s
- [ ] Create staging database schema
- [ ] Create staging deployment.yaml (reduced resources)
- [ ] Configure staging IngressRoute (staging-api.lurus.cn)
- [ ] Add GitHub Actions step to deploy to staging on main push
- [ ] Document staging environment usage

---

### Story 5.3: Operational Runbook / 运维手册

As a **platform operator**,
I want **documented procedures for common operational tasks**,
So that **incidents can be resolved quickly without tribal knowledge**.

**Acceptance Criteria:**

**Given** common operational scenarios occur
**When** the operator consults the runbook
**Then** step-by-step procedures exist for: deployment, rollback, database backup/restore, tenant onboarding, incident response

**Tasks:**
- [ ] Create `doc/runbook/deployment.md` (build, deploy, verify, rollback)
- [ ] Create `doc/runbook/database.md` (backup, restore, migration)
- [ ] Create `doc/runbook/tenant-onboarding.md` (Zitadel + lurus steps)
- [ ] Create `doc/runbook/incident-response.md` (triage, escalation, postmortem)
- [ ] Link runbooks from main README.md
