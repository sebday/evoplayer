package art

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// FetchImage downloads a cover image URL with headers Discogs expects.
func FetchImage(imageURL string) ([]byte, error) {
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return nil, fmt.Errorf("empty art url")
	}
	req, err := http.NewRequest(http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	if strings.Contains(imageURL, "discogs.com") {
		req.Header.Set("Referer", "https://www.discogs.com/")
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("art fetch %d", resp.StatusCode)
	}
	return body, nil
}
