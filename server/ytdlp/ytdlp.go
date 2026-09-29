package ytdlp

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var ErrDRM = errors.New("drm protected")

func Bin() (string, error) {
	return exec.LookPath("yt-dlp")
}

// Browsers returns cookie sources to try in order; "" means no browser cookies.
func Browsers(first ...string) []string {
	seen := map[string]bool{}
	var out []string
	for _, b := range append(first, "", "brave", "chromium") {
		if !seen[b] {
			seen[b] = true
			out = append(out, b)
		}
	}
	return out
}

func CookieArgs(browser string) []string {
	if browser == "" {
		return nil
	}
	return []string{"--cookies-from-browser", browser}
}

// Netrc writes credentials to a private netrc file so they stay off argv.
func Netrc(machine, login, password string) ([]string, func(), error) {
	f, err := os.CreateTemp("", "evoplayer-netrc-*")
	if err != nil {
		return nil, nil, err
	}
	path := f.Name()
	cleanup := func() { _ = os.Remove(path) }
	_, werr := fmt.Fprintf(f, "machine %s login %s password %s\n", machine, login, password)
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		cleanup()
		return nil, nil, werr
	}
	return []string{"--netrc", "--netrc-location", path}, cleanup, nil
}

// Run executes yt-dlp and passes each line of its combined output to onLine.
func Run(ctx context.Context, bin string, args []string, onLine func(string)) error {
	cmd := exec.CommandContext(ctx, bin, args...)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return err
	}
	var last, lastErr string
	drm := false
	sc := bufio.NewScanner(out)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		last = line
		if strings.HasPrefix(line, "ERROR:") {
			lastErr = line
			if strings.Contains(strings.ToLower(line), "drm protected") {
				drm = true
			}
		}
		if onLine != nil {
			onLine(line)
		}
	}
	_, _ = io.Copy(io.Discard, out)
	err = cmd.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err == nil {
		return nil
	}
	if drm {
		return ErrDRM
	}
	if lastErr == "" {
		lastErr = last
	}
	if lastErr == "" {
		return fmt.Errorf("yt-dlp: %w", err)
	}
	return fmt.Errorf("yt-dlp: %s", lastErr)
}

// Output executes yt-dlp and returns stdout, reporting stderr's last line on failure.
func Output(ctx context.Context, bin string, args []string) ([]byte, error) {
	out, err := exec.CommandContext(ctx, bin, args...).Output()
	if err == nil {
		return out, nil
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		if line := lastLine(string(exit.Stderr)); line != "" {
			if strings.Contains(strings.ToLower(line), "drm protected") {
				return nil, ErrDRM
			}
			return nil, fmt.Errorf("yt-dlp: %s", line)
		}
	}
	return nil, fmt.Errorf("yt-dlp: %w", err)
}

// ToMP3 converts src with ffmpeg; dest only appears once the conversion completes.
// onProgress, when set, receives the encoded position in seconds.
func ToMP3(ctx context.Context, src, dest string, onProgress func(sec float64)) error {
	tmp := dest + ".part"
	args := []string{"-y", "-hide_banner", "-loglevel", "error",
		"-i", src, "-codec:a", "libmp3lame", "-q:a", "0", "-f", "mp3"}
	if onProgress != nil {
		args = append(args, "-progress", "pipe:1", "-nostats")
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", append(args, tmp)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := runFFmpeg(cmd, onProgress)
	if err != nil {
		_ = os.Remove(tmp)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if line := lastLine(stderr.String()); line != "" {
			return fmt.Errorf("ffmpeg: %s", line)
		}
		return fmt.Errorf("ffmpeg: %w", err)
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func runFFmpeg(cmd *exec.Cmd, onProgress func(sec float64)) error {
	if onProgress == nil {
		return cmd.Run()
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		key, val, ok := strings.Cut(sc.Text(), "=")
		if !ok || key != "out_time_us" {
			continue
		}
		if us, err := strconv.ParseFloat(val, 64); err == nil && us > 0 {
			onProgress(us / 1e6)
		}
	}
	_, _ = io.Copy(io.Discard, stdout)
	return cmd.Wait()
}

func lastLine(text string) string {
	lines := strings.Split(text, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return ""
}
