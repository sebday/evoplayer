package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/soundcloud"
)

func CmdSoundCloud(env paths.Env, args []string) error {
	if len(args) == 0 || args[0] != "search" {
		return fmt.Errorf("usage: evoplayer soundcloud search [--json] <query>")
	}
	jsonOut := false
	parts := make([]string, 0, len(args))
	for _, arg := range args[1:] {
		if arg == "--json" {
			jsonOut = true
			continue
		}
		parts = append(parts, arg)
	}
	query := strings.TrimSpace(strings.Join(parts, " "))
	if query == "" {
		return fmt.Errorf("usage: evoplayer soundcloud search [--json] <query>")
	}
	tracks, err := soundcloud.Search(env, query, 12)
	if err != nil {
		return err
	}
	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"tracks": tracks})
	}
	for _, track := range tracks {
		fmt.Printf("%s — %s\n", track.Artist, track.Title)
	}
	return nil
}
