package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sebday/evoplayer/internal/cli"
	"github.com/sebday/evoplayer/internal/daemon"
	"github.com/sebday/evoplayer/internal/paths"
	"github.com/sebday/evoplayer/internal/tui"
)

func main() {
	env := paths.Load(repoRoot())
	exe, _ := os.Executable()

	if len(os.Args) < 2 {
		if err := tui.Run(env, exe); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "serve":
		err = daemon.New(env).Run()
	case "tui":
		err = tui.Run(env, exe)
	case "start":
		err = cli.CmdStart(env, exe)
	case "restart":
		err = cli.CmdRestart(env, exe)
	case "stop":
		err = cli.CmdStop(env)
	case "close":
		err = cli.CmdClose(env)
	case "toggle":
		err = cli.CmdToggle(env, exe)
	case "next":
		err = cli.CmdNext(env, exe)
	case "prev":
		err = cli.CmdPrev(env, exe)
	case "seek":
		sec := "0"
		if len(args) > 0 {
			sec = args[0]
		}
		err = cli.CmdSeek(env, exe, sec)
	case "shuffle":
		mode := "toggle"
		if len(args) > 0 {
			mode = args[0]
		}
		err = cli.CmdShuffle(env, exe, mode)
	case "volume":
		arg := "0"
		val := ""
		if len(args) > 0 {
			arg = args[0]
		}
		if len(args) > 1 {
			val = args[1]
		}
		err = cli.CmdVolume(env, exe, arg, val)
	case "load":
		err = cli.CmdLoad(env, exe, args)
	case "status":
		err = cli.CmdStatus(env, exe, hasJSON(args))
	case "open":
		err = cli.CmdOpen(env, exe, hasJSON(args))
	case "queue":
		if len(args) == 0 {
			err = fmt.Errorf("usage: evoplayer queue <append|play|extend|up-next>")
			break
		}
		switch args[0] {
		case "append":
			err = cli.CmdQueueAppend(env, exe, args[1:])
		case "play":
			err = cli.CmdQueuePlay(env, exe, args[1:])
		case "extend":
			err = cli.CmdQueueExtend(env, exe, hasJSON(args))
		case "up-next":
			err = cli.CmdQueueUpNext(env, exe, args[1:])
		default:
			err = fmt.Errorf("usage: evoplayer queue <append|play|extend|up-next>")
		}
	case "history":
		err = cli.CmdHistory(env, args)
	case "config":
		err = cli.CmdConfig(env, args)
	case "tags":
		err = cli.CmdTags(env, args)
	case "vinyl":
		err = cli.CmdVinyl(env, args)
	case "find":
		err = cli.CmdFind(env, args)
	case "meta":
		err = cli.CmdMeta(env, args)
	case "browse":
		err = cli.CmdBrowse(env, args)
	case "genres":
		err = cli.CmdGenres(env, args)
	case "tracks":
		err = cli.CmdTracks(env, args)
	case "library":
		err = cli.CmdLibrary(env, args)
	case "cache":
		err = cli.CmdCache(env, args)
	case "scrobble":
		err = cli.CmdScrobble(env, args)
	case "playback":
		err = cli.CmdPlaybackGroup(env, exe, args)
	case "placement":
		err = cli.CmdPlacement(env, args)
	case "jsonlog":
		err = cli.CmdJSONLog(env, args)
	case "lastfm":
		err = cli.CmdLastfm(env, args)
	case "waveform":
		err = cli.CmdWaveform(env, args)
	case "playlist":
		err = cli.CmdPlaylist(env, args)
	case "favorite":
		err = cli.CmdFavorite(env, args)
	case "current":
		err = cli.CmdCurrent(env, args)
	case "warm":
		err = cli.CmdWarm(env, args)
	case "download":
		err = cli.CmdDownload(env, args)
	case "stats":
		err = cli.CmdStats(env, args)
	case "job":
		err = cli.CmdJob(env, args)
	case "art":
		err = cli.CmdArt(env, args)
	case "sort":
		err = cli.CmdSort(env, args)
	case "viz":
		err = cli.CmdViz(env, exe, args)
	default:
		err = fmt.Errorf("evoplayer: unknown command %q", cmd)
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func hasJSON(args []string) bool {
	for _, a := range args {
		if a == "--json" {
			return true
		}
	}
	return false
}

func repoRoot() string {
	if v := os.Getenv("EVOPLAYER_ROOT"); v != "" {
		return v
	}
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	exe, _ = filepath.EvalSymlinks(exe)
	dir := filepath.Dir(exe)
	candidates := []string{
		dir,
		filepath.Join(dir, ".."),
		filepath.Join(dir, "..", ".."),
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "go.mod")); err == nil {
			return c
		}
		if _, err := os.Stat(filepath.Join(c, "bin", "evoplayer")); err == nil {
			return c
		}
	}
	return dir
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: evoplayer [command] [args...]")
	fmt.Fprintln(os.Stderr, "  (none)  terminal player")
	fmt.Fprintln(os.Stderr, "  serve   backend (playback, library, ipc)")
	fmt.Fprintln(os.Stderr, "  tui     alias for the terminal player")
	os.Exit(1)
}
