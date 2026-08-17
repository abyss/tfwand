package gitfiles_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/abyss/tfwand/internal/gitfiles"
)

func TestRepoRoot(t *testing.T) {
	t.Run("returns the repo root when called with the root itself", func(t *testing.T) {
		root := t.TempDir()
		initGitRepo(t, root)

		got, err := gitfiles.RepoRoot(root)
		if err != nil {
			t.Fatal(err)
		}
		wantRoot, err := filepath.EvalSymlinks(root)
		must(t, err)
		if got != wantRoot {
			t.Errorf("got %q, want %q", got, wantRoot)
		}
	})

	t.Run("returns the repo root when called from a nested subdirectory", func(t *testing.T) {
		root := t.TempDir()
		initGitRepo(t, root)

		sub := filepath.Join(root, "modules", "network")
		must(t, os.MkdirAll(sub, 0o755))

		got, err := gitfiles.RepoRoot(sub)
		if err != nil {
			t.Fatal(err)
		}
		wantRoot, err := filepath.EvalSymlinks(root)
		must(t, err)
		if got != wantRoot {
			t.Errorf("got %q, want %q", got, wantRoot)
		}
	})

	t.Run("returns an absolute path", func(t *testing.T) {
		root := t.TempDir()
		initGitRepo(t, root)

		got, err := gitfiles.RepoRoot(root)
		if err != nil {
			t.Fatal(err)
		}
		if !filepath.IsAbs(got) {
			t.Errorf("expected absolute path, got %q", got)
		}
	})

	t.Run("returns an error for a directory that is not in a git repository", func(t *testing.T) {
		dir := t.TempDir()

		_, err := gitfiles.RepoRoot(dir)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("result is stable when called from two different subdirectories of the same repo", func(t *testing.T) {
		root := t.TempDir()
		initGitRepo(t, root)

		subA := filepath.Join(root, "a")
		subB := filepath.Join(root, "b", "c")
		must(t, os.MkdirAll(subA, 0o755))
		must(t, os.MkdirAll(subB, 0o755))

		gotA, err := gitfiles.RepoRoot(subA)
		if err != nil {
			t.Fatal(err)
		}
		gotB, err := gitfiles.RepoRoot(subB)
		if err != nil {
			t.Fatal(err)
		}
		if gotA != gotB {
			t.Errorf("got %q and %q, want equal", gotA, gotB)
		}
	})
}
