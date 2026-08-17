// Package source implements the pure text transformations behind
// `wand source local`, `wand source remote`, and `wand source status`:
// swapping pinned git module sources in .tf file content for local
// relative paths, and reversing that swap. Everything here operates on
// []byte content — no filesystem access, no os/exec, no git — so it is
// unit-testable without fixtures.
package source

import (
	"bytes"
	"fmt"
	"regexp"
)

// gitURLPrefix matches the scheme portion of a git module source, up to the
// host and org path: an optional git:: prefix, then the SSH, ssh:// or https://
// form. Both regexes below build on it, so a new scheme is added in one place.
// It contains only non-capturing groups, so it does not shift group numbering.
const gitURLPrefix = `(?:git::)?(?:git@[^"/:]+:|ssh://git@[^"/]+/|https://[^"/]+/)`

// gitSourceLine matches an active (un-commented) pinned git module source.
// Anchoring on ^(\s*)source means an already-commented line never matches,
// which is what makes `local` idempotent for free.
//
//	1: indentation, reused on both emitted lines
//	2: full repo URL through ".git"
//	3: repo name — last segment before ".git"; [^"/]+ permits dots
//	4: module subpath after "//", absent for a root-module source
//	5: trailing text after the closing quote, preserved verbatim
var gitSourceLine = regexp.MustCompile(
	`^(\s*)source\s*=\s*"(` + gitURLPrefix + `[^"]*?([^"/]+)\.git)(?://([^"?]*))?(?:\?[^"]*)?"(.*)$`)

// commentedSourceLine matches the marker line `local` writes.
//
//	1: indentation
//	2: original line body, from "source" through trailing text — re-emitted as-is
//	3: the quoted source value alone, for status reporting
var commentedSourceLine = regexp.MustCompile(
	`^(\s*)#\s*(source\s*=\s*"(` + gitURLPrefix + `[^"]*\.git[^"]*)".*)$`)

// localSourceLine matches the relative-path line `local` inserts. Requiring the
// path to start with "./" or "../" IS the verification that stops `remote` from
// deleting an unrelated line.
//
//	1: indentation   2: the relative path
var localSourceLine = regexp.MustCompile(
	`^(\s*)source\s*=\s*"(\.{1,2}/[^"]*)"\s*(?:#.*)?$`)

// Swap describes one active local source swap.
type Swap struct {
	File   string // .tf file containing it — set by the caller, not by scan
	Line   int    // 1-indexed line of the "# source = ..." marker
	Source string // original git source, verbatim from the comment
	Local  string // relative path currently in effect
}

// Problem describes a marker line whose swap block could not be verified.
type Problem struct {
	File   string // set by the caller, not by scan/restoreToRemote
	Line   int
	Reason string
}

// String renders a problem as one report line, so all three commands report it
// identically whether they print it or fold it into a returned error.
func (p Problem) String() string {
	return fmt.Sprintf("❌  %s:%d — %s", p.File, p.Line, p.Reason)
}

// found is one active git source located during the validation pass.
type found struct {
	line int    // 1-indexed
	repo string // repo name, e.g. "some-repo"
	path string // module subpath, "" for a root-module source
}

// splitLines splits content into lines that each keep their terminator
// ("\n" or "\r\n"). Concatenating the result reproduces content byte for byte,
// including the absence of a final newline.
func splitLines(content []byte) [][]byte {
	var lines [][]byte
	start := 0
	for start < len(content) {
		idx := bytes.IndexByte(content[start:], '\n')
		if idx == -1 {
			lines = append(lines, content[start:])
			break
		}
		end := start + idx + 1
		lines = append(lines, content[start:end])
		start = end
	}
	return lines
}

// splitTerm splits a line into body and terminator; term is "" on a final
// line with no trailing newline.
func splitTerm(line []byte) (body, term []byte) {
	n := len(line)
	if n >= 2 && line[n-2] == '\r' && line[n-1] == '\n' {
		return line[:n-2], line[n-2:]
	}
	if n >= 1 && line[n-1] == '\n' {
		return line[:n-1], line[n-1:]
	}
	return line, nil
}

// defaultTerm returns the terminator to use for inserted lines: "\r\n" if the
// file's first terminator is CRLF, otherwise "\n".
func defaultTerm(content []byte) []byte {
	for _, line := range splitLines(content) {
		_, term := splitTerm(line)
		switch len(term) {
		case 2:
			return []byte("\r\n")
		case 1:
			return []byte("\n")
		}
	}
	return []byte("\n")
}

// parse returns every active git source in content, for validation before any write.
func parse(content []byte) []found {
	var out []found
	for i, line := range splitLines(content) {
		body, _ := splitTerm(line)
		m := gitSourceLine.FindSubmatch(body)
		if m == nil {
			continue
		}
		out = append(out, found{
			line: i + 1,
			repo: string(m[3]),
			path: string(m[4]),
		})
	}
	return out
}

// swapToLocal rewrites content, replacing each active git source with the original
// commented out followed by a local relative-path line. locals is keyed by the
// 1-indexed line number of the source, pre-resolved and pre-validated by the caller.
// A matched line with no entry in locals is left unchanged.
func swapToLocal(content []byte, locals map[int]string) []byte {
	var buf bytes.Buffer
	for i, line := range splitLines(content) {
		body, term := splitTerm(line)
		m := gitSourceLine.FindSubmatch(body)
		if m == nil {
			buf.Write(line)
			continue
		}
		rel, ok := locals[i+1]
		if !ok {
			buf.Write(line)
			continue
		}

		indent := m[1]
		trailing := m[5]
		rest := body[len(indent):] // body without its own leading indent, since indent is written separately

		markerTerm, localTerm := term, term
		if len(term) == 0 {
			// Final line with no trailing newline: the marker now takes the
			// file's default terminator, and the inserted line — now the new
			// final line — takes none, so the file still ends without one.
			markerTerm = defaultTerm(content)
			localTerm = nil
		}

		buf.Write(indent)
		buf.WriteString("# ")
		buf.Write(rest)
		buf.Write(markerTerm)

		buf.Write(indent)
		buf.WriteString(`source = "`)
		buf.WriteString(rel)
		buf.WriteString(`"`)
		buf.Write(trailing)
		buf.Write(localTerm)
	}
	return buf.Bytes()
}

// restoreToRemote uncomments each verified swap block and drops the local-path line
// after it. Blocks failing verification are left byte-identical and returned as
// problems.
func restoreToRemote(content []byte) ([]byte, int, []Problem) {
	lines := splitLines(content)
	var buf bytes.Buffer
	var problems []Problem
	count := 0

	for i := 0; i < len(lines); i++ {
		body, _ := splitTerm(lines[i])
		m := commentedSourceLine.FindSubmatch(body)
		if m == nil {
			buf.Write(lines[i])
			continue
		}

		if i+1 >= len(lines) {
			buf.Write(lines[i])
			problems = append(problems, Problem{Line: i + 1, Reason: "marker has no following line"})
			continue
		}

		nextBody, nextTerm := splitTerm(lines[i+1])
		if !localSourceLine.Match(nextBody) {
			buf.Write(lines[i])
			problems = append(problems, Problem{Line: i + 2, Reason: "line after marker is not a local source path"})
			continue
		}

		// The collapsed line takes its terminator from the local-path line
		// being consumed (lines[i+1]), not the marker (lines[i]). swapToLocal
		// always gives the marker a real terminator so the two-line block is
		// well formed, even when the file has no final newline — in that case
		// it's the local-path line that carries the "no trailing newline"
		// state, and that state must survive the round trip.
		indent := m[1]
		origBody := m[2]
		buf.Write(indent)
		buf.Write(origBody)
		buf.Write(nextTerm)
		count++
		i++ // skip the local-path line, already consumed
	}

	return buf.Bytes(), count, problems
}

// scan returns the verified swaps in content plus any unverifiable markers.
func scan(content []byte) ([]Swap, []Problem) {
	lines := splitLines(content)
	var swaps []Swap
	var problems []Problem

	for i := 0; i < len(lines); i++ {
		body, _ := splitTerm(lines[i])
		m := commentedSourceLine.FindSubmatch(body)
		if m == nil {
			continue
		}

		if i+1 >= len(lines) {
			problems = append(problems, Problem{Line: i + 1, Reason: "marker has no following line"})
			continue
		}

		nextBody, _ := splitTerm(lines[i+1])
		lm := localSourceLine.FindSubmatch(nextBody)
		if lm == nil {
			problems = append(problems, Problem{Line: i + 2, Reason: "line after marker is not a local source path"})
			continue
		}

		swaps = append(swaps, Swap{
			Line:   i + 1,
			Source: string(m[3]),
			Local:  string(lm[2]),
		})
		i++ // skip the local-path line, already consumed
	}

	return swaps, problems
}
