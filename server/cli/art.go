package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/sebday/evoplayer/server/art"
	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
)

func CmdArt(env paths.Env, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: evoplayer art <search|set|apply|clear|maintain|notify-cache>")
	}
	libEnv := library.EnvFrom(env)
	jsonOut := false
	scope := "track"
	switch args[0] {
	case "search":
		path := ""
		query := ""
		for _, a := range args[1:] {
			switch a {
			case "--json":
				jsonOut = true
			case "--query":
				continue
			default:
				if strings.HasPrefix(a, "--") {
					continue
				}
				if path == "" && !strings.Contains(a, " ") {
					if st, err := os.Stat(a); err == nil && !st.IsDir() {
						path = a
						continue
					}
				}
				if query == "" {
					query = a
				}
			}
		}
		for i := 1; i < len(args); i++ {
			if args[i] == "--query" && i+1 < len(args) {
				query = args[i+1]
			}
		}
		var res art.SearchResponse
		var err error
		if query != "" {
			res = art.SearchQuery(query)
		} else if path != "" {
			res, err = art.SearchTrack(env, path)
		} else {
			return fmt.Errorf("usage: evoplayer art search <path> [--json] | --query <text>")
		}
		if err != nil {
			return err
		}
		if jsonOut {
			return printJSON(res)
		}
		for _, r := range res.Results {
			fmt.Printf("%s\t%s\n", r.Label, r.URL)
		}
		return nil
	case "set":
		path, image, err := parseArtPathImage(args[1:])
		if err != nil {
			return err
		}
		for _, a := range args[1:] {
			if a == "--album" {
				scope = "album"
			}
			if a == "--json" {
				jsonOut = true
			}
		}
		out, err := library.InstallImage(libEnv, path, image, scope)
		if err != nil {
			return err
		}
		if jsonOut {
			return printJSON(out)
		}
		fmt.Println(out.Art)
		return nil
	case "apply":
		path, url, err := parseArtPathURL(args[1:])
		if err != nil {
			return err
		}
		for _, a := range args[1:] {
			if a == "--album" {
				scope = "album"
			}
			if a == "--json" {
				jsonOut = true
			}
		}
		out, err := library.ApplyImageURL(libEnv, path, url, scope)
		if err != nil {
			return err
		}
		if jsonOut {
			return printJSON(out)
		}
		fmt.Println(out.Art)
		return nil
	case "clear":
		path, err := parseArtPath(args[1:])
		if err != nil {
			return err
		}
		for _, a := range args[1:] {
			if a == "--json" {
				jsonOut = true
			}
		}
		if err := library.ClearArt(libEnv, path); err != nil {
			return err
		}
		if jsonOut {
			return printJSON(map[string]any{"ok": true, "path": path})
		}
		fmt.Printf("cleared %s\n", path)
		return nil
	case "maintain":
		return runLibraryJob(env, "library.art.maintain", nil)
	case "notify-cache":
		path, err := parseArtPath(args[1:])
		if err != nil {
			return err
		}
		out, err := art.NotifyCache(env, path)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	default:
		return fmt.Errorf("unknown art command: %s", args[0])
	}
}

func parseArtPath(args []string) (string, error) {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			return a, nil
		}
	}
	return "", fmt.Errorf("usage: path required")
}

func parseArtPathImage(args []string) (string, string, error) {
	var path, image string
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		if path == "" {
			path = a
		} else if image == "" {
			image = a
		}
	}
	if path == "" || image == "" {
		return "", "", fmt.Errorf("usage: evoplayer art set <track> <image>")
	}
	return path, image, nil
}

func parseArtPathURL(args []string) (string, string, error) {
	var path, u string
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		if path == "" {
			path = a
		} else if u == "" {
			u = a
		}
	}
	if path == "" || u == "" {
		return "", "", fmt.Errorf("usage: evoplayer art apply <track> <url>")
	}
	return path, u, nil
}
