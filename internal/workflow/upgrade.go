package workflow

import (
	"fmt"
	"os/exec"

	"github.com/fatih/color"
)

// Upgrade runs `<tfBin> init -upgrade` in each directory and captures output.
// It continues past failures, prints the output of failed directories, and
// returns an error if any directory failed.
func Upgrade(dirs []string, tfBin string) error {
	if len(dirs) == 0 {
		color.Yellow("No matching directories found.")
		return nil
	}

	dim := color.New(color.FgHiBlack)
	red := color.New(color.FgRed)
	var failed []string
	for i, dir := range dirs {
		dim.Printf("[%d/%d] ", i+1, len(dirs))
		fmt.Printf("Upgrading %s...\n", dir)

		cmd := exec.Command(tfBin, "init", "-upgrade", "-input=false", "-no-color")
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			failed = append(failed, dir)
			red.Printf("❌  %s — init -upgrade failed: %v\n%s\n", dir, err, out)
		}
	}

	if len(failed) > 0 {
		color.New(color.FgRed, color.Bold).Printf("\nFailed in %d of %d directories:\n", len(failed), len(dirs))
		for _, dir := range failed {
			red.Printf("  %s\n", dir)
		}
		return fmt.Errorf("%d upgrade(s) failed", len(failed))
	}

	noun := "directories"
	if len(dirs) == 1 {
		noun = "directory"
	}
	color.New(color.FgGreen, color.Bold).Printf("\nDone — upgraded %d %s\n", len(dirs), noun)
	return nil
}
