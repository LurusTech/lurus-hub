# Tenant Onboarding Runbook / 租户入驻手册

> Auth: Zitadel (auth.lurus.cn) | API: api.lurus.cn | Flow: Zitadel Org → API Tenant → User Mapping

---

## Overview

```
Zitadel Organization ──→ API Tenant Record ──→ User Identity Mapping
    (manual)              (API or auto)          (auto on first login)
```

Two modes:
- **Auto-create**: Set `ZITADEL_AUTO_CREATE_TENANT=true` — tenant created on first user login
- **Manual**: Admin creates tenant via API, then maps to Zitadel Org ID

---

## Phase 1: Zitadel Setup (Manual)

### 1.1 Create Organization

1. Login to Zitadel console: https://auth.lurus.cn
2. Navigate to **Organizations** → **+ Create New Organization**
3. Fill in:
   - **Name**: e.g., "Acme Corporation"
   - **Primary Domain**: e.g., "acme-corp"
4. Record the **Organization ID** (e.g., `285895506344386561`)

### 1.2 Create Project

1. Enter the Organization → **Projects** tab
2. Click **+ Create New Project**
3. Fill in:
   - **Name**: `lurus-api`
   - **Role Assertion**: Enabled
   - **Role Check**: Enabled

### 1.3 Create OIDC Application

1. In Project → **Applications** → **+ New**
2. Select **Web** application type
3. Configure:
   - **Name**: `lurus-api-backend`
   - **Auth Method**: PKCE (recommended)
   - **Redirect URIs**:
     ```
     https://api.lurus.cn/api/v2/oauth/callback
     ```
   - **Post Logout Redirect URIs**:
     ```
     https://api.lurus.cn/logout
     ```
   - **Grant Types**: Authorization Code + Refresh Token
   - **Token Settings**:
     - Access Token Type: JWT
     - Access Token Lifetime: 3600s
     - Refresh Token Idle Expiration: 2592000s (30d)
     - Refresh Token Expiration: 7776000s (90d)
4. Save **Client ID** and **Client Secret** immediately (shown only once)

### 1.4 Create Project Roles

In Project → **Roles** tab, create:

| Role Key | Display Name | Description |
|----------|-------------|-------------|
| `admin` | Administrator | Full tenant access |
| `user` | User | Standard access |
| `billing_manager` | Billing Manager | Billing/subscription management |

### 1.5 Update K8s Secrets

```bash
ssh root@100.98.57.55 "kubectl create secret generic lurus-api-secrets \
  --from-literal=ZITADEL_CLIENT_ID='<client_id>' \
  --from-literal=ZITADEL_CLIENT_SECRET='<client_secret>' \
  --from-literal=SQL_DSN='postgres://...' \
  --from-literal=SESSION_SECRET='...' \
  -n lurus-system --dry-run=client -o yaml | kubectl apply -f -"

ssh root@100.98.57.55 "kubectl rollout restart deployment/lurus-api -n lurus-system"
```

---

## Phase 2: API Tenant Creation

### Option A: Auto-Create (Recommended)

With `ZITADEL_AUTO_CREATE_TENANT=true`, the first user login from a new Zitadel Org automatically creates the tenant record.

No manual API call needed.

### Option B: Manual Create via Admin API

```bash
curl -X POST https://api.lurus.cn/api/v2/admin/tenants \
  -H "Content-Type: application/json" \
  -H "Cookie: session=<platform_admin_session>" \
  -d '{
    "zitadel_org_id": "285895506344386561",
    "slug": "acme-corp",
    "name": "Acme Corporation",
    "plan_type": "pro",
    "max_users": 500,
    "max_quota": 5000000
  }'
```

**Response** (201):
```json
{
  "success": true,
  "data": {
    "id": "uuid-xxx",
    "zitadel_org_id": "285895506344386561",
    "slug": "acme-corp",
    "name": "Acme Corporation",
    "status": 1,
    "plan_type": "pro"
  }
}
```

### Tenant Status Values

| Status | Meaning | Effect |
|--------|---------|--------|
| 1 | Enabled | Normal operation |
| 2 | Disabled | Login blocked, data preserved |
| 3 | Suspended | Login blocked, API blocked |

---

## Phase 3: User First Login

### Flow

```
User → /api/v2/acme-corp/auth/login
     → 302 to auth.lurus.cn/oauth/v2/authorize
     → Zitadel login/consent
     → 302 to /api/v2/oauth/callback?code=xxx&state=yyy
     → Exchange code for tokens
     → ZitadelAuth middleware auto-maps user
     → Session created, redirect to app
```

### What Happens Automatically

1. **JWT validated** via JWKS from auth.lurus.cn
2. **Tenant resolved** from `urn:zitadel:iam:org:id` claim → `tenants.zitadel_org_id`
3. **User mapped** from `sub` claim → `user_identity_mappings` record created
4. **Lurus user created** with default quota from tenant plan
5. **Tenant context injected** into request for data isolation

---

## Phase 4: Verification

### Check Tenant Exists

```bash
# Via admin API
curl -s https://api.lurus.cn/api/v2/admin/tenants \
  -H "Cookie: session=<admin_session>" | jq '.data[] | {id, slug, name, status}'

# Via database
psql "postgres://lurus:LurusOps2026@100.94.177.10:30543/lurusapi" \
  -c "SELECT id, slug, name, status, plan_type FROM tenants;"
```

### Check User Mapping

```bash
psql "postgres://lurus:LurusOps2026@100.94.177.10:30543/lurusapi" \
  -c "SELECT zitadel_user_id, lurus_user_id, tenant_id, email FROM user_identity_mappings WHERE tenant_id = '<tenant_id>';"
```

### Test Login Flow

```bash
# Should redirect to Zitadel
curl -v https://api.lurus.cn/api/v2/acme-corp/auth/login?redirect_url=/dashboard
# Expect: 302 → https://auth.lurus.cn/oauth/v2/authorize?...
```

---

## Phase 5: Tenant Management

### Admin Operations

| Operation | Endpoint | Method |
|-----------|----------|--------|
| List tenants | `/api/v2/admin/tenants` | GET |
| Create tenant | `/api/v2/admin/tenants` | POST |
| Get tenant | `/api/v2/admin/tenants/:id` | GET |
| Update tenant | `/api/v2/admin/tenants/:id` | PUT |
| Enable | `/api/v2/admin/tenants/:id/enable` | POST |
| Disable | `/api/v2/admin/tenants/:id/disable` | POST |
| Suspend | `/api/v2/admin/tenants/:id/suspend` | POST |
| Stats | `/api/v2/admin/tenants/:id/stats` | GET |

### Disable a Tenant

```bash
curl -X POST https://api.lurus.cn/api/v2/admin/tenants/<id>/disable \
  -H "Cookie: session=<admin_session>"
```

Effect: All users in tenant lose login access. Data preserved.

### Adjust Quota

```bash
curl -X PUT https://api.lurus.cn/api/v2/admin/tenants/<id> \
  -H "Content-Type: application/json" \
  -H "Cookie: session=<admin_session>" \
  -d '{"max_quota": 10000000, "max_users": 1000}'
```

---

## Troubleshooting

| Problem | Check |
|---------|-------|
| Login redirects but never completes | Verify OIDC redirect URI matches exactly |
| "Tenant not found" on login | Check `ZITADEL_AUTO_CREATE_TENANT=true` or create manually |
| JWT verification fails | Check `ZITADEL_ISSUER`, verify JWKS endpoint reachable |
| User not created on login | Check `ZITADEL_AUTO_CREATE_USER=true` |
| Cross-tenant data visible | Verify tenant_id in request context, check GORM plugin |

### Environment Variables

```bash
ZITADEL_ENABLED=true
ZITADEL_ISSUER=https://auth.lurus.cn
ZITADEL_CLIENT_ID=<from Phase 1.3>
ZITADEL_CLIENT_SECRET=<from Phase 1.3>
ZITADEL_REDIRECT_URI=https://api.lurus.cn/api/v2/oauth/callback
ZITADEL_JWKS_URI=https://auth.lurus.cn/oauth/v2/keys
ZITADEL_AUTO_CREATE_TENANT=true
ZITADEL_AUTO_CREATE_USER=true
ZITADEL_ENABLE_PKCE=true
```
