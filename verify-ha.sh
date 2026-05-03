#!/usr/bin/env bash
set -euo pipefail

PORT="${PORT:-8080}"
URL="http://localhost:${PORT}"

echo "==> Starting stack..."
docker compose up -d
trap 'docker compose down' EXIT

echo "==> Waiting for nginx to come up..."
for i in {1..10}; do
    if curl -sf "$URL/healthz" >/dev/null; then break; fi
    sleep 1
done

echo "==> Initial health check..."
curl -sf "$URL/healthz" >/dev/null && echo "OK"

echo "==> Killing one instance via /chaos..."
curl -s -X POST "$URL/chaos" >/dev/null || true

echo "==> Verifying survivor responds within 2s..."
start=$(date +%s)
curl -sf --max-time 2 "$URL/healthz" >/dev/null
elapsed=$(($(date +%s) - start))
echo "OK (recovered in ${elapsed}s)"

echo "==> HA verification PASSED"