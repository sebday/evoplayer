package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sebday/evoplayer/internal/config"
	"github.com/sebday/evoplayer/internal/history"
	"github.com/sebday/evoplayer/internal/paths"
)

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
		b, err := json.Marshal(report)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}
	fmt.Printf("scrobbles: %d artists: %d\n", report.Totals.Scrobbles, report.Totals.Artists)
	return nil
}

func CmdConfig(env paths.Env, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: evoplayer config <get|set|toml-get|...>")
	}
	switch args[0] {
	case "get":
		return cmdConfigGet(env, args[1:])
	case "set":
		return cmdConfigSet(env, args[1:])
	case "toml-get":
		return cmdConfigTomlGet(env, args[1:])
	case "toml-set":
		return cmdConfigTomlSet(env, args[1:])
	case "toml-json":
		return cmdConfigTomlJSON(env)
	case "toml-prune-derived":
		return cmdConfigTomlPrune(env)
	case "read-root":
		return cmdConfigReadRoot(args[1:])
	case "skip-dirs":
		return cmdConfigSkipDirs(env)
	case "pick":
		return cmdConfigPick(env)
	default:
		return fmt.Errorf("evoplayer: unknown config subcommand: %s", args[0])
	}
}

func cmdConfigGet(env paths.Env, args []string) error {
	jsonOut := false
	for _, arg := range args {
		if arg == "--json" {
			jsonOut = true
		}
	}
	if err := env.EnsureDirs(); err != nil {
		return err
	}
	view, err := config.JSON(env.MusicConfig, env.MusicRoot)
	if err != nil {
		return err
	}
	if jsonOut {
		return printJSON(view)
	}
	fmt.Printf("user: %s\noauth_token: %s\n", view.Soundcloud.User, maskSecret(view.Soundcloud.OAuthToken))
	return nil
}

func maskSecret(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "…" + s[len(s)-2:]
}

func cmdConfigSet(env paths.Env, args []string) error {
	jsonOut := false
	key := ""
	value := ""
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOut = true
		default:
			if key == "" {
				key = arg
			} else if value == "" {
				value = arg
			}
		}
	}
	if key == "" || value == "" {
		return fmt.Errorf("evoplayer: usage: evoplayer config set section.key value")
	}
	section, field, ok := strings.Cut(key, ".")
	if !ok || section == "" || field == "" {
		return fmt.Errorf("evoplayer: usage: evoplayer config set section.key value")
	}
	if err := env.EnsureDirs(); err != nil {
		return err
	}
	if section == "paths" && field == "root" {
		value = strings.TrimRight(value, "/")
		if err := config.ValidateMusicRoot(value); err != nil {
			return err
		}
		env.MusicRoot = value
		if err := os.MkdirAll(env.CacheDir, 0o755); err != nil {
			return err
		}
	}
	if err := config.Set(env.MusicConfig, section, field, value); err != nil {
		return err
	}
	if jsonOut {
		view, err := config.JSON(env.MusicConfig, env.MusicRoot)
		if err != nil {
			return err
		}
		return printJSON(view)
	}
	fmt.Println("ok")
	return nil
}

func cmdConfigTomlGet(env paths.Env, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: evoplayer config toml-get <section> <key> [default]")
	}
	defaultVal := ""
	if len(args) > 2 {
		defaultVal = args[2]
	}
	val, err := config.Get(env.MusicConfig, args[0], args[1], defaultVal)
	if err != nil {
		return err
	}
	fmt.Print(val)
	return nil
}

func cmdConfigTomlSet(env paths.Env, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: evoplayer config toml-set <section> <key> <value>")
	}
	if err := env.EnsureDirs(); err != nil {
		return err
	}
	return config.Set(env.MusicConfig, args[0], args[1], args[2])
}

func cmdConfigTomlJSON(env paths.Env) error {
	view, err := config.JSON(env.MusicConfig, env.MusicRoot)
	if err != nil {
		return err
	}
	b, err := json.Marshal(view)
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func cmdConfigTomlPrune(env paths.Env) error {
	return config.PruneDerived(env.MusicConfig)
}

func cmdConfigReadRoot(args []string) error {
	root := config.ReadRoot(args...)
	fmt.Print(root)
	return nil
}

func cmdConfigSkipDirs(env paths.Env) error {
	dirs, err := config.SkipDirs(env.MusicConfig)
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		fmt.Println(dir)
	}
	return nil
}

func cmdConfigPick(env paths.Env) error {
	start := env.MusicRoot
	if st, err := os.Stat(start); err != nil || !st.IsDir() {
		if home, err := os.UserHomeDir(); err == nil {
			start = home
		}
	}
	out, err := exec.Command("zenity", "--file-selection", "--directory",
		"--title=Select music library", "--filename="+start+"/").Output()
	if err != nil {
		return nil
	}
	selected := strings.TrimSpace(string(out))
	if selected == "" {
		return nil
	}
	return cmdConfigSet(env, []string{"paths.root", selected, "--json"})
}
