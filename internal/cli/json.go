package cli

import (
	"encoding/json"
	"fmt"

	"github.com/sebday/evoplayer/internal/paths"
	"github.com/sebday/evoplayer/internal/playback"
	"github.com/sebday/evoplayer/internal/status"
)

func printJSON(v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = fmt.Println(string(b))
	return err
}

func decodeStatus(data interface{}) (playback.Status, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return playback.Status{}, err
	}
	var st playback.Status
	if err := json.Unmarshal(b, &st); err != nil {
		return playback.Status{}, err
	}
	return st.WithLabels(), nil
}

func savedStatusJSON(env paths.Env) playback.Status {
	return status.Saved(env)
}
