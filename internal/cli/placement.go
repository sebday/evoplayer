package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/sebday/evoplayer/internal/paths"
	"github.com/sebday/evoplayer/internal/placement"
)

func CmdPlacement(env paths.Env, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: evoplayer placement <log|undo-plan>")
	}
	switch args[0] {
	case "log":
		return cmdPlacementLog(env, args[1:])
	case "undo-plan":
		return cmdPlacementUndoPlan(env, args[1:])
	default:
		return fmt.Errorf("usage: evoplayer placement <log|undo-plan>")
	}
}

func cmdPlacementLog(env paths.Env, args []string) error {
	jsonOut := false
	limit := 50
	undoable := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOut = true
		case "--undoable":
			undoable = true
		case "--limit":
			if i+1 < len(args) {
				i++
				limit, _ = strconv.Atoi(args[i])
			}
		}
	}
	if err := env.EnsureDirs(); err != nil {
		return err
	}
	logPath := placementLogPath(env)
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		if jsonOut {
			fmt.Println("[]")
		}
		return nil
	}
	rows, err := placement.Log(logPath, limit, undoable)
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSONRows(rows)
	}
	for _, row := range rows {
		fmt.Printf("%s\t%s\t%s\t->\t%s\n", fmt.Sprint(row["at"]), fmt.Sprint(row["op"]), fmt.Sprint(row["from"]), fmt.Sprint(row["to"]))
	}
	return nil
}

func cmdPlacementUndoPlan(env paths.Env, args []string) error {
	last := 1
	for i := 0; i < len(args); i++ {
		if args[i] == "--last" && i+1 < len(args) {
			i++
			last, _ = strconv.Atoi(args[i])
		}
	}
	if err := env.EnsureDirs(); err != nil {
		return err
	}
	rows, err := placement.UndoPlan(placementLogPath(env), last)
	if err != nil {
		return err
	}
	return printJSONRows(rows)
}
