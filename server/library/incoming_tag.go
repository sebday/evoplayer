package library

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sebday/evoplayer/server/tags"
)

const incomingTagsFile = "tags.json"

func GenreChoices(env Env) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	if names, err := listGenreNames(env); err == nil {
		for _, name := range names {
			add(name)
		}
	}
	sort.Strings(out)
	return out
}

func SetIncomingGenre(env Env, path, genre string) (map[string]any, error) {
	src, err := incomingAudioPath(env, path)
	if err != nil {
		return nil, err
	}
	folder := matchLibraryFolder(env, genre)
	if folder == "" {
		return nil, fmt.Errorf("unknown library folder: %s", genre)
	}
	if err := writeIncomingOverlayGenre(env, src, folder); err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(src))
	if ext == ".mp3" || ext == ".mp2" {
		if err := tags.EmbedMP3(src, map[string]string{"genre": folder}, nil, ""); err != nil {
			fmt.Fprintf(os.Stderr, "evoplayer: warn: incoming genre embed: %v\n", err)
		}
	}
	return PreviewIncoming(env, src), nil
}

func incomingAudioPath(env Env, path string) (string, error) {
	incoming := filepath.Join(env.MusicRoot, ".incoming")
	clean := filepath.Clean(path)
	rel, err := filepath.Rel(incoming, clean)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("not an incoming file")
	}
	st, err := os.Stat(clean)
	if err != nil || st.IsDir() {
		return "", fmt.Errorf("not an incoming file")
	}
	return clean, nil
}

func overlayGenre(env Env, path string) string {
	overlay := readIncomingOverlay(env)
	return overlay[filepath.Base(path)]
}

func readIncomingOverlay(env Env) map[string]string {
	raw, err := os.ReadFile(incomingOverlayPath(env))
	if err != nil || len(raw) == 0 {
		return map[string]string{}
	}
	out := map[string]string{}
	if json.Unmarshal(raw, &out) != nil {
		return map[string]string{}
	}
	return out
}

func writeIncomingOverlayGenre(env Env, path, genre string) error {
	incoming := filepath.Join(env.MusicRoot, ".incoming")
	if err := os.MkdirAll(incoming, 0o755); err != nil {
		return err
	}
	overlay := readIncomingOverlay(env)
	overlay[filepath.Base(path)] = genre
	raw, err := json.Marshal(overlay)
	if err != nil {
		return err
	}
	tmp := incomingOverlayPath(env) + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, incomingOverlayPath(env))
}

func incomingOverlayPath(env Env) string {
	return filepath.Join(env.MusicRoot, ".incoming", incomingTagsFile)
}
