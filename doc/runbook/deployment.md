# Deployment Runbook / 部署手册

> Service: lurus-api | Namespace: lurus-system | Host: api.lurus.cn

---

## 1. Build

### CI/CD Pipeline (GitOps)

Push to `main` triggers GitHub Actions → GHCR → ArgoCD sync.

```
main push → .github/workflows/docker-image-main.yml
           → ghcr.io/LurusTech/lurus-api:main-YYYYMMDD-<sha>
           → ArgoCD detects new image → auto-sync
```

### Manual Build (Local)

```bash
# Frontend
cd web && bun install && bun run build

# Backend
CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -trimpath -o lurus-api ./cmd/server

# Docker
docker build -t lurus-api:local .
```

### Image Tags

| Branch | Tag Format | Registry |
|--------|-----------|----------|
| main | `main-YYYYMMDD-<sha>` | GHCR |
| alpha | `alpha-YYYYMMDD-<sha>` | GHCR + Docker Hub |
| tag (release) | `v1.2.3`, `latest` | GHCR + Docker Hub |

---

## 2. Deploy

### Prerequisites

- K3s access: `ssh root@100.98.57.55`
- `kubectl` configured for lurus-system namespace
- ArgoCD synced (check ArgoCD dashboard)

### Deploy via ArgoCD (Standard)

```bash
# Merge to main → CI builds image → ArgoCD auto-syncs
git push origin main

# Verify ArgoCD sync status
ssh root@100.98.57.55 "kubectl get application lurus-api -n argocd -o jsonpath='{.status.sync.status}'"
```

### Deploy via kubectl (Manual Override)

```bash
# Update image manually
ssh root@100.98.57.55 "kubectl set image deployment/lurus-api \
  lurus-api=ghcr.io/LurusTech/lurus-api:<tag> \
  -n lurus-system"

# Or restart current deployment
ssh root@100.98.57.55 "kubectl rollout restart deployment/lurus-api -n lurus-system"
```

### Kustomize Apply (Full Manifest Update)

```bash
ssh root@100.98.57.55 "kubectl apply -k deploy/k8s/"
```

---

## 3. Verify

### Health Check

```bash
# Readiness (should return 200)
curl -s -o /dev/null -w "%{http_code}" https://api.lurus.cn/api/status

# From inside cluster
ssh root@100.98.57.55 "kubectl exec -n lurus-system deploy/lurus-api -- wget -qO- http://localhost:3000/api/status"
```

### Pod Status

```bash
ssh root@100.98.57.55 "kubectl get pods -n lurus-system -l app=lurus-api"
ssh root@100.98.57.55 "kubectl describe pod -n lurus-system -l app=lurus-api"
```

### Logs

```bash
# Recent logs
ssh root@100.98.57.55 "kubectl logs -n lurus-system deploy/lurus-api --tail=100"

# Follow logs
ssh root@100.98.57.55 "kubectl logs -n lurus-system deploy/lurus-api -f"

# Previous container (after crash/restart)
ssh root@100.98.57.55 "kubectl logs -n lurus-system deploy/lurus-api --previous"
```

### Smoke Test

```bash
# v1 API
curl -s https://api.lurus.cn/api/status | jq .

# v2 API (requires auth)
curl -s -H "Authorization: Bearer <token>" https://api.lurus.cn/api/v2/<tenant>/tokens | jq .
```

---

## 4. Rollback

### ArgoCD Rollback

```bash
# List revision history
ssh root@100.98.57.55 "kubectl get application lurus-api -n argocd -o jsonpath='{.status.history}' | jq"

# Rollback to previous revision via ArgoCD CLI
ssh root@100.98.57.55 "argocd app rollback lurus-api"
```

### kubectl Rollback

```bash
# View rollout history
ssh root@100.98.57.55 "kubectl rollout history deployment/lurus-api -n lurus-system"

# Undo to previous version
ssh root@100.98.57.55 "kubectl rollout undo deployment/lurus-api -n lurus-system"

# Undo to specific revision
ssh root@100.98.57.55 "kubectl rollout undo deployment/lurus-api -n lurus-system --to-revision=<N>"

# Watch rollout status
ssh root@100.98.57.55 "kubectl rollout status deployment/lurus-api -n lurus-system"
```

### Pin to Specific Image

```bash
ssh root@100.98.57.55 "kubectl set image deployment/lurus-api \
  lurus-api=ghcr.io/LurusTech/lurus-api:main-20260201-abc1234 \
  -n lurus-system"
```

---

## 5. Configuration

### Environment Variables (via K8s Secrets)

```bash
# View current secrets (base64 encoded)
ssh root@100.98.57.55 "kubectl get secret lurus-api-secrets -n lurus-system -o yaml"

# Update a secret value
ssh root@100.98.57.55 "kubectl create secret generic lurus-api-secrets \
  --from-literal=SQL_DSN='postgres://...' \
  --from-literal=SESSION_SECRET='...' \
  -n lurus-system --dry-run=client -o yaml | kubectl apply -f -"

# Restart to pick up new secrets
ssh root@100.98.57.55 "kubectl rollout restart deployment/lurus-api -n lurus-system"
```

### Key Environment Variables

| Variable | Source | Required |
|----------|--------|----------|
| `SQL_DSN` | Secret | Yes (PostgreSQL connection) |
| `SESSION_SECRET` | Secret | Yes |
| `REDIS_CONN_STRING` | ConfigMap | Optional (in-memory if unset) |
| `ZITADEL_CLIENT_ID` | Secret | For v2 auth |
| `ZITADEL_CLIENT_SECRET` | Secret | For v2 auth |
| `MEILI_API_KEY` | Secret | For search |
| `NODE_TYPE` | Env | "master" (default) or "slave" |

---

## 6. Resource Limits

| Resource | Request | Limit |
|----------|---------|-------|
| CPU | 100m | 500m |
| Memory | 256Mi | 1Gi |

Adjust in `deploy/k8s/deployment.yaml` if OOMKilled or CPU throttled.

---

## 7. Pre-Deploy Checklist

- [ ] All tests pass: `go test ./...`
- [ ] No secrets in code: check `.gitignore`
- [ ] Database migration reviewed (GORM AutoMigrate on master startup)
- [ ] Health endpoint responds: `/api/status`
- [ ] ArgoCD sync status: Synced + Healthy
