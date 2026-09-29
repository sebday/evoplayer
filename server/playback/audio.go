package playback

import "github.com/sebday/evoplayer/server/audio"

func IsSupportedPath(path string) bool {
	return audio.IsAudio(path)
}

func OpenDecoder(path string) (StreamSeeker, Format, error) {
	stream, format, err := openFFmpegDecoder(path)
	if err != nil {
		return nil, Format{}, err
	}
	return BoundStream(stream), format, nil
}
