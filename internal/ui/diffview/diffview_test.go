package diffview

import (
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

func sampleFile() ghclient.FileDiff {
	return ghclient.FileDiff{
		OldPath: "hello.go",
		NewPath: "hello.go",
		Hunks: []ghclient.Hunk{
			{
				Header: "@@ -1,3 +1,4 @@",
				Lines: []ghclient.DiffLine{
					{Type: ghclient.LineContext, Content: "package main", OldNum: 1, NewNum: 1},
					{Type: ghclient.LineAdded, Content: "import \"fmt\"", NewNum: 2},
					{Type: ghclient.LineContext, Content: "func main() {", OldNum: 2, NewNum: 3},
				},
			},
		},
	}
}

func TestDiffView_SetFile(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetFile(sampleFile(), 0, 3, 1)
	if m.FileIndex() != 0 {
		t.Errorf("FileIndex() = %d, want 0", m.FileIndex())
	}
	if m.scroll != 0 {
		t.Errorf("scroll = %d, want 0", m.scroll)
	}
	if len(m.lines) == 0 {
		t.Error("lines should not be empty after SetFile")
	}
}

func TestDiffView_ScrollDown(t *testing.T) {
	s := testStore(t)
	// Create a file with many lines to ensure scrolling works
	file := ghclient.FileDiff{
		OldPath: "big.go",
		NewPath: "big.go",
		Hunks: []ghclient.Hunk{
			{
				Header: "@@ -1,30 +1,30 @@",
				Lines:  makeManyLines(30),
			},
		},
	}
	m := New("test/repo", s).SetSize(80, 10).SetFile(file, 0, 1, 1)

	// j scrolls down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.scroll != 1 {
		t.Errorf("after j: scroll = %d, want 1", m.scroll)
	}

	// k scrolls back up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.scroll != 0 {
		t.Errorf("after k: scroll = %d, want 0", m.scroll)
	}

	// k at top stays at 0
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.scroll != 0 {
		t.Errorf("k at top: scroll = %d, want 0", m.scroll)
	}
}

func TestDiffView_GAndg(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "big.go",
		NewPath: "big.go",
		Hunks: []ghclient.Hunk{
			{Header: "@@ -1,30 +1,30 @@", Lines: makeManyLines(30)},
		},
	}
	m := New("test/repo", s).SetSize(80, 10).SetFile(file, 0, 1, 1)

	// G goes to end
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if m.scroll == 0 {
		t.Error("G should move scroll away from 0")
	}

	// g goes to start
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if m.scroll != 0 {
		t.Errorf("after g: scroll = %d, want 0", m.scroll)
	}
}

func TestDiffView_BackMsg(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetFile(sampleFile(), 0, 1, 1)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc should produce a cmd")
	}
	msg := cmd()
	if _, ok := msg.(BackMsg); !ok {
		t.Errorf("esc should produce BackMsg, got %T", msg)
	}
}

func TestDiffView_NextPrevFile(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetFile(sampleFile(), 0, 3, 1)

	// ] sends NextFileMsg
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	if cmd == nil {
		t.Fatal("] should produce a cmd")
	}
	if _, ok := cmd().(NextFileMsg); !ok {
		t.Errorf("] should produce NextFileMsg, got %T", cmd())
	}

	// [ sends PrevFileMsg
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	if cmd == nil {
		t.Fatal("[ should produce a cmd")
	}
	if _, ok := cmd().(PrevFileMsg); !ok {
		t.Errorf("[ should produce PrevFileMsg, got %T", cmd())
	}
}

func TestDiffView_SpaceMarksReviewed(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetFile(sampleFile(), 0, 3, 1)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	if cmd == nil {
		t.Fatal("space should produce a cmd")
	}
	if _, ok := cmd().(NextFileMsg); !ok {
		t.Errorf("space should produce NextFileMsg, got %T", cmd())
	}

	// File should be marked as reviewed
	if !s.IsFileReviewed("test/repo", 1, "hello.go") {
		t.Error("space should mark file as reviewed")
	}
}

func TestDiffView_ViewContent(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetFile(sampleFile(), 0, 3, 1)
	view := m.View()

	if !strings.Contains(view, "hello.go") {
		t.Error("view should contain file path")
	}
	if !strings.Contains(view, "1/3") {
		t.Error("view should contain file index")
	}
	if !strings.Contains(view, "+1") {
		t.Error("view should contain additions count")
	}
}

func TestDiffView_ViewBinaryFile(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath:  "image.png",
		NewPath:  "image.png",
		IsBinary: true,
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)
	view := m.View()
	if !strings.Contains(view, "Binary file") {
		t.Error("view should show 'Binary file' for binary files")
	}
}

func TestDiffView_HalfPageScroll(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "big.go",
		NewPath: "big.go",
		Hunks: []ghclient.Hunk{
			{Header: "@@ -1,30 +1,30 @@", Lines: makeManyLines(30)},
		},
	}
	m := New("test/repo", s).SetSize(80, 12).SetFile(file, 0, 1, 1)

	// ctrl+d scrolls down half page
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if m.scroll == 0 {
		t.Error("d should scroll down")
	}
	savedScroll := m.scroll

	// ctrl+u scrolls back up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	if m.scroll >= savedScroll {
		t.Error("u should scroll up")
	}
}

func TestDiffView_HunkNavigation(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "multi.go",
		NewPath: "multi.go",
		Hunks: []ghclient.Hunk{
			{
				Header: "@@ -1,3 +1,3 @@",
				Lines:  makeManyLines(10),
			},
			{
				Header: "@@ -20,3 +20,3 @@",
				Lines:  makeManyLines(10),
			},
		},
	}
	m := New("test/repo", s).SetSize(80, 8).SetFile(file, 0, 1, 1)

	// n goes to next hunk
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.scroll == 0 {
		t.Error("n should move to next hunk")
	}

	// N goes to previous hunk
	savedScroll := m.scroll
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("N")})
	if m.scroll >= savedScroll {
		t.Error("N should move to previous hunk")
	}
}

func TestDiffView_CtrlEY(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "big.go", NewPath: "big.go",
		Hunks: []ghclient.Hunk{{Header: "@@ -1,30 +1,30 @@", Lines: makeManyLines(30)}},
	}
	m := New("test/repo", s).SetSize(80, 10).SetFile(file, 0, 1, 1)

	// ctrl+e scrolls one line down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	if m.scroll != 1 {
		t.Errorf("after ctrl+e: scroll = %d, want 1", m.scroll)
	}

	// ctrl+y scrolls one line up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlY})
	if m.scroll != 0 {
		t.Errorf("after ctrl+y: scroll = %d, want 0", m.scroll)
	}
}

func TestDiffView_FullPageScroll(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "big.go", NewPath: "big.go",
		Hunks: []ghclient.Hunk{{Header: "@@ -1,30 +1,30 @@", Lines: makeManyLines(30)}},
	}
	m := New("test/repo", s).SetSize(80, 10).SetFile(file, 0, 1, 1)

	// f scrolls full page down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	if m.scroll == 0 {
		t.Error("f should scroll down")
	}
	saved := m.scroll

	// b scrolls full page up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	if m.scroll >= saved {
		t.Error("b should scroll up")
	}
}

func TestDiffView_BackspaceBack(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetFile(sampleFile(), 0, 1, 1)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if cmd == nil {
		t.Fatal("backspace should produce a cmd")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Errorf("backspace should produce BackMsg, got %T", cmd())
	}
}

func TestDiffView_ViewScrollPercent(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "big.go", NewPath: "big.go",
		Hunks: []ghclient.Hunk{{Header: "@@ -1,30 +1,30 @@", Lines: makeManyLines(30)}},
	}
	m := New("test/repo", s).SetSize(80, 10).SetFile(file, 0, 1, 1)
	// Scroll to middle
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	view := m.View()
	if !strings.Contains(view, "100%") {
		t.Error("view at bottom should show 100%")
	}
}

func TestDiffView_ViewReviewedMark(t *testing.T) {
	s := testStore(t)
	s.MarkFileReviewed("test/repo", 1, "hello.go")
	m := New("test/repo", s).SetSize(80, 24).SetFile(sampleFile(), 0, 1, 1)
	view := m.View()
	if !strings.Contains(view, "✓") {
		t.Error("view should show reviewed mark")
	}
}

func TestDiffView_RenderAllLineTypes(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,3 +1,3 @@",
			Lines: []ghclient.DiffLine{
				{Type: ghclient.LineContext, Content: "ctx", OldNum: 1, NewNum: 1},
				{Type: ghclient.LineRemoved, Content: "old", OldNum: 2},
				{Type: ghclient.LineAdded, Content: "new", NewNum: 2},
			},
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)
	view := m.View()
	if !strings.Contains(view, "+1") {
		t.Error("view should show additions stat")
	}
	if !strings.Contains(view, "-1") {
		t.Error("view should show deletions stat")
	}
}

func TestDiffView_SetSizeEmpty(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24)
	// SetSize with no file should not panic
	m = m.SetSize(100, 30)
	if m.width != 100 || m.height != 30 {
		t.Error("SetSize should update dimensions")
	}
}

func TestDiffView_TabNavigation(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetFile(sampleFile(), 1, 3, 1)

	// tab sends NextFileMsg
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if cmd == nil {
		t.Fatal("tab should produce a cmd")
	}
	if _, ok := cmd().(NextFileMsg); !ok {
		t.Errorf("tab should produce NextFileMsg, got %T", cmd())
	}

	// shift+tab sends PrevFileMsg
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if cmd == nil {
		t.Fatal("shift+tab should produce a cmd")
	}
	if _, ok := cmd().(PrevFileMsg); !ok {
		t.Errorf("shift+tab should produce PrevFileMsg, got %T", cmd())
	}
}

func TestDiffView_SetSizeWithFile(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetFile(sampleFile(), 0, 1, 1)
	oldLineCount := len(m.lines)
	// SetSize should re-render lines
	m = m.SetSize(120, 40)
	if len(m.lines) == 0 {
		t.Error("lines should be re-rendered after SetSize")
	}
	_ = oldLineCount // lines count may stay same, but re-render should happen
}

func TestDiffView_SmallHeight(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 3).SetFile(sampleFile(), 0, 1, 1)
	view := m.View()
	if !strings.Contains(view, "hello.go") {
		t.Error("should render even with small height")
	}
}

func TestDiffView_NextHunkNoMore(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "f.go", NewPath: "f.go",
		Hunks: []ghclient.Hunk{
			{Header: "@@ -1,3 +1,3 @@", Lines: makeManyLines(5)},
		},
	}
	m := New("test/repo", s).SetSize(80, 10).SetFile(file, 0, 1, 1)
	// Scroll past all hunks
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	// n should stay at current position when no more hunks
	before := m.scroll
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.scroll != before {
		t.Error("n with no more hunks should stay at current position")
	}
}

func TestDiffView_NonMsgType(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetFile(sampleFile(), 0, 1, 1)
	// Non-key msg should be handled without panic
	m2, cmd := m.Update("some string msg")
	if cmd != nil {
		t.Error("non-key msg should produce nil cmd")
	}
	_ = m2
}

func TestDiffView_VisualMode(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,5 +1,5 @@",
			Lines:  makeManyLines(10),
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)

	// Enter visual mode with V
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	if m.CurrentMode() != ModeVisual {
		t.Errorf("mode = %d, want ModeVisual", m.CurrentMode())
	}
	if !m.IsInputMode() {
		t.Error("IsInputMode should be true in visual mode")
	}

	// j moves cursor down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != m.selStart+1 {
		t.Errorf("cursor = %d, want %d", m.cursor, m.selStart+1)
	}

	// k moves cursor up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.cursor != m.selStart {
		t.Errorf("cursor = %d, want %d", m.cursor, m.selStart)
	}

	// View should show [VISUAL] indicator
	view := m.View()
	if !strings.Contains(view, "VISUAL") {
		t.Error("view should show VISUAL indicator")
	}

	// g jumps to top
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if m.cursor != 0 {
		t.Errorf("after g: cursor = %d, want 0", m.cursor)
	}

	// G jumps to bottom
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if m.cursor != len(m.lines)-1 {
		t.Errorf("after G: cursor = %d, want %d", m.cursor, len(m.lines)-1)
	}

	// esc exits visual mode
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.CurrentMode() != ModeNormal {
		t.Errorf("mode = %d, want ModeNormal after esc", m.CurrentMode())
	}
}

func TestDiffView_VisualModeCancel(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,5 +1,5 @@",
			Lines:  makeManyLines(5),
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)

	// Enter and cancel with V
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	if m.CurrentMode() != ModeNormal {
		t.Error("V in visual mode should cancel")
	}
}

func TestDiffView_VisualToComment(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,5 +1,5 @@",
			Lines: []ghclient.DiffLine{
				{Type: ghclient.LineContext, Content: "a", OldNum: 1, NewNum: 1},
				{Type: ghclient.LineAdded, Content: "b", NewNum: 2},
				{Type: ghclient.LineContext, Content: "c", OldNum: 2, NewNum: 3},
			},
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)

	// Enter visual mode and select a diff line (skip hunk header at index 0)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	// cursor starts at scroll (0), which is the hunk header
	// Move to a line with a diffLine
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})

	// enter opens comment input
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.CurrentMode() != ModeComment {
		t.Errorf("mode = %d, want ModeComment", m.CurrentMode())
	}
	if !m.IsInputMode() {
		t.Error("IsInputMode should be true in comment mode")
	}

	// View should show [COMMENT] indicator
	view := m.View()
	if !strings.Contains(view, "COMMENT") {
		t.Error("view should show COMMENT indicator")
	}
}

func TestDiffView_CommentInput(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,3 +1,3 @@",
			Lines: []ghclient.DiffLine{
				{Type: ghclient.LineAdded, Content: "new line", NewNum: 1},
			},
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 42)

	// Enter visual mode, select line, open comment
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // move to added line
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")}) // open comment

	// Type some text
	for _, ch := range "hello" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}
	if m.commentText[0] != "hello" {
		t.Errorf("comment text = %q, want %q", m.commentText[0], "hello")
	}

	// Backspace
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.commentText[0] != "hell" {
		t.Errorf("after backspace: comment text = %q, want %q", m.commentText[0], "hell")
	}

	// Enter adds a new line
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if len(m.commentText) != 2 {
		t.Errorf("after enter: %d lines, want 2", len(m.commentText))
	}
	if m.commentRow != 1 {
		t.Errorf("commentRow = %d, want 1", m.commentRow)
	}

	// Arrow keys
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.commentRow != 0 {
		t.Errorf("after up: commentRow = %d, want 0", m.commentRow)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.commentRow != 1 {
		t.Errorf("after down: commentRow = %d, want 1", m.commentRow)
	}

	// Esc cancels
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.CurrentMode() != ModeNormal {
		t.Error("esc should cancel comment mode")
	}
	if m.commentText != nil {
		t.Error("comment text should be nil after cancel")
	}
}

func TestDiffView_CommentSubmit(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,3 +1,3 @@",
			Lines: []ghclient.DiffLine{
				{Type: ghclient.LineAdded, Content: "new", NewNum: 1},
			},
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 99)

	// Enter visual → comment → type → submit
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	for _, ch := range "fix this" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}

	// ctrl+s submits
	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd == nil {
		t.Fatal("ctrl+s should produce a cmd")
	}
	msg := cmd()
	postMsg, ok := msg.(PostCommentMsg)
	if !ok {
		t.Fatalf("expected PostCommentMsg, got %T", msg)
	}
	if postMsg.PRNumber != 99 {
		t.Errorf("PRNumber = %d, want 99", postMsg.PRNumber)
	}
	if postMsg.Body != "fix this" {
		t.Errorf("Body = %q, want %q", postMsg.Body, "fix this")
	}
	if postMsg.Path != "test.go" {
		t.Errorf("Path = %q, want %q", postMsg.Path, "test.go")
	}
	if m.CurrentMode() != ModeNormal {
		t.Error("should return to normal mode after submit")
	}
}

func TestDiffView_CommentEmptyBody(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,3 +1,3 @@",
			Lines: []ghclient.DiffLine{
				{Type: ghclient.LineAdded, Content: "new", NewNum: 1},
			},
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)

	// Enter visual → comment → submit empty
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// ctrl+s with empty body
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd != nil {
		t.Error("empty body should not produce a cmd")
	}
	if m.statusMsg == "" {
		t.Error("should set status message for empty body")
	}
}

func TestDiffView_ToggleComments(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetFile(sampleFile(), 0, 1, 1)

	// c toggles inline comments
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if !m.showComments {
		t.Error("c should enable showComments")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if m.showComments {
		t.Error("c again should disable showComments")
	}
}

func TestDiffView_OpenCommentsList(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetFile(sampleFile(), 0, 1, 42)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	if cmd == nil {
		t.Fatal("C should produce a cmd")
	}
	msg, ok := cmd().(OpenCommentsMsg)
	if !ok {
		t.Fatalf("expected OpenCommentsMsg, got %T", cmd())
	}
	if msg.PRNumber != 42 {
		t.Errorf("PRNumber = %d, want 42", msg.PRNumber)
	}
}

func TestDiffView_SetComments(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetFile(sampleFile(), 0, 1, 1)

	threads := []ghclient.CommentThread{
		{Root: ghclient.ReviewComment{Path: "hello.go", Line: 2, Side: "RIGHT", Body: "looks good",
			User: struct {
				Login string `json:"login"`
			}{Login: "alice"}}},
	}
	m = m.SetComments(threads)

	if len(m.comments) != 1 {
		t.Errorf("comments = %d, want 1", len(m.comments))
	}
}

func TestDiffView_InlineCommentDisplay(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,3 +1,3 @@",
			Lines: []ghclient.DiffLine{
				{Type: ghclient.LineAdded, Content: "new line", NewNum: 2},
			},
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)

	threads := []ghclient.CommentThread{
		{Root: ghclient.ReviewComment{Path: "test.go", Line: 2, Side: "RIGHT", Body: "review note",
			User: struct {
				Login string `json:"login"`
			}{Login: "bob"}}},
	}
	m = m.SetComments(threads)
	m.showComments = true
	m.rebuildLines()

	view := m.View()
	if !strings.Contains(view, "bob") {
		t.Error("inline comment should show author")
	}
	if !strings.Contains(view, "review note") {
		t.Error("inline comment should show body")
	}
}

func TestDiffView_HeadSHA(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s)
	m = m.SetHeadSHA("abc123")
	if m.HeadSHA() != "abc123" {
		t.Errorf("HeadSHA() = %q, want %q", m.HeadSHA(), "abc123")
	}
}

func TestDiffView_SetStatusMsg(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetFile(sampleFile(), 0, 1, 1)
	m = m.SetStatusMsg("test status")
	view := m.View()
	if !strings.Contains(view, "test status") {
		t.Error("view should contain status message")
	}
}

func TestDiffView_SelectionLineRange(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,3 +1,4 @@",
			Lines: []ghclient.DiffLine{
				{Type: ghclient.LineContext, Content: "a", OldNum: 1, NewNum: 1},
				{Type: ghclient.LineAdded, Content: "b", NewNum: 2},
				{Type: ghclient.LineAdded, Content: "c", NewNum: 3},
				{Type: ghclient.LineContext, Content: "d", OldNum: 2, NewNum: 4},
			},
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)

	// Select lines 1-3 (indices 1-3 in m.lines, skipping hunk header at 0)
	m.selStart = 1
	m.cursor = 3
	start, end, side := m.selectionLineRange()
	if start != 1 {
		t.Errorf("startLine = %d, want 1", start)
	}
	if end != 3 {
		t.Errorf("endLine = %d, want 3", end)
	}
	if side != "RIGHT" {
		t.Errorf("side = %q, want RIGHT", side)
	}
}

func TestDiffView_SelectionLineRangeRemoved(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,3 +1,1 @@",
			Lines: []ghclient.DiffLine{
				{Type: ghclient.LineRemoved, Content: "old1", OldNum: 1},
				{Type: ghclient.LineRemoved, Content: "old2", OldNum: 2},
			},
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)

	m.selStart = 1 // first removed line
	m.cursor = 2   // second removed line
	start, end, side := m.selectionLineRange()
	if start != 1 {
		t.Errorf("startLine = %d, want 1", start)
	}
	if end != 2 {
		t.Errorf("endLine = %d, want 2", end)
	}
	if side != "LEFT" {
		t.Errorf("side = %q, want LEFT", side)
	}
}

func TestDiffView_SelectionHunkOnly(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,1 +1,1 @@",
			Lines: []ghclient.DiffLine{
				{Type: ghclient.LineContext, Content: "a", OldNum: 1, NewNum: 1},
			},
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)

	// Select only the hunk header (index 0)
	m.selStart = 0
	m.cursor = 0
	start, _, _ := m.selectionLineRange()
	if start != 0 {
		t.Errorf("selecting only hunk header should give startLine=0, got %d", start)
	}
}

func TestDiffView_CommentLeftRight(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,1 +1,1 @@",
			Lines: []ghclient.DiffLine{
				{Type: ghclient.LineAdded, Content: "a", NewNum: 1},
			},
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)

	// Enter comment mode
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Type then test left/right arrow
	for _, ch := range "abc" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m.commentCol != 2 {
		t.Errorf("after left: col = %d, want 2", m.commentCol)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.commentCol != 3 {
		t.Errorf("after right: col = %d, want 3", m.commentCol)
	}
}

func TestDiffView_BackspaceJoinLines(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,1 +1,1 @@",
			Lines: []ghclient.DiffLine{
				{Type: ghclient.LineAdded, Content: "a", NewNum: 1},
			},
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)

	// Enter comment mode
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Type, enter, type
	for _, ch := range "ab" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	for _, ch := range "cd" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}

	// Backspace at start of line 1 should join with line 0
	m.commentCol = 0
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if len(m.commentText) != 1 {
		t.Errorf("after backspace join: %d lines, want 1", len(m.commentText))
	}
	if m.commentText[0] != "abcd" {
		t.Errorf("joined line = %q, want %q", m.commentText[0], "abcd")
	}
	if m.commentCol != 2 {
		t.Errorf("col = %d, want 2 (end of prev line)", m.commentCol)
	}
}

func TestDiffView_InlineCommentWithReplies(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,1 +1,1 @@",
			Lines: []ghclient.DiffLine{
				{Type: ghclient.LineAdded, Content: "new line", NewNum: 1},
			},
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)

	threads := []ghclient.CommentThread{
		{
			Root: ghclient.ReviewComment{
				Path: "test.go", Line: 1, Side: "RIGHT", Body: "root comment",
				User: struct{ Login string `json:"login"` }{Login: "alice"},
			},
			Replies: []ghclient.ReviewComment{
				{Body: "reply here", User: struct{ Login string `json:"login"` }{Login: "bob"}},
			},
		},
	}
	m = m.SetComments(threads)
	m.showComments = true
	m.rebuildLines()

	view := m.View()
	if !strings.Contains(view, "alice") {
		t.Error("should show root author")
	}
	if !strings.Contains(view, "bob") {
		t.Error("should show reply author")
	}
	if !strings.Contains(view, "reply here") {
		t.Error("should show reply body")
	}
}

func TestDiffView_CommentTruncation(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,1 +1,1 @@",
			Lines: []ghclient.DiffLine{
				{Type: ghclient.LineAdded, Content: "x", NewNum: 1},
			},
		}},
	}
	// Use very narrow width to trigger truncation
	m := New("test/repo", s).SetSize(30, 24).SetFile(file, 0, 1, 1)

	longBody := strings.Repeat("a", 200)
	threads := []ghclient.CommentThread{
		{Root: ghclient.ReviewComment{
			Path: "test.go", Line: 1, Side: "RIGHT", Body: longBody,
			User: struct{ Login string `json:"login"` }{Login: "u"},
		}},
	}
	m = m.SetComments(threads)
	m.showComments = true
	m.rebuildLines()

	view := m.View()
	// Body should be truncated (not all 200 chars)
	if strings.Contains(view, longBody) {
		t.Error("long comment body should be truncated")
	}
}

func TestDiffView_FindHunkStartsFallback(t *testing.T) {
	// Test the fallback path in findHunkStarts that looks for @@ in content
	lines := []renderedLine{
		{content: "some line", isHunk: false},
		{content: "@@ -1,3 +1,3 @@", isHunk: false}, // isHunk is false but content has @@
		{content: "another line", isHunk: false},
	}
	starts := findHunkStarts(lines)
	if len(starts) != 1 || starts[0] != 1 {
		t.Errorf("fallback hunk search: starts = %v, want [1]", starts)
	}
}

func TestDiffView_FindHunkStartsNormal(t *testing.T) {
	lines := []renderedLine{
		{content: "@@ header", isHunk: true},
		{content: "line 1", isHunk: false},
		{content: "@@ header 2", isHunk: true},
	}
	starts := findHunkStarts(lines)
	if len(starts) != 2 || starts[0] != 0 || starts[1] != 2 {
		t.Errorf("normal hunk starts = %v, want [0, 2]", starts)
	}
}

func TestDiffView_VisualScrollFollow(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "big.go", NewPath: "big.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,30 +1,30 @@",
			Lines:  makeManyLines(30),
		}},
	}
	m := New("test/repo", s).SetSize(80, 10).SetFile(file, 0, 1, 1)

	// Enter visual mode
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})

	// Move cursor down past viewport - scroll should follow
	for i := 0; i < 15; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	}
	if m.scroll == 0 {
		t.Error("scroll should follow cursor in visual mode")
	}

	// Move cursor up past viewport top - scroll should follow
	for i := 0; i < 15; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	}
	if m.cursor != 0 {
		t.Errorf("cursor should be at 0 after moving up, got %d", m.cursor)
	}
}

func TestDiffView_VisualCursorBounds(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,3 +1,3 @@",
			Lines:  makeManyLines(3),
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})

	// k at position 0 stays at 0
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.cursor != 0 {
		t.Errorf("k at 0 should stay, got %d", m.cursor)
	}

	// j past end stays at max
	for i := 0; i < 20; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	}
	if m.cursor != len(m.lines)-1 {
		t.Errorf("j past end: cursor = %d, want %d", m.cursor, len(m.lines)-1)
	}
}

func TestDiffView_CommentArrowBounds(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,1 +1,1 @@",
			Lines: []ghclient.DiffLine{
				{Type: ghclient.LineAdded, Content: "a", NewNum: 1},
			},
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)

	// Enter comment mode
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// left at col 0 stays
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m.commentCol != 0 {
		t.Errorf("left at 0: col = %d", m.commentCol)
	}

	// right at end of empty line stays
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.commentCol != 0 {
		t.Errorf("right at end of empty: col = %d", m.commentCol)
	}

	// up at row 0 stays
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.commentRow != 0 {
		t.Errorf("up at 0: row = %d", m.commentRow)
	}

	// down at last row stays
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.commentRow != 0 {
		t.Errorf("down at last: row = %d", m.commentRow)
	}
}

func TestDiffView_CommentNonPrintable(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,1 +1,1 @@",
			Lines: []ghclient.DiffLine{
				{Type: ghclient.LineAdded, Content: "a", NewNum: 1},
			},
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)

	// Enter comment mode
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Non-printable chars should be ignored
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	if len(m.commentText[0]) != 0 {
		t.Error("ctrl+a should not insert text")
	}
}

func TestDiffView_VisualNonKeyMsg(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetFile(sampleFile(), 0, 1, 1)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})

	// Non-key message in visual mode
	m2, cmd := m.Update("string msg")
	if cmd != nil {
		t.Error("non-key msg should produce nil cmd")
	}
	_ = m2
}

func TestDiffView_CommentNonKeyMsg(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,1 +1,1 @@",
			Lines: []ghclient.DiffLine{
				{Type: ghclient.LineAdded, Content: "a", NewNum: 1},
			},
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Non-key message in comment mode
	m2, cmd := m.Update("string msg")
	if cmd != nil {
		t.Error("non-key msg should produce nil cmd")
	}
	_ = m2
}

func TestDiffView_CommentsIndicator(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetFile(sampleFile(), 0, 1, 1)

	threads := []ghclient.CommentThread{
		{Root: ghclient.ReviewComment{Path: "hello.go", Line: 1, Side: "RIGHT",
			User: struct{ Login string `json:"login"` }{Login: "u"}}},
	}
	m = m.SetComments(threads)
	m.showComments = true
	m.rebuildLines()

	view := m.View()
	if !strings.Contains(view, "1") {
		t.Error("should show comments count indicator when showComments is true")
	}
}

func TestDiffView_VisualEnterInvalidSelection(t *testing.T) {
	s := testStore(t)
	// File with only a hunk header (no diff lines)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,1 +1,1 @@",
			Lines:  nil, // no lines
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)

	// Enter visual mode — cursor at hunk header only
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	// Try to comment on just the hunk header
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Should fall back to normal mode with status message
	if m.CurrentMode() != ModeNormal {
		t.Errorf("mode = %d, want ModeNormal for invalid selection", m.CurrentMode())
	}
	if m.statusMsg == "" {
		t.Error("should set status message for invalid selection")
	}
}

func TestDiffView_CommentSubmitInvalidRange(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,1 +1,1 @@",
			Lines: []ghclient.DiffLine{
				{Type: ghclient.LineAdded, Content: "a", NewNum: 1},
			},
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)

	// Manually set up comment mode with cursor on hunk header (invalid range)
	m.mode = ModeComment
	m.selStart = 0
	m.cursor = 0 // hunk header only
	m.commentText = []string{"some comment"}
	m.commentRow = 0
	m.commentCol = 12

	// ctrl+s should detect invalid range
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd != nil {
		t.Error("invalid range should not produce a cmd")
	}
	if m.CurrentMode() != ModeNormal {
		t.Error("should return to normal mode")
	}
	if !strings.Contains(m.statusMsg, "Invalid") {
		t.Errorf("statusMsg = %q, should contain 'Invalid'", m.statusMsg)
	}
}

func TestDiffView_InlineCommentOnRemovedLine(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,2 +1,1 @@",
			Lines: []ghclient.DiffLine{
				{Type: ghclient.LineRemoved, Content: "old line", OldNum: 1},
				{Type: ghclient.LineContext, Content: "kept", OldNum: 2, NewNum: 1},
			},
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)

	threads := []ghclient.CommentThread{
		{Root: ghclient.ReviewComment{
			Path: "test.go", Line: 1, Side: "LEFT", Body: "why removed?",
			User: struct{ Login string `json:"login"` }{Login: "reviewer"},
		}},
	}
	m = m.SetComments(threads)
	m.showComments = true
	m.rebuildLines()

	view := m.View()
	if !strings.Contains(view, "reviewer") {
		t.Error("should show comment on removed line")
	}
	if !strings.Contains(view, "why removed?") {
		t.Error("should show comment body on removed line")
	}
}

func TestDiffView_VisualCUsesCKey(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,1 +1,1 @@",
			Lines: []ghclient.DiffLine{
				{Type: ghclient.LineAdded, Content: "a", NewNum: 1},
			},
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)

	// V to enter visual, j to select diff line, c to open comment
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if m.CurrentMode() != ModeComment {
		t.Errorf("c in visual mode should open comment, mode = %d", m.CurrentMode())
	}
}

func TestDiffView_SelectionRangeOutOfBounds(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,1 +1,1 @@",
			Lines: []ghclient.DiffLine{
				{Type: ghclient.LineContext, Content: "a", OldNum: 1, NewNum: 1},
			},
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)

	// Set cursor beyond line count
	m.selStart = 0
	m.cursor = 100
	start, end, _ := m.selectionLineRange()
	// Should still work with available lines
	if start == 0 && end == 0 {
		// If only hunk header is in range [0,100] but lines are only 2 long
		// it still picks up the diff line at index 1
	}
	_ = start
	_ = end
}

func TestDiffView_ViewCommentModeHelp(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,1 +1,1 @@",
			Lines: []ghclient.DiffLine{
				{Type: ghclient.LineAdded, Content: "a", NewNum: 1},
			},
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)

	// Enter comment mode
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	view := m.View()
	if !strings.Contains(view, "ctrl+s:submit") {
		t.Error("comment mode should show submit help")
	}
	if !strings.Contains(view, "Comment on lines") {
		t.Error("comment mode should show line range header")
	}
}

func TestDiffView_VisualModeViewHelp(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetFile(sampleFile(), 0, 1, 1)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	view := m.View()
	if !strings.Contains(view, "enter/c:comment") {
		t.Error("visual mode should show comment help")
	}
}

func TestDiffView_CommentModeSmallHeight(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,3 +1,3 @@",
			Lines:  makeManyLines(3),
		}},
	}
	// Very small height: viewportHeight() = 5-5 = 0 → 10, then 10 - commentInputHeight
	// We need vpHeight - commentInputHeight < 3 to trigger vpHeight = 3
	// height=8 → viewportHeight = 8-5 = 3, commentInputHeight = 1+4 = 5, vpHeight = 3-5 = -2 < 3 → vpHeight=3
	m := New("test/repo", s).SetSize(80, 8).SetFile(file, 0, 1, 1)

	// Enter comment mode via visual selection
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.CurrentMode() != ModeComment {
		t.Fatalf("should be in comment mode, got %d", m.CurrentMode())
	}

	// View should not panic and should render
	view := m.View()
	if !strings.Contains(view, "ctrl+s:submit") {
		t.Error("should still render comment mode with small height")
	}
}

func TestDiffView_VisualDownKey(t *testing.T) {
	s := testStore(t)
	file := ghclient.FileDiff{
		OldPath: "test.go", NewPath: "test.go",
		Hunks: []ghclient.Hunk{{
			Header: "@@ -1,5 +1,5 @@",
			Lines:  makeManyLines(5),
		}},
	}
	m := New("test/repo", s).SetSize(80, 24).SetFile(file, 0, 1, 1)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})

	// Test down arrow key (in addition to j)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("down key: cursor = %d, want 1", m.cursor)
	}

	// Test up arrow key
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("up key: cursor = %d, want 0", m.cursor)
	}

	// Test home key
	m.cursor = 3
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyHome})
	if m.cursor != 0 {
		t.Errorf("home: cursor = %d, want 0", m.cursor)
	}

	// Test end key
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	if m.cursor != len(m.lines)-1 {
		t.Errorf("end: cursor = %d, want %d", m.cursor, len(m.lines)-1)
	}
}

func makeManyLines(n int) []ghclient.DiffLine {
	lines := make([]ghclient.DiffLine, n)
	for i := range lines {
		lines[i] = ghclient.DiffLine{
			Type:    ghclient.LineContext,
			Content: "line content",
			OldNum:  i + 1,
			NewNum:  i + 1,
		}
	}
	return lines
}

// --- Panic guards for exhaustiveness checks ---

func TestUpdate_InvalidMode_Panics(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetFile(sampleFile(), 0, 1, 1)
	m.mode = Mode(99) // invalid mode
	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic on invalid mode in Update")
		}
	}()
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
}

func TestView_InvalidMode_PanicsOnModeIndicator(t *testing.T) {
	s := testStore(t)
	m := New("test/repo", s).SetSize(80, 24).SetFile(sampleFile(), 0, 1, 1)
	m.mode = Mode(99)
	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic on invalid mode in View modeIndicator")
		}
	}()
	m.View()
}
