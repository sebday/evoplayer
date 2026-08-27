# Evoplayer agent guide

## CLI

Public command: `evoplayer` (`~/.local/bin/evoplayer`)

Native Go commands (daemon IPC or direct library access):
- `serve`, `start`, `restart`, `stop`, `toggle`, `next`, `prev`, `seek`, `volume`, `shuffle`
- `load`, `status`, `open`, `queue append|play|extend|up-next` (`up-next` reads the daemon playback queue)
- `playback …`, `library …`, `browse`, `meta`, `genres`, `tracks`, `cache`, `find`, `scrobble`
- `history`, `config`, `tags`, `vinyl`, `placement`, `waveform`, `lastfm`, `jsonlog`
- `warm`, `download` (soundcloud likes sync; `download url <url>` for youtube via yt-dlp, or soundcloud), `import`, `stats`, `job`
- `art` (search, set, apply, clear, maintain, notify-cache), `sort`, `playlist`, `favorite`, `current`

Unknown subcommands return a usage error from the Go CLI.

## Daemon

Single runtime: `evoplayer serve` owns playback (ffmpeg+oto), queue, volume, NDJSON IPC at `$XDG_RUNTIME_DIR/evoplayer.sock`, MPRIS (`org.mpris.MediaPlayer2.evoplayer`), and async jobs (`library.import`, `library.cache`, `library.soundcloud.download`, `library.download`, `library.art.maintain`).

IPC protocol reference: [docs/ipc.md](docs/ipc.md) (`capabilities`, `queue_revision` / `if_revision`, `spectrum.get`, error codes).

`evoplayer status --json` returns the same player fields as `state.get`, including `queue_revision`, `waveform`, and `art` paths when known.

Viz NDJSON stream for scripts/Waybar:

```bash
evoplayer viz stream --fps 30
evoplayer viz get
```

The Omarchy monitor service (`seb.evoplayer`) runs `evoplayer start` on load if the socket is absent.

SoundCloud likes sync reads `oauth_token` from a logged-in Brave (then Chromium) cookie, else `pass show omarchy/soundcloud/oauth-token` (`EVOPLAYER_PASS_PREFIX`). It is not stored in music.toml.

### IPC tracing (playback stops / tab switch)

Set `EVOPLAYER_TRACE_IPC=1` before the daemon starts. Logs playback, queue, viz, and MPRIS actions to stderr (`evoplayer: ipc …`, `evoplayer: mpris …`). QML logs `[evoplayer-ipc]` lines when the same env var is set.

```bash
# restart daemon with tracing (stderr -> /tmp/evoplayer-serve.log)
bash scripts/trace-player-ipc.sh

# follow ipc lines in another terminal
bash scripts/trace-player-ipc.sh watch
```

Use `EVOPLAYER_TRACE_IPC=all` to also log `state.get`, `subscribe`, and library IPC.

## State, cache, and secrets

- State: `$XDG_STATE_HOME/evoplayer` (override: `EVO_PLAYER_MUSIC_STATE`)
- Cache: `$XDG_CACHE_HOME/evoplayer`
- Library index: `$XDG_CACHE_HOME/evoplayer/library.sqlite3`
- Last.fm credentials: `pass show omarchy/lastfm/api-key` (and `api-secret`, `session-key`)
- SoundCloud oauth: Brave cookie `oauth_token` on soundcloud.com, else `pass show omarchy/soundcloud/oauth-token`
- Pass prefix override: `EVOPLAYER_PASS_PREFIX` (default `omarchy`)

## Install (Omarchy shell)

```bash
bash scripts/link-into-omarchy
omarchy-shell shell rescanPlugins
```

Creates:
- `~/.config/omarchy/plugins/seb.evoplayer` → `omarchy-plugin/` (with `panel/` → `qml/panel/`)
- `~/.local/lib/evoplayer/evoplayer` → `.build/evoplayer`
- `~/.local/bin/evoplayer` → `~/.local/lib/evoplayer/evoplayer`
- enables `seb.evoplayer` in `~/.config/omarchy/shell.json` (plugins + bar center)

Also runs `scripts/migrate-evoshell-data.sh` to move legacy `evoshell/panel/player` state/cache into `evoplayer/` when present.

### Panel control

```bash
omarchy-shell shell summon seb.evoplayer '{}'
omarchy-shell shell toggle seb.evoplayer '{}'
omarchy-shell shell hide seb.evoplayer
omarchy-shell evoplayer toggle   # IpcHandler on monitor service
```

### Optional: menu + autostart

Add to `~/.config/omarchy/extensions/omarchy-menu.jsonc`:

```jsonc
"trigger.evoplayer": {
  "icon": "󰎈",
  "label": "Music player",
  "action": "omarchy-shell shell toggle seb.evoplayer '{}'"
}
```

Optional daemon autostart in `~/.config/hypr/autostart.lua` (service also starts the daemon):

```lua
o.launch_on_start("evoplayer serve")
```

## QML integration

Plugin id: `seb.evoplayer` (`omarchy-plugin/manifest.json`)

- `panel/Service.qml` — monitor service (socket IPC, daemon bootstrap, notifications)
- `panel/Player.qml` — summoned dashboard panel
- `BarWidget.qml` — bar now-playing widget

QML styling uses `qml/panel/compat/` (Theme/Util shim over `qs.Commons`). Dashboard transport must use the monitor service IPC (`ipcCall` / `ipcCallVoid`), not CLI subprocesses for hot paths.

### Experimental cliamp playback backend

Set `EVOPLAYER_BACKEND=cliamp` on the omarchy-shell process to route playback, transport, and live spectrum through [cliamp](https://github.com/bjarneo/cliamp) instead of the evoplayer Go daemon. Library IPC, scrobble, and warm jobs still use evoplayer (`evoplayer serve` is started alongside `cliamp --daemon`).

| Capability | `EVOPLAYER_BACKEND=evoplayer` (default) | `EVOPLAYER_BACKEND=cliamp` (spike) |
|------------|----------------------------------------|-------------------------------------|
| Playback / transport | evoplayer daemon + MPRIS | cliamp daemon (`~/.config/cliamp/cliamp.sock`) |
| Live viz | `viz.subscribe` (CAVA-style) | `cliamp visstream` bands |
| Static waveform peaks | ffmpeg cache in `$XDG_CACHE_HOME/evoplayer/waveforms/` | unchanged (evoplayer warm/CLI) |
| Library browse / filetree | evoplayer sqlite index | unchanged (evoplayer library IPC) |
| Scrobble / SoundCloud / art jobs | evoplayer | unchanged (evoplayer) |

Optional overrides: `EVOPLAYER_CLIAMP_BIN`, `EVOPLAYER_CLIAMP_SOCKET`.

**Non-goals for the cliamp spike:** replacing the library dashboard, static waveform generation inside cliamp, or dropping evoplayer MPRIS/scrobble integration.

```bash
EVOPLAYER_BACKEND=cliamp bash tests/test-cliamp-bridge-spike
```

## Tests

```bash
bash tests/run-all.sh
```

Individual suites:

```bash
mise exec -- go test ./...
mise exec -- go test -tags=integration ./tests/integration/...
bash tests/test-evoplayer-cli
bash tests/test-evoplayer-art
bash tests/test-omarchy-plugin          # manifest + compat static checks
bash tests/test-evoplayer-qml-panel     # player panel QML syntax
bash tests/test-evoplayer-qml-smoke     # headless quickshell load + toggle IPC
bash tests/test-evoplayer-errors        # enriched IPC broadcasts + QML/static checks
bash tests/run-qml-tests.sh
```

Integration tests use isolated temp dirs and `EVOPLAYER_SOCKET` so they do not conflict with a running daemon.

## Commit messages and branching

Use `type: imperative lowercase subject` — see [CONTRIB.md](CONTRIB.md) for commit format and branch naming (`master` + `<type>/<subject>` topic branches).
