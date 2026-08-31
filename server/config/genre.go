package config

import (
	"fmt"
	"strings"
)

// Canonical genre keys (normalized master IDs).
const (
	GenreDrumAndBass = "drumandbass"
	GenreDubstep     = "dubstep"
	GenreGrime       = "grime"
	GenreHipHop      = "hiphop"
	GenreHouse       = "house"
	GenreElectronic  = "electronic"
)

// DefaultGenreAliases maps normalized tag aliases to canonical genre keys.
func DefaultGenreAliases() map[string]string {
	return map[string]string{
		"jungle":       GenreDrumAndBass,
		"dnb":          GenreDrumAndBass,
		"drumandbass":  GenreDrumAndBass,
		"drumbass":     GenreDrumAndBass,
		"jumpup":       GenreDrumAndBass,
		"neurofunk":    GenreDrumAndBass,
		"liquid":       GenreDrumAndBass,
		"halftime":     GenreDrumAndBass,
		"dubstep":      GenreDubstep,
		"140":          GenreDubstep,
		"bassmusic":    GenreDubstep,
		"riddim":       GenreDubstep,
		"brostep":      GenreDubstep,
		"hiphop":       GenreHipHop,
		"hiphopandrap": GenreHipHop,
		"hiphoprap":    GenreHipHop,
		"raphiphop":    GenreHipHop,
		"rap":          GenreHipHop,
		"trap":         GenreHipHop,
		"phonk":        GenreHipHop,
		"drill":        GenreHipHop,
		"grime":        GenreGrime,
		"ukgrime":      GenreGrime,
		"house":        GenreHouse,
		"deephouse":    GenreHouse,
		"techhouse":    GenreHouse,
		"garage":       GenreHouse,
		"electronic":   GenreElectronic,
		"danceandedm":  GenreElectronic,
		"dance":        GenreElectronic,
		"edm":          GenreElectronic,
		"techno":       GenreElectronic,
		"trance":       GenreElectronic,
		"ambient":      GenreElectronic,
		"breakbeat":    GenreElectronic,
	}
}

// DefaultGenreFolders maps canonical keys to preferred on-disk folder names.
func DefaultGenreFolders() map[string]string {
	return map[string]string{
		GenreDrumAndBass: "drum&bass",
		GenreDubstep:     "dubstep",
		GenreGrime:       "grime",
		GenreHipHop:      "hiphop",
		GenreHouse:       "house",
		GenreElectronic:  "electronic",
	}
}

// GenreAliases loads [genre_aliases] from music.toml merged over defaults.
func GenreAliases(path string) (map[string]string, error) {
	out := DefaultGenreAliases()
	if path == "" {
		return out, nil
	}
	data, err := Load(path)
	if err != nil {
		return nil, err
	}
	sec := data["genre_aliases"]
	if sec == nil {
		return out, nil
	}
	for alias, val := range sec {
		canon := strings.TrimSpace(fmt.Sprint(val))
		if canon == "" {
			continue
		}
		key := normalizeGenreConfigKey(alias)
		if key == "" {
			continue
		}
		out[key] = normalizeGenreConfigKey(canon)
	}
	return out, nil
}

// GenreFolders loads [genres] from music.toml merged over defaults.
func GenreFolders(path string) (map[string]string, error) {
	out := DefaultGenreFolders()
	if path == "" {
		return out, nil
	}
	data, err := Load(path)
	if err != nil {
		return nil, err
	}
	sec := data["genres"]
	if sec == nil {
		return out, nil
	}
	for canon, val := range sec {
		folder := strings.TrimSpace(fmt.Sprint(val))
		if folder == "" {
			continue
		}
		key := normalizeGenreConfigKey(canon)
		if key == "" {
			continue
		}
		out[key] = folder
	}
	return out, nil
}

// PlaylistFolders loads [playlist_folders] from music.toml (soundcloud playlist title -> disk folder).
func PlaylistFolders(path string) (map[string]string, error) {
	out := map[string]string{}
	if path == "" {
		return out, nil
	}
	data, err := Load(path)
	if err != nil {
		return nil, err
	}
	sec := data["playlist_folders"]
	if sec == nil {
		return out, nil
	}
	for name, val := range sec {
		folder := strings.TrimSpace(fmt.Sprint(val))
		if folder == "" {
			continue
		}
		title := strings.TrimSpace(name)
		if title == "" {
			continue
		}
		out[strings.ToLower(title)] = folder
		out[normalizeGenreConfigKey(title)] = folder
	}
	return out, nil
}

// PlaylistFolder returns the on-disk folder for a soundcloud playlist title.
func PlaylistFolder(path, playlistTitle string) string {
	folders, err := PlaylistFolders(path)
	if err != nil || len(folders) == 0 {
		return ""
	}
	title := strings.TrimSpace(playlistTitle)
	if title == "" {
		return ""
	}
	if folder, ok := folders[strings.ToLower(title)]; ok {
		return folder
	}
	return folders[normalizeGenreConfigKey(title)]
}

// SeedGenreConfig writes default [genres] and [genre_aliases] when missing.
func SeedGenreConfig(path string) error {
	data, err := Load(path)
	if err != nil {
		return err
	}
	changed := false
	if data["genres"] == nil {
		data["genres"] = map[string]any{}
		for canon, folder := range DefaultGenreFolders() {
			data["genres"][canon] = folder
		}
		changed = true
	}
	if data["genre_aliases"] == nil {
		data["genre_aliases"] = map[string]any{}
		for alias, canon := range DefaultGenreAliases() {
			data["genre_aliases"][alias] = canon
		}
		changed = true
	}
	if !changed {
		return nil
	}
	return write(path, data)
}

func normalizeGenreConfigKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "&", " and ")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}
