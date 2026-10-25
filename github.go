package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
)

type Check struct {
	Name   string
	URL    string
	Status string
}

type PR struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Draft   bool   `json:"isDraft"`
	HeadRef string `json:"headRefName"`
	BaseRef string `json:"baseRefName"`
}

type checkRunsResponse struct {
	CheckRuns []checkRun `json:"check_runs"`
}

type checkRun struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	StartedAt  string `json:"started_at"`
	DetailsURL string `json:"details_url"`
	App        struct {
		Slug string `json:"slug"`
	} `json:"app"`
}

func fetchChecks(client *api.RESTClient, info RepoInfo) ([]Check, error) {
	path := fmt.Sprintf("repos/%s/%s/commits/%s/check-runs?per_page=100", info.Owner, info.Repo, url.PathEscape(info.Branch))
	var resp checkRunsResponse
	if err := client.Get(path, &resp); err != nil {
		var httpErr *api.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusUnprocessableEntity {
			return nil, nil // branch not found upstream — treat as no checks
		}
		return nil, fmt.Errorf("failed to fetch check runs: %w", err)
	}

	if len(resp.CheckRuns) == 0 {
		return nil, nil
	}

	// Deduplicate by name — keep the most recent run per check
	grouped := make(map[string][]checkRun)
	for _, cr := range resp.CheckRuns {
		grouped[cr.Name] = append(grouped[cr.Name], cr)
	}

	checks := make([]Check, 0, len(grouped))
	for name, runs := range grouped {
		// Sort by started_at descending, pick latest
		sort.Slice(runs, func(i, j int) bool {
			return runs[i].StartedAt > runs[j].StartedAt
		})
		latest := runs[0]
		status := latest.Status
		if latest.Conclusion != "" {
			status = latest.Conclusion
		}
		checks = append(checks, Check{
			Name:   name,
			URL:    latest.DetailsURL,
			Status: status,
		})
	}

	sort.Slice(checks, func(i, j int) bool {
		return strings.ToLower(checks[i].Name) < strings.ToLower(checks[j].Name)
	})

	return checks, nil
}

// prQuery fetches open PRs from and into a branch in one round trip. Like
// `gh pr list --head`, headRefName matches the branch name on any fork.
const prQuery = `query($owner: String!, $repo: String!, $branch: String!) {
  repository(owner: $owner, name: $repo) {
    from: pullRequests(headRefName: $branch, states: OPEN, first: 100, orderBy: {field: CREATED_AT, direction: DESC}) { nodes { ...pr } }
    into: pullRequests(baseRefName: $branch, states: OPEN, first: 100, orderBy: {field: CREATED_AT, direction: DESC}) { nodes { ...pr } }
  }
}
fragment pr on PullRequest { number title url isDraft headRefName baseRefName }`

// fetchPRs returns open PRs whose head is branch, then those targeting it.
func fetchPRs(client *api.GraphQLClient, info RepoInfo) (from, into []PR, err error) {
	var resp struct {
		Repository struct {
			From struct{ Nodes []PR } `json:"from"`
			Into struct{ Nodes []PR } `json:"into"`
		} `json:"repository"`
	}
	vars := map[string]any{"owner": info.Owner, "repo": info.Repo, "branch": info.Branch}
	if err := client.Do(prQuery, vars, &resp); err != nil {
		return nil, nil, fmt.Errorf("failed to list PRs: %w", err)
	}
	return resp.Repository.From.Nodes, resp.Repository.Into.Nodes, nil
}
