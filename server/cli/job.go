package cli

import (
	"fmt"

	"github.com/sebday/evoplayer/server/paths"
)

func CmdJob(env paths.Env, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: evoplayer job <status|stop> [--json]")
	}
	switch args[0] {
	case "status":
		if !DaemonUp(env) {
			return printJSON(map[string]any{"status": "idle"})
		}
		resp, err := ipcOK(env, "job.status", nil)
		if err != nil {
			return err
		}
		return printJSON(resp.Data)
	case "stop", "cancel":
		if err := EnsureDaemon(env, findExe()); err != nil {
			return err
		}
		resp, err := ipcOK(env, "job.cancel", nil)
		if err != nil {
			return err
		}
		return printJSON(resp.Data)
	default:
		return fmt.Errorf("unknown job command: %s", args[0])
	}
}
