# Lurus API

LLM unified gateway. Go 1.25 + Gin + GORM + PostgreSQL/SQLite + Redis + Meilisearch.

Frontend: React 18 + Vite + Semi UI (in `web/`, use **bun**).

## Structure

```
cmd/server/              # Entry point
internal/
├── biz/service/         # Business logic (service layer)
├── biz/relay/           # AI model relay + adaptors
├── data/model/          # GORM models
├── server/controller/   # API controllers
├── server/middleware/    # HTTP middleware
├── server/router/       # Route definitions
└── pkg/                 # Shared utilities (config, logger, search, setting)
web/                     # React frontend
deploy/k8s/              # Kubernetes manifests
doc/                     # Documentation
```

## Commands

```bash
# Backend
go build -o lurus-api ./cmd/server
go test -v ./...

# Frontend
cd web && bun install && bun run dev
cd web && bun run typecheck && bun run lint && bun run test

# Deploy
# GitOps via ArgoCD — push to main, ArgoCD syncs
```

## Key References

- `重要信息.md` — production credentials (local only, gitignored)
- `doc/zitadel-setup-guide.md` — auth setup
- API docs: https://docs.lurus.cn/ / https://api.lurus.cn/

## Runbooks

- `doc/runbook/deployment.md` — build, deploy, verify, rollback
- `doc/runbook/database.md` — backup, restore, migration
- `doc/runbook/tenant-onboarding.md` — Zitadel + API tenant setup
- `doc/runbook/incident-response.md` — triage, escalation, postmortem

## BMAD

| Resource | Path |
|----------|------|
| PRD | `../_bmad-output/planning-artifacts/prd-api.md` |
| Epics | `../_bmad-output/planning-artifacts/epics-api.md` |
| Architecture | `../_bmad-output/planning-artifacts/architecture-api.md` |

Worker rules: read assigned story → code in this dir only → do NOT modify `_bmad-output/`
