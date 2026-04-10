package tffiles

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// FindDirs walks root recursively and returns a sorted, deduplicated list of
// directories that contain at least one .tf file. Directories named .terraform
// are skipped entirely.
func FindDirs(root string) ([]string, error) {
	seen := map[string]struct{}{}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".terraform" {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".tf") {
			dir := filepath.Dir(path)
			seen[dir] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	dirs := make([]string, 0, len(seen))
	for d := range seen {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	return dirs, nil
}
