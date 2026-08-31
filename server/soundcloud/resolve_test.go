package soundcloud

import "testing"

func TestOrderedTranscodingsPrefersProgressive(t *testing.T) {
	track := &Track{
		ID: 1,
		Media: struct {
			Transcodings []Transcoding `json:"transcodings"`
		}{
			Transcodings: []Transcoding{
				{URL: "ctr", Format: struct {
					Protocol string `json:"protocol"`
					MimeType string `json:"mime_type"`
				}{Protocol: "ctr-encrypted-hls"}, Quality: "hq"},
				{URL: "hls", Format: struct {
					Protocol string `json:"protocol"`
					MimeType string `json:"mime_type"`
				}{Protocol: "hls"}},
				{URL: "prog", Format: struct {
					Protocol string `json:"protocol"`
					MimeType string `json:"mime_type"`
				}{Protocol: "progressive"}, Quality: "hq"},
				{URL: "prog-sd", Format: struct {
					Protocol string `json:"protocol"`
					MimeType string `json:"mime_type"`
				}{Protocol: "progressive"}},
			},
		},
	}
	order := orderedTranscodings(track)
	if len(order) != 4 {
		t.Fatalf("len = %d", len(order))
	}
	if order[0].URL != "prog" || order[1].URL != "prog-sd" || order[2].URL != "hls" || order[3].URL != "ctr" {
		t.Fatalf("order = %#v", order)
	}
	got, err := pickTranscoding(track)
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != "prog" {
		t.Fatalf("pick = %q", got.URL)
	}
}
