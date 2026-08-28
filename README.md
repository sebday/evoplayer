# Evoplayer

Terminal music library and playback.

## Layout

```
cmd/evoplayer/     Go daemon + CLI + TUI
internal/          playback, library, ipc, tui
tests/
scripts/           install (binary + desktop entry)
```

## Install

```bash
bash scripts/install
```

Puts `evoplayer` on `PATH` (`~/.local/bin/evoplayer` → `.build/evoplayer`).

## Usage

```bash
evoplayer          # terminal player (starts serve if needed)
evoplayer serve    # backend only
```

See [AGENTS.md](AGENTS.md) for daemon IPC, state paths, and tracing.

## Library folders

The music library is a tree of folders, not a genre taxonomy. Top-level dirs under the music root (`Drum & Bass`, `House`, …) are where tracks are filed.

A genre tag is only a hint for which of those folders to use. Untagged downloads stay in `.incoming` until you pick a folder; import then moves the file into that folder (`youtube/` or `mixes/` plus year for long mixes).

## Dev

```bash
bash tests/test-evoplayer-cli
mise exec -- go test ./internal/tui/
```

See [CONTRIB.md](CONTRIB.md) for commit message and branching conventions.
