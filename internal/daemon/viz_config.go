package daemon

import (
	"strconv"

	"github.com/sebday/evoplayer/internal/config"
	"github.com/sebday/evoplayer/internal/viz"
)

func (d *Daemon) applyVizConfig() error {
	cfg, err := config.VizConfig(d.Env.MusicConfig)
	if err != nil {
		return err
	}
	d.Actor.VizAnalyzer().ApplyConfig(cfg)
	return nil
}

func (d *Daemon) vizConfigView() map[string]any {
	cfg := d.Actor.VizAnalyzer().Config()
	return map[string]any{
		"sensitivity":     cfg.Sensitivity,
		"autosens":        cfg.Autosens,
		"noise_reduction": cfg.NoiseReduction,
		"monstercat":      cfg.Monstercat,
		"frame_rate":      cfg.FrameRate,
		"low_cutoff":      cfg.LowCutoff,
		"high_cutoff":     cfg.HighCutoff,
	}
}

func (d *Daemon) reloadVizFromFile() (map[string]any, error) {
	if err := d.applyVizConfig(); err != nil {
		return nil, err
	}
	return d.vizConfigView(), nil
}

func mergeVizPatch(base viz.Config, patch map[string]any) viz.Config {
	out := base
	if v, ok := config.IntFromMap(patch, "sensitivity"); ok {
		out.Sensitivity = v
	}
	if v, ok := config.IntFromMap(patch, "autosens"); ok {
		out.Autosens = v
	}
	if v, ok := config.IntFromMap(patch, "noise_reduction"); ok {
		out.NoiseReduction = v
	}
	if f, ok := config.FloatFromMap(patch, "monstercat"); ok {
		out.Monstercat = f
	}
	if v, ok := config.IntFromMap(patch, "frame_rate"); ok {
		out.FrameRate = v
	}
	if v, ok := config.IntFromMap(patch, "low_cutoff"); ok {
		out.LowCutoff = v
	}
	if v, ok := config.IntFromMap(patch, "high_cutoff"); ok {
		out.HighCutoff = v
	}
	return out
}

func (d *Daemon) persistVizConfig(cfg viz.Config) error {
	fields := map[string]string{
		"sensitivity":     strconv.Itoa(cfg.Sensitivity),
		"autosens":        strconv.Itoa(cfg.Autosens),
		"noise_reduction": strconv.Itoa(cfg.NoiseReduction),
		"monstercat":      strconv.FormatFloat(cfg.Monstercat, 'f', -1, 64),
		"frame_rate":      strconv.Itoa(cfg.FrameRate),
		"low_cutoff":      strconv.Itoa(cfg.LowCutoff),
		"high_cutoff":     strconv.Itoa(cfg.HighCutoff),
	}
	for key, val := range fields {
		if err := config.Set(d.Env.MusicConfig, "viz", key, val); err != nil {
			return err
		}
	}
	return nil
}
