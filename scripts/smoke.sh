#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"

check() {
  local path="$1"
  local expected="${2:-}"

  echo "Checking ${BASE_URL}${path}..."
  out="$(curl -fsS --retry 30 --retry-delay 1 --retry-all-errors --max-time 2 "${BASE_URL}${path}")"

  if [ -n "${expected}" ]; then
    echo "${out}" | grep -q "^${expected}$"
  fi
}

# Liveness: process is up
check "/healthz" "ok"
echo "Health check passed ✅"

# Readiness: DB reachable (may fail if DB not ready yet, hence retry)
check "/readyz" "ready"
echo "Readiness check passed ✅"

# Metrics endpoint exists (Prometheus scrape target)
check "/metrics"
echo "Metrics endpoint reachable ✅"
