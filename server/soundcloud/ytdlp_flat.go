package soundcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sebday/evoplayer/server/syncarchive"
	"github.com/sebday/evoplayer/server/tags"
)

type ytdlpFlatEntry struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Uploader   string `json:"uploader"`
	WebpageURL string `json:"webpage_url"`
	URL        string `json:"url"`
	IEKey      string `json:"ie_key"`
	Extractor  string `json:"extractor"`
}

type ytdlpFlatPlaylist struct {
	Entries []ytdlpFlatEntry `json:"entries"`
}

func ytdlpFlatEntries(ctx context.Context, pageURL, oauthToken, clientID string) ([]ytdlpFlatEntry, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	bin, err := exec.LookPath("yt-dlp")
	if err != nil {
		return nil, err
	}
	try := func(extra []string) ([]ytdlpFlatEntry, error) {
		base := ytDlpSoundcloudBaseArgs(oauthToken, clientID, "-", false)
		base = removeOutTemplate(base)
		args := append([]string{"-J", "--flat-playlist"}, base...)
		args = append(args, pageURL)
		if len(extra) > 0 {
			args = append(extra, args...)
		}
		cmd := exec.CommandContext(ctx, bin, args...)
		outBytes, err := cmd.Output()
		if err != nil {
			return nil, err
		}
		return parseFlatPlaylist(outBytes)
	}
	if strings.TrimSpace(oauthToken) != "" {
		if entries, err := try(nil); err == nil {
			return entries, nil
		}
	}
	for _, browser := range []string{"brave", "chromium"} {
		if entries, err := try([]string{"--cookies-from-browser", browser}); err == nil {
			return entries, nil
		}
	}
	return try(nil)
}

func removeOutTemplate(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "-o" {
			i++
			continue
		}
		if args[i] == "-f" {
			i++
			continue
		}
		out = append(out, args[i])
	}
	return out
}

func parseFlatPlaylist(raw []byte) ([]ytdlpFlatEntry, error) {
	var playlist ytdlpFlatPlaylist
	if err := json.Unmarshal(raw, &playlist); err == nil && len(playlist.Entries) > 0 {
		return filterFlatEntries(playlist.Entries), nil
	}
	var one ytdlpFlatEntry
	if err := json.Unmarshal(raw, &one); err != nil {
		return nil, fmt.Errorf("yt-dlp: parse playlist: %w", err)
	}
	if strings.TrimSpace(one.ID) == "" {
		return nil, fmt.Errorf("yt-dlp: empty playlist")
	}
	return filterFlatEntries([]ytdlpFlatEntry{one}), nil
}

func filterFlatEntries(in []ytdlpFlatEntry) []ytdlpFlatEntry {
	out := make([]ytdlpFlatEntry, 0, len(in))
	for _, e := range in {
		if strings.TrimSpace(e.ID) == "" {
			continue
		}
		out = append(out, e)
	}
	return out
}

func flatEntryURL(e ytdlpFlatEntry) string {
	if u := strings.TrimSpace(e.WebpageURL); u != "" {
		return u
	}
	return strings.TrimSpace(e.URL)
}

func flatDestPath(incoming string, e ytdlpFlatEntry) string {
	artist := tags.SanitizeFilenamePart(strings.TrimSpace(e.Uploader))
	if artist == "" {
		artist = "SoundCloud"
	}
	title := tags.SanitizeFilenamePart(strings.TrimSpace(e.Title))
	if title == "" {
		title = strings.TrimSpace(e.ID)
	}
	return filepath.Join(incoming, artist+" - "+title+".mp3")
}

func archiveHasFlatEntry(a *syncarchive.Archive, e ytdlpFlatEntry) bool {
	id := strings.TrimSpace(e.ID)
	key := strings.ToLower(strings.TrimSpace(e.IEKey))
	ext := strings.ToLower(strings.TrimSpace(e.Extractor))
	if strings.Contains(key, "youtube") || strings.Contains(ext, "youtube") {
		return a.HasYT(id)
	}
	if n, err := strconv.ParseInt(id, 10, 64); err == nil {
		return a.HasSC(n)
	}
	return false
}

func archiveAddFlatEntry(a *syncarchive.Archive, e ytdlpFlatEntry) error {
	id := strings.TrimSpace(e.ID)
	key := strings.ToLower(strings.TrimSpace(e.IEKey))
	ext := strings.ToLower(strings.TrimSpace(e.Extractor))
	if strings.Contains(key, "youtube") || strings.Contains(ext, "youtube") {
		return a.AddYT(id)
	}
	if n, err := strconv.ParseInt(id, 10, 64); err == nil {
		return a.AddSC(n)
	}
	return nil
}
