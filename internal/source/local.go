package source

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

// Local swaps every pinned git module source in files to a local relative path
// pointing at a sibling checkout. It validates every source before writing any file.
func Local(files []string) error {
	if len(files) == 0 {
		color.Yellow("No .tf files found.")
		return nil
	}

	type fileData struct {
		path    string
		content []byte
		locals  map[int]string
	}

	r := newResolver()
	datas := make([]fileData, 0, len(files))
	var problems []Problem

	// Pass 1: validate every source in every file. Nothing is written here.
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		found := parse(content)
		locals := make(map[int]string, len(found))
		for _, f := range found {
			rel, err := r.localPath(filepath.Dir(file), f.repo, f.path)
			if err != nil {
				problems = append(problems, Problem{File: file, Line: f.line, Reason: err.Error()})
				continue
			}
			locals[f.line] = rel
		}

		datas = append(datas, fileData{path: file, content: content, locals: locals})
	}

	// Nothing has been written yet, so a single error block is the whole report —
	// unlike Remote, there is no per-file progress output to interleave it with.
	if len(problems) > 0 {
		msgs := make([]string, 0, len(problems))
		for _, p := range problems {
			msgs = append(msgs, p.String())
		}
		return fmt.Errorf("%d source(s) could not be resolved, no files modified:\n%s",
			len(problems), strings.Join(msgs, "\n"))
	}

	// Pass 2: every source resolved cleanly, safe to write.
	count := 0
	for _, d := range datas {
		if len(d.locals) == 0 {
			continue
		}
		out := swapToLocal(d.content, d.locals)
		if bytes.Equal(out, d.content) {
			continue
		}
		if err := os.WriteFile(d.path, out, 0o644); err != nil {
			return err
		}
		color.New(color.Bold).Print("Swapped ")
		color.New(color.FgCyan).Println(d.path)
		count++
	}

	noun := "file"
	if count != 1 {
		noun = "files"
	}
	color.New(color.FgGreen, color.Bold).Printf("Swapped %d %s\n", count, noun)
	return nil
}
