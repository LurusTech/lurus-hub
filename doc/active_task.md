# Active Task: README rename + push

## Context
GitHub repo renamed from `LurusTech/lurus-api` to `LurusTech/lurus-hub`. README still says "Lurus API" everywhere; project positioning shifted to "AI 数据处理枢纽" per CLAUDE.md. User wants README updated, then entire working tree (375 modified + untracked files) committed and pushed.

## Critical Files
- README.md
- (commit target: all modified + untracked files in working tree)

## Step-by-Step Plan
- [ ] 1. Edit README.md — title `Lurus API` → `Lurus Hub`, tagline → AI Data Processing Hub, overview rewrite, architecture diagram label, **keep runtime resource names** (lurus-api binary, deployment/lurus-api, ghcr.io/LurusTech/lurus-api).
- [ ] 2. `git add -A` — stage all 375 modifications + untracked files.
- [ ] 3. `git commit` with message covering README rename + bundled in-progress work (openrouter pool/sync, governance, billing, etc.).
- [ ] 4. `git push origin main` — GitHub auto-redirects lurus-api.git → lurus-hub.git.
- [ ] 5. Verify push succeeded; clear this file.

## Current Status
- [ ] In Progress
