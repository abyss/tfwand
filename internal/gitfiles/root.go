package gitfiles

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// RepoRoot returns the absolute root of the git repository containing path.
//
// The result is passed through filepath.EvalSymlinks so callers get a fully
// resolved path. This matters on macOS, where git resolves symlinks in its
// output (e.g. reporting /private/var/... for a path under /var/...), which
// would otherwise disagree with paths callers have already resolved via
// filepath.EvalSymlinks themselves.
func RepoRoot(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = abs
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}

	root := strings.TrimSpace(string(out))

	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	return resolved, nil
}
