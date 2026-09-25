package library

import "testing"

func TestGuessFolderName(t *testing.T) {
	cases := map[string]string{
		"Deep Dubstep":  "dubstep",
		"Jungle / DnB":  "drum&bass",
		"Pop":           "hiphop",
		"Innamind":      "dubstep",
		"drum & bass":   "drum&bass",
		"UK Funky/Breaks": "garage",
	}
	for in, want := range cases {
		if got := GuessFolderName(in); got != want {
			t.Fatalf("GuessFolderName(%q) = %q, want %q", in, got, want)
		}
	}
}
