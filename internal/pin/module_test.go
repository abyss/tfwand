package pin_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/abyss/tfwand/internal/pin"
)

func TestUpdateModule(t *testing.T) {
	tests := []struct {
		name     string
		module   string
		version  string
		input    string
		expected string
	}{
		// ── exact match, no existing version ──────────────────────────────────
		{
			name:     "subdirectory match, no version",
			module:   "test",
			version:  "v2.0.0",
			input:    `source = "git@github.com:example/repo.git//test"`,
			expected: `source = "git@github.com:example/repo.git//test?ref=v2.0.0"`,
		},
		// ── exact match, replace existing version ─────────────────────────────
		{
			name:     "subdirectory match, existing version",
			module:   "test",
			version:  "v2.0.0",
			input:    `source = "git@github.com:example/repo.git//test?ref=v1.0.0"`,
			expected: `source = "git@github.com:example/repo.git//test?ref=v2.0.0"`,
		},
		// ── nested path exact match ───────────────────────────────────────────
		{
			name:     "nested subdirectory match, no version",
			module:   "test/example",
			version:  "v2.0.0",
			input:    `source = "git@github.com:example/repo.git//test/example"`,
			expected: `source = "git@github.com:example/repo.git//test/example?ref=v2.0.0"`,
		},
		{
			name:     "nested subdirectory match, existing version",
			module:   "test/example",
			version:  "v2.0.0",
			input:    `source = "git@github.com:example/repo.git//test/example?ref=v1"`,
			expected: `source = "git@github.com:example/repo.git//test/example?ref=v2.0.0"`,
		},
		// ── non-matches — must be left unchanged ──────────────────────────────
		{
			name:     "raw repo without subdirectory not matched",
			module:   "test",
			version:  "v2.0.0",
			input:    `source = "git@github.com:example/repo.git"`,
			expected: `source = "git@github.com:example/repo.git"`,
		},
		{
			name:     "raw repo with version tag not matched",
			module:   "test",
			version:  "v2.0.0",
			input:    `source = "git@github.com:example/repo.git?ref=v1.0.0"`,
			expected: `source = "git@github.com:example/repo.git?ref=v1.0.0"`,
		},
		{
			name:     "deeper path not matched when searching for prefix",
			module:   "test",
			version:  "v2.0.0",
			input:    `source = "git@github.com:example/repo.git//test/example"`,
			expected: `source = "git@github.com:example/repo.git//test/example"`,
		},
		{
			name:     "deeper path with version not matched when searching for prefix",
			module:   "test",
			version:  "v2.0.0",
			input:    `source = "git@github.com:example/repo.git//test/example?ref=v1"`,
			expected: `source = "git@github.com:example/repo.git//test/example?ref=v1"`,
		},
		{
			name:     "shallow path not matched when searching for nested",
			module:   "test/example",
			version:  "v2.0.0",
			input:    `source = "git@github.com:example/repo.git//test"`,
			expected: `source = "git@github.com:example/repo.git//test"`,
		},
		// ── non-GitHub SSH hosts ──────────────────────────────────────────────
		{
			name:     "GitLab SSH matched",
			module:   "test",
			version:  "v2.0.0",
			input:    `source = "git@gitlab.com:example/repo.git//test"`,
			expected: `source = "git@gitlab.com:example/repo.git//test?ref=v2.0.0"`,
		},
		{
			name:     "Bitbucket SSH matched",
			module:   "test",
			version:  "v2.0.0",
			input:    `source = "git@bitbucket.org:example/repo.git//test?ref=v1.0.0"`,
			expected: `source = "git@bitbucket.org:example/repo.git//test?ref=v2.0.0"`,
		},
		{
			name:     "self-hosted SSH matched",
			module:   "test",
			version:  "v2.0.0",
			input:    `source = "git@git.internal.example.com:example/repo.git//test"`,
			expected: `source = "git@git.internal.example.com:example/repo.git//test?ref=v2.0.0"`,
		},
		{
			name:     "explicit ssh:// protocol matched",
			module:   "test",
			version:  "v2.0.0",
			input:    `source = "ssh://git@git.example.com/example/repo.git//test"`,
			expected: `source = "ssh://git@git.example.com/example/repo.git//test?ref=v2.0.0"`,
		},
		// ── non-GitHub HTTPS hosts ────────────────────────────────────────────
		{
			name:     "GitLab HTTPS matched",
			module:   "test",
			version:  "v2.0.0",
			input:    `source = "https://gitlab.com/example/repo.git//test"`,
			expected: `source = "https://gitlab.com/example/repo.git//test?ref=v2.0.0"`,
		},
		{
			name:     "Bitbucket HTTPS matched",
			module:   "test",
			version:  "v2.0.0",
			input:    `source = "https://bitbucket.org/example/repo.git//test?ref=v1.0.0"`,
			expected: `source = "https://bitbucket.org/example/repo.git//test?ref=v2.0.0"`,
		},
		{
			name:     "self-hosted HTTPS matched",
			module:   "test",
			version:  "v2.0.0",
			input:    `source = "https://git.example.com/example/repo.git//test"`,
			expected: `source = "https://git.example.com/example/repo.git//test?ref=v2.0.0"`,
		},
		{
			name:     "HTTPS with embedded credentials matched",
			module:   "test",
			version:  "v2.0.0",
			input:    `source = "https://user:token@git.example.com/example/repo.git//test"`,
			expected: `source = "https://user:token@git.example.com/example/repo.git//test?ref=v2.0.0"`,
		},
		{
			name:     "HTTPS with non-standard port matched",
			module:   "test",
			version:  "v2.0.0",
			input:    `source = "https://git.example.com:8443/example/repo.git//test"`,
			expected: `source = "https://git.example.com:8443/example/repo.git//test?ref=v2.0.0"`,
		},
		// ── HTTPS URL support (GitHub) ────────────────────────────────────────
		{
			name:     "GitHub HTTPS matched",
			module:   "test",
			version:  "v2.0.0",
			input:    `source = "https://github.com/example/repo.git//test"`,
			expected: `source = "https://github.com/example/repo.git//test?ref=v2.0.0"`,
		},
		// ── multi-line file with mixed matches ────────────────────────────────
		{
			name:    "only matching module updated in multi-module file",
			module:  "test",
			version: "v2.0.0",
			input: `module "a" {
  source = "git@github.com:example/repo.git//test"
}

module "b" {
  source = "git@github.com:example/repo.git//test/example"
}

module "c" {
  source = "git@github.com:example/repo.git//test?ref=v1.0.0"
}`,
			expected: `module "a" {
  source = "git@github.com:example/repo.git//test?ref=v2.0.0"
}

module "b" {
  source = "git@github.com:example/repo.git//test/example"
}

module "c" {
  source = "git@github.com:example/repo.git//test?ref=v2.0.0"
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

			if err := pin.UpdateModule(dir, tt.module, tt.version); err != nil {
				t.Fatalf("UpdateModule: %v", err)
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
