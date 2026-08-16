# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project status

The Go module is scaffolded (`cmd/api`, `internal/{config,db,cache,tenant,middleware,server}`), containerized, and verified end-to-end against Postgres + Redis via docker-compose, including a real RLS-enforced migration. It implements one demo route (`/v1/ping`) that shows the required tenant-scoping pattern — real domain features still need to be built. The architecture and technology decisions below (see `PROJECT_PLAN.md` for full rationale) are fixed starting points, not open choices to re-litigate.

Module path is `github.com/DineshKuppan/multi-tenant-saas`, matching the repo at `git@github.com:DineshKuppan/multi-tenant-saas.git`.

## Commands

- `make build` — `go build -o bin/api ./cmd/api`
- `make run` — run locally (needs `DATABASE_URL` and, optionally, `REDIS_ADDR`/`PORT` env vars set — see `docker-compose.yml` for example values)
- `make test` — `go test ./...`
- `make test-one PKG=./internal/tenant RUN=TestFromContext` — run a single test
- `make lint` — `golangci-lint run ./...` (not vendored; install separately). `.golangci.yml` uses the v2 config schema (`version: "2"`) — install golangci-lint v2.x (CI pins `v2.12.2`), a v1.x binary will fail with `can't load config: unsupported version`.
- `make fmt` — `gofmt -l -w .`
- `make migrate-up` / `make migrate-down DATABASE_URL=...` — needs the `migrate` CLI (golang-migrate) installed separately; not a Go module dependency
- `make docker-up` / `make docker-down` — full stack (api + postgres + redis) via docker-compose. Host ports are remapped to 5433 (postgres) and 6380 (redis) to avoid colliding with services that may already be running on the host; container-to-container traffic still uses 5432/6379 internally.
- After `docker-up`, the schema isn't applied automatically yet — run migrations manually: `docker compose exec -T postgres psql -U postgres -d app -f - < migrations/000001_init_schema.up.sql` (or via `make migrate-up` if the `migrate` CLI is installed). Automating this is an open item, not yet decided (in-binary `go:embed` migrations vs. an init container vs. a CI/CD step).
- `kubectl kustomize deploy/k8s/base` (or `overlays/staging`, `overlays/production`) — render manifests; `kubectl apply -k <same path>` to apply. Manifests pin `ghcr.io/dineshkuppan/multi-tenant-saas:latest`.
- CI (`.github/workflows/ci.yml`, GitHub Actions): `go vet`/`build`/`test`/`golangci-lint` on every push and PR to `main`; on merge to `main`, builds and pushes the image to `ghcr.io/dineshkuppan/multi-tenant-saas` (`:latest` and `:<sha>`) using the built-in `GITHUB_TOKEN` — no manually configured registry credentials. The image path is lowercased in the workflow (`tr '[:upper:]' '[:lower:]'` on `github.repository`) since ghcr.io rejects uppercase, even though the GitHub repo itself is `DineshKuppan/multi-tenant-saas`. There is no CD step; rolling a new image out to a cluster is still a manual `kubectl apply -k`.

## Fixed architectural decisions

- **Language**: Go.
- **Datastore**: PostgreSQL, single shared database.
- **Multi-tenancy**: row-level isolation — every tenant-owned table has a `tenant_id` column, enforced by Postgres RLS *and* by explicit `tenant_id` scoping in every query. RLS is defense-in-depth, not a substitute for query scoping. Never trust a client-supplied tenant identifier for authorization; always derive tenant context from the authenticated request (JWT claim).
- **Service shape**: modular monolith to start. Do not split into microservices preemptively — that's a Phase 3 decision in `PROJECT_PLAN.md`, made from real bottleneck data.
- **Caching/ephemeral state**: Redis.
- **Containerization**: Docker with multi-stage builds; local dev via `docker-compose`.
- **Orchestration**: Kubernetes, kept cloud-agnostic — avoid writing application code against a specific cloud provider's managed-service APIs.
- **HTTP routing**: stdlib `net/http` (Go 1.22+ `ServeMux` method/path-param routing) — no third-party router. Revisit only if handler/middleware composition genuinely outgrows it; don't add a framework preemptively.
- **Postgres driver**: `pgx/v5` (`internal/db`). Tenant-scoped DB work goes through `db.WithTenant(ctx, tenantID, fn)`, which sets the `app.tenant_id` session variable inside a transaction so RLS policies apply — don't issue tenant-scoped queries outside of it.
- **Migrations**: plain SQL files in `migrations/`, applied via the `golang-migrate` CLI (not a Go dependency, so the binary stays lean) — see Commands below.

## Non-negotiable when writing tenant-facing code

Any code that reads or writes tenant-owned data must be scoped by `tenant_id` derived from the authenticated request context (`internal/tenant.FromContext`) — never from a path/query/body parameter alone. Cross-tenant data leakage is the primary correctness risk in this codebase; when adding tests for tenant-facing features, include a test that asserts another tenant cannot read/write the data.

`internal/middleware.DevOnlyTenantFromHeader` (used by `/v1/ping`) trusts a client-supplied header and is explicitly dev-only scaffolding — it must be replaced by JWT-derived tenant extraction before any tenant-facing route ships for real. Don't extend its use or copy its pattern into new routes.

## Reference

Full architecture, scaling roadmap (Phase 1/2/3), and remaining open implementation decisions live in `PROJECT_PLAN.md`.
