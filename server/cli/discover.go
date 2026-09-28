package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playback"
	"github.com/sebday/evoplayer/server/soundcloud"
)

func CmdDiscover(env paths.Env, args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "keep":
			return discoverID(env, "discover.keep", args[1:])
		case "dismiss":
			return discoverID(env, "discover.dismiss", args[1:])
		}
	}
	return discoverSimilar(env, args)
}

func discoverID(env paths.Env, method string, args []string) error {
	raw := ""
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		raw = strings.TrimSpace(a)
		break
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id == 0 {
		return fmt.Errorf("usage: evoplayer discover %s <id>", discoverVerb(method))
	}
	return runLibraryJob(env, method, map[string]any{"id": id})
}

func discoverVerb(method string) string {
	if i := strings.LastIndex(method, "."); i >= 0 {
		return method[i+1:]
	}
	return method
}

func discoverSimilar(env paths.Env, args []string) error {
	jsonOut := hasFlag(args, "--json")
	if err := EnsureDaemon(env, findExe(env)); err != nil {
		return err
	}
	st, err := PlaybackStatus(env)
	if err != nil {
		return err
	}
	if strings.TrimSpace(st.Path) == "" {
		return fmt.Errorf("evoplayer: nothing playing")
	}
	resp, err := IPC(env, "discover.similar", map[string]any{"path": st.Path})
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	if jsonOut {
		return printJSON(resp.Data)
	}
	return printDiscover(resp.Data)
}

func printDiscover(data interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	var res soundcloud.SimilarResult
	if err := json.Unmarshal(b, &res); err != nil {
		return err
	}
	seed := strings.TrimSpace(res.SeedArtist)
	if title := strings.TrimSpace(res.SeedTitle); title != "" {
		if seed != "" {
			seed += " — " + title
		} else {
			seed = title
		}
	}
	if seed != "" {
		fmt.Printf("similar to %s\n", seed)
	}
	if len(res.Tracks) == 0 {
		fmt.Println("nothing new")
		return nil
	}
	for _, track := range res.Tracks {
		label := strings.TrimSpace(track.Artist)
		if title := strings.TrimSpace(track.Title); title != "" {
			if label != "" {
				label += " — " + title
			} else {
				label = title
			}
		}
		fmt.Printf("%d\t%s\t%s\n", track.ID, playback.FormatTime(track.Duration), label)
	}
	return nil
}
