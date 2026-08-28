#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${root}"

echo "== go test (unit) =="
mise exec -- go test ./...

echo "== go test (integration) =="
mise exec -- go test -tags=integration ./tests/integration/...

echo "== bash cli smoke =="
bash tests/test-evoplayer-cli

echo "== bash art smoke =="
bash tests/test-evoplayer-art

echo "all tests passed"
