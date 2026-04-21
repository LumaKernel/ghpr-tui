package ghclient

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Client wraps gh CLI interactions.
type Client struct {
	repo string // owner/repo or empty for current repo
}

// NewClient creates a new gh CLI client.
func NewClient(repo string) *Client {
	return &Client{repo: repo}
}

func (c *Client) ghArgs(args ...string) []string {
	if c.repo != "" {
		args = append(args, "--repo", c.repo)
	}
	return args
}

func (c *Client) run(args ...string) (string, error) {
	cmd := exec.Command("gh", c.ghArgs(args...)...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("gh %s: %s", strings.Join(args, " "), string(exitErr.Stderr))
		}
		return "", fmt.Errorf("gh %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

// ListPRs returns open pull requests.
func (c *Client) ListPRs(limit int) ([]PR, error) {
	fields := "number,title,author,state,isDraft,additions,deletions,updatedAt,url,headRefName,baseRefName,labels"
	out, err := c.run("pr", "list", "--state", "open", "--limit", fmt.Sprintf("%d", limit), "--json", fields)
	if err != nil {
		return nil, err
	}
	var raw []prJSON
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return nil, fmt.Errorf("parsing PR list: %w", err)
	}
	prs := make([]PR, len(raw))
	for i, p := range raw {
		prs[i] = prFromJSON(p)
	}
	return prs, nil
}

// GetDiff returns the raw diff for a PR.
// Falls back to local git diff if gh pr diff fails (e.g. diff too large).
func (c *Client) GetDiff(pr PR) (string, error) {
	raw, err := c.run("pr", "diff", fmt.Sprintf("%d", pr.Number))
	if err == nil {
		return raw, nil
	}

	// Fallback: fetch remote refs and use local git diff
	return c.getLocalDiff(pr)
}

func (c *Client) getLocalDiff(pr PR) (string, error) {
	remote := "origin"

	// Fetch both branches
	fetchCmd := exec.Command("git", "fetch", remote,
		fmt.Sprintf("+refs/heads/%s:refs/remotes/%s/%s", pr.HeadRef, remote, pr.HeadRef),
		fmt.Sprintf("+refs/heads/%s:refs/remotes/%s/%s", pr.BaseRef, remote, pr.BaseRef),
	)
	if out, err := fetchCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git fetch: %s", string(out))
	}

	// Three-dot diff: changes on head since it diverged from base
	diffCmd := exec.Command("git", "diff",
		fmt.Sprintf("%s/%s...%s/%s", remote, pr.BaseRef, remote, pr.HeadRef))
	out, err := diffCmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git diff: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("git diff: %w", err)
	}
	return string(out), nil
}

// GetParsedDiff returns a parsed diff for a PR.
func (c *Client) GetParsedDiff(pr PR) (ParsedDiff, error) {
	raw, err := c.GetDiff(pr)
	if err != nil {
		return ParsedDiff{}, err
	}
	return ParseDiff(raw), nil
}

// MergeSettings represents which merge methods a repo allows.
type MergeSettings struct {
	AllowSquash bool
	AllowMerge  bool
	AllowRebase bool
}

// AllowedMethods returns the list of allowed merge method names.
func (ms MergeSettings) AllowedMethods() []string {
	var methods []string
	if ms.AllowSquash {
		methods = append(methods, "squash")
	}
	if ms.AllowMerge {
		methods = append(methods, "merge")
	}
	if ms.AllowRebase {
		methods = append(methods, "rebase")
	}
	return methods
}

// GetMergeSettings returns the repo's allowed merge methods.
func (c *Client) GetMergeSettings() (MergeSettings, error) {
	repo := c.repo
	if repo == "" {
		var err error
		repo, err = ResolveRepo()
		if err != nil {
			return MergeSettings{}, err
		}
	}
	out, err := exec.Command("gh", "api", fmt.Sprintf("repos/%s", repo),
		"--jq", "{squash: .allow_squash_merge, merge: .allow_merge_commit, rebase: .allow_rebase_merge}").Output()
	if err != nil {
		// Fallback: allow all
		return MergeSettings{AllowSquash: true, AllowMerge: true, AllowRebase: true}, nil
	}
	var raw struct {
		Squash bool `json:"squash"`
		Merge  bool `json:"merge"`
		Rebase bool `json:"rebase"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return MergeSettings{AllowSquash: true, AllowMerge: true, AllowRebase: true}, nil
	}
	return MergeSettings{
		AllowSquash: raw.Squash,
		AllowMerge:  raw.Merge,
		AllowRebase: raw.Rebase,
	}, nil
}

// MarkReady marks a draft PR as ready for review.
func (c *Client) MarkReady(number int) error {
	_, err := c.run("pr", "ready", fmt.Sprintf("%d", number))
	return err
}

// MergePR merges a PR. method is one of "merge", "squash", "rebase".
// If undraft is true, marks the PR as ready for review first.
func (c *Client) MergePR(number int, method string, undraft bool) error {
	if undraft {
		if err := c.MarkReady(number); err != nil {
			return fmt.Errorf("undraft: %w", err)
		}
	}
	args := []string{"pr", "merge", fmt.Sprintf("%d", number), "--" + method, "--delete-branch"}
	_, err := c.run(args...)
	return err
}

// ToggleDraft toggles the draft status of a PR.
func (c *Client) ToggleDraft(number int, isDraft bool) error {
	if isDraft {
		// Currently draft → mark ready
		return c.MarkReady(number)
	}
	// Currently ready → convert to draft
	_, err := c.run("pr", "ready", fmt.Sprintf("%d", number), "--undo")
	return err
}

// OpenInBrowser opens a PR in the default browser.
func (c *Client) OpenInBrowser(number int) error {
	_, err := c.run("pr", "view", fmt.Sprintf("%d", number), "--web")
	return err
}

// ResolveRepo returns the current repo in owner/repo format.
func ResolveRepo() (string, error) {
	cmd := exec.Command("gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("could not determine repository: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
