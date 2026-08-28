package download

import "testing"

func TestDetectSource(t *testing.T) {
	cases := []struct {
		url    string
		source string
	}{
		{"https://www.youtube.com/watch?v=sBamYpy-wIU", "youtube"},
		{"https://youtu.be/sBamYpy-wIU", "youtube"},
		{"https://soundcloud.com/artist/track", "soundcloud"},
		{"https://example.com/foo", ""},
	}
	for _, c := range cases {
		got := DetectSource(c.url)
		if got != c.source {
			t.Fatalf("DetectSource(%q) = %q, want %q", c.url, got, c.source)
		}
	}
}
