package history

type WeekInfo struct {
	From  int    `json:"from"`
	To    int    `json:"to"`
	Label string `json:"label"`
}

type Totals struct {
	Scrobbles int     `json:"scrobbles"`
	Artists   int     `json:"artists"`
	Albums    int     `json:"albums"`
	Tracks    int     `json:"tracks"`
	Hours     float64 `json:"hours"`
}

type DailyCount struct {
	Day   string `json:"day"`
	Count int    `json:"count"`
}

type NamedCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type TrackCount struct {
	Artist string `json:"artist"`
	Title  string `json:"title"`
	Count  int    `json:"count"`
}

type RecapTop struct {
	Name   string `json:"name,omitempty"`
	Artist string `json:"artist,omitempty"`
	Title  string `json:"title,omitempty"`
	Count  int    `json:"count"`
}

type RecapItem struct {
	Count    int      `json:"count"`
	Prev     int      `json:"prev"`
	DeltaPct int      `json:"delta_pct"`
	Up       bool     `json:"up"`
	Peak     bool     `json:"peak"`
	AllTime  bool     `json:"all_time,omitempty"`
	Top      RecapTop `json:"top"`
}

type Report struct {
	Source         string               `json:"source"`
	Period         string               `json:"period,omitempty"`
	User           string               `json:"user,omitempty"`
	Week           WeekInfo             `json:"week"`
	Totals         Totals               `json:"totals"`
	Daily          []DailyCount         `json:"daily"`
	TopArtists     []NamedCount         `json:"top_artists"`
	TopAlbums      []NamedCount         `json:"top_albums"`
	TopTracks      []TrackCount         `json:"top_tracks"`
	Recent         []map[string]any     `json:"recent"`
	AvailableWeeks []WeekInfo           `json:"available_weeks,omitempty"`
	Recap          map[string]RecapItem `json:"recap,omitempty"`
}

type Params struct {
	CacheDir    string
	User        string
	APIKey      string
	WeekFrom    string
	ScrobbleLog string
	Limit       int
}
