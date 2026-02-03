# Database Runbook / 数据库手册

> Engine: PostgreSQL 15 | Host: 100.94.177.10:30543 | DB: lurusapi

---

## 1. Connection

### Production

```bash
# From local (via Tailscale/VPN)
psql "postgres://lurus:LurusOps2026@100.94.177.10:30543/lurusapi?sslmode=disable"

# From K3s pod
ssh root@100.98.57.55 "kubectl exec -n lurus-system deploy/lurus-api -- env | grep SQL_DSN"
```

### Connection Pool (App-Level)

| Variable | Default | Description |
|----------|---------|-------------|
| `SQL_MAX_IDLE_CONNS` | 100 | Idle connections in pool |
| `SQL_MAX_OPEN_CONNS` | 1000 | Max concurrent connections |
| `SQL_MAX_LIFETIME` | 60s | Connection max lifetime |

---

## 2. Backup

### Full Backup

```bash
# Backup entire database
pg_dump "postgres://lurus:LurusOps2026@100.94.177.10:30543/lurusapi" \
  --format=custom \
  --file=lurusapi_$(date +%Y%m%d_%H%M%S).dump

# Backup specific tables
pg_dump "postgres://lurus:LurusOps2026@100.94.177.10:30543/lurusapi" \
  --format=custom \
  --table=users --table=tokens --table=channels \
  --file=lurusapi_core_$(date +%Y%m%d_%H%M%S).dump

# Schema only (no data)
pg_dump "postgres://lurus:LurusOps2026@100.94.177.10:30543/lurusapi" \
  --schema-only \
  --file=lurusapi_schema_$(date +%Y%m%d_%H%M%S).sql
```

### Automated Backup (Cron)

```bash
# Add to crontab on DB host
0 2 * * * pg_dump "postgres://lurus:LurusOps2026@100.94.177.10:30543/lurusapi" \
  --format=custom --file=/backups/lurusapi_$(date +\%Y\%m\%d).dump \
  && find /backups -name "lurusapi_*.dump" -mtime +30 -delete
```

---

## 3. Restore

### Full Restore

```bash
# Restore from custom format dump
pg_restore --dbname="postgres://lurus:LurusOps2026@100.94.177.10:30543/lurusapi" \
  --clean --if-exists \
  lurusapi_20260203.dump

# Restore from SQL file
psql "postgres://lurus:LurusOps2026@100.94.177.10:30543/lurusapi" \
  < lurusapi_schema.sql
```

### Restore Specific Tables

```bash
pg_restore --dbname="postgres://lurus:LurusOps2026@100.94.177.10:30543/lurusapi" \
  --table=users --clean --if-exists \
  lurusapi_20260203.dump
```

### Point-in-Time Recovery

Requires WAL archiving configured on PostgreSQL. Not currently enabled — see HA deployment plan for CNPG-based continuous backup.

---

## 4. Migration

### How Migrations Work

GORM `AutoMigrate` runs on **master node startup** only (`NODE_TYPE=master`).

```
Startup → InitDB() → if IsMasterNode → migrateDB()
```

**AutoMigrate behavior:**
- Creates tables that don't exist
- Adds new columns from model definitions
- Does NOT delete columns or change types
- Does NOT drop tables

### Tables Managed by AutoMigrate

| Table | Model | Description |
|-------|-------|-------------|
| channels | Channel | LLM provider channels |
| tokens | Token | API access tokens |
| users | User | User accounts |
| passkey_credentials | PasskeyCredential | WebAuthn credentials |
| options | Option | System configuration |
| redemptions | Redemption | Quota redemption codes |
| abilities | Ability | Channel-model capabilities |
| logs | Log | Request/usage logs |
| midjourneys | Midjourney | Image generation tasks |
| top_ups | TopUp | Payment top-ups |
| quota_data | QuotaData | Quota tracking |
| tasks | Task | Background tasks |
| models | Model | Model definitions |
| vendors | Vendor | Provider vendors |
| prefill_groups | PrefillGroup | Channel groups |
| setups | Setup | System setup state |
| two_fas | TwoFA | 2FA configuration |
| two_fa_backup_codes | TwoFABackupCode | 2FA backup codes |
| checkins | Checkin | User check-in records |
| subscriptions | Subscription | Billing subscriptions |
| internal_api_keys | InternalApiKey | Internal service keys |
| invitation_codes | InvitationCode | Invite system |
| tenants | Tenant | Multi-tenant orgs |
| user_identity_mappings | UserIdentityMapping | Zitadel→Lurus user map |
| tenant_configs | TenantConfig | Per-tenant settings |

### Manual Migration (Schema Changes)

For changes AutoMigrate can't handle (column renames, type changes, drops):

```bash
# Connect to DB
psql "postgres://lurus:LurusOps2026@100.94.177.10:30543/lurusapi"

# Example: rename column
ALTER TABLE users RENAME COLUMN old_name TO new_name;

# Example: add index
CREATE INDEX CONCURRENTLY idx_logs_tenant_id ON logs(tenant_id);

# Example: change column type
ALTER TABLE tokens ALTER COLUMN quota TYPE bigint;
```

### Pre-Migration Checklist

- [ ] Backup database first (see Section 2)
- [ ] Test migration on local/staging DB
- [ ] Check for long-running queries: `SELECT * FROM pg_stat_activity WHERE state = 'active';`
- [ ] If adding index on large table, use `CONCURRENTLY`

---

## 5. Log Database

Optional separate database for high-volume request logs.

```bash
# Enable separate log DB
LOG_SQL_DSN=postgres://lurus:LurusOps2026@100.94.177.10:30543/lurusapi_logs
```

If `LOG_SQL_DSN` is unset, logs share the main database (default).

---

## 6. Monitoring

### Connection Count

```sql
SELECT count(*) FROM pg_stat_activity WHERE datname = 'lurusapi';
```

### Table Sizes

```sql
SELECT relname AS table,
       pg_size_pretty(pg_total_relation_size(relid)) AS total_size
FROM pg_catalog.pg_statio_user_tables
ORDER BY pg_total_relation_size(relid) DESC;
```

### Slow Queries

```sql
-- Requires pg_stat_statements extension
SELECT query, calls, mean_exec_time, total_exec_time
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 20;
```

### Dead Tuples (Vacuum Status)

```sql
SELECT relname, n_dead_tup, last_vacuum, last_autovacuum
FROM pg_stat_user_tables
ORDER BY n_dead_tup DESC;
```

---

## 7. Disaster Recovery

### Database Unreachable

1. Check PostgreSQL is running: `systemctl status postgresql` on DB host
2. Check network: `telnet 100.94.177.10 30543`
3. Check connection pool exhaustion: app logs for "too many connections"
4. Restart app pods to reset pool: `kubectl rollout restart deployment/lurus-api -n lurus-system`

### Data Corruption

1. Stop the application: scale to 0 replicas
2. Restore from latest backup (Section 3)
3. Verify data integrity
4. Scale back up

### Schema Drift

If AutoMigrate fails on startup:

```bash
# Check master node logs for migration errors
ssh root@100.98.57.55 "kubectl logs -n lurus-system deploy/lurus-api | grep -i migration"

# Manual fix: connect to DB and resolve conflicts
psql "postgres://lurus:LurusOps2026@100.94.177.10:30543/lurusapi"
```
