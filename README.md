# tfwand

`wand` is an OpenTofu/Terraform utility toolkit. It provides commands for pinning module versions across your codebase and running plan/apply operations across multiple directories.

## Installation

```bash
brew install abyss/tools/tfwand
```

Or build from source:

```bash
go install github.com/abyss/tfwand@latest
```

## Usage

### Pin module versions

Update a specific module path to a new version:

```bash
wand pin module network v2.1.0
wand pin module aws/vpc v1.3.0
```

Update all references to a repository regardless of subdirectory:

```bash
wand pin repo my-modules v3.0.0
```

### Plan

Summarise `tf plan` output across multiple directories:

```bash
wand plan all          # all directories containing .tf files
wand plan git          # directories with git changes
wand plan staged       # directories with staged git changes
wand plan dir ./prod   # a single directory
```

### Apply

Run `tf init` + `tf apply` across multiple directories:

```bash
wand apply all         # all directories containing .tf files
wand apply git         # directories with git changes
wand apply staged      # directories with staged git changes
wand apply dir ./prod  # a single directory
```

### Options

The `tf` binary defaults to `tf`. Override with a flag or environment variable:

```bash
wand --tf tofu plan all
WAND_TF_BIN=terraform wand plan all
```
