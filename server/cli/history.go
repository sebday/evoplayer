package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/sebday/evoplayer/server/history"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/secrets"
)

func CmdHistory(env paths.Env, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: evoplayer history report")
	}
	switch args[0] {
	case "report":
		secrets.Load()
		return CmdHistoryReport(env, args[1:])
	default:
		return fmt.Errorf("usage: evoplayer history report")
	}
}

func CmdHistoryReport(env paths.Env, args []string) error {
	jsonOut := false
	week := "0"
	limit := 12
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOut = true
		case "--week":
			if i+1 < len(args) {
				i++
				week = args[i]
			}
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
	user := os.Getenv("LASTFM_USER")
	if user == "" {
		user = "distortedmind"
	}
	report, err := history.Generate(history.Params{
		CacheDir:    filepath.Join(env.CacheDir, "history"),
		User:        user,
		APIKey:      os.Getenv("LASTFM_API_KEY"),
		WeekFrom:    week,
		ScrobbleLog: filepath.Join(env.StateDir, "scrobble.jsonl"),
		Limit:       limit,
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSON(report)
	}
	fmt.Printf("scrobbles: %d artists: %d\n", report.Totals.Scrobbles, report.Totals.Artists)
	return nil
}
