package soundcloud

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sebday/evoplayer/server/jobs"
	"github.com/sebday/evoplayer/server/syncarchive"
)

func isDRMError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "drm protected") ||
		strings.Contains(msg, "exit status 183") ||
		strings.Contains(msg, "exit status 234")
}

func isFFmpegDecryptErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "exit status 183") || strings.Contains(msg, "exit status 234")
}

func ytDlpSoundcloudBaseArgs(oauthToken, clientID, outTemplate string, single bool) []string {
	extractorArgs := "soundcloud:formats=*_aac,*_mp3"
	if strings.TrimSpace(clientID) != "" {
		extractorArgs = fmt.Sprintf("soundcloud:client_id=%s;formats=*_aac,*_mp3", strings.TrimSpace(clientID))
	}
	args := []string{
		"--no-warnings",
		"--use-extractors", "soundcloud.*",
		"--extractor-args", extractorArgs,
		"-f", "ba",
		"-o", outTemplate,
	}
	if single {
		args = append([]string{"--no-playlist"}, args...)
	}
	if tok := strings.TrimSpace(oauthToken); tok != "" {
		args = append(args, "-u", "oauth", "-p", tok)
	}
	return args
}

func ytDlpSoundcloudArgs(oauthToken, clientID, outTemplate, pageURL string, single bool) []string {
	args := ytDlpSoundcloudBaseArgs(oauthToken, clientID, outTemplate, single)
	if pageURL != "" {
		args = append(args, pageURL)
	}
	return args
}

func downloadYtDlp(ctx context.Context, pageURL, dest, oauthToken, clientID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	bin, err := exec.LookPath("yt-dlp")
	if err != nil {
		return err
	}
	dir := filepath.Dir(dest)
	base := strings.TrimSuffix(filepath.Base(dest), filepath.Ext(dest))
	out := filepath.Join(dir, base+".%(ext)s")

	try := func(extra []string) error {
		args := ytDlpSoundcloudArgs(oauthToken, clientID, out, pageURL, true)
		if len(extra) > 0 {
			args = append(extra, args...)
		}
		cmd := exec.CommandContext(ctx, bin, args...)
		outBytes, err := cmd.CombinedOutput()
		if err != nil {
			text := strings.ToLower(string(outBytes))
			if strings.Contains(text, "drm protected") {
				return fmt.Errorf("drm protected")
			}
			if errText := strings.TrimSpace(string(outBytes)); errText != "" {
				return fmt.Errorf("yt-dlp: %s", lastNonEmptyLine(errText))
			}
			return err
		}
		return finalizeYtDlpOutput(ctx, dir, base, dest)
	}

	if strings.TrimSpace(oauthToken) != "" {
		if err := try(nil); err == nil {
			return nil
		} else if isDRMError(err) {
			return err
		}
	}
	for _, browser := range []string{"brave", "chromium"} {
		if err := try([]string{"--cookies-from-browser", browser}); err == nil {
			return nil
		} else if isDRMError(err) {
			return err
		}
	}
	return try(nil)
}

func finalizeYtDlpOutput(ctx context.Context, dir, base, dest string) error {
	matches, _ := filepath.Glob(filepath.Join(dir, base+".*"))
	for _, f := range matches {
		if strings.HasSuffix(f, ".part") {
			continue
		}
		if f == dest {
			return nil
		}
		if strings.EqualFold(filepath.Ext(f), ".mp3") {
			if err := os.Rename(f, dest); err == nil {
				return nil
			}
		}
		if err := ffmpegToMP3(ctx, f, dest); err == nil {
			_ = os.Remove(f)
			return nil
		}
	}
	return fmt.Errorf("yt-dlp wrote no audio")
}

func lastNonEmptyLine(text string) string {
	lines := strings.Split(text, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			return line
		}
	}
	return strings.TrimSpace(text)
}

func downloadYtDlpCollection(ctx context.Context, pageURL, incomingDir, oauthToken, clientID string, archive *syncarchive.Archive, rep jobs.Reporter) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if rep != nil {
		rep.Progress(jobs.Progress{Phase: "fetching track list"})
		rep.Line(jobs.LogInfo("fetching track list"))
	}
	stopBeat := startProgressHeartbeat(ctx, rep, "fetching track list")
	defer stopBeat()
	entries, err := ytdlpFlatEntries(ctx, pageURL, oauthToken, clientID)
	if err != nil {
		return nil, err
	}
	if rep != nil {
		rep.Line(jobs.LogInfof("%d tracks in collection", len(entries)))
	}
	pending := make([]ytdlpFlatEntry, 0, len(entries))
	archived := 0
	for _, e := range entries {
		if archive != nil && archiveHasFlatEntry(archive, e) {
			archived++
			continue
		}
		pending = append(pending, e)
	}
	if archived > 0 && rep != nil {
		rep.Line(jobs.LogInfof("%d already archived", archived))
	}
	if len(pending) == 0 {
		return nil, nil
	}
	stillPending := make([]ytdlpFlatEntry, 0, len(pending))
	for _, e := range pending {
		if dest := locateFlatDownload(incomingDir, e); dest != "" {
			if archive != nil {
				_ = archiveAddFlatEntry(archive, e)
			}
			if rep != nil {
				rep.Line(jobs.LogSkip(filepath.Base(dest)))
			}
			continue
		}
		stillPending = append(stillPending, e)
	}
	pending = stillPending
	if len(pending) == 0 {
		return nil, nil
	}
	if rep != nil {
		rep.Line(jobs.LogInfof("%d tracks to download", len(pending)))
	}

	before, err := incomingFileSet(incomingDir)
	if err != nil {
		return nil, err
	}
	if rep != nil {
		rep.Progress(jobs.Progress{Phase: "downloading tracks", Done: 0, Total: len(pending)})
	}
	if err := downloadYtDlpBatch(ctx, incomingDir, pageURL, pending, len(pending) == len(entries), oauthToken, clientID, rep, len(pending)); err != nil {
		return nil, err
	}
	if err := normalizeIncomingBatch(ctx, incomingDir); err != nil {
		return nil, err
	}

	var added []string
	done := 0
	total := len(pending)
	for _, e := range pending {
		if err := ctx.Err(); err != nil {
			return added, err
		}
		dest := locateFlatDownload(incomingDir, e)
		label := filepath.Base(dest)
		if dest == "" {
			if rep != nil {
				title := strings.TrimSpace(e.Title)
				if title == "" {
					title = e.ID
				}
				rep.Line(jobs.LogFail(fmt.Sprintf("%s (not downloaded)", title)))
				rep.Progress(jobs.Progress{Phase: "downloading tracks", Done: done, Total: total})
			}
			continue
		}
		if !before[filepath.Base(dest)] {
			added = append(added, dest)
		}
		if archive != nil {
			if err := archiveAddFlatEntry(archive, e); err != nil && rep != nil {
				rep.Line(jobs.LogWarn(fmt.Sprintf("archive write failed: %v", err)))
			}
		}
		done++
		if rep != nil {
			msg := jobs.LogOK(label)
			rep.Line(msg)
			rep.Progress(jobs.Progress{Phase: msg, Done: done, Total: total})
		}
	}
	return added, nil
}

func downloadYtDlpBatch(ctx context.Context, incomingDir, pageURL string, pending []ytdlpFlatEntry, allInCollection bool, oauthToken, clientID string, rep jobs.Reporter, total int) error {
	if ctx == nil {
		ctx = context.Background()
	}
	bin, err := exec.LookPath("yt-dlp")
	if err != nil {
		return err
	}
	out := filepath.Join(incomingDir, "%(uploader)s - %(title)s.%(ext)s")
	run := func(extra []string, urlArg string, archiveFile bool) error {
		args := ytDlpSoundcloudBaseArgs(oauthToken, clientID, out, false)
		args = append(args, "--newline", "--progress")
		if archiveFile {
			args = append(args, "-a", urlArg)
		} else {
			args = append(args, urlArg)
		}
		if len(extra) > 0 {
			args = append(extra, args...)
		}
		cmd := exec.CommandContext(ctx, bin, args...)
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return err
		}
		var outBuf strings.Builder
		doneCh := make(chan struct{})
		go func() {
			defer close(doneCh)
			sc := bufio.NewScanner(stderr)
			for sc.Scan() {
				line := sc.Text()
				outBuf.WriteString(line)
				outBuf.WriteByte('\n')
				if rep != nil {
					if phase := ytdlpProgressPhase(line); phase != "" {
						rep.Progress(jobs.Progress{Phase: phase, Total: total})
					}
				}
			}
		}()
		if err := cmd.Start(); err != nil {
			return err
		}
		err = cmd.Wait()
		<-doneCh
		if err != nil {
			text := strings.ToLower(outBuf.String())
			if strings.Contains(text, "drm protected") {
				return fmt.Errorf("drm protected")
			}
			if errText := strings.TrimSpace(outBuf.String()); errText != "" {
				return fmt.Errorf("yt-dlp: %s", lastNonEmptyLine(errText))
			}
			return err
		}
		return nil
	}
	urlFile, cleanup, err := writePendingURLFile(pending)
	if err != nil {
		return err
	}
	defer cleanup()
	if rep != nil {
		rep.Line(jobs.LogInfof("yt-dlp batch (%d urls)", len(pending)))
	}

	type attempt struct {
		extra       []string
		urlArg      string
		archiveFile bool
	}
	attempts := []attempt{{nil, urlFile, true}}
	if allInCollection && strings.TrimSpace(pageURL) != "" {
		attempts = append([]attempt{{nil, pageURL, false}}, attempts...)
	}
	for _, browser := range []string{"brave", "chromium"} {
		extra := []string{"--cookies-from-browser", browser}
		if allInCollection && strings.TrimSpace(pageURL) != "" {
			attempts = append(attempts, attempt{extra, pageURL, false})
		}
		attempts = append(attempts, attempt{extra, urlFile, true})
	}

	var lastErr error
	for _, a := range attempts {
		if err := run(a.extra, a.urlArg, a.archiveFile); err == nil {
			return nil
		} else {
			lastErr = err
			if isRetryableYtDlpErr(err) {
				continue
			}
			if isDRMError(err) {
				return err
			}
		}
	}
	return lastErr
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

func locateFlatDownload(incoming string, e ytdlpFlatEntry) string {
	candidates := []string{
		flatDestPath(incoming, e),
		filepath.Join(incoming, strings.TrimSpace(e.Uploader)+" - "+strings.TrimSpace(e.Title)+".mp3"),
	}
	seen := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		c = strings.TrimSpace(c)
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		if _, err := os.Stat(c); err == nil {
			return c
		}
		base := strings.TrimSuffix(c, filepath.Ext(c))
		matches, _ := filepath.Glob(base + ".*")
		for _, m := range matches {
			if strings.HasSuffix(m, ".part") {
				continue
			}
			return m
		}
	}
	return ""
}

func normalizeIncomingBatch(ctx context.Context, incomingDir string) error {
	entries, err := os.ReadDir(incomingDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if e.IsDir() || strings.HasSuffix(e.Name(), ".part") {
			continue
		}
		f := filepath.Join(incomingDir, e.Name())
		if strings.EqualFold(filepath.Ext(f), ".mp3") {
			continue
		}
		base := strings.TrimSuffix(f, filepath.Ext(f))
		dest := base + ".mp3"
		if _, err := os.Stat(dest); err == nil {
			_ = os.Remove(f)
			continue
		}
		if err := ffmpegToMP3(ctx, f, dest); err != nil {
			continue
		}
		_ = os.Remove(f)
	}
	return nil
}

func startProgressHeartbeat(ctx context.Context, rep jobs.Reporter, phase string) func() {
	if rep == nil {
		return func() {}
	}
	if ctx == nil {
		ctx = context.Background()
	}
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

func incomingFileSet(dir string) (map[string]bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]bool{}, nil
		}
		return nil, err
	}
	set := make(map[string]bool, len(entries))
	for _, e := range entries {
		if e.IsDir() || strings.HasSuffix(e.Name(), ".part") {
			continue
		}
		set[e.Name()] = true
	}
	return set, nil
}
