package tags

import "testing"

func TestTrimFrameDropsPadding(t *testing.T) {
	if got := TrimFrame("Liquid\x00\x00\x00\x00\x00"); got != "Liquid" {
		t.Fatalf("got %q", got)
	}
	if got := TrimFrame("Liquid"); got != "Liquid" {
		t.Fatalf("got %q", got)
	}
}

func TestParseTagList(t *testing.T) {
	got := ParseTagList(" Space Jazz, space jazz; Jazzstep ")
	if FormatTagList(got) != "Space Jazz; Jazzstep" {
		t.Fatalf("got %q", FormatTagList(got))
	}
	if TagKey("Space Jazz") != TagKey("space  jazz") {
		t.Fatal("space jazz keys differ")
	}
	if TagKey("spacejazz") == TagKey("space jazz") {
		t.Fatal("spacejazz collapsed into space jazz")
	}
}
