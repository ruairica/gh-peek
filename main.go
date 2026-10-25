package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/term"
)

// checkNameWidth is the column width check names are padded to so status
// badges align; child rows subtract their "├ " prefix from it.
const checkNameWidth = 55

// splitSubStage splits a check name of the form "X (Stage Job)" into the
// parent name "X" and the stage text, when a check named exactly "X" exists
// in names. The longest matching parent wins, so "X (a) (b)" nests under
// "X (a)" when that check also exists.
func splitSubStage(name string, names map[string]bool) (parent, stage string, ok bool) {
	base, hadParen := strings.CutSuffix(name, ")")
	if !hadParen {
		return "", "", false
	}
	for idx := strings.LastIndex(base, " ("); idx > 0; idx = strings.LastIndex(base[:idx], " (") {
		if names[base[:idx]] && idx+2 < len(base) {
			return base[:idx], base[idx+2:], true
		}
	}
	return "", "", false
}

// statusBadge renders the status colored on top of base, so the picker can
// carry a selection background through the badge.
func statusBadge(status string, base lipgloss.Style) string {
	switch status {
	case "success":
		return base.Foreground(lipgloss.Color("2")).Render("✓ " + status)
	case "failure":
		return base.Foreground(lipgloss.Color("1")).Render("✗ " + status)
	case "in_progress":
		return base.Foreground(lipgloss.Color("3")).Render("⟳ " + status)
	default:
		return base.Render(status)
	}
}

const usageText = `Open CI check runs or pull requests for a branch.

Usage:
  gh peek ci [branch] [flags]   Open CI check runs (current branch, or [branch] if given)
  gh peek pr [branch] [flags]   Open pull requests from or into the branch

Flags:
  --wait    Poll until check runs or pull requests are available
  --plain   Print tab-separated text instead of opening the picker (for scripts and
            AI agents); also used automatically when stdin or stdout is not a terminal

Exit codes:
  0  Results found
  1  Error
  2  No check runs or pull requests found (plain output only)
`

func usage() {
	fmt.Fprint(os.Stderr, usageText)
}

// errNoResults signals that nothing was found; in -plain mode it exits with
// code 2 so callers can tell "nothing yet" apart from a real error.
var errNoResults = errors.New("no results")

// notFound reports an empty result. In plain mode the message goes to stderr
// so stdout only ever holds results.
func notFound(plain bool, msg string) error {
	if plain {
		fmt.Fprintln(os.Stderr, msg)
		return errNoResults
	}
	fmt.Println(msg)
	return nil
}

// waiting reports a -wait poll. The picker path redraws one line in place;
// plain mode logs it once to stderr so agents don't get a stream of \r updates.
func waiting(plain bool, polls int, msg string) {
	if !plain {
		fmt.Print("\r" + msg)
	} else if polls == 0 {
		fmt.Fprintln(os.Stderr, msg)
	}
}

// plainField makes free text safe for a tab-separated column.
func plainField(s string) string {
	return strings.NewReplacer("\t", " ", "\n", " ", "\r", " ").Replace(s)
}

func runChecks(info RepoInfo, wait, plain bool) error {
	client, err := api.NewRESTClient(api.ClientOptions{Host: info.Host})
	if err != nil {
		return err
	}

	var checks []Check
	for polls := 0; ; polls++ {
		checks, err = fetchChecks(client, info)
		if err != nil {
			return err
		}
		if len(checks) > 0 {
			break
		}
		if !wait {
			return notFound(plain, fmt.Sprintf("No check runs found on branch '%s'.\nThis branch may not have triggered any checks yet. Try pushing first.", info.Branch))
		}
		waiting(plain, polls, fmt.Sprintf("Waiting for checks on '%s'...", info.Branch))
		time.Sleep(time.Second)
	}

	// Split checks into top-level runs and their sub-stages. A check named
	// "X (Stage Job)" is a sub-stage of a check named exactly "X" (Azure
	// Pipelines / GitHub Actions convention); anything else stays top-level.
	names := make(map[string]bool, len(checks))
	for _, c := range checks {
		names[c.Name] = true
	}

	var parents []Check
	children := make(map[string][]Check) // parent name → sub-stages, Name rewritten to the stage text
	for _, c := range checks {
		if parent, stage, ok := splitSubStage(c.Name, names); ok {
			c.Name = stage
			children[parent] = append(children[parent], c)
			continue
		}
		parents = append(parents, c)
	}

	if plain {
		fmt.Printf("# gh peek ci: check runs for %s/%s (%s)\n", info.Owner, info.Repo, info.Branch)
		fmt.Println("status\tname\turl")
		for _, p := range parents {
			fmt.Printf("%s\t%s\t%s\n", p.Status, plainField(p.Name), p.URL)
		}
		return nil
	}

	var items []pickerItem
	for _, p := range parents {
		item := pickerItem{name: fmt.Sprintf("%-*s", checkNameWidth, p.Name), status: p.Status, url: p.URL}
		if n := len(children[p.Name]); n > 0 {
			item.extra = fmt.Sprintf("+%d", n)
		}
		items = append(items, item)
		for _, c := range children[p.Name] {
			items = append(items, pickerItem{
				name:   fmt.Sprintf("├ %-*s", checkNameWidth-2, c.Name),
				status: c.Status,
				url:    c.URL,
				child:  true,
			})
		}
	}

	return runPicker(fmt.Sprintf("Checks — %s/%s (%s)", info.Owner, info.Repo, info.Branch), items)
}

func runPR(info RepoInfo, wait, plain bool) error {
	client, err := api.NewGraphQLClient(api.ClientOptions{Host: info.Host})
	if err != nil {
		return err
	}

	var prs []PR
	for polls := 0; ; polls++ {
		prsFrom, prsInto, err := fetchPRs(client, info)
		if err != nil {
			return err
		}

		// Deduplicate: a PR from this branch could also target this branch (unlikely but possible)
		seen := make(map[int]bool)
		prs = nil
		for _, pr := range prsFrom {
			seen[pr.Number] = true
			prs = append(prs, pr)
		}
		for _, pr := range prsInto {
			if !seen[pr.Number] {
				prs = append(prs, pr)
			}
		}

		if len(prs) > 0 {
			break
		}
		if !wait {
			return notFound(plain, fmt.Sprintf("No open pull requests found for branch '%s'.", info.Branch))
		}
		waiting(plain, polls, fmt.Sprintf("Waiting for pull requests on '%s'...", info.Branch))
		time.Sleep(time.Second)
	}

	if plain {
		fmt.Printf("# gh peek pr: open pull requests from or into %s in %s/%s\n", info.Branch, info.Owner, info.Repo)
		fmt.Println("number\tdraft\thead\tbase\turl\ttitle")
		for _, pr := range prs {
			fmt.Printf("%d\t%t\t%s\t%s\t%s\t%s\n", pr.Number, pr.Draft, pr.HeadRef, pr.BaseRef, pr.URL, plainField(pr.Title))
		}
		return nil
	}

	items := make([]pickerItem, len(prs))
	for i, pr := range prs {
		draft := ""
		if pr.Draft {
			draft = " [draft]"
		}
		label := fmt.Sprintf("#%-6d %s (%s → %s)%s", pr.Number, pr.Title, pr.HeadRef, pr.BaseRef, draft)
		items[i] = pickerItem{name: label, url: pr.URL}
	}

	return runPicker(fmt.Sprintf("Pull Requests — %s", info.Branch), items)
}

// parseArgs parses a subcommand's flags and optional branch. Flags may come
// before or after the branch, as with gh's own commands.
func parseArgs(cmd string, args []string) (branch string, wait, plain bool, err error) {
	fs := flag.NewFlagSet("gh peek "+cmd, flag.ContinueOnError)
	fs.SetOutput(io.Discard) // errors and usage are reported by main
	fs.BoolVar(&wait, "wait", false, "")
	fs.BoolVar(&plain, "plain", false, "")

	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return "", false, false, err
		}
		if fs.NArg() == 0 {
			break
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}

	switch len(positional) {
	case 0:
	case 1:
		branch = positional[0]
	default:
		return "", false, false, fmt.Errorf("too many arguments; usage: gh peek %s [branch] [flags]", cmd)
	}
	return branch, wait, plain, nil
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd, args := os.Args[1], os.Args[2:]
	run := map[string]func(RepoInfo, bool, bool) error{"ci": runChecks, "pr": runPR}[cmd]
	if run == nil {
		switch name := strings.TrimLeft(cmd, "-"); {
		case name == "h" || name == "help":
			usage()
			os.Exit(0)
		case name == "ci" || name == "pr":
			fmt.Fprintf(os.Stderr, "Error: %s is now a subcommand; use: gh peek %s\n", cmd, name)
		default:
			fmt.Fprintf(os.Stderr, "Error: unknown command %q\n\n", cmd)
			usage()
		}
		os.Exit(1)
	}

	branch, wait, plain, err := parseArgs(cmd, args)
	if errors.Is(err, flag.ErrHelp) {
		usage()
		os.Exit(0)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		usage()
		os.Exit(1)
	}

	// The picker needs a real terminal to draw on and read keys from, so fall
	// back to plain output when either end is redirected (pipes, most agents).
	plain = plain || !term.FromEnv().IsTerminalOutput() || !term.IsTerminal(os.Stdin)

	info, err := getRepoInfo(branch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	err = run(info, wait, plain)
	if errors.Is(err, errNoResults) {
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
