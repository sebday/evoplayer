package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/sebday/evoplayer/server/jsonlog"
	"github.com/sebday/evoplayer/server/paths"
)

func CmdJSONLog(env paths.Env, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: evoplayer jsonlog <scrobble-recent|queue-up-next|merge-tracks>")
	}
	switch args[0] {
	case "scrobble-recent":
		if len(args) < 3 {
			return fmt.Errorf("usage: evoplayer jsonlog scrobble-recent <log> <limit>")
		}
		limit, _ := strconv.Atoi(args[2])
		rows, err := jsonlog.ScrobbleRecent(args[1], limit)
		if err != nil {
			return err
		}
		return printJSONRows(rows)
	case "queue-up-next":
		if len(args) < 4 {
			return fmt.Errorf("usage: evoplayer jsonlog queue-up-next <tracks.json> <current> <limit>")
		}
		limit, _ := strconv.Atoi(args[3])
		rows, err := jsonlog.QueueUpNext(args[1], args[2], limit)
		if err != nil {
			return err
		}
		return printJSONRows(rows)
	case "merge-tracks":
		if len(args) < 5 {
			return fmt.Errorf("usage: evoplayer jsonlog merge-tracks <base.json> <workdir> <count> <out>")
		}
		count, _ := strconv.Atoi(args[3])
		return jsonlog.MergeTrackCache(args[1], args[2], count, args[4])
	default:
		return fmt.Errorf("unknown jsonlog command: %s", args[0])
	}
}

func printJSONRows(rows any) error {
	b, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func tracksCacheDir(env paths.Env) string {
	return filepath.Join(env.CacheDir, "tracks")
}

func placementLogPath(env paths.Env) string {
	return filepath.Join(env.StateDir, "placement.jsonl")
}
