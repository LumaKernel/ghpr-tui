package ghclient

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Client wraps gh CLI interactions.
type Client struct {
	repo           string                                                   // owner/repo or empty for current repo
	runFn          func(args ...string) (string, error)                     // injectable for testing; nil = use real gh CLI
	execFn         func(name string, args ...string) ([]byte, error)        // injectable for testing; nil = use real exec
	execStdinFn    func(stdin, name string, args ...string) ([]byte, error) // injectable for testing; nil = use real exec with stdin
	resolveRepoFn  func() (string, error)                                   // injectable for testing; nil = use real ResolveRepo
}

// NewClient creates a new gh CLI client.
func NewClient(repo string) *Client {
	return &Client{repo: repo}
}

// NewTestClient creates a client with a custom run function (for testing).
func NewTestClient(repo string, runFn func(args ...string) (string, error)) *Client {
	return &Client{repo: repo, runFn: runFn}
}

func (c *Client) ghArgs(args ...string) []string {
	if c.repo != "" {
		args = append(args, "--repo", c.repo)
	}
	return args
}

func (c *Client) doExec(name string, args ...string) ([]byte, error) {
	if c.execFn != nil {
		return c.execFn(name, args...)
	}
	return exec.Command(name, args...).Output()
}

func (c *Client) doExecWithStdin(stdin, name string, args ...string) ([]byte, error) {
	if c.execStdinFn != nil {
		return c.execStdinFn(stdin, name, args...)
	}
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(stdin)
	return cmd.CombinedOutput()
}

// ResolveRepo returns the current repo in owner/repo format.
// Uses the injectable resolveRepoFn if set, otherwise calls gh CLI.
func (c *Client) ResolveRepo() (string, error) {
	if c.resolveRepoFn != nil {
		return c.resolveRepoFn()
	}
	out, err := c.doExec("gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner")
	if err != nil {
		return "", fmt.Errorf("could not determine repository: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (c *Client) run(args ...string) (string, error) {
	if c.runFn != nil {
		return c.runFn(args...)
	}
	out, err := c.doExec("gh", c.ghArgs(args...)...)
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
	fields := "number,title,author,state,isDraft,additions,deletions,updatedAt,url,headRefName,baseRefName,labels,statusCheckRollup"
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
	_, err := c.doExec("git", "fetch", remote,
		fmt.Sprintf("+refs/heads/%s:refs/remotes/%s/%s", pr.HeadRef, remote, pr.HeadRef),
		fmt.Sprintf("+refs/heads/%s:refs/remotes/%s/%s", pr.BaseRef, remote, pr.BaseRef),
	)
	if err != nil {
		return "", fmt.Errorf("git fetch: %w", err)
	}

	// Three-dot diff: changes on head since it diverged from base
	out, err := c.doExec("git", "diff",
		fmt.Sprintf("%s/%s...%s/%s", remote, pr.BaseRef, remote, pr.HeadRef))
	if err != nil {
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

// GetChecks returns the full check details for a PR.
func (c *Client) GetChecks(number int) ([]Check, error) {
	fields := "bucket,completedAt,description,event,link,name,startedAt,state,workflow"
	out, err := c.run("pr", "checks", fmt.Sprintf("%d", number), "--json", fields)
	if err != nil {
		return nil, err
	}
	var checks []Check
	if err := json.Unmarshal([]byte(out), &checks); err != nil {
		return nil, fmt.Errorf("parsing checks: %w", err)
	}
	return checks, nil
}

// OpenURL opens a URL in the default browser.
func (c *Client) OpenURL(url string) error {
	if c.runFn != nil {
		_, err := c.runFn("open-url", url)
		return err
	}
	_, err := c.doExec("open", url)
	return err
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
		repo, err = c.ResolveRepo()
		if err != nil {
			return MergeSettings{}, err
		}
	}
	out, err := c.doExec("gh", "api", fmt.Sprintf("repos/%s", repo),
		"--jq", "{squash: .allow_squash_merge, merge: .allow_merge_commit, rebase: .allow_rebase_merge}")
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

// GetReviewComments fetches all review comments for a PR.
func (c *Client) GetReviewComments(number int) ([]ReviewComment, error) {
	repo := c.repo
	if repo == "" {
		var err error
		repo, err = c.ResolveRepo()
		if err != nil {
			return nil, err
		}
	}
	out, err := c.doExec("gh", "api", "--paginate",
		fmt.Sprintf("repos/%s/pulls/%d/comments", repo, number))
	if err != nil {
		return nil, fmt.Errorf("get comments: %w", err)
	}
	var comments []ReviewComment
	if err := json.Unmarshal(out, &comments); err != nil {
		return nil, fmt.Errorf("parsing comments: %w", err)
	}
	return comments, nil
}

// CreateReviewComment posts a new review comment on a PR.
func (c *Client) CreateReviewComment(number int, body, path, commitID string, line, startLine int, side string) error {
	repo := c.repo
	if repo == "" {
		var err error
		repo, err = c.ResolveRepo()
		if err != nil {
			return err
		}
	}

	payload := map[string]interface{}{
		"body":      body,
		"path":      path,
		"commit_id": commitID,
		"line":      line,
		"side":      side,
	}
	if startLine > 0 && startLine != line {
		payload["start_line"] = startLine
		payload["start_side"] = side
	}

	// json.Marshal cannot fail for map[string]interface{} with string/int values
	payloadJSON, _ := json.Marshal(payload)

	out, err := c.doExecWithStdin(string(payloadJSON), "gh", "api",
		fmt.Sprintf("repos/%s/pulls/%d/comments", repo, number),
		"-X", "POST",
		"--input", "-")
	if err != nil {
		return fmt.Errorf("post comment: %s", string(out))
	}
	return nil
}

// ReplyToComment replies to an existing review comment thread.
func (c *Client) ReplyToComment(number, inReplyToID int, body string) error {
	repo := c.repo
	if repo == "" {
		var err error
		repo, err = c.ResolveRepo()
		if err != nil {
			return err
		}
	}

	payload := map[string]interface{}{
		"body":        body,
		"in_reply_to": inReplyToID,
	}
	// json.Marshal cannot fail for map[string]interface{} with string/int values
	payloadJSON, _ := json.Marshal(payload)

	out, err := c.doExecWithStdin(string(payloadJSON), "gh", "api",
		fmt.Sprintf("repos/%s/pulls/%d/comments", repo, number),
		"-X", "POST",
		"--input", "-")
	if err != nil {
		return fmt.Errorf("reply comment: %s", string(out))
	}
	return nil
}

// GetPRHeadSHA returns the head commit SHA for a PR.
func (c *Client) GetPRHeadSHA(number int) (string, error) {
	out, err := c.run("pr", "view", fmt.Sprintf("%d", number), "--json", "headRefOid", "-q", ".headRefOid")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

