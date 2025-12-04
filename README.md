# WhoKnows (Go) - DevOps Elective Project

A lightweight Go web application with:

- Pages: search / about / login / register / weather
- Session-based authentication (SQLite + cookies)
- Search with optional FTS5 and external Wikipedia enrichment
- Observability via Prometheus + Grafana
- Full CI/CD (lint, tests, migrations, smoke, deploy)

---

## Quick Start

### Requirements

- Go 1.25+
- Docker (for `docker compose`)
- `sqlite3` CLI (optional)

### Run locally

Creates the SQLite DB automatically if missing:

```bash
SEED_ON_BOOT=1 PORT=8080 go run ./cmd/server
```

DB path:

```text
data/seed/whoknows.db
```

### Run with Docker Compose

Uses the published image from GHCR:

```bash
docker compose up -d
```

Exposes:

- App: http://localhost:8080
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000

### Build your own image

```bash
docker build -t whoknows-app:local .
docker run -p 8080:8080 -v ${PWD}/data:/app/data whoknows-app:local
```

---

## Configuration (Environment Variables)

| Variable          | Description                                                  |
| ----------------- | ------------------------------------------------------------ |
| `PORT`            | HTTP port (default: `8080`)                                  |
| `DATABASE_PATH`   | SQLite path (e.g. `data/seed/whoknows.db`)                   |
| `SEED_ON_BOOT`    | `1` = seed DB automatically if tables are missing            |
| `SESSION_KEY`     | 64-byte secret for secure sessions                           |
| `SEARCH_FTS`      | `1` to enable FTS5 search (requires migration `0003`)        |
| `APP_ENV`         | Set to `prod` in production to disable dev admin seeding     |
| `WIKI_USER_AGENT` | Optional custom User-Agent for Wikipedia API                 |
| `DMI_API_KEY`     | API key for the weather page                                 |

---

## Data and Migrations

- SQL migrations live in `migrations/`.
- Run with `make`:

  ```bash
  DATABASE_PATH=data/seed/whoknows.db make migrate
  ```

- Or via script:

  ```bash
  scripts/migrate.sh data/seed/whoknows.db
  ```

Notes:

- Seeding runs automatically if `users` is missing or `SEED_ON_BOOT=1`.
- `migrations/0003_pages_fts.sql` creates the `pages_fts` virtual table used when `SEARCH_FTS=1`.

---

## API and Routes

### Pages

- `/` - Search
- `/about`
- `/login`
- `/register`
- `/weather`

### API

- `POST /api/register`
- `POST /api/login`
- `POST /api/logout` / `GET /api/logout`
- `GET /api/search?q=<term>&language=<en|da>`

### Observability

- `GET /healthz`
- `GET /metrics` (Prometheus)
- `GET /swagger/index.html`

Key metrics:

- `app_search_total`
- `app_search_with_result_total`
- `app_search_duration_seconds`

---

## Observability Stack

### Prometheus

Configured in `monitoring/prometheus/prometheus.yml`, scraping:

- `whoknows-app:8080`
- `node_exporter:9100`

### Grafana

Dashboard provisioned from `monitoring/grafana/search-monitoring-dashboard.json`.

Default login credentials for Grafana are set via environment variables in your Docker Compose configuration.

For development/demo, you can set credentials in a `.env` file or via environment variables:

```env
GF_SECURITY_ADMIN_USER=yourusername
GF_SECURITY_ADMIN_PASSWORD=yourpassword

---

## Development, Testing and CI

### Tests

```bash
go test ./...
```

### Full local check (if `make` is available)

```bash
make check
```

Includes:

- `go fmt`, `go vet`, `golangci-lint`
- Race + coverage tests
- Local build
- Smoke tests
- Docker build

### CI

`.github/workflows/ci.yml` runs:

- `go vet`, `golangci-lint`, `go test ./...`
- Migrations sanity-check
- `/healthz` smoke test
- Dockerfile lint
- Build and push to GHCR
- SSH deploy on `main`

---

## Repository Structure (overview)

```text
.github/           CI workflows, PR/issue templates
cmd/server/        Application bootstrap + router
handlers/          Page, auth, search, weather, health handlers
internal/db/       Schema, migrations, seeding helpers
internal/metrics/  Prometheus metrics (counters, histograms)
internal/scraper/  Wikipedia scraper
templates/         HTML templates
static/            CSS and assets
data/              Seeded SQLite DB
migrations/        SQL migrations
monitoring/        Prometheus and Grafana config
scripts/           Local helper scripts
tests/             Handler and integration tests
docs/              Swagger and course notes
rewrite/           Experiments
src/backend/       Older prototype templates
```
