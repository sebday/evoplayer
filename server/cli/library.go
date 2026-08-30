package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
)

func CmdMeta(env paths.Env, args []string) error {
	jsonOut := false
	path := ""
	for _, a := range args {
		if a == "--json" {
			jsonOut = true
		} else if !strings.HasPrefix(a, "-") {
			path = a
		}
	}
	if path == "" {
		return fmt.Errorf("usage: evoplayer meta <path> [--json]")
	}
	row, err := library.Meta(library.EnvFrom(env), path, readPlaylistName(env))
	if err != nil {
		if jsonOut {
			fmt.Println("{}")
		}
		return err
	}
	if jsonOut {
		return printJSON(row)
	}
	fmt.Printf("%s — %s\n", row.Artist, row.Title)
	return nil
}

func CmdBrowse(env paths.Env, args []string) error {
	opt := library.BrowseOptions{}
	jsonOut := false
	for _, a := range args {
		switch a {
		case "--json":
			jsonOut = true
		case "--queue":
			opt.Queue = true
		case "--paths-only":
			opt.QueuePathsOnly = true
		case "--offset":
			continue
		case "--limit":
			continue
		default:
			if !strings.HasPrefix(a, "-") && opt.Rel == "" {
				opt.Rel = a
			}
		}
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--offset":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &opt.Offset)
			}
		case "--limit":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &opt.Limit)
			}
		}
	}
	out, err := library.Browse(library.EnvFrom(env), opt)
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSON(out)
	}
	for _, e := range out.Entries {
		if e.Type == "dir" {
			fmt.Println(e.Name)
		} else {
			fmt.Println(e.Path)
		}
	}
	return nil
}

func CmdGenres(env paths.Env, args []string) error {
	jsonOut := false
	for _, a := range args {
		if a == "--json" {
			jsonOut = true
		}
	}
	rows, err := library.Genres(library.EnvFrom(env))
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSON(rows)
	}
	for _, row := range rows {
		fmt.Println(row["name"])
	}
	return nil
}

func CmdTracks(env paths.Env, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: evoplayer tracks <genre> [--json]")
	}
	genre := args[0]
	jsonOut := false
	for _, a := range args[1:] {
		if a == "--json" {
			jsonOut = true
		}
	}
	rows, err := library.TracksForGenre(library.EnvFrom(env), genre)
	if err != nil {
		return fmt.Errorf("evoplayer: unknown genre: %s", genre)
	}
	if jsonOut {
		return printJSON(rows)
	}
	for _, row := range rows {
		fmt.Println(row.Path)
	}
	return nil
}

func CmdLibrary(env paths.Env, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: evoplayer library <browse|meta|import|cache|download>")
	}
	switch args[0] {
	case "browse":
		return CmdBrowse(env, args[1:])
	case "meta":
		if len(args) < 2 {
			return fmt.Errorf("usage: evoplayer library meta <path> [--json]")
		}
		return CmdMeta(env, args[1:])
	case "import":
		return runLibraryJob(env, "library.import", nil)
	case "cache":
		params := map[string]any{"force": hasFlag(args, "--force")}
		for _, a := range args[1:] {
			if !strings.HasPrefix(a, "-") {
				params["genre"] = a
				break
			}
		}
		return runLibraryJob(env, "library.cache", params)
	case "download":
		return CmdDownload(env, args[1:])
	default:
		return fmt.Errorf("unknown library command: %s", args[0])
	}
}

func CmdCache(env paths.Env, args []string) error {
	force := hasFlag(args, "--force")
	pruneArt := hasFlag(args, "--prune-art")
	jsonOut := hasFlag(args, "--json")
	libEnv := library.EnvFrom(env)
	if pruneArt {
		n, err := library.PruneArt(libEnv)
		if err != nil {
			return err
		}
		if jsonOut {
			return printJSON(map[string]int{"pruned": n})
		}
		return nil
	}
	genre := ""
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			genre = a
			break
		}
	}
	var res library.CacheResult
	var err error
	if genre != "" {
		res, err = library.CacheGenre(libEnv, genre, force)
	} else {
		res, err = library.CacheAll(libEnv, force)
	}
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSON(res)
	}
	fmt.Fprintf(os.Stderr, "evoplayer: cached %d track(s), %d already fresh (%d total)\n", res.Built, res.Skipped, res.Total)
	return nil
}

func runLibraryJob(env paths.Env, method string, params map[string]any) error {
	if err := EnsureDaemon(env, findExe(env)); err != nil {
		return err
	}
	resp, err := IPC(env, method, params)
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	return printJSON(resp.Data)
}

func readPlaylistName(env paths.Env) string {
	st, err := os.ReadFile(env.PlayerState)
	if err != nil {
		return ""
	}
	var saved struct {
		Playlist string `json:"playlist"`
	}
	if json.Unmarshal(st, &saved) != nil {
		return ""
	}
	return saved.Playlist
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func findExe(env paths.Env) string {
	candidates := []string{
		os.Getenv("EVOPLAYER_BIN"),
	}
	if home, _ := os.UserHomeDir(); home != "" {
		candidates = append(candidates,
			home+"/.local/bin/evoplayer",
			home+"/.local/lib/evoplayer/evoplayer",
		)
	}
	for _, c := range candidates {
		if c != "" {
			if st, err := os.Stat(c); err == nil && !st.IsDir() {
				return c
			}
		}
	}
	return "evoplayer"
}
