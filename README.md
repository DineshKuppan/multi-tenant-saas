# multi-tenant-saas

A multi-tenant SaaS API in Go, built for a starting scale of ~100k users with a documented path to scale further. Tenants are isolated at the database layer via `tenant_id` + Postgres Row-Level Security. Full architecture and rationale: [`PROJECT_PLAN.md`](PROJECT_PLAN.md). Guidance for working on this repo with Claude Code: [`CLAUDE.md`](CLAUDE.md), [`skills.md`](skills.md).

**Status**: scaffold stage. One demo route (`/v1/ping`) shows the required tenant-scoping pattern; there's no real authentication yet. See "Future Enhancements" below.

## Clone

```bash
git clone <repo-url> multi-tenant-saas
cd multi-tenant-saas
```

(No remote is configured yet — replace `<repo-url>` once this repo has a real hosting location.)

## Download dependencies

Requires Go 1.25+.

```bash
go mod download
```

## Build

```bash
make build       # -> bin/api
# or
go build -o bin/api ./cmd/api
```

Other common commands:

```bash
make test                                   # go test ./...
make test-one PKG=./internal/tenant RUN=TestFromContext   # run a single test
make lint                                   # golangci-lint run ./... (install separately)
make fmt                                    # gofmt -l -w .
```

## Run locally

The API needs Postgres and Redis. Easiest path is `docker-compose` (below), then run the binary against it:

```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5433/app?sslmode=disable"
export REDIS_ADDR="localhost:6380"
make run
```

## Dockerize

Build the image:

```bash
docker build -t multitenantsaas-api:latest .
```

Or bring up the full stack (api + postgres + redis) with `docker-compose`:

```bash
make docker-up      # docker compose up --build
```

Host ports are remapped to avoid colliding with services that might already be running locally: Postgres on `5433`, Redis on `6380`, API on `8080`. Container-to-container traffic still uses the standard `5432`/`6379` internally.

The schema isn't applied automatically yet — after the containers are healthy, run the migration once:

```bash
docker compose exec -T postgres psql -U postgres -d app -f - < migrations/000001_init_schema.up.sql
```

Verify it's up:

```bash
curl http://localhost:8080/healthz
curl -H "X-Debug-Tenant-ID: 11111111-1111-1111-1111-111111111111" http://localhost:8080/v1/ping
```

Tear down:

```bash
make docker-down    # docker compose down
```

### Migrations (outside docker-compose)

Migrations are plain SQL files in `migrations/`, applied with the [`golang-migrate`](https://github.com/golang-migrate/migrate) CLI (installed separately — it's not a Go module dependency, to keep the binary lean):

```bash
make migrate-up DATABASE_URL="postgres://postgres:postgres@localhost:5433/app?sslmode=disable"
make migrate-down DATABASE_URL="postgres://postgres:postgres@localhost:5433/app?sslmode=disable"
```

## Kubernetes

Manifests live under `deploy/k8s/`: a `base` (Deployment, Service, HPA, Ingress, ConfigMap) plus `staging`/`production` overlays, built with `kubectl`'s built-in Kustomize support. They're written to be cloud-agnostic — no managed-service-specific APIs.

Render and inspect before applying:

```bash
kubectl kustomize deploy/k8s/overlays/staging
```

Create the database secret out-of-band — it's deliberately **not** committed as a real manifest (see `deploy/k8s/base/secret.example.yaml` for the template and format):

```bash
kubectl create secret generic api-secrets \
  --from-literal=database-url="postgres://user:pass@host:5432/app?sslmode=require" \
  -n staging
```

Apply an overlay:

```bash
kubectl apply -k deploy/k8s/overlays/staging
# or
kubectl apply -k deploy/k8s/overlays/production
```

Notes:
- `deployment.yaml` pulls `multitenantsaas-api:latest` by default — point it at a real registry image before deploying anywhere but a local cluster (e.g. via `kustomize edit set image` in CI, or a per-environment overlay patch).
- The Ingress assumes an nginx-ingress-controller and cert-manager are already installed in the cluster; adjust `ingressClassName`, `host`, and the `cert-manager.io/cluster-issuer` annotation to match your cluster.
- HPA scales the API deployment 2–10 replicas on CPU utilization; tune once real load data exists.
- In a real cluster, source the database credential from a secrets manager via an external-secrets integration rather than `kubectl create secret` with a literal — see `PROJECT_PLAN.md`.

## Future Enhancements

Tracked in more detail in `PROJECT_PLAN.md`'s "Open Decisions" section:

- **Real authentication**: replace `internal/middleware.DevOnlyTenantFromHeader` (a dev-only placeholder that trusts a client header) with JWT-based tenant/user extraction.
- **Automatic migrations**: currently a manual `psql`/`migrate` invocation — decide between an init container, `go:embed`-based in-binary migration on startup, or a CI/CD step.
- **CI/CD pipeline**: not yet chosen or configured.
- **Billing/subscription integration**, if applicable to the product.
- **Read replicas + PgBouncer** and **Redis-backed caching** for the Phase 2 (~100k users) scaling step described in `PROJECT_PLAN.md`.
- **Per-tenant database escape hatch** for large/enterprise tenants needing stronger isolation or data residency, without changing the default row-level-isolation query patterns.
- **Service decomposition** out of the modular monolith — a Phase 3 decision, made from real bottleneck data rather than upfront.
- **Multi-region deployment**, if/when latency or data-residency requirements demand it.
