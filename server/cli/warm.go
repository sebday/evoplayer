package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/warm"
)

type warmProgressPrinter struct {
	total    int
	lastDone int
	step     int
}

func newWarmProgressPrinter(total int) *warmProgressPrinter {
	step := total / 50
	if step < 1 {
		step = 1
	}
	return &warmProgressPrinter{total: total, step: step}
}

func (p *warmProgressPrinter) report(done int, folder string) {
	if done <= 0 {
		return
	}
	if done != p.total && done != 1 && done-p.lastDone < p.step {
		return
	}
	p.lastDone = done
	fmt.Fprintf(os.Stderr, "evoplayer: warming %d/%d%s\n", done, p.total, warmFolderSuffix(folder))
}

func CmdWarm(env paths.Env, args []string) error {
	jsonOut := false
	doBatch := false
	warmAll := false
	path := ""
	folder := ""
	var batchPaths []string
	for _, a := range args {
		switch a {
		case "--json":
			jsonOut = true
		case "--batch":
			doBatch = true
		case "--all":
			warmAll = true
		default:
			if !strings.HasPrefix(a, "-") && folder == "" && warmAll {
				folder = a
				continue
			}
			if path == "" && !doBatch && !warmAll {
				path = a
			} else if doBatch || path != "" {
				batchPaths = append(batchPaths, a)
			}
		}
	}
	if warmAll {
		libEnv := library.EnvFrom(env)
		trackPaths, err := library.ListTrackPathsInDir(libEnv, folder)
		if err != nil {
			return err
		}
		if len(trackPaths) == 0 {
			return fmt.Errorf("evoplayer: no indexed tracks to warm")
		}
		progress := newWarmProgressPrinter(len(trackPaths))
		out, err := warm.BatchTracks(env, trackPaths, warm.DefaultWorkers, func(p warm.BatchProgress) {
			progress.report(p.Done, p.Folder)
		})
		if err != nil {
			return err
		}
		built := 0
		withArt := 0
		for _, res := range out {
			if res.Art != "" {
				withArt++
			}
			if res.ArtBuilt {
				built++
			}
		}
		if jsonOut {
			return printJSON(map[string]any{
				"total":      len(out),
				"with_art":   withArt,
				"art_built":  built,
				"folder":     folder,
			})
		}
		msg := fmt.Sprintf("evoplayer: warmed %d track(s), %d with art", len(out), withArt)
		if built > 0 {
			msg += fmt.Sprintf(" (%d newly extracted)", built)
		}
		fmt.Fprintln(os.Stderr, msg)
		return nil
	}
	if doBatch {
		out, err := warm.BatchTracks(env, batchPaths, warm.DefaultWorkers, nil)
		if err != nil {
			return err
		}
		if jsonOut {
			return printJSON(out)
		}
		fmt.Printf("warmed %d tracks\n", len(out))
		return nil
	}
	if path == "" {
		return fmt.Errorf("usage: evoplayer warm <path> [--json] | evoplayer warm --all [<folder>] [--json] | evoplayer warm --batch <paths...>")
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

func warmFolderSuffix(folder string) string {
	if folder == "" {
		return ""
	}
	return " (" + folder + ")"
}
