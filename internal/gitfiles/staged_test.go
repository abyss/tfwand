package gitfiles_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/abyss/tfwand/internal/gitfiles"
)

func TestStagedDirs(t *testing.T) {
	t.Run("no staged changes returns empty list", func(t *testing.T) {
		root := t.TempDir()
		initGitRepo(t, root)

		got, err := gitfiles.StagedDirs(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty, got %v", got)
		}
	})

	t.Run("unstaged .tf file not included", func(t *testing.T) {
		root := t.TempDir()
		initGitRepo(t, root)
		must(t, os.WriteFile(filepath.Join(root, "main.tf"), nil, 0o644))
		// not staged — don't git add

		got, err := gitfiles.StagedDirs(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty (unstaged), got %v", got)
		}
	})

	t.Run("staged .tf file returns its directory", func(t *testing.T) {
		root := t.TempDir()
		initGitRepo(t, root)

		sub := filepath.Join(root, "modules", "network")
		must(t, os.MkdirAll(sub, 0o755))
		must(t, os.WriteFile(filepath.Join(sub, "main.tf"), nil, 0o644))
		gitExec(t, root, "add", filepath.Join("modules", "network", "main.tf"))

		got, err := gitfiles.StagedDirs(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 dir, got %v", got)
		}
		want := filepath.Join("modules", "network")
		if got[0] != want {
			t.Errorf("got %q, want %q", got[0], want)
		}
	})

	t.Run("staged non-.tf file not included", func(t *testing.T) {
		root := t.TempDir()
		initGitRepo(t, root)
		must(t, os.WriteFile(filepath.Join(root, "README.md"), []byte("hi"), 0o644))
		gitExec(t, root, "add", "README.md")

		got, err := gitfiles.StagedDirs(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty, got %v", got)
		}
	})

	t.Run("multiple staged files in same dir deduplicated", func(t *testing.T) {
		root := t.TempDir()
		initGitRepo(t, root)

		sub := filepath.Join(root, "env")
		must(t, os.MkdirAll(sub, 0o755))
		must(t, os.WriteFile(filepath.Join(sub, "main.tf"), nil, 0o644))
		must(t, os.WriteFile(filepath.Join(sub, "variables.tf"), nil, 0o644))
		gitExec(t, root, "add", ".")

		got, err := gitfiles.StagedDirs(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Errorf("expected 1 dir, got %v", got)
		}
	})

	t.Run("staged changes in multiple dirs all returned", func(t *testing.T) {
		root := t.TempDir()
		initGitRepo(t, root)

		for _, sub := range []string{"dev", "prod"} {
			path := filepath.Join(root, sub)
			must(t, os.MkdirAll(path, 0o755))
			must(t, os.WriteFile(filepath.Join(path, "main.tf"), nil, 0o644))
		}
		gitExec(t, root, "add", ".")

		got, err := gitfiles.StagedDirs(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Errorf("expected 2 dirs, got %v", got)
		}
	})

	t.Run(".terraform dirs excluded", func(t *testing.T) {
		root := t.TempDir()
		initGitRepo(t, root)

		terraDir := filepath.Join(root, ".terraform")
		must(t, os.MkdirAll(terraDir, 0o755))
		must(t, os.WriteFile(filepath.Join(terraDir, "main.tf"), nil, 0o644))
		gitExec(t, root, "add", ".")

		got, err := gitfiles.StagedDirs(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty (skipped .terraform), got %v", got)
		}
	})

	t.Run("non-ASCII directory name handled correctly", func(t *testing.T) {
		root := t.TempDir()
		initGitRepo(t, root)

		sub := filepath.Join(root, "caf\u00e9")
		must(t, os.MkdirAll(sub, 0o755))
		must(t, os.WriteFile(filepath.Join(sub, "main.tf"), nil, 0o644))
		gitExec(t, root, "add", ".")

		got, err := gitfiles.StagedDirs(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 dir, got %v", got)
		}
		if got[0] != "caf\u00e9" {
			t.Errorf("got %q, want %q", got[0], "caf\u00e9")
		}
	})
}

func gitExec(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
