# skills.md

Guidance on which Claude Code skills to reach for at different points in building this project, plus the technical domains this project touches. Read alongside `CLAUDE.md` (operating rules) and `PROJECT_PLAN.md` (architecture).

## Claude Code skill → project task mapping

| When you're about to... | Use this skill |
|---|---|
| Design a new feature, endpoint, or tenant-facing behavior before writing code | `brainstorming` |
| Turn a spec/requirement into a concrete multi-step implementation plan | `writing-plans` |
| Implement any feature or bugfix | `test-driven-development` — write the tenant-isolation test first, especially for anything touching `tenant_id`-scoped data |
| Investigate a bug, test failure, or unexpected behavior | `systematic-debugging` — before proposing a fix |
| Execute a plan that has independent, parallelizable tasks | `subagent-driven-development` or `dispatching-parallel-agents` |
| Start feature work that shouldn't disturb the current workspace | `using-git-worktrees` |
| Finish a feature and decide how to land it | `finishing-a-development-branch` |
| Verify a bugfix/feature is actually done before saying so | `verification-before-completion` |
| Before merging, or after implementing anything touching auth/tenant isolation/secrets | `security-review` — non-optional given the multi-tenant threat model in `PROJECT_PLAN.md` |
| Review a completed chunk of work against the plan | `code-review` (or `review-panel` for a fuller multi-lens pass) |
| Get review feedback and decide what to act on | `receiving-code-review` |
| Clean up/simplify already-working code | `simplify` |
| Run the app locally to confirm a change works | `run` |

## Technical skill domains this project requires

These aren't Claude Code skills — they're the engineering competencies the codebase will exercise, useful context when scoping who/what works on a given task:

- **Go backend development**: idiomatic Go, concurrency patterns, `net/http`-based or lightweight-router-based services.
- **PostgreSQL**: schema design with `tenant_id` on every tenant-owned table, Row-Level Security policy authoring, migrations, read-replica-aware query patterns.
- **Redis**: caching patterns, session/rate-limit storage.
- **Docker**: multi-stage builds, minimal runtime images.
- **Kubernetes**: Deployments, HPA, Ingress/TLS, Secrets management, writing manifests/Helm charts that stay cloud-agnostic.
- **Multi-tenant security**: authorization scoping, tenant-isolation testing, JWT-based tenant context propagation.
- **Observability**: structured logging (with `tenant_id` on every line), metrics, tracing — required from Phase 1, not bolted on later.

## Notes

- This file should be revisited once real code exists — add any project-specific skills (e.g. a custom `/deploy` or `/migrate` skill) here as they're created under `.claude/skills/`.
