# Agent guidance

## Engineering principles

- Minimize diff scope; match existing package patterns and naming in the surrounding code.
- Search for existing implementations before adding helpers or abstractions.
- Build with `go build ./cmd/evoplayer` for validation.
- The daemon (`evoplayer serve`) is the single runtime for playback, IPC, and async jobs; heavy work (likes sync, URL downloads, import, cache) runs in supervised child processes.
- No secrets in the repo; credentials come from pass or browser cookies per project conventions.

## Commits

Use `Imperative lowercase subject`.
