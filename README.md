# WhoKnows (Go) - DevOps-valgfag

Go webapp med search/about/login/register/weather sider, sessionbaseret auth, SQLite med seed og valgfri FTS5, API til auth/search og observability endpoints (healthz, metrics, Swagger).

## Kom i gang
- Krav: Go 1.25+, Docker til compose, sqlite3 CLI (valgfrit).
- Lokal koersel: `SEED_ON_BOOT=1 PORT=8080 go run ./cmd/server` opretter `data/seed/whoknows.db` hvis den mangler.
- Docker: `docker compose up -d` bruger GHCR-imagen og eksponerer app (8080), Prometheus (9090) og Grafana (3000). Byg selv: `docker build -t whoknows-app:local .` og koer via compose eller `docker run -p 8080:8080 -v ${PWD}/data:/app/data whoknows-app:local`.

## Konfiguration (env)
- `PORT` (8080), `DATABASE_PATH` (`data/seed/whoknows.db`), `SEED_ON_BOOT=1`, `SESSION_KEY` (64-byte secret), `SEARCH_FTS=1` (kraever migration 0003), `DMI_API_KEY` til vejrdata.

## Data og migrationer
- Migrationer i `migrations/`; koer med `DATABASE_PATH=... make migrate` eller `scripts/migrate.sh data/seed/whoknows.db`.
- Seed sker automatisk hvis `users` mangler eller `SEED_ON_BOOT=1`.
- `migrations/0003_pages_fts.sql` aktiverer FTS5-soegning naar `SEARCH_FTS=1`.

## API og routes
- Pages: `/`, `/about`, `/login`, `/register`, `/weather`.
- API: `POST /api/register|login|logout`, `GET /api/search?q=<term>&language=<en|da>`.
- Observability: `/healthz`, `/metrics`, `/swagger/index.html`; metrics bl.a. `app_search_total`, `app_search_with_result_total`, `app_search_duration_seconds`.

## Observability stack
- Compose-scrape: `monitoring/prometheus/prometheus.yml` (scraper `whoknows-app:8080`); alternativ host-config: `prometheus.yml` (scraper host.docker.internal:8080 + node_exporter paa 9100).
- Grafana dashboard skabelon: `monitoring/grafana/search-monitoring-dashboard.json` (default login admin/admin).
- Monitoring-only stack: `docker-compose.monitoring.yml` (Prometheus + Grafana + node_exporter).

## Udvikling, test og CI
- Tests: `go test ./...`; fuld pakke: `make check` (fmt, vet, lint, race+cover tests, build, smoke, docker build).
- Smoke mod koerende app: `PORT=8080 make smoke` (via `scripts/smoke.sh`).
- CI: `.github/workflows/ci.yml` koerer vet/lint/tests/migrations-check/healthz/hadolint, bygger/pusher til GHCR og deployer via SSH; `.github/workflows/cron.yml` timer/manuel sanity-check; PR/issue-templates i `.github/`.

## Struktur (kort overblik)
- `.github/` workflows + PR/issue-templates; `cmd/server/` main + router/bootstrap; `handlers/` sider/auth/search/weather/health; `internal/db` schema + seed; `internal/metrics` Prometheus metrics.
- `templates/`, `static/` web assets; `data/` seed-db; `migrations/` SQL; `scripts/` helpers; `monitoring/` Prom/Grafana; `tests/` integration; `docs/` swagger/kursusnoter; `rewrite/` eksperimenter; `src/backend/templates` gammel prototype.
- Oevrige filer: `docker-compose.yml`, `docker-compose.monitoring.yml`, `Dockerfile`, `.env` eksempel, `makefile`, `graph.dot`/`graph.png`, `Taskfile.yml`, `KPI Report`, `LICENSE` (MIT), byggede binarier (`app`, `server`), `stash.patch`.
