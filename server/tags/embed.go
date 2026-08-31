package tags

import (
	"fmt"
	"strings"

	"github.com/bogem/id3v2"
)

// EmbedMP3 writes text tags and optional front-cover APIC into an mp3 file.
func EmbedMP3(path string, targets map[string]string, picture []byte, pictureMIME string) error {
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("tags: open mp3: %w", err)
	}
	defer tag.Close()
	if v, ok := targets["title"]; ok {
		tag.SetTitle(v)
	}
	if v, ok := targets["artist"]; ok {
		tag.SetArtist(v)
	}
	if v, ok := targets["genre"]; ok {
		tag.SetGenre(v)
	}
	if v, ok := targets["year"]; ok {
		tag.AddTextFrame("TDRC", tag.DefaultEncoding(), v)
	}
	if v, ok := targets["album"]; ok {
		tag.SetAlbum(v)
	}
	if v, ok := targets["comment"]; ok && v != "" {
		tag.AddCommentFrame(id3v2.CommentFrame{
			Encoding: tag.DefaultEncoding(),
			Language: "eng",
			Text:     v,
		})
	}
	if v, ok := targets["duration_ms"]; ok && v != "" {
		tag.DeleteFrames("TLEN")
		tag.AddTextFrame("TLEN", tag.DefaultEncoding(), strings.TrimSpace(v))
	}
	if len(picture) > 0 {
		mime := pictureMIME
		if mime == "" {
			mime = "image/jpeg"
		}
		tag.DeleteFrames("APIC")
		tag.AddAttachedPicture(id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    mime,
			PictureType: id3v2.PTFrontCover,
			Description: "cover",
			Picture:     picture,
		})
	}
	return tag.Save()
}

// PictureMIME guesses image MIME from magic bytes.
func PictureMIME(data []byte) string {
	if len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 {
		return "image/jpeg"
	}
	if len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n" {
		return "image/png"
	}
	if len(data) >= 6 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a") {
		return "image/gif"
	}
	if len(data) >= 12 && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}
	return "image/jpeg"
}

// ArtworkURLLarge returns a higher-resolution SoundCloud artwork URL when possible.
func ArtworkURLLarge(url string) string {
	url = strings.TrimSpace(url)
	if url == "" {
		return ""
	}
	if strings.Contains(url, "-large") {
		return strings.Replace(url, "-large", "-original", 1)
	}
	if strings.Contains(url, "-t500x500") {
		return strings.Replace(url, "-t500x500", "-original", 1)
	}
	return url
}
