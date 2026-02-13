# P0-1: Git History Cleanup Guide

**Issue**: `deploy/k8s/secrets.yaml` contains real credentials in Git history (5 commits)

**Status**: ⏳ Pending execution — requires Linux/macOS environment or Git Bash

---

## Current Situation

✅ **Fixed**:
- Current `secrets.yaml` is now a template (PLACEHOLDER values only)
- `.gitignore` updated to prevent `secrets.prod.yaml` from being committed
- All team members aware of the issue

❌ **Still vulnerable**:
- Git history contains real database password `LurusOps2026`
- Git history contains real session secret `LurusApiSessionSecret2026Secure!`
- Anyone with repository access can view history: `git log -p -- deploy/k8s/secrets.yaml`

**Leaked values**:
```yaml
SQL_DSN: "postgres://lurus:LurusOps2026@100.94.177.10:30543/lurusapi?sslmode=disable"
SESSION_SECRET: "LurusApiSessionSecret2026Secure!"
```

**Commits containing real secrets**:
```
4c3ee830 fix(security): P1 fixes - ✅ CLEANED (template only)
78fe6aea feat: add Alipay payment ❌ CONTAINS REAL VALUES
c9a736b2 refactor: migrate to clean ❌ CONTAINS REAL VALUES
08476474 feat: multi-tenant V2 API ❌ CONTAINS REAL VALUES
5a946e78 feat: add login configuration ❌ CONTAINS REAL VALUES
```

---

## Why This Matters

### Security Risks

1. **Database Access**: Anyone who has ever cloned the repo knows the database password
2. **Session Hijacking**: Old session secret could be used to forge sessions
3. **Compliance**: Leaked credentials in Git history violate security best practices
4. **Audit Trail**: Future security audits will flag this as a vulnerability

### Mitigation Options

| Option | Security Impact | Operational Impact |
|--------|----------------|-------------------|
| **A. Clean Git history** | ✅ Eliminates leak | ⚠️ Team must re-clone |
| **B. Rotate credentials** | ✅ Old values invalid | ⚠️ Update all services |
| **C. Accept risk** | ❌ Leak persists | ✅ No disruption |

**Recommended**: **Option A + B** (clean history AND rotate credentials)

---

## Execution Plan

### Prerequisites

1. **Environment**: Linux, macOS, or Git Bash (Windows native Git has file locking issues)
2. **Coordination**: Notify all team members (re-clone required)
3. **Timing**: During low-activity period (avoid active development)
4. **Backup**: Current state already backed up to `backup-before-cleanup` branch

### Automated Script

**Location**: `scripts/cleanup-secrets-history.sh`

**Usage**:
```bash
cd /path/to/lurus-api
bash scripts/cleanup-secrets-history.sh
```

**What it does**:
1. Creates timestamped backup branch
2. Removes `secrets.yaml` from entire Git history
3. Re-adds current template version
4. Provides force-push instructions

**Estimated time**: 5-10 minutes (depending on repo size)

---

## Manual Cleanup Steps

If you prefer manual control, follow these steps:

### Step 1: Create Backup

```bash
git branch backup-before-cleanup
git log --oneline -5  # Verify current state
```

### Step 2: Remove File from History

**Option A: git-filter-repo (recommended)**

```bash
# Install (if not already)
pip install git-filter-repo

# Remove file from history
git filter-repo --path deploy/k8s/secrets.yaml --invert-paths --force
```

**Option B: git filter-branch (slower)**

```bash
git filter-branch --force --index-filter \
  'git rm --cached --ignore-unmatch deploy/k8s/secrets.yaml' \
  --prune-empty --tag-name-filter cat -- --all

# Clean up
rm -rf .git/refs/original/
git reflog expire --expire=now --all
git gc --prune=now --aggressive
```

### Step 3: Re-add Template

```bash
# Restore current template version
git checkout backup-before-cleanup -- deploy/k8s/secrets.yaml

# Commit
git add deploy/k8s/secrets.yaml
git commit -m "chore: re-add secrets.yaml as template (no real values)"
```

### Step 4: Verify Cleanup

```bash
# Should only show the re-add commit
git log --all -- deploy/k8s/secrets.yaml

# Should return empty (no real values in history)
git grep -i "LurusOps2026" $(git rev-list --all) -- deploy/k8s/secrets.yaml || echo "✅ Clean"
```

### Step 5: Force Push

```bash
# Push rewritten history
git push origin --force --all
git push origin --force --tags
```

---

## Team Coordination Checklist

### Before Execution

- [ ] Schedule cleanup during low-activity period
- [ ] Notify all team members via Slack/email
- [ ] Ensure all pending work is committed/pushed
- [ ] Create backup branches: `git branch backup-before-cleanup`

### Communication Template

```
Subject: [ACTION REQUIRED] Git Repository History Rewrite - 2026-02-13

Team,

We will perform a Git history cleanup to remove leaked credentials from the repository history.

**Timeline**:
- Execution: [DATE] at [TIME]
- Expected duration: 15 minutes

**Required Actions**:
1. Commit and push all pending work BEFORE [TIME]
2. AFTER cleanup is complete:
   - Delete your local clone: rm -rf lurus-api/
   - Re-clone: git clone <repo-url>
   - DO NOT attempt to merge/pull from old clone

**Why**: Security audit found database credentials in Git history (commits 78fe6aea, c9a736b2, 08476474, 5a946e78)

**Reference**: doc/code-review/P0-1-git-history-cleanup-guide.md

Questions? Reply to this thread.
```

### After Execution

- [ ] Verify cleanup: `git log --all -- deploy/k8s/secrets.yaml` (should be 1 commit)
- [ ] Test clone in new directory: `git clone <repo-url> /tmp/verify && cd /tmp/verify`
- [ ] Notify team that cleanup is complete
- [ ] Update DEPLOY.md with new session secret instructions
- [ ] **IMPORTANT**: Rotate credentials (see Credential Rotation below)

---

## Credential Rotation

Even after cleaning Git history, **rotate all leaked credentials**:

### 1. Generate New Session Secret

```bash
openssl rand -base64 32
```

Update in:
- `deploy/k8s/secrets.prod.yaml`
- All deployed environments (staging, production)

### 2. Rotate Database Password (Optional)

```sql
-- Connect to database
psql "postgres://lurus:LurusOps2026@100.94.177.10:30543/lurusapi"

-- Change password
ALTER USER lurus WITH PASSWORD 'NewSecurePassword2026!';
```

Update in:
- `重要信息.md`
- `deploy/k8s/secrets.prod.yaml`
- All services using this database

**Note**: According to `DEPLOY.md` line 114, team decided to keep database password as company standard. If this is still the case, at least clean Git history to limit exposure.

### 3. Invalidate Old Sessions

```bash
# Restart all pods to invalidate sessions with old secret
kubectl rollout restart deployment/lurus-api -n lurus-system
```

---

## Troubleshooting

### Issue: "Cannot remove .git-rewrite/"

**Cause**: Windows file locking or interrupted filter-branch

**Solution**:
```bash
# Kill any Git processes
taskkill /F /IM git.exe

# Manually delete (PowerShell)
Remove-Item -Recurse -Force .git-rewrite

# Or use Git Bash / WSL
```

### Issue: "Reference already exists"

**Cause**: Previous cleanup attempt left refs

**Solution**:
```bash
rm -rf .git/refs/original/
git update-ref -d refs/original/refs/heads/main
```

### Issue: "Team member can't push after cleanup"

**Cause**: Their local repo still has old history

**Solution**:
```bash
# DO NOT merge or rebase - delete and re-clone
cd /path/to/projects
rm -rf lurus-api/
git clone <repo-url>
```

---

## Alternative: BFG Repo-Cleaner

For faster cleanup (10x faster than filter-branch):

```bash
# Install
brew install bfg  # macOS
# Or download JAR from: https://rtyley.github.io/bfg-repo-cleaner/

# Clone mirror
git clone --mirror <repo-url> lurus-api-mirror.git
cd lurus-api-mirror.git

# Remove file
bfg --delete-files secrets.yaml

# Clean up
git reflog expire --expire=now --all
git gc --prune=now --aggressive

# Push
git push
```

---

## Decision Log

**2026-02-13 10:56 UTC**: Attempted cleanup on Windows with `git filter-branch`

**Result**: ❌ Failed at commit 1095/5013

**Error**:
```
error: invalid path 'web/.prettierrc.mjs '
Could not initialize the index
```

**Root Cause**: Historical commit contains filename with trailing space (`web/.prettierrc.mjs `), which is invalid on Windows filesystem. Git filter-branch cannot process this file on Windows.

**Repository Status**: ✅ Intact (filter-branch auto-rollback, no damage)

**Solutions**:

1. **Option A (Recommended)**: Execute on Linux/macOS
   - Use automated script: `bash scripts/cleanup-secrets-history.sh`
   - No trailing space issue on Unix filesystems
   - Estimated time: 5-10 minutes

2. **Option B**: Use BFG Repo-Cleaner on Windows
   - BFG handles Windows path issues better than filter-branch
   - Download: https://rtyley.github.io/bfg-repo-cleaner/
   - Faster than filter-branch (10x speed)

3. **Option C**: Fix trailing space first (advanced)
   ```bash
   # Identify problematic commit
   git log --all --pretty="%H %s" -- 'web/.prettierrc.mjs '

   # Use git-filter-repo (better than filter-branch)
   pip install git-filter-repo
   git filter-repo --path deploy/k8s/secrets.yaml --invert-paths
   ```

**Recommended next action**:
1. Schedule cleanup for [DATE/TIME]
2. Use **Linux/macOS environment** (not Windows)
3. Use automated script: `scripts/cleanup-secrets-history.sh`
4. Coordinate with team (re-clone required)

**Risk if not cleaned**:
- Database credentials remain in Git history
- Potential compliance violation
- Security audit finding

**Acceptable delay**: Up to 1 week (current `secrets.yaml` is clean, only history affected)

---

## References

- [git-filter-repo documentation](https://github.com/newren/git-filter-repo/)
- [BFG Repo-Cleaner](https://rtyley.github.io/bfg-repo-cleaner/)
- [Removing sensitive data from a repository (GitHub Docs)](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository)
- Code Review Report: `doc/code-review/2026-02-13-adversarial-code-review.md`
