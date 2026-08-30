#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${root}"

if command -v go >/dev/null 2>&1; then
  go test ./...
elif command -v mise >/dev/null 2>&1; then
  mise exec -- go test ./...
else
  echo "fail: go toolchain required" >&2
  exit 1
fi
