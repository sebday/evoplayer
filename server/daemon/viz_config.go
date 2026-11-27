package daemon

import (
	"github.com/sebday/evoplayer/server/config"
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
