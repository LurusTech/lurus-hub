# ADR: High Availability Deployment

**Status**: Accepted
**Date**: 2026-02-03
**Relates to**: ADR-API-006 (architecture.md)

## Context

Lurus API runs on a single K3s node (100.98.57.55). Production requires zero-downtime deployments and resilience to single-pod failures. The team is 2 people, so operational complexity must stay minimal.

## Decision

Use Kubernetes-native HA with 2 replicas, RollingUpdate strategy (maxUnavailable=0, maxSurge=1), PodDisruptionBudget (minAvailable=1), and pod anti-affinity (soft, by hostname).

All application state is externalized:
- **Sessions & rate limiting**: Redis (shared across replicas)
- **Data**: PostgreSQL (CNPG)
- **Search index**: Meilisearch (optional, graceful fallback)
- **Auth keys**: Zitadel JWKS (each replica refreshes independently)

No sticky sessions, no local file state, no in-process cache that requires sync.

## Consequences

- (+) Zero-downtime rolling deploys with standard `kubectl rollout`
- (+) No additional infrastructure (no service mesh, no custom LB)
- (+) Single-node K3s still benefits from multi-pod resilience
- (-) Channel cache has ~60s lag between replicas (acceptable for load balancing)
- (-) Multi-node HA requires Redis Sentinel / CNPG failover (future work)

## References

- Operational runbook: `doc/runbook/ha-deployment.md`
- K8s manifests: `deploy/k8s/`
