package gitfiles

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// StagedDirs runs `git diff --cached --name-only -z` in root and returns a
// sorted, deduplicated list of directories that have staged changes and contain
// at least one .tf file. The -z flag gives NUL-delimited raw UTF-8 paths,
// bypassing core.quotePath encoding entirely.
func StagedDirs(root string) ([]string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("git", "diff", "--cached", "--name-only", "-z")
	cmd.Dir = abs
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff --cached: %w", err)
	}

	seen := map[string]struct{}{}
	for _, entry := range bytes.Split(out, []byte{0}) {
		filePath := strings.TrimSpace(string(entry))
		if filePath == "" {
			continue
		}

		dir := filepath.Join(abs, filepath.Dir(filePath))

		if containsTerraDotDir(dir) {
			continue
		}

		if hasTFFiles(dir) {
			rel, err := filepath.Rel(abs, dir)
			if err != nil {
				rel = dir
			}
			seen[rel] = struct{}{}
		}
	}

	dirs := make([]string, 0, len(seen))
	for d := range seen {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	return dirs, nil
}
