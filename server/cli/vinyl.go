package cli

import (
	"fmt"

	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/vinyl"
)

func CmdVinyl(env paths.Env, args []string) error {
	if len(args) == 0 || args[0] != "by-label" {
		return fmt.Errorf("usage: evoplayer vinyl by-label [root] [--execute]")
	}
	root := ""
	execute := false
	for _, arg := range args[1:] {
		switch arg {
		case "--execute":
			execute = true
		default:
			if root == "" {
				root = arg
			}
		}
	}
	result, code := vinyl.ByLabel(root, execute)
	vinyl.PrintResult(result)
	if code != 0 {
		return fmt.Errorf("exit %d", code)
	}
	return nil
}
