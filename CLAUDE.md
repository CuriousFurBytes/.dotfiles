# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Repo Is

Personal dotfiles managed by [chezmoi](https://www.chezmoi.io/), plus a Go TUI installer built with [Charm](https://charm.sh/) libraries (Huh, BubbleTea, Lipgloss). The chezmoi source directory lives at `~/.local/share/chezmoi`.

## Chezmoi Commands

```bash
chezmoi diff                  # Preview what applying would change
chezmoi apply -v              # Apply dotfiles to the home directory
chezmoi edit ~/.zshrc         # Edit a managed file (opens source)
chezmoi add ~/.config/app/rc  # Start tracking a new file
chezmoi update -v             # Pull latest + apply
```

## Go Installer Commands

The installer lives in `installer/` and is excluded from chezmoi tracking (see `.chezmoiignore`).

```bash
# Build
cd installer && go build -o /tmp/dotfiles-installer .

# Run (from repo root, auto-detects packages.json)
/tmp/dotfiles-installer

# Run with verbose output (streams subprocess output instead of spinner)
/tmp/dotfiles-installer --verbose

# Point to a custom source dir
/tmp/dotfiles-installer --source /path/to/chezmoi/source
```

## File Naming Conventions (chezmoi)

Chezmoi maps source filenames to target paths using prefixes:

| Source prefix | Meaning |
|---|---|
| `dot_` | Maps to `.` in home dir (e.g. `dot_zshrc` → `~/.zshrc`) |
| `private_` | Sets 0600 permissions |
| `private_dot_` | Both private and dot-prefixed |
| `.tmpl` suffix | Processed as a Go template with values from `.chezmoi.toml.tmpl` |

Template data variables (set during `chezmoi init`): `email`, `name`, `github_username`, `is_work`, `osid`, `nextdns_id`, `nextdns_profile_name`, `proton_pass_ssh_item`.

## Chezmoi Scripts (`.chezmoiscripts/`)

Scripts run automatically during `chezmoi apply`. The naming encodes behavior:

- `run_once_` — runs only the first time
- `run_onchange_` — runs whenever the script file content changes
- Number prefix controls execution order (e.g. `25`, `40`, `45`)
- `.tmpl` suffix allows OS-conditional logic via `{{ if eq .chezmoi.os "darwin" }}`

The `darwin/` subdirectory holds macOS-only scripts.

## Package System (`packages.json`)

All installable packages are declared in `packages.json`. Keys starting with `_` are metadata (`_brew_taps`). Each package entry:

```json
"my-tool": {
    "description": "...",
    "packages": {
        "darwin": { "brew": "my-tool" },
        "ubuntu": { "apt": "my-tool" },
        "fedora": { "dnf": "my-tool" },
        "pop_os": { "apt": "my-tool" }
    }
}
```

Supported install methods: `brew`, `cask`, `apt`, `dnf`, `cargo`, `go_tool`, `uv_tool`, `snap`, `flatpak`, `gh_extension`, `eget`, `npm_global`, `manual`.

For `manual`, supported types are `script`, `git_clone`, `dmg`, `zip`, `tar_gz`, `deb`, `rpm`, `appimage`. GitHub release assets are resolved via `gh api` using `repo` + `asset_pattern` (pipe-separated substrings, all must match).

To categorize a new package in the TUI, add it to `categoryMap` in `installer/packages.go`.

## Installer Architecture (`installer/`)

`app.go` implements an 11-step sequential state machine:
1. OS detection + welcome
2. Load `packages.json` + Huh multi-select form (one page per category)
3. Install chezmoi
4. Install Proton Pass + CLI
5. `pass-cli login` (interactive)
6. `chezmoi init`
7. `chezmoi apply -v` (interactive terminal)
8. `gh auth login` (interactive)
9. Install gh-dash
10. Install selected packages (system packages batched; secondary packages parallelized 4-wide)
11. Summary

Each step calls `ConfirmStep()` before running. System packages (brew/cask/apt/dnf) are installed first in a batch pass; everything else (`cargo`, `go_tool`, `uv_tool`, etc.) runs in parallel with a semaphore of 4.

`InstalledCache` in `installer.go` lazily loads and caches the full installed-package list per method (one `brew list` call, not one per package).

`Verbose` is a package-level bool toggled by `--verbose`; `spinOrRun()` / `pi.run()` respect it to switch between spinners and streamed output.
