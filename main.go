package main

import (
	"os"
	"strings"

	"github.com/abyss/tfwand/internal/gitfiles"
	"github.com/abyss/tfwand/internal/pin"
	"github.com/abyss/tfwand/internal/source"
	"github.com/abyss/tfwand/internal/tffiles"
	"github.com/abyss/tfwand/internal/workflow"
	"github.com/spf13/cobra"
)

var tfBin string

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:          "wand",
		Short:        "tfwand — OpenTofu/Terraform utility toolkit",
		SilenceUsage: true,
	}

	defaultTF := os.Getenv("WAND_TF_BIN")
	if defaultTF == "" {
		defaultTF = "tf"
	}
	root.PersistentFlags().StringVar(&tfBin, "tf", defaultTF, "OpenTofu/Terraform binary to use (env: WAND_TF_BIN)")

	root.AddCommand(pinCmd())
	root.AddCommand(applyCmd())
	root.AddCommand(planCmd())
	root.AddCommand(upgradeCmd())
	root.AddCommand(sourceCmd())

	return root
}

// ── pin ──────────────────────────────────────────────────────────────────────

func pinCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pin",
		Short: "Pin module or repository versions in .tf files",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "module <path> <version>",
		Short: "Pin a specific module path to a version",
		Long: `Updates source = "...//path?ref=..." for an exact module path.

  wand pin module network v2.1.0
  wand pin module aws/vpc v1.3.0`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return pin.UpdateModule(".", args[0], args[1])
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "repo <name> <version>",
		Short: "Pin all references to a repository to a version",
		Long: `Updates all source = ".../<name>.git/...?ref=..." regardless of subdirectory.

  wand pin repo my-modules v3.0.0`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return pin.UpdateRepo(".", args[0], args[1])
		},
	})
	return cmd
}

// filterDirs removes directories that equal an exclude entry or are nested under one.
func filterDirs(dirs []string, excludes []string) []string {
	if len(excludes) == 0 {
		return dirs
	}
	out := make([]string, 0, len(dirs))
	for _, d := range dirs {
		excluded := false
		for _, ex := range excludes {
			if d == ex || strings.HasPrefix(d, ex+"/") {
				excluded = true
				break
			}
		}
		if !excluded {
			out = append(out, d)
		}
	}
	return out
}

// sourceFiles returns the .tf files in scope: everything under the current directory
// recursively, or only those directly inside dir when --dir is given.
func sourceFiles(dir string) ([]string, error) {
	if dir != "" {
		return tffiles.FilesInDir(dir)
	}
	return tffiles.Files(".")
}

// dirCmd builds a command whose git/all/staged/dir subcommands pass the selected
// directories to run. action is the help-text verb phrase, e.g. "Apply in".
func dirCmd(use, short, action string, run func(dirs []string, tfBin string) error) *cobra.Command {
	var excludes []string
	cmd := &cobra.Command{Use: use, Short: short}
	cmd.PersistentFlags().StringArrayVar(&excludes, "exclude", nil, "Exclude directories matching this prefix (repeatable)")

	scoped := func(find func(root string) ([]string, error)) func(*cobra.Command, []string) error {
		return func(*cobra.Command, []string) error {
			dirs, err := find(".")
			if err != nil {
				return err
			}
			return run(filterDirs(dirs, excludes), tfBin)
		}
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "git",
		Short: action + " directories with git changes",
		RunE:  scoped(gitfiles.ChangedDirs),
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "all",
		Short: action + " all directories containing .tf files",
		RunE:  scoped(tffiles.FindDirs),
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "staged",
		Short: action + " directories with staged git changes",
		RunE:  scoped(gitfiles.StagedDirs),
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "dir <path>",
		Short: action + " a specific directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run([]string{args[0]}, tfBin)
		},
	})
	return cmd
}

// ── apply ────────────────────────────────────────────────────────────────────

func applyCmd() *cobra.Command {
	return dirCmd("apply", "Run tf init + tf apply across Terraform directories", "Apply in", workflow.Apply)
}

// ── plan ─────────────────────────────────────────────────────────────────────

func planCmd() *cobra.Command {
	return dirCmd("plan", "Run tf plan and summarize results across Terraform directories", "Summarize plan for", workflow.Plan)
}

// ── upgrade ──────────────────────────────────────────────────────────────────

func upgradeCmd() *cobra.Command {
	return dirCmd("upgrade", "Run tf init -upgrade across Terraform directories", "Upgrade in", workflow.Upgrade)
}

// ── source ───────────────────────────────────────────────────────────────────

func sourceCmd() *cobra.Command {
	var dir string
	var check bool
	cmd := &cobra.Command{
		Use:     "source",
		Aliases: []string{"src"},
		Short:   "Swap module sources between git refs and local sibling checkouts",
	}
	cmd.PersistentFlags().StringVar(&dir, "dir", "", "Only process .tf files in this directory (no recursion)")

	cmd.AddCommand(&cobra.Command{
		Use:   "local",
		Short: "Point sources at local sibling checkouts",
		Long: `Rewrites every pinned git source under the current directory to a relative path
pointing at a sibling checkout of that repo. The dependency repo must be cloned
alongside this repo's git root, in a directory named after the repo. Validates
every source first; if any sibling checkout is missing, nothing is written.

  wand source local
  wand source local --dir stacks/prod`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			files, err := sourceFiles(dir)
			if err != nil {
				return err
			}
			return source.Local(files)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "remote",
		Short: "Restore sources to their pinned git refs",
		Long: `Restores each swapped source to the exact git ref preserved in the comment.
Only removes a local-path line it can positively identify, so a hand-edit
inside a swapped block is reported rather than deleted.

  wand source remote
  wand source remote --dir stacks/prod`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			files, err := sourceFiles(dir)
			if err != nil {
				return err
			}
			return source.Remote(files)
		},
	})

	st := &cobra.Command{
		Use:   "status",
		Short: "List active local source swaps",
		Long: `Lists active swaps. --check exits nonzero when any exist, which makes it
usable as a pre-commit guard against committing a dev-machine-only local path.

  wand source status
  wand source status --check`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			files, err := sourceFiles(dir)
			if err != nil {
				return err
			}
			return source.Status(files, check)
		},
	}
	st.Flags().BoolVar(&check, "check", false, "Exit nonzero if any local source swap is active")
	cmd.AddCommand(st)
	return cmd
}
