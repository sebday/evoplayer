package cli

import (
	"fmt"

	"github.com/sebday/evoplayer/internal/paths"
)

func CmdJob(env paths.Env, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: evoplayer job <status|stop> [--json]")
	}
	jsonOut := hasFlag(args, "--json")
	switch args[0] {
	case "status":
		if err := EnsureDaemon(env, findExe(env)); err != nil {
			return err
		}
		resp, err := IPC(env, "job.status", nil)
		if err != nil {
			return err
		}
		if !resp.OK {
			return fmt.Errorf("%s", resp.Error)
		}
		if jsonOut {
			return printJSON(resp.Data)
		}
		return printJSON(resp.Data)
	case "stop", "cancel":
		if err := EnsureDaemon(env, findExe(env)); err != nil {
			return err
		}
		resp, err := IPC(env, "job.cancel", nil)
		if err != nil {
			return err
		}
		if !resp.OK {
			return fmt.Errorf("%s", resp.Error)
		}
		if jsonOut {
			return printJSON(resp.Data)
		}
		return printJSON(resp.Data)
	default:
		return fmt.Errorf("unknown job command: %s", args[0])
	}
}
