package tags

import (
	"regexp"
	"strings"
)

var trackNumPrefix = regexp.MustCompile(`^(?i)(?:\d{1,3}|incomplete\d{1,3})[-_.]`)

func ParseFilenameArtistTitle(stem string) (artist, title string) {
	stem = strings.TrimSpace(stem)
	if stem == "" {
		return "", ""
	}
	stem = strings.TrimSuffix(stem, "-def")
	stem = strings.TrimSuffix(stem, "_def")
	if strings.Contains(stem, " - ") {
		parts := strings.SplitN(stem, " - ", 2)
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	if trackNumPrefix.MatchString(stem) {
		stem = trackNumPrefix.ReplaceAllString(stem, "")
	}
	parts := strings.Split(stem, "-")
	if len(parts) >= 2 {
		artist = strings.ReplaceAll(parts[0], "_", " ")
		title = strings.ReplaceAll(strings.Join(parts[1:], "-"), "_", " ")
		return strings.TrimSpace(artist), strings.TrimSpace(title)
	}
	parts = strings.Split(stem, "_")
	if len(parts) >= 3 {
		artist = strings.ReplaceAll(parts[0], "-", " ")
		title = strings.ReplaceAll(strings.Join(parts[1:], "_"), "-", " ")
		return strings.TrimSpace(artist), strings.TrimSpace(title)
	}
	return "", strings.TrimSpace(stem)
}
