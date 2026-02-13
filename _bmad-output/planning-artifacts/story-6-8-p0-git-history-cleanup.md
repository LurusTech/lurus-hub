# Story 6-8: P0 Git History Cleanup

## Meta
- **Story ID**: 6-8
- **Epic**: 6 (Code Review & Security Hardening)
- **Priority**: P0
- **Status**: done
- **Date**: 2026-02-13

## Problem

`deploy/k8s/secrets.yaml` contains real database credentials in Git history across 4 commits:

| Commit | Contains Real Secrets |
|--------|----------------------|
| `5a946e78` feat: add login configuration | Yes |
| `08476474` feat: multi-tenant V2 API | Yes |
| `c9a736b2` refactor: migrate to clean arch | Yes |
| `78fe6aea` feat: add Alipay payment | Yes |
| `4c3ee830` fix(security): P1 fixes | No (template) |

Leaked values: SQL_DSN (database password `LurusOps2026`), SESSION_SECRET.

Current `secrets.yaml` is already a safe template (fixed in Story 6-4).

## Previous Attempt

2026-02-13 10:56 UTC: Attempted `git filter-branch` on Windows.
- **Failed** at commit 1095/5013 due to `web/.prettierrc.mjs ` (trailing space in filename).
- Repository intact (auto-rollback).

## Approach

Use `git-filter-repo` (Python tool, handles Windows path issues better) to remove
`deploy/k8s/secrets.yaml` from entire history, then re-add the clean template.

### Steps
1. Stash uncommitted changes
2. Create timestamped backup branch
3. Run `git-filter-repo --path deploy/k8s/secrets.yaml --invert-paths --force`
4. Re-add remote origin
5. Re-add template secrets.yaml
6. Pop stash to restore working changes
7. Verify: no secrets in history
8. User confirms force push

## Risk Assessment

- **Data loss**: Mitigated by backup branch + stash
- **Team coordination**: Required before force push (all must re-clone)
- **Trailing space issue**: `git-filter-repo` handles this better than `filter-branch`
- **Rollback**: `git reset --hard backup-before-history-cleanup-*`

## Credential Rotation

Even after history cleanup, leaked credentials should be rotated:
- Generate new SESSION_SECRET: `openssl rand -base64 32`
- Database password: team decision (company standard)
- Redeploy with new secrets

## Files Affected

- `deploy/k8s/secrets.yaml` (removed from history, re-added as template)
- `scripts/cleanup-secrets-history.sh` (reference script, already exists)
- `doc/code-review/P0-1-git-history-cleanup-guide.md` (updated with results)

## Definition of Done

- [x] `git-filter-repo` executed successfully (5.37s, 5019 commits rewritten)
- [x] No secrets found in history: `git log --all -p -S "LurusOps2026" -- deploy/k8s/secrets.yaml` returns empty
- [x] Template `secrets.yaml` re-added to HEAD
- [x] All working changes preserved (stash pop successful, 30 files + 3 untracked)
- [ ] Force push to remote completed (requires user confirmation)
- [x] Credential rotation plan documented (in P0-1-git-history-cleanup-guide.md)
- [x] `go build ./cmd/server` passes after cleanup
- [x] `go test -short ./internal/pkg/...` passes after cleanup (all PASS)

## Note on Remaining Secrets in Documentation

The leaked password also appears in documentation files (DEPLOY.md, deploy/k8s/README.md,
doc/runbook/database.md, doc/code-review/*.md). These are deliberate documentation references
per team decision ("company standard password"). Cleaning these is out of scope for this story
but should be addressed in a follow-up if password rotation is performed.
