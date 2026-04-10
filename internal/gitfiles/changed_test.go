package gitfiles_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/abyss/tfwand/internal/gitfiles"
)

// initGitRepo creates a bare git repo in dir so git status works.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", dir},
		{"-C", dir, "config", "user.email", "test@example.com"},
		{"-C", dir, "config", "user.name", "Test"},
	} {
		cmd := exec.Command("git", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

func TestChangedDirs(t *testing.T) {
	t.Run("no changes returns empty list", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir)

		got, err := gitfiles.ChangedDirs(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty, got %v", got)
		}
	})

	t.Run("untracked .tf file returns its directory", func(t *testing.T) {
		root := t.TempDir()
		initGitRepo(t, root)

		sub := filepath.Join(root, "modules", "network")
		must(t, os.MkdirAll(sub, 0o755))
		must(t, os.WriteFile(filepath.Join(sub, "main.tf"), []byte(`resource "null_resource" "x" {}`), 0o644))

		got, err := gitfiles.ChangedDirs(root)
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

	t.Run("non-.tf changed file not included", func(t *testing.T) {
		root := t.TempDir()
		initGitRepo(t, root)
		must(t, os.WriteFile(filepath.Join(root, "README.md"), []byte("hi"), 0o644))

		got, err := gitfiles.ChangedDirs(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty, got %v", got)
		}
	})

	t.Run("multiple changed files in same dir deduplicated", func(t *testing.T) {
		root := t.TempDir()
		initGitRepo(t, root)

		sub := filepath.Join(root, "env")
		must(t, os.MkdirAll(sub, 0o755))
		must(t, os.WriteFile(filepath.Join(sub, "main.tf"), nil, 0o644))
		must(t, os.WriteFile(filepath.Join(sub, "variables.tf"), nil, 0o644))

		got, err := gitfiles.ChangedDirs(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Errorf("expected 1 dir, got %v", got)
		}
	})

	t.Run("changes in multiple dirs all returned", func(t *testing.T) {
		root := t.TempDir()
		initGitRepo(t, root)

		for _, sub := range []string{"dev", "prod"} {
			path := filepath.Join(root, sub)
			must(t, os.MkdirAll(path, 0o755))
			must(t, os.WriteFile(filepath.Join(path, "main.tf"), nil, 0o644))
		}

		got, err := gitfiles.ChangedDirs(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Errorf("expected 2 dirs, got %v", got)
		}
	})

	t.Run("non-ASCII directory name decoded correctly", func(t *testing.T) {
		root := t.TempDir()
		initGitRepo(t, root)

		// git will quote this path with octal escapes: "caf\303\251"
		sub := filepath.Join(root, "caf\u00e9") // café
		must(t, os.MkdirAll(sub, 0o755))
		must(t, os.WriteFile(filepath.Join(sub, "main.tf"), nil, 0o644))

		got, err := gitfiles.ChangedDirs(root)
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

	t.Run("path containing ' -> ' not misread as rename", func(t *testing.T) {
		root := t.TempDir()
		initGitRepo(t, root)

		// directory name that contains the rename separator literally
		sub := filepath.Join(root, "a -> b")
		must(t, os.MkdirAll(sub, 0o755))
		must(t, os.WriteFile(filepath.Join(sub, "main.tf"), nil, 0o644))

		got, err := gitfiles.ChangedDirs(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 dir, got %v", got)
		}
		if got[0] != "a -> b" {
			t.Errorf("got %q, want %q", got[0], "a -> b")
		}
	})

	t.Run("renamed .tf file returns new directory", func(t *testing.T) {
		root := t.TempDir()
		initGitRepo(t, root)

		// create and commit a file in old-dir
		oldDir := filepath.Join(root, "old-dir")
		newDir := filepath.Join(root, "new-dir")
		must(t, os.MkdirAll(oldDir, 0o755))
		must(t, os.MkdirAll(newDir, 0o755))
		must(t, os.WriteFile(filepath.Join(oldDir, "main.tf"), nil, 0o644))

		git := func(args ...string) {
			t.Helper()
			cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git %v: %v\n%s", args, err, out)
			}
		}
		git("add", ".")
		git("commit", "-m", "init")

		// rename to new-dir using git mv
		git("mv", filepath.Join("old-dir", "main.tf"), filepath.Join("new-dir", "main.tf"))

		got, err := gitfiles.ChangedDirs(root)
		if err != nil {
			t.Fatal(err)
		}
		// should include new-dir (destination of rename)
		found := false
		for _, d := range got {
			if d == "new-dir" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected new-dir in results, got %v", got)
		}
	})

	t.Run(".terraform dirs excluded", func(t *testing.T) {
		root := t.TempDir()
		initGitRepo(t, root)

		terraDir := filepath.Join(root, ".terraform")
		must(t, os.MkdirAll(terraDir, 0o755))
		must(t, os.WriteFile(filepath.Join(terraDir, "main.tf"), nil, 0o644))

		got, err := gitfiles.ChangedDirs(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty (skipped .terraform), got %v", got)
		}
	})
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
