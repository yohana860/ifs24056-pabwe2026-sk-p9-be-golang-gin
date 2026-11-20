#!/usr/bin/env bash
# Gagal bila total statement coverage backend < 100%.
set -euo pipefail
TOTAL=$(go tool cover -func="${1:-coverage.out}" | tail -1 | awk '{print $3}')
echo "Total coverage: $TOTAL"
[ "$TOTAL" = "100.0%" ]
