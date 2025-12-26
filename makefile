.PHONY: check fmt vet lint test test-race build smoke docker verify-metrics grafana-ds-uid

PORT ?= 8080
LOG  ?= /tmp/whoknows.log
GRAFANA_HOST ?= http://localhost:3000

check: fmt vet lint test build smoke docker

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./... ; \
	else \
		echo "golangci-lint missing - skipping"; \
	fi

# Lokal test (hurtig). CI kan køre race/coverage selv.
test:
	go test ./...

# Valgfri lokalt: race-run
test-race:
	go test -race ./...

build:
	go build -o server ./cmd/server

smoke: build
	@set -e; \
	./server >"$(LOG)" 2>&1 & echo $$! > .app.pid; \
	sleep 1; \
	echo "Smoke: GET /healthz"; \
	curl -fsS --max-time 2 "http://127.0.0.1:$(PORT)/healthz" | grep -qx "ok"; \
	echo "Smoke: GET /readyz"; \
	curl -fsS --max-time 2 "http://127.0.0.1:$(PORT)/readyz"  | grep -qx "ready"; \
	kill `cat .app.pid` >/dev/null 2>&1 || true; rm -f .app.pid

verify-metrics:
	@set -e; \
	echo "Checking server is up on /healthz and /readyz (PORT=$(PORT))..."; \
	curl -fsS --max-time 2 "http://127.0.0.1:$(PORT)/healthz" >/dev/null || { \
		echo "FAIL: server not responding on http://127.0.0.1:$(PORT)/healthz"; \
		echo "Start it first (e.g. make build && ./server)"; \
		exit 1; \
	}; \
	curl -fsS --max-time 2 "http://127.0.0.1:$(PORT)/readyz" >/dev/null || { \
		echo "FAIL: server not responding on http://127.0.0.1:$(PORT)/readyz"; \
		echo "Start it first (e.g. make build && ./server)"; \
		exit 1; \
	}; \
	echo "Curl /search and /about to populate metrics..."; \
	curl -fsS --max-time 5 "http://127.0.0.1:$(PORT)/search?q=abc" >/dev/null; \
	curl -fsS --max-time 5 "http://127.0.0.1:$(PORT)/about" >/dev/null; \
	echo "Expect both path=\"/search\" and path=\"/about\" in app_http_requests_total:"; \
	metrics="$$(curl -fsS --max-time 5 "http://127.0.0.1:$(PORT)/metrics" | grep 'app_http_requests_total' || true)"; \
	echo "$$metrics" | grep -q 'path=\"/search\"' && echo "OK: /search present" || { echo "FAIL: /search missing"; exit 1; }; \
	echo "$$metrics" | grep -q 'path=\"/about\"'  && echo "OK: /about present"  || { echo "FAIL: /about missing"; exit 1; }

docker:
	@if [ -f Dockerfile ]; then \
		if command -v hadolint >/dev/null 2>&1; then hadolint Dockerfile; else echo "hadolint missing - skipping"; fi; \
		docker build -t whoknows-app:local . ; \
	fi

grafana-ds-uid:
	@command -v curl >/dev/null 2>&1 || { echo "curl not installed"; exit 1; }
	@command -v jq   >/dev/null 2>&1 || { echo "jq not installed"; exit 1; }
	@set -e; \
	token="$(GRAFANA_API_TOKEN)"; \
	if [ -z "$$token" ] && [ -f .env ]; then \
		token=$$(grep -E '^GRAFANA_API_TOKEN=' .env | tail -1 | cut -d= -f2- | tr -d '\r'); \
	fi; \
	if [ -z "$$token" ]; then \
		echo "Set GRAFANA_API_TOKEN (env or .env) before running"; \
		exit 1; \
	fi; \
	echo "Grafana API: $(GRAFANA_HOST)/api/datasources"; \
	curl -fsS --max-time 10 -H "Authorization: Bearer $$token" "$(GRAFANA_HOST)/api/datasources" | \
		jq -r '.[].name + " => " + .uid'