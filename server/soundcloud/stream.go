package soundcloud

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (c *Client) DownloadTrackStream(ctx context.Context, track *Track, destPath string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	order := orderedTranscodings(track)
	if len(order) == 0 {
		return fmt.Errorf("soundcloud: no transcodings for track %d", track.ID)
	}
	var lastErr error
	drmFailed := false
	for _, tc := range order {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := c.downloadTranscoding(ctx, tc, destPath); err != nil {
			lastErr = err
			if strings.Contains(tc.Format.Protocol, "encrypted") && isFFmpegDecryptErr(err) {
				drmFailed = true
			}
			continue
		}
		return nil
	}
	if drmFailed {
		return fmt.Errorf("drm protected")
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("soundcloud: no transcodings for track %d", track.ID)
	}
	return lastErr
}

func (c *Client) downloadTranscoding(ctx context.Context, tc Transcoding, destPath string) error {
	streamURL, err := c.streamInfoURL(tc.URL)
	if err != nil {
		return err
	}
	tmp := destPath + ".part"
	if strings.Contains(tc.Format.Protocol, "hls") {
		if err := ffmpegToMP3(ctx, streamURL, tmp); err != nil {
			os.Remove(tmp)
			return err
		}
	} else if err := downloadFile(ctx, c.HTTP, streamURL, tmp); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, destPath); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

func downloadFile(ctx context.Context, client *http.Client, url, dest string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download %s: %s", url, resp.Status)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, resp.Body)
	closeErr := f.Close()
	if err != nil {
		os.Remove(dest)
		return err
	}
	return closeErr
}

func fetchBytes(client *http.Client, url string) ([]byte, string, error) {
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("fetch %s: %s", url, resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return nil, "", err
	}
	mime := resp.Header.Get("Content-Type")
	if i := strings.Index(mime, ";"); i > 0 {
		mime = mime[:i]
	}
	return body, strings.TrimSpace(mime), nil
}
