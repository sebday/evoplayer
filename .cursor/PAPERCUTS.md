# Papercuts

Small, non-blocking friction in the repository itself — the kind that will
waste the next contributor's time too. Log it in the moment; review and fix
entries in a separate, user-requested cleanup pass.

This is not a completed-work log, a bug tracker, or a place for the agent's own
sandbox/shell/network hiccups. Never include secrets, credentials, personal
data, or sensitive paths.

## Open

- `go build ./cmd/evoplayer` overwrites the tracked root `evoplayer` binary, leaving an unrelated binary diff after routine validation.
- Quickshell keeps panel QML from process start. Edits under `gui/omarchy-plugin` do not show until `omarchy-shell shell rescanPlugins` (reopening the panel is not enough).
- The MPRIS service under `gui/omarchy-plugin/media/Service.qml` stays on the already-loaded instance after a plugin file save and after `omarchy-shell shell rescanPlugins`. It picks up service changes only on `omarchy restart shell`.

## Resolved

Move fixed entries here, mark them checked, and append the resolving date or commit.
