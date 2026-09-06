# evo.evoplayer

Omarchy shell plugin `evo.evoplayer`.

## Removing

```bash
omarchy plugin remove evo.evoplayer
```

That deletes the plugin directory. It does not delete:

- evoplayer's own socket under `$XDG_RUNTIME_DIR`

Network: none in the bar plugin (the player process may network on its own).
