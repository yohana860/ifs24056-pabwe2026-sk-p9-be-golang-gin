#!/usr/bin/env bash
# Build produksi: frontend React -> embed ke binary Go. Jalankan dari root repo backend.
# FE_DIR = folder repo frontend (default: ../ifs24056-pabwe2026-sk-p9-fe-reactjs)
set -euo pipefail
FE_DIR="${FE_DIR:-../ifs24056-pabwe2026-sk-p9-fe-reactjs}"
(cd "$FE_DIR" && npm ci && npm run build)
find internal/web/dist -mindepth 1 ! -name .gitkeep -delete
cp -r "$FE_DIR"/dist/. internal/web/dist/
mkdir -p bin
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/server ./cmd/server
echo "Binary siap: bin/server (jalankan: ./bin/server)"
