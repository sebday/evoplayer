package library

import (
	"database/sql"
)

type Stats struct {
	Root    string `json:"root"`
	Genres  int    `json:"genres"`
	Tracks  int    `json:"tracks"`
}

func LibraryStats(env Env) (Stats, error) {
	out := Stats{Root: env.MusicRoot}
	genres, err := listGenreNames(env)
	if err != nil {
		return out, err
	}
	out.Genres = len(genres)
	db, err := OpenDB(env.LibraryDB)
	if err == nil {
		defer db.Close()
		if n, err := countTracks(db); err == nil && n > 0 {
			out.Tracks = n
			return out, nil
		}
	}
	for _, g := range genres {
		items, err := TracksForGenre(env, g)
		if err == nil {
			out.Tracks += len(items)
		}
	}
	return out, nil
}

func countTracks(db *sql.DB) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM tracks`).Scan(&n)
	return n, err
}
