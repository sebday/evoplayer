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

var clientIDRe = regexp.MustCompile(`client_id=([a-zA-Z0-9]+)`)

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
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Get("https://soundcloud.com/")
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
	return "", fmt.Errorf("soundcloud: client_id not found")
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
		return nil, fmt.Errorf("soundcloud: GET %s: %s: %s", url, resp.Status, strings.TrimSpace(string(body)))
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

func decodeJSON(body []byte, dst interface{}) error {
	if err := json.Unmarshal(body, dst); err != nil {
		return fmt.Errorf("soundcloud: decode: %w", err)
	}
	return nil
}
