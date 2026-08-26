package cli

import (
	"fmt"

	"github.com/sebday/evoplayer/internal/paths"
	"github.com/sebday/evoplayer/internal/waveform"
)

func CmdWaveform(env paths.Env, args []string) error {
	_ = env
	if len(args) < 1 || args[0] != "build" {
		return fmt.Errorf("usage: evoplayer waveform build <path> <out>")
	}
	if len(args) < 3 {
		return fmt.Errorf("usage: evoplayer waveform build <path> <out>")
	}
	return waveform.Build(args[1], args[2])
}
