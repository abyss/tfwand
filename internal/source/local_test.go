package source

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocal(t *testing.T) {
	t.Run("a sub-module source is swapped and content matches the commented original plus the local path line", func(t *testing.T) {
		_, consumerRoot := newSiblingLayout(t, "some-repo")
		file := filepath.Join(consumerRoot, "main.tf")
		original := "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n"
		if err := os.WriteFile(file, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := Local([]string{file}); err != nil {
			t.Fatal(err)
		}

		want := "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n" +
			"source = \"../some-repo/modules/vpc\"\n"
		got, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("\ngot:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("a root-module source swaps to the bare sibling directory", func(t *testing.T) {
		_, consumerRoot := newSiblingLayout(t, "some-repo")
		file := filepath.Join(consumerRoot, "main.tf")
		original := "source = \"git@github.com:org/some-repo.git?ref=v1.2.0\"\n"
		if err := os.WriteFile(file, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := Local([]string{file}); err != nil {
			t.Fatal(err)
		}

		want := "# source = \"git@github.com:org/some-repo.git?ref=v1.2.0\"\n" +
			"source = \"../some-repo\"\n"
		got, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("\ngot:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("a file with two sources has both swapped", func(t *testing.T) {
		parent, consumerRoot := newSiblingLayout(t, "repo-a")
		if err := os.MkdirAll(filepath.Join(parent, "repo-b", "modules", "b"), 0o755); err != nil {
			t.Fatal(err)
		}
		file := filepath.Join(consumerRoot, "main.tf")
		original := "source = \"git@github.com:org/repo-a.git//modules/vpc?ref=v1.0.0\"\n" +
			"source = \"git@github.com:org/repo-b.git//modules/b?ref=v2.0.0\"\n"
		if err := os.WriteFile(file, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := Local([]string{file}); err != nil {
			t.Fatal(err)
		}

		want := "# source = \"git@github.com:org/repo-a.git//modules/vpc?ref=v1.0.0\"\n" +
			"source = \"../repo-a/modules/vpc\"\n" +
			"# source = \"git@github.com:org/repo-b.git//modules/b?ref=v2.0.0\"\n" +
			"source = \"../repo-b/modules/b\"\n"
		got, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("\ngot:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("multiple files in one call all get swapped", func(t *testing.T) {
		_, consumerRoot := newSiblingLayout(t, "some-repo")
		fileA := filepath.Join(consumerRoot, "a.tf")
		fileB := filepath.Join(consumerRoot, "b.tf")
		original := "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n"
		if err := os.WriteFile(fileA, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fileB, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := Local([]string{fileA, fileB}); err != nil {
			t.Fatal(err)
		}

		want := "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n" +
			"source = \"../some-repo/modules/vpc\"\n"
		for _, f := range []string{fileA, fileB} {
			got, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != want {
				t.Errorf("%s:\ngot:\n%s\nwant:\n%s", f, got, want)
			}
		}
	})

	t.Run("a file with zero git sources is left byte-identical", func(t *testing.T) {
		_, consumerRoot := newSiblingLayout(t, "some-repo")
		file := filepath.Join(consumerRoot, "main.tf")
		original := "resource \"aws_instance\" \"x\" {}\n"
		if err := os.WriteFile(file, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(file)
		if err != nil {
			t.Fatal(err)
		}
		mtime := info.ModTime()

		if err := Local([]string{file}); err != nil {
			t.Fatal(err)
		}

		got, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != original {
			t.Errorf("got %q, want %q", got, original)
		}
		info, err = os.Stat(file)
		if err != nil {
			t.Fatal(err)
		}
		if !info.ModTime().Equal(mtime) {
			t.Errorf("file was rewritten even though it had no git sources")
		}
	})

	t.Run("running Local twice leaves the file identical to after the first run", func(t *testing.T) {
		_, consumerRoot := newSiblingLayout(t, "some-repo")
		file := filepath.Join(consumerRoot, "main.tf")
		original := "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n"
		if err := os.WriteFile(file, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := Local([]string{file}); err != nil {
			t.Fatal(err)
		}
		afterFirst, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}

		if err := Local([]string{file}); err != nil {
			t.Fatal(err)
		}
		afterSecond, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}

		if string(afterSecond) != string(afterFirst) {
			t.Errorf("second run changed content\nafter first:\n%s\nafter second:\n%s", afterFirst, afterSecond)
		}
	})

	t.Run("all-or-nothing: a missing sibling checkout in file two leaves file one untouched and returns an error", func(t *testing.T) {
		_, consumerRoot := newSiblingLayout(t, "repo-a")
		fileA := filepath.Join(consumerRoot, "a.tf")
		fileB := filepath.Join(consumerRoot, "b.tf")

		originalA := "source = \"git@github.com:org/repo-a.git//modules/vpc?ref=v1.0.0\"\n"
		originalB := "source = \"git@github.com:org/repo-missing.git//modules/x?ref=v1.0.0\"\n"
		if err := os.WriteFile(fileA, []byte(originalA), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fileB, []byte(originalB), 0o644); err != nil {
			t.Fatal(err)
		}

		err := Local([]string{fileA, fileB})
		if err == nil {
			t.Fatal("expected an error, got nil")
		}

		gotA, err2 := os.ReadFile(fileA)
		if err2 != nil {
			t.Fatal(err2)
		}
		if string(gotA) != originalA {
			t.Errorf("file A was modified despite file B failing validation\ngot:\n%s\nwant (unchanged):\n%s", gotA, originalA)
		}

		gotB, err2 := os.ReadFile(fileB)
		if err2 != nil {
			t.Fatal(err2)
		}
		if string(gotB) != originalB {
			t.Errorf("file B was modified despite failing validation\ngot:\n%s\nwant (unchanged):\n%s", gotB, originalB)
		}
	})

	t.Run("a missing sibling checkout error mentions the expected path", func(t *testing.T) {
		parent := t.TempDir()
		consumerRoot := filepath.Join(parent, "my-stacks")
		if err := os.MkdirAll(consumerRoot, 0o755); err != nil {
			t.Fatal(err)
		}
		initGitConsumerRepo(t, consumerRoot)

		file := filepath.Join(consumerRoot, "main.tf")
		original := "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n"
		if err := os.WriteFile(file, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}

		err := Local([]string{file})
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		wantSubstr := filepath.Join(parent, "some-repo")
		if !strings.Contains(err.Error(), wantSubstr) {
			t.Errorf("error %q does not mention expected path %q", err.Error(), wantSubstr)
		}
	})

	t.Run("empty files slice returns nil and does nothing", func(t *testing.T) {
		if err := Local(nil); err != nil {
			t.Errorf("got error %v, want nil", err)
		}
	})

	t.Run("a registry source in the same file as a git source: only the git one changes", func(t *testing.T) {
		_, consumerRoot := newSiblingLayout(t, "some-repo")
		file := filepath.Join(consumerRoot, "main.tf")
		original := "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n" +
			"source = \"terraform-aws-modules/vpc/aws\"\n"
		if err := os.WriteFile(file, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := Local([]string{file}); err != nil {
			t.Fatal(err)
		}

		want := "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n" +
			"source = \"../some-repo/modules/vpc\"\n" +
			"source = \"terraform-aws-modules/vpc/aws\"\n"
		got, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("\ngot:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("file permissions survive a write", func(t *testing.T) {
		_, consumerRoot := newSiblingLayout(t, "some-repo")
		file := filepath.Join(consumerRoot, "main.tf")
		original := "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n"
		if err := os.WriteFile(file, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := Local([]string{file}); err != nil {
			t.Fatal(err)
		}

		info, err := os.Stat(file)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o644 {
			t.Errorf("mode = %v, want 0644", info.Mode().Perm())
		}
		got, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		want := "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n" +
			"source = \"../some-repo/modules/vpc\"\n"
		if string(got) != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
