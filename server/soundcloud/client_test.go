package soundcloud

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiscoverClientIDFromScriptBundles(t *testing.T) {
	const want = "Pb72ranhoyt6gw7hM7TkzUItXlMWSNSo"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/":
			_, _ = w.Write([]byte(`<script src="https://a-v2.sndcdn.com/assets/test.js"></script>`))
		case r.URL.Path == "/bundle.js":
			_, _ = w.Write([]byte(`,client_id:"` + want + `"`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := &http.Client{Transport: rewriteTransport(srv.URL)}
	got, err := discoverClientIDFrom(srv.URL+"/", client)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("client_id = %q", got)
	}
}

func TestDiscoverClientIDFallsBackToHomepageQuery(t *testing.T) {
	home := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<a href="https://api-v2.soundcloud.com/x?client_id=abc123XYZ">`))
	}))
	defer home.Close()

	got, err := discoverClientIDFrom(home.URL+"/", &http.Client{})
	if err != nil {
		t.Fatal(err)
	}
	if got != "abc123XYZ" {
		t.Fatalf("client_id = %q", got)
	}
}

func rewriteTransport(base string) http.RoundTripper {
	return roundTripper(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Host, "sndcdn.com") {
			req = req.Clone(req.Context())
			req.URL.Scheme = "http"
			req.URL.Host = strings.TrimPrefix(strings.TrimPrefix(base, "http://"), "https://")
			req.URL.Path = "/bundle.js"
		} else if strings.HasPrefix(req.URL.String(), base) {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, "/")
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
		}
		return http.DefaultTransport.RoundTrip(req)
	})
}

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
