package worker

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sebday/evoplayer/server/jobs"
)

func TestNDJSONReporterRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	rep := NewNDJSONReporter(&buf)
	rep.Line("hello")
	rep.Progress(jobs.Progress{Phase: "fetching", Done: 1, Total: 3})

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(lines), buf.String())
	}
	var logEv, progEv Event
	if err := json.Unmarshal([]byte(lines[0]), &logEv); err != nil {
		t.Fatal(err)
	}
	if logEv.Type != "log" || logEv.Line != "hello" {
		t.Fatalf("log event = %+v", logEv)
	}
	if err := json.Unmarshal([]byte(lines[1]), &progEv); err != nil {
		t.Fatal(err)
	}
	if progEv.Type != "progress" || progEv.Done != 1 || progEv.Total != 3 || progEv.Phase != "fetching" {
		t.Fatalf("progress event = %+v", progEv)
	}
}
