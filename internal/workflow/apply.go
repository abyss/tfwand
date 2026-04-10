package workflow

import (
	"fmt"
	"os"
	"os/exec"
)

// Apply runs `<tfBin> init` then `<tfBin> apply` in each directory,
// streaming output live. Stops on first error.
func Apply(dirs []string, tfBin string) error {
	if len(dirs) == 0 {
		fmt.Println("No matching directories found.")
		return nil
	}

	for _, dir := range dirs {
		fmt.Printf("\n\033[95m==> Applying in %s\033[0m\n", dir)

		if err := runStreamed(dir, tfBin, "init", "-input=false"); err != nil {
			return fmt.Errorf("tf init failed in %s: %w", dir, err)
		}
		if err := runStreamed(dir, tfBin, "apply", "-input=false"); err != nil {
			return fmt.Errorf("tf apply failed in %s: %w", dir, err)
		}
	}
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
