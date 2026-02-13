# P0-1: Git History Cleanup Guide

**Issue**: `deploy/k8s/secrets.yaml` contained real credentials in Git history (5 commits)

**Status**: DONE - Cleaned on 2026-02-13 using `git-filter-repo`

---

## Execution Summary

**Date**: 2026-02-13
**Tool**: `git-filter-repo` (Python, v2.47.0)
**Environment**: Windows 10 (succeeded where `git filter-branch` failed)
**Duration**: 5.37 seconds (5019 commits rewritten)

### What Was Done

1. Stashed all uncommitted changes (30 modified + 3 untracked files)
2. Created backup branch `backup-before-filter-repo-20260213`
3. Ran `git-filter-repo --path deploy/k8s/secrets.yaml --invert-paths --force`
4. Re-added origin remote
5. Re-added `secrets.yaml` as clean template (PLACEHOLDER values only)
6. Restored stash (all working changes preserved)

### Verification Results

```
git log --all --oneline -- deploy/k8s/secrets.yaml
  -> Only 1 commit (re-add template)

git log --all -p -S "<redacted-db-password>" -- deploy/k8s/secrets.yaml
  -> Empty (no matches in secrets.yaml history)

git log --all -p -S "<redacted-session-secret>" -- deploy/k8s/secrets.yaml
  -> Empty (no matches in secrets.yaml history)
```

### Remaining Actions

- [ ] Force push to remote: `git push origin --force --all`
- [ ] Notify team members to re-clone
- [ ] Rotate credentials (see Credential Rotation below)
- [ ] GitHub: contact support to clear cached views if repo is public

---

## Previous Attempt

**2026-02-13 10:56 UTC**: `git filter-branch` failed on Windows at commit 1095/5013.

**Error**: `error: invalid path 'web/.prettierrc.mjs '` (trailing space in historical filename).

**Resolution**: `git-filter-repo` handled this correctly.

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
-- Connect to database and change password
ALTER USER lurus WITH PASSWORD '<new-secure-password>';
```

Update in:
- Local credentials file
- `deploy/k8s/secrets.prod.yaml`
- All services using this database

### 3. Invalidate Old Sessions

```bash
# Restart all pods to invalidate sessions with old secret
kubectl rollout restart deployment/lurus-api -n lurus-system
```

---

## Team Coordination

### After Force Push

All team members must:

```bash
# DO NOT merge or rebase from old clone
# Delete local clone and re-clone
cd ~/projects
rm -rf lurus-api/
git clone https://github.com/hanmahong5-arch/lurus-api.git
```

---

## Backup Branches

| Branch | Purpose |
|--------|---------|
| `backup-before-cleanup` | First backup (before filter-branch attempt) |
| `backup-before-history-cleanup-20260213-105623` | Second backup (during filter-branch) |
| `backup-before-filter-repo-20260213` | Final backup (before successful filter-repo) |

These branches contain the OLD history with leaked secrets. They should be deleted
from remote after verifying the cleanup is complete:

```bash
git push origin --delete backup-before-cleanup
git push origin --delete backup-before-history-cleanup-20260213-105623
git push origin --delete backup-before-filter-repo-20260213
```

---

## References

- [git-filter-repo documentation](https://github.com/newren/git-filter-repo/)
- [Removing sensitive data from a repository (GitHub Docs)](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/removing-sensitive-data-from-a-repository)
- Code Review Report: `doc/code-review/2026-02-13-adversarial-code-review.md`
