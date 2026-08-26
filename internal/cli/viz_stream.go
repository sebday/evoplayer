package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sebday/evoplayer/internal/paths"
)

func cmdVizStream(env paths.Env, exe string, fps int) error {
	if fps <= 0 {
		fps = 30
	}
	if fps > 60 {
		fps = 60
	}
	if err := EnsureDaemon(env, exe); err != nil {
		return err
	}
	if _, err := IPC(env, "viz.subscribe", nil); err != nil {
		return err
	}
	defer func() { _, _ = IPC(env, "viz.unsubscribe", nil) }()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sig)

	ticker := time.NewTicker(time.Second / time.Duration(fps))
	defer ticker.Stop()

	for {
		select {
		case <-sig:
			return nil
		case <-ticker.C:
			resp, err := IPC(env, "spectrum.get", nil)
			if err != nil {
				return err
			}
			if !resp.OK {
				if resp.Code != "" {
					return fmt.Errorf("%s: %s", resp.Code, resp.Error)
				}
				return fmt.Errorf("%s", resp.Error)
			}
			data, ok := resp.Data.(map[string]any)
			if !ok {
				continue
			}
			levels, _ := data["levels"].([]any)
			if len(levels) == 0 {
				continue
			}
			if err := printJSON(map[string]any{"ok": true, "levels": levelsToFloat64(levels)}); err != nil {
				return err
			}
		}
	}
}

func cmdSpectrumGet(env paths.Env, exe string) error {
	if err := EnsureDaemon(env, exe); err != nil {
		return err
	}
	resp, err := IPC(env, "spectrum.get", nil)
	if err != nil {
		return err
	}
	if !resp.OK {
		if resp.Code != "" {
			return fmt.Errorf("%s: %s", resp.Code, resp.Error)
		}
		return fmt.Errorf("%s", resp.Error)
	}
	return printJSON(resp.Data)
}

func levelsToFloat64(raw []any) []float64 {
	out := make([]float64, len(raw))
	for i, v := range raw {
		switch n := v.(type) {
		case float64:
			out[i] = n
		case float32:
			out[i] = float64(n)
		}
	}
	return out
}
