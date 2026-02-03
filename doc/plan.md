# Lurus API Sprint Backlog / 冲刺待办
# Lurus API Sprint Backlog

> Last Updated: 2026-02-02
> Sprint Cadence: 2-week iterations
> Epic Source: `_bmad-output/planning-artifacts/epics-api.md`
> Sprint Tracking: `_bmad-output/planning-artifacts/sprint-status.yaml`

---

## Current State / 当前状态

### Completed (2026-01 ~ 2026-02-02)

| Item | Status | Details |
|------|--------|---------|
| Multi-tenant SaaS code (Phase 1-6) | ✅ Code complete | 26 v2 endpoints, tenant isolation, OAuth, billing |
| BMAD Improvements P0-P3 | ✅ Complete | 14 items: security, service layer, BaseAdaptor, config, slog |
| Graceful shutdown | ✅ Complete | errgroup, signal handling, 30s timeout |
| go-redis v8→v9 | ✅ Complete | Drop-in upgrade |
| SafeGo utilities | ✅ Complete | Panic recovery for goroutines |
| BMAD artifacts | ✅ Complete | PRD, architecture, product brief, gap analysis |

### Blockers / 阻塞项

| Blocker | Owner | Impact |
|---------|-------|--------|
| Zitadel console configuration | Anita (manual) | Blocks Epic 1 (multi-tenant production) |

---

## Sprint 1: Foundation (2026-02-03 ~ 2026-02-16)

**Sprint Goal**: Close test coverage gaps and prepare for multi-tenant production deployment.

**Scope**: Epic 1 (Story 1.1) + Epic 2 (Stories 2.1, 2.2) + Epic 5 (Story 5.3)

| Story | Epic | Priority | Status | Notes |
|-------|------|----------|--------|-------|
| 1.1 Configure Zitadel Instance | E1 | P0 | Backlog | Manual - Anita |
| 2.1 Service Layer Test Coverage | E2 | P1 | ✅ Done | 187 subtests, 7 files |
| 2.2 Relay Adaptor Test Coverage | E2 | P1 | ✅ Done | ~100+ subtests, 5 files |
| 5.3 Operational Runbook | E5 | P2 | Backlog | deployment, database, tenant onboarding |

**Acceptance**: biz/ test coverage ≥80%, Zitadel configured (if unblocked)

---

## Sprint 2: Production Launch (2026-02-17 ~ 2026-03-02)

**Sprint Goal**: Deploy multi-tenant to production and verify end-to-end.

**Scope**: Epic 1 (Stories 1.2-1.4) + Epic 2 (Story 2.3)

| Story | Epic | Priority | Status | Notes |
|-------|------|----------|--------|-------|
| 1.2 Production Database Migrations | E1 | P0 | Backlog | Requires DB backup first |
| 1.3 Deploy v2 API to K3s | E1 | P0 | Backlog | Requires 1.1 + 1.2 |
| 1.4 E2E Multi-Tenant Testing | E1 | P0 | Backlog | Requires 1.3 |
| 2.3 Controller Layer Test Coverage | E2 | P1 | ✅ Done | ~50 subtests, 10 files |

**Acceptance**: Multi-tenant working in production, 2 tenants created, v1 backward compatible

---

## Sprint 3: Performance & Observability (2026-03-03 ~ 2026-03-16)

**Sprint Goal**: Establish performance baselines, add metrics, and prepare HA deployment.

**Scope**: Epic 3 (Stories 3.1-3.2) + Epic 4 (Story 4.1) + Epic 2 (Story 2.4)

| Story | Epic | Priority | Status | Notes |
|-------|------|----------|--------|-------|
| 3.1 Performance Benchmark | E3 | P1 | Backlog | Establish p95 baseline |
| 3.2 Optimize Hot Paths | E3 | P1 | Backlog | Target: p95 < 50ms |
| 4.1 Prometheus Metrics Endpoint | E4 | P2 | Backlog | /metrics + Grafana dashboard |
| 2.4 Security Regression Tests | E2 | P1 | Backlog | CI security gate |

**Acceptance**: p95 measured and documented, /metrics endpoint live, security tests in CI

---

## Sprint 4: Reliability & DX (2026-03-17 ~ 2026-03-30)

**Sprint Goal**: Enable HA deployment, add tracing, and improve API documentation.

**Scope**: Epic 3 (Story 3.3) + Epic 4 (Stories 4.2-4.3) + Epic 5 (Stories 5.1-5.2)

| Story | Epic | Priority | Status | Notes |
|-------|------|----------|--------|-------|
| 3.3 HA Deployment Preparation | E3 | P1 | Backlog | 2 replicas, PDB |
| 4.2 Request Tracing | E4 | P2 | Backlog | OpenTelemetry + Jaeger |
| 4.3 Alerting Rules | E4 | P2 | Backlog | PrometheusRule CRs |
| 5.1 OpenAPI Specification | E5 | P2 | Backlog | v1 + v2 endpoints |
| 5.2 Staging Environment | E5 | P2 | Backlog | lurus-staging namespace |

**Acceptance**: HA tested, tracing live, staging accessible, API docs published

---

## Success Criteria Tracking / 成功标准跟踪

| ID | Metric | Current | Target | Sprint |
|----|--------|---------|--------|--------|
| SC-1 | Monthly uptime | ~98% | ≥99.5% | S3-S4 |
| SC-2 | Gateway overhead p95 | ~80ms | <50ms | S3 |
| SC-4 | Multi-tenant support | Code complete | Production | S2 |
| SC-6 | Test coverage (biz/) | ~70% | ≥80% | S1 |
| SC-7 | Test coverage (controller/) | ~50% | ≥50% | S2 |
| SC-8 | Active tenants | 1 (default) | 5+ | S2+ |

---

## Previous Plan (Archived)

The multi-tenant SaaS transformation plan (6 phases) that previously occupied this file has been completed and is documented in `doc/process.md` (entries from 2026-01-25 to 2026-02-02). Key decisions are recorded in `doc/decisions/`.
