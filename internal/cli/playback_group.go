package cli

import (
	"fmt"

	"github.com/sebday/evoplayer/internal/paths"
)

func CmdPlaybackGroup(env paths.Env, exe string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: evoplayer playback <toggle|next|prev|stop|seek|volume|shuffle>")
	}
	switch args[0] {
	case "toggle":
		return CmdToggle(env, exe)
	case "next":
		return CmdNext(env, exe)
	case "prev":
		return CmdPrev(env, exe)
	case "stop":
		return CmdStop(env)
	case "seek":
		sec := "0"
		if len(args) > 1 {
			sec = args[1]
		}
		return CmdSeek(env, exe, sec)
	case "volume":
		arg := "0"
		val := ""
		if len(args) > 1 {
			arg = args[1]
		}
		if len(args) > 2 {
			val = args[2]
		}
		return CmdVolume(env, exe, arg, val)
	case "shuffle":
		mode := "toggle"
		if len(args) > 1 {
			mode = args[1]
		}
		return CmdShuffle(env, exe, mode)
	default:
		return fmt.Errorf("usage: evoplayer playback <toggle|next|prev|stop|seek|volume|shuffle>")
	}
}
