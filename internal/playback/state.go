package playback

import (
	"fmt"
	"math"
)

type Status struct {
	State          string  `json:"state"`
	Path           string  `json:"path"`
	Title          string  `json:"title"`
	Artist         string  `json:"artist"`
	Genre          string  `json:"genre"`
	Album          string  `json:"album"`
	Year           string  `json:"year"`
	Label          string  `json:"label"`
	Position       float64 `json:"position"`
	Duration       float64 `json:"duration"`
	Volume         int     `json:"volume"`
	Art            string  `json:"art"`
	Waveform       string  `json:"waveform"`
	Playlist       string  `json:"playlist"`
	Shuffle        bool    `json:"shuffle"`
	PlaylistPos    int     `json:"playlist_pos"`
	PlaylistCount  int     `json:"playlist_count"`
	QueueRevision  uint64  `json:"queue_revision,omitempty"`
	Liked          bool    `json:"liked"`
	PositionLabel  string  `json:"position_label"`
	DurationLabel  string  `json:"duration_label"`
}

func FormatTime(sec float64) string {
	total := int(math.Max(0, math.Floor(sec)))
	min := total / 60
	s := total % 60
	return fmt.Sprintf("%d:%02d", min, s)
}

func (s Status) WithLabels() Status {
	out := s
	out.PositionLabel = FormatTime(s.Position)
	out.DurationLabel = FormatTime(s.Duration)
	return out
}

func StoppedStatus() Status {
	return Status{State: "stopped"}.WithLabels()
}
