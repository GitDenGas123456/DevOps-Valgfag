PHONY: check fmt vet lint test build smoke docker
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

docker:
	@if [ -f Dockerfile ]; then \
	  if command -v hadolint >/dev/null 2>&1; then hadolint Dockerfile; else echo "hadolint missing — skipping"; fi; \
	  docker build -t whoknows-app:local . ; \
	fi