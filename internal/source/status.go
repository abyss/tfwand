package source

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

// Find returns every active swap in files, plus any unverifiable markers.
func Find(files []string) ([]Swap, []Problem, error) {
	var swaps []Swap
	var problems []Problem

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, nil, err
		}

		fileSwaps, fileProblems := scan(content)
		for i := range fileSwaps {
			fileSwaps[i].File = file
		}
		for i := range fileProblems {
			fileProblems[i].File = file
		}
		swaps = append(swaps, fileSwaps...)
		problems = append(problems, fileProblems...)
	}

	return swaps, problems, nil
}

// Status prints active swaps; with check it returns an error when any exist.
func Status(files []string, check bool) error {
	if len(files) == 0 {
		color.Yellow("No .tf files found.")
		return nil
	}

	swaps, problems, err := Find(files)
	if err != nil {
		return err
	}

	cyan := color.New(color.FgCyan)
	cyan.Println("\n══════════════════════════════════════════")
	color.New(color.FgCyan, color.Bold).Println("  Source Status")
	cyan.Println("══════════════════════════════════════════")

	yellow := color.New(color.FgYellow)
	redBold := color.New(color.FgRed, color.Bold)

	for _, s := range swaps {
		yellow.Printf("⚠️   %s:%d\n      %s\n      -> %s\n", s.File, s.Line, s.Source, s.Local)
	}
	for _, p := range problems {
		redBold.Println(p)
	}
	if len(swaps) == 0 && len(problems) == 0 {
		color.New(color.FgGreen).Println("✅  No active source swaps")
	}

	fmt.Printf("\nTotal: %d active swap(s), %d problem(s)\n", len(swaps), len(problems))

	if check && (len(swaps) > 0 || len(problems) > 0) {
		return fmt.Errorf("%d active swap(s), %d unverifiable marker(s)", len(swaps), len(problems))
	}
	return nil
}
