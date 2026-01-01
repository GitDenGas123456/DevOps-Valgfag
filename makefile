.PHONY: check fmt vet lint test build smoke docker verify-metrics grafana-ds-uid

PORT ?= 8080
LOG  ?= /tmp/whoknows.log
BASE_URL ?= http://127.0.0.1:$(PORT)
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

test:
	go test ./...

build:
	go build -o server ./cmd/server

# Start server locally, run scripts/smoke.sh, then stop server again.
smoke: build
	@set -e; \
	./server >"$(LOG)" 2>&1 & echo $$! > .app.pid; \
	trap 'kill $$(cat .app.pid) >/dev/null 2>&1 || true; rm -f .app.pid' EXIT; \
	sleep 2; \
	BASE_URL="$(BASE_URL)" ./scripts/smoke.sh

verify-metrics:
	@set -e; \
	echo "Checking server is up on /healthz and /readyz (BASE_URL=$(BASE_URL))..."; \
	curl -fsS --max-time 2 "$(BASE_URL)/healthz" >/dev/null; \
	curl -fsS --max-time 2 "$(BASE_URL)/readyz"  >/dev/null; \
	echo "Curl /search and /about to populate metrics..."; \
	curl -fsS --max-time 5 "$(BASE_URL)/search?q=abc" >/dev/null; \
	curl -fsS --max-time 5 "$(BASE_URL)/about" >/dev/null; \
	metrics="$$(curl -fsS --max-time 5 "$(BASE_URL)/metrics" | grep 'app_http_requests_total' || true)"; \
	echo "$$metrics" | grep -q 'path=\"/search\"' || { echo "FAIL: /search missing"; exit 1; }; \
	echo "$$metrics" | grep -q 'path=\"/about\"'  || { echo "FAIL: /about missing"; exit 1; }; \
	echo "OK: metrics verified ✅"

docker:
	@if [ -f Dockerfile ]; then \
		if command -v hadolint >/dev/null 2>&1; then hadolint Dockerfile; else echo "hadolint missing - skipping"; fi; \
		docker build -t whoknows-app:local . ; \
	fi