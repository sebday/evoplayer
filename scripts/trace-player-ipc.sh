#!/usr/bin/env bash
# Restart the player daemon with IPC tracing enabled.
#
# Usage:
#   bash scripts/trace-player-ipc.sh          # restart daemon, trace to stderr
#   bash scripts/trace-player-ipc.sh watch    # follow omarchy-shell journal for ipc lines
#
# Traces playback/queue/viz IPC on the daemon (stderr) and QML client (journal).
# Set EVOPLAYER_TRACE_IPC=all to include state.get/subscribe and library calls.

set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
exe="${EVOPLAYER_BIN:-${HOME}/.local/lib/evoplayer/evoplayer}"
state="${EVO_PLAYER_MUSIC_STATE:-${XDG_STATE_HOME:-$HOME/.local/state}/evoplayer}"
lock="${state}/daemon.lock"

if [[ "${1:-}" == "watch" ]]; then
  echo "watching journal for evoplayer ipc lines (Ctrl-C to stop)…" >&2
  exec journalctl --user -f _COMM=omarchy-shell --since "1 min ago" 2>/dev/null \
    | grep --line-buffered -E 'evoplayer: ipc|evoplayer: mpris|\[evoplayer-ipc\]'
fi

if [[ ! -x "$exe" ]]; then
  echo "missing evoplayer binary: $exe (run: bash scripts/link-into-omarchy)" >&2
  exit 1
fi

export EVOPLAYER_TRACE_IPC="${EVOPLAYER_TRACE_IPC:-1}"

stop_serve_pids() {
  local pid
  while read -r pid; do
    [[ -n "$pid" ]] || continue
    echo "stopping daemon pid=$pid…" >&2
    kill "$pid" 2>/dev/null || true
    for _ in $(seq 1 40); do
      kill -0 "$pid" 2>/dev/null || break
      sleep 0.05
    done
    kill -9 "$pid" 2>/dev/null || true
  done < <(pgrep -f "${exe} serve$" 2>/dev/null || true)
}

if [[ -f "$lock" ]]; then
  pid="$(tr -d ' \t\r\n' <"$lock" 2>/dev/null || true)"
  if [[ -n "${pid:-}" ]] && kill -0 "$pid" 2>/dev/null; then
    echo "stopping lock pid=$pid…" >&2
    kill "$pid" 2>/dev/null || true
    for _ in $(seq 1 40); do
      kill -0 "$pid" 2>/dev/null || break
      sleep 0.05
    done
    kill -9 "$pid" 2>/dev/null || true
  fi
  rm -f "$lock"
fi

stop_serve_pids

runtime="${XDG_RUNTIME_DIR:-/tmp}"
rm -f "${runtime}/evoplayer.sock"

echo "starting daemon with EVOPLAYER_TRACE_IPC=$EVOPLAYER_TRACE_IPC" >&2
echo "  daemon logs: journalctl --user -f _COMM=omarchy-shell | grep -E 'evoplayer: ipc|evoplayer: mpris'" >&2
echo "  qml logs:    journalctl --user -f _COMM=omarchy-shell | grep '\\[evoplayer-ipc\\]'" >&2
echo "  or run:      bash scripts/trace-player-ipc.sh watch" >&2

EVOPLAYER_TRACE_IPC="$EVOPLAYER_TRACE_IPC" "$exe" serve >>"${TMPDIR:-/tmp}/evoplayer-serve.log" 2>&1 &
serve_pid=$!
echo "daemon pid=$serve_pid (stderr -> ${TMPDIR:-/tmp}/evoplayer-serve.log)" >&2

for _ in $(seq 1 80); do
  if [[ -S "${runtime}/evoplayer.sock" ]]; then
    echo "socket ready: ${runtime}/evoplayer.sock" >&2
    exit 0
  fi
  if ! kill -0 "$serve_pid" 2>/dev/null; then
    echo "daemon exited; see ${TMPDIR:-/tmp}/evoplayer-serve.log" >&2
    tail -20 "${TMPDIR:-/tmp}/evoplayer-serve.log" >&2 || true
    exit 1
  fi
  sleep 0.05
done

echo "daemon did not create socket in time; see ${TMPDIR:-/tmp}/evoplayer-serve.log" >&2
exit 1
