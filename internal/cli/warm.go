package cli

import (
	"fmt"

	"github.com/sebday/evoplayer/internal/paths"
	"github.com/sebday/evoplayer/internal/warm"
)

func CmdWarm(env paths.Env, args []string) error {
	jsonOut := false
	doBatch := false
	path := ""
	var batchPaths []string
	for _, a := range args {
		switch a {
		case "--json":
			jsonOut = true
		case "--batch":
			doBatch = true
		default:
			if path == "" && !doBatch {
				path = a
			} else if doBatch || path != "" {
				batchPaths = append(batchPaths, a)
			}
		}
	}
	if doBatch {
		out, err := warm.BatchThumbs(env, batchPaths, warm.DefaultWorkers)
		if err != nil {
			return err
		}
		if jsonOut {
			return printJSON(out)
		}
		fmt.Printf("warmed %d thumbs\n", len(out))
		return nil
	}
	if path == "" {
		return fmt.Errorf("usage: evoplayer warm <path> [--json] | evoplayer warm --batch <paths...>")
	}
	out, err := warm.Track(env, path)
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSON(out)
	}
	fmt.Println(out.Art)
	return nil
}
