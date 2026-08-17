package source

import (
	"bytes"
	"testing"
)

func TestSwapToLocal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		locals   map[int]string
		expected string
	}{
		// ── core shapes ─────────────────────────────────────────────────────
		{
			name:     "sub-module SSH source swapped, ref preserved verbatim in comment",
			input:    "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n",
			locals:   map[int]string{1: "../../some-repo/modules/vpc"},
			expected: "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\nsource = \"../../some-repo/modules/vpc\"\n",
		},
		{
			name:     "https source",
			input:    "source = \"https://github.com/org/some-repo.git//modules/vpc?ref=v1.2.0\"\n",
			locals:   map[int]string{1: "../../some-repo/modules/vpc"},
			expected: "# source = \"https://github.com/org/some-repo.git//modules/vpc?ref=v1.2.0\"\nsource = \"../../some-repo/modules/vpc\"\n",
		},
		{
			name:     "ssh:// source",
			input:    "source = \"ssh://git@github.com/org/some-repo.git//modules/vpc\"\n",
			locals:   map[int]string{1: "../../some-repo/modules/vpc"},
			expected: "# source = \"ssh://git@github.com/org/some-repo.git//modules/vpc\"\nsource = \"../../some-repo/modules/vpc\"\n",
		},
		{
			name:     "git:: prefixed source",
			input:    "source = \"git::https://github.com/org/some-repo.git//modules/vpc?ref=v1.2.0\"\n",
			locals:   map[int]string{1: "../../some-repo/modules/vpc"},
			expected: "# source = \"git::https://github.com/org/some-repo.git//modules/vpc?ref=v1.2.0\"\nsource = \"../../some-repo/modules/vpc\"\n",
		},
		{
			name:     "repo name containing dots",
			input:    "source = \"git@github.com:org/foo.bar.git//modules/vpc\"\n",
			locals:   map[int]string{1: "../../foo.bar/modules/vpc"},
			expected: "# source = \"git@github.com:org/foo.bar.git//modules/vpc\"\nsource = \"../../foo.bar/modules/vpc\"\n",
		},
		{
			name:     "root-module source with no //path",
			input:    "source = \"git@github.com:org/some-repo.git?ref=v1.2.0\"\n",
			locals:   map[int]string{1: "../../some-repo"},
			expected: "# source = \"git@github.com:org/some-repo.git?ref=v1.2.0\"\nsource = \"../../some-repo\"\n",
		},
		{
			name:     "source with no ?ref= at all",
			input:    "source = \"git@github.com:org/some-repo.git//modules/vpc\"\n",
			locals:   map[int]string{1: "../../some-repo/modules/vpc"},
			expected: "# source = \"git@github.com:org/some-repo.git//modules/vpc\"\nsource = \"../../some-repo/modules/vpc\"\n",
		},
		// ── formatting preserved ────────────────────────────────────────────
		{
			name:     "leading indentation preserved on both emitted lines",
			input:    "  source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n",
			locals:   map[int]string{1: "../../some-repo/modules/vpc"},
			expected: "  # source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n  source = \"../../some-repo/modules/vpc\"\n",
		},
		{
			name:     "trailing comment after closing quote preserved on local line",
			input:    "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\" # pin\n",
			locals:   map[int]string{1: "../../some-repo/modules/vpc"},
			expected: "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\" # pin\nsource = \"../../some-repo/modules/vpc\" # pin\n",
		},
		// ── non-matches / no-ops — must be left unchanged ──────────────────
		{
			name:     "already-commented line is not matched (idempotency)",
			input:    "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n",
			locals:   map[int]string{1: "../../some-repo/modules/vpc"},
			expected: "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n",
		},
		{
			name:     "registry source untouched",
			input:    "source = \"terraform-aws-modules/vpc/aws\"\n",
			locals:   map[int]string{1: "../../whatever"},
			expected: "source = \"terraform-aws-modules/vpc/aws\"\n",
		},
		{
			name:     "already-local source untouched",
			input:    "source = \"../foo\"\n",
			locals:   map[int]string{1: "../../bar"},
			expected: "source = \"../foo\"\n",
		},
		{
			name:     "matched line with no entry in locals left unchanged",
			input:    "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n",
			locals:   map[int]string{},
			expected: "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n",
		},
		{
			name: "file with git source AND registry source — only git one changes",
			input: "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n" +
				"source = \"terraform-aws-modules/vpc/aws\"\n",
			locals: map[int]string{1: "../../some-repo/modules/vpc"},
			expected: "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n" +
				"source = \"../../some-repo/modules/vpc\"\n" +
				"source = \"terraform-aws-modules/vpc/aws\"\n",
		},
		// ── byte-exactness ──────────────────────────────────────────────────
		{
			name:     "CRLF content stays pure CRLF",
			input:    "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\r\n",
			locals:   map[int]string{1: "../../some-repo/modules/vpc"},
			expected: "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\r\nsource = \"../../some-repo/modules/vpc\"\r\n",
		},
		{
			name:     "content with no final newline still has none afterwards",
			input:    "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"",
			locals:   map[int]string{1: "../../some-repo/modules/vpc"},
			expected: "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\nsource = \"../../some-repo/modules/vpc\"",
		},
		{
			name: "two sources in one file, both swapped",
			input: "source = \"git@github.com:org/repo-a.git//modules/a?ref=v1.0.0\"\n" +
				"source = \"git@github.com:org/repo-b.git//modules/b?ref=v2.0.0\"\n",
			locals: map[int]string{1: "../../repo-a/modules/a", 2: "../../repo-b/modules/b"},
			expected: "# source = \"git@github.com:org/repo-a.git//modules/a?ref=v1.0.0\"\n" +
				"source = \"../../repo-a/modules/a\"\n" +
				"# source = \"git@github.com:org/repo-b.git//modules/b?ref=v2.0.0\"\n" +
				"source = \"../../repo-b/modules/b\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := swapToLocal([]byte(tt.input), tt.locals)
			if string(got) != tt.expected {
				t.Errorf("\ngot:\n%q\nwant:\n%q", got, tt.expected)
			}

			// Byte-exactness sanity check: if the input is CRLF, the output
			// must never degrade to a bare \n or double up into \r\r.
			if bytes.Contains([]byte(tt.input), []byte("\r\n")) {
				if bytes.Contains(got, []byte("\r\r")) {
					t.Errorf("output contains \\r\\r: %q", got)
				}
				for i := range got {
					if got[i] == '\n' && (i == 0 || got[i-1] != '\r') {
						t.Errorf("bare \\n found in CRLF output at byte %d: %q", i, got)
					}
				}
			}
		})
	}
}

func TestRestoreToRemote(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		wantContent      string
		wantCount        int
		wantProblemLines []int
	}{
		{
			name:        "verified block restored, count 1, no problems",
			input:       "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\nsource = \"../../some-repo/modules/vpc\"\n",
			wantContent: "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n",
			wantCount:   1,
		},
		{
			// THE CRITICAL CASE: a hand-inserted line sits between the marker
			// and the local-path line. The Python tf-source-remote deleted the
			// line unconditionally; this must instead leave the whole block
			// byte-identical and report exactly one Problem.
			name: "hand-inserted line between marker and local path is never deleted",
			input: "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n" +
				"resource \"aws_instance\" \"x\" {}\n" +
				"source = \"../../some-repo/modules/vpc\"\n",
			wantContent: "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n" +
				"resource \"aws_instance\" \"x\" {}\n" +
				"source = \"../../some-repo/modules/vpc\"\n",
			wantCount:        0,
			wantProblemLines: []int{2},
		},
		{
			name: "one verified block and one unverified block in same file",
			input: "# source = \"git@github.com:org/repo-a.git//modules/a?ref=v1.0.0\"\n" +
				"source = \"../../repo-a/modules/a\"\n" +
				"# source = \"git@github.com:org/repo-b.git//modules/b?ref=v2.0.0\"\n" +
				"# manual note\n",
			wantContent: "source = \"git@github.com:org/repo-a.git//modules/a?ref=v1.0.0\"\n" +
				"# source = \"git@github.com:org/repo-b.git//modules/b?ref=v2.0.0\"\n" +
				"# manual note\n",
			wantCount:        1,
			wantProblemLines: []int{4},
		},
		{
			name:             "marker is the final line of the file",
			input:            "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n",
			wantContent:      "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n",
			wantCount:        0,
			wantProblemLines: []int{1},
		},
		{
			name: "line after marker is another git source line",
			input: "# source = \"git@github.com:org/repo-a.git//modules/a?ref=v1.0.0\"\n" +
				"source = \"git@github.com:org/repo-b.git//modules/b?ref=v2.0.0\"\n",
			wantContent: "# source = \"git@github.com:org/repo-a.git//modules/a?ref=v1.0.0\"\n" +
				"source = \"git@github.com:org/repo-b.git//modules/b?ref=v2.0.0\"\n",
			wantCount:        0,
			wantProblemLines: []int{2},
		},
		{
			name:        "no markers at all — content byte-identical",
			input:       "resource \"aws_instance\" \"x\" {}\n",
			wantContent: "resource \"aws_instance\" \"x\" {}\n",
			wantCount:   0,
		},
		{
			name:        "CRLF round trip yields CRLF",
			input:       "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\r\nsource = \"../../some-repo/modules/vpc\"\r\n",
			wantContent: "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\r\n",
			wantCount:   1,
		},
		{
			name: "#source (no space) and #   source (extra spaces) both matched",
			input: "#source = \"git@github.com:org/repo-a.git//modules/a?ref=v1.0.0\"\n" +
				"source = \"../../repo-a/modules/a\"\n" +
				"#   source = \"git@github.com:org/repo-b.git//modules/b?ref=v2.0.0\"\n" +
				"source = \"../../repo-b/modules/b\"\n",
			wantContent: "source = \"git@github.com:org/repo-a.git//modules/a?ref=v1.0.0\"\n" +
				"source = \"git@github.com:org/repo-b.git//modules/b?ref=v2.0.0\"\n",
			wantCount: 2,
		},
		{
			name:        "trailing text after closing quote in marker preserved on restore",
			input:       "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\" # pin\nsource = \"../../some-repo/modules/vpc\" # pin\n",
			wantContent: "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\" # pin\n",
			wantCount:   1,
		},
		{
			name:        "restoring already-restored content is idempotent",
			input:       "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n",
			wantContent: "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n",
			wantCount:   0,
		},
		{
			// The regression this task exists to fix: the marker line always
			// has a real terminator (swapToLocal gives it defaultTerm so the
			// two-line block is well formed), but the local-path line being
			// consumed here has none — that "no trailing newline" state must
			// win, not the marker's terminator.
			name:        "restoring a no-final-newline swapped block yields no trailing terminator",
			input:       "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\nsource = \"../../some-repo/modules/vpc\"",
			wantContent: "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"",
			wantCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotContent, gotCount, gotProblems := restoreToRemote([]byte(tt.input))

			if string(gotContent) != tt.wantContent {
				t.Errorf("\ncontent got:\n%q\nwant:\n%q", gotContent, tt.wantContent)
			}
			if gotCount != tt.wantCount {
				t.Errorf("count = %d, want %d", gotCount, tt.wantCount)
			}
			if len(gotProblems) != len(tt.wantProblemLines) {
				t.Fatalf("got %d problems (%+v), want %d", len(gotProblems), gotProblems, len(tt.wantProblemLines))
			}
			for i, wantLine := range tt.wantProblemLines {
				if gotProblems[i].Line != wantLine {
					t.Errorf("problem[%d].Line = %d, want %d", i, gotProblems[i].Line, wantLine)
				}
				if gotProblems[i].Reason == "" {
					t.Errorf("problem[%d].Reason is empty", i)
				}
			}
		})
	}
}

func TestScan(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		wantSwaps        []Swap
		wantProblemLines []int
	}{
		{
			name: "verified block yields exact Swap",
			input: "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n" +
				"source = \"../../some-repo/modules/vpc\"\n",
			wantSwaps: []Swap{
				{Line: 1, Source: `git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0`, Local: "../../some-repo/modules/vpc"},
			},
		},
		{
			name: "unverified marker yields Problem",
			input: "# source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n" +
				"not a local path\n",
			wantProblemLines: []int{2},
		},
		{
			name:  "clean content yields neither",
			input: "resource \"aws_instance\" \"x\" {}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSwaps, gotProblems := scan([]byte(tt.input))

			if len(gotSwaps) != len(tt.wantSwaps) {
				t.Fatalf("got %d swaps (%+v), want %d", len(gotSwaps), gotSwaps, len(tt.wantSwaps))
			}
			for i, want := range tt.wantSwaps {
				if gotSwaps[i] != want {
					t.Errorf("swap[%d] = %+v, want %+v", i, gotSwaps[i], want)
				}
			}

			if len(gotProblems) != len(tt.wantProblemLines) {
				t.Fatalf("got %d problems (%+v), want %d", len(gotProblems), gotProblems, len(tt.wantProblemLines))
			}
			for i, wantLine := range tt.wantProblemLines {
				if gotProblems[i].Line != wantLine {
					t.Errorf("problem[%d].Line = %d, want %d", i, gotProblems[i].Line, wantLine)
				}
			}
		})
	}
}

func TestParse(t *testing.T) {
	input := "source = \"git@github.com:org/repo-a.git//modules/a?ref=v1.0.0\"\n" +
		"source = \"git@github.com:org/repo-b.git?ref=v2.0.0\"\n" +
		"# source = \"git@github.com:org/repo-c.git//modules/c\"\n" +
		"source = \"terraform-aws-modules/vpc/aws\"\n"

	want := []found{
		{line: 1, repo: "repo-a", path: "modules/a"},
		{line: 2, repo: "repo-b", path: ""},
	}

	got := parse([]byte(input))
	if len(got) != len(want) {
		t.Fatalf("got %d found (%+v), want %d", len(got), got, len(want))
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("found[%d] = %+v, want %+v", i, got[i], w)
		}
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "LF-only",
			input:    "a\nb\nc\n",
			expected: []string{"a\n", "b\n", "c\n"},
		},
		{
			name:     "CRLF-only",
			input:    "a\r\nb\r\n",
			expected: []string{"a\r\n", "b\r\n"},
		},
		{
			name:     "mixed",
			input:    "a\r\nb\nc",
			expected: []string{"a\r\n", "b\n", "c"},
		},
		{
			name:     "no final newline",
			input:    "abc",
			expected: []string{"abc"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitLines([]byte(tt.input))

			// The byte-exactness property: joining the lines reproduces the
			// input exactly, terminators and all.
			if joined := bytes.Join(got, nil); string(joined) != tt.input {
				t.Errorf("bytes.Join(splitLines(c), nil) = %q, want %q", joined, tt.input)
			}

			if len(got) != len(tt.expected) {
				t.Fatalf("got %d lines (%q), want %d", len(got), got, len(tt.expected))
			}
			for i, want := range tt.expected {
				if string(got[i]) != want {
					t.Errorf("line[%d] = %q, want %q", i, got[i], want)
				}
			}
		})
	}
}

func TestDefaultTerm(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "LF-only", input: "a\nb\n", expected: "\n"},
		{name: "CRLF-only", input: "a\r\nb\r\n", expected: "\r\n"},
		{name: "mixed — first terminator wins", input: "a\r\nb\n", expected: "\r\n"},
		{name: "no final newline", input: "abc", expected: "\n"},
		{name: "empty input", input: "", expected: "\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultTerm([]byte(tt.input))
			if string(got) != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestRoundTrip proves the headline invariant directly: a local → remote
// round trip must reproduce the original file byte for byte, for both LF and
// CRLF line endings.
func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		original string
		locals   map[int]string
	}{
		{
			name: "LF fixture",
			original: "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n" +
				"resource \"aws_instance\" \"x\" {}\n",
			locals: map[int]string{1: "../../some-repo/modules/vpc"},
		},
		{
			name: "CRLF fixture",
			original: "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\r\n" +
				"resource \"aws_instance\" \"x\" {}\r\n",
			locals: map[int]string{1: "../../some-repo/modules/vpc"},
		},
		{
			// The swapped line is not the final line, so the file's trailing
			// newline (or lack of it) sits entirely on the line after it.
			name: "no final newline, LF file — last line is not the source",
			original: "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\n" +
				"resource \"aws_instance\" \"x\" {}",
			locals: map[int]string{1: "../../some-repo/modules/vpc"},
		},
		{
			// The exact case the bug report was filed against: a single-line
			// file with no trailing newline at all, where the swapped block's
			// local-path line becomes the new final line of the file.
			name:     "no final newline, single-line file",
			original: "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"",
			locals:   map[int]string{1: "../../some-repo/modules/vpc"},
		},
		{
			// Contrast case for the one above: a CRLF file that DOES end with
			// a proper \r\n — the ordinary CRLF fixture already covers this,
			// but restated here explicitly alongside its no-newline sibling
			// below so the two are checked side by side.
			name: "CRLF file with a proper trailing \\r\\n",
			original: "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\r\n" +
				"resource \"aws_instance\" \"x\" {}\r\n",
			locals: map[int]string{1: "../../some-repo/modules/vpc"},
		},
		{
			// CRLF file whose last line has no terminator at all: the source
			// line itself is not final here, so this exercises the "earlier
			// lines are CRLF, final line has nothing" combination distinct
			// from the CRLF-marker case below.
			name: "CRLF file, last line has no terminator at all",
			original: "source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"\r\n" +
				"resource \"aws_instance\" \"x\" {}",
			locals: map[int]string{1: "../../some-repo/modules/vpc"},
		},
		{
			// Two source lines; only the last physical line of the file (the
			// second source) lacks a terminator. Confirms the fix applies
			// independently per collapsed block, not just to a single-block file.
			name: "multi-source file, only the last line lacks a terminator",
			original: "source = \"git@github.com:org/repo-a.git//modules/a?ref=v1.0.0\"\n" +
				"source = \"git@github.com:org/repo-b.git//modules/b?ref=v2.0.0\"",
			locals: map[int]string{1: "../../repo-a/modules/a", 2: "../../repo-b/modules/b"},
		},
		{
			// The adjacent case flagged for a double-check: the swapped
			// block's local-path line is the new final line of the file with
			// no terminator, while the marker line above it was assigned
			// defaultTerm CRLF (because an earlier line in the file is CRLF).
			// The restored line must end with nothing — not a stray \r left
			// behind from the marker's terminator.
			name: "CRLF marker, but swapped local-path line is final with no newline",
			original: "resource \"aws_instance\" \"x\" {}\r\n" +
				"source = \"git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0\"",
			locals: map[int]string{2: "../../some-repo/modules/vpc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			swapped := swapToLocal([]byte(tt.original), tt.locals)
			restored, count, problems := restoreToRemote(swapped)

			if string(restored) != tt.original {
				t.Errorf("\nround trip got:\n%q\nwant original:\n%q", restored, tt.original)
			}
			if count != len(tt.locals) {
				t.Errorf("count = %d, want %d", count, len(tt.locals))
			}
			if len(problems) != 0 {
				t.Errorf("unexpected problems: %+v", problems)
			}
		})
	}
}
