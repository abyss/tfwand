package source

import (
	"bytes"
	"fmt"
	"os"

	"github.com/fatih/color"
)

// Remote restores swapped sources to their original git URLs, verifying each block
// before removing its local-path line.
func Remote(files []string) error {
	if len(files) == 0 {
		color.Yellow("No .tf files found.")
		return nil
	}

	var problems []Problem
	filesRestored := 0

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		out, _, probs := restoreToRemote(content)
		for i := range probs {
			probs[i].File = file
		}
		problems = append(problems, probs...)

		if bytes.Equal(out, content) {
			continue
		}
		if err := os.WriteFile(file, out, 0o644); err != nil {
			return err
		}
		color.New(color.Bold).Print("Restored ")
		color.New(color.FgCyan).Println(file)
		filesRestored++
	}

	redBold := color.New(color.FgRed, color.Bold)
	for _, p := range problems {
		redBold.Println(p)
	}

	noun := "file"
	if filesRestored != 1 {
		noun = "files"
	}
	color.New(color.FgGreen, color.Bold).Printf("Restored %d %s\n", filesRestored, noun)

	if len(problems) > 0 {
		return fmt.Errorf("%d unverified swap block(s) left in place", len(problems))
	}
	return nil
}
