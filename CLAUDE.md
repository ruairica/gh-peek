# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -ldflags="-s -w" -o gh-util.exe .

# Run
go run . -ci                # open CI check runs for current branch
go run . -ci -wait          # poll until CI check runs appear, then open
go run . -pr                # open PRs for current branch
go run . -ci -plain         # tab-separated output, no TUI (for agents); exit 2 if none

# Install to GOPATH/bin (for testing from any repo)
go install -ldflags="-s -w" .

# Auto-modernize code after Go upgrades
go fix ./...
go fix -diff ./...          # preview changes
```

## Project Conventions

- Go 1.26+ — use modern language features
- External tools via `os/exec`: `git`, `gh` (GitHub CLI) — requires both installed and `gh` authenticated
- TUI via `charmbracelet/bubbletea` (custom list picker in `picker.go`)
- Status styling via `charmbracelet/lipgloss` (colors and symbols)
- Strip binaries with `-ldflags="-s -w"` for size reduction
