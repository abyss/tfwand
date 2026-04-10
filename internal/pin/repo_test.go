package pin_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/abyss/tfwand/internal/pin"
)

func TestUpdateRepo(t *testing.T) {
	tests := []struct {
		name     string
		repo     string
		version  string
		input    string
		expected string
	}{
		// ── all repo.git variants should be updated ───────────────────────────
		{
			name:     "raw repo, no version",
			repo:     "repo",
			version:  "v3.0.0",
			input:    `source = "git@github.com:example/repo.git"`,
			expected: `source = "git@github.com:example/repo.git?ref=v3.0.0"`,
		},
		{
			name:     "repo with existing version",
			repo:     "repo",
			version:  "v3.0.0",
			input:    `source = "git@github.com:example/repo.git?ref=v1.0.0"`,
			expected: `source = "git@github.com:example/repo.git?ref=v3.0.0"`,
		},
		{
			name:     "repo with subdirectory, no version",
			repo:     "repo",
			version:  "v3.0.0",
			input:    `source = "git@github.com:example/repo.git//test"`,
			expected: `source = "git@github.com:example/repo.git//test?ref=v3.0.0"`,
		},
		{
			name:     "repo with subdirectory and existing version",
			repo:     "repo",
			version:  "v3.0.0",
			input:    `source = "git@github.com:example/repo.git//test?ref=v1.0.0"`,
			expected: `source = "git@github.com:example/repo.git//test?ref=v3.0.0"`,
		},
		{
			name:     "repo with nested subdirectory, no version",
			repo:     "repo",
			version:  "v3.0.0",
			input:    `source = "git@github.com:example/repo.git//test/example"`,
			expected: `source = "git@github.com:example/repo.git//test/example?ref=v3.0.0"`,
		},
		{
			name:     "repo with nested subdirectory and existing version",
			repo:     "repo",
			version:  "v3.0.0",
			input:    `source = "git@github.com:example/repo.git//test/example?ref=v1"`,
			expected: `source = "git@github.com:example/repo.git//test/example?ref=v3.0.0"`,
		},
		// ── non-GitHub SSH hosts ──────────────────────────────────────────────
		{
			name:     "GitLab SSH raw repo",
			repo:     "repo",
			version:  "v3.0.0",
			input:    `source = "git@gitlab.com:example/repo.git"`,
			expected: `source = "git@gitlab.com:example/repo.git?ref=v3.0.0"`,
		},
		{
			name:     "Bitbucket SSH with subdirectory",
			repo:     "repo",
			version:  "v3.0.0",
			input:    `source = "git@bitbucket.org:example/repo.git//modules?ref=v1.0.0"`,
			expected: `source = "git@bitbucket.org:example/repo.git//modules?ref=v3.0.0"`,
		},
		{
			name:     "self-hosted SSH matched",
			repo:     "repo",
			version:  "v3.0.0",
			input:    `source = "git@git.internal.example.com:example/repo.git//modules/network"`,
			expected: `source = "git@git.internal.example.com:example/repo.git//modules/network?ref=v3.0.0"`,
		},
		{
			name:     "explicit ssh:// protocol matched",
			repo:     "repo",
			version:  "v3.0.0",
			input:    `source = "ssh://git@git.example.com/example/repo.git//modules"`,
			expected: `source = "ssh://git@git.example.com/example/repo.git//modules?ref=v3.0.0"`,
		},
		// ── non-GitHub HTTPS hosts ────────────────────────────────────────────
		{
			name:     "GitLab HTTPS raw repo",
			repo:     "repo",
			version:  "v3.0.0",
			input:    `source = "https://gitlab.com/example/repo.git"`,
			expected: `source = "https://gitlab.com/example/repo.git?ref=v3.0.0"`,
		},
		{
			name:     "Bitbucket HTTPS with subdirectory and version",
			repo:     "repo",
			version:  "v3.0.0",
			input:    `source = "https://bitbucket.org/example/repo.git//modules/test?ref=v2.0.0"`,
			expected: `source = "https://bitbucket.org/example/repo.git//modules/test?ref=v3.0.0"`,
		},
		{
			name:     "self-hosted HTTPS matched",
			repo:     "repo",
			version:  "v3.0.0",
			input:    `source = "https://git.example.com/example/repo.git//modules"`,
			expected: `source = "https://git.example.com/example/repo.git//modules?ref=v3.0.0"`,
		},
		{
			name:     "HTTPS with embedded credentials matched",
			repo:     "repo",
			version:  "v3.0.0",
			input:    `source = "https://user:token@git.example.com/example/repo.git//modules"`,
			expected: `source = "https://user:token@git.example.com/example/repo.git//modules?ref=v3.0.0"`,
		},
		{
			name:     "HTTPS with non-standard port matched",
			repo:     "repo",
			version:  "v3.0.0",
			input:    `source = "https://git.example.com:8443/example/repo.git//modules"`,
			expected: `source = "https://git.example.com:8443/example/repo.git//modules?ref=v3.0.0"`,
		},
		// ── GitHub HTTPS (existing coverage, kept for completeness) ───────────
		{
			name:     "GitHub HTTPS raw repo",
			repo:     "repo",
			version:  "v3.0.0",
			input:    `source = "https://github.com/example/repo.git"`,
			expected: `source = "https://github.com/example/repo.git?ref=v3.0.0"`,
		},
		{
			name:     "GitHub HTTPS with subdirectory and version",
			repo:     "repo",
			version:  "v3.0.0",
			input:    `source = "https://github.com/example/repo.git//modules/test?ref=v2.0.0"`,
			expected: `source = "https://github.com/example/repo.git//modules/test?ref=v3.0.0"`,
		},
		// ── non-matches ───────────────────────────────────────────────────────
		{
			name:     "different repo name not matched",
			repo:     "repo",
			version:  "v3.0.0",
			input:    `source = "git@github.com:example/other-repo.git"`,
			expected: `source = "git@github.com:example/other-repo.git"`,
		},
		{
			name:     "different repo with subdirectory not matched",
			repo:     "repo",
			version:  "v3.0.0",
			input:    `source = "git@github.com:example/other-repo.git//some/path"`,
			expected: `source = "git@github.com:example/other-repo.git//some/path"`,
		},
		{
			name:     "similar repo name not matched",
			repo:     "repo",
			version:  "v3.0.0",
			input:    `source = "git@github.com:example/repo-extended.git//test"`,
			expected: `source = "git@github.com:example/repo-extended.git//test"`,
		},
		// ── multi-module file ─────────────────────────────────────────────────
		{
			name:    "all subdirs of same repo updated, other repos untouched",
			repo:    "repo",
			version: "v3.0.0",
			input: `module "vpc" {
  source = "git@github.com:example/repo.git//network?ref=v2.5.0"
}

module "db" {
  source = "git@github.com:example/repo.git//database?ref=v2.3.0"
}

module "other" {
  source = "git@github.com:example/other-repo.git//module?ref=v1.0.0"
}`,
			expected: `module "vpc" {
  source = "git@github.com:example/repo.git//network?ref=v3.0.0"
}

module "db" {
  source = "git@github.com:example/repo.git//database?ref=v3.0.0"
}

module "other" {
  source = "git@github.com:example/other-repo.git//module?ref=v1.0.0"
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "main.tf")

			if err := os.WriteFile(file, []byte(tt.input), 0o644); err != nil {
				t.Fatal(err)
			}

			if err := pin.UpdateRepo(dir, tt.repo, tt.version); err != nil {
				t.Fatalf("UpdateRepo: %v", err)
			}

			got, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.expected {
				t.Errorf("\ngot:\n%s\nwant:\n%s", got, tt.expected)
			}
		})
	}
}
