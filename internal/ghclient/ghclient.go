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
func (c *Client) GetDiff(number int) (string, error) {
	return c.run("pr", "diff", fmt.Sprintf("%d", number))
}

// GetParsedDiff returns a parsed diff for a PR.
func (c *Client) GetParsedDiff(number int) (ParsedDiff, error) {
	raw, err := c.GetDiff(number)
	if err != nil {
		return ParsedDiff{}, err
	}
	return ParseDiff(raw), nil
}

// MergePR merges a PR. method is one of "merge", "squash", "rebase".
func (c *Client) MergePR(number int, method string) error {
	args := []string{"pr", "merge", fmt.Sprintf("%d", number), "--" + method, "--delete-branch"}
	_, err := c.run(args...)
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
