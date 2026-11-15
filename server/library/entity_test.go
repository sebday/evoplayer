package library

import (
	"reflect"
	"testing"
)

func TestArtistCredits(t *testing.T) {
	cases := map[string][]string{
		"Ed Rush & Optical":        {"Ed Rush & Optical", "Ed Rush", "Optical"},
		"Skream":                   {"Skream"},
		"Wiley feat. Kano, Ghetts": {"Wiley feat. Kano, Ghetts", "Wiley", "Kano", "Ghetts"},
		"DJ Hype vs Pascal":        {"DJ Hype vs Pascal", "DJ Hype", "Pascal"},
		"  Calibre FT. DRS ":       {"Calibre FT. DRS", "Calibre", "DRS"},
		"":                         nil,
		"Bailey & GQ (MC)":         {"Bailey & GQ (MC)", "Bailey", "GQ (MC)"},
		"Foo (feat. Bar)":          {"Foo (feat. Bar)", "Foo", "Bar"},
	}
	for in, want := range cases {
		if got := ArtistCredits(in); !reflect.DeepEqual(got, want) {
			t.Errorf("ArtistCredits(%q) = %#v, want %#v", in, got, want)
		}
	}
}

func TestIndexSortKey(t *testing.T) {
	cases := map[string]string{
		"\"FIRED UP\"": "fired up\"",
		"$AW":          "aw",
		"The Prodigy":  "prodigy",
		"Aceyalone":    "aceyalone",
	}
	for in, want := range cases {
		if got := indexSortKey(in); got != want {
			t.Errorf("indexSortKey(%q) = %q, want %q", in, got, want)
		}
	}
	if indexSortKey("...") <= indexSortKey("zz top") {
		t.Fatal("names with no letters should sort last")
	}
}

func TestCreditedTo(t *testing.T) {
	if !creditedTo("Ed Rush & Optical", "optical") {
		t.Fatal("a duo credit should count for each member")
	}
	if !creditedTo("Ed Rush & Optical", "ed rush & optical") {
		t.Fatal("the whole credit should match itself")
	}
	if creditedTo("Opticality", "optical") {
		t.Fatal("a substring is not a credit")
	}
}
