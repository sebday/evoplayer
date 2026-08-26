#!/usr/bin/env bash
# One-shot migration from legacy evoshell panel/player paths to omarchy evoplayer paths.
set -euo pipefail

state_home="${XDG_STATE_HOME:-$HOME/.local/state}"
cache_home="${XDG_CACHE_HOME:-$HOME/.cache}"

legacy_state="${state_home}/evoshell/panel/player"
legacy_cache="${cache_home}/evoshell/panel/player"
target_state="${state_home}/evoplayer"
target_cache="${cache_home}/evoplayer"

migrate_dir() {
  local legacy="$1"
  local target="$2"
  local label="$3"

  if [[ ! -e "$legacy" ]]; then
    return 0
  fi
  if [[ -e "$target" ]]; then
    if [[ -L "$target" && "$(readlink -f "$target")" == "$(readlink -f "$legacy")" ]]; then
      return 0
    fi
    echo "evoplayer migrate: ${label} target already exists (${target}); leaving legacy in place" >&2
    return 0
  fi
  mkdir -p "$(dirname "$target")"
  mv "$legacy" "$target"
  echo "evoplayer migrate: moved ${label} ${legacy} -> ${target}"
}

migrate_dir "$legacy_state" "$target_state" "state"
migrate_dir "$legacy_cache" "$target_cache" "cache"

mkdir -p "${target_state}/playlists" "${target_cache}/art" "${target_cache}/thumbs" "${target_cache}/waveforms" "${target_cache}/tracks"
