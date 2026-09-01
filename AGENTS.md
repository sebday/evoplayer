# Agent guidance

## Engineering principles

- Minimize diff scope; match existing package patterns and naming in the surrounding code.
- Search for existing implementations before adding helpers or abstractions.
- Build with `go build ./cmd/evoplayer` for validation.
- The daemon (`evoplayer serve`) is the single runtime for playback, IPC, and async jobs; heavy work (likes sync, URL downloads, import, cache) runs in supervised child processes.
- No secrets in the repo; credentials come from pass or browser cookies per project conventions.

## Log papercuts

When small, non-blocking repository friction occurs — a retried tool call, confusing setup step, flaky command, misleading error, or non-obvious gotcha — use the `papercuts` skill and append it to [.cursor/PAPERCUTS.md](.cursor/PAPERCUTS.md) in the moment. Continue the current task. Real bugs and tracked work are not papercuts, and sensitive data must never be logged.

Do not mine an entire session for papercuts or start a broad cleanup unless the user explicitly asks.

## Skills

Project skills live in `.cursor/skills/`.

| Skill | When to load |
|-------|--------------|
| `evoplayer-dev` | CLI, daemon, IPC, install, paths, tests, downloads/import (default repo context) |
| `papercuts` | Logging or reviewing repository friction |
| `create-repo-skill` | Adding or updating a skill |

## Commits

Use `Imperative lowercase subject`.
