# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -ldflags="-s -w" -o gh-peek.exe .

# Run
go run . ci                 # open CI check runs for current branch
go run . ci --wait          # poll until CI check runs appear, then open
go run . pr [branch]        # open PRs from/into current branch (or [branch])
go run . ci --plain         # tab-separated output, no TUI (for agents); exit 2 if none

# Install as a local gh extension (run as `gh peek ...`); rebuild to update
gh extension install .

# Install to GOPATH/bin as a standalone `gh-peek`
go install -ldflags="-s -w" .

# Auto-modernize code after Go upgrades
go fix ./...
go fix -diff ./...          # preview changes
```

## Project Conventions

- Go 1.26+ — use modern language features
- Shipped as a `gh` extension (`gh peek`); releases are built by `.github/workflows/release.yml` on `v*` tag push
- GitHub access via `cli/go-gh/v2`: `repository.Current()` for repo detection, REST client for check runs, GraphQL client for PRs, `browser` for opening URLs, `term` for TTY detection. Only `git rev-parse` (current branch) still shells out
- Plain output is forced when stdin or stdout isn't a terminal, since the picker can't run there
- TUI via `charmbracelet/bubbletea` (custom list picker in `picker.go`)
- Status styling via `charmbracelet/lipgloss` (colors and symbols)
- Strip binaries with `-ldflags="-s -w"` for size reduction
