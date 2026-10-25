A [GitHub CLI](https://cli.github.com/) extension that quickly opens CI check runs or pull requests for your current Git branch.

## Install

```sh
gh extension install ruairica/gh-peek   # once a release is published
```

From a clone (local development):

```sh
go build -ldflags="-s -w" -o gh-peek.exe .   # drop .exe on macOS/Linux
gh extension install .
```

`go install github.com/ruairica/gh-peek@latest` also works if you'd rather run it as `gh-peek` directly.

## Usage

```
gh peek ci [branch] [flags]   Open CI check runs (current branch, or [branch] if given)
gh peek pr [branch] [flags]   Open pull requests from or into the branch
```

Pass a branch name to inspect another branch without checking it out, e.g. `gh peek pr main` to see the review queue targeting `main`.

Flags (before or after the branch):

- `--wait` polls until check runs or pull requests appear, then opens them. Use it when CI hasn't started yet, e.g. `gh peek ci --wait`.
- `--plain` prints tab-separated text instead of opening the picker (for scripts and AI agents). Check runs are shown collapsed (sub-stages omitted). Progress and "not found" messages go to stderr; exits 2 when nothing is found, e.g. `gh peek ci --plain` or `gh peek pr main --plain`.

Plain output is also used automatically when stdin or stdout isn't a terminal (piped, redirected, or run by most agents), since the picker can't run there. Agents should still pass `--plain` explicitly: some run commands in a pseudo-terminal, which looks like a real one.

The repository is resolved the same way `gh` resolves it (git remotes, or `GH_REPO=OWNER/REPO`), links open with `gh`'s browser setting (`GH_BROWSER`, `gh config set browser`, `BROWSER`), and API calls use your `gh auth` login.
