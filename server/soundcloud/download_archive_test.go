package soundcloud

import (
	"os"
	"testing"

	"github.com/sebday/evoplayer/server/syncarchive"
)

func TestArchiveSCPersists(t *testing.T) {
	dir := t.TempDir()
	path := syncarchive.Path(dir)
	archive, err := syncarchive.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	archiveSC(archive, 424242, nil)
	reloaded, err := syncarchive.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.HasSC(424242) {
		t.Fatal("expected archived track id to persist")
	}
}

func TestArchiveSCSkipsZeroID(t *testing.T) {
	dir := t.TempDir()
	path := syncarchive.Path(dir)
	archive, err := syncarchive.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	archiveSC(archive, 0, nil)
	if _, err := os.Stat(path); err == nil {
		t.Fatal("expected no archive file for zero id")
	}
}

func TestDRMTrackSkippedOnNextSync(t *testing.T) {
	dir := t.TempDir()
	path := syncarchive.Path(dir)
	archive, err := syncarchive.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	const drmID int64 = 99887766
	if !isDRMError(errDRMProtected()) {
		t.Fatal("fixture should be a drm error")
	}
	archiveSC(archive, drmID, nil)

	reloaded, err := syncarchive.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	tracks := []Track{{ID: drmID}, {ID: 1}}
	pending := 0
	for _, track := range tracks {
		if !reloaded.HasSC(track.ID) {
			pending++
		}
	}
	if pending != 1 {
		t.Fatalf("pending = %d, want 1 (drm track should be archived)", pending)
	}
}

func errDRMProtected() error {
	return errString("drm protected")
}

type errString string

func (e errString) Error() string { return string(e) }
