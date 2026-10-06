package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/soundcloud"
)

func CmdSoundCloud(env paths.Env, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: evoplayer soundcloud <search|refresh-art>")
	}
	switch args[0] {
	case "refresh-art":
		return cmdSoundcloudRefreshArt(env, args[1:])
	case "search":
	default:
		return fmt.Errorf("usage: evoplayer soundcloud <search|refresh-art>")
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

func cmdSoundcloudRefreshArt(env paths.Env, args []string) error {
	limit := 0
	for i := 0; i < len(args); i++ {
		if args[i] != "--limit" {
			return fmt.Errorf("usage: evoplayer soundcloud refresh-art [--limit N]")
		}
		if i+1 >= len(args) {
			return fmt.Errorf("usage: evoplayer soundcloud refresh-art [--limit N]")
		}
		n, err := strconv.Atoi(args[i+1])
		if err != nil || n < 0 {
			return fmt.Errorf("usage: evoplayer soundcloud refresh-art [--limit N]")
		}
		limit = n
		i++
	}
	stats, err := soundcloud.RefreshSharedDumpArt(context.Background(), env, limit)
	fmt.Printf("queued %d, updated %d, cleared %d, failed %d\n", stats.Queued, stats.Updated, stats.Cleared, stats.Failed)
	if err != nil {
		return err
	}
	if stats.Queued > 0 && stats.Updated == 0 && stats.Cleared == 0 && stats.Failed > 0 {
		return fmt.Errorf("soundcloud: artwork refresh failed for %d tracks", stats.Failed)
	}
	return nil
}
