#!/usr/bin/env bash
# Re-run evoplayer error checks until clean or max attempts (for local/agent loops).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${root}"

max="${1:-5}"
attempt=1

while ((attempt <= max)); do
  echo "== attempt ${attempt}/${max} =="
  if bash tests/test-evoplayer-errors; then
    echo "clean after ${attempt} attempt(s)"
    exit 0
  fi
  ((attempt++))
  sleep 1
done

echo "still failing after ${max} attempts — fix remaining issues and re-run" >&2
exit 1
