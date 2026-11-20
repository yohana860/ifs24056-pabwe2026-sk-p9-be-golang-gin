#!/usr/bin/env bash
# Smoke test binary: butuh PostgreSQL aktif sesuai .env / environment.
set -euo pipefail
PORT="${APP_PORT:-8081}"
./bin/server & PID=$!
trap 'kill $PID 2>/dev/null || true' EXIT
for i in $(seq 1 30); do curl -fs "http://localhost:$PORT/api.json" >/dev/null && break; sleep 1; done
curl -fs "http://localhost:$PORT/api.json" | grep -q '"swagger"'
curl -fs "http://localhost:$PORT/swagger/index.html" >/dev/null
curl -fs "http://localhost:$PORT/" | grep -q 'id="root"'
curl -fs "http://localhost:$PORT/author" | grep -q 'id="root"'
curl -fs "http://localhost:$PORT/admin" | grep -q 'id="root"'
CODE=$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:$PORT/api/tidak-ada")
[ "$CODE" = "404" ]
echo "Smoke test OK"
