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

echo "== bash error regression =="
bash tests/test-evoplayer-errors

echo "== omarchy plugin static =="
bash tests/test-omarchy-plugin

echo "== qml panel syntax =="
bash tests/test-evoplayer-qml-panel

echo "== qml quickshell smoke =="
bash tests/test-evoplayer-qml-smoke

echo "== qml tests =="
bash tests/run-qml-tests.sh

echo "all tests passed"
