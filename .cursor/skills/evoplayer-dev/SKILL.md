---
name: evoplayer-dev
description: >-
  Evoplayer development reference — CLI commands, daemon IPC, install, paths,
  secrets, tests, and download/import architecture. Use when working on evoplayer,
  evoplayer serve, the TUI, SoundCloud sync, downloads, library import, music.toml,
  job status, warm workers, MPRIS, or anything under server/, tui/, or cmd/.
---

# Evoplayer development

## CLI

Public command: `evoplayer` (`~/.local/bin/evoplayer`)

With no arguments, `evoplayer` opens the terminal player (and starts `serve` if the socket is missing). `evoplayer tui` is an alias.

Native Go commands (daemon IPC or direct library access):
- `serve`, `tui`, `start`, `restart`, `stop`, `close`, `toggle`, `next`, `prev`, `seek`, `volume`, `shuffle`
- `load`, `status`, `open`, `queue append|play|extend|up-next` (`up-next` reads the daemon playback queue)
- `playback …`, `library …`, `browse`, `meta`, `genres`, `tracks`, `cache`, `find`, `scrobble`
- `history`, `config`, `tags`, `vinyl`, `placement`, `lastfm`, `jsonlog`
- `warm`, `download` (soundcloud likes sync; `download url <url>` for youtube via yt-dlp, or soundcloud), `stats`, `job`
- `art` (search, set, apply, clear, notify-cache), `sort`, `playlist`, `favorite`, `current`
- `viz` (`stream`, `get`)

Unknown subcommands return a usage error from the Go CLI.

The Quickshell dashboard lives in `gui/` (optional; install stays TUI-only). Default `evoplayer` is the terminal player.

## Daemon

Single runtime: `evoplayer serve` owns playback (ffmpeg+oto), queue, volume, NDJSON IPC at `$XDG_RUNTIME_DIR/evoplayer.sock`, MPRIS (`org.mpris.MediaPlayer2.evoplayer`), and async jobs (`library.import`, `library.cache`, `library.soundcloud.download`, `library.download`, `library.art.maintain`). SoundCloud likes sync, URL downloads, `.incoming` import, and user-triggered cache run in supervised deprioritized child processes (`evoplayer _job soundcloud-download`, `evoplayer _job download-url`, `evoplayer _job import-incoming`, `evoplayer _job cache`); the daemon relays NDJSON progress to `job.status` and pauses warm workers while those jobs run.

IPC protocol reference: [docs/ipc.md](../../../docs/ipc.md) (`capabilities`, `queue_revision` / `if_revision`, `spectrum.get`, error codes).

`evoplayer status --json` returns the same player fields as `state.get`, including `queue_revision` and `art` paths when known.

Viz NDJSON stream for scripts/Waybar:

```bash
evoplayer viz stream --fps 30
evoplayer viz get
```

SoundCloud likes sync reads `oauth_token` from a logged-in Brave (then Chromium) cookie, else `pass show omarchy/soundcloud/oauth-token` (`EVOPLAYER_PASS_PREFIX`). It is not stored in music.toml.

### IPC tracing (playback stops)

Set `EVOPLAYER_TRACE_IPC=1` before the daemon starts. Logs playback, queue, viz, and MPRIS actions to stderr (`evoplayer: ipc …`, `evoplayer: mpris …`).

```bash
# restart daemon with tracing (stderr -> /tmp/evoplayer-serve.log)
bash scripts/trace-player-ipc.sh

# follow ipc lines in another terminal
bash scripts/trace-player-ipc.sh watch
```

Use `EVOPLAYER_TRACE_IPC=all` to also log `state.get`, `subscribe`, and library IPC.

## Architecture (downloads and import)

- Downloads land in `{music_root}/.incoming/` first.
- Import moves matched tracks into `{genre}/soundcloud/` or `{genre}/mixes/{year}/` based on embedded SoundCloud genre/tags matched against existing library genre folders (`library.MatchLibraryGenre`). There is no default genre fallback — unmatched files stay in `.incoming`.
- `sync-archive.txt` records handled SoundCloud IDs (successes, DRM skips, etc.). Archive presence does not mean the file is in the library.
- Heavy work runs in supervised child processes (`_job soundcloud-download`, `_job download-url`, `_job import-incoming`, `_job cache`); poll with `evoplayer job status --json` (uses `DaemonUp` only — does not restart the daemon). A pasted SoundCloud likes URL uses the same likes-sync worker as `evoplayer download`.

## State, cache, and secrets

- Music library: `[paths] root` in `$XDG_STATE_HOME/evoplayer/music.toml` (`evoplayer config set paths.root /path`). If unset or the folder is missing, `~/music` when that folder exists, else `~/Music`, else `~/music`.
- State: `$XDG_STATE_HOME/evoplayer` (override: `EVO_PLAYER_MUSIC_STATE`)
- Cache: `$XDG_CACHE_HOME/evoplayer`
- Library index: `$XDG_CACHE_HOME/evoplayer/library.sqlite3`
- Last.fm credentials: `pass show omarchy/lastfm/api-key` (and `api-secret`, `session-key`)
- SoundCloud oauth: Brave cookie `oauth_token` on soundcloud.com, else `pass show omarchy/soundcloud/oauth-token`
- Pass prefix override: `EVOPLAYER_PASS_PREFIX` (default `omarchy`)

## Install

```bash
bash scripts/link-into-omarchy
```

Creates:
- `~/.local/lib/evoplayer/evoplayer` → `.build/evoplayer`
- `~/.local/bin/evoplayer` → `~/.local/lib/evoplayer/evoplayer`
- `~/.local/share/applications/evoplayer.desktop` (opens the TUI)
- removes a leftover `evo.evoplayer` Quickshell plugin if present

Optional daemon autostart in `~/.config/hypr/autostart.lua`:

```lua
o.launch_on_start("evoplayer serve")
```
