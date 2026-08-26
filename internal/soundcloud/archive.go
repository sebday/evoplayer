package soundcloud

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Archive struct {
	path string
	seen map[string]bool
}

func LoadArchive(path string) (*Archive, error) {
	a := &Archive{path: path, seen: map[string]bool{}}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return a, nil
		}
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if strings.Contains(line, " ") {
			for _, part := range strings.Fields(line) {
				if isTrackID(part) {
					a.seen[part] = true
				}
			}
			continue
		}
		if isTrackID(line) {
			a.seen[line] = true
		}
	}
	return a, sc.Err()
}

func isTrackID(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func (a *Archive) Has(id int64) bool {
	if id == 0 {
		return false
	}
	return a.seen[formatID(id)]
}

func (a *Archive) Add(id int64) error {
	key := formatID(id)
	if a.seen[key] {
		return nil
	}
	a.seen[key] = true
	f, err := os.OpenFile(a.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(key + "\n")
	return err
}

func formatID(id int64) string {
	return fmt.Sprintf("%d", id)
}
