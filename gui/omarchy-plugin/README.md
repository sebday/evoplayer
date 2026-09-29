# evo.player

Omarchy shell plugin `evo.player`. The bar icon is the volume control. Left click opens now playing and recent scrobbles. While Brave is playing, that title is shown beside the icon.

This plugin replaces the built-in media service (`clonedFrom: omarchy.media`), so `omarchy-shell media playPause` hits evoplayer.

## Removing

```bash
omarchy plugin remove evo.player
```

That deletes the plugin directory. It does not delete:

- evoplayer's own socket under `$XDG_RUNTIME_DIR`

Network: none in the bar plugin (the player process may network on its own).
