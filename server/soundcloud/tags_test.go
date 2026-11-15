package soundcloud

import "testing"

func TestExtraSoundCloudTags(t *testing.T) {
	track := &Track{
		Genre:   "Drum & Bass",
		TagList: `"Space Jazz" house "Drum & Bass"`,
	}
	got := extraSoundCloudTags(track, "Drum & Bass")
	if got != "Space Jazz; house" {
		t.Fatalf("got %q", got)
	}
	if extraSoundCloudTags(track, "House") != "Drum & Bass; Space Jazz" {
		t.Fatalf("folder skip failed: %q", extraSoundCloudTags(track, "House"))
	}
}
