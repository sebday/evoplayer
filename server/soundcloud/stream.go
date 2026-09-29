package soundcloud

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sebday/evoplayer/server/ytdlp"
)

func (c *Client) DownloadTrackStream(ctx context.Context, track *Track, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	order := orderedTranscodings(track)
	if len(order) == 0 {
		for _, tc := range track.Media.Transcodings {
			if strings.Contains(tc.Format.Protocol, "encrypted") {
				return ytdlp.ErrDRM
			}
		}
		return fmt.Errorf("soundcloud: no transcodings for track %d", track.ID)
	}
	var lastErr error
	for _, tc := range order {
		if err := ctx.Err(); err != nil {
			return err
		}
		if lastErr = c.downloadTranscoding(ctx, tc, destPath); lastErr == nil {
			return nil
		}
	}
	return lastErr
}

func (c *Client) downloadTranscoding(ctx context.Context, tc Transcoding, destPath string) error {
	streamURL, err := c.streamInfoURL(tc.URL)
	if err != nil {
		return err
	}
	if tc.Format.Protocol == "hls" {
		return ytdlp.ToMP3(ctx, streamURL, destPath, nil)
	}
	tmp := destPath + ".part"
	if err := downloadFile(ctx, c.HTTP, streamURL, tmp); err != nil {
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
