package source

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/abyss/tfwand/internal/gitfiles"
)

// resolver resolves sibling-checkout paths, caching the git root per directory so a
// recursive run shells out to git once per directory rather than once per source.
type resolver struct {
	roots map[string]string
}

// newResolver returns a resolver with an initialised cache.
func newResolver() *resolver {
	return &resolver{roots: make(map[string]string)}
}

// localPath returns the relative source path to write for a git source found in tfDir.
// It locates tfDir's repo root, treats <parent-of-root>/<repoName> as the dependency
// checkout, verifies that directory and the module subpath both exist, and derives the
// path with filepath.Rel.
func (r *resolver) localPath(tfDir, repoName, modulePath string) (string, error) {
	abs, err := filepath.Abs(tfDir)
	if err != nil {
		return "", err
	}
	dir, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}

	root, ok := r.roots[dir]
	if !ok {
		root, err = gitfiles.RepoRoot(dir)
		if err != nil {
			return "", err
		}
		r.roots[dir] = root
	}

	checkout := filepath.Join(filepath.Dir(root), repoName)
	info, err := os.Stat(checkout)
	if err != nil {
		return "", fmt.Errorf("sibling checkout not found: %s (clone %s alongside %s)", checkout, repoName, root)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("sibling checkout path is not a directory: %s", checkout)
	}

	target := filepath.Join(checkout, modulePath)
	tinfo, err := os.Stat(target)
	if err != nil || !tinfo.IsDir() {
		return "", fmt.Errorf("module path not found in sibling checkout: %s", target)
	}

	rel, err := filepath.Rel(dir, target)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}
