package cli

import (
	"encoding/json"
	"fmt"

	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/tags"
)

func CmdTags(env paths.Env, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: evoplayer tags standardize <path>")
	}
	switch args[0] {
	case "standardize":
		if len(args) < 2 {
			return fmt.Errorf("usage: evoplayer tags standardize <path>")
		}
		result, code, err := tags.StandardizePath(env.MusicRoot, args[1])
		if err != nil {
			return err
		}
		switch v := result.(type) {
		case tags.FileResult:
			tags.PrintFileResult(v)
		default:
			b, err := json.Marshal(v)
			if err != nil {
				return err
			}
			fmt.Println(string(b))
		}
		if code != 0 {
			return fmt.Errorf("exit %d", code)
		}
		return nil
	case "read":
		if len(args) < 2 {
			return fmt.Errorf("usage: evoplayer tags read <path>")
		}
		b, err := tags.ReadTagsJSON(args[1])
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	case "sanitize":
		if len(args) < 2 {
			return fmt.Errorf("usage: evoplayer tags sanitize <string>")
		}
		fmt.Print(tags.SanitizeFilenamePart(args[1]))
		return nil
	case "slugify":
		if len(args) < 2 {
			return fmt.Errorf("usage: evoplayer tags slugify <string>")
		}
		fmt.Print(tags.Slugify(args[1]))
		return nil
	default:
		return fmt.Errorf("usage: evoplayer tags <standardize|read|sanitize|slugify>")
	}
}
