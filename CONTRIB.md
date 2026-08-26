# Contributing

Agents welcome :)

## Commit messages

```
type(scope): imperative lowercase subject
```

| Part    | Rule                                                       |
| ------- | ---------------------------------------------------------- |
| type    | `fix`, `feat`, `chore`, `docs`, `refactor`, or `test`      |
| scope   | `player`, `library`, `daemon`, `ui`, `test`, `docs` |
| subject | imperative, lowercase, no trailing period                  |

### Examples

```
fix(player): album art flicker on track change
feat(daemon): backend integration for playback
fix(library): tree view selection after filter
chore(daemon): normalize ipc socket path
docs(contrib): document commit message convention
```

Add a body only when the reason or breaking change is not obvious.

## Branching

Default branch: `master`. All pull requests target `master`.

Topic branches use GitHub Flow — one focused change per branch, deleted after merge.

```
<type>/<scope>-<subject>
```

| Part | Rule |
|------|------|
| type | same as commits: `fix`, `feat`, `chore`, `docs`, `refactor`, or `test` |
| slug | `<scope>-<subject>` in lowercase kebab-case (flat, no extra path segments) |

### Examples

```
feat/player-mpris-seek
fix/library-tree-selection
feat/daemon-ipc-socket-path
chore/test-integration-harness
docs/contrib-branch-convention
```

### Workflow

- Branch from `master`
- Keep one concern per branch
- First commit on the branch should match `type(scope): subject`
- Delete the topic branch after merge

Retired spikes may live under `archive/*` (historical bookmarks, not active development).
