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

// ── apply ────────────────────────────────────────────────────────────────────

func applyCmd() *cobra.Command {
	var excludes []string
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Run tf init + tf apply across Terraform directories",
	}
	cmd.PersistentFlags().StringArrayVar(&excludes, "exclude", nil, "Exclude directories matching this prefix (repeatable)")
	cmd.AddCommand(&cobra.Command{
		Use:   "git",
		Short: "Apply in all directories with git changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			dirs, err := gitfiles.ChangedDirs(".")
			if err != nil {
				return err
			}
			return workflow.Apply(filterDirs(dirs, excludes), tfBin)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "all",
		Short: "Apply in all directories containing .tf files",
		RunE: func(cmd *cobra.Command, args []string) error {
			dirs, err := tffiles.FindDirs(".")
			if err != nil {
				return err
			}
			return workflow.Apply(filterDirs(dirs, excludes), tfBin)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "staged",
		Short: "Apply in directories with staged git changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			dirs, err := gitfiles.StagedDirs(".")
			if err != nil {
				return err
			}
			return workflow.Apply(filterDirs(dirs, excludes), tfBin)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "dir <path>",
		Short: "Apply in a specific directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return workflow.Apply([]string{args[0]}, tfBin)
		},
	})
	return cmd
}

// ── plan ─────────────────────────────────────────────────────────────────────

func planCmd() *cobra.Command {
	var excludes []string
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Run tf plan and summarize results across Terraform directories",
	}
	cmd.PersistentFlags().StringArrayVar(&excludes, "exclude", nil, "Exclude directories matching this prefix (repeatable)")
	cmd.AddCommand(&cobra.Command{
		Use:   "git",
		Short: "Summarize plan for directories with git changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			dirs, err := gitfiles.ChangedDirs(".")
			if err != nil {
				return err
			}
			return workflow.Plan(filterDirs(dirs, excludes), tfBin)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "all",
		Short: "Summarize plan for all directories containing .tf files",
		RunE: func(cmd *cobra.Command, args []string) error {
			dirs, err := tffiles.FindDirs(".")
			if err != nil {
				return err
			}
			return workflow.Plan(filterDirs(dirs, excludes), tfBin)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "staged",
		Short: "Summarize plan for directories with staged git changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			dirs, err := gitfiles.StagedDirs(".")
			if err != nil {
				return err
			}
			return workflow.Plan(filterDirs(dirs, excludes), tfBin)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "dir <path>",
		Short: "Summarize plan for a specific directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return workflow.Plan([]string{args[0]}, tfBin)
		},
	})
	return cmd
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
