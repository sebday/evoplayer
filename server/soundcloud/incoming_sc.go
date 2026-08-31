package soundcloud

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/sebday/evoplayer/server/config"
	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/tags"
)

type incomingSCMeta struct {
	artist    string
	title     string
	playlists []string
	folder    string
}

type incomingSCRow struct {
	file      string
	artist    string
	title     string
	genre     string
	playlists []string
	folder    string
	status    string
}

var scTrackIDRe = regexp.MustCompile(`/tracks/(\d+)`)

// IncomingSCReview writes import-review.txt for .incoming using soundcloud playlists.
func IncomingSCReview(env library.Env, setsURL, oauth string) (string, error) {
	playlistFolder, byKey, err := buildIncomingSCIndex(env, oauth, setsURL)
	if err != nil {
		return "", err
	}
	outPath := filepath.Join(env.MusicRoot, ".incoming", "import-review.txt")
	rows := collectIncomingSCRows(env, byKey)
	if err := writeIncomingSCReview(outPath, env.MusicRoot, setsURL, playlistFolder, rows); err != nil {
		return "", err
	}
	return outPath, nil
}

// IncomingSCApply tags .incoming files from soundcloud playlist folders.
func IncomingSCApply(env library.Env, setsURL, oauth string, dryRun bool) (tagged, skipped, failed int, err error) {
	_, byKey, err := buildIncomingSCIndex(env, oauth, setsURL)
	if err != nil {
		return 0, 0, 0, err
	}

	incoming := filepath.Join(env.MusicRoot, ".incoming")
	entries, err := os.ReadDir(incoming)
	if err != nil {
		return 0, 0, 0, err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(incoming, e.Name())
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".mp3" && ext != ".flac" && ext != ".m4a" {
			continue
		}

		preview := library.PreviewIncoming(env, path)
		if folder := strings.TrimSpace(fmt.Sprint(preview["folder"])); folder != "" {
			skipped++
			continue
		}

		probed, _ := tags.ProbeImport(path)
		meta, ok := lookupIncomingSCMeta(byKey, probed.Tag.Artist, probed.Tag.Title, e.Name(), ext)
		if !ok || strings.TrimSpace(meta.folder) == "" {
			skipped++
			continue
		}

		pl := strings.Join(meta.playlists, ", ")
		if dryRun {
			fmt.Printf("tag %s -> %s (%s)\n", e.Name(), meta.folder, pl)
			tagged++
			continue
		}

		if _, err := library.SetIncomingGenre(env, path, meta.folder); err != nil {
			fmt.Fprintf(os.Stderr, "evoplayer: fail %s: %v\n", e.Name(), err)
			failed++
			continue
		}
		fmt.Printf("tagged %s -> %s (%s)\n", e.Name(), meta.folder, pl)
		tagged++
	}
	return tagged, skipped, failed, nil
}

func buildIncomingSCIndex(env library.Env, oauth, setsURL string) (map[string]string, map[string]incomingSCMeta, error) {
	playlistFolder := map[string]string{}
	urlToMeta := map[string]incomingSCMeta{}
	keyToMeta := map[string]incomingSCMeta{}

	client := NewClient("", oauth)
	fmt.Fprintln(os.Stderr, "evoplayer: fetching likes...")
	likes, err := client.LikesTracksProgressCtx(nil, nil)
	if err != nil {
		return nil, nil, err
	}
	fmt.Fprintf(os.Stderr, "evoplayer: %d liked tracks\n", len(likes))

	idToKey := map[int64]string{}
	for _, tr := range likes {
		u := normIncomingSCURL(tr.PermalinkURL)
		key := incomingSCMatchKey(tr.User.Username, tr.Title)
		if tr.ID != 0 {
			idToKey[tr.ID] = key
		}
		if u != "" {
			urlToMeta[u] = incomingSCMeta{artist: tr.User.Username, title: tr.Title}
		}
		if _, ok := keyToMeta[key]; !ok {
			keyToMeta[key] = incomingSCMeta{artist: tr.User.Username, title: tr.Title}
		}
	}

	sets, err := ytdlpFlatEntries(nil, setsURL, oauth, "")
	if err != nil {
		return nil, nil, err
	}
	fmt.Fprintf(os.Stderr, "evoplayer: found %d playlists\n", len(sets))

	for _, pl := range sets {
		title := strings.TrimSpace(pl.Title)
		plURL := strings.TrimSpace(pl.URL)
		if plURL == "" {
			plURL = strings.TrimSpace(pl.WebpageURL)
		}
		if title == "" || plURL == "" {
			continue
		}
		folder := config.PlaylistFolder(env.MusicConfig, title)
		if folder == "" {
			folder = library.MatchLibraryGenre(env, title)
		}
		if folder == "" {
			folder = guessIncomingSCFolder(title)
		}
		playlistFolder[title] = folder

		tracks, err := ytdlpFlatEntries(nil, plURL, oauth, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "evoplayer: warn: playlist %q: %v\n", title, err)
			continue
		}
		hits := 0
		for _, tr := range tracks {
			trackURL := strings.TrimSpace(tr.URL)
			if trackURL == "" {
				trackURL = strings.TrimSpace(tr.WebpageURL)
			}
			key, ok := lookupIncomingSCPlaylistTrack(trackURL, idToKey, urlToMeta)
			if !ok {
				continue
			}
			hits++
			meta := keyToMeta[key]
			meta.playlists = appendUniqueString(meta.playlists, title)
			if meta.folder == "" && folder != "" {
				meta.folder = folder
			}
			keyToMeta[key] = meta
		}
		fmt.Fprintf(os.Stderr, "evoplayer: %q: %d tracks, %d matched likes -> %q\n", title, len(tracks), hits, folder)
	}
	return playlistFolder, keyToMeta, nil
}

func lookupIncomingSCPlaylistTrack(rawURL string, idToKey map[int64]string, urlToMeta map[string]incomingSCMeta) (string, bool) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", false
	}
	if m := scTrackIDRe.FindStringSubmatch(rawURL); len(m) == 2 {
		var id int64
		fmt.Sscanf(m[1], "%d", &id)
		if key, ok := idToKey[id]; ok {
			return key, true
		}
	}
	u := normIncomingSCURL(rawURL)
	if meta, ok := urlToMeta[u]; ok && u != "" {
		return incomingSCMatchKey(meta.artist, meta.title), true
	}
	return "", false
}

func normIncomingSCURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return strings.ToLower(strings.TrimRight(raw, "/"))
	}
	path := strings.TrimRight(strings.ToLower(u.Path), "/")
	if path == "" {
		return ""
	}
	return "https://soundcloud.com" + path
}

func appendUniqueString(ss []string, s string) []string {
	for _, x := range ss {
		if x == s {
			return ss
		}
	}
	return append(ss, s)
}

func incomingSCMatchKey(artist, title string) string {
	artist = normIncomingSCMatch(artist)
	title = normIncomingSCMatch(title)
	if artist == "" && title == "" {
		return ""
	}
	return artist + "\x00" + title
}

func normIncomingSCMatch(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "&", "and")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func guessIncomingSCFolder(playlistTitle string) string {
	n := library.NormalizeGenreKey(playlistTitle)
	switch {
	case strings.Contains(n, "drumandbass"), strings.Contains(n, "dnb"), n == "jungle", n == "liquid", n == "neurofunk":
		return "drum&bass"
	case strings.Contains(n, "dubstep"), n == "140", n == "dub", strings.Contains(n, "riddim"):
		return "dubstep"
	case strings.Contains(n, "garage"), n == "ukg", strings.Contains(n, "2step"):
		return "garage"
	case strings.Contains(n, "grime"):
		return "grime"
	case strings.Contains(n, "hiphop"), strings.Contains(n, "rap"), strings.Contains(n, "phonk"), n == "trap", n == "beats":
		return "hiphop"
	case strings.Contains(n, "house"):
		return "house"
	case strings.Contains(n, "electronic"), strings.Contains(n, "edm"), strings.Contains(n, "techno"), strings.Contains(n, "chill"):
		return "electronic"
	case strings.Contains(n, "calibre"), strings.Contains(n, "metalheadz"), strings.Contains(n, "doc"), strings.Contains(n, "skeptical"), strings.Contains(n, "marcus"):
		return "drum&bass"
	case strings.Contains(n, "oldskool"):
		return "drum&bass"
	case strings.Contains(n, "spacejazz"):
		return "drum&bass"
	}
	return ""
}

func lookupIncomingSCMeta(byKey map[string]incomingSCMeta, artist, title, filename, ext string) (incomingSCMeta, bool) {
	if m, ok := byKey[incomingSCMatchKey(artist, title)]; ok {
		return m, true
	}
	if i := strings.Index(filename, " - "); i > 0 {
		fnArtist := strings.TrimSpace(filename[:i])
		fnTitle := strings.TrimSpace(strings.TrimSuffix(filename[i+3:], ext))
		if m, ok := byKey[incomingSCMatchKey(fnArtist, fnTitle)]; ok {
			return m, true
		}
	}
	return incomingSCMeta{}, false
}

func collectIncomingSCRows(env library.Env, byKey map[string]incomingSCMeta) []incomingSCRow {
	incoming := filepath.Join(env.MusicRoot, ".incoming")
	entries, err := os.ReadDir(incoming)
	if err != nil {
		return nil
	}

	var rows []incomingSCRow
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(incoming, e.Name())
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".mp3" && ext != ".flac" && ext != ".m4a" {
			continue
		}
		probed, _ := tags.ProbeImport(path)
		artist := strings.TrimSpace(probed.Tag.Artist)
		title := strings.TrimSpace(probed.Tag.Title)
		genre := strings.TrimSpace(probed.Tag.Genre)
		preview := library.PreviewIncoming(env, path)
		currentFolder := fmt.Sprint(preview["folder"])

		meta, ok := lookupIncomingSCMeta(byKey, artist, title, e.Name(), ext)
		status := "manual"
		folder := currentFolder
		var pls []string
		if ok {
			pls = meta.playlists
			if folder == "" && meta.folder != "" {
				folder = meta.folder
				status = "sc playlist"
			} else if folder != "" {
				status = "ready"
			} else if len(pls) > 0 {
				status = "sc playlist (unmapped)"
			}
		} else if folder != "" {
			status = "ready"
		}

		rows = append(rows, incomingSCRow{
			file: e.Name(), artist: artist, title: title, genre: genre,
			playlists: pls, folder: folder, status: status,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].status != rows[j].status {
			return rows[i].status < rows[j].status
		}
		if rows[i].folder != rows[j].folder {
			return rows[i].folder < rows[j].folder
		}
		return rows[i].file < rows[j].file
	})
	return rows
}

func writeIncomingSCReview(outPath, musicRoot, setsURL string, playlistFolder map[string]string, rows []incomingSCRow) error {
	var b strings.Builder
	fmt.Fprintf(&b, "evoplayer .incoming import review (soundcloud playlists)\n")
	fmt.Fprintf(&b, "generated: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(&b, "profile: %s\n", setsURL)
	fmt.Fprintf(&b, "music root: %s\n", musicRoot)
	fmt.Fprintf(&b, "total files: %d\n\n", len(rows))

	ready, scSuggested := 0, 0
	for _, r := range rows {
		if r.status == "ready" {
			ready++
		}
		if strings.HasPrefix(r.status, "sc playlist") {
			scSuggested++
		}
	}
	fmt.Fprintf(&b, "SUMMARY\n-------\n")
	fmt.Fprintf(&b, "ready (existing genre tag): %d\n", ready)
	fmt.Fprintf(&b, "suggested (sc playlist):  %d\n", scSuggested)
	fmt.Fprintf(&b, "needs manual:              %d\n\n", len(rows)-ready-scSuggested)

	fmt.Fprintf(&b, "PLAYLIST -> FOLDER MAP\n--------------------\n")
	plTitles := make([]string, 0, len(playlistFolder))
	for t := range playlistFolder {
		plTitles = append(plTitles, t)
	}
	sort.Strings(plTitles)
	for _, t := range plTitles {
		f := playlistFolder[t]
		if f == "" {
			f = "(unmapped)"
		}
		fmt.Fprintf(&b, "  %q -> %s\n", t, f)
	}
	fmt.Fprintf(&b, "\n")

	for _, sec := range []struct {
		title string
		match func(incomingSCRow) bool
	}{
		{"READY (genre tag)", func(r incomingSCRow) bool { return r.status == "ready" }},
		{"SUGGESTED (soundcloud playlist)", func(r incomingSCRow) bool { return strings.HasPrefix(r.status, "sc playlist") }},
		{"NEEDS MANUAL", func(r incomingSCRow) bool { return r.status == "manual" }},
	} {
		var group []incomingSCRow
		for _, r := range rows {
			if sec.match(r) {
				group = append(group, r)
			}
		}
		if len(group) == 0 {
			continue
		}
		fmt.Fprintf(&b, "%s (%d)\n%s\n", sec.title, len(group), strings.Repeat("-", len(sec.title)+5))
		for _, r := range group {
			fmt.Fprintf(&b, "file:      %s\n", r.file)
			if r.artist != "" || r.title != "" {
				fmt.Fprintf(&b, "track:     %s — %s\n", r.artist, r.title)
			}
			fmt.Fprintf(&b, "genre tag: %q\n", r.genre)
			if len(r.playlists) > 0 {
				fmt.Fprintf(&b, "playlist:  %s\n", strings.Join(r.playlists, ", "))
			}
			if r.folder != "" {
				fmt.Fprintf(&b, "folder:    %s\n", r.folder)
			} else {
				fmt.Fprintf(&b, "folder:    (none)\n")
			}
			fmt.Fprintln(&b)
		}
	}
	return os.WriteFile(outPath, []byte(b.String()), 0o644)
}
