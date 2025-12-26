# WhoKnows (Go) – DevOps Valgfag Project

WhoKnows is a lightweight Go web application built as part of the *DevOps Valgfag* course.  
The project demonstrates modern DevOps practices including containerization, CI/CD, observability, health checks, and infrastructure parity between development and production.

---

## Features

- Web pages: search, about, login, register, weather
- Session-based authentication (cookies + PostgreSQL)
- Search with optional Full-Text Search (FTS) and external enrichment
- Weather data via external API (DMI)
- Observability with Prometheus + Grafana
- Health and readiness probes (`/healthz`, `/readyz`)
- OpenAPI / Swagger documentation
- Full CI/CD pipeline (lint, tests, smoke tests, Docker build & deploy)

---

## Quick Start (Docker Compose – recommended)

This project is intended to be run via **Docker Compose**, which mirrors production most closely.

⚠️ **Important**  
Docker Compose **requires a `.env` file** with mandatory environment variables  
(Postgres credentials, session key, weather API key, Grafana credentials, etc.).

👉 **Read the full run & deployment guide here:**  
`docs/How-to-run-the-server.md`

Start locally:

```bash
docker compose up --build
```

Exposes:

- App: http://localhost:8080
- Swagger UI: http://localhost:8080/swagger/index.html
- Grafana: http://localhost:3000/grafana/

Stop:

```bash
docker compose down
```

---

## Running without Docker Compose (advanced)

You can also run the app directly with Go or with a standalone Docker container,  
but this **requires an existing PostgreSQL database** and additional configuration.

These setups are documented in detail in:

`docs/How-to-run-the-server.md`

---

## Configuration

### Core environment variables

| Variable | Description |
| --- | --- |
| `PORT` | HTTP port (default: `8080`) |
| `APP_ENV` | `dev` or `prod` |
| `SESSION_KEY` | Secret for signing session cookies (**32+ bytes in prod**) |
| `DATABASE_URL` | Full PostgreSQL DSN |
| `POSTGRES_USER` | Database user (Compose) |
| `POSTGRES_PASSWORD` | Database password |
| `POSTGRES_DB` | Database name |
| `APP_IMAGE_TAG` | Image tag used by Docker Compose |
| `DMI_API_KEY` | API key for weather endpoint |

### Optional / advanced configuration

| Variable | Description |
| --- | --- |
| `SEARCH_FTS` | Enable Full-Text Search |
| `EXTERNAL_SEARCH` | Enable external search enrichment |
| `WIKI_USER_AGENT` | User-Agent for Wikipedia scraping |
| `DB_MAX_OPEN_CONNS` | Max DB connections |
| `DB_MAX_IDLE_CONNS` | Idle DB connections |
| `DB_CONN_MAX_LIFETIME` | DB connection lifetime |
| `DMI_HTTP_TIMEOUT` | Weather API timeout |
| `DMI_API_URL` | Override weather API base URL |
| `GF_SERVER_DOMAIN` | Grafana domain (use `localhost` locally) |

---

## Database & migrations

- PostgreSQL is used in all runtime environments.
- Migrations are applied automatically on startup.
- Migration logic lives in `internal/migrate`.
- SQL migration files live in `migrations/`.

### Health checks

- `GET /healthz` – process is running
- `GET /readyz` – database connectivity OK

---

## API & routes

### Pages

- `/` – search
- `/about`
- `/login`
- `/register`
- `/weather`

### API endpoints

- `POST /api/register`
- `POST /api/login`
- `POST /api/logout` (POST-only)
- `GET /api/search?q=<term>&language=<en|da>`
- `GET /api/weather`

### Observability

- `GET /metrics` – Prometheus metrics
- `GET /swagger/index.html` – Swagger UI

---

## Swagger / OpenAPI

Swagger documentation is generated using `swaggo/swag`.

Regenerate docs:

```bash
swag init -g cmd/server/main.go -o docs
```

Generated files:

- `docs/swagger.json`
- `docs/swagger.yaml`
- `docs/docs.go` (auto-generated – do not edit manually)

---

## Observability stack

### Prometheus

- Scrapes `/metrics`
- Config: `monitoring/prometheus/prometheus.yml`

### Grafana

- Dashboards auto-provisioned
- Served under `/grafana/`
- Credentials via environment variables:

```env
GF_SECURITY_ADMIN_USER=admin
GF_SECURITY_ADMIN_PASSWORD=admin
```

---

## Testing & CI

### Tests (local)

```bash
go test ./...
```

Notes:

- Tests use in-memory SQLite.
- Runtime uses PostgreSQL.
- This keeps tests fast and isolated.

### CI pipeline

GitHub Actions (`.github/workflows/ci.yml`) runs:

- `go vet`
- `golangci-lint`
- `go test -race`
- Dockerfile lint (hadolint)
- Smoke tests (`/healthz`, `/readyz`)
- Docker build & push (on `main`)
- VM deployment via SSH (on `main`)

---

## Repository structure

```text
.github/            CI workflows
cmd/server/         Application entrypoint & router
handlers/           HTTP handlers
internal/db/        DB helpers & schema
internal/migrate/   PostgreSQL migration logic
internal/scraper/   External data scrapers
internal/metrics/   Prometheus metrics
migrations/         SQL migration files
monitoring/         Prometheus & Grafana config
postman/            Postman QA collection
templates/          HTML templates
static/             CSS / JS / assets
tests/              Unit & integration tests (SQLite)
docs/               Swagger & documentation
scripts/            Helper scripts
```

---

## Notes

- SQLite is **only** used in tests.
- PostgreSQL is required at runtime.
- `SESSION_KEY` must be strong in production.
- Docker Compose runs the app with `APP_ENV=prod` by default.
- Never commit real secrets to git.

