package cli

import (
	"fmt"
	"strings"

	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/playlist"
)

func CmdPlaylist(env paths.Env, args []string) error {
	jsonOut := false
	offset := -1
	limit := -1
	name := ""
	sub := ""
	positional := []string{}
	for _, a := range args {
		switch a {
		case "--json":
			jsonOut = true
		case "--offset", "--limit":
			continue
		default:
			if !strings.HasPrefix(a, "-") {
				positional = append(positional, a)
			}
		}
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--offset":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &offset)
			}
		case "--limit":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &limit)
			}
		}
	}
	if len(positional) > 0 {
		switch positional[0] {
		case "create", "rename", "delete", "star":
			sub = positional[0]
			positional = positional[1:]
		default:
			name = positional[0]
		}
	}
	penv := playlist.EnvFrom(env)
	switch sub {
	case "create":
		n := firstArg(positional)
		out, err := playlist.CreateUser(penv, n)
		if err != nil {
			return err
		}
		if jsonOut {
			return printJSON(out)
		}
		fmt.Printf("created playlist %s\n", n)
		return nil
	case "rename":
		if len(positional) < 2 {
			return fmt.Errorf("usage: evoplayer playlist rename <old> <new> [--json]")
		}
		out, err := playlist.RenameUser(penv, positional[0], positional[1])
		if err != nil {
			return err
		}
		if jsonOut {
			return printJSON(out)
		}
		fmt.Printf("renamed playlist %s -> %s\n", positional[0], positional[1])
		return nil
	case "delete":
		n := firstArg(positional)
		out, err := playlist.DeleteUser(penv, n)
		if err != nil {
			return err
		}
		if jsonOut {
			return printJSON(out)
		}
		fmt.Printf("deleted playlist %s\n", n)
		return nil
	case "star":
		if len(positional) == 0 || positional[0] != "toggle" {
			return fmt.Errorf("usage: evoplayer playlist star toggle <name> [--json]")
		}
		positional = positional[1:]
		n := firstArg(positional)
		out, err := playlist.StarToggle(penv, n)
		if err != nil {
			return err
		}
		if jsonOut {
			return printJSON(out)
		}
		if out.Starred {
			fmt.Printf("starred %s\n", n)
		} else {
			fmt.Printf("unstarred %s\n", n)
		}
		return nil
	}
	if name == "" {
		out, err := playlist.ListIndex(penv)
		if err != nil {
			return err
		}
		if jsonOut {
			return printJSON(out)
		}
		for _, item := range out {
			fmt.Println(item.Name)
		}
		return nil
	}
	if offset >= 0 || limit >= 0 {
		if limit <= 0 {
			limit = 50
		}
		if offset < 0 {
			offset = 0
		}
		out, err := playlist.TracksPageFor(penv, name, offset, limit)
		if err != nil {
			return err
		}
		if jsonOut {
			return printJSON(out)
		}
		for _, item := range out.Items {
			fmt.Println(item.Path)
		}
		return nil
	}
	out, err := playlist.TracksAll(penv, name)
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSON(out)
	}
	for _, item := range out {
		fmt.Println(item.Path)
	}
	return nil
}

func CmdFavorite(env paths.Env, args []string) error {
	jsonOut := false
	path := ""
	for _, a := range args {
		switch a {
		case "--json":
			jsonOut = true
		case "favorite", "toggle":
			continue
		default:
			if !strings.HasPrefix(a, "-") && path == "" {
				path = a
			}
		}
	}
	if path == "" {
		return fmt.Errorf("usage: evoplayer favorite toggle <path> [--json]")
	}
	out, err := playlist.FavoriteToggle(playlist.EnvFrom(env), path)
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSON(out)
	}
	if out.Liked {
		fmt.Println("liked")
	} else {
		fmt.Println("unliked")
	}
	return nil
}

func CmdCurrent(env paths.Env, args []string) error {
	sub := ""
	jsonOut := false
	for _, a := range args {
		switch a {
		case "--json":
			jsonOut = true
		default:
			if !strings.HasPrefix(a, "-") && sub == "" {
				sub = a
			}
		}
	}
	switch sub {
	case "load", "":
		out, err := playlist.LoadCurrent(playlist.EnvFrom(env))
		if err != nil {
			return err
		}
		if jsonOut || sub == "" {
			return printJSON(out)
		}
		for _, item := range out {
			fmt.Println(item.Path)
		}
		return nil
	case "save":
		var pathsArg []string
		for _, a := range args {
			if a == sub || a == "--json" || strings.HasPrefix(a, "-") {
				continue
			}
			pathsArg = append(pathsArg, a)
		}
		if len(pathsArg) == 0 {
			return fmt.Errorf("usage: evoplayer current save <path> [...]")
		}
		return playlist.SaveCurrent(playlist.EnvFrom(env), pathsArg)
	case "clear":
		return playlist.ClearCurrent(playlist.EnvFrom(env))
	default:
		return fmt.Errorf("usage: evoplayer current <load|save|clear>")
	}
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
