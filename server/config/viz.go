package config

import (
	"strconv"
	"strings"

	"github.com/sebday/evoplayer/server/viz"
)

func IntFromMap(m map[string]any, key string) (int, bool) {
	val, ok := m[key]
	if !ok {
		return 0, false
	}
	switch v := val.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func FloatFromMap(m map[string]any, key string) (float64, bool) {
	val, ok := m[key]
	if !ok {
		return 0, false
	}
	switch v := val.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func VizConfig(path string) (viz.Config, error) {
	out := viz.DefaultConfig()
	data, err := Load(path)
	if err != nil {
		return out, err
	}
	sec := data["viz"]
	if sec == nil {
		return out, nil
	}
	if v, ok := IntFromMap(sec, "sensitivity"); ok {
		out.Sensitivity = v
	}
	if v, ok := IntFromMap(sec, "autosens"); ok {
		out.Autosens = v
	}
	if v, ok := IntFromMap(sec, "noise_reduction"); ok {
		out.NoiseReduction = v
	}
	if v, ok := FloatFromMap(sec, "monstercat"); ok {
		out.Monstercat = v
	}
	if v, ok := IntFromMap(sec, "frame_rate"); ok {
		out.FrameRate = v
	}
	if v, ok := IntFromMap(sec, "low_cutoff"); ok {
		out.LowCutoff = v
	}
	if v, ok := IntFromMap(sec, "high_cutoff"); ok {
		out.HighCutoff = v
	}
	return out, nil
}

func VizJSON(path string) (VizView, error) {
	cfg, err := VizConfig(path)
	if err != nil {
		return VizView{}, err
	}
	return VizView{
		Sensitivity:    cfg.Sensitivity,
		Autosens:       cfg.Autosens,
		NoiseReduction: cfg.NoiseReduction,
		Monstercat:     cfg.Monstercat,
		FrameRate:      cfg.FrameRate,
		LowCutoff:      cfg.LowCutoff,
		HighCutoff:     cfg.HighCutoff,
	}, nil
}
