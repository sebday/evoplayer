package lastfm

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const apiBase = "https://ws.audioscrobbler.com/2.0/"

func AuthSession(apiKey, secret, token string) (string, error) {
	sig := md5Hex("api_key" + apiKey + "methodauth.getSessiontoken" + token + secret)
	form := url.Values{}
	form.Set("method", "auth.getSession")
	form.Set("api_key", apiKey)
	form.Set("token", token)
	form.Set("api_sig", sig)
	form.Set("format", "json")
	body, err := postForm(apiBase, form)
	if err != nil {
		return "", err
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if session, ok := payload["session"].(map[string]any); ok {
		if key, ok := session["key"].(string); ok && key != "" {
			return key, nil
		}
	}
	errMsg := fmt.Sprint(payload["error"])
	if msg, ok := payload["message"].(string); ok && msg != "" {
		errMsg = msg
	}
	if errMsg == "" {
		errMsg = "auth failed"
	}
	return "", fmt.Errorf("evoplayer: %s", errMsg)
}

type ScrobbleParams struct {
	Method      string
	APIKey      string
	Secret      string
	Session     string
	Artist      string
	Title       string
	Album       string
	Duration    string
	Timestamp   string
	AlbumArtist string
	MBID        string
}

func APICall(p ScrobbleParams) error {
	params := map[string]string{
		"method":  p.Method,
		"api_key": p.APIKey,
		"sk":      p.Session,
		"artist":  p.Artist,
		"track":   p.Title,
	}
	if p.Album != "" {
		params["album"] = p.Album
	}
	if p.AlbumArtist != "" && p.AlbumArtist != p.Artist {
		params["albumArtist"] = p.AlbumArtist
	}
	if dur, err := strconv.ParseFloat(p.Duration, 64); err == nil && dur > 0 {
		params["duration"] = strconv.Itoa(int(dur))
	}
	if p.Timestamp != "" {
		if ts, err := strconv.ParseFloat(p.Timestamp, 64); err == nil {
			params["timestamp"] = strconv.Itoa(int(ts))
		}
	}
	if p.MBID != "" {
		params["mbid"] = p.MBID
	}
	if p.Method == "track.scrobble" {
		params["chosenByUser"] = "1"
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sigSrc := strings.Builder{}
	for _, k := range keys {
		sigSrc.WriteString(k)
		sigSrc.WriteString(params[k])
	}
	sigSrc.WriteString(p.Secret)
	params["api_sig"] = md5Hex(sigSrc.String())
	params["format"] = "json"

	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	body, err := postForm(apiBase, form)
	if err != nil {
		return fmt.Errorf("evoplayer: last.fm request failed: %w", err)
	}
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if errCode, ok := resp["error"]; ok && errCode != nil {
		msg, _ := resp["message"].(string)
		return fmt.Errorf("evoplayer: last.fm error %v: %s", errCode, msg)
	}
	if p.Method == "track.scrobble" {
		scrobblesBody, _ := resp["scrobbles"].(map[string]any)
		raw := scrobblesBody["scrobble"]
		var rows []map[string]any
		switch v := raw.(type) {
		case map[string]any:
			rows = []map[string]any{v}
		case []any:
			for _, item := range v {
				if m, ok := item.(map[string]any); ok {
					rows = append(rows, m)
				}
			}
		}
		for _, row := range rows {
			if attr, ok := row["@attr"].(map[string]any); ok {
				if msg, ok := attr["ignoredMessage"].(string); ok && msg != "" {
					return fmt.Errorf("evoplayer: last.fm ignored scrobble: %s", msg)
				}
			}
		}
	}
	return nil
}

func RecordingMBID(artist, title, album string) (string, error) {
	terms := []string{`artist:"` + artist + `"`, `recording:"` + title + `"`}
	if album != "" {
		terms = append(terms, `release:"`+album+`"`)
	}
	q := url.Values{}
	q.Set("query", strings.Join(terms, " AND "))
	q.Set("fmt", "json")
	q.Set("limit", "1")
	req, err := http.NewRequest(http.MethodGet, "https://musicbrainz.org/ws/2/recording/?"+q.Encode(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "evoplayer/1.0 (local scrobbler)")
	time.Sleep(time.Second)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	recordings, _ := data["recordings"].([]any)
	for _, item := range recordings {
		rec, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if id, ok := rec["id"].(string); ok && id != "" {
			return id, nil
		}
	}
	return "", errors.New("not found")
}

func postForm(endpoint string, form url.Values) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}
