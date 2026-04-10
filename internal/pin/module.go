package pin

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/color"
)

// UpdateModule walks root for .tf files and updates source references that
// use exactly modulePath (e.g. "network" matches //network but not //network/sub).
func UpdateModule(root, modulePath, version string) error {
	// Pattern captures:
	//   group 1: everything up to and including //
	//   group 2: the module path
	//   group 3: optional suffix (?ref=... or /subpath...)
	//   group 4: closing quote
	pattern := regexp.MustCompile(
		`(source\s*=\s*"[^"]+//)` +
			`(` + regexp.QuoteMeta(modulePath) + `)` +
			`(\?ref=[^"]*|/[^"]*|)` +
			`(")`,
	)

	replace := func(match string) string {
		groups := pattern.FindStringSubmatch(match)
		if groups == nil {
			return match
		}
		prefix := groups[1]
		module := groups[2]
		suffix := groups[3]
		quote := groups[4]

		// If suffix starts with '/', the full captured path is module+suffix.
		// We only update if the full path equals exactly modulePath (no extra segments).
		if strings.HasPrefix(suffix, "/") {
			fullPath := module + suffix
			// strip any ?ref= to get the pure path
			if idx := strings.Index(fullPath, "?ref="); idx >= 0 {
				fullPath = fullPath[:idx]
			}
			if fullPath != modulePath {
				return match // different path — leave unchanged
			}
			// path matches; rebuild with new version, stripping old ?ref=
			pathOnly := module + strings.SplitN(suffix, "?ref=", 2)[0]
			return fmt.Sprintf("%s%s?ref=%s%s", prefix, pathOnly, version, quote)
		}

		// suffix is either empty or starts with ?ref=
		return fmt.Sprintf("%s%s?ref=%s%s", prefix, module, version, quote)
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

func walkAndUpdate(root string, transform func(string) string) (int, error) {
	var count int
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".terraform" {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".tf") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		original := string(data)
		updated := transform(original)
		if updated == original {
			return nil
		}
		color.New(color.Bold).Print("Updated ")
		color.New(color.FgCyan).Println(path)
		count++
		return os.WriteFile(path, []byte(updated), 0o644)
	})
	return count, err
}
