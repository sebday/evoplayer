package soundcloud

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const apiBase = "https://api-v2.soundcloud.com"

var (
	clientIDRe       = regexp.MustCompile(`client_id=([a-zA-Z0-9]+)`)
	clientIDJSRe     = regexp.MustCompile(`client_id:"([a-zA-Z0-9]+)"`)
	scriptAssetJSRe  = regexp.MustCompile(`https://a-v2\.sndcdn\.com/assets/[^"'\s]+\.js`)
)

type Client struct {
	ClientID   string
	OAuthToken string
	HTTP       *http.Client
}

func NewClient(clientID, oauthToken string) *Client {
	return &Client{
		ClientID:   strings.TrimSpace(clientID),
		OAuthToken: strings.TrimSpace(oauthToken),
		HTTP: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *Client) ensureClientID() error {
	if c.ClientID != "" {
		return nil
	}
	id, err := discoverClientID(c.HTTP)
	if err != nil {
		return err
	}
	c.ClientID = id
	return nil
}

func discoverClientID(client *http.Client) (string, error) {
	return discoverClientIDFrom("https://soundcloud.com/", client)
}

func discoverClientIDFrom(homeURL string, client *http.Client) (string, error) {
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Get(homeURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	if m := clientIDRe.FindSubmatch(body); len(m) > 1 {
		return string(m[1]), nil
	}
	seen := map[string]bool{}
	var scripts []string
	for _, raw := range scriptAssetJSRe.FindAll(body, -1) {
		u := string(raw)
		if seen[u] {
			continue
		}
		seen[u] = true
		scripts = append(scripts, u)
	}
	for i := len(scripts) - 1; i >= 0; i-- {
		if id, err := clientIDFromScript(client, scripts[i]); err == nil && id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("soundcloud: client_id not found")
}

func clientIDFromScript(client *http.Client, scriptURL string) (string, error) {
	resp, err := client.Get(scriptURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", err
	}
	if m := clientIDJSRe.FindSubmatch(body); len(m) > 1 {
		return string(m[1]), nil
	}
	return "", fmt.Errorf("soundcloud: client_id not found in %s", scriptURL)
}

func (c *Client) authHeader() string {
	if c.OAuthToken == "" {
		return ""
	}
	return "OAuth " + c.OAuthToken
}

func (c *Client) getJSON(url string) ([]byte, error) {
	if err := c.ensureClientID(); err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if auth := c.authHeader(); auth != "" {
		req.Header.Set("Authorization", auth)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if resp.StatusCode == http.StatusUnauthorized && c.OAuthToken != "" {
			if strings.Contains(url, "/me/") || strings.Contains(url, "/me?") {
				return nil, fmt.Errorf("soundcloud: oauth token expired or invalid (log in at soundcloud.com in Brave)")
			}
			c.OAuthToken = ""
			return c.getJSON(url)
		}
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("soundcloud: oauth token expired or invalid (log in at soundcloud.com in Brave)")
		}
		return nil, fmt.Errorf("soundcloud: GET %s: %s: %s", url, resp.Status, msg)
	}
	return body, nil
}

func (c *Client) getJSONWithClientID(path string) ([]byte, error) {
	if err := c.ensureClientID(); err != nil {
		return nil, err
	}
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	url := apiBase + path + sep + "client_id=" + c.ClientID
	return c.getJSON(url)
}

func (c *Client) getJSONPublic(url string) ([]byte, error) {
	if err := c.ensureClientID(); err != nil {
		return nil, err
	}
	if !strings.Contains(url, "client_id=") {
		sep := "?"
		if strings.Contains(url, "?") {
			sep = "&"
		}
		url = url + sep + "client_id=" + c.ClientID
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		return nil, fmt.Errorf("soundcloud: GET %s: %s: %s", url, resp.Status, msg)
	}
	return body, nil
}

func decodeJSON(body []byte, dst interface{}) error {
	if err := json.Unmarshal(body, dst); err != nil {
		return fmt.Errorf("soundcloud: decode: %w", err)
	}
	return nil
}
