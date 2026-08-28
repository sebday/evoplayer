package vinyl

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const defaultRoot = "/mnt/external/music/dubstep/vinyl"

type Move struct {
	Src   string
	Dest  string
	Label string
}

type Result struct {
	Root    string
	Execute bool
	Moves   []Move
	Skipped []string
}

func ByLabel(root string, execute bool) (Result, int) {
	if root == "" {
		root = defaultRoot
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return Result{}, 1
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return Result{}, 1
	}
	result := Result{Root: root, Execute: execute}
	unmapped := []string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !isReleaseFolder(name) {
			continue
		}
		label := getLabel(name)
		if label == "" {
			unmapped = append(unmapped, name)
			continue
		}
		src := filepath.Join(root, name)
		dest := filepath.Join(root, label, name)
		if pathExists(dest) {
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s -> %s/ (destination exists)", name, label))
			continue
		}
		srcClean, _ := filepath.EvalSymlinks(src)
		destClean, _ := filepath.EvalSymlinks(dest)
		if srcClean != "" && srcClean == destClean {
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s -> %s/ (already in place)", name, label))
			continue
		}
		result.Moves = append(result.Moves, Move{Src: src, Dest: dest, Label: label})
	}
	if len(unmapped) > 0 {
		fmt.Fprintln(os.Stderr, "error: could not determine label for:")
		for _, name := range unmapped {
			fmt.Fprintf(os.Stderr, "  %s\n", name)
		}
		return result, 1
	}
	sort.Slice(result.Moves, func(i, j int) bool {
		return filepath.Base(result.Moves[i].Src) < filepath.Base(result.Moves[j].Src)
	})
	if execute {
		labels := map[string]struct{}{}
		for _, move := range result.Moves {
			labels[move.Label] = struct{}{}
		}
		labelNames := make([]string, 0, len(labels))
		for label := range labels {
			labelNames = append(labelNames, label)
		}
		sort.Strings(labelNames)
		for _, label := range labelNames {
			_ = os.MkdirAll(filepath.Join(root, label), 0o755)
		}
		for _, move := range result.Moves {
			if err := os.Rename(move.Src, move.Dest); err != nil {
				fmt.Fprintf(os.Stderr, "error: move failed %s: %v\n", move.Src, err)
				return result, 1
			}
		}
	}
	return result, 0
}

func PrintResult(res Result) {
	labelCounts := map[string]int{}
	labels := map[string]struct{}{}
	for _, move := range res.Moves {
		labelCounts[move.Label]++
		labels[move.Label] = struct{}{}
	}
	labelNames := make([]string, 0, len(labels))
	for label := range labels {
		labelNames = append(labelNames, label)
	}
	sort.Strings(labelNames)

	mode := "dry-run"
	if res.Execute {
		mode = "execute"
	}
	fmt.Printf("root: %s\n", res.Root)
	fmt.Printf("mode: %s\n", mode)
	fmt.Printf("label folders to use/create: %d\n", len(labelNames))
	fmt.Printf("releases to move: %d\n", len(res.Moves))
	fmt.Printf("skipped: %d\n", len(res.Skipped))
	fmt.Println()
	for _, move := range res.Moves {
		fmt.Printf("  %s -> %s/\n", filepath.Base(move.Src), move.Label)
	}
	if len(res.Skipped) > 0 {
		fmt.Println()
		fmt.Println("skipped:")
		for _, line := range res.Skipped {
			fmt.Printf("  %s\n", line)
		}
	}
	fmt.Println()
	fmt.Println("label summary:")
	for _, label := range labelNames {
		fmt.Printf("  %s/: %d\n", label, labelCounts[label])
	}
	if !res.Execute {
		fmt.Println()
		fmt.Println("dry-run complete; re-run with --execute to apply")
		return
	}
	fmt.Println()
	fmt.Printf("moved %d release folders into %d label folders\n", len(res.Moves), len(labelNames))
}

func isReleaseFolder(name string) bool {
	if strings.HasPrefix(name, "medi-") {
		return true
	}
	if strings.HasPrefix(name, "whitelabel-") {
		return true
	}
	if name == "dubstep_collection_9" {
		return true
	}
	matched, _ := regexp.MatchString(`^[a-z]+\d`, name)
	return matched
}

func getLabel(name string) string {
	if strings.HasPrefix(name, "medi-") {
		return "medi"
	}
	if strings.HasPrefix(name, "whitelabel-") {
		return "whitelabel"
	}
	if name == "dubstep_collection_9" {
		return "_misc"
	}
	re := regexp.MustCompile(`^([a-z]+)\d`)
	if m := re.FindStringSubmatch(name); len(m) > 1 {
		return m[1]
	}
	return ""
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
