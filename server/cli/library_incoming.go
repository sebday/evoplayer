package cli

import (
	"fmt"
	"os"

	"github.com/sebday/evoplayer/server/config"
	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/secrets"
	"github.com/sebday/evoplayer/server/soundcloud"
)

func CmdIncoming(env paths.Env, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: evoplayer library incoming <review|apply> [--dry-run]")
	}
	libEnv := library.EnvFrom(env)
	tok := secrets.SoundcloudOAuth()
	if tok.Token == "" {
		return fmt.Errorf("soundcloud oauth required (log in at soundcloud.com in Brave)")
	}
	fmt.Fprintf(os.Stderr, "evoplayer: soundcloud auth from %s\n", tok.Source)

	user, err := config.Get(env.MusicConfig, "soundcloud", "user", "seb-day")
	if err != nil {
		return err
	}
	setsURL := soundcloud.SetsURL(user)

	switch args[0] {
	case "review":
		path, err := soundcloud.IncomingSCReview(libEnv, setsURL, tok.Token)
		if err != nil {
			return err
		}
		fmt.Printf("wrote %s\n", path)
		return nil
	case "apply":
		dryRun := hasFlag(args[1:], "--dry-run")
		tagged, skipped, failed, err := soundcloud.IncomingSCApply(libEnv, setsURL, tok.Token, dryRun)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "evoplayer: tagged %d, skipped %d, failed %d\n", tagged, skipped, failed)
		if dryRun {
			fmt.Fprintln(os.Stderr, "dry-run only; re-run without --dry-run to apply")
		} else if tagged > 0 {
			fmt.Fprintln(os.Stderr, "next: evoplayer library import")
		}
		if failed > 0 {
			return fmt.Errorf("incoming apply had %d failure(s)", failed)
		}
		return nil
	default:
		return fmt.Errorf("usage: evoplayer library incoming <review|apply> [--dry-run]")
	}
}
