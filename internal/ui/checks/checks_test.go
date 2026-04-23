package checks

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/LumaKernel/ghpr-tui/internal/ghclient"
)

func sampleChecks() []ghclient.Check {
	return []ghclient.Check{
		{Name: "lint", Bucket: "pass", Workflow: "CI", State: "SUCCESS",
			StartedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			CompletedAt: time.Date(2026, 1, 1, 10, 0, 7, 0, time.UTC)},
		{Name: "test", Bucket: "fail", Workflow: "CI", State: "FAILURE",
			StartedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			CompletedAt: time.Date(2026, 1, 1, 10, 4, 1, 0, time.UTC)},
		{Name: "deploy", Bucket: "pending", Workflow: "CD", State: "PENDING",
			StartedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)},
	}
}

func TestChecks_SetChecks_SortOrder(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(ghclient.PR{Number: 1}).SetChecks(sampleChecks())
	// fail should come first
	if m.checks[0].Bucket != "fail" {
		t.Errorf("first check should be fail, got %q", m.checks[0].Bucket)
	}
	if m.checks[1].Bucket != "pending" {
		t.Errorf("second check should be pending, got %q", m.checks[1].Bucket)
	}
	if m.checks[2].Bucket != "pass" {
		t.Errorf("third check should be pass, got %q", m.checks[2].Bucket)
	}
}

func TestChecks_Navigation(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(ghclient.PR{Number: 1}).SetChecks(sampleChecks())

	// Initial cursor at 0
	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.cursor)
	}

	// j moves down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != 1 {
		t.Errorf("after j: cursor = %d, want 1", m.cursor)
	}

	// k moves up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.cursor != 0 {
		t.Errorf("after k: cursor = %d, want 0", m.cursor)
	}

	// k at top stays at 0
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.cursor != 0 {
		t.Errorf("k at top: cursor = %d, want 0", m.cursor)
	}

	// G goes to end
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if m.cursor != 2 {
		t.Errorf("after G: cursor = %d, want 2", m.cursor)
	}

	// g goes to start
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if m.cursor != 0 {
		t.Errorf("after g: cursor = %d, want 0", m.cursor)
	}
}

func TestChecks_BackMsg(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(ghclient.PR{Number: 1}).SetChecks(sampleChecks())
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc should produce a cmd")
	}
	msg := cmd()
	if _, ok := msg.(BackMsg); !ok {
		t.Errorf("esc should produce BackMsg, got %T", msg)
	}
}

func TestChecks_OpenBrowser(t *testing.T) {
	chks := []ghclient.Check{
		{Name: "test", Bucket: "fail", Link: "https://example.com/log"},
	}
	m := New().SetSize(80, 24).SetPR(ghclient.PR{Number: 1}).SetChecks(chks)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter should produce a cmd")
	}
	msg := cmd()
	openMsg, ok := msg.(OpenBrowserMsg)
	if !ok {
		t.Fatalf("enter should produce OpenBrowserMsg, got %T", msg)
	}
	if openMsg.URL != "https://example.com/log" {
		t.Errorf("URL = %q, want %q", openMsg.URL, "https://example.com/log")
	}
}

func TestChecks_Refresh(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(ghclient.PR{Number: 5}).SetChecks(sampleChecks())
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})
	if cmd == nil {
		t.Fatal("R should produce a cmd")
	}
	msg := cmd()
	refreshMsg, ok := msg.(RefreshMsg)
	if !ok {
		t.Fatalf("R should produce RefreshMsg, got %T", msg)
	}
	if refreshMsg.Number != 5 {
		t.Errorf("RefreshMsg.Number = %d, want 5", refreshMsg.Number)
	}
}

func TestChecks_View(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(ghclient.PR{Number: 1}).SetChecks(sampleChecks())
	view := m.View()

	if !strings.Contains(view, "#1 Checks") {
		t.Error("view should contain '#1 Checks'")
	}
	if !strings.Contains(view, "passed") {
		t.Error("view should contain 'passed'")
	}
	if !strings.Contains(view, "failed") {
		t.Error("view should contain 'failed'")
	}
	if !strings.Contains(view, "pending") {
		t.Error("view should contain 'pending'")
	}
}

func TestChecks_ViewEmpty(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(ghclient.PR{Number: 1}).SetChecks(nil)
	view := m.View()
	if !strings.Contains(view, "No checks") {
		t.Error("view should show 'No checks' for empty list")
	}
}

func TestChecks_ViewLoading(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(ghclient.PR{Number: 1})
	view := m.View()
	if !strings.Contains(view, "Loading") {
		t.Error("view should show loading state")
	}
}

func TestCheckIcon(t *testing.T) {
	tests := []struct {
		name    string
		summary ghclient.CheckSummary
		want    string // substring to find
	}{
		{"no checks", ghclient.CheckSummary{}, "—"},
		{"all pass", ghclient.CheckSummary{Total: 2, Pass: 2}, "✓"},
		{"fail", ghclient.CheckSummary{Total: 2, Pass: 1, Fail: 1}, "✗"},
		{"pending", ghclient.CheckSummary{Total: 2, Pass: 1, Pending: 1}, "◌"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckIcon(tt.summary)
			if !strings.Contains(got, tt.want) {
				t.Errorf("CheckIcon() = %q, should contain %q", got, tt.want)
			}
		})
	}
}

func TestChecks_SetError(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(ghclient.PR{Number: 1})
	m = m.SetError(fmt.Errorf("api error"))
	view := m.View()
	if !strings.Contains(view, "Error") {
		t.Error("view should show error state")
	}
	if !strings.Contains(view, "api error") {
		t.Error("view should contain error message")
	}
}

func TestChecks_AllBucketTypes(t *testing.T) {
	chks := []ghclient.Check{
		{Name: "pass-check", Bucket: "pass", Workflow: "CI",
			StartedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			CompletedAt: time.Date(2026, 1, 1, 10, 0, 30, 0, time.UTC)},
		{Name: "fail-check", Bucket: "fail", Workflow: "CI",
			StartedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			CompletedAt: time.Date(2026, 1, 1, 10, 2, 0, 0, time.UTC)},
		{Name: "skip-check", Bucket: "skipping", Workflow: "CI",
			StartedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			CompletedAt: time.Date(2026, 1, 1, 10, 0, 5, 0, time.UTC)},
		{Name: "cancel-check", Bucket: "cancel", Workflow: "CI",
			StartedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			CompletedAt: time.Date(2026, 1, 1, 10, 0, 10, 0, time.UTC)},
		{Name: "pending-started", Bucket: "pending", Workflow: "CI",
			StartedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)},
		{Name: "pending-not-started", Bucket: "pending", Workflow: "CI"},
	}
	m := New().SetSize(80, 24).SetPR(ghclient.PR{Number: 1}).SetChecks(chks)
	view := m.View()

	if !strings.Contains(view, "skipped") {
		t.Error("view should contain 'skipped'")
	}
	if !strings.Contains(view, "cancelled") {
		t.Error("view should contain 'cancelled'")
	}
	if !strings.Contains(view, "running") {
		t.Error("view should contain 'running' for pending-started")
	}
}

func TestChecks_EnterNoLink(t *testing.T) {
	chks := []ghclient.Check{
		{Name: "test", Bucket: "pass", Link: ""},
	}
	m := New().SetSize(80, 24).SetPR(ghclient.PR{Number: 1}).SetChecks(chks)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("enter with no link should not produce a cmd")
	}
}

func TestChecks_PageScroll(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(ghclient.PR{Number: 1}).SetChecks(sampleChecks())

	// ctrl+d
	m.cursor = 0
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	// Should move cursor (may be clamped to end)
	if m.cursor < 0 {
		t.Error("cursor should not be negative")
	}

	// ctrl+u from end
	m.cursor = 2
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	if m.cursor > 2 {
		t.Error("cursor should not exceed length")
	}

	// ctrl+f
	m.cursor = 0
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	if m.cursor < 0 {
		t.Error("cursor should not be negative after ctrl+f")
	}

	// ctrl+b
	m.cursor = 2
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlB})
	if m.cursor > 2 {
		t.Error("cursor should not exceed length after ctrl+b")
	}
}

func TestChecks_SortAllBuckets(t *testing.T) {
	chks := []ghclient.Check{
		{Name: "pass", Bucket: "pass"},
		{Name: "cancel", Bucket: "cancel"},
		{Name: "skip", Bucket: "skipping"},
		{Name: "pending", Bucket: "pending"},
		{Name: "fail", Bucket: "fail"},
		{Name: "unknown", Bucket: "weird"},
	}
	m := New().SetSize(80, 24).SetPR(ghclient.PR{Number: 1}).SetChecks(chks)
	expected := []string{"fail", "pending", "cancel", "pass", "skipping", "weird"}
	for i, want := range expected {
		if m.checks[i].Bucket != want {
			t.Errorf("checks[%d].Bucket = %q, want %q", i, m.checks[i].Bucket, want)
		}
	}
}

func TestChecks_SetChecksCursorClamp(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(ghclient.PR{Number: 1}).SetChecks(sampleChecks())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if m.cursor != 2 {
		t.Fatalf("cursor = %d, want 2", m.cursor)
	}
	// SetChecks with 1 item should clamp
	m = m.SetChecks([]ghclient.Check{{Name: "only", Bucket: "pass"}})
	if m.cursor != 0 {
		t.Errorf("cursor after clamp = %d, want 0", m.cursor)
	}
}

func TestCheckIcon_SkipCancel(t *testing.T) {
	// skip+cancel with no fail should show dash
	cs := ghclient.CheckSummary{Total: 2, Skip: 1, Cancel: 1}
	got := CheckIcon(cs)
	if !strings.Contains(got, "—") {
		t.Errorf("CheckIcon with only skip/cancel should show dash, got %q", got)
	}
}

func TestCheckSummaryLine_Pending(t *testing.T) {
	cs := ghclient.CheckSummary{Total: 2, Pass: 1, Pending: 1}
	got := CheckSummaryLine(cs)
	if !strings.Contains(got, "1 pending") {
		t.Errorf("should contain '1 pending', got %q", got)
	}
	if !strings.Contains(got, "1 passed") {
		t.Errorf("should contain '1 passed', got %q", got)
	}
}

func TestChecks_WorkflowSameAsName(t *testing.T) {
	chks := []ghclient.Check{
		{Name: "build", Bucket: "pass", Workflow: "build",
			StartedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			CompletedAt: time.Date(2026, 1, 1, 10, 0, 5, 0, time.UTC)},
	}
	m := New().SetSize(80, 24).SetPR(ghclient.PR{Number: 1}).SetChecks(chks)
	view := m.View()
	// When workflow == name, should not show "(build)" duplicate
	if strings.Contains(view, "(build)") {
		t.Error("should not show workflow in parens when same as name")
	}
}

func TestChecks_UnknownBucket(t *testing.T) {
	chks := []ghclient.Check{
		{Name: "unknown-check", Bucket: "weird_status", Workflow: "CI",
			StartedAt:   time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			CompletedAt: time.Date(2026, 1, 1, 10, 0, 5, 0, time.UTC)},
	}
	m := New().SetSize(80, 24).SetPR(ghclient.PR{Number: 1}).SetChecks(chks)
	view := m.View()
	// Unknown bucket should render without panic
	if !strings.Contains(view, "unknown-check") {
		t.Error("view should contain check name")
	}
}

func TestChecks_SmallHeight(t *testing.T) {
	// visibleHeight with very small height should fallback to 10
	m := New().SetSize(80, 3).SetPR(ghclient.PR{Number: 1}).SetChecks(sampleChecks())
	// Use page scroll to trigger visibleHeight() with small height
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	view := m.View()
	if !strings.Contains(view, "#1 Checks") {
		t.Error("should render even with small height")
	}
}

func TestChecks_ViewScrollWindow(t *testing.T) {
	// Create enough checks to trigger scroll window in View
	var chks []ghclient.Check
	for i := 0; i < 30; i++ {
		chks = append(chks, ghclient.Check{
			Name: fmt.Sprintf("check-%d", i), Bucket: "pass", Workflow: "CI",
			StartedAt:   time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			CompletedAt: time.Date(2026, 1, 1, 10, 0, 5, 0, time.UTC),
		})
	}
	m := New().SetSize(80, 15).SetPR(ghclient.PR{Number: 1}).SetChecks(chks)
	// Navigate to bottom to trigger scroll window
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	view := m.View()
	if !strings.Contains(view, "check-29") {
		t.Error("should show last check when scrolled to bottom")
	}
	// First checks should not be visible
	if strings.Contains(view, "check-0 ") {
		t.Error("first check should not be visible when scrolled to bottom")
	}
}

func TestChecks_LongDuration(t *testing.T) {
	// Duration >= 1 minute should show Xm XXs format
	chks := []ghclient.Check{
		{Name: "long-test", Bucket: "pass", Workflow: "CI",
			StartedAt:   time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			CompletedAt: time.Date(2026, 1, 1, 10, 3, 45, 0, time.UTC)},
	}
	m := New().SetSize(80, 24).SetPR(ghclient.PR{Number: 1}).SetChecks(chks)
	view := m.View()
	if !strings.Contains(view, "3m 45s") {
		t.Errorf("should show '3m 45s' duration, got: %s", view)
	}
}

func TestCheckSummaryLine(t *testing.T) {
	t.Run("no checks", func(t *testing.T) {
		got := CheckSummaryLine(ghclient.CheckSummary{})
		if got != "" {
			t.Errorf("should be empty for no checks, got %q", got)
		}
	})

	t.Run("with checks", func(t *testing.T) {
		cs := ghclient.CheckSummary{Total: 3, Pass: 2, Fail: 1}
		got := CheckSummaryLine(cs)
		if !strings.Contains(got, "2 passed") {
			t.Errorf("should contain '2 passed', got %q", got)
		}
		if !strings.Contains(got, "1 failed") {
			t.Errorf("should contain '1 failed', got %q", got)
		}
	})
}
