# Evoplayer IPC

Unix socket NDJSON protocol used by the Omarchy panel, CLI, and external scripts.

Default socket: `$XDG_RUNTIME_DIR/evoplayer.sock` (override: `EVOPLAYER_SOCKET`).

## Requests and responses

Each line is one JSON object.

Request:

```json
{"id": 1, "method": "state.get"}
```

Success:

```json
{"id": 1, "ok": true, "data": { ... }}
```

Error:

```json
{"id": 1, "ok": false, "code": "conflict", "error": "queue revision conflict", "data": {"queue_revision": 12}}
```

Fire-and-forget (no response): omit `id` or set `id` to `0`.

## Error codes

| Code | Meaning |
|------|---------|
| `invalid_params` | Malformed request or params |
| `conflict` | Stale `if_revision` on a queue mutation |
| `not_found` | Track or resource missing |
| `unavailable` | Daemon busy (e.g. job already running) |
| `unknown_method` | Unrecognized `method` |

## Discovery

```json
{"id": 1, "method": "capabilities"}
```

Returns `methods`, `events`, and `error_codes`.

## Subscriptions

```json
{"id": 1, "method": "subscribe"}
```

Adds the client to the broadcast list. Then call `state.get` or rely on `state` events.

Optional follow-up for live visualization:

```json
{"id": 2, "method": "viz.subscribe"}
```

## Events

| Event | Payload |
|-------|---------|
| `state` | Player status (see below) |
| `viz` | `{ "levels": [ ... ], "sequence": N, "generation": N }` |
| `job` | Async job status |
| `warm` | `{ "path": "...", "art": true }` |

## Player status (`state.get` / `state` events)

Stable fields on the enriched status object:

| Field | Type | Description |
|-------|------|-------------|
| `state` | string | `playing`, `paused`, or `stopped` |
| `path` | string | Absolute track path |
| `title`, `artist`, `album`, `genre`, `year`, `label` | string | Metadata |
| `position`, `duration` | number | Seconds |
| `position_label`, `duration_label` | string | Formatted times |
| `volume` | int | 0–100 |
| `art` | string | Absolute path to album art cache file |
| `shuffle` | bool | Shuffle mode |
| `playlist_pos`, `playlist_count` | int | Queue index (1-based pos) and length |
| `queue_revision` | int | Monotonic queue version for `if_revision` guards |
| `liked` | bool | Favorite flag |

CLI equivalent: `evoplayer status --json`

## Queue mutations and revision guards

Queue-changing methods accept optional `if_revision`. When present, the daemon rejects stale UI actions:

```json
{"id": 3, "method": "queue.replace", "params": {"paths": ["..."], "start_path": "...", "if_revision": 5}}
```

On conflict:

```json
{"id": 3, "ok": false, "code": "conflict", "error": "queue revision conflict", "data": {"queue_revision": 7}}
```

Methods with `if_revision` support:

- `queue.replace` / `queue.load`
- `queue.play_path`
- `queue.play_current`
- `queue.append`
- `queue.append_folder`

Successful mutations may return `{ "queue_revision": N }` in `data`.

## Playback

| Method | Params |
|--------|--------|
| `playback.toggle` | |
| `playback.next` | |
| `playback.prev` | |
| `playback.stop` | |
| `playback.seek` | `{ "seconds": 42.5 }` |
| `playback.volume.set` | `{ "volume": 80 }` |
| `playback.volume.delta` | `{ "delta": 5 }` |
| `playback.shuffle` | `{ "on": true }` |

## Visualization

| Method | Description |
|--------|-------------|
| `viz.subscribe` | Enable analyzer tap + `viz` events |
| `viz.unsubscribe` | Disable when last client unsubscribes |
| `spectrum.get` | One-shot `{ "ok": true, "levels": [...], "sequence": N }` |
| `viz.config` / `viz.config.set` / `viz.config.apply` | CAVA-style analyzer settings |

CLI stream (NDJSON frames, cliamp-style):

```bash
evoplayer viz stream --fps 30
evoplayer viz get
```

## Library (selection)

| Method | Params |
|--------|--------|
| `library.meta` | `{ "path": "..." }` |
| `library.browse` | `{ "path": "..." }` |
| `library.warm` | `{ "path": "..." }` |
| `library.warm.batch` | `{ "paths": [], "workers": 8, "art": true }` |

See `capabilities` for the full method list.

## Example session

```bash
socat - UNIX-CONNECT:$XDG_RUNTIME_DIR/evoplayer.sock
{"id":1,"method":"capabilities"}
{"id":2,"method":"subscribe"}
{"id":3,"method":"state.get"}
{"id":4,"method":"playback.toggle"}
```
