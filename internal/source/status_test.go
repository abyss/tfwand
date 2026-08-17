package source

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFind(t *testing.T) {
	t.Run("clean files yield no swaps and no problems", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "main.tf")
		content := "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n"
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		swaps, problems, err := Find([]string{file})
		if err != nil {
			t.Fatal(err)
		}
		if len(swaps) != 0 {
			t.Errorf("got %d swaps, want 0", len(swaps))
		}
		if len(problems) != 0 {
			t.Errorf("got %d problems, want 0", len(problems))
		}
	})

	t.Run("one active swap is reported with File, Line, Source and Local all correct", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "main.tf")
		content := "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n" +
			"source = \"../../some-repo/modules/vpc\"\n"
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		swaps, problems, err := Find([]string{file})
		if err != nil {
			t.Fatal(err)
		}
		if len(problems) != 0 {
			t.Fatalf("got %d problems, want 0", len(problems))
		}
		if len(swaps) != 1 {
			t.Fatalf("got %d swaps, want 1", len(swaps))
		}
		want := Swap{
			File:   file,
			Line:   1,
			Source: "git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0",
			Local:  "../../some-repo/modules/vpc",
		}
		if swaps[0] != want {
			t.Errorf("got %+v, want %+v", swaps[0], want)
		}
	})

	t.Run("an unverifiable marker yields a Problem with File set", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "main.tf")
		content := "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n" +
			"not a local path\n"
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		swaps, problems, err := Find([]string{file})
		if err != nil {
			t.Fatal(err)
		}
		if len(swaps) != 0 {
			t.Errorf("got %d swaps, want 0", len(swaps))
		}
		if len(problems) != 1 {
			t.Fatalf("got %d problems, want 1", len(problems))
		}
		if problems[0].File != file {
			t.Errorf("problem.File = %q, want %q", problems[0].File, file)
		}
	})

	t.Run("swaps across two files are both reported", func(t *testing.T) {
		dir := t.TempDir()
		fileA := filepath.Join(dir, "a.tf")
		fileB := filepath.Join(dir, "b.tf")
		contentA := "# source = \"git@github.com:org/repo-a.git//modules/a?ref=v1.0.0\"\n" +
			"source = \"../../repo-a/modules/a\"\n"
		contentB := "# source = \"git@github.com:org/repo-b.git//modules/b?ref=v2.0.0\"\n" +
			"source = \"../../repo-b/modules/b\"\n"
		if err := os.WriteFile(fileA, []byte(contentA), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fileB, []byte(contentB), 0o644); err != nil {
			t.Fatal(err)
		}

		swaps, problems, err := Find([]string{fileA, fileB})
		if err != nil {
			t.Fatal(err)
		}
		if len(problems) != 0 {
			t.Fatalf("got %d problems, want 0", len(problems))
		}
		if len(swaps) != 2 {
			t.Fatalf("got %d swaps, want 2", len(swaps))
		}
		files := map[string]bool{swaps[0].File: true, swaps[1].File: true}
		if !files[fileA] || !files[fileB] {
			t.Errorf("got swaps from files %v, want both %q and %q", files, fileA, fileB)
		}
	})
}

func TestStatus(t *testing.T) {
	t.Run("clean files: check true returns nil", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "main.tf")
		content := "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n"
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := Status([]string{file}, true); err != nil {
			t.Errorf("got error %v, want nil", err)
		}
	})

	t.Run("an active swap: check true returns an error, check false returns nil", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "main.tf")
		content := "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n" +
			"source = \"../../some-repo/modules/vpc\"\n"
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := Status([]string{file}, true); err == nil {
			t.Error("check=true: expected an error, got nil")
		}
		if err := Status([]string{file}, false); err != nil {
			t.Errorf("check=false: got error %v, want nil", err)
		}
	})

	t.Run("an unverifiable marker: check true returns an error", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "main.tf")
		content := "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n" +
			"not a local path\n"
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		if err := Status([]string{file}, true); err == nil {
			t.Error("expected an error, got nil")
		}
	})

	t.Run("empty files slice returns nil regardless of check", func(t *testing.T) {
		if err := Status(nil, true); err != nil {
			t.Errorf("check=true: got error %v, want nil", err)
		}
		if err := Status(nil, false); err != nil {
			t.Errorf("check=false: got error %v, want nil", err)
		}
	})
}
