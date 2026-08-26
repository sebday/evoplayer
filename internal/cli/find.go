package cli

import (
	"fmt"

	"github.com/sebday/evoplayer/internal/library/find"
	"github.com/sebday/evoplayer/internal/paths"
)

func CmdFind(env paths.Env, args []string) error {
	mode := "search"
	query := ""
	jsonOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOut = true
		case "--artist":
			mode = "artist"
			if i+1 < len(args) {
				i++
				query = args[i]
			}
		case "--genre":
			mode = "genre"
			if i+1 < len(args) {
				i++
				query = args[i]
			}
		case "--year":
			mode = "year"
			if i+1 < len(args) {
				i++
				query = args[i]
			}
		case "--album":
			mode = "album"
			if i+1 < len(args) {
				i++
				query = args[i]
			}
		case "--label":
			mode = "label"
			if i+1 < len(args) {
				i++
				query = args[i]
			}
		default:
			if query == "" {
				query = args[i]
			}
		}
	}
	if query == "" {
		return fmt.Errorf("evoplayer: find query required")
	}
	if err := env.EnsureDirs(); err != nil {
		return err
	}
	items, err := find.Tracks(tracksCacheDir(env), mode, query)
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSONRows(items)
	}
	for _, item := range items {
		path, _ := item["path"].(string)
		if path != "" {
			fmt.Println(path)
		}
	}
	return nil
}
