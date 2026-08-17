package source

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemote(t *testing.T) {
	t.Run("full round trip: Local then Remote leaves an LF file byte-identical to the original", func(t *testing.T) {
		_, consumerRoot := newSiblingLayout(t, "some-repo")
		file := filepath.Join(consumerRoot, "main.tf")
		original := "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n" +
			"resource \"aws_instance\" \"x\" {}\n"
		if err := os.WriteFile(file, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := Local([]string{file}); err != nil {
			t.Fatal(err)
		}
		if err := Remote([]string{file}); err != nil {
			t.Fatal(err)
		}

		got, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != original {
			t.Errorf("\ngot:\n%q\nwant:\n%q", got, original)
		}
	})

	t.Run("full round trip: Local then Remote leaves a CRLF file byte-identical to the original", func(t *testing.T) {
		_, consumerRoot := newSiblingLayout(t, "some-repo")
		file := filepath.Join(consumerRoot, "main.tf")
		original := "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\r\n" +
			"resource \"aws_instance\" \"x\" {}\r\n"
		if err := os.WriteFile(file, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := Local([]string{file}); err != nil {
			t.Fatal(err)
		}
		if err := Remote([]string{file}); err != nil {
			t.Fatal(err)
		}

		got, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != original {
			t.Errorf("\ngot:\n%q\nwant:\n%q", got, original)
		}
	})

	t.Run("a hand-inserted line between a marker and its local-path line survives and Remote returns an error", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "main.tf")
		inserted := "resource \"aws_instance\" \"x\" {}\n"
		content := "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n" +
			inserted +
			"source = \"../../some-repo/modules/vpc\"\n"
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		err := Remote([]string{file})
		if err == nil {
			t.Fatal("expected an error, got nil")
		}

		got, err2 := os.ReadFile(file)
		if err2 != nil {
			t.Fatal(err2)
		}
		if !strings.Contains(string(got), inserted) {
			t.Errorf("inserted line was lost, got:\n%s", got)
		}
		if !strings.HasPrefix(string(got), "# source =") {
			t.Errorf("marker was uncommented despite failing verification, got:\n%s", got)
		}
	})

	t.Run("one verified block and one unverified block: the verified one is restored, the other reported, error returned", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "main.tf")
		content := "# source = \"git@github.com:org/repo-a.git//modules/a?ref=v1.0.0\"\n" +
			"source = \"../../repo-a/modules/a\"\n" +
			"# source = \"git@github.com:org/repo-b.git//modules/b?ref=v2.0.0\"\n" +
			"# manual note\n"
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		err := Remote([]string{file})
		if err == nil {
			t.Fatal("expected an error, got nil")
		}

		want := "source = \"git@github.com:org/repo-a.git//modules/a?ref=v1.0.0\"\n" +
			"# source = \"git@github.com:org/repo-b.git//modules/b?ref=v2.0.0\"\n" +
			"# manual note\n"
		got, err2 := os.ReadFile(file)
		if err2 != nil {
			t.Fatal(err2)
		}
		if string(got) != want {
			t.Errorf("\ngot:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("a file with no markers is left byte-identical and Remote returns nil", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "main.tf")
		original := "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n"
		if err := os.WriteFile(file, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := Remote([]string{file}); err != nil {
			t.Fatal(err)
		}

		got, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != original {
			t.Errorf("got %q, want %q", got, original)
		}
	})

	t.Run("running Remote twice is idempotent", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "main.tf")
		content := "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n" +
			"source = \"../../some-repo/modules/vpc\"\n"
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := Remote([]string{file}); err != nil {
			t.Fatal(err)
		}
		afterFirst, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}

		if err := Remote([]string{file}); err != nil {
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

	t.Run("multiple files, mixed clean and swapped", func(t *testing.T) {
		dir := t.TempDir()
		cleanFile := filepath.Join(dir, "clean.tf")
		swappedFile := filepath.Join(dir, "swapped.tf")

		cleanContent := "source = \"git@github.com:org/other-repo.git//modules/x?ref=v1.0.0\"\n"
		swappedContent := "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n" +
			"source = \"../../some-repo/modules/vpc\"\n"
		if err := os.WriteFile(cleanFile, []byte(cleanContent), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(swappedFile, []byte(swappedContent), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := Remote([]string{cleanFile, swappedFile}); err != nil {
			t.Fatal(err)
		}

		gotClean, err := os.ReadFile(cleanFile)
		if err != nil {
			t.Fatal(err)
		}
		if string(gotClean) != cleanContent {
			t.Errorf("clean file changed, got %q, want %q", gotClean, cleanContent)
		}

		wantSwapped := "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n"
		gotSwapped, err := os.ReadFile(swappedFile)
		if err != nil {
			t.Fatal(err)
		}
		if string(gotSwapped) != wantSwapped {
			t.Errorf("got %q, want %q", gotSwapped, wantSwapped)
		}
	})

	t.Run("empty files slice returns nil and does nothing", func(t *testing.T) {
		if err := Remote(nil); err != nil {
			t.Errorf("got error %v, want nil", err)
		}
	})
}
