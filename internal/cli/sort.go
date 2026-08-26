package cli

import (
	"fmt"
	"strings"

	"github.com/sebday/evoplayer/internal/library"
	"github.com/sebday/evoplayer/internal/paths"
)

func CmdSort(env paths.Env, args []string) error {
	jsonOut := false
	folder := ""
	for _, a := range args {
		switch a {
		case "--json":
			jsonOut = true
		default:
			if !strings.HasPrefix(a, "-") && folder == "" {
				folder = a
			}
		}
	}
	if folder == "" {
		return fmt.Errorf("usage: evoplayer sort <folder> [--json]")
	}
	rel := strings.TrimPrefix(strings.TrimPrefix(folder, env.MusicRoot), "/")
	res, err := library.SortFolder(library.EnvFrom(env), rel)
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSON(res)
	}
	fmt.Printf("sorted %s: moved %d, failed %d\n", res.Folder, res.Moved, res.Failed)
	if res.Failed > 0 {
		return fmt.Errorf("sort had failures")
	}
	return nil
}
