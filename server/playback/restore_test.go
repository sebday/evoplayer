package playback

import "testing"

func TestRestoreSetsPausedTrackWithoutLoading(t *testing.T) {
	a := NewActor(nil)
	paths := []string{"/a.mp3", "/b.mp3", "/c.mp3"}
	if err := a.Restore(paths, "/b.mp3", 42); err != nil {
		t.Fatal(err)
	}
	st := a.Snapshot()
	if st.Path != "/b.mp3" {
		t.Fatalf("path %q", st.Path)
	}
	if st.State != "paused" {
		t.Fatalf("state %q", st.State)
	}
	if st.Position != 42 {
		t.Fatalf("position %v", st.Position)
	}
	if st.PlaylistPos != 1 {
		t.Fatalf("index %d", st.PlaylistPos)
	}
	if a.stream != nil {
		t.Fatal("restore should not open a decoder")
	}
}

func TestSeekRemembersPositionWithoutStream(t *testing.T) {
	a := NewActor(nil)
	if err := a.Restore([]string{"/a.mp3"}, "/a.mp3", 10); err != nil {
		t.Fatal(err)
	}
	if err := a.Seek(33); err != nil {
		t.Fatal(err)
	}
	st := a.Snapshot()
	if st.Position != 33 {
		t.Fatalf("position %v", st.Position)
	}
	if st.State != "paused" {
		t.Fatalf("state %q", st.State)
	}
}
