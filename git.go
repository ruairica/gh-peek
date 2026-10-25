package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
)

type RepoInfo struct {
	Host   string
	Owner  string
	Repo   string
	Branch string
}

// getRepoInfo resolves the repository the same way gh does (GH_REPO, then
// git remotes on known hosts). branch overrides the checked-out branch.
func getRepoInfo(branch string) (RepoInfo, error) {
	if os.Getenv("GH_REPO") == "" && exec.Command("git", "rev-parse", "--git-dir").Run() != nil {
		return RepoInfo{}, errors.New("not in a git repository (set GH_REPO=OWNER/REPO to run elsewhere)")
	}
	repo, err := repository.Current()
	if err != nil {
		return RepoInfo{}, err
	}

	if branch == "" {
		branch = "main"
		if out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
			branch = strings.TrimSpace(string(out))
		}
	}

	return RepoInfo{Host: repo.Host, Owner: repo.Owner, Repo: repo.Name, Branch: branch}, nil
}
