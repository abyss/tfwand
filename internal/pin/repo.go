package pin

import (
	"fmt"
	"regexp"

	"github.com/fatih/color"
)

// UpdateRepo walks root for .tf files and updates all source references to
// repoName.git regardless of subdirectory path.
func UpdateRepo(root, repoName, version string) error {
	// Pattern captures:
	//   group 1: everything up to and including the final /
	//   group 2: repoName.git
	//   group 3: optional subdirectory path (//path/...)
	//   group 4: optional existing ?ref=...
	//   group 5: closing quote
	pattern := regexp.MustCompile(
		`(source\s*=\s*"[^"]*/)(` + regexp.QuoteMeta(repoName) + `\.git)(//[^"?]*|)(\?ref=[^"]*|)(")`,
	)

	replace := func(match string) string {
		groups := pattern.FindStringSubmatch(match)
		if groups == nil {
			return match
		}
		prefix := groups[1]
		repo := groups[2]
		subdir := groups[3]
		// groups[4] is the old ?ref= — discarded
		quote := groups[5]
		return fmt.Sprintf("%s%s%s?ref=%s%s", prefix, repo, subdir, version, quote)
	}

	n, err := walkAndUpdate(root, func(content string) string {
		return pattern.ReplaceAllStringFunc(content, replace)
	})
	if err != nil {
		return err
	}

	noun := "files"
	if n == 1 {
		noun = "file"
	}
	color.New(color.FgGreen, color.Bold).Printf("Pinned %d %s\n", n, noun)
	return nil
}
