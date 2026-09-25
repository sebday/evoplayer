package soundcloud

import (
	"strings"

	"github.com/sebday/evoplayer/server/library"
)

func parseTagList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []string
	var cur strings.Builder
	inQuote := false
	for i := 0; i < len(raw); i++ {
		switch raw[i] {
		case '"':
			inQuote = !inQuote
		case ' ':
			if inQuote {
				cur.WriteByte(' ')
			} else if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteByte(raw[i])
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

func trackEmbedGenre(track *Track, opts DownloadOptions) string {
	if track == nil {
		return ""
	}
	env := library.Env{MusicRoot: opts.MusicRoot, MusicConfig: opts.MusicConfig}
	candidates := []string{strings.TrimSpace(track.Genre)}
	candidates = append(candidates, parseTagList(track.TagList)...)
	if folder := library.MatchLibraryGenre(env, candidates...); folder != "" {
		return folder
	}
	if folder := library.GuessFolderFromCandidates(env, candidates...); folder != "" {
		return folder
	}
	if g := strings.TrimSpace(track.Genre); g != "" {
		return g
	}
	for _, tag := range candidates[1:] {
		if tag = strings.TrimSpace(tag); tag != "" {
			return tag
		}
	}
	return ""
}

// FolderFromLikedTrack returns a library folder for a liked SoundCloud track.
func FolderFromLikedTrack(env library.Env, track *Track) string {
	if track == nil {
		return ""
	}
	opts := DownloadOptions{MusicRoot: env.MusicRoot, MusicConfig: env.MusicConfig}
	if g := strings.TrimSpace(trackEmbedGenre(track, opts)); g != "" {
		if folder := library.MatchLibraryGenre(env, g); folder != "" {
			return folder
		}
		if folder := library.GuessFolderFromCandidates(env, g); folder != "" {
			return folder
		}
	}
	candidates := []string{strings.TrimSpace(track.Genre)}
	candidates = append(candidates, parseTagList(track.TagList)...)
	return library.GuessFolderFromCandidates(env, candidates...)
}
