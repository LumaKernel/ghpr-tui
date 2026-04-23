package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/LumaKernel/ghpr-tui/internal/version"
)

func TestParseArgs_NoArgs(t *testing.T) {
	action, repo := parseArgs([]string{"ghpr-tui"})
	if action != actionRun {
		t.Errorf("action = %d, want actionRun", action)
	}
	if repo != "" {
		t.Errorf("repo = %q, want empty", repo)
	}
}

func TestParseArgs_Version(t *testing.T) {
	for _, flag := range []string{"-v", "--version"} {
		action, _ := parseArgs([]string{"ghpr-tui", flag})
		if action != actionVersion {
			t.Errorf("parseArgs(%q): action = %d, want actionVersion", flag, action)
		}
	}
}

func TestParseArgs_Help(t *testing.T) {
	for _, flag := range []string{"-h", "--help"} {
		action, _ := parseArgs([]string{"ghpr-tui", flag})
		if action != actionHelp {
			t.Errorf("parseArgs(%q): action = %d, want actionHelp", flag, action)
		}
	}
}

func TestParseArgs_Repo(t *testing.T) {
	action, repo := parseArgs([]string{"ghpr-tui", "owner/repo"})
	if action != actionRun {
		t.Errorf("action = %d, want actionRun", action)
	}
	if repo != "owner/repo" {
		t.Errorf("repo = %q, want %q", repo, "owner/repo")
	}
}

func TestPrintVersion(t *testing.T) {
	var buf bytes.Buffer
	printVersion(&buf)
	got := buf.String()
	if !strings.Contains(got, version.Version) {
		t.Errorf("printVersion() = %q, should contain %q", got, version.Version)
	}
	if !strings.Contains(got, "ghpr-tui") {
		t.Errorf("printVersion() = %q, should contain %q", got, "ghpr-tui")
	}
}

func TestPrintHelp(t *testing.T) {
	var buf bytes.Buffer
	printHelp(&buf)
	got := buf.String()
	if !strings.Contains(got, "Usage:") {
		t.Errorf("printHelp() should contain 'Usage:'")
	}
	if !strings.Contains(got, "Key bindings") {
		t.Errorf("printHelp() should contain 'Key bindings'")
	}
}

func TestHelpText(t *testing.T) {
	// Ensure helpText covers all three screen sections
	if !strings.Contains(helpText, "PR list") {
		t.Error("helpText should mention PR list")
	}
	if !strings.Contains(helpText, "file list") {
		t.Error("helpText should mention file list")
	}
	if !strings.Contains(helpText, "diff view") {
		t.Error("helpText should mention diff view")
	}
}

// --- setup() function tests ---

func TestSetup_Version(t *testing.T) {
	var stdout, stderr bytes.Buffer
	model, err := setup([]string{"ghpr-tui", "--version"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != nil {
		t.Error("model should be nil for --version")
	}
	if !strings.Contains(stdout.String(), version.Version) {
		t.Errorf("stdout = %q, should contain %q", stdout.String(), version.Version)
	}
}

func TestSetup_Help(t *testing.T) {
	var stdout, stderr bytes.Buffer
	model, err := setup([]string{"ghpr-tui", "--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != nil {
		t.Error("model should be nil for --help")
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Errorf("stdout = %q, should contain 'Usage:'", stdout.String())
	}
}

func TestSetup_NoRepo_InGitRepo(t *testing.T) {
	// When run inside a git repo with gh available, setup with no repo arg
	// should resolve the repo successfully
	var stdout, stderr bytes.Buffer
	model, err := setup([]string{"ghpr-tui"}, &stdout, &stderr)
	if err != nil {
		// gh not available or not in a git repo — this is expected in some environments
		t.Skipf("gh repo resolve failed (expected in CI): %v", err)
	}
	if model == nil {
		t.Error("model should not be nil when repo resolves successfully")
	}
}

func TestSetup_WithRepo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	model, err := setup([]string{"ghpr-tui", "owner/repo"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model == nil {
		t.Error("model should not be nil with valid repo")
	}
}

// --- run() function tests ---

func TestRun_Version(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"ghpr-tui", "--version"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), version.Version) {
		t.Errorf("stdout = %q, should contain %q", stdout.String(), version.Version)
	}
}

func TestRun_VersionShort(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"ghpr-tui", "-v"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "ghpr-tui") {
		t.Errorf("stdout = %q, should contain 'ghpr-tui'", stdout.String())
	}
}

func TestRun_Help(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"ghpr-tui", "--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Errorf("stdout = %q, should contain 'Usage:'", stdout.String())
	}
}

func TestRun_HelpShort(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"ghpr-tui", "-h"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Key bindings") {
		t.Errorf("stdout = %q, should contain 'Key bindings'", stdout.String())
	}
}

func TestRun_NoRepo_ResolveError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"ghpr-tui"}, &stdout, &stderr)
	if err == nil {
		t.Error("expected error when not in a git repo")
	}
	if !strings.Contains(stderr.String(), "Error:") {
		t.Errorf("stderr = %q, should contain 'Error:'", stderr.String())
	}
}
