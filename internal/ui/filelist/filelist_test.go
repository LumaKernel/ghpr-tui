package filelist

import (
	"fmt"
	"strings"
	"testing"

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

func sampleDiff() ghclient.ParsedDiff {
	return ghclient.ParsedDiff{
		Files: []ghclient.FileDiff{
			{OldPath: "a.go", NewPath: "a.go", Hunks: []ghclient.Hunk{
				{Header: "@@ -1,1 +1,2 @@", Lines: []ghclient.DiffLine{
					{Type: ghclient.LineAdded, Content: "new", NewNum: 1},
				}},
			}},
			{OldPath: "b.go", NewPath: "b.go"},
			{OldPath: "c.go", NewPath: "c.go"},
		},
	}
}

func samplePR() ghclient.PR {
	return ghclient.PR{Number: 42, Title: "Test PR", Author: "user", HeadRef: "feat", BaseRef: "main"}
}

func TestFileList_Navigation(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPR(samplePR()).SetDiff(sampleDiff())

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

func TestFileList_SelectFile(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPR(samplePR()).SetDiff(sampleDiff())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter should produce a cmd")
	}
	msg := cmd()
	sel, ok := msg.(SelectFileMsg)
	if !ok {
		t.Fatalf("enter should produce SelectFileMsg, got %T", msg)
	}
	if sel.Index != 0 {
		t.Errorf("Index = %d, want 0", sel.Index)
	}
}

func TestFileList_BackMsg(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPR(samplePR()).SetDiff(sampleDiff())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc should produce a cmd")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Errorf("esc should produce BackMsg, got %T", cmd())
	}
}

func TestFileList_ToggleReviewed(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPR(samplePR()).SetDiff(sampleDiff())

	// space toggles reviewed
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	if !s.IsFileReviewed("test/repo", 42, "a.go") {
		t.Error("space should mark file as reviewed")
	}

	// space again toggles back
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	if s.IsFileReviewed("test/repo", 42, "a.go") {
		t.Error("space again should unmark file")
	}
}

func TestFileList_MarkAllReviewed(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPR(samplePR()).SetDiff(sampleDiff())

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if s.ReviewedFileCount("test/repo", 42) != 3 {
		t.Errorf("a should mark all %d files as reviewed, got %d", 3, s.ReviewedFileCount("test/repo", 42))
	}
}

func TestFileList_OpenChecks(t *testing.T) {
	s := testStore(t)
	pr := samplePR()
	m := New("test/repo", s).SetSize(80, 24).SetPR(pr).SetDiff(sampleDiff())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if cmd == nil {
		t.Fatal("c should produce a cmd")
	}
	msg := cmd()
	checksMsg, ok := msg.(OpenChecksMsg)
	if !ok {
		t.Fatalf("c should produce OpenChecksMsg, got %T", msg)
	}
	if checksMsg.PR.Number != 42 {
		t.Errorf("PR.Number = %d, want 42", checksMsg.PR.Number)
	}
}

func TestFileList_MergeConfirmation(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPR(samplePR()).SetDiff(sampleDiff())

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
	msg := cmd()
	mergeMsg, ok := msg.(MergeMsg)
	if !ok {
		t.Fatalf("enter should produce MergeMsg, got %T", msg)
	}
	if mergeMsg.Number != 42 {
		t.Errorf("Number = %d, want 42", mergeMsg.Number)
	}
	if mergeMsg.Method != "squash" {
		t.Errorf("Method = %q, want %q", mergeMsg.Method, "squash")
	}
}

func TestFileList_MergeCancel(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPR(samplePR()).SetDiff(sampleDiff())

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.confirmMerge {
		t.Error("esc should cancel merge confirmation")
	}
}

func TestFileList_AllowedMergeMethods(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPR(samplePR()).SetDiff(sampleDiff())
	m = m.SetAllowedMergeMethods([]string{"rebase"})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	mergeMsg := msg.(MergeMsg)
	if mergeMsg.Method != "rebase" {
		t.Errorf("Method = %q, want %q", mergeMsg.Method, "rebase")
	}
}

func TestFileList_ViewLoading(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPR(samplePR())
	view := m.View()
	if !strings.Contains(view, "Loading") {
		t.Error("view should show loading state")
	}
}

func TestFileList_ViewWithDiff(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPR(samplePR()).SetDiff(sampleDiff())
	view := m.View()

	if !strings.Contains(view, "#42") {
		t.Error("view should contain PR number")
	}
	if !strings.Contains(view, "Test PR") {
		t.Error("view should contain PR title")
	}
	if !strings.Contains(view, "a.go") {
		t.Error("view should contain file name")
	}
	if !strings.Contains(view, "0/3 reviewed") {
		t.Error("view should contain review progress")
	}
}

func TestFileList_ViewError(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPR(samplePR())
	m = m.SetError(fmt.Errorf("test error"))
	view := m.View()
	if !strings.Contains(view, "Error") {
		t.Error("view should show error")
	}
}

func TestFileList_Accessors(t *testing.T) {
	s := testStore(t)
	pr := samplePR()
	diff := sampleDiff()
	m := New("test/repo", s).SetSize(80, 24).SetPR(pr).SetDiff(diff)

	if m.FileCount() != 3 {
		t.Errorf("FileCount() = %d, want 3", m.FileCount())
	}
	if m.PR().Number != 42 {
		t.Errorf("PR().Number = %d, want 42", m.PR().Number)
	}
	if len(m.Diff().Files) != 3 {
		t.Errorf("Diff().Files = %d, want 3", len(m.Diff().Files))
	}
}

func TestFileList_SetMergeResult(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPR(samplePR()).SetDiff(sampleDiff())
	m = m.SetMergeResult("Merged!")
	view := m.View()
	if !strings.Contains(view, "Merged!") {
		t.Error("view should show merge result")
	}
	// Any key should clear merge result
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	view = m.View()
	if strings.Contains(view, "Merged!") {
		t.Error("merge result should be cleared after keypress")
	}
}

func TestFileList_MergingView(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPR(samplePR()).SetDiff(sampleDiff())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// After confirm, should be merging
	view := m.View()
	if !strings.Contains(view, "Merging") {
		t.Error("view should show 'Merging...' state")
	}
}

func TestFileList_RenderFileStatuses(t *testing.T) {
	s := testStore(t)
	diff := ghclient.ParsedDiff{
		Files: []ghclient.FileDiff{
			{NewPath: "new.go", IsNew: true, Hunks: []ghclient.Hunk{{Lines: []ghclient.DiffLine{{Type: ghclient.LineAdded}}}}},
			{OldPath: "old.go", IsDelete: true, Hunks: []ghclient.Hunk{{Lines: []ghclient.DiffLine{{Type: ghclient.LineRemoved}}}}},
			{OldPath: "a.go", NewPath: "b.go", IsRename: true},
			{OldPath: "img.png", NewPath: "img.png", IsBinary: true},
		},
	}
	m := New("test/repo", s).SetSize(80, 24).SetPR(samplePR()).SetDiff(diff)
	view := m.View()
	if !strings.Contains(view, "[new]") {
		t.Error("should show [new] tag")
	}
	if !strings.Contains(view, "[del]") {
		t.Error("should show [del] tag")
	}
	if !strings.Contains(view, "[ren]") {
		t.Error("should show [ren] tag")
	}
	if !strings.Contains(view, "[bin]") {
		t.Error("should show [bin] tag")
	}
}

func TestFileList_PageScroll(t *testing.T) {
	s := testStore(t)
	// Use small height to trigger visibleHeight fallback
	m := New("test/repo", s).SetSize(80, 5).SetPR(samplePR()).SetDiff(sampleDiff())

	// ctrl+d
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	if m.cursor < 0 {
		t.Error("cursor should not be negative")
	}

	// ctrl+u
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	if m.cursor < 0 {
		t.Error("cursor should not be negative")
	}
}

func TestFileList_MergeConfirmY(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPR(samplePR()).SetDiff(sampleDiff())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if cmd == nil {
		t.Fatal("y should confirm merge")
	}
	mergeMsg := cmd().(MergeMsg)
	if mergeMsg.Number != 42 {
		t.Errorf("Number = %d, want 42", mergeMsg.Number)
	}
}

func TestFileList_MergeCancelN(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPR(samplePR()).SetDiff(sampleDiff())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.confirmMerge {
		t.Error("n should cancel merge")
	}
}

func TestFileList_DraftMergeLabel(t *testing.T) {
	s := testStore(t)
	pr := samplePR()
	pr.IsDraft = true
	m := New("test/repo", s).SetSize(80, 24).SetPR(pr).SetDiff(sampleDiff())

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	view := m.View()
	if !strings.Contains(view, "Undraft") {
		t.Error("view should show 'Undraft & Merge' for draft PRs")
	}

	// Confirm sends undraft=true
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mergeMsg := cmd().(MergeMsg)
	if !mergeMsg.Undraft {
		t.Error("Undraft should be true for draft PRs")
	}
}

func TestFileList_LongPathTruncation(t *testing.T) {
	s := testStore(t)
	longPath := strings.Repeat("a/", 50) + "file.go"
	diff := ghclient.ParsedDiff{
		Files: []ghclient.FileDiff{
			{OldPath: longPath, NewPath: longPath},
		},
	}
	m := New("test/repo", s).SetSize(60, 24).SetPR(samplePR()).SetDiff(diff)
	view := m.View()
	if !strings.Contains(view, "…") {
		t.Error("long path should be truncated with ellipsis")
	}
}

func TestFileList_LongPathNarrowWidth(t *testing.T) {
	s := testStore(t)
	longPath := strings.Repeat("x", 50) + ".go"
	diff := ghclient.ParsedDiff{
		Files: []ghclient.FileDiff{
			{OldPath: longPath, NewPath: longPath},
		},
	}
	// Width < 50 triggers pathWidth < 20 fallback
	m := New("test/repo", s).SetSize(40, 24).SetPR(samplePR()).SetDiff(diff)
	view := m.View()
	if !strings.Contains(view, "…") {
		t.Error("long path should be truncated with ellipsis even at narrow width")
	}
}

func TestFileList_SmallHeight(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 3).SetPR(samplePR()).SetDiff(sampleDiff())
	view := m.View()
	if !strings.Contains(view, "#42") {
		t.Error("should render even with small height")
	}
}

func TestFileList_FullPageScroll(t *testing.T) {
	s := testStore(t)
	// Use small height to trigger visibleHeight fallback (h < 1 => 10)
	m := New("test/repo", s).SetSize(80, 5).SetPR(samplePR()).SetDiff(sampleDiff())

	// ctrl+f
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	if m.cursor < 0 {
		t.Error("cursor should not be negative after ctrl+f")
	}

	// ctrl+b
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlB})
	if m.cursor < 0 {
		t.Error("cursor should not be negative after ctrl+b")
	}
}

func TestFileList_MergeCancelQ(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPR(samplePR()).SetDiff(sampleDiff())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if m.confirmMerge {
		t.Error("q should cancel merge")
	}
}

func TestFileList_BackspaceBack(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetPR(samplePR()).SetDiff(sampleDiff())
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if cmd == nil {
		t.Fatal("backspace should produce a cmd")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Errorf("backspace should produce BackMsg, got %T", cmd())
	}
}

func TestFileList_CheckSummaryInHeader(t *testing.T) {
	s := testStore(t)
	pr := samplePR()
	pr.CheckSummary = ghclient.CheckSummary{Total: 2, Pass: 1, Fail: 1}
	m := New("test/repo", s).SetSize(80, 24).SetPR(pr).SetDiff(sampleDiff())
	view := m.View()
	if !strings.Contains(view, "1 passed") {
		t.Error("view should show check summary")
	}
}

func TestFileList_ReviewedFile(t *testing.T) {
	s := testStore(t)
	s.MarkFileReviewed("test/repo", 42, "a.go")
	m := New("test/repo", s).SetSize(80, 24).SetPR(samplePR()).SetDiff(sampleDiff())
	view := m.View()
	if !strings.Contains(view, "✓") {
		t.Error("view should show review checkmark")
	}
	if !strings.Contains(view, "1/3 reviewed") {
		t.Error("view should show 1/3 reviewed")
	}
}

func TestFileList_ViewScrollWindow(t *testing.T) {
	s := testStore(t)
	// Create many files so cursor can go beyond viewport
	files := make([]ghclient.FileDiff, 30)
	for i := range files {
		name := fmt.Sprintf("file%d.go", i)
		files[i] = ghclient.FileDiff{OldPath: name, NewPath: name}
	}
	diff := ghclient.ParsedDiff{Files: files}

	// height=15 → visibleHeight = 15-7 = 8
	m := New("test/repo", s).SetSize(80, 15).SetPR(samplePR()).SetDiff(diff)
	// Move cursor beyond visible area to trigger startIdx = m.cursor - visibleHeight + 1
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if m.cursor != 29 {
		t.Fatalf("cursor = %d, want 29", m.cursor)
	}
	view := m.View()
	// Last file should be visible
	if !strings.Contains(view, "file29.go") {
		t.Error("should show last file when scrolled to bottom")
	}
	// First files should not be visible (startIdx should be > 0)
	if strings.Contains(view, "file0.go") {
		t.Error("first file should not be visible when scrolled to bottom")
	}
}

func TestFileList_VisibleHeightFallback(t *testing.T) {
	s := testStore(t)
	// Create diff with many files so scroll keys have effect
	files := make([]ghclient.FileDiff, 20)
	for i := range files {
		name := fmt.Sprintf("file%d.go", i)
		files[i] = ghclient.FileDiff{OldPath: name, NewPath: name}
	}
	diff := ghclient.ParsedDiff{Files: files}

	// height=3 → h=3-7=-4 < 1 → visibleHeight()=10 (fallback)
	m := New("test/repo", s).SetSize(80, 3).SetPR(samplePR()).SetDiff(diff)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	if m.cursor != 5 {
		t.Errorf("ctrl+d with fallback visibleHeight: cursor = %d, want 5", m.cursor)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	if m.cursor != 0 {
		t.Errorf("ctrl+u: cursor = %d, want 0", m.cursor)
	}

	// height=24 → h=24-7=17 → visibleHeight()=17 (normal path)
	m2 := New("test/repo", s).SetSize(80, 24).SetPR(samplePR()).SetDiff(diff)
	m2, _ = m2.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	if m2.cursor != 8 { // 17/2 = 8
		t.Errorf("ctrl+d with normal visibleHeight: cursor = %d, want 8", m2.cursor)
	}
}
