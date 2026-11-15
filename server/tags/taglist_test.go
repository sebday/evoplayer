package tags

import (
	"os"
	"path/filepath"
	"testing"

	id3v2 "github.com/bogem/id3v2"
)

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

func TestWriteClearsTagsAndLabel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.mp3")
	if err := os.WriteFile(path, make([]byte, 32), 0o644); err != nil {
		t.Fatal(err)
	}
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		t.Fatal(err)
	}
	tag.SetGenre("Garage")
	tag.AddTextFrame("TPUB", tag.DefaultEncoding(), "Some Label")
	tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
		Encoding: tag.DefaultEncoding(), Description: "TAGS", Value: "bass; edm; remix",
	})
	tag.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
		Encoding: tag.DefaultEncoding(), Description: "soundcloud_id", Value: "99",
	})
	if err := tag.Save(); err != nil {
		t.Fatal(err)
	}
	tag.Close()
	if err := Write(path, map[string]string{"tags": "", "publisher": ""}); err != nil {
		t.Fatal(err)
	}
	got, err := Probe(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Tag.Genre != "Garage" {
		t.Fatalf("genre = %q", got.Tag.Genre)
	}
	if got.Tag.Label != "" || len(got.Tag.Tags) != 0 {
		t.Fatalf("label %q tags %#v", got.Tag.Label, got.Tag.Tags)
	}
	if got.Tag.SoundcloudID != "99" {
		t.Fatalf("soundcloud_id = %q", got.Tag.SoundcloudID)
	}
}
