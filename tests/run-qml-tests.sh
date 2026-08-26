#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
runner="$(command -v qmltestrunner 2>/dev/null || true)"

if [[ -z "${runner}" ]]; then
  for candidate in \
    /usr/lib/qt6/bin/qmltestrunner \
    /usr/lib/qt5/bin/qmltestrunner; do
    if [[ -x "${candidate}" ]]; then
      runner="${candidate}"
      break
    fi
  done
fi

if [[ -z "${runner}" ]]; then
  echo "fail: qmltestrunner not found (install qt6-declarative)" >&2
  exit 1
fi

cd "${root}/tests/qml"
exec "${runner}" -input tst_filetree.qml \
  -import "${root}/qml/panel/filetree"
