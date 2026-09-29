package jsonlog

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
)

func ScrobbleRecent(logPath string, limit int) ([]map[string]any, error) {
	if logPath == "" || limit <= 0 {
		return []map[string]any{}, nil
	}
	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]any{}, nil
		}
		return nil, err
	}
	defer f.Close()
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	seen := map[string]struct{}{}
	out := make([]map[string]any, 0, limit)
	for i := len(lines) - 1; i >= 0; i-- {
		var row map[string]any
		if err := json.Unmarshal([]byte(lines[i]), &row); err != nil {
			continue
		}
		key, _ := row["path"].(string)
		if key == "" {
			artist, _ := row["artist"].(string)
			title, _ := row["title"].(string)
			key = artist + "\x00" + title
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, row)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func QueueUpNext(tracksPath, currentPath string, limit int) ([]map[string]any, error) {
	raw, err := os.ReadFile(tracksPath)
	if err != nil {
		return []map[string]any{}, err
	}
	var tracks []map[string]any
	if err := json.Unmarshal(raw, &tracks); err != nil {
		return []map[string]any{}, err
	}
	idx := -1
	for i, row := range tracks {
		path, _ := row["path"].(string)
		if path == currentPath {
			idx = i
			break
		}
	}
	if idx < 0 {
		return []map[string]any{}, nil
	}
	end := idx + 1 + limit
	if end > len(tracks) {
		end = len(tracks)
	}
	if idx+1 >= end {
		return []map[string]any{}, nil
	}
	return tracks[idx+1 : end], nil
}
