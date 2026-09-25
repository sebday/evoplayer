package library

import "strings"

// GuessFolderFromCandidates maps free-text genre or tag strings to a library folder.
func GuessFolderFromCandidates(env Env, names ...string) string {
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if folder := MatchLibraryGenre(env, name); folder != "" {
			return folder
		}
		if folder := guessFolderFromText(env, name); folder != "" {
			return folder
		}
	}
	return ""
}

func guessFolderFromText(env Env, raw string) string {
	n := NormalizeGenreKey(raw)
	if n == "" {
		return ""
	}
	folder := guessFolderNormalized(n)
	if folder == "" {
		return ""
	}
	return matchLibraryFolderExact(env, folder)
}

// GuessFolderName maps free text to a canonical library folder name without checking disk.
func GuessFolderName(raw string) string {
	return guessFolderNormalized(NormalizeGenreKey(raw))
}

func guessFolderNormalized(n string) string {
	switch {
	case strings.Contains(n, "drumandbass"), strings.Contains(n, "drumbass"), strings.Contains(n, "dnb"),
		strings.Contains(n, "jungle"), strings.Contains(n, "neurofunk"), strings.Contains(n, "jumpup"),
		n == "liquid", strings.Contains(n, "halftime"), strings.Contains(n, "autonomic"):
		return "drum&bass"
	case strings.Contains(n, "dubstep"), strings.Contains(n, "deepdubstep"), n == "140", strings.Contains(n, "riddim"),
		strings.Contains(n, "brostep"), strings.Contains(n, "bassmusic"), n == "bass", strings.Contains(n, "wardub"),
		strings.Contains(n, "innamind"), strings.Contains(n, "deepheads"), n == "dub",
		strings.Contains(n, "flumestep"), strings.Contains(n, "gantz"):
		return "dubstep"
	case strings.Contains(n, "garage"), n == "ukg", strings.Contains(n, "ukgarage"), strings.Contains(n, "2step"),
		strings.Contains(n, "funky"), strings.Contains(n, "baile"):
		return "garage"
	case strings.Contains(n, "grime"), strings.Contains(n, "ukdrill"):
		return "grime"
	case strings.Contains(n, "hiphop"), strings.Contains(n, "hiphoprap"), strings.Contains(n, "rap"),
		strings.Contains(n, "phonk"), strings.Contains(n, "memphis"), strings.Contains(n, "shadowrap"),
		strings.Contains(n, "trillwave"), strings.Contains(n, "jerseyclub"), strings.Contains(n, "lofihiphop"):
		return "hiphop"
	case strings.Contains(n, "house"), strings.Contains(n, "techhouse"), strings.Contains(n, "tribalhouse"):
		return "house"
	case strings.Contains(n, "electronic"), strings.Contains(n, "edm"), strings.Contains(n, "techno"),
		strings.Contains(n, "trance"), strings.Contains(n, "ambient"), strings.Contains(n, "breakbeat"),
		strings.Contains(n, "triphop"), strings.Contains(n, "indie"), strings.Contains(n, "rock"),
		strings.Contains(n, "piano"):
		return "electronic"
	case strings.Contains(n, "calibre"), strings.Contains(n, "metalheadz"), strings.Contains(n, "skeptical"),
		strings.Contains(n, "marcus"), strings.Contains(n, "oldskool"), strings.Contains(n, "spacejazz"),
		strings.Contains(n, "shogun"), strings.Contains(n, "enei"), strings.Contains(n, "ivylab"),
		strings.Contains(n, "hospital"), strings.Contains(n, "dispatch"), strings.Contains(n, "mako"):
		return "drum&bass"
	case n == "pop", strings.Contains(n, "randb"), strings.Contains(n, "randbsoul"):
		// SoundCloud often mis-tags bass and hip-hop tracks as Pop/R&B.
		return "hiphop"
	}
	return ""
}
