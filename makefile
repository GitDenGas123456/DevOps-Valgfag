PHONY: check fmt vet lint test build smoke docker verify-metrics
PORT ?= 8080
LOG   ?= /tmp/whoknows.log

check: fmt vet lint test build smoke docker

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
	  golangci-lint run ./...; \
	else \
	  echo "golangci-lint not found - skipping (install: https://golangci-lint.run/)"; \
	fi

test:
	go test -race -cover ./...

build:
	go build ./cmd/server

smoke:
	@SEED_ON_BOOT=0 PORT=$(PORT) go run ./cmd/server > $(LOG) 2>&1 & echo $$! > .app.pid; \
	sleep 1; \
	URL="http://127.0.0.1:$(PORT)/healthz" scripts/smoke.sh; \
	kill `cat .app.pid` >/dev/null 2>&1 || true; rm -f .app.pid

verify-metrics:
	@echo "Curl /search and / to populate metrics (PORT=$(PORT))"
	@curl -s "http://127.0.0.1:$(PORT)/search?q=abc" >/dev/null
	@curl -s "http://127.0.0.1:$(PORT)/?q=abc" >/dev/null
	@echo "Expect both path=\"/search\" and path=\"/\" in app_http_requests_total:"
	@curl -s "http://127.0.0.1:$(PORT)/metrics" | grep 'app_http_requests_total' | grep -E 'path=\"/search\"|path=\"/\"'

docker:
	@if [ -f Dockerfile ]; then \
	  if command -v hadolint >/dev/null 2>&1; then hadolint Dockerfile; else echo "hadolint missing - skipping"; fi; \
	  docker build -t whoknows-app:local . ; \
	fi

# Legacy SQLite migration helpers (runtime now uses PostgreSQL)
DB ?= $(DATABASE_PATH)
DB := $(if $(DB),$(DB),data/seed/whoknows.db)

.PHONY: migrate.init migrate migrate.check

migrate.init:
	@mkdir -p migrations
	@command -v sqlite3 >/dev/null 2>&1 || { echo "sqlite3 not installed - skipping"; exit 0; }
	@sqlite3 "$(DB)" "CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY);"
	@echo "Init done -> $(DB)"

migrate: migrate.init
	@command -v sqlite3 >/dev/null 2>&1 || { echo "sqlite3 not installed - skipping"; exit 0; }
	@for f in $$(ls -1 migrations/*.sql 2>/dev/null | sort); do \
	  ver=$$(basename $$f .sql); \
	  applied=$$(sqlite3 "$(DB)" "SELECT 1 FROM schema_migrations WHERE version='$$ver' LIMIT 1;"); \
	  if [ "$$applied" != "1" ]; then \
	    echo "Applying $$ver ..."; \
	    sqlite3 "$(DB)" < "$$f" && sqlite3 "$(DB)" "INSERT INTO schema_migrations(version) VALUES('$$ver');" || { echo "FAIL $$ver failed"; exit 1; }; \
	  fi; \
	done; echo "OK. All migrations applied"

migrate.check: migrate.init
	@command -v sqlite3 >/dev/null 2>&1 || { echo "sqlite3 not installed - can't verify"; exit 0; }
	@pending=0; for f in $$(ls -1 migrations/*.sql 2>/dev/null | sort); do \
	  ver=$$(basename $$f .sql); \
	  applied=$$(sqlite3 "$(DB)" "SELECT 1 FROM schema_migrations WHERE version='$$ver' LIMIT 1;"); \
	  if [ "$$applied" != "1" ]; then echo "Pending: $$ver"; pending=1; fi; \
	done; test $$pending -eq 0 && echo "OK. No pending migrations" || (echo "FAIL Pending migrations"; exit 1)
