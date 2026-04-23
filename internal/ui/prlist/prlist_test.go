package prlist

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/LumaKernel/ghpr-tui/internal/ghclient"
	"github.com/LumaKernel/ghpr-tui/internal/state"
)

func testStore(t *testing.T) *state.Store {
	t.Helper()
	s, err := state.NewWithPath(t.TempDir() + "/state.json")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func samplePRs() []ghclient.PR {
	return []ghclient.PR{
		{Number: 1, Title: "First PR", Author: "alice", HeadRef: "feat-a", BaseRef: "main",
			Additions: 10, Deletions: 3, UpdatedAt: time.Now().Add(-2 * time.Hour)},
		{Number: 2, Title: "Second PR", Author: "bob", HeadRef: "feat-b", BaseRef: "main",
			IsDraft: true, Additions: 5, Deletions: 1, UpdatedAt: time.Now().Add(-30 * time.Minute)},
		{Number: 3, Title: "Third PR", Author: "carol", HeadRef: "fix", BaseRef: "main",
			Additions: 1, Deletions: 0, UpdatedAt: time.Now().Add(-5 * 24 * time.Hour)},
	}
}

func TestPRList_Navigation(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())

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

func TestPRList_SelectPR(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter should produce a cmd")
	}
	msg := cmd()
	sel, ok := msg.(SelectMsg)
	if !ok {
		t.Fatalf("enter should produce SelectMsg, got %T", msg)
	}
	if sel.PR.Number != 1 {
		t.Errorf("PR.Number = %d, want 1", sel.PR.Number)
	}

	// Should mark as read
	if !s.IsRead("test/repo", 1) {
		t.Error("enter should mark PR as read")
	}
}

func TestPRList_ToggleRead(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())

	// r toggles read
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if !s.IsRead("test/repo", 1) {
		t.Error("r should mark as read")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if s.IsRead("test/repo", 1) {
		t.Error("r again should toggle back to unread")
	}
}

func TestPRList_Refresh(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})
	if cmd == nil {
		t.Fatal("R should produce a cmd")
	}
	if _, ok := cmd().(RefreshMsg); !ok {
		t.Errorf("R should produce RefreshMsg, got %T", cmd())
	}
	if !m.loading {
		t.Error("R should set loading state")
	}
}

func TestPRList_OpenBrowser(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	if cmd == nil {
		t.Fatal("o should produce a cmd")
	}
	msg := cmd()
	openMsg, ok := msg.(OpenBrowserMsg)
	if !ok {
		t.Fatalf("o should produce OpenBrowserMsg, got %T", msg)
	}
	if openMsg.Number != 1 {
		t.Errorf("Number = %d, want 1", openMsg.Number)
	}
}

func TestPRList_OpenChecks(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if cmd == nil {
		t.Fatal("c should produce a cmd")
	}
	msg := cmd()
	checksMsg, ok := msg.(OpenChecksMsg)
	if !ok {
		t.Fatalf("c should produce OpenChecksMsg, got %T", msg)
	}
	if checksMsg.PR.Number != 1 {
		t.Errorf("PR.Number = %d, want 1", checksMsg.PR.Number)
	}
}

func TestPRList_ToggleDraft(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if cmd == nil {
		t.Fatal("d should produce a cmd")
	}
	msg := cmd()
	draftMsg, ok := msg.(ToggleDraftMsg)
	if !ok {
		t.Fatalf("d should produce ToggleDraftMsg, got %T", msg)
	}
	if draftMsg.Number != 1 {
		t.Errorf("Number = %d, want 1", draftMsg.Number)
	}
}

func TestPRList_MergeConfirmation(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())

	// m opens merge confirmation
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	if !m.confirmMerge {
		t.Fatal("m should open merge confirmation")
	}

	// l cycles method
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if m.mergeMethod != 1 {
		t.Errorf("after l: mergeMethod = %d, want 1", m.mergeMethod)
	}

	// h cycles back
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	if m.mergeMethod != 0 {
		t.Errorf("after h: mergeMethod = %d, want 0", m.mergeMethod)
	}

	// enter confirms
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter should produce merge cmd")
	}
	mergeMsg := cmd().(MergeMsg)
	if mergeMsg.Number != 1 {
		t.Errorf("Number = %d, want 1", mergeMsg.Number)
	}
	if mergeMsg.Method != "squash" {
		t.Errorf("Method = %q, want %q", mergeMsg.Method, "squash")
	}
	if mergeMsg.Undraft {
		t.Error("Undraft should be false for non-draft PR")
	}
}

func TestPRList_MergeDraftPR(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())

	// Navigate to draft PR (#2)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	mergeMsg := cmd().(MergeMsg)
	if mergeMsg.Number != 2 {
		t.Errorf("Number = %d, want 2", mergeMsg.Number)
	}
	if !mergeMsg.Undraft {
		t.Error("Undraft should be true for draft PR")
	}
}

func TestPRList_MergeCancel(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.confirmMerge {
		t.Error("esc should cancel merge confirmation")
	}
}

func TestPRList_AllowedMergeMethods(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())
	m = m.SetAllowedMergeMethods([]string{"merge"})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mergeMsg := cmd().(MergeMsg)
	if mergeMsg.Method != "merge" {
		t.Errorf("Method = %q, want %q", mergeMsg.Method, "merge")
	}
}

func TestPRList_SetMergeResult(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())
	m = m.SetMergeResult("Merged successfully")
	view := m.View()
	if !strings.Contains(view, "Merged successfully") {
		t.Error("view should contain merge result message")
	}
}

func TestPRList_ViewLoading(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24)
	view := m.View()
	if !strings.Contains(view, "Loading") {
		t.Error("view should show loading state")
	}
}

func TestPRList_ViewWithPRs(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())
	view := m.View()

	if !strings.Contains(view, "ghpr-tui") {
		t.Error("view should contain app name")
	}
	if !strings.Contains(view, "First PR") {
		t.Error("view should contain PR title")
	}
	if !strings.Contains(view, "[draft]") {
		t.Error("view should contain draft tag")
	}
	if !strings.Contains(view, "alice") {
		t.Error("view should contain author name")
	}
}

func TestPRList_ViewEmpty(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(nil)
	view := m.View()
	if !strings.Contains(view, "No open pull requests") {
		t.Error("view should show empty state")
	}
}

func TestPRList_ViewError(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24)
	m = m.SetError(fmt.Errorf("connection failed"))
	view := m.View()
	if !strings.Contains(view, "Error") {
		t.Error("view should show error")
	}
}

func TestPRList_ViewMergeConfirm(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	view := m.View()

	if !strings.Contains(view, "Merge #1?") {
		t.Error("view should show merge confirmation")
	}
	if !strings.Contains(view, "squash") {
		t.Error("view should show squash method")
	}
}

func TestPRList_ViewDraftMergeConfirm(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())
	// Navigate to draft PR
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	view := m.View()

	if !strings.Contains(view, "Undraft & Merge") {
		t.Error("view should show 'Undraft & Merge' for draft PRs")
	}
}

func TestPRList_EmptyListKeysNoOp(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(nil)

	// These should not panic
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("enter on empty list should produce nil cmd")
	}
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	if cmd != nil {
		t.Error("m on empty list should produce nil cmd")
	}
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if cmd != nil {
		t.Error("c on empty list should produce nil cmd")
	}
}

func TestPRList_SetPRsCursorClamp(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())
	// Move to last PR
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if m.cursor != 2 {
		t.Fatalf("cursor = %d, want 2", m.cursor)
	}

	// SetPRs with fewer items should clamp cursor
	m = m.SetPRs([]ghclient.PR{{Number: 1, Title: "Only"}})
	if m.cursor != 0 {
		t.Errorf("cursor after clamp = %d, want 0", m.cursor)
	}
}

func TestPRList_MergeConfirmY(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if cmd == nil {
		t.Fatal("y should confirm merge")
	}
}

func TestPRList_MergeCancelN(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.confirmMerge {
		t.Error("n should cancel merge confirmation")
	}
}

func TestPRList_MergeCancelQ(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if m.confirmMerge {
		t.Error("q should cancel merge confirmation")
	}
}

func TestPRList_MergingView(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	view := m.View()
	if !strings.Contains(view, "Merging") {
		t.Error("view should show merging state")
	}
}

func TestPRList_MergeResultClears(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())
	m = m.SetMergeResult("Done!")
	// Any key clears
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.mergeResult != "" {
		t.Error("merge result should be cleared after keypress")
	}
}

func TestPRList_PageScroll(t *testing.T) {
	s := testStore(t)
	// Use small height to trigger visibleHeight fallback (h < 1 => 10)
	m := New("test/repo", s).SetSize(80, 3).SetPRs(samplePRs())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	if m.cursor < 0 {
		t.Error("cursor should not be negative after ctrl+d")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	if m.cursor < 0 {
		t.Error("cursor should not be negative after ctrl+u")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	if m.cursor < 0 {
		t.Error("cursor should not be negative after ctrl+f")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlB})
	if m.cursor < 0 {
		t.Error("cursor should not be negative after ctrl+b")
	}
}

func TestPRList_LongTitleTruncation(t *testing.T) {
	s := testStore(t)
	prs := []ghclient.PR{
		{Number: 1, Title: strings.Repeat("A", 200), Author: "alice",
			UpdatedAt: time.Now()},
	}
	m := New("test/repo", s).SetSize(80, 24).SetPRs(prs)
	view := m.View()
	if !strings.Contains(view, "…") {
		t.Error("long title should be truncated with ellipsis")
	}
}

func TestPRList_NarrowWidthTruncation(t *testing.T) {
	s := testStore(t)
	prs := []ghclient.PR{
		{Number: 1, Title: strings.Repeat("B", 100), Author: "alice",
			UpdatedAt: time.Now()},
	}
	// Width < 68 triggers titleWidth < 20 fallback
	m := New("test/repo", s).SetSize(40, 24).SetPRs(prs)
	view := m.View()
	if !strings.Contains(view, "…") {
		t.Error("long title should be truncated with ellipsis at narrow width")
	}
}

func TestPRList_ViewScrollWindow(t *testing.T) {
	s := testStore(t)
	var prs []ghclient.PR
	for i := 1; i <= 30; i++ {
		prs = append(prs, ghclient.PR{
			Number: i, Title: fmt.Sprintf("PR %d", i), Author: "user",
			UpdatedAt: time.Now()})
	}
	m := New("test/repo", s).SetSize(80, 15).SetPRs(prs)
	// Navigate to bottom
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	view := m.View()
	if !strings.Contains(view, "PR 30") {
		t.Error("should show last PR when scrolled to bottom")
	}
}

func TestPRList_UnreadIndicator(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())
	view := m.View()
	// Unread PRs should have bullet indicator
	if !strings.Contains(view, "●") {
		t.Error("unread PRs should show bullet indicator")
	}

	// Mark as read
	s.MarkRead("test/repo", 1)
	m = m.SetPRs(samplePRs()) // re-render
	// First PR should no longer have unread indicator in its render
}

func TestPRList_CheckSummaryInRender(t *testing.T) {
	s := testStore(t)
	prs := []ghclient.PR{
		{Number: 1, Title: "PR with checks", Author: "alice",
			UpdatedAt:    time.Now(),
			CheckSummary: ghclient.CheckSummary{Total: 3, Pass: 3}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetPRs(prs)
	view := m.View()
	if !strings.Contains(view, "✓") {
		t.Error("view should show check pass icon")
	}
}

func TestRelativeTime(t *testing.T) {
	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{"just now", time.Now().Add(-10 * time.Second), "just now"},
		{"minutes", time.Now().Add(-15 * time.Minute), "15m ago"},
		{"hours", time.Now().Add(-3 * time.Hour), "3h ago"},
		{"days", time.Now().Add(-2 * 24 * time.Hour), "2d ago"},
		{"old", time.Now().Add(-30 * 24 * time.Hour), time.Now().Add(-30 * 24 * time.Hour).Format("Jan 2")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relativeTime(tt.t)
			if got != tt.want {
				t.Errorf("relativeTime() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPRList_SmallHeight(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 3).SetPRs(samplePRs())
	view := m.View()
	if !strings.Contains(view, "ghpr-tui") {
		t.Error("should render even with small height")
	}
}

func TestPRList_SetLoading(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())
	m = m.SetLoading(true)
	view := m.View()
	if !strings.Contains(view, "Loading") {
		t.Error("SetLoading(true) should show loading state")
	}
	m = m.SetLoading(false)
	if m.loading {
		t.Error("SetLoading(false) should clear loading")
	}
}

func TestPRList_MergeWhileMerging(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())
	// Start merge
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// Try to merge again while merging — should not open confirm
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	if m.confirmMerge {
		t.Error("should not open merge confirm while already merging")
	}
}

func TestPRList_JAtBottom(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(samplePRs())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	// j at bottom should stay
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != 2 {
		t.Errorf("j at bottom: cursor = %d, want 2", m.cursor)
	}
}

func TestPRList_EmptyGNoOp(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPRs(nil)
	// G on empty list should not panic
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if m.cursor != 0 {
		t.Errorf("G on empty: cursor = %d, want 0", m.cursor)
	}
}

func TestPRList_VisibleHeightFallback(t *testing.T) {
	s := testStore(t)
	prs := make([]ghclient.PR, 20)
	for i := range prs {
		prs[i] = ghclient.PR{Number: i + 1, Title: fmt.Sprintf("PR %d", i+1), Author: "u"}
	}

	// height=2 → h=2-4=-2 < 1 → visibleHeight()=10 (fallback)
	m := New("test/repo", s).SetSize(80, 2).SetPRs(prs)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	if m.cursor != 5 {
		t.Errorf("ctrl+d with fallback visibleHeight: cursor = %d, want 5", m.cursor)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	if m.cursor != 0 {
		t.Errorf("ctrl+u: cursor = %d, want 0", m.cursor)
	}

	// height=24 → h=24-4=20 → visibleHeight()=20 (normal path)
	m2 := New("test/repo", s).SetSize(80, 24).SetPRs(prs)
	m2, _ = m2.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	if m2.cursor != 10 { // 20/2 = 10
		t.Errorf("ctrl+d with normal visibleHeight: cursor = %d, want 10", m2.cursor)
	}
}
