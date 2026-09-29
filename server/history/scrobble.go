package history

import (
	"bufio"
	"encoding/json"
	"os"
	"sort"
	"strings"
	"time"
)

var weekdayOrder = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

func parseTS(value string) (int64, bool) {
	if value == "" {
		return 0, false
	}
	value = strings.ReplaceAll(value, "Z", "+00:00")
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05-07:00", value)
	}
	if err != nil {
		return 0, false
	}
	return t.Unix(), true
}

// scrobbleFallback builds a report from the local scrobble log; endTS 0 means all time.
func scrobbleFallback(logPath string, startTS, endTS, limit int) *Report {
	overall := endTS == 0
	daily := map[string]int{}
	artists := map[string]int{}
	albums := map[string]int{}
	tracks := map[[2]string]int{}
	recent := make([]map[string]any, 0)
	hours := 0.0
	if logPath == "" {
		return emptyReportBody(startTS, endTS, "local", recent, daily, artists, albums, tracks, hours, overall)
	}
	f, err := os.Open(logPath)
	if err != nil {
		return emptyReportBody(startTS, endTS, "local", recent, daily, artists, albums, tracks, hours, overall)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}
		event, _ := row["event"].(string)
		if event != "submit" && event != "scrobble" {
			continue
		}
		ts, ok := parseTS(fmtString(row["at"]))
		if !overall && (!ok || ts < int64(startTS) || ts >= int64(endTS)) {
			continue
		}
		artist := strings.TrimSpace(fmtString(row["artist"]))
		title := strings.TrimSpace(fmtString(row["title"]))
		album := strings.TrimSpace(fmtString(row["album"]))
		if artist == "" || title == "" {
			continue
		}
		if ok {
			daily[time.Unix(ts, 0).Format("Mon")]++
		}
		artists[artist]++
		if album != "" {
			albums[artist+" — "+album]++
		}
		tracks[[2]string{artist, title}]++
		hours += 3.5 / 60.0
		recent = append(recent, row)
	}
	sort.Slice(recent, func(i, j int) bool {
		return fmtString(recent[i]["at"]) > fmtString(recent[j]["at"])
	})
	if len(recent) > limit {
		recent = recent[:limit]
	}
	return emptyReportBody(startTS, endTS, "local", recent, daily, artists, albums, tracks, hours, overall)
}

func fmtString(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	default:
		return ""
	}
}
