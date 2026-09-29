package daemon

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sebday/evoplayer/server/config"
	"github.com/sebday/evoplayer/server/playback"
)

// eqPresetCount slots are saved as p1..p3 in [eq]; a nil slot is empty.
const eqPresetCount = 3

func (d *Daemon) applyEQConfig() error {
	cfg, err := loadEQConfig(d.Env.MusicConfig)
	if err != nil {
		return err
	}
	presets, err := loadEQPresets(d.Env.MusicConfig)
	if err != nil {
		return err
	}
	d.eqMu.Lock()
	d.eqPresets = presets
	d.eqMu.Unlock()
	d.Actor.SetEQ(cfg)
	return nil
}

// eqView expects eqMu held.
func (d *Daemon) eqView() map[string]any {
	cfg := d.Actor.EQConfig()
	presets := make([]any, eqPresetCount)
	for i, p := range d.eqPresets {
		if p != nil {
			presets[i] = eqPresetValues(*p)
		}
	}
	return map[string]any{
		"enabled": cfg.Enabled,
		"preamp":  cfg.Preamp,
		"bands":   eqBandsView(cfg),
		"presets": presets,
	}
}

func eqBandsView(cfg playback.EQConfig) []map[string]any {
	bands := make([]map[string]any, len(playback.EQFreqs))
	for i, hz := range playback.EQFreqs {
		bands[i] = map[string]any{"hz": hz, "gain": cfg.Gains[i]}
	}
	return bands
}

func (d *Daemon) eqGet() map[string]any {
	d.eqMu.Lock()
	defer d.eqMu.Unlock()
	return d.eqView()
}

// updateEQ applies and saves under eqMu so overlapping slider requests
// cannot save an older snapshot over a newer one.
func (d *Daemon) updateEQ(change func(playback.EQConfig) playback.EQConfig) (map[string]any, error) {
	d.eqMu.Lock()
	defer d.eqMu.Unlock()
	d.Actor.SetEQ(change(d.Actor.EQConfig()))
	if err := d.persistEQ(); err != nil {
		return nil, err
	}
	return d.eqView(), nil
}

func (d *Daemon) loadEQPreset(slot int) (map[string]any, error) {
	if slot < 1 || slot > eqPresetCount {
		return nil, fmt.Errorf("eq preset: slot must be 1-%d", eqPresetCount)
	}
	d.eqMu.Lock()
	p := d.eqPresets[slot-1]
	d.eqMu.Unlock()
	if p == nil {
		return nil, fmt.Errorf("eq preset %d is empty", slot)
	}
	return d.updateEQ(func(playback.EQConfig) playback.EQConfig {
		cfg := *p
		cfg.Enabled = true
		return cfg
	})
}

func (d *Daemon) saveEQPreset(slot int) (map[string]any, error) {
	if slot < 1 || slot > eqPresetCount {
		return nil, fmt.Errorf("eq preset: slot must be 1-%d", eqPresetCount)
	}
	d.eqMu.Lock()
	defer d.eqMu.Unlock()
	cfg := d.Actor.EQConfig()
	vals := eqPresetValues(cfg)
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = formatDB(v)
	}
	key := "p" + strconv.Itoa(slot)
	if err := config.Set(d.Env.MusicConfig, "eq", key, strings.Join(parts, ",")); err != nil {
		return nil, err
	}
	d.eqPresets[slot-1] = &cfg
	return d.eqView(), nil
}

// eqPresetValues is preamp followed by the ten band gains.
func eqPresetValues(cfg playback.EQConfig) []float64 {
	return append([]float64{cfg.Preamp}, cfg.Gains[:]...)
}

func loadEQPresets(path string) ([eqPresetCount]*playback.EQConfig, error) {
	var out [eqPresetCount]*playback.EQConfig
	for i := range out {
		raw, err := config.Get(path, "eq", "p"+strconv.Itoa(i+1), "")
		if err != nil {
			return out, err
		}
		parts := strings.Split(raw, ",")
		if len(parts) != 1+len(playback.EQFreqs) {
			continue
		}
		var cfg playback.EQConfig
		ok := true
		for j, part := range parts {
			v, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
			if err != nil {
				ok = false
				break
			}
			if j == 0 {
				cfg.Preamp = v
			} else {
				cfg.Gains[j-1] = v
			}
		}
		if ok {
			out[i] = &cfg
		}
	}
	return out, nil
}

func loadEQConfig(path string) (playback.EQConfig, error) {
	cfg := playback.EQConfig{Enabled: true}
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
	fields := [][2]string{
		{"enabled", strconv.FormatBool(cfg.Enabled)},
		{"preamp", formatDB(cfg.Preamp)},
	}
	for i, key := range playback.EQKeys {
		fields = append(fields, [2]string{key, formatDB(cfg.Gains[i])})
	}
	return config.SetAll(d.Env.MusicConfig, "eq", fields)
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
	bands, _ := patch["bands"].([]any)
	for i := 0; i < len(bands) && i < len(out.Gains); i++ {
		if n, ok := bands[i].(float64); ok {
			out.Gains[i] = n
		}
	}
	return out
}
