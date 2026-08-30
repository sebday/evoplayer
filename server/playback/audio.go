package playback

import (
	"fmt"
	"path/filepath"
	"strings"
)

var supportedExts = map[string]struct{}{
	".mp3": {}, ".flac": {}, ".ogg": {}, ".wav": {},
	".m4a": {}, ".aac": {}, ".opus": {}, ".wma": {},
}

func IsSupportedPath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := supportedExts[ext]
	return ok
}

func OpenDecoder(path string) (StreamSeekCloser, Format, error) {
	if !ffmpegAvailable() {
		return nil, Format{}, fmt.Errorf("ffmpeg not available")
	}
	stream, format, err := openFFmpegDecoder(path)
	if err != nil {
		return nil, Format{}, err
	}
	return BoundStream(stream), format, nil
}
