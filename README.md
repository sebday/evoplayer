# Evoplayer

Inspired by Bjarne's terminal music player, I made a local player with just the features I want and use. Styled like btop, because btop looks awesome.

I used Soundcloud for years and ignored my mp3 collection. This TUI (and probably Quickshell) music player is so I listen to my local library again, update the album art and download new music from Youtube and Soundcloud into my library.

## Layout

```
cmd/evoplayer/     thin main (serve, tui, CLI)
server/            daemon, playback, library, ipc, cli
tui/               terminal player
gui/               quickshell dashboard (optional, not default)
scripts/           install (binary + desktop entry)
```

## Install

```bash
bash scripts/install
```

Puts `evoplayer` on `PATH` (`~/.local/bin/evoplayer` → `.build/evoplayer`).

## Usage

With no arguments, `evoplayer` opens the terminal player (and starts `serve` if the socket is missing). `evoplayer tui` is the same.

### Player and daemon

```bash
evoplayer                    # TUI (default)
evoplayer tui
evoplayer serve              # daemon only (playback, IPC, MPRIS, jobs)
evoplayer start              # start daemon if needed
evoplayer restart
evoplayer stop
evoplayer close              # stop playback
evoplayer version
```

### Transport

```bash
evoplayer toggle
evoplayer next
evoplayer prev
evoplayer seek <seconds>
evoplayer shuffle [on|off|toggle]
evoplayer volume <delta|set> [value]
evoplayer load <path> [--folder] [--json]
evoplayer status [--json]
evoplayer open [--json]
evoplayer playback <toggle|next|prev|stop|seek|volume|shuffle> …
```

### Queue

```bash
evoplayer queue append <path> [...]
evoplayer queue play <start-path> <path> [...]
evoplayer queue extend [--json]
evoplayer queue up-next [--limit N] [--json]
```

### Library

```bash
evoplayer browse <path> [--json] [--offset N] [--limit N]
evoplayer meta <path> [--json]
evoplayer genres [--json]
evoplayer tracks <genre> [--json]
evoplayer find <query> [--artist|--genre|--year|--album|--label <value>] [--json]
evoplayer cache [--force] [<genre>] [--prune-art] [--json]
evoplayer stats [--json]
evoplayer library browse|meta|import|cache|download …
```

### Downloads and jobs

```bash
evoplayer download [--import]                    # SoundCloud likes sync
evoplayer download <url> [--no-import]           # YouTube or SoundCloud URL
evoplayer job status|stop|cancel [--json]
```

### Playlists and favorites

```bash
evoplayer playlist [name] [--json] [--offset N] [--limit N]
evoplayer playlist create|rename|delete|star <name> …
evoplayer favorite <path>
evoplayer current [load|save|clear] [--json]
```

### Tags, art, and library tools

```bash
evoplayer tags read|standardize <path>
evoplayer tags sanitize|slugify <string>
evoplayer art search|set|apply|clear|maintain|notify-cache …
evoplayer sort <folder> [--json]
evoplayer vinyl by-label [root] [--execute]
evoplayer warm <path> [--json]
evoplayer warm --all [<folder>] [--json]
evoplayer warm --batch <paths...>
evoplayer placement log|undo-plan [--json] [--undoable] [--limit N]
```

### Scrobbling and history

```bash
evoplayer scrobble auth|token|nowplaying|submit|recent|touch …
evoplayer lastfm auth-session|scrobble-api|recording-mbid …
evoplayer history report [--json] [--week N] [--limit N]
evoplayer jsonlog scrobble-recent|queue-up-next|merge-tracks …
```

### Config and viz

```bash
evoplayer config get|set|toml-get|toml-set|toml-json|toml-prune-derived|read-root|skip-dirs|pick …
evoplayer viz apply|stream [--fps N]|get
```

### Examples

```bash
evoplayer download https://soundcloud.com/you/likes
evoplayer cache --force drum&bass
evoplayer viz stream --fps 30
```

## Library folders

The music library is a tree of folders. Set the root with `evoplayer config set paths.root /path/to/music` (stored as `[paths] root` in `~/.config/evoplayer/music.toml`). Genre tag aliases for import live in the same file under `[genre_aliases]` (canonical keys like `drumandbass`) and `[genres]` (folder names like `drum&bass`). If unset or that folder is missing, evoplayer uses `~/music` or `~/Music`.

A genre tag is only a hint for which of those folders to use. Untagged downloads stay in `.incoming` until you pick a folder; import then moves the file into that folder (`youtube/` or `mixes/` plus year for long mixes).
