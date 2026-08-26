package cli

import (
	"fmt"

	"github.com/sebday/evoplayer/internal/paths"
)

func CmdHistory(env paths.Env, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: evoplayer history report")
	}
	switch args[0] {
	case "report":
		return CmdHistoryReport(env, args[1:])
	default:
		return fmt.Errorf("usage: evoplayer history report")
	}
}
