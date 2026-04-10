package workflow

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
)

// Apply runs `<tfBin> init` then `<tfBin> apply` in each directory,
// streaming output live. Stops on first error.
func Apply(dirs []string, tfBin string) error {
	if len(dirs) == 0 {
		color.Yellow("No matching directories found.")
		return nil
	}

	dim := color.New(color.FgHiBlack)
	dirColor := color.New(color.FgCyan, color.Bold)
	for i, dir := range dirs {
		dim.Printf("\n[%d/%d] ", i+1, len(dirs))
		fmt.Print("Applying in ")
		dirColor.Println(dir)

		if err := runStreamed(dir, tfBin, "init", "-input=false"); err != nil {
			return fmt.Errorf("tf init failed in %s: %w", dir, err)
		}
		if err := runStreamed(dir, tfBin, "apply", "-input=false"); err != nil {
			return fmt.Errorf("tf apply failed in %s: %w", dir, err)
		}
	}

	noun := "directories"
	if len(dirs) == 1 {
		noun = "directory"
	}
	color.New(color.FgGreen, color.Bold).Printf("\nDone — applied %d %s\n", len(dirs), noun)
	return nil
}

func runStreamed(dir, bin string, args ...string) error {
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
