package workflow

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type planResult struct {
	dir     string
	status  string // "no-change" | "changed" | "output-only" | "failed"
	summary string // the Plan: X to add... line, if present
	err     error
}

// Plan runs `<tfBin> init` then `<tfBin> plan` in each directory, captures
// output, and prints a one-line summary per directory followed by a totals block.
func Plan(dirs []string, tfBin string) error {
	if len(dirs) == 0 {
		fmt.Println("No matching directories found.")
		return nil
	}

	results := make([]planResult, 0, len(dirs))

	for _, dir := range dirs {
		fmt.Printf("Planning %s...\n", dir)
		r := planDir(dir, tfBin)
		results = append(results, r)
	}

	fmt.Println()
	fmt.Println("══════════════════════════════════════════")
	fmt.Println("  Plan Summary")
	fmt.Println("══════════════════════════════════════════")

	var changed, noChange, outputOnly, failed int
	for _, r := range results {
		switch r.status {
		case "changed":
			changed++
			if r.summary != "" {
				fmt.Printf("🚨  %s\n    %s\n", r.dir, r.summary)
			} else {
				fmt.Printf("🚨  %s — changes detected\n", r.dir)
			}
		case "output-only":
			outputOnly++
			fmt.Printf("⚠️   %s — output-only changes\n", r.dir)
		case "failed":
			failed++
			fmt.Printf("❌  %s — plan failed: %v\n", r.dir, r.err)
		default:
			noChange++
			fmt.Printf("✅  %s\n", r.dir)
		}
	}

	fmt.Println()
	fmt.Printf("Total: %d dirs — 🚨 %d changed, ⚠️  %d output-only, ✅ %d no changes, ❌ %d failed\n",
		len(results), changed, outputOnly, noChange, failed)

	if failed > 0 {
		return fmt.Errorf("%d plan(s) failed", failed)
	}
	return nil
}

func planDir(dir, tfBin string) planResult {
	r := planResult{dir: dir}

	initCmd := exec.Command(tfBin, "init", "-input=false", "-no-color")
	initCmd.Dir = dir
	if out, err := initCmd.CombinedOutput(); err != nil {
		r.status = "failed"
		r.err = fmt.Errorf("init: %w\n%s", err, string(out))
		return r
	}

	planCmd := exec.Command(tfBin, "plan", "-input=false", "-no-color")
	planCmd.Dir = dir
	out, err := planCmd.CombinedOutput()
	// exit code 2 means "changes present" — not an error
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 2 {
			r.status = "failed"
			r.err = fmt.Errorf("plan: %w", err)
			return r
		}
	}

	r.summary, r.status = parsePlanOutput(out)
	return r
}

func parsePlanOutput(out []byte) (summary, status string) {
	var hasSummaryLine, hasOutputChange bool

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()

		// "Plan: X to add, Y to change, Z to destroy."
		if strings.HasPrefix(line, "Plan: ") {
			summary = line
			hasSummaryLine = true
		}
		// "Changes to outputs:" — output-only change
		if strings.Contains(strings.ToLower(line), "changes to outputs:") {
			hasOutputChange = true
		}
	}

	switch {
	case hasSummaryLine && summary != "Plan: 0 to add, 0 to change, 0 to destroy.":
		status = "changed"
	case hasOutputChange:
		status = "output-only"
		summary = ""
	default:
		status = "no-change"
		summary = ""
	}
	return summary, status
}
