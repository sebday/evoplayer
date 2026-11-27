package cli

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/sebday/evoplayer/server/jsonlog"
	"github.com/sebday/evoplayer/server/paths"
)

func CmdJSONLog(env paths.Env, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: evoplayer jsonlog <scrobble-recent|queue-up-next>")
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
		return printJSON(rows)
	case "queue-up-next":
		if len(args) < 4 {
			return fmt.Errorf("usage: evoplayer jsonlog queue-up-next <tracks.json> <current> <limit>")
		}
		limit, _ := strconv.Atoi(args[3])
		rows, err := jsonlog.QueueUpNext(args[1], args[2], limit)
		if err != nil {
			return err
		}
		return printJSON(rows)
	default:
		return fmt.Errorf("unknown jsonlog command: %s", args[0])
	}
}

func placementLogPath(env paths.Env) string {
	return filepath.Join(env.StateDir, "placement.jsonl")
}
