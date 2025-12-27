# WhoKnows (Go) - DevOps Valgfag Project

WhoKnows is a lightweight Go web application built for the DevOps Valgfag course. It demonstrates end-to-end DevOps practices with production parity, automation, and operational correctness.

---

## Features

- Web pages: search, about, login, register, weather
- Session-based authentication (gorilla/sessions + PostgreSQL)
- Search with optional Full-Text Search (FTS) and optional external enrichment
- Weather data via the DMI API
- Observability with Prometheus and Grafana
- Health and readiness probes (`/healthz`, `/readyz`)
- OpenAPI / Swagger documentation
- Automated CI/CD (linting, testing, smoke tests, container build, deployment)

---

## Quick Start (Docker Compose - recommended)

Run via Docker Compose to mirror CI and production.

1. Install Docker with the Compose plugin.
2. Copy the example env file and fill required values:

```bash
cp .env.example .env
```

Required for Compose:

- `APP_IMAGE_TAG` (e.g. `latest` or a commit SHA)
- `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`
- `SESSION_KEY` (32+ bytes in prod)
- `DMI_API_KEY`
- `GF_SECURITY_ADMIN_USER`, `GF_SECURITY_ADMIN_PASSWORD`
- `GF_SERVER_DOMAIN` (use `localhost` for local dev)

Start the stack:

```bash
docker compose up --build
```

Stop the stack:

```bash
docker compose down
```

Exposed locally:

- App: http://localhost:8080
- Swagger UI: http://localhost:8080/swagger/index.html
- Grafana: http://localhost:3000/grafana/ (served under a subpath)

More run/deploy options are in `docs/How-to-run-the-server.md`.

---

## Running without Compose (advanced)

You can run the Go binary or a standalone container against an existing PostgreSQL instance. Provide either `DB_HOST` with `POSTGRES_*` vars or a full `DATABASE_URL`. See `docs/How-to-run-the-server.md` for commands and caveats.

---

## Configuration

### Core runtime

| Variable | Description |
| --- | --- |
| `PORT` | HTTP port (default `8080`) |
| `APP_ENV` | `dev` or `prod` (Compose sets `prod`) |
| `SESSION_KEY` | Secret used to sign session cookies (**32+ bytes in prod**) |
| `APP_IMAGE_TAG` | Docker image tag used by Compose |
| `DATABASE_URL` | Full PostgreSQL DSN (preferred for managed DBs/CI) |
| `DB_HOST` | DB host when composing a DSN from individual vars |
| `POSTGRES_USER` | DB user (default `devops`) |
| `POSTGRES_PASSWORD` | DB password |
| `POSTGRES_DB` | DB name (default `whoknows`) |
| `POSTGRES_PORT` | DB port (default `5432`) |
| `POSTGRES_SSLMODE` | TLS mode for Postgres (default `disable`; use `require`/`verify-full` in prod) |
| `DB_MAX_OPEN_CONNS` | Max open DB connections (default `10`) |
| `DB_MAX_IDLE_CONNS` | Max idle DB connections (default `10`) |
| `DB_CONN_MAX_LIFETIME` | Connection lifetime (default `30m`) |

### Feature toggles

| Variable | Description |
| --- | --- |
| `SEARCH_FTS` | Enable Full-Text Search (`1` to enable) |
| `EXTERNAL_SEARCH` | Enable external search enrichment (`1` to enable) |
| `WIKI_USER_AGENT` | User-Agent used for Wikipedia scraping |

### Weather (DMI)

| Variable | Description |
| --- | --- |
| `DMI_API_KEY` | Required API key for weather endpoint |
| `DMI_API_URL` | Override base URL (defaults to `https://dmigw.govcloud.dk`) |
| `DMI_HTTP_TIMEOUT` | HTTP timeout for the DMI client (default `20s`) |

### Grafana / monitoring

| Variable | Description |
| --- | --- |
| `GF_SECURITY_ADMIN_USER` | Grafana admin user (required by Compose) |
| `GF_SECURITY_ADMIN_PASSWORD` | Grafana admin password (required by Compose) |
| `GF_SERVER_DOMAIN` | Grafana domain; set to `localhost` for local dev |

---

## Database and migrations

- PostgreSQL is used at runtime; migrations run automatically on startup.
- Migration logic: `internal/migrate`
- SQL files: `migrations/`

---

## API and routes

### Pages

- `/` - search
- `/about`
- `/login`
- `/register`
- `/weather`

### API endpoints

- `POST /api/register`
- `POST /api/login`
- `POST /api/logout` (POST only)
- `GET /api/search?q=<term>&language=<en|da>`
- `GET /api/weather`

### Observability and diagnostics

- `GET /healthz` - liveness
- `GET /readyz` - readiness (checks DB)
- `GET /metrics` - Prometheus metrics
- `GET /swagger/index.html` - Swagger UI

---

## Swagger / OpenAPI

Generated with `swaggo/swag`.

Regenerate:

```bash
swag init -g cmd/server/main.go -o docs
```

Generated files:

- `docs/swagger.json`
- `docs/swagger.yaml`
- `docs/docs.go` (auto-generated; do not edit)

---

## Postman QA collection

`postman/WhoKnows.postman_collection.json`

`BASE_URL` defaults to `http://localhost:8080`; override for staging/production.

---

## Testing

```bash
go test ./...
```

- Tests run against in-memory SQLite for speed; no local Postgres is required for unit/integration tests.
- Runtime still uses PostgreSQL.

---

## CI/CD

GitHub Actions workflow: `.github/workflows/ci.yml`

- Triggers: `push` to `main`/`develop` and `pull_request` targeting `main`/`develop`
- Concurrency: `ci-${{ github.ref }}` with cancel-in-progress

Jobs:

- `build`: Go 1.24.x, Postgres service, `go vet`, `golangci-lint`, `go test -race` with coverage upload
- `hadolint`: Dockerfile lint
- `healthz`: build binary, run against Postgres service, execute `scripts/smoke.sh`
- `compose-integration`: build image tagged with SHA, write `.env`, `docker compose up` app+db, wait for `/readyz`, run smoke, tear down
- `docker-build`: on `main`, build and push `ghcr.io/gitdengas123456/devops-valgfag:latest`
- `deploy`: on `main`, SSH to VM, write locked-down `.env`, `docker compose pull && up -d`

CI note: the Postgres DSN uses `sslmode=disable` because the database runs inside the GitHub Actions network. Enable TLS (`require`/`verify-full`) in production deployments.

---

## Repository structure

```text
.github/            CI workflows
cmd/server/         Application entrypoint and router
handlers/           HTTP handlers
internal/           Shared packages (metrics, migrate, scraper, etc.)
migrations/         SQL migration files
monitoring/         Prometheus and Grafana configuration
postman/            Postman QA collection
templates/          HTML templates
static/             CSS / JS / assets
docs/               Swagger and runbook
scripts/            Helper scripts
tests/              Unit and integration tests (SQLite)
```

---

## Notes

- `SESSION_KEY` must be strong in production.
- Docker Compose sets `APP_ENV=prod` by default.
- Ensure port 8080 is open when deploying to a VM.
- Never commit real secrets to git.
