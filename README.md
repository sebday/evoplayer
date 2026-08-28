# Evoplayer

Inspired by Bjarne's cliamp, I made a local music player with just the features I want and use. Styled like btop, because btop looks awesome.

I used Soundcloud for years and ignored my mp3 collection. This TUI (and probably Quickshell) music player is so I listen to my local library again, update the album art and download new music from Youtube and Soundcloud into my library.

## Layout

```
cmd/evoplayer/     thin main (serve, tui, CLI)
server/            daemon, playback, library, ipc, cli
tui/               terminal player
gui/               Quickshell dashboard (optional, not default)
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

The music library is a tree of folders, not a genre taxonomy. Set the root with `evoplayer config set paths.root /path/to/music` (stored as `[paths] root` in `music.toml`). If unset or that folder is missing, evoplayer uses `~/music` when that folder exists, otherwise `~/Music`. Top-level dirs under the music root (`Drum & Bass`, `House`, …) are where tracks are filed.

A genre tag is only a hint for which of those folders to use. Untagged downloads stay in `.incoming` until you pick a folder; import then moves the file into that folder (`youtube/` or `mixes/` plus year for long mixes).

## Dev

```bash
go test ./...
```

See [CONTRIB.md](CONTRIB.md) for commit message and branching conventions.
