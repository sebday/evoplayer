package playback_test

import (
	"testing"

	"github.com/sebday/evoplayer/internal/playback"
)

func TestPresentationDelaySamplesDefault(t *testing.T) {
	var out playback.PlayerOutput
	got := out.PresentationDelaySamples()
	if got < 960 {
		t.Fatalf("delay = %d, want at least 20ms floor", got)
	}
	if got > 2400 {
		t.Fatalf("delay = %d, want ~20ms floor without oto player, not 500ms", got)
	}
}

func TestReanchorPresentation(t *testing.T) {
	var out playback.PlayerOutput
	out.ReanchorPresentation(10, 48000)
	want := int64(10*48000) + int64(out.PresentationDelaySamples())
	if got := out.SubmittedSamples(); got != want {
		t.Fatalf("submitted = %d, want %d", got, want)
	}
}
