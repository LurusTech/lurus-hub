# Development Progress / 开发进度

> Last Updated: 2026-02-13
> Archive: doc/archive/process_v20260205.md (entries before 2026-02-04)
> **New Rule**: 每条目 ≤ 15 行（HARD LIMIT），只记录已完成工作的极简摘要

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
