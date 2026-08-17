package source

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initGitConsumerRepo runs a real `git init` in dir, following the same shape
// as gitfiles_test's initGitRepo helper (which lives in a different package
// and so cannot be imported directly).
func initGitConsumerRepo(t *testing.T, dir string) {
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

// newSiblingLayout builds <parent>/my-stacks (a git repo) alongside
// <parent>/<repoName>/modules/vpc (a plain, non-git dependency checkout),
// and returns the parent dir and the consumer repo root.
func newSiblingLayout(t *testing.T, repoName string) (parent, consumerRoot string) {
	t.Helper()
	parent = t.TempDir()

	consumerRoot = filepath.Join(parent, "my-stacks")
	if err := os.MkdirAll(consumerRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	initGitConsumerRepo(t, consumerRoot)

	vpc := filepath.Join(parent, repoName, "modules", "vpc")
	if err := os.MkdirAll(vpc, 0o755); err != nil {
		t.Fatal(err)
	}

	return parent, consumerRoot
}

func TestLocalPath(t *testing.T) {
	t.Run("resolves from the consuming repo root", func(t *testing.T) {
		_, consumerRoot := newSiblingLayout(t, "some-repo")

		r := newResolver()
		got, err := r.localPath(consumerRoot, "some-repo", "modules/vpc")
		if err != nil {
			t.Fatal(err)
		}
		want := "../some-repo/modules/vpc"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("resolves from a nested directory", func(t *testing.T) {
		_, consumerRoot := newSiblingLayout(t, "some-repo")
		nested := filepath.Join(consumerRoot, "stacks", "prod")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatal(err)
		}

		r := newResolver()
		got, err := r.localPath(nested, "some-repo", "modules/vpc")
		if err != nil {
			t.Fatal(err)
		}
		want := "../../../some-repo/modules/vpc"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("root-module source with empty module path", func(t *testing.T) {
		_, consumerRoot := newSiblingLayout(t, "some-repo")

		r := newResolver()
		got, err := r.localPath(consumerRoot, "some-repo", "")
		if err != nil {
			t.Fatal(err)
		}
		want := "../some-repo"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("nested module subpath", func(t *testing.T) {
		parent, consumerRoot := newSiblingLayout(t, "some-repo")
		nestedModule := filepath.Join(parent, "some-repo", "modules", "aws", "vpc")
		if err := os.MkdirAll(nestedModule, 0o755); err != nil {
			t.Fatal(err)
		}

		r := newResolver()
		got, err := r.localPath(consumerRoot, "some-repo", "modules/aws/vpc")
		if err != nil {
			t.Fatal(err)
		}
		want := "../some-repo/modules/aws/vpc"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("missing sibling checkout returns an error naming the expected path", func(t *testing.T) {
		parent := t.TempDir()
		consumerRoot := filepath.Join(parent, "my-stacks")
		if err := os.MkdirAll(consumerRoot, 0o755); err != nil {
			t.Fatal(err)
		}
		initGitConsumerRepo(t, consumerRoot)

		r := newResolver()
		_, err := r.localPath(consumerRoot, "some-repo", "modules/vpc")
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		wantSubstr := filepath.Join(parent, "some-repo")
		if !strings.Contains(err.Error(), wantSubstr) {
			t.Errorf("error %q does not mention expected path %q", err.Error(), wantSubstr)
		}
		if !strings.Contains(err.Error(), "sibling checkout not found") {
			t.Errorf("error %q missing expected wording", err.Error())
		}
	})

	t.Run("sibling checkout exists but module subpath does not", func(t *testing.T) {
		_, consumerRoot := newSiblingLayout(t, "some-repo")

		r := newResolver()
		_, err := r.localPath(consumerRoot, "some-repo", "modules/does-not-exist")
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		if !strings.Contains(err.Error(), "module path not found in sibling checkout") {
			t.Errorf("error %q missing expected wording", err.Error())
		}
	})

	t.Run("sibling path exists but is a file, not a directory", func(t *testing.T) {
		parent := t.TempDir()
		consumerRoot := filepath.Join(parent, "my-stacks")
		if err := os.MkdirAll(consumerRoot, 0o755); err != nil {
			t.Fatal(err)
		}
		initGitConsumerRepo(t, consumerRoot)

		if err := os.WriteFile(filepath.Join(parent, "some-repo"), []byte("not a dir"), 0o644); err != nil {
			t.Fatal(err)
		}

		r := newResolver()
		_, err := r.localPath(consumerRoot, "some-repo", "modules/vpc")
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		if !strings.Contains(err.Error(), "not a directory") {
			t.Errorf("error %q missing expected wording", err.Error())
		}
	})

	t.Run("tfDir is not inside a git repository", func(t *testing.T) {
		dir := t.TempDir()

		r := newResolver()
		_, err := r.localPath(dir, "some-repo", "modules/vpc")
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
	})

	t.Run("the cache is populated per directory, not per call", func(t *testing.T) {
		_, consumerRoot := newSiblingLayout(t, "some-repo")

		r := newResolver()
		got1, err := r.localPath(consumerRoot, "some-repo", "modules/vpc")
		if err != nil {
			t.Fatal(err)
		}
		got2, err := r.localPath(consumerRoot, "some-repo", "modules/vpc")
		if err != nil {
			t.Fatal(err)
		}
		if got1 != got2 {
			t.Errorf("got %q and %q, want identical results", got1, got2)
		}
		if len(r.roots) != 1 {
			t.Errorf("len(r.roots) = %d, want 1 after two calls for the same dir", len(r.roots))
		}

		nested := filepath.Join(consumerRoot, "stacks", "prod")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := r.localPath(nested, "some-repo", "modules/vpc"); err != nil {
			t.Fatal(err)
		}
		if len(r.roots) != 2 {
			t.Errorf("len(r.roots) = %d, want 2 after calls for two different dirs", len(r.roots))
		}
	})

	t.Run("repo name containing a dot", func(t *testing.T) {
		_, consumerRoot := newSiblingLayout(t, "foo.bar")

		r := newResolver()
		got, err := r.localPath(consumerRoot, "foo.bar", "modules/vpc")
		if err != nil {
			t.Fatal(err)
		}
		want := "../foo.bar/modules/vpc"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("returned path uses forward slashes", func(t *testing.T) {
		_, consumerRoot := newSiblingLayout(t, "some-repo")

		r := newResolver()
		got, err := r.localPath(consumerRoot, "some-repo", "modules/vpc")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(got, `\`) {
			t.Errorf("got %q, want no backslashes", got)
		}
	})
}
