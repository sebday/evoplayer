package library

// GuessFolderFromCandidates returns a library folder when a candidate is that folder's name.
func GuessFolderFromCandidates(env Env, names ...string) string {
	return MatchLibraryGenre(env, names...)
}

// GuessFolderName does not invent a folder from keywords. Filing uses the genre on the file.
func GuessFolderName(raw string) string {
	return ""
}
