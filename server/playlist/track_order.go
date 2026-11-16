package playlist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sebday/evoplayer/server/library"
)

func trackOrderPath(env Env) string {
	return filepath.Join(env.StateDir, "playlist-track-order.json")
}

func loadTrackOrders(env Env) (map[string][]string, error) {
	raw, err := os.ReadFile(trackOrderPath(env))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string][]string{}, nil
		}
		return nil, err
	}
	var orders map[string][]string
	if err := json.Unmarshal(raw, &orders); err != nil {
		return map[string][]string{}, nil
	}
	if orders == nil {
		orders = map[string][]string{}
	}
	return orders, nil
}

// SetTrackOrder stores the dragged track order for one playlist.
func SetTrackOrder(env Env, name string, paths []string) error {
	name = strings.TrimSpace(name)
	if name == "" || strings.EqualFold(name, "current") {
		return fmt.Errorf("playlist name required")
	}
	if env.StateDir == "" {
		return os.ErrInvalid
	}
	paths = cleanPaths(paths)
	if len(paths) == 0 {
		return fmt.Errorf("paths required")
	}
	orders, err := loadTrackOrders(env)
	if err != nil {
		return err
	}
	orders[name] = paths
	if err := os.MkdirAll(env.StateDir, 0o755); err != nil {
		return err
	}
	raw, err := json.Marshal(orders)
	if err != nil {
		return err
	}
	return library.WriteFileAtomic(trackOrderPath(env), append(raw, '\n'), 0o644)
}

func orderedPaths(env Env, name string, paths []string) ([]string, error) {
	orders, err := loadTrackOrders(env)
	if err != nil {
		return nil, err
	}
	return applyPathOrder(paths, orders[name]), nil
}

func applyPathOrder(paths, saved []string) []string {
	if len(saved) == 0 || len(paths) < 2 {
		return paths
	}
	have := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		have[path] = struct{}{}
	}
	out := make([]string, 0, len(paths))
	used := map[string]struct{}{}
	for _, path := range saved {
		if _, ok := have[path]; !ok {
			continue
		}
		if _, seen := used[path]; seen {
			continue
		}
		used[path] = struct{}{}
		out = append(out, path)
	}
	for _, path := range paths {
		if _, seen := used[path]; seen {
			continue
		}
		out = append(out, path)
	}
	return out
}

func cleanPaths(paths []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out
}
