#!/usr/bin/env bash
# One-time fix: duration >= 25m is the only mix rule.
# 1. Undo today's sort moves from soundcloud -> mixes when duration < 25m
# 2. Sort any remaining 25m+ tracks from soundcloud into mixes/{year}/
set -euo pipefail

MIN_SEC=$((25 * 60))
GENRE="drum&bass"
EVO="${EVOPLAYER_BIN:-$(dirname "$0")/../.build/evoplayer}"
PLACEMENT="${XDG_STATE_HOME:-$HOME/.local/state}/evoplayer/placement.jsonl"

duration_sec() {
  ffprobe -v quiet -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 "$1" 2>/dev/null | awk '{printf "%.0f", $1}'
}

moved_back=0
if [[ -f "$PLACEMENT" ]]; then
  while IFS= read -r line; do
    [[ "$line" == *'"op":"sort"'* ]] || continue
    [[ "$line" == *'/soundcloud/'* ]] || continue
    [[ "$line" == *'/mixes/'* ]] || continue
    from=$(printf '%s' "$line" | sed -n 's/.*"from":"\([^"]*\)".*/\1/p' | sed 's/\\u0026/\&/g')
    to=$(printf '%s' "$line" | sed -n 's/.*"to":"\([^"]*\)".*/\1/p' | sed 's/\\u0026/\&/g')
    [[ -n "$from" && -n "$to" ]] || continue
    [[ -f "$to" ]] || continue
    dur=$(duration_sec "$to" || echo 0)
    if [[ "$dur" -ge "$MIN_SEC" ]]; then
      continue
    fi
    if [[ -e "$from" ]]; then
      echo "skip (from exists): $(basename "$from") (${dur}s)" >&2
      continue
    fi
    echo "back -> soundcloud: $(basename "$from") (${dur}s)"
    mv "$to" "$from"
    moved_back=$((moved_back + 1))
  done < "$PLACEMENT"
fi

echo "moved back to soundcloud: $moved_back"

if [[ ! -x "$EVO" ]]; then
  echo "building evoplayer..."
  (cd "$(dirname "$0")/.." && go build -o .build/evoplayer ./cmd/evoplayer)
fi

echo "sorting 25m+ tracks still in soundcloud..."
"$EVO" sort "$GENRE/soundcloud" --json
