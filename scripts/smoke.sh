#!/usr/bin/env bash
URL="http://127.0.0.1:8080/healthz"

echo "Checking $URL..."
if curl -fsS --retry 30 --retry-delay 1 --retry-all-errors "$URL" | grep -q '^ok$'; then
  echo "Health check passed ✅"
  exit 0
else
  echo "Health check failed ❌"
  exit 1
fi