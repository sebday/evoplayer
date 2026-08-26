package audio

import (
	"path/filepath"
	"strings"
)

var exts = map[string]struct{}{
	"flac": {}, "mp3": {}, "m4a": {}, "aac": {}, "ogg": {},
	"opus": {}, "wav": {}, "aiff": {}, "aif": {}, "wma": {},
	"ape": {}, "wv": {}, "alac": {}, "mka": {},
}

func IsAudio(path string) bool {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	_, ok := exts[ext]
	return ok
}
