package art

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sebday/evoplayer/internal/tags"
)

var catnoRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*[0-9][A-Za-z0-9._/-]*$`)

type trackMeta struct {
	Artist string
	Album  string
	Title  string
	Catno  string
	Year   string
}

func readTrackMeta(path string) (trackMeta, error) {
	tag, err := tags.ReadTags(path)
	if err != nil {
		return trackMeta{}, err
	}
	meta := trackMeta{
		Artist: firstNonEmpty(tag.Artist, tag.AlbumArtist),
		Album:  tag.Album,
		Title:  tag.Title,
		Catno:  tag.CatalogNumber,
		Year:   tag.Year,
	}
	if meta.Catno == "" && looksLikeCatno(meta.Album) {
		meta.Catno = meta.Album
	}
	if meta.Title == "" {
		meta.Title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	return meta, nil
}

func looksLikeCatno(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 3 || len(s) > 24 {
		return false
	}
	if strings.Contains(s, " ") {
		return false
	}
	return catnoRe.MatchString(s)
}

func normCatno(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func discogsJoinQuery(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " - ")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
