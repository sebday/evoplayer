# Evoplayer

Music library and playback for Omarchy shell.

[![Evoplayer player panel](docs/screenshots/player.png)](docs/screenshots/player.png)

## Layout

```
cmd/evoplayer/     Go daemon + CLI (playback, library, config, history, tags)
qml/panel/         dashboard + monitor service QML
omarchy-plugin/    Omarchy manifest, bar widget, install symlink target
tests/
scripts/           install + link into omarchy-shell
```

## Install (Omarchy)

```bash
bash scripts/install
omarchy-shell shell rescanPlugins
```

Defaults:
- Plugin → `~/.config/omarchy/plugins/seb.evoplayer`
- QML → `qml/panel/` (via plugin `panel/` symlink)
- Runtime → `~/.local/bin/evoplayer` (symlink to `~/.local/lib/evoplayer/evoplayer`)

Open the player:

```bash
omarchy-shell shell toggle seb.evoplayer '{}'
```

See [AGENTS.md](AGENTS.md) for menu entry, autostart, state paths, and IPC tracing.

## Dev

```bash
bash tests/test-omarchy-plugin
bash tests/test-evoplayer-cli
bash tests/test-evoplayer-qml-smoke
omarchy restart shell
```

See [CONTRIB.md](CONTRIB.md) for commit message and branching conventions.
