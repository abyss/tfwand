package gitfiles

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// ChangedDirs runs `git status --porcelain -z` in root and returns a sorted,
// deduplicated list of directories that have git changes and contain at least
// one .tf file. Directories named .terraform are skipped.
//
// The -z flag gives NUL-delimited raw UTF-8 paths, bypassing core.quotePath
// encoding and rename " -> " syntax entirely. Rename entries emit two
// NUL-terminated records; the old-path record has no "XY " prefix and is
// skipped automatically.
func ChangedDirs(root string) ([]string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("git", "status", "--porcelain", "-z")
	cmd.Dir = abs
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}

	seen := map[string]struct{}{}
	for _, entry := range bytes.Split(out, []byte{0}) {
		// Each record is "XY path". Old-path records from renames have no
		// "XY " prefix — len < 3 or entry[2] != ' ' — skip them.
		if len(entry) < 3 || entry[2] != ' ' {
			continue
		}
		filePath := string(entry[3:])

		// git shows entire untracked directories with a trailing slash.
		// Walk the directory for .tf files rather than using filepath.Dir.
		fullPath := filepath.Join(abs, strings.TrimSuffix(filePath, "/"))
		info, statErr := os.Stat(fullPath)
		if statErr == nil && info.IsDir() {
			collectTFDirs(fullPath, abs, seen)
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

func containsTerraDotDir(path string) bool {
	for _, part := range strings.Split(path, string(os.PathSeparator)) {
		if part == ".terraform" {
			return true
		}
	}
	return false
}

// collectTFDirs walks dir recursively and adds any subdirectory containing
// .tf files to seen, using paths relative to abs.
func collectTFDirs(dir, abs string, seen map[string]struct{}) {
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".terraform" {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".tf") {
			parent := filepath.Dir(path)
			rel, relErr := filepath.Rel(abs, parent)
			if relErr != nil {
				rel = parent
			}
			seen[rel] = struct{}{}
		}
		return nil
	})
}

func hasTFFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".tf") {
			return true
		}
	}
	return false
}
