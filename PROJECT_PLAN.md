# Project Plan — Multi-Tenant SaaS Platform

Status: **greenfield / pre-scaffold**. This document records the architectural decisions made before any code exists, so that scaffolding work (by humans or Claude Code) starts from a consistent foundation instead of ad-hoc choices.

## Goals

- Multi-tenant SaaS application serving multiple customer organizations (tenants) from one deployment.
- Support ~100,000 users at initial scale, with a clear path to scale further without a rewrite.
- Containerized with Docker, orchestrated with Kubernetes, cloud-agnostic (no managed-service lock-in baked into the core design).

## Tech Stack Decisions

| Concern | Decision | Rationale |
|---|---|---|
| Backend language | Go | Low resource footprint per instance, strong concurrency model, good fit for horizontal scaling on modest infra at 100k-user scale. |
| Primary datastore | PostgreSQL | Mature, supports row-level security (RLS) natively, strong consistency guarantees needed for billing/tenant data. |
| Multi-tenancy model | Shared database, row-level isolation (`tenant_id` column + Postgres RLS policies) | Cheapest to operate and simplest to migrate/backup at 100k-user scale versus schema- or database-per-tenant. See "Multi-Tenancy Strategy" below. |
| Caching / sessions / rate limiting | Redis | Standard choice for ephemeral shared state across horizontally-scaled Go instances. |
| Containerization | Docker, multi-stage builds | Small production images, fast CI builds. |
| Orchestration | Kubernetes (cloud-agnostic manifests/Helm, no managed-service-specific APIs in app code) | Avoids lock-in; can run on EKS/GKE/AKS or self-managed clusters interchangeably. |
| Local dev | docker-compose | Fast local spin-up of Postgres/Redis/app without needing a cluster. |

These are starting decisions, not permanent constraints — revisit if a specific bottleneck (e.g. a single very large tenant) makes a different tenancy model cheaper.

## Multi-Tenancy Strategy

**Shared database, row-level isolation** is the default:

- Every tenant-owned table carries a `tenant_id` column.
- Postgres Row-Level Security (RLS) policies enforce that a connection can only read/write rows matching the tenant context set for that session/request — this is the last line of defense, not the only one; application code must also scope every query explicitly.
- Tenant context is derived from the authenticated request (JWT claim or equivalent) and set per-request/transaction, never trusted from client-supplied identifiers alone.

**Why not schema-per-tenant or database-per-tenant at this stage:** both scale operationally worse than row-level isolation once tenant count grows into the hundreds/thousands (migrations, connection pool sizing, and backups all multiply per schema/database). Row-level isolation keeps operations linear in data volume, not tenant count.

**Escape hatch for large/enterprise tenants:** the schema should be designed so a specific tenant can be moved to a dedicated database later (e.g. for a large enterprise customer with strict isolation or data-residency requirements) without changing application query patterns — this is a migration path, not a day-one requirement.

## Architecture Overview

- **Ingress / API gateway**: TLS termination, routing into the cluster; also the place rate limiting and request-size limits are enforced.
- **Auth**: issues/validates JWTs carrying `tenant_id` + `user_id` + roles; every downstream service derives tenant context from the token, never from a client-supplied header/body field.
- **Core application**: start as a modular monolith (single Go binary/deployment, internally organized by domain module) rather than microservices from day one — splitting out services is a scaling decision to make once a specific module's load profile diverges from the rest, not an upfront one.
- **Background work**: a job queue (e.g. for emails, exports, billing runs) backed by Redis or Postgres-backed queue, run as separate worker pods so bursty async work doesn't compete with request-serving pods for autoscaling signals.
- **Observability**: structured logging with `tenant_id` on every log line, metrics (Prometheus-compatible), tracing — required from the start because multi-tenant incidents are almost always "which tenant, which query" questions.

## Scaling Roadmap

**Phase 1 — MVP / correctness first**
Modular monolith, single Postgres instance, manual scaling. Priority is correct tenant isolation (tested, not just assumed) and core product functionality. No premature service decomposition or sharding.

**Phase 2 — ~100k users**
- Horizontal Pod Autoscaling on the API deployment based on CPU/request latency.
- Postgres read replicas for read-heavy paths; connection pooling via PgBouncer (Go's per-instance connection counts multiply quickly across autoscaled pods).
- Redis-backed caching for hot, infrequently-changing tenant data.
- Background workers scaled independently from API pods.

**Phase 3 — beyond 100k**
- Revisit tenancy model per-tenant where needed (escape hatch above).
- Consider splitting the highest-load domain modules out of the monolith into separate services, guided by actual bottleneck data, not speculation.
- Multi-region if latency or data-residency requirements demand it.

## Docker

- Multi-stage Dockerfile: build stage compiles the static Go binary, final stage is a minimal (distroless/alpine) runtime image.
- `docker-compose.yml` for local dev: app + Postgres + Redis, seeded with a couple of sample tenants for testing isolation locally.

## Kubernetes

- Namespace-per-environment (e.g. `staging`, `production`).
- Helm chart (or Kustomize) for the app deployment — resource requests/limits set explicitly, not left to defaults.
- Secrets via the cluster's external-secrets integration (not hardcoded manifests) — specific provider chosen at implementation time based on where the cluster runs.
- HPA on the API deployment; separate deployment (and HPA) for background workers.
- Ingress + TLS via cert-manager or equivalent.

## Security & Tenant Isolation

- Tenant isolation must be covered by automated tests that assert cross-tenant access is impossible (not just "not exercised by the happy path").
- RLS policies are defense-in-depth, not a substitute for scoping queries by `tenant_id` in application code.
- No tenant-supplied identifier is ever trusted for authorization decisions without cross-checking against the authenticated session's tenant context.

## Decisions Made During Scaffolding

- **Web framework/router**: stdlib `net/http` (Go 1.22+ `ServeMux`), no third-party router — kept dependencies minimal for the initial scaffold.
- **Migration tool**: `golang-migrate` CLI against plain SQL files in `migrations/`, run as an ops step rather than embedded in the binary.
- **Module path**: `github.com/DineshKuppan/multi-tenant-saas`, matching the repo at `git@github.com:DineshKuppan/multi-tenant-saas.git`.
- **CI**: GitHub Actions (`.github/workflows/ci.yml`). Lint/test/build on every push and PR to `main`; on merge to `main`, build and push the image to GitHub Container Registry as `ghcr.io/dineshkuppan/multi-tenant-saas` using the built-in `GITHUB_TOKEN` — no separate registry account/credentials to manage. The image path is lowercased in the workflow since ghcr.io rejects uppercase, even though the GitHub repo itself is `DineshKuppan/multi-tenant-saas`. Chosen over Docker Hub specifically to avoid a second account/secret; chosen over adding a CD/auto-deploy step because there's no live cluster yet to deploy to safely.
- Schema is not yet applied automatically on `docker-compose up`; that's an open item (see below).

## Open Decisions (to make at implementation time, not guessed here)

- Continuous deployment: CI publishes images to GHCR but nothing applies them to a cluster automatically. Add a deploy job once a real cluster and credentials exist — likely `kubectl apply -k deploy/k8s/overlays/staging` on merge to `main`, `overlays/production` gated on a tag or manual approval.
- Billing/subscription integration, if applicable.
- How migrations get applied automatically (init container, `go:embed` + in-binary migration on startup, or a CI/CD step) — currently a manual `psql`/`migrate` invocation.
- Real authentication/JWT issuance and validation — `internal/middleware.DevOnlyTenantFromHeader` is a dev-only placeholder, not a design for the real auth path.
- `deploy/k8s/base/deployment.yaml` pins `ghcr.io/dineshkuppan/multi-tenant-saas:latest` — fine for now, but a pinned digest/sha tag would be safer than `:latest` for anything beyond a dev cluster.
