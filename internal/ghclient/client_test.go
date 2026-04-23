package ghclient

import (
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// mockRunner returns a runFn that records calls and returns canned responses.
type mockCall struct {
	args []string
}

func newMockRunner(output string, err error) (func(args ...string) (string, error), *[]mockCall) {
	var calls []mockCall
	fn := func(args ...string) (string, error) {
		calls = append(calls, mockCall{args: args})
		return output, err
	}
	return fn, &calls
}

func newMockRunnerMulti(responses []struct {
	output string
	err    error
}) (func(args ...string) (string, error), *[]mockCall) {
	var calls []mockCall
	idx := 0
	fn := func(args ...string) (string, error) {
		calls = append(calls, mockCall{args: args})
		if idx < len(responses) {
			r := responses[idx]
			idx++
			return r.output, r.err
		}
		return "", errors.New("no more mock responses")
	}
	return fn, &calls
}

func TestListPRs_Success(t *testing.T) {
	prs := []prJSON{
		{
			Number:      1,
			Title:       "Test PR",
			State:       "OPEN",
			HeadRefName: "feature",
			BaseRefName: "main",
		},
	}
	prs[0].Author.Login = "user1"
	raw, _ := json.Marshal(prs)
	runFn, calls := newMockRunner(string(raw), nil)

	c := &Client{repo: "owner/repo", runFn: runFn}
	result, err := c.ListPRs(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 PR, got %d", len(result))
	}
	if result[0].Number != 1 {
		t.Errorf("Number = %d, want 1", result[0].Number)
	}
	if result[0].Author != "user1" {
		t.Errorf("Author = %q, want %q", result[0].Author, "user1")
	}
	if result[0].HeadRef != "feature" {
		t.Errorf("HeadRef = %q, want %q", result[0].HeadRef, "feature")
	}

	// Check args
	if len(*calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(*calls))
	}
	args := (*calls)[0].args
	if args[0] != "pr" || args[1] != "list" {
		t.Errorf("args = %v, want pr list ...", args)
	}
}

func TestListPRs_RunError(t *testing.T) {
	runFn, _ := newMockRunner("", errors.New("gh failed"))
	c := &Client{repo: "owner/repo", runFn: runFn}
	_, err := c.ListPRs(10)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListPRs_InvalidJSON(t *testing.T) {
	runFn, _ := newMockRunner("not json", nil)
	c := &Client{repo: "owner/repo", runFn: runFn}
	_, err := c.ListPRs(10)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing PR list") {
		t.Errorf("error = %q, want containing 'parsing PR list'", err.Error())
	}
}

func TestListPRs_Empty(t *testing.T) {
	runFn, _ := newMockRunner("[]", nil)
	c := &Client{repo: "owner/repo", runFn: runFn}
	result, err := c.ListPRs(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 PRs, got %d", len(result))
	}
}

func TestListPRs_WithLabelsAndChecks(t *testing.T) {
	prs := []prJSON{
		{
			Number: 42,
			Title:  "Feature",
			StatusCheckRollup: []statusCheckJSON{
				{Typename: "CheckRun", Conclusion: "SUCCESS"},
				{Typename: "CheckRun", Conclusion: "FAILURE"},
			},
		},
	}
	prs[0].Labels = []struct {
		Name string `json:"name"`
	}{{Name: "bug"}, {Name: "priority"}}
	raw, _ := json.Marshal(prs)
	runFn, _ := newMockRunner(string(raw), nil)

	c := &Client{repo: "owner/repo", runFn: runFn}
	result, err := c.ListPRs(50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pr := result[0]
	if len(pr.Labels) != 2 {
		t.Errorf("Labels = %v, want 2 labels", pr.Labels)
	}
	if pr.CheckSummary.Total != 2 {
		t.Errorf("CheckSummary.Total = %d, want 2", pr.CheckSummary.Total)
	}
	if pr.CheckSummary.Pass != 1 {
		t.Errorf("CheckSummary.Pass = %d, want 1", pr.CheckSummary.Pass)
	}
	if pr.CheckSummary.Fail != 1 {
		t.Errorf("CheckSummary.Fail = %d, want 1", pr.CheckSummary.Fail)
	}
}

func TestGetDiff_Success(t *testing.T) {
	runFn, calls := newMockRunner("diff content here\n", nil)
	c := &Client{repo: "owner/repo", runFn: runFn}
	pr := PR{Number: 5}
	result, err := c.GetDiff(pr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "diff content here\n" {
		t.Errorf("result = %q, want diff content", result)
	}
	args := (*calls)[0].args
	if args[0] != "pr" || args[1] != "diff" || args[2] != "5" {
		t.Errorf("args = %v, want [pr diff 5]", args)
	}
}

func TestGetParsedDiff_Success(t *testing.T) {
	diffText := `diff --git a/f.go b/f.go
index 1234..5678 100644
--- a/f.go
+++ b/f.go
@@ -1,2 +1,3 @@
 line1
+added
 line2
`
	runFn, _ := newMockRunner(diffText, nil)
	c := &Client{repo: "owner/repo", runFn: runFn}
	pr := PR{Number: 1}
	parsed, err := c.GetParsedDiff(pr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parsed.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(parsed.Files))
	}
	if parsed.Files[0].NewPath != "f.go" {
		t.Errorf("NewPath = %q, want %q", parsed.Files[0].NewPath, "f.go")
	}
	if parsed.Files[0].Additions() != 1 {
		t.Errorf("Additions = %d, want 1", parsed.Files[0].Additions())
	}
}

func TestGetParsedDiff_Error(t *testing.T) {
	// GetDiff fails → getLocalDiff also fails → error propagated
	// Since getLocalDiff uses exec.Command directly (not runFn), we can only test when
	// the first run call fails AND getLocalDiff also fails
	// But getLocalDiff calls exec.Command directly, so in test it will fail naturally
	runFn, _ := newMockRunner("", errors.New("diff failed"))
	c := &Client{repo: "owner/repo", runFn: runFn}
	_, err := c.GetParsedDiff(PR{Number: 1})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetChecks_Success(t *testing.T) {
	checks := []Check{
		{
			Name:        "CI",
			State:       "completed",
			Bucket:      "pass",
			Description: "Build passed",
			Link:        "https://example.com/ci",
			Workflow:    "build",
			Event:       "push",
			StartedAt:   time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			CompletedAt: time.Date(2026, 1, 1, 10, 5, 0, 0, time.UTC),
		},
		{
			Name:   "Lint",
			State:  "completed",
			Bucket: "fail",
		},
	}
	raw, _ := json.Marshal(checks)
	runFn, calls := newMockRunner(string(raw), nil)

	c := &Client{repo: "owner/repo", runFn: runFn}
	result, err := c.GetChecks(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(result))
	}
	if result[0].Name != "CI" {
		t.Errorf("checks[0].Name = %q, want %q", result[0].Name, "CI")
	}
	if result[1].Bucket != "fail" {
		t.Errorf("checks[1].Bucket = %q, want %q", result[1].Bucket, "fail")
	}
	args := (*calls)[0].args
	if args[0] != "pr" || args[1] != "checks" || args[2] != "42" {
		t.Errorf("args = %v, want [pr checks 42 ...]", args)
	}
}

func TestGetChecks_RunError(t *testing.T) {
	runFn, _ := newMockRunner("", errors.New("checks failed"))
	c := &Client{repo: "owner/repo", runFn: runFn}
	_, err := c.GetChecks(1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetChecks_InvalidJSON(t *testing.T) {
	runFn, _ := newMockRunner("invalid", nil)
	c := &Client{repo: "owner/repo", runFn: runFn}
	_, err := c.GetChecks(1)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing checks") {
		t.Errorf("error = %q, want containing 'parsing checks'", err.Error())
	}
}

func TestMarkReady_Success(t *testing.T) {
	runFn, calls := newMockRunner("", nil)
	c := &Client{repo: "owner/repo", runFn: runFn}
	err := c.MarkReady(7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	args := (*calls)[0].args
	if args[0] != "pr" || args[1] != "ready" || args[2] != "7" {
		t.Errorf("args = %v, want [pr ready 7]", args)
	}
}

func TestMarkReady_Error(t *testing.T) {
	runFn, _ := newMockRunner("", errors.New("ready failed"))
	c := &Client{repo: "owner/repo", runFn: runFn}
	err := c.MarkReady(7)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMergePR_Success(t *testing.T) {
	runFn, calls := newMockRunner("", nil)
	c := &Client{repo: "owner/repo", runFn: runFn}
	err := c.MergePR(10, "squash", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(*calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(*calls))
	}
	args := (*calls)[0].args
	if args[0] != "pr" || args[1] != "merge" || args[2] != "10" || args[3] != "--squash" || args[4] != "--delete-branch" {
		t.Errorf("args = %v, want [pr merge 10 --squash --delete-branch]", args)
	}
}

func TestMergePR_WithUndraft(t *testing.T) {
	responses := []struct {
		output string
		err    error
	}{
		{"", nil}, // MarkReady
		{"", nil}, // MergePR
	}
	runFn, calls := newMockRunnerMulti(responses)
	c := &Client{repo: "owner/repo", runFn: runFn}
	err := c.MergePR(10, "merge", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(*calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(*calls))
	}
	// First call: MarkReady
	if (*calls)[0].args[1] != "ready" {
		t.Errorf("first call args = %v, want pr ready", (*calls)[0].args)
	}
	// Second call: merge
	if (*calls)[1].args[1] != "merge" {
		t.Errorf("second call args = %v, want pr merge", (*calls)[1].args)
	}
}

func TestMergePR_UndraftFails(t *testing.T) {
	runFn, _ := newMockRunner("", errors.New("ready failed"))
	c := &Client{repo: "owner/repo", runFn: runFn}
	err := c.MergePR(10, "squash", true)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "undraft") {
		t.Errorf("error = %q, want containing 'undraft'", err.Error())
	}
}

func TestMergePR_MergeFails(t *testing.T) {
	responses := []struct {
		output string
		err    error
	}{
		{"", nil},                       // MarkReady succeeds
		{"", errors.New("merge fail")},  // Merge fails
	}
	runFn, _ := newMockRunnerMulti(responses)
	c := &Client{repo: "owner/repo", runFn: runFn}
	err := c.MergePR(10, "rebase", true)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMergePR_Methods(t *testing.T) {
	for _, method := range []string{"squash", "merge", "rebase"} {
		t.Run(method, func(t *testing.T) {
			runFn, calls := newMockRunner("", nil)
			c := &Client{repo: "owner/repo", runFn: runFn}
			err := c.MergePR(1, method, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			args := (*calls)[0].args
			if args[3] != "--"+method {
				t.Errorf("args[3] = %q, want %q", args[3], "--"+method)
			}
		})
	}
}

func TestToggleDraft_IsDraft(t *testing.T) {
	runFn, calls := newMockRunner("", nil)
	c := &Client{repo: "owner/repo", runFn: runFn}
	err := c.ToggleDraft(5, true) // isDraft → mark ready
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	args := (*calls)[0].args
	if args[0] != "pr" || args[1] != "ready" || args[2] != "5" {
		t.Errorf("args = %v, want [pr ready 5]", args)
	}
	// Should NOT have --undo
	for _, a := range args {
		if a == "--undo" {
			t.Error("isDraft=true should not pass --undo")
		}
	}
}

func TestToggleDraft_IsReady(t *testing.T) {
	runFn, calls := newMockRunner("", nil)
	c := &Client{repo: "owner/repo", runFn: runFn}
	err := c.ToggleDraft(5, false) // ready → convert to draft
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	args := (*calls)[0].args
	if args[0] != "pr" || args[1] != "ready" || args[2] != "5" {
		t.Errorf("args = %v, want [pr ready 5 --undo]", args)
	}
	foundUndo := false
	for _, a := range args {
		if a == "--undo" {
			foundUndo = true
		}
	}
	if !foundUndo {
		t.Error("isDraft=false should pass --undo")
	}
}

func TestToggleDraft_Error(t *testing.T) {
	runFn, _ := newMockRunner("", errors.New("toggle failed"))
	c := &Client{repo: "owner/repo", runFn: runFn}
	err := c.ToggleDraft(5, true)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpenInBrowser_Success(t *testing.T) {
	runFn, calls := newMockRunner("", nil)
	c := &Client{repo: "owner/repo", runFn: runFn}
	err := c.OpenInBrowser(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	args := (*calls)[0].args
	if args[0] != "pr" || args[1] != "view" || args[2] != "42" || args[3] != "--web" {
		t.Errorf("args = %v, want [pr view 42 --web]", args)
	}
}

func TestOpenInBrowser_Error(t *testing.T) {
	runFn, _ := newMockRunner("", errors.New("open failed"))
	c := &Client{repo: "owner/repo", runFn: runFn}
	err := c.OpenInBrowser(1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetPRHeadSHA_Success(t *testing.T) {
	runFn, calls := newMockRunner("abc123def456\n", nil)
	c := &Client{repo: "owner/repo", runFn: runFn}
	sha, err := c.GetPRHeadSHA(7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sha != "abc123def456" {
		t.Errorf("sha = %q, want %q", sha, "abc123def456")
	}
	args := (*calls)[0].args
	if args[0] != "pr" || args[1] != "view" || args[2] != "7" {
		t.Errorf("args = %v, want [pr view 7 ...]", args)
	}
}

func TestGetPRHeadSHA_Error(t *testing.T) {
	runFn, _ := newMockRunner("", errors.New("sha failed"))
	c := &Client{repo: "owner/repo", runFn: runFn}
	_, err := c.GetPRHeadSHA(1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetPRHeadSHA_TrimsWhitespace(t *testing.T) {
	runFn, _ := newMockRunner("  abc123  \n\n", nil)
	c := &Client{repo: "owner/repo", runFn: runFn}
	sha, err := c.GetPRHeadSHA(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sha != "abc123" {
		t.Errorf("sha = %q, want %q", sha, "abc123")
	}
}

func TestListPRs_FullFields(t *testing.T) {
	now := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC)
	prs := []prJSON{
		{
			Number:      99,
			Title:       "Full PR",
			State:       "OPEN",
			IsDraft:     true,
			Additions:   100,
			Deletions:   50,
			UpdatedAt:   now,
			URL:         "https://github.com/owner/repo/pull/99",
			HeadRefName: "feature-branch",
			BaseRefName: "main",
		},
	}
	prs[0].Author.Login = "developer"
	raw, _ := json.Marshal(prs)
	runFn, _ := newMockRunner(string(raw), nil)

	c := &Client{repo: "owner/repo", runFn: runFn}
	result, err := c.ListPRs(100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pr := result[0]
	if pr.Title != "Full PR" {
		t.Errorf("Title = %q", pr.Title)
	}
	if pr.State != "OPEN" {
		t.Errorf("State = %q", pr.State)
	}
	if !pr.IsDraft {
		t.Error("expected IsDraft=true")
	}
	if pr.Additions != 100 {
		t.Errorf("Additions = %d", pr.Additions)
	}
	if pr.Deletions != 50 {
		t.Errorf("Deletions = %d", pr.Deletions)
	}
	if pr.URL != "https://github.com/owner/repo/pull/99" {
		t.Errorf("URL = %q", pr.URL)
	}
	if pr.BaseRef != "main" {
		t.Errorf("BaseRef = %q", pr.BaseRef)
	}
}

func TestNewTestClient(t *testing.T) {
	called := false
	c := NewTestClient("owner/repo", func(args ...string) (string, error) {
		called = true
		return "ok", nil
	})
	if c.repo != "owner/repo" {
		t.Errorf("repo = %q, want %q", c.repo, "owner/repo")
	}
	out, err := c.run("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "ok" {
		t.Errorf("output = %q, want %q", out, "ok")
	}
	if !called {
		t.Error("runFn was not called")
	}
}

func TestRun_WithRunFn(t *testing.T) {
	called := false
	c := &Client{
		repo: "owner/repo",
		runFn: func(args ...string) (string, error) {
			called = true
			if args[0] != "test" || args[1] != "arg" {
				t.Errorf("args = %v, want [test arg]", args)
			}
			return "output", nil
		},
	}
	out, err := c.run("test", "arg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "output" {
		t.Errorf("output = %q, want %q", out, "output")
	}
	if !called {
		t.Error("runFn was not called")
	}
}

// --- Tests for execFn-based methods ---

type execCall struct {
	name string
	args []string
}

func newMockExec(output []byte, err error) (func(name string, args ...string) ([]byte, error), *[]execCall) {
	var calls []execCall
	fn := func(name string, args ...string) ([]byte, error) {
		calls = append(calls, execCall{name: name, args: args})
		return output, err
	}
	return fn, &calls
}

type stdinExecCall struct {
	stdin string
	name  string
	args  []string
}

func newMockExecStdin(output []byte, err error) (func(stdin, name string, args ...string) ([]byte, error), *[]stdinExecCall) {
	var calls []stdinExecCall
	fn := func(stdin, name string, args ...string) ([]byte, error) {
		calls = append(calls, stdinExecCall{stdin: stdin, name: name, args: args})
		return output, err
	}
	return fn, &calls
}

func newMockExecMulti(responses []struct {
	output []byte
	err    error
}) (func(name string, args ...string) ([]byte, error), *[]execCall) {
	var calls []execCall
	idx := 0
	fn := func(name string, args ...string) ([]byte, error) {
		calls = append(calls, execCall{name: name, args: args})
		if idx < len(responses) {
			r := responses[idx]
			idx++
			return r.output, r.err
		}
		return nil, errors.New("no more mock responses")
	}
	return fn, &calls
}

func TestGetLocalDiff_Success(t *testing.T) {
	responses := []struct {
		output []byte
		err    error
	}{
		{[]byte(""), nil},               // git fetch
		{[]byte("diff content\n"), nil}, // git diff
	}
	execFn, calls := newMockExecMulti(responses)
	// runFn returns error to trigger fallback to getLocalDiff
	runFn, _ := newMockRunner("", errors.New("gh pr diff failed"))
	c := &Client{repo: "owner/repo", runFn: runFn, execFn: execFn}
	pr := PR{Number: 1, HeadRef: "feature", BaseRef: "main"}
	result, err := c.GetDiff(pr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "diff content\n" {
		t.Errorf("result = %q, want %q", result, "diff content\n")
	}
	if len(*calls) != 2 {
		t.Fatalf("expected 2 exec calls, got %d", len(*calls))
	}
	if (*calls)[0].name != "git" {
		t.Errorf("first call name = %q, want git", (*calls)[0].name)
	}
	if (*calls)[0].args[0] != "fetch" {
		t.Errorf("first call args[0] = %q, want fetch", (*calls)[0].args[0])
	}
	if (*calls)[1].name != "git" {
		t.Errorf("second call name = %q, want git", (*calls)[1].name)
	}
	if (*calls)[1].args[0] != "diff" {
		t.Errorf("second call args[0] = %q, want diff", (*calls)[1].args[0])
	}
}

func TestGetLocalDiff_FetchError(t *testing.T) {
	execFn, _ := newMockExec(nil, errors.New("fetch failed"))
	runFn, _ := newMockRunner("", errors.New("gh pr diff failed"))
	c := &Client{repo: "owner/repo", runFn: runFn, execFn: execFn}
	pr := PR{Number: 1, HeadRef: "feature", BaseRef: "main"}
	_, err := c.GetDiff(pr)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "git fetch") {
		t.Errorf("error = %q, want containing 'git fetch'", err.Error())
	}
}

func TestGetLocalDiff_DiffError(t *testing.T) {
	responses := []struct {
		output []byte
		err    error
	}{
		{[]byte(""), nil},                         // git fetch succeeds
		{nil, errors.New("diff command failed")},  // git diff fails
	}
	execFn, _ := newMockExecMulti(responses)
	runFn, _ := newMockRunner("", errors.New("gh pr diff failed"))
	c := &Client{repo: "owner/repo", runFn: runFn, execFn: execFn}
	pr := PR{Number: 1, HeadRef: "feature", BaseRef: "main"}
	_, err := c.GetDiff(pr)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "git diff") {
		t.Errorf("error = %q, want containing 'git diff'", err.Error())
	}
}

func TestGetMergeSettings_Success(t *testing.T) {
	raw := `{"squash": true, "merge": false, "rebase": true}`
	execFn, calls := newMockExec([]byte(raw), nil)
	c := &Client{repo: "owner/repo", execFn: execFn}
	ms, err := c.GetMergeSettings()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ms.AllowSquash {
		t.Error("expected AllowSquash=true")
	}
	if ms.AllowMerge {
		t.Error("expected AllowMerge=false")
	}
	if !ms.AllowRebase {
		t.Error("expected AllowRebase=true")
	}
	if len(*calls) != 1 {
		t.Fatalf("expected 1 exec call, got %d", len(*calls))
	}
	if (*calls)[0].name != "gh" {
		t.Errorf("exec name = %q, want gh", (*calls)[0].name)
	}
}

func TestGetMergeSettings_ExecError_Fallback(t *testing.T) {
	execFn, _ := newMockExec(nil, errors.New("api failed"))
	c := &Client{repo: "owner/repo", execFn: execFn}
	ms, err := c.GetMergeSettings()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Fallback: all allowed
	if !ms.AllowSquash || !ms.AllowMerge || !ms.AllowRebase {
		t.Errorf("expected all allowed on fallback, got %+v", ms)
	}
}

func TestGetMergeSettings_InvalidJSON_Fallback(t *testing.T) {
	execFn, _ := newMockExec([]byte("not json"), nil)
	c := &Client{repo: "owner/repo", execFn: execFn}
	ms, err := c.GetMergeSettings()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ms.AllowSquash || !ms.AllowMerge || !ms.AllowRebase {
		t.Errorf("expected all allowed on fallback, got %+v", ms)
	}
}

func TestGetMergeSettings_ResolveRepo(t *testing.T) {
	raw := `{"squash": true, "merge": true, "rebase": false}`
	execFn, _ := newMockExec([]byte(raw), nil)
	c := &Client{
		repo:   "", // empty repo triggers resolveRepo
		execFn: execFn,
		resolveRepoFn: func() (string, error) {
			return "resolved/repo", nil
		},
	}
	ms, err := c.GetMergeSettings()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ms.AllowSquash || !ms.AllowMerge || ms.AllowRebase {
		t.Errorf("unexpected settings: %+v", ms)
	}
}

func TestGetMergeSettings_ResolveRepoError(t *testing.T) {
	c := &Client{
		repo: "",
		resolveRepoFn: func() (string, error) {
			return "", errors.New("resolve failed")
		},
	}
	_, err := c.GetMergeSettings()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetReviewComments_Success(t *testing.T) {
	comments := []ReviewComment{
		{ID: 1, Body: "looks good", Path: "main.go", Line: 10},
		{ID: 2, Body: "fix this", Path: "util.go", Line: 5},
	}
	raw, _ := json.Marshal(comments)
	execFn, calls := newMockExec(raw, nil)
	c := &Client{repo: "owner/repo", execFn: execFn}
	result, err := c.GetReviewComments(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(result))
	}
	if result[0].Body != "looks good" {
		t.Errorf("comment[0].Body = %q", result[0].Body)
	}
	if (*calls)[0].name != "gh" {
		t.Errorf("exec name = %q, want gh", (*calls)[0].name)
	}
}

func TestGetReviewComments_Error(t *testing.T) {
	execFn, _ := newMockExec(nil, errors.New("api failed"))
	c := &Client{repo: "owner/repo", execFn: execFn}
	_, err := c.GetReviewComments(1)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "get comments") {
		t.Errorf("error = %q, want containing 'get comments'", err.Error())
	}
}

func TestGetReviewComments_InvalidJSON(t *testing.T) {
	execFn, _ := newMockExec([]byte("not json"), nil)
	c := &Client{repo: "owner/repo", execFn: execFn}
	_, err := c.GetReviewComments(1)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing comments") {
		t.Errorf("error = %q, want containing 'parsing comments'", err.Error())
	}
}

func TestGetReviewComments_ResolveRepo(t *testing.T) {
	comments := []ReviewComment{{ID: 1, Body: "test"}}
	raw, _ := json.Marshal(comments)
	execFn, _ := newMockExec(raw, nil)
	c := &Client{
		repo:   "",
		execFn: execFn,
		resolveRepoFn: func() (string, error) {
			return "resolved/repo", nil
		},
	}
	result, err := c.GetReviewComments(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 comment, got %d", len(result))
	}
}

func TestGetReviewComments_ResolveRepoError(t *testing.T) {
	c := &Client{
		repo: "",
		resolveRepoFn: func() (string, error) {
			return "", errors.New("resolve failed")
		},
	}
	_, err := c.GetReviewComments(1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateReviewComment_Success(t *testing.T) {
	stdinFn, calls := newMockExecStdin([]byte("{}"), nil)
	c := &Client{repo: "owner/repo", execStdinFn: stdinFn}
	err := c.CreateReviewComment(42, "nice code", "main.go", "abc123", 10, 0, "RIGHT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(*calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(*calls))
	}
	call := (*calls)[0]
	if call.name != "gh" {
		t.Errorf("name = %q, want gh", call.name)
	}
	// Verify payload contains expected fields
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(call.stdin), &payload); err != nil {
		t.Fatalf("failed to parse stdin payload: %v", err)
	}
	if payload["body"] != "nice code" {
		t.Errorf("body = %v", payload["body"])
	}
	if payload["path"] != "main.go" {
		t.Errorf("path = %v", payload["path"])
	}
	if payload["commit_id"] != "abc123" {
		t.Errorf("commit_id = %v", payload["commit_id"])
	}
	if payload["side"] != "RIGHT" {
		t.Errorf("side = %v", payload["side"])
	}
	// startLine=0 should not add start_line
	if _, ok := payload["start_line"]; ok {
		t.Error("start_line should not be present when startLine=0")
	}
}

func TestCreateReviewComment_WithStartLine(t *testing.T) {
	stdinFn, calls := newMockExecStdin([]byte("{}"), nil)
	c := &Client{repo: "owner/repo", execStdinFn: stdinFn}
	err := c.CreateReviewComment(42, "range comment", "main.go", "abc123", 20, 15, "RIGHT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte((*calls)[0].stdin), &payload); err != nil {
		t.Fatalf("failed to parse stdin payload: %v", err)
	}
	if payload["start_line"] != float64(15) {
		t.Errorf("start_line = %v, want 15", payload["start_line"])
	}
	if payload["start_side"] != "RIGHT" {
		t.Errorf("start_side = %v", payload["start_side"])
	}
}

func TestCreateReviewComment_StartLineEqualsLine(t *testing.T) {
	stdinFn, calls := newMockExecStdin([]byte("{}"), nil)
	c := &Client{repo: "owner/repo", execStdinFn: stdinFn}
	err := c.CreateReviewComment(42, "single line", "main.go", "abc123", 10, 10, "RIGHT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte((*calls)[0].stdin), &payload); err != nil {
		t.Fatalf("failed to parse stdin payload: %v", err)
	}
	// When startLine == line, start_line should not be set
	if _, ok := payload["start_line"]; ok {
		t.Error("start_line should not be present when startLine == line")
	}
}

func TestCreateReviewComment_Error(t *testing.T) {
	stdinFn, _ := newMockExecStdin([]byte("error output"), errors.New("api failed"))
	c := &Client{repo: "owner/repo", execStdinFn: stdinFn}
	err := c.CreateReviewComment(42, "body", "path", "sha", 1, 0, "RIGHT")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "post comment") {
		t.Errorf("error = %q, want containing 'post comment'", err.Error())
	}
}

func TestCreateReviewComment_ResolveRepo(t *testing.T) {
	stdinFn, _ := newMockExecStdin([]byte("{}"), nil)
	c := &Client{
		repo:        "",
		execStdinFn: stdinFn,
		resolveRepoFn: func() (string, error) {
			return "resolved/repo", nil
		},
	}
	err := c.CreateReviewComment(1, "body", "path", "sha", 1, 0, "RIGHT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateReviewComment_ResolveRepoError(t *testing.T) {
	c := &Client{
		repo: "",
		resolveRepoFn: func() (string, error) {
			return "", errors.New("resolve failed")
		},
	}
	err := c.CreateReviewComment(1, "body", "path", "sha", 1, 0, "RIGHT")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReplyToComment_Success(t *testing.T) {
	stdinFn, calls := newMockExecStdin([]byte("{}"), nil)
	c := &Client{repo: "owner/repo", execStdinFn: stdinFn}
	err := c.ReplyToComment(42, 100, "thanks for the review")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(*calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(*calls))
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte((*calls)[0].stdin), &payload); err != nil {
		t.Fatalf("failed to parse stdin payload: %v", err)
	}
	if payload["body"] != "thanks for the review" {
		t.Errorf("body = %v", payload["body"])
	}
	if payload["in_reply_to"] != float64(100) {
		t.Errorf("in_reply_to = %v, want 100", payload["in_reply_to"])
	}
}

func TestReplyToComment_Error(t *testing.T) {
	stdinFn, _ := newMockExecStdin([]byte("error"), errors.New("api failed"))
	c := &Client{repo: "owner/repo", execStdinFn: stdinFn}
	err := c.ReplyToComment(42, 100, "reply")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "reply comment") {
		t.Errorf("error = %q, want containing 'reply comment'", err.Error())
	}
}

func TestReplyToComment_ResolveRepo(t *testing.T) {
	stdinFn, _ := newMockExecStdin([]byte("{}"), nil)
	c := &Client{
		repo:        "",
		execStdinFn: stdinFn,
		resolveRepoFn: func() (string, error) {
			return "resolved/repo", nil
		},
	}
	err := c.ReplyToComment(1, 50, "reply body")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReplyToComment_ResolveRepoError(t *testing.T) {
	c := &Client{
		repo: "",
		resolveRepoFn: func() (string, error) {
			return "", errors.New("resolve failed")
		},
	}
	err := c.ReplyToComment(1, 50, "reply")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDoExec_WithExecFn(t *testing.T) {
	execFn, calls := newMockExec([]byte("output"), nil)
	c := &Client{execFn: execFn}
	out, err := c.doExec("test", "arg1", "arg2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "output" {
		t.Errorf("output = %q, want %q", string(out), "output")
	}
	if len(*calls) != 1 || (*calls)[0].name != "test" {
		t.Errorf("calls = %v", *calls)
	}
}

func TestDoExecWithStdin_WithStdinFn(t *testing.T) {
	stdinFn, calls := newMockExecStdin([]byte("result"), nil)
	c := &Client{execStdinFn: stdinFn}
	out, err := c.doExecWithStdin("input data", "cmd", "arg1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "result" {
		t.Errorf("output = %q, want %q", string(out), "result")
	}
	if len(*calls) != 1 || (*calls)[0].stdin != "input data" || (*calls)[0].name != "cmd" {
		t.Errorf("calls = %v", *calls)
	}
}

func TestOpenURL_WithRunFn(t *testing.T) {
	var calledArgs []string
	c := &Client{
		runFn: func(args ...string) (string, error) {
			calledArgs = args
			return "", nil
		},
	}
	err := c.OpenURL("https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calledArgs) != 2 || calledArgs[0] != "open-url" || calledArgs[1] != "https://example.com" {
		t.Errorf("args = %v, want [open-url https://example.com]", calledArgs)
	}
}

func TestOpenURL_WithRunFn_Error(t *testing.T) {
	c := &Client{
		runFn: func(args ...string) (string, error) {
			return "", errors.New("open failed")
		},
	}
	err := c.OpenURL("https://example.com")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveRepo_WithResolveRepoFn(t *testing.T) {
	c := &Client{
		resolveRepoFn: func() (string, error) {
			return "custom/repo", nil
		},
	}
	repo, err := c.ResolveRepo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo != "custom/repo" {
		t.Errorf("repo = %q, want %q", repo, "custom/repo")
	}
}

// --- Tests for resolveRepo() fallback path (resolveRepoFn=nil, execFn set) ---

func TestResolveRepo_FallbackSuccess(t *testing.T) {
	c := &Client{
		execFn: func(name string, args ...string) ([]byte, error) {
			if name != "gh" {
				t.Errorf("expected name=gh, got %q", name)
			}
			return []byte("owner/repo\n"), nil
		},
	}
	repo, err := c.ResolveRepo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo != "owner/repo" {
		t.Errorf("repo = %q, want %q", repo, "owner/repo")
	}
}

func TestResolveRepo_FallbackError(t *testing.T) {
	c := &Client{
		execFn: func(name string, args ...string) ([]byte, error) {
			return nil, errors.New("not a git repo")
		},
	}
	_, err := c.ResolveRepo()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "could not determine repository") {
		t.Errorf("error = %v, should contain 'could not determine repository'", err)
	}
}

func TestResolveRepo_FallbackTrimsWhitespace(t *testing.T) {
	c := &Client{
		execFn: func(name string, args ...string) ([]byte, error) {
			return []byte("  owner/repo  \n"), nil
		},
	}
	repo, err := c.ResolveRepo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo != "owner/repo" {
		t.Errorf("repo = %q, want %q (trimmed)", repo, "owner/repo")
	}
}

// --- Tests for run() fallback path (runFn=nil, execFn set) ---
// When runFn is nil, run() delegates to doExec which uses execFn.
// This lets us test run()'s error handling without real subprocesses.

func TestRun_FallbackSuccess(t *testing.T) {
	c := &Client{
		// runFn is nil — run() will call doExec → execFn
		execFn: func(name string, args ...string) ([]byte, error) {
			if name != "gh" {
				t.Errorf("expected name=gh, got %q", name)
			}
			return []byte("output data"), nil
		},
	}
	out, err := c.run("pr", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "output data" {
		t.Errorf("output = %q, want %q", out, "output data")
	}
}

func TestRun_FallbackGenericError(t *testing.T) {
	c := &Client{
		execFn: func(name string, args ...string) ([]byte, error) {
			return nil, errors.New("connection refused")
		},
	}
	_, err := c.run("pr", "list")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "gh pr list") {
		t.Errorf("error should contain command args, got: %v", err)
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("error should contain underlying error, got: %v", err)
	}
}

func TestRun_FallbackExitError(t *testing.T) {
	// Create a real ExitError by running a command that fails
	failCmd := exec.Command("sh", "-c", "echo 'auth required' >&2; exit 1")
	_, realErr := failCmd.Output()

	exitErr, ok := realErr.(*exec.ExitError)
	if !ok {
		t.Skipf("could not create ExitError: %T", realErr)
	}

	c := &Client{
		execFn: func(name string, args ...string) ([]byte, error) {
			return nil, exitErr
		},
	}
	_, err := c.run("pr", "list")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "gh pr list") {
		t.Errorf("error should contain command args, got: %v", err)
	}
	if !strings.Contains(err.Error(), "auth required") {
		t.Errorf("error should contain stderr, got: %v", err)
	}
}

// --- Tests for real exec fallback paths (no mock, using simple system commands) ---

func TestDoExec_RealFallback(t *testing.T) {
	// No execFn set → uses real exec.Command
	c := &Client{}
	out, err := c.doExec("echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(string(out)) != "hello" {
		t.Errorf("output = %q, want %q", strings.TrimSpace(string(out)), "hello")
	}
}

func TestDoExec_RealFallback_Error(t *testing.T) {
	c := &Client{}
	_, err := c.doExec("false")
	if err == nil {
		t.Error("expected error from 'false' command")
	}
}

func TestDoExecWithStdin_RealFallback(t *testing.T) {
	// No execStdinFn set → uses real exec.Command with stdin
	c := &Client{}
	out, err := c.doExecWithStdin("hello from stdin", "cat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(string(out)) != "hello from stdin" {
		t.Errorf("output = %q, want %q", strings.TrimSpace(string(out)), "hello from stdin")
	}
}

func TestDoExecWithStdin_RealFallback_Error(t *testing.T) {
	c := &Client{}
	_, err := c.doExecWithStdin("input", "false")
	if err == nil {
		t.Error("expected error from 'false' command")
	}
}

func TestOpenURL_RealFallback_ViaExecFn(t *testing.T) {
	// OpenURL now uses doExec, so we can test via execFn
	execFn, calls := newMockExec(nil, nil)
	c := &Client{execFn: execFn}
	err := c.OpenURL("https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(*calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(*calls))
	}
	if (*calls)[0].name != "open" {
		t.Errorf("name = %q, want open", (*calls)[0].name)
	}
	if (*calls)[0].args[0] != "https://example.com" {
		t.Errorf("args = %v, want [https://example.com]", (*calls)[0].args)
	}
}

func TestOpenURL_RealFallback_Error(t *testing.T) {
	execFn, _ := newMockExec(nil, errors.New("open failed"))
	c := &Client{execFn: execFn}
	err := c.OpenURL("https://example.com")
	if err == nil {
		t.Error("expected error")
	}
}

func TestRun_FallbackWithRepo(t *testing.T) {
	var capturedArgs []string
	c := &Client{
		repo: "owner/repo",
		execFn: func(name string, args ...string) ([]byte, error) {
			capturedArgs = args
			return []byte("ok"), nil
		},
	}
	_, err := c.run("pr", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have --repo appended via ghArgs
	found := false
	for i, a := range capturedArgs {
		if a == "--repo" && i+1 < len(capturedArgs) && capturedArgs[i+1] == "owner/repo" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected --repo owner/repo in args, got %v", capturedArgs)
	}
}
