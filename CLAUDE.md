# Lurus API

LLM unified gateway. Go 1.25 + Gin + GORM + PostgreSQL/SQLite + Redis + Meilisearch.

Frontend: React 18 + Vite + Semi UI (in `web/`, use **bun**).

## Structure

```
cmd/server/              # Entry point
internal/
├── domain/entity/       # Domain entities (struct definitions, value objects)
├── app/                 # Use case orchestration (business logic)
│   ├── relay/           # AI model relay handlers
│   └── passkey/         # Passkey authentication service
├── adapter/
│   ├── handler/         # HTTP handlers (controllers)
│   │   └── router/      # Route definitions
│   ├── middleware/       # HTTP middleware
│   ├── repo/            # GORM repositories (data access)
│   └── provider/        # AI vendor adaptors
│       ├── common/      # Shared relay utilities
│       ├── constant/    # Relay mode constants
│       └── <vendor>/    # Per-vendor implementations
├── lifecycle/           # Application lifecycle management
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
- `doc/runbook/ha-deployment.md` — high availability deployment
- `doc/runbook/staging-environment.md` — staging environment setup

## BMAD

| Resource | Path |
|----------|------|
| PRD | `./_bmad-output/planning-artifacts/prd.md` |
| Epics | `./_bmad-output/planning-artifacts/epics.md` |
| Architecture | `./_bmad-output/planning-artifacts/architecture.md` |
| Sprint Status | `./_bmad-output/planning-artifacts/sprint-status.yaml` |

### BMAD Workflow Rules (MANDATORY)

**Story 文档强制规则** - 从 Epic 7 开始严格执行：

1. ✅ **Story 文档先行** - 任何 story 在实现前必须先创建文档（使用 `/bmad:bmm:workflows:dev-story`）
2. ❌ **禁止无文档标 done** - 没有 story 文档 = 不能标记为 "done"，只能标记 "in-progress"
3. ✅ **Definition of Done** - 必须通过 `dev-story/checklist.md` 所有检查项才能标 "review"
4. ✅ **验证证据必需** - Story 文档必须包含测试输出、验证命令、实际结果
5. ⚠️ **历史债务标注** - Epic 1-5 标注为 "[Pre-BMAD]"，无 story 文档但有 Git commit 证据

**违反规则 = 工作无效**，必须补文档才能 merge。
