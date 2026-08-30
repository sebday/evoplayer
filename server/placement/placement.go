package placement

import (
	"bufio"
	"encoding/json"
	"os"
)

type Entry map[string]any

func readRows(logPath string) ([]Entry, error) {
	f, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rows := make([]Entry, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := trim(scanner.Text())
		if line == "" {
			continue
		}
		var row Entry
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}
		rows = append(rows, row)
	}
	return rows, scanner.Err()
}

func undoneRefs(rows []Entry) map[any]struct{} {
	set := map[any]struct{}{}
	for _, row := range rows {
		if str(row["event"]) == "undo" && row["ref"] != nil {
			set[row["ref"]] = struct{}{}
		}
	}
	return set
}

func placementsFrom(rows []Entry, undoable bool) []Entry {
	undone := undoneRefs(rows)
	out := make([]Entry, 0)
	for _, row := range rows {
		if str(row["event"]) == "undo" {
			continue
		}
		if str(row["from"]) == "" || str(row["to"]) == "" {
			continue
		}
		if undoable {
			if _, skip := undone[row["id"]]; skip {
				continue
			}
		}
		out = append(out, row)
	}
	return out
}

func Log(logPath string, limit int, undoable bool) ([]Entry, error) {
	rows, err := readRows(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, err
	}
	placements := placementsFrom(rows, undoable)
	if limit > 0 && len(placements) > limit {
		placements = placements[len(placements)-limit:]
	}
	for i, j := 0, len(placements)-1; i < j; i, j = i+1, j-1 {
		placements[i], placements[j] = placements[j], placements[i]
	}
	return placements, nil
}

func UndoPlan(logPath string, last int) ([]Entry, error) {
	rows, err := readRows(logPath)
	if err != nil {
		return nil, err
	}
	placements := placementsFrom(rows, true)
	if last > 0 && len(placements) > last {
		placements = placements[len(placements)-last:]
	}
	selected := make([]Entry, len(placements))
	for i := range placements {
		selected[i] = placements[len(placements)-1-i]
	}
	return selected, nil
}

func str(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func trim(s string) string {
	return stringsTrimSpace(s)
}

func stringsTrimSpace(s string) string {
	start, end := 0, len(s)
	for start < end {
		c := s[start]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			break
		}
		start++
	}
	for end > start {
		c := s[end-1]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			break
		}
		end--
	}
	return s[start:end]
}
