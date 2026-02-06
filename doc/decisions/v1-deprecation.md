# ADR: V1 API Deprecation Plan

**Status**: Proposed
**Date**: 2026-02-03
**Relates to**: ADR-API-007 (architecture.md)

## Context

Lurus API has dual API surfaces: v1 (session-based auth, single-tenant) and v2 (Zitadel JWT, multi-tenant). Maintaining both increases security audit surface and code complexity. However, v1 still serves the embedded React SPA and existing integrations.

The OpenAI-compatible relay routes (`/v1/chat/completions`, etc.) are **out of scope** for deprecation; they follow an industry-standard path and use independent Token-based auth.

## Decision

4-phase deprecation after v2 reaches production parity:

| Phase | Action | Trigger |
|-------|--------|---------|
| Announce | Inject `Deprecation: true` + `Sunset` HTTP headers on v1 responses | v2 production verified |
| Migration | Frontend switches to v2 endpoints; publish migration guide | T+2~8w after announce |
| Monitor | v1 usage tracking middleware; weekly reports on migration progress | T+6~14w |
| Sunset | v1 returns 410 Gone, then remove code | T+16w+ |

### Prerequisites (all must be met before Phase 1)

- v2 API feature-complete (26 endpoints covering all v1 functionality)
- Zitadel OIDC production-verified with real tenants
- Database migration stable (tenant_id backfill complete)
- E2E test coverage for v2 endpoints

## Consequences

- (+) Eliminates dual auth path, reducing security audit surface
- (+) Simplifies middleware stack (remove session store, CSRF)
- (+) Enables pure stateless JWT architecture
- (-) Requires frontend migration work
- (-) External integrations using v1 session auth need migration notice

## Current Status

v2 is production-deployed. v1 remains fully functional. Phase 1 (Announce) has not started pending broader tenant adoption validation.
