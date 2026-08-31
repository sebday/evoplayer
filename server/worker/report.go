package worker

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/sebday/evoplayer/server/jobs"
)

// Event is one NDJSON line emitted by a worker process.
type Event struct {
	Type    string `json:"type"`
	Line    string `json:"line,omitempty"`
	Phase   string `json:"phase,omitempty"`
	Done    int    `json:"done,omitempty"`
	Total   int    `json:"total,omitempty"`
	Folder  string `json:"folder,omitempty"`
	Message string `json:"message,omitempty"`
}

// NDJSONReporter streams job progress to w as NDJSON lines.
type NDJSONReporter struct {
	w io.Writer
}

func NewNDJSONReporter(w io.Writer) *NDJSONReporter {
	return &NDJSONReporter{w: w}
}

func (r *NDJSONReporter) Progress(p jobs.Progress) {
	_ = r.write(Event{
		Type:   "progress",
		Phase:  p.Phase,
		Done:   p.Done,
		Total:  p.Total,
		Folder: p.Folder,
	})
}

func (r *NDJSONReporter) Line(s string) {
	_ = r.write(Event{Type: "log", Line: s})
}

func (r *NDJSONReporter) Error(message string) error {
	return r.write(Event{Type: "error", Message: message})
}

func (r *NDJSONReporter) write(ev Event) error {
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(r.w, "%s\n", b)
	return err
}

// ParseEvent decodes one NDJSON worker event line.
func ParseEvent(line []byte) (Event, error) {
	var ev Event
	if err := json.Unmarshal(line, &ev); err != nil {
		return Event{}, err
	}
	return ev, nil
}

// ApplyEvent updates a jobs manager from a worker event.
func ApplyEvent(jm interface {
	AppendLog(string)
	SetProgress(jobs.Progress)
}, ev Event) {
	switch ev.Type {
	case "log":
		if ev.Line != "" {
			jm.AppendLog(ev.Line)
		}
	case "progress":
		jm.SetProgress(jobs.Progress{
			Phase:  ev.Phase,
			Done:   ev.Done,
			Total:  ev.Total,
			Folder: ev.Folder,
		})
	}
}
