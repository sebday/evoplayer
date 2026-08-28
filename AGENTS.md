# Evoplayer agent guide

## CLI

Public command: `evoplayer` (`~/.local/bin/evoplayer`)

With no arguments, `evoplayer` opens the terminal player (and starts `serve` if the socket is missing). `evoplayer tui` is an alias.

Native Go commands (daemon IPC or direct library access):
- `serve`, `tui`, `start`, `restart`, `stop`, `toggle`, `next`, `prev`, `seek`, `volume`, `shuffle`
- `load`, `status`, `open`, `queue append|play|extend|up-next` (`up-next` reads the daemon playback queue)
- `playback …`, `library …`, `browse`, `meta`, `genres`, `tracks`, `cache`, `find`, `scrobble`
- `history`, `config`, `tags`, `vinyl`, `placement`, `lastfm`, `jsonlog`
- `warm`, `download` (soundcloud likes sync; `download url <url>` for youtube via yt-dlp, or soundcloud), `import`, `stats`, `job`
- `art` (search, set, apply, clear, notify-cache), `sort`, `playlist`, `favorite`, `current`

Unknown subcommands return a usage error from the Go CLI.

The Quickshell dashboard lives in `gui/` (optional; install stays TUI-only). Default `evoplayer` is the terminal player.

## Daemon

Single runtime: `evoplayer serve` owns playback (ffmpeg+oto), queue, volume, NDJSON IPC at `$XDG_RUNTIME_DIR/evoplayer.sock`, MPRIS (`org.mpris.MediaPlayer2.evoplayer`), and async jobs (`library.import`, `library.cache`, `library.soundcloud.download`, `library.download`, `library.art.maintain`).

IPC protocol reference: [docs/ipc.md](docs/ipc.md) (`capabilities`, `queue_revision` / `if_revision`, `spectrum.get`, error codes).

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

## Tests

```bash
go test ./...
```

## Commit messages and branching

Use `type: imperative lowercase subject` — see [CONTRIB.md](CONTRIB.md) for commit format and branch naming (`master` + `<type>/<subject>` topic branches). Retired spikes live under `archive/*`.
