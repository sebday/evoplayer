package jsonlog

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

func MergeTrackCache(basePath, workDir string, updateCount int, outPath string) error {
	baseRaw, err := os.ReadFile(basePath)
	if err != nil {
		return err
	}
	var base []map[string]any
	if err := json.Unmarshal(baseRaw, &base); err != nil {
		return err
	}
	byPath := map[string]map[string]any{}
	for _, item := range base {
		path, _ := item["path"].(string)
		if path == "" {
			continue
		}
		if st, err := os.Stat(path); err != nil || st.IsDir() {
			continue
		}
		byPath[path] = item
	}
	for i := 0; i < updateCount; i++ {
		path := filepath.Join(workDir, strconv.Itoa(i)+".json")
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var item map[string]any
		if err := json.Unmarshal(raw, &item); err != nil {
			continue
		}
		if p, _ := item["path"].(string); p != "" {
			byPath[p] = item
		}
	}
	merged := make([]map[string]any, 0, len(byPath))
	for _, item := range byPath {
		merged = append(merged, item)
	}
	sortByPath(merged)
	out, err := json.Marshal(merged)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, out, 0o644)
}

func ScrobbleRecent(logPath string, limit int) ([]map[string]any, error) {
	if logPath == "" {
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
		line := stringsTrim(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	seen := map[any]struct{}{}
	out := make([]map[string]any, 0, limit)
	for i := len(lines) - 1; i >= 0; i-- {
		var row map[string]any
		if err := json.Unmarshal([]byte(lines[i]), &row); err != nil {
			continue
		}
		key := any(row["path"])
		if key == nil || key == "" {
			key = []any{row["artist"], row["title"]}
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

func sortByPath(items []map[string]any) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			a, _ := items[i]["path"].(string)
			b, _ := items[j]["path"].(string)
			if b < a {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func stringsTrim(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
