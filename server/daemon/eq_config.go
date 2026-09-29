package daemon

import (
	"strconv"
	"strings"

	"github.com/sebday/evoplayer/server/config"
	"github.com/sebday/evoplayer/server/playback"
)

func (d *Daemon) applyEQConfig() error {
	cfg, err := loadEQConfig(d.Env.MusicConfig)
	if err != nil {
		return err
	}
	d.Actor.SetEQ(cfg)
	return nil
}

func (d *Daemon) eqView() map[string]any {
	return eqView(d.Actor.EQConfig())
}

func eqView(cfg playback.EQConfig) map[string]any {
	bands := make([]map[string]any, len(playback.EQFreqs))
	for i, hz := range playback.EQFreqs {
		bands[i] = map[string]any{"hz": hz, "gain": cfg.Gains[i]}
	}
	return map[string]any{
		"enabled": cfg.Enabled,
		"preamp":  cfg.Preamp,
		"bands":   bands,
	}
}

func loadEQConfig(path string) (playback.EQConfig, error) {
	cfg := playback.FlatEQ()
	enabled, err := config.Get(path, "eq", "enabled", "true")
	if err != nil {
		return cfg, err
	}
	cfg.Enabled = strings.TrimSpace(enabled) != "false" && strings.TrimSpace(enabled) != "0"
	pre, err := config.Get(path, "eq", "preamp", "0")
	if err != nil {
		return cfg, err
	}
	if v, err := strconv.ParseFloat(strings.TrimSpace(pre), 64); err == nil {
		cfg.Preamp = v
	}
	for i, key := range playback.EQKeys {
		raw, err := config.Get(path, "eq", key, "0")
		if err != nil {
			return cfg, err
		}
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil {
			cfg.Gains[i] = v
		}
	}
	return cfg, nil
}

func (d *Daemon) persistEQ() error {
	cfg := d.Actor.EQConfig()
	on := "false"
	if cfg.Enabled {
		on = "true"
	}
	fields := [][2]string{
		{"enabled", on},
		{"preamp", formatDB(cfg.Preamp)},
	}
	for i, key := range playback.EQKeys {
		fields = append(fields, [2]string{key, formatDB(cfg.Gains[i])})
	}
	for _, f := range fields {
		if err := config.Set(d.Env.MusicConfig, "eq", f[0], f[1]); err != nil {
			return err
		}
	}
	return nil
}

func formatDB(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func patchEQ(base playback.EQConfig, patch map[string]any) playback.EQConfig {
	out := base
	if v, ok := patch["enabled"].(bool); ok {
		out.Enabled = v
	}
	if v, ok := config.FloatFromMap(patch, "preamp"); ok {
		out.Preamp = v
	}
	raw, ok := patch["bands"]
	if !ok || raw == nil {
		return out
	}
	switch bands := raw.(type) {
	case []any:
		for i := 0; i < len(bands) && i < len(out.Gains); i++ {
			switch n := bands[i].(type) {
			case float64:
				out.Gains[i] = n
			case int:
				out.Gains[i] = float64(n)
			}
		}
	}
	return out
}
