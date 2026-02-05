# High Availability Deployment Guide

## Overview

This document describes the high availability (HA) deployment configuration for lurus-api, enabling zero-downtime deployments and improved reliability.

## Architecture

```
                    ┌─────────────────┐
                    │   Traefik LB    │
                    │  (IngressRoute) │
                    └────────┬────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
        ┌─────▼─────┐                ┌─────▼─────┐
        │  Pod A    │                │  Pod B    │
        │ lurus-api │                │ lurus-api │
        └─────┬─────┘                └─────┬─────┘
              │                             │
              └──────────────┬──────────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
   ┌────▼────┐         ┌────▼────┐         ┌────▼────┐
   │PostgreSQL│         │  Redis  │         │MeiliSearch│
   └──────────┘         └─────────┘         └──────────┘
```

## Prerequisites

- Redis must be enabled (`REDIS_CONN_STRING` configured)
- PostgreSQL as primary database
- Shared session secret across replicas (`SESSION_SECRET` env var)

## Configuration

### Deployment Configuration

```yaml
spec:
  replicas: 2  # Minimum for HA
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 0  # Zero-downtime: never remove existing pod before new one is ready
      maxSurge: 1        # Allow 1 extra pod during rollout
```

### Pod Anti-Affinity

Pods are spread across nodes when possible:

```yaml
affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchLabels:
              app: lurus-api
          topologyKey: kubernetes.io/hostname
```

### PodDisruptionBudget

Ensures at least 1 pod remains available during voluntary disruptions:

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: lurus-api-pdb
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: lurus-api
```

## State Management

### Stateless Components

| Component | State Storage | HA Behavior |
|-----------|---------------|-------------|
| Sessions | Redis | Shared across replicas |
| Rate Limiting | Redis | Shared across replicas |
| Channel Cache | PostgreSQL | Each replica syncs independently |
| JWKS Cache | Zitadel endpoint | Each replica refreshes independently |
| Model Ratios | PostgreSQL | Each replica loads from DB |

### Potential Issues

1. **Channel Cache Lag**: Each replica syncs every 60 seconds. During this window, different replicas may have slightly different channel lists. This is acceptable for load balancing.

2. **JWKS Key Rotation**: If Zitadel rotates keys, each replica discovers new keys on next JWKS refresh (1 hour) or on first key-not-found error.

## Deployment Procedures

### Scale Up

```bash
kubectl scale deployment lurus-api -n lurus-system --replicas=3
```

### Scale Down

```bash
kubectl scale deployment lurus-api -n lurus-system --replicas=2
```

### Rolling Update

Rolling updates are automatic when deployment spec changes:

```bash
kubectl rollout status deployment/lurus-api -n lurus-system
```

### Rollback

```bash
kubectl rollout undo deployment/lurus-api -n lurus-system
```

## Health Checks

### Liveness Probe

Checks if the application is running:

```yaml
livenessProbe:
  httpGet:
    path: /api/status
    port: 3000
  initialDelaySeconds: 30
  periodSeconds: 15
```

### Readiness Probe

Checks if the application can serve traffic:

```yaml
readinessProbe:
  httpGet:
    path: /api/status
    port: 3000
  initialDelaySeconds: 10
  periodSeconds: 5
```

## Monitoring

### Key Metrics

| Metric | Threshold | Action |
|--------|-----------|--------|
| `request_duration_seconds_p95` | > 500ms | Investigate latency |
| `error_rate_5m` | > 5% | Check logs, consider rollback |
| `pod_restart_count` | > 3/hour | Check OOM, probe failures |

### Verify HA Status

```bash
# Check replica count
kubectl get deployment lurus-api -n lurus-system

# Check pod distribution
kubectl get pods -n lurus-system -l app=lurus-api -o wide

# Check PDB status
kubectl get pdb lurus-api-pdb -n lurus-system
```

## Troubleshooting

### Pods Not Spreading Across Nodes

Check if nodes have the required label and capacity:

```bash
kubectl get nodes -l lurus.cn/vpn=true
kubectl describe node <node-name> | grep -A5 Allocatable
```

### Rolling Update Stuck

Check for readiness probe failures:

```bash
kubectl describe pod -l app=lurus-api -n lurus-system | grep -A10 Events
```

### Session Loss After Deployment

Ensure `SESSION_SECRET` is identical across all replicas and Redis is accessible.

## Capacity Planning

| Replicas | Expected RPS | Memory | CPU |
|----------|--------------|--------|-----|
| 2 | 100-500 | 2Gi | 1 core |
| 3 | 500-1000 | 3Gi | 1.5 cores |
| 4+ | 1000+ | 4Gi+ | 2+ cores |

## References

- [Kubernetes Rolling Updates](https://kubernetes.io/docs/tutorials/kubernetes-basics/update/update-intro/)
- [Pod Disruption Budgets](https://kubernetes.io/docs/tasks/run-application/configure-pdb/)
- [Pod Anti-Affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity)
