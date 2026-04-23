# Staging Environment Runbook

## Overview

The staging environment (`lurus-staging` namespace) provides a pre-production testing environment that mirrors production configuration with reduced resources.

| Property | Value |
|----------|-------|
| Namespace | `lurus-staging` |
| URL | https://staging-api.lurus.cn |
| Database | `lurusapi_staging` (separate schema) |
| Redis DB | 1 (production uses 0) |
| Replicas | 1 |
| Image Tag | `staging` |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     K3s Cluster                              │
├─────────────────────────────────┬───────────────────────────┤
│     lurus-system (Production)   │   lurus-staging           │
├─────────────────────────────────┼───────────────────────────┤
│  lurus-api (2 replicas)         │  lurus-api (1 replica)    │
│  PostgreSQL: lurusapi           │  PostgreSQL: lurusapi_stg │
│  Redis DB: 0                    │  Redis DB: 1              │
│  Meilisearch: default           │  Meilisearch: staging_*   │
└─────────────────────────────────┴───────────────────────────┘
```

## Setup Instructions

### 1. Create Staging Database

```bash
# Connect to PostgreSQL
ssh root@100.98.57.55 "kubectl exec -it postgres-0 -n databases -- psql -U lurus"

# Create staging database
CREATE DATABASE lurusapi_staging;
GRANT ALL PRIVILEGES ON DATABASE lurusapi_staging TO lurus;
```

### 2. Create Staging Secrets

```bash
# Generate session secret
SESSION_SECRET=$(openssl rand -hex 32)

# Create secrets
kubectl -n lurus-staging create secret generic lurus-api-staging-secrets \
  --from-literal=SESSION_SECRET="$SESSION_SECRET" \
  --from-literal=SQL_DSN='postgres://lurus:YOUR_PASSWORD@100.94.177.10:30543/lurusapi_staging' \
  --from-literal=ZITADEL_CLIENT_ID='YOUR_STAGING_CLIENT_ID'
```

### 3. Create Zitadel Staging Application

1. Login to https://auth.lurus.cn
2. Create new OIDC Application "lurus-api-staging"
3. Set redirect URI: `https://staging-api.lurus.cn/api/v2/oauth/callback`
4. Copy Client ID to secrets

### 4. Deploy Staging Environment

```bash
# Apply manifests
kubectl apply -k deploy/k8s/staging/

# Verify deployment
kubectl -n lurus-staging get pods
kubectl -n lurus-staging get svc
kubectl -n lurus-staging get ingressroute
```

### 5. Configure DNS

Add DNS record for staging:
```
staging-api.lurus.cn  A  <K3s Ingress IP>
```

## Deployment

### Automatic Deployment

Staging is automatically deployed when:
- Push to `main` branch
- PR merged to `main`
- Manual workflow dispatch

GitHub Actions workflow: `.github/workflows/deploy-staging.yml`

### Manual Deployment

```bash
# Build and push staging image
docker build -t ghcr.io/LurusTech/lurus-api:staging .
docker push ghcr.io/LurusTech/lurus-api:staging

# Deploy
kubectl apply -k deploy/k8s/staging/
kubectl -n lurus-staging rollout restart deployment/lurus-api
```

## Verification

### Health Check

```bash
curl https://staging-api.lurus.cn/api/status
```

Expected response:
```json
{"success": true, "message": "pong", "data": {...}}
```

### OAuth Flow Test

1. Navigate to: https://staging-api.lurus.cn/api/v2/staging/auth/login
2. Complete Zitadel authentication
3. Verify redirect and session creation

### API Test

```bash
# Create test token (after login)
curl -X POST https://staging-api.lurus.cn/api/v2/staging/tokens \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "test-token"}'
```

## Monitoring

### View Logs

```bash
# Stream logs
kubectl -n lurus-staging logs -f deployment/lurus-api

# View recent logs
kubectl -n lurus-staging logs deployment/lurus-api --tail=100
```

### View Metrics

Staging metrics are available at:
```
https://staging-api.lurus.cn/metrics
```

### View Traces

Staging has 100% trace sampling. View in Jaeger:
```
https://jaeger.lurus.cn (filter by service: lurus-api, environment: staging)
```

## Troubleshooting

### Pod Not Starting

```bash
# Check pod status
kubectl -n lurus-staging describe pod -l app=lurus-api

# Check events
kubectl -n lurus-staging get events --sort-by=.lastTimestamp
```

### Database Connection Issues

```bash
# Test database connectivity
kubectl -n lurus-staging exec -it deployment/lurus-api -- \
  sh -c 'nc -zv 100.94.177.10 30543'
```

### Certificate Issues

```bash
# Check certificate status
kubectl -n lurus-staging get certificate
kubectl -n cert-manager logs -l app=cert-manager
```

## Differences from Production

| Aspect | Production | Staging |
|--------|------------|---------|
| Replicas | 2 | 1 |
| Resources | 256Mi-1Gi / 100m-500m | 128Mi-512Mi / 50m-250m |
| Redis DB | 0 | 1 |
| Trace Sampling | 10% | 100% |
| Database | lurusapi | lurusapi_staging |
| PDB | Yes (minAvailable: 1) | No |

## Cleanup

To remove staging environment:

```bash
kubectl delete namespace lurus-staging
```

Note: This will delete all staging resources but preserve the database.
