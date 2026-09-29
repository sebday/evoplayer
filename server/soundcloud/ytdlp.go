package soundcloud

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/syncarchive"
	"github.com/sebday/evoplayer/server/ytdlp"
)

func isDRMError(err error) bool {
	return errors.Is(err, ytdlp.ErrDRM)
}

// ytDlpArgs returns SoundCloud flags; cleanup removes the netrc holding the oauth token.
func ytDlpArgs(oauthToken, clientID string) ([]string, func(), error) {
	extractorArgs := "soundcloud:formats=*_aac,*_mp3"
	if id := strings.TrimSpace(clientID); id != "" {
		extractorArgs = fmt.Sprintf("soundcloud:client_id=%s;formats=*_aac,*_mp3", id)
	}
	args := []string{
		"--no-warnings",
		"--use-extractors", "soundcloud.*",
		"--extractor-args", extractorArgs,
	}
	tok := strings.TrimSpace(oauthToken)
	if tok == "" {
		return args, func() {}, nil
	}
	netrc, cleanup, err := ytdlp.Netrc("soundcloud", "oauth", tok)
	if err != nil {
		return nil, nil, err
	}
	return append(args, netrc...), cleanup, nil
}

func cookieOrder(oauthToken string) []string {
	if strings.TrimSpace(oauthToken) != "" {
		return ytdlp.Browsers("")
	}
	return ytdlp.Browsers("brave", "chromium")
}

func downloadYtDlp(ctx context.Context, pageURL, dest, oauthToken, clientID string) error {
	bin, err := ytdlp.Bin()
	if err != nil {
		return err
	}
	base, cleanup, err := ytDlpArgs(oauthToken, clientID)
	if err != nil {
		return err
	}
	defer cleanup()
	tmpDir, err := os.MkdirTemp(filepath.Dir(dest), ".evoplayer-dl-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	out := filepath.Join(tmpDir, "%(id)s.%(ext)s")
	var lastErr error
	for _, browser := range cookieOrder(oauthToken) {
		args := slices.Concat(base, ytdlp.CookieArgs(browser), []string{"--no-playlist", "-f", "ba", "-o", out, "--", pageURL})
		if lastErr = ytdlp.Run(ctx, bin, args, nil); lastErr == nil {
			return finalizeYtDlpOutput(ctx, tmpDir, dest)
		}
		if isDRMError(lastErr) || ctx.Err() != nil {
			return lastErr
		}
	}
	return lastErr
}

func finalizeYtDlpOutput(ctx context.Context, dir, dest string) error {
	var src string
	for _, f := range ytDlpOutputFiles(dir) {
		src = f
	}
	if src == "" {
		return fmt.Errorf("yt-dlp wrote no audio")
	}
	return placeAudio(ctx, src, dest)
}

// ytDlpOutputFiles maps each finished download in dir by the id prefix of its name.
func ytDlpOutputFiles(dir string) map[string]string {
	entries, _ := os.ReadDir(dir)
	out := make(map[string]string, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasSuffix(name, ".part") || strings.HasSuffix(name, ".ytdl") {
			continue
		}
		id, _, _ := strings.Cut(name, ".")
		out[id] = filepath.Join(dir, name)
	}
	return out
}

func placeAudio(ctx context.Context, src, dest string) error {
	if strings.EqualFold(filepath.Ext(src), ".mp3") {
		return moveFile(src, dest)
	}
	return ytdlp.ToMP3(ctx, src, dest, nil)
}

func downloadYtDlpCollection(ctx context.Context, pageURL, incomingDir, oauthToken, clientID string, archive *syncarchive.Archive, rep jobs.Reporter) ([]string, error) {
	rep.Progress(jobs.Progress{Phase: "fetching track list"})
	rep.Line(jobs.LogInfo("fetching track list"))
	stopBeat := startProgressHeartbeat(ctx, rep, "fetching track list")
	entries, err := ytdlpFlatEntries(ctx, pageURL, oauthToken, clientID)
	stopBeat()
	if err != nil {
		return nil, err
	}
	rep.Line(jobs.LogInfof("%d tracks in collection", len(entries)))
	pending := make([]ytdlpFlatEntry, 0, len(entries))
	archived := 0
	for _, e := range entries {
		if archiveHasFlatEntry(archive, e) {
			archived++
			continue
		}
		if dest := flatDestPath(incomingDir, e); fileExists(dest) {
			_ = archiveAddFlatEntry(archive, e)
			rep.Line(jobs.LogSkip(filepath.Base(dest)))
			continue
		}
		pending = append(pending, e)
	}
	if archived > 0 {
		rep.Line(jobs.LogInfof("%d already archived", archived))
	}
	if len(pending) == 0 {
		return nil, nil
	}
	rep.Line(jobs.LogInfof("%d tracks to download", len(pending)))

	tmpDir, err := os.MkdirTemp(incomingDir, ".evoplayer-dl-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)
	total := len(pending)
	rep.Progress(jobs.Progress{Phase: "downloading tracks", Done: 0, Total: total})
	batchErr := downloadYtDlpBatch(ctx, tmpDir, pending, oauthToken, clientID, rep)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if batchErr != nil {
		rep.Line(jobs.LogWarn(batchErr.Error()))
	}

	files := ytDlpOutputFiles(tmpDir)
	var added []string
	done := 0
	for _, e := range pending {
		if err := ctx.Err(); err != nil {
			return added, err
		}
		title := strings.TrimSpace(e.Title)
		if title == "" {
			title = e.ID
		}
		src, ok := files[strings.TrimSpace(e.ID)]
		if !ok {
			rep.Line(jobs.LogFail(fmt.Sprintf("%s (not downloaded)", title)))
			rep.Progress(jobs.Progress{Phase: "downloading tracks", Done: done, Total: total})
			continue
		}
		dest := flatDestPath(incomingDir, e)
		if err := placeAudio(ctx, src, dest); err != nil {
			rep.Line(jobs.LogFail(fmt.Sprintf("%s (%v)", title, err)))
			continue
		}
		added = append(added, dest)
		if err := archiveAddFlatEntry(archive, e); err != nil {
			rep.Line(jobs.LogWarn(fmt.Sprintf("archive write failed: %v", err)))
		}
		done++
		msg := jobs.LogOK(filepath.Base(dest))
		rep.Line(msg)
		rep.Progress(jobs.Progress{Phase: msg, Done: done, Total: total})
	}
	if len(added) == 0 && batchErr != nil {
		return nil, batchErr
	}
	return added, nil
}

// downloadYtDlpBatch fetches pending entries into dir, moving to the next cookie
// source only when an attempt made no progress or could not read cookies.
func downloadYtDlpBatch(ctx context.Context, dir string, pending []ytdlpFlatEntry, oauthToken, clientID string, rep jobs.Reporter) error {
	bin, err := ytdlp.Bin()
	if err != nil {
		return err
	}
	base, cleanup, err := ytDlpArgs(oauthToken, clientID)
	if err != nil {
		return err
	}
	defer cleanup()
	out := filepath.Join(dir, "%(id)s.%(ext)s")
	total := len(pending)
	onLine := func(line string) {
		if phase := ytdlpProgressPhase(line); phase != "" {
			rep.Progress(jobs.Progress{Phase: phase, Total: total})
		}
	}
	var lastErr error
	for _, browser := range cookieOrder(oauthToken) {
		missing := missingFlatEntries(dir, pending)
		if len(missing) == 0 {
			return nil
		}
		urlFile, cleanupURLs, err := writePendingURLFile(missing)
		if err != nil {
			return err
		}
		rep.Line(jobs.LogInfof("yt-dlp batch (%d urls)", len(missing)))
		args := slices.Concat(base, ytdlp.CookieArgs(browser), []string{"-f", "ba", "-o", out, "--newline", "--progress", "-a", urlFile})
		lastErr = ytdlp.Run(ctx, bin, args, onLine)
		cleanupURLs()
		if lastErr == nil || ctx.Err() != nil {
			return lastErr
		}
		if !isRetryableYtDlpErr(lastErr) && len(missingFlatEntries(dir, missing)) < len(missing) {
			return lastErr
		}
	}
	return lastErr
}

func missingFlatEntries(dir string, entries []ytdlpFlatEntry) []ytdlpFlatEntry {
	files := ytDlpOutputFiles(dir)
	out := make([]ytdlpFlatEntry, 0, len(entries))
	for _, e := range entries {
		if _, ok := files[strings.TrimSpace(e.ID)]; !ok {
			out = append(out, e)
		}
	}
	return out
}

func isRetryableYtDlpErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "cookies database") ||
		strings.Contains(msg, "could not find") && strings.Contains(msg, "cookie")
}

func ytdlpProgressPhase(line string) string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "[download]") {
		return ""
	}
	phase := strings.TrimSpace(strings.TrimPrefix(line, "[download]"))
	if len(phase) > 72 {
		phase = phase[:72]
	}
	return phase
}

func writePendingURLFile(pending []ytdlpFlatEntry) (string, func(), error) {
	f, err := os.CreateTemp("", "evoplayer-sc-urls-*.txt")
	if err != nil {
		return "", func() {}, err
	}
	path := f.Name()
	cleanup := func() { _ = os.Remove(path) }
	for _, e := range pending {
		if page := flatEntryURL(e); page != "" {
			if _, err := io.WriteString(f, page+"\n"); err != nil {
				f.Close()
				cleanup()
				return "", func() {}, err
			}
		}
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return path, cleanup, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func startProgressHeartbeat(ctx context.Context, rep jobs.Reporter, phase string) func() {
	done := make(chan struct{})
	started := time.Now()
	rep.Progress(jobs.Progress{Phase: phase})
	go func() {
		tick := time.NewTicker(2 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-tick.C:
				elapsed := int(time.Since(started).Seconds())
				rep.Progress(jobs.Progress{Phase: fmt.Sprintf("%s (%ds)", phase, elapsed)})
			}
		}
	}()
	return func() { close(done) }
}
