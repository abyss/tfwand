package tffiles_test

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/abyss/tfwand/internal/tffiles"
)

func TestFindDirs(t *testing.T) {
	t.Run("empty directory returns no dirs", func(t *testing.T) {
		dir := t.TempDir()
		got, err := tffiles.FindDirs(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty, got %v", got)
		}
	})

	t.Run("single .tf file returns its parent dir", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "main.tf")

		got, err := tffiles.FindDirs(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0] != dir {
			t.Errorf("expected [%s], got %v", dir, got)
		}
	})

	t.Run("non-.tf files not included", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "README.md")
		touch(t, dir, "variables.json")

		got, err := tffiles.FindDirs(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty, got %v", got)
		}
	})

	t.Run("nested dirs each returned", func(t *testing.T) {
		root := t.TempDir()
		subA := filepath.Join(root, "envs", "dev")
		subB := filepath.Join(root, "envs", "prod")
		must(t, os.MkdirAll(subA, 0o755))
		must(t, os.MkdirAll(subB, 0o755))
		touch(t, subA, "main.tf")
		touch(t, subB, "main.tf")

		got, err := tffiles.FindDirs(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 dirs, got %v", got)
		}
		if got[0] != subA || got[1] != subB {
			t.Errorf("expected [%s %s], got %v", subA, subB, got)
		}
	})

	t.Run("multiple .tf files in same dir deduplicated", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "main.tf")
		touch(t, dir, "variables.tf")
		touch(t, dir, "outputs.tf")

		got, err := tffiles.FindDirs(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Errorf("expected 1 dir, got %v", got)
		}
	})

	t.Run(".terraform directories are skipped", func(t *testing.T) {
		root := t.TempDir()
		terraDir := filepath.Join(root, ".terraform")
		must(t, os.MkdirAll(terraDir, 0o755))
		// .tf file inside .terraform — should be ignored
		touch(t, terraDir, "modules.tf")
		// .tf file at root — should be found
		touch(t, root, "main.tf")

		got, err := tffiles.FindDirs(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0] != root {
			t.Errorf("expected only root [%s], got %v", root, got)
		}
	})

	t.Run("results are sorted", func(t *testing.T) {
		root := t.TempDir()
		for _, name := range []string{"z", "a", "m"} {
			sub := filepath.Join(root, name)
			must(t, os.MkdirAll(sub, 0o755))
			touch(t, sub, "main.tf")
		}

		got, err := tffiles.FindDirs(root)
		if err != nil {
			t.Fatal(err)
		}
		for i := 1; i < len(got); i++ {
			if got[i] < got[i-1] {
				t.Errorf("results not sorted: %v", got)
			}
		}
	})
}

func TestWalkFiles(t *testing.T) {
	t.Run("visits only .tf files", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "main.tf")
		touch(t, dir, "README.md")
		touch(t, dir, "variables.json")

		var visited []string
		err := tffiles.WalkFiles(dir, func(path string) error {
			visited = append(visited, path)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(visited) != 1 || visited[0] != filepath.Join(dir, "main.tf") {
			t.Errorf("expected [%s], got %v", filepath.Join(dir, "main.tf"), visited)
		}
	})

	t.Run("skips .terraform directories entirely", func(t *testing.T) {
		root := t.TempDir()
		terraDir := filepath.Join(root, ".terraform")
		must(t, os.MkdirAll(terraDir, 0o755))
		touch(t, terraDir, "modules.tf")
		touch(t, root, "main.tf")

		var visited []string
		err := tffiles.WalkFiles(root, func(path string) error {
			visited = append(visited, path)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(visited) != 1 || visited[0] != filepath.Join(root, "main.tf") {
			t.Errorf("expected only [%s], got %v", filepath.Join(root, "main.tf"), visited)
		}
	})

	t.Run("propagates an error returned by fn", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "main.tf")

		wantErr := errors.New("boom")
		err := tffiles.WalkFiles(dir, func(path string) error {
			return wantErr
		})
		if err != wantErr {
			t.Errorf("expected %v, got %v", wantErr, err)
		}
	})
}

func TestFiles(t *testing.T) {
	t.Run("returns all .tf files recursively, sorted", func(t *testing.T) {
		root := t.TempDir()
		subA := filepath.Join(root, "envs", "dev")
		must(t, os.MkdirAll(subA, 0o755))
		touch(t, root, "main.tf")
		touch(t, subA, "variables.tf")

		got, err := tffiles.Files(root)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{filepath.Join(root, "main.tf"), filepath.Join(subA, "variables.tf")}
		sort.Strings(want)
		if len(got) != len(want) {
			t.Fatalf("expected %v, got %v", want, got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("got %q, want %q", got[i], want[i])
			}
		}
	})

	t.Run("skips .terraform", func(t *testing.T) {
		root := t.TempDir()
		terraDir := filepath.Join(root, ".terraform")
		must(t, os.MkdirAll(terraDir, 0o755))
		touch(t, terraDir, "modules.tf")
		touch(t, root, "main.tf")

		got, err := tffiles.Files(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0] != filepath.Join(root, "main.tf") {
			t.Errorf("expected only [%s], got %v", filepath.Join(root, "main.tf"), got)
		}
	})
}

func TestFilesInDir(t *testing.T) {
	t.Run("returns only files directly in the dir, not from subdirectories", func(t *testing.T) {
		dir := t.TempDir()
		sub := filepath.Join(dir, "sub")
		must(t, os.MkdirAll(sub, 0o755))
		touch(t, dir, "main.tf")
		touch(t, sub, "nested.tf")

		got, err := tffiles.FilesInDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0] != filepath.Join(dir, "main.tf") {
			t.Errorf("expected [%s], got %v", filepath.Join(dir, "main.tf"), got)
		}
	})

	t.Run("ignores non-.tf files", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "main.tf")
		touch(t, dir, "README.md")

		got, err := tffiles.FilesInDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0] != filepath.Join(dir, "main.tf") {
			t.Errorf("expected [%s], got %v", filepath.Join(dir, "main.tf"), got)
		}
	})

	t.Run("on a directory with no .tf files returns empty, no error", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "README.md")

		got, err := tffiles.FilesInDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty, got %v", got)
		}
	})
}

func touch(t *testing.T, dir, name string) {
	t.Helper()
	must(t, os.WriteFile(filepath.Join(dir, name), nil, 0o644))
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
