package playlist

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestSortPlaylistsUsesDisplayNames(t *testing.T) {
	items := []IndexItem{
		{Name: "all"},
		{Name: "mixes"},
		{Name: "House"},
		{Name: "Calibre"},
		{Name: "current"},
	}
	sortPlaylists(items)
	var names []string
	for _, item := range items {
		names = append(names, item.Name)
	}
	want := []string{"Calibre", "House", "all", "mixes", "current"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("sorted = %v, want %v", names, want)
	}
}

func TestApplyPathOrderKeepsNewTracks(t *testing.T) {
	got := applyPathOrder([]string{"a", "b", "c", "d"}, []string{"c", "a", "gone"})
	want := []string{"c", "a", "b", "d"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestSetTrackOrderRoundTrip(t *testing.T) {
	env := Env{}
	env.StateDir = filepath.Join(t.TempDir(), "state")
	if err := SetTrackOrder(env, "Drum & Bass", []string{"/music/b.mp3", "/music/a.mp3"}); err != nil {
		t.Fatal(err)
	}
	got, err := orderedPaths(env, "Drum & Bass", []string{"/music/a.mp3", "/music/b.mp3", "/music/c.mp3"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/music/b.mp3", "/music/a.mp3", "/music/c.mp3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %v, want %v", got, want)
	}
}

func TestForgetTrackPathDropsSavedOrder(t *testing.T) {
	env := Env{}
	env.StateDir = filepath.Join(t.TempDir(), "state")
	if err := SetTrackOrder(env, "Drum & Bass", []string{"/music/b.mp3", "/music/a.mp3"}); err != nil {
		t.Fatal(err)
	}
	if err := SetTrackOrder(env, "House", []string{"/music/a.mp3", "/music/gone.mp3"}); err != nil {
		t.Fatal(err)
	}
	if err := forgetTrackPath(env, "/music/gone.mp3"); err != nil {
		t.Fatal(err)
	}
	got, err := orderedPaths(env, "Drum & Bass", []string{"/music/b.mp3", "/music/a.mp3"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"/music/b.mp3", "/music/a.mp3"}) {
		t.Fatalf("kept order = %v", got)
	}
	house, err := orderedPaths(env, "House", []string{"/music/a.mp3"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(house, []string{"/music/a.mp3"}) {
		t.Fatalf("house order = %v", house)
	}
}
