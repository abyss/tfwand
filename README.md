# tfwand

`wand` is an OpenTofu/Terraform utility toolkit. It provides commands for pinning module versions across your codebase, swapping module sources to local checkouts for development, and running plan/apply operations across multiple directories.

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

### Swap sources for local development

Swap pinned git sources for local relative paths, to test an uncommitted change in a dependency repo without pushing a tag (`src` is an alias for `source`):

```bash
wand source local              # rewrite every pinned git source in scope
wand source remote             # restore the original git ref
wand source status             # list active swaps
wand source status --check     # exit nonzero if any swap is active
```

`local` comments out each matched line and adds one pointing at a sibling checkout of that repo, which has to exist alongside your repo's root:

```hcl
# source = "git@github.com:org/some-repo.git//modules/vpc?ref=v1.2.0"
source = "../some-repo/modules/vpc"
```

Scope is recursive from the current directory, skipping `.terraform`. Use `--dir <path>` on any subcommand to narrow it to one directory, without recursion.

Sources with no `//path` resolve to the repo directory itself, and the `ssh://`, `https://` and `git::` forms all work. `local` validates every source before writing anything. `remote` only removes a local-path line it can positively identify, so it reports a hand-edited block instead of deleting it. Both directions are idempotent, which makes `remote` safe to run before every commit and `--check` usable as a pre-commit hook. `--check` also fails on a block that can no longer be verified.

`remote` restores the ref verbatim and knows nothing of newer tags, so bump afterwards:

```bash
wand source remote
wand pin repo some-repo v1.3.0
```

### Plan

Summarise `tf plan` output across multiple directories:

```bash
wand plan all          # all directories containing .tf files
wand plan git          # directories with git changes
wand plan staged       # directories with staged git changes
wand plan dir ./prod   # a single directory
```

Use `--exclude` to skip directories matching a prefix (repeatable):

```bash
wand plan all --exclude prod
wand plan all --exclude prod --exclude staging/legacy
```

### Apply

Run `tf init` + `tf apply` across multiple directories:

```bash
wand apply all         # all directories containing .tf files
wand apply git         # directories with git changes
wand apply staged      # directories with staged git changes
wand apply dir ./prod  # a single directory
```

Use `--exclude` to skip directories matching a prefix (repeatable):

```bash
wand apply all --exclude prod
wand apply git --exclude prod --exclude staging/legacy
```

### Upgrade

Run `tf init -upgrade` across multiple directories. A failure in one directory does not stop the rest; failed directories are listed at the end and the command exits nonzero:

```bash
wand upgrade all         # all directories containing .tf files
wand upgrade git         # directories with git changes
wand upgrade staged      # directories with staged git changes
wand upgrade dir ./prod  # a single directory
```

`--exclude` works the same as for `apply`.

### Options

The `tf` binary defaults to `tf`. Override with a flag or environment variable:

```bash
wand --tf tofu plan all
WAND_TF_BIN=terraform wand plan all
```
