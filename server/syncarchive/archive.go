package syncarchive

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const archiveFile = "sync-archive.txt"

// Path returns the persistent download log under state dir.
func Path(stateDir string) string {
	return strings.TrimRight(stateDir, "/") + "/" + archiveFile
}

type Archive struct {
	path string
	seen map[string]bool
}

func Load(path string) (*Archive, error) {
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
		a.ingestLine(strings.TrimSpace(sc.Text()))
	}
	return a, sc.Err()
}

func (a *Archive) ingestLine(line string) {
	if line == "" {
		return
	}
	if strings.HasPrefix(line, "sc:") || strings.HasPrefix(line, "yt:") {
		a.seen[line] = true
		return
	}
	if strings.Contains(line, " ") {
		for _, part := range strings.Fields(line) {
			if isNumericID(part) {
				a.seen[SCKeyString(part)] = true
				a.seen[part] = true
			}
		}
		return
	}
	if isNumericID(line) {
		a.seen[SCKeyString(line)] = true
		a.seen[line] = true
	}
}

func isNumericID(s string) bool {
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

func SCKey(id int64) string {
	return fmt.Sprintf("sc:%d", id)
}

func SCKeyString(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	if strings.HasPrefix(id, "sc:") {
		return id
	}
	return "sc:" + id
}

func YTKey(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	if strings.HasPrefix(id, "yt:") {
		return id
	}
	return "yt:" + id
}

func (a *Archive) HasSC(id int64) bool {
	if id == 0 {
		return false
	}
	key := SCKey(id)
	if a.seen[key] {
		return true
	}
	legacy := fmt.Sprintf("%d", id)
	return a.seen[legacy]
}

func (a *Archive) HasYT(id string) bool {
	key := YTKey(id)
	return key != "" && a.seen[key]
}

func (a *Archive) AddSC(id int64) error {
	if id == 0 {
		return nil
	}
	return a.addKey(SCKey(id))
}

func (a *Archive) AddYT(id string) error {
	key := YTKey(id)
	if key == "" {
		return nil
	}
	return a.addKey(key)
}

func (a *Archive) addKey(key string) error {
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
