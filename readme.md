A small Go CLI that quickly opens CI check runs or pull requests for your current Git branch.

Clone and run `go install -ldflags="-s -w" .` to add it to GOPATH. (requires gh cli)

Usage: gh-util [flags] [branch]

Flags:

  -ci [branch]   Open CI check runs (current branch by default; pass a branch name to inspect another without checking it out)

  -wait          Poll until CI check runs or pull requests appear for the branch, then open them (use with -ci or -pr when they haven't started yet; combines with a branch name, e.g. `gh-util -ci -wait some-branch` or `gh-util -pr -wait some-branch`)

  -pr [branch]   Open pull requests (current branch by default; pass a branch name e.g. `main` to see the review queue targeting it)

  -plain         Print results as tab-separated text instead of opening the picker (for scripts and AI agents). Check runs are shown collapsed (sub-stages omitted). Progress and "not found" messages go to stderr; exits 2 when nothing is found, e.g. `gh-util -ci -plain` or `gh-util -pr -plain main`
