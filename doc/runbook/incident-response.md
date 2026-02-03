# Incident Response Runbook / 事件响应手册

> Service: lurus-api | Namespace: lurus-system | On-call: Anita

---

## 1. Health Checks

### Quick Status

```bash
# External (public endpoint)
curl -s -o /dev/null -w "%{http_code}" https://api.lurus.cn/api/status

# Internal (from cluster)
ssh root@100.98.57.55 "kubectl exec -n lurus-system deploy/lurus-api -- wget -qO- http://localhost:3000/api/status"

# Pod status
ssh root@100.98.57.55 "kubectl get pods -n lurus-system -l app=lurus-api -o wide"
```

### Expected Response

- `/api/status` → HTTP 200
- Pod status → `Running`, `READY 1/1`
- Readiness probe: every 5s at `/api/status`
- Liveness probe: every 15s at `/api/status`

---

## 2. Triage Decision Tree

```
Service unreachable?
├── YES → Check Pod status (Section 3)
│   ├── CrashLoopBackOff → Check logs (Section 4)
│   ├── OOMKilled → Increase memory limit
│   ├── ImagePullBackOff → Check GHCR auth / image tag
│   └── Pending → Check node resources / scheduling
│
├── 5xx errors → Check application logs (Section 4)
│   ├── DB connection error → Database runbook
│   ├── Redis connection error → Check Redis pod
│   ├── Panic/goroutine crash → Check SafeGo logs, restart pod
│   └── Upstream LLM timeout → Check channel status
│
└── Slow responses → Check resource usage (Section 5)
    ├── High CPU → Profile with pprof
    ├── High memory → Check for leaks, increase limit
    └── High DB latency → Check pg_stat_activity
```

---

## 3. Pod Issues

### CrashLoopBackOff

```bash
# Check why pod is crashing
ssh root@100.98.57.55 "kubectl describe pod -n lurus-system -l app=lurus-api"
ssh root@100.98.57.55 "kubectl logs -n lurus-system deploy/lurus-api --previous"

# Common causes:
# - DB unreachable at startup → check SQL_DSN, network
# - Missing required env vars → check secrets
# - Failed migration → check DB schema
```

### OOMKilled

```bash
# Check current memory usage
ssh root@100.98.57.55 "kubectl top pod -n lurus-system -l app=lurus-api"

# Increase memory limit (temporary)
ssh root@100.98.57.55 "kubectl set resources deployment/lurus-api \
  --limits=memory=2Gi -n lurus-system"
```

### ImagePullBackOff

```bash
# Check image exists
ssh root@100.98.57.55 "kubectl describe pod -n lurus-system -l app=lurus-api | grep -A5 'Events'"

# Verify GHCR credentials
ssh root@100.98.57.55 "kubectl get secret ghcr-secret -n lurus-system -o yaml"
```

---

## 4. Application Logs

### Log Locations

```bash
# Structured logs (slog)
ssh root@100.98.57.55 "kubectl logs -n lurus-system deploy/lurus-api --tail=200"

# Filter errors
ssh root@100.98.57.55 "kubectl logs -n lurus-system deploy/lurus-api | grep -i 'error\|panic\|fatal'"

# Filter by component
ssh root@100.98.57.55 "kubectl logs -n lurus-system deploy/lurus-api | grep 'relay'"
ssh root@100.98.57.55 "kubectl logs -n lurus-system deploy/lurus-api | grep 'database'"
```

### Common Error Patterns

| Log Pattern | Meaning | Action |
|-------------|---------|--------|
| `failed to initialize database` | DB unreachable at startup | Check DB host, credentials, network |
| `JWKS fetch failed` | Can't reach Zitadel JWKS | Check auth.lurus.cn, network |
| `channel error` | Upstream LLM provider failed | Check channel config, provider status |
| `quota exceeded` | User/tenant over quota limit | Check billing, adjust quota |
| `panic recovered by SafeGo` | Goroutine panic caught | Check stack trace, fix root cause |

---

## 5. Resource Monitoring

### Current Usage

```bash
# Pod CPU/memory
ssh root@100.98.57.55 "kubectl top pod -n lurus-system"

# Node resources
ssh root@100.98.57.55 "kubectl top node"
```

### pprof (Debug Mode)

If `DEBUG=true` or pprof endpoint enabled:

```bash
# CPU profile (30s)
curl -o cpu.prof http://localhost:3000/debug/pprof/profile?seconds=30
go tool pprof cpu.prof

# Memory profile
curl -o mem.prof http://localhost:3000/debug/pprof/heap
go tool pprof mem.prof

# Goroutine dump
curl http://localhost:3000/debug/pprof/goroutine?debug=2
```

### Database Connections

```sql
-- Active connections
SELECT count(*) FROM pg_stat_activity WHERE datname = 'lurusapi';

-- Long-running queries
SELECT pid, now() - pg_stat_activity.query_start AS duration, query
FROM pg_stat_activity
WHERE state = 'active' AND datname = 'lurusapi'
ORDER BY duration DESC;

-- Kill a stuck query
SELECT pg_terminate_backend(<pid>);
```

---

## 6. Common Scenarios

### Scenario: All Relay Requests Failing

1. Check channel status in admin panel
2. Verify upstream provider API key validity
3. Check channel error logs: `kubectl logs ... | grep "channel error"`
4. Test direct upstream connection from pod:
   ```bash
   ssh root@100.98.57.55 "kubectl exec -n lurus-system deploy/lurus-api -- \
     wget -qO- --header='Authorization: Bearer <key>' https://api.openai.com/v1/models"
   ```
5. If provider outage → enable backup channels or notify users

### Scenario: High Latency

1. Check pod resources: `kubectl top pod`
2. Check DB query latency: `pg_stat_statements`
3. Check Redis connectivity: `redis-cli ping`
4. Check if MeiliSearch is reachable (search-heavy requests)
5. Profile with pprof if sustained

### Scenario: Tenant Login Broken

1. Check Zitadel availability: `curl https://auth.lurus.cn/oauth/v2/keys`
2. Check OIDC config matches: Client ID, redirect URI, issuer
3. Check app logs for JWT validation errors
4. Verify tenant record exists in `tenants` table
5. Test callback flow manually with `curl -v`

---

## 7. Escalation

| Severity | Criteria | Response |
|----------|----------|----------|
| P0 | Service completely down, all users affected | Immediate. Rollback if recent deploy. |
| P1 | Major feature broken (relay, auth), >50% users affected | Within 1 hour. Investigate and fix or rollback. |
| P2 | Single tenant affected, degraded performance | Within 4 hours. |
| P3 | Minor issue, workaround available | Next business day. |

### Escalation Contacts

| Role | Contact | When |
|------|---------|------|
| On-call (Anita) | Primary | All incidents |
| Infrastructure | DB host, K3s node | P0/P1 infra issues |

---

## 8. Postmortem Template

After any P0/P1 incident, create `doc/postmortems/YYYY-MM-DD-title.md`:

```markdown
# Incident: <title>
**Date**: YYYY-MM-DD
**Duration**: HH:MM - HH:MM (X minutes)
**Severity**: P0/P1
**Impact**: <who was affected, how>

## Timeline
- HH:MM — <event>
- HH:MM — <event>

## Root Cause
<what caused the incident>

## Resolution
<what fixed it>

## Action Items
- [ ] <preventive measure>
- [ ] <monitoring improvement>
```

---

## 9. Recovery Commands (Quick Reference)

```bash
# Restart pod
ssh root@100.98.57.55 "kubectl rollout restart deployment/lurus-api -n lurus-system"

# Scale to 0 (emergency stop)
ssh root@100.98.57.55 "kubectl scale deployment/lurus-api --replicas=0 -n lurus-system"

# Scale back up
ssh root@100.98.57.55 "kubectl scale deployment/lurus-api --replicas=1 -n lurus-system"

# Rollback to previous version
ssh root@100.98.57.55 "kubectl rollout undo deployment/lurus-api -n lurus-system"

# Force delete stuck pod
ssh root@100.98.57.55 "kubectl delete pod <pod-name> -n lurus-system --force --grace-period=0"

# Check all resources in namespace
ssh root@100.98.57.55 "kubectl get all -n lurus-system"
```
