package tffiles

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// isTF reports whether name is a Terraform config file, so the recursive and
// non-recursive listings below agree on what counts as one.
func isTF(name string) bool {
	return strings.HasSuffix(name, ".tf")
}

// WalkFiles walks root and calls fn for every .tf file, skipping .terraform dirs.
func WalkFiles(root string, fn func(path string) error) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".terraform" {
			return filepath.SkipDir
		}
		if d.IsDir() || !isTF(d.Name()) {
			return nil
		}
		return fn(path)
	})
}

// FindDirs walks root recursively and returns a sorted, deduplicated list of
// directories that contain at least one .tf file. Directories named .terraform
// are skipped entirely.
func FindDirs(root string) ([]string, error) {
	seen := map[string]struct{}{}

	err := WalkFiles(root, func(path string) error {
		seen[filepath.Dir(path)] = struct{}{}
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

// Files returns every .tf file under root, sorted, skipping .terraform dirs.
func Files(root string) ([]string, error) {
	var files []string

	err := WalkFiles(root, func(path string) error {
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

// FilesInDir returns the .tf files directly inside dir, sorted, without recursing.
func FilesInDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() || !isTF(e.Name()) {
			continue
		}
		files = append(files, filepath.Join(dir, e.Name()))
	}
	sort.Strings(files)
	return files, nil
}
