package comments

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/LumaKernel/ghpr-tui/internal/ghclient"
)

func samplePR() ghclient.PR {
	return ghclient.PR{Number: 42, Title: "Test PR"}
}

func sampleThreads() []ghclient.CommentThread {
	return []ghclient.CommentThread{
		{
			Root: ghclient.ReviewComment{
				ID: 1, Body: "First comment", Path: "main.go", Line: 10,
				CreatedAt: time.Now().Add(-5 * time.Minute),
				User:      struct{ Login string `json:"login"` }{Login: "alice"},
			},
		},
		{
			Root: ghclient.ReviewComment{
				ID: 2, Body: "Second comment", Path: "util.go", Line: 20,
				CreatedAt: time.Now().Add(-2 * time.Hour),
				User:      struct{ Login string `json:"login"` }{Login: "bob"},
			},
			Replies: []ghclient.ReviewComment{
				{ID: 3, Body: "Reply", InReplyToID: 2},
			},
		},
	}
}

func TestComments_New(t *testing.T) {
	m := New()
	if m.loading {
		t.Error("new model should not be loading")
	}
}

func TestComments_SetPR(t *testing.T) {
	m := New().SetPR(samplePR())
	if !m.loading {
		t.Error("SetPR should set loading")
	}
	if m.cursor != 0 {
		t.Error("SetPR should reset cursor")
	}
}

func TestComments_SetComments(t *testing.T) {
	m := New().SetPR(samplePR()).SetComments(sampleThreads())
	if m.loading {
		t.Error("SetComments should clear loading")
	}
	if len(m.threads) != 2 {
		t.Errorf("threads = %d, want 2", len(m.threads))
	}
}

func TestComments_SetError(t *testing.T) {
	m := New().SetError(nil)
	if m.loading {
		t.Error("SetError should clear loading")
	}
}

func TestComments_SetSize(t *testing.T) {
	m := New().SetSize(100, 50)
	if m.width != 100 || m.height != 50 {
		t.Error("SetSize should update dimensions")
	}
}

func TestComments_Navigation(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(samplePR()).SetComments(sampleThreads())

	// j moves down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != 1 {
		t.Errorf("after j: cursor = %d, want 1", m.cursor)
	}

	// j at bottom stays
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.cursor != 1 {
		t.Errorf("j at bottom: cursor = %d, want 1", m.cursor)
	}

	// k moves up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.cursor != 0 {
		t.Errorf("after k: cursor = %d, want 0", m.cursor)
	}

	// k at top stays
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m.cursor != 0 {
		t.Errorf("k at top: cursor = %d, want 0", m.cursor)
	}
}

func TestComments_GAndG(t *testing.T) {
	threads := make([]ghclient.CommentThread, 20)
	for i := range threads {
		threads[i] = ghclient.CommentThread{
			Root: ghclient.ReviewComment{ID: i + 1, Path: "f.go", Line: i + 1,
				User: struct{ Login string `json:"login"` }{Login: "user"}},
		}
	}
	m := New().SetSize(80, 24).SetPR(samplePR()).SetComments(threads)

	// G goes to end
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if m.cursor != 19 {
		t.Errorf("after G: cursor = %d, want 19", m.cursor)
	}

	// g goes to start
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if m.cursor != 0 {
		t.Errorf("after g: cursor = %d, want 0", m.cursor)
	}
}

func TestComments_HalfPageScroll(t *testing.T) {
	threads := make([]ghclient.CommentThread, 30)
	for i := range threads {
		threads[i] = ghclient.CommentThread{
			Root: ghclient.ReviewComment{ID: i + 1, Path: "f.go", Line: i + 1,
				User: struct{ Login string `json:"login"` }{Login: "user"}},
		}
	}
	m := New().SetSize(80, 20).SetPR(samplePR()).SetComments(threads)

	// ctrl+d scrolls down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	if m.cursor == 0 {
		t.Error("ctrl+d should move cursor")
	}

	// ctrl+u scrolls up
	saved := m.cursor
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	if m.cursor >= saved {
		t.Error("ctrl+u should move cursor up")
	}
}

func TestComments_BackMsg(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(samplePR()).SetComments(sampleThreads())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc should produce a cmd")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Errorf("esc should produce BackMsg, got %T", cmd())
	}
}

func TestComments_JumpToFile(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(samplePR()).SetComments(sampleThreads())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter should produce a cmd")
	}
	msg, ok := cmd().(JumpToFileMsg)
	if !ok {
		t.Fatalf("enter should produce JumpToFileMsg, got %T", cmd())
	}
	if msg.Path != "main.go" {
		t.Errorf("Path = %q, want %q", msg.Path, "main.go")
	}
	if msg.Line != 10 {
		t.Errorf("Line = %d, want 10", msg.Line)
	}
}

func TestComments_Refresh(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(samplePR()).SetComments(sampleThreads())

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})
	if cmd == nil {
		t.Fatal("R should produce a cmd")
	}
	msg, ok := cmd().(RefreshMsg)
	if !ok {
		t.Fatalf("R should produce RefreshMsg, got %T", cmd())
	}
	if msg.Number != 42 {
		t.Errorf("Number = %d, want 42", msg.Number)
	}
	if !m.loading {
		t.Error("R should set loading")
	}
}

func TestComments_Reply(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(samplePR()).SetComments(sampleThreads())

	// r enters reply mode
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if !m.replyMode {
		t.Error("r should enter reply mode")
	}

	// Type reply
	for _, ch := range "thanks" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}
	if m.replyText[0] != "thanks" {
		t.Errorf("reply text = %q, want %q", m.replyText[0], "thanks")
	}

	// ctrl+s submits
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd == nil {
		t.Fatal("ctrl+s should produce a cmd")
	}
	msg, ok := cmd().(ReplyMsg)
	if !ok {
		t.Fatalf("ctrl+s should produce ReplyMsg, got %T", cmd())
	}
	if msg.Body != "thanks" {
		t.Errorf("Body = %q, want %q", msg.Body, "thanks")
	}
	if msg.PRNumber != 42 {
		t.Errorf("PRNumber = %d, want 42", msg.PRNumber)
	}
}

func TestComments_ReplyCancel(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(samplePR()).SetComments(sampleThreads())

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.replyMode {
		t.Error("esc should exit reply mode")
	}
}

func TestComments_ReplyEmptyBody(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(samplePR()).SetComments(sampleThreads())

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	// Submit empty
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd != nil {
		t.Error("empty reply should not produce a cmd")
	}
}

func TestComments_ReplyBackspace(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(samplePR()).SetComments(sampleThreads())

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	for _, ch := range "ab" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.replyText[0] != "a" {
		t.Errorf("after backspace: %q, want %q", m.replyText[0], "a")
	}
}

func TestComments_ReplyMultiline(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(samplePR()).SetComments(sampleThreads())

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	for _, ch := range "line1" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	for _, ch := range "line2" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}

	if len(m.replyText) != 2 {
		t.Errorf("lines = %d, want 2", len(m.replyText))
	}

	// Backspace at start of line 2 joins
	m.replyCol = 0
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if len(m.replyText) != 1 {
		t.Errorf("after join: lines = %d, want 1", len(m.replyText))
	}
}

func TestComments_View(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(samplePR()).SetComments(sampleThreads())
	view := m.View()

	if !strings.Contains(view, "#42") {
		t.Error("view should contain PR number")
	}
	if !strings.Contains(view, "alice") {
		t.Error("view should contain comment author")
	}
	if !strings.Contains(view, "main.go") {
		t.Error("view should contain file path")
	}
	if !strings.Contains(view, "+1 replies") {
		t.Error("view should show reply count")
	}
}

func TestComments_ViewEmpty(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(samplePR()).SetComments(nil)
	view := m.View()
	if !strings.Contains(view, "No review comments") {
		t.Error("view should show no comments message")
	}
}

func TestComments_ViewLoading(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(samplePR())
	view := m.View()
	if !strings.Contains(view, "Loading") {
		t.Error("view should show loading message")
	}
}

func TestComments_ViewError(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(samplePR())
	m = m.SetError(nil)
	// SetError with nil clears error but stops loading
	if m.loading {
		t.Error("SetError should stop loading")
	}
}

func TestComments_ViewReplyMode(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(samplePR()).SetComments(sampleThreads())
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	view := m.View()
	if !strings.Contains(view, "Reply") {
		t.Error("view should show reply prompt")
	}
	if !strings.Contains(view, "ctrl+s") {
		t.Error("view should show submit hint")
	}
}

func TestComments_NonKeyMsg(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(samplePR()).SetComments(sampleThreads())
	m2, cmd := m.Update("string msg")
	if cmd != nil {
		t.Error("non-key msg should produce nil cmd")
	}
	_ = m2
}

func TestComments_SetCommentsClampsCursor(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(samplePR()).SetComments(sampleThreads())
	m.cursor = 5 // beyond range
	m = m.SetComments(sampleThreads())
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (clamped)", m.cursor)
	}
}

func TestComments_ViewErrorState(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(samplePR())
	m = m.SetError(fmt.Errorf("something went wrong"))
	view := m.View()
	if !strings.Contains(view, "Error") {
		t.Error("view should show error")
	}
	if !strings.Contains(view, "something went wrong") {
		t.Error("view should show error message")
	}
}

func TestComments_SmallHeight(t *testing.T) {
	// Test visibleHeight fallback when height < 7+1
	m := New().SetSize(80, 3).SetPR(samplePR()).SetComments(sampleThreads())
	// Should not panic
	view := m.View()
	if !strings.Contains(view, "#42") {
		t.Error("should render even with small height")
	}
}

func TestComments_NarrowWidth(t *testing.T) {
	// Test body truncation with narrow width
	threads := []ghclient.CommentThread{
		{Root: ghclient.ReviewComment{
			ID: 1, Body: strings.Repeat("x", 200), Path: "f.go", Line: 1,
			CreatedAt: time.Now(),
			User:      struct{ Login string `json:"login"` }{Login: "u"},
		}},
	}
	m := New().SetSize(30, 24).SetPR(samplePR()).SetComments(threads)
	view := m.View()
	// Body should be truncated
	if strings.Contains(view, strings.Repeat("x", 200)) {
		t.Error("long body should be truncated")
	}
}

func TestComments_MultiLineRange(t *testing.T) {
	// Test thread where StartLine != Line
	threads := []ghclient.CommentThread{
		{Root: ghclient.ReviewComment{
			ID: 1, Body: "multi-line", Path: "f.go", Line: 10, StartLine: 5,
			CreatedAt: time.Now(),
			User:      struct{ Login string `json:"login"` }{Login: "u"},
		}},
	}
	m := New().SetSize(80, 24).SetPR(samplePR()).SetComments(threads)
	view := m.View()
	// Should show range format "f.go:5-10"
	if !strings.Contains(view, "5-10") {
		t.Error("should show line range for multi-line comments")
	}
}

func TestComments_EnterNoThreads(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(samplePR()).SetComments(nil)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("enter with no threads should not produce a cmd")
	}
}

func TestComments_ReplyNoThreads(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(samplePR()).SetComments(nil)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if m.replyMode {
		t.Error("r with no threads should not enter reply mode")
	}
}

func TestComments_GNoThreads(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(samplePR()).SetComments(nil)
	// G with no threads should not panic
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

func TestComments_QuitKeys(t *testing.T) {
	m := New().SetSize(80, 24).SetPR(samplePR()).SetComments(sampleThreads())

	// q should produce BackMsg
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("q should produce a cmd")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Errorf("q should produce BackMsg, got %T", cmd())
	}

	// backspace should produce BackMsg
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if cmd == nil {
		t.Fatal("backspace should produce a cmd")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Errorf("backspace should produce BackMsg, got %T", cmd())
	}
}

func TestComments_ViewScrollWindow(t *testing.T) {
	// Create enough threads to trigger startIdx = m.cursor - vpHeight + 1
	threads := make([]ghclient.CommentThread, 30)
	for i := range threads {
		threads[i] = ghclient.CommentThread{
			Root: ghclient.ReviewComment{
				ID: i + 1, Body: fmt.Sprintf("comment %d", i), Path: "f.go", Line: i + 1,
				CreatedAt: time.Now().Add(-time.Duration(i) * time.Minute),
				User:      struct{ Login string `json:"login"` }{Login: "user"},
			},
		}
	}
	// height=15 → visibleHeight = 15-7 = 8
	m := New().SetSize(80, 15).SetPR(samplePR()).SetComments(threads)
	// Move cursor to bottom
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	if m.cursor != 29 {
		t.Fatalf("cursor = %d, want 29", m.cursor)
	}
	view := m.View()
	// Last thread should be visible
	if !strings.Contains(view, "comment 29") {
		t.Error("should show last comment when scrolled to bottom")
	}
	// First threads should not be visible
	if strings.Contains(view, "comment 0 ") {
		t.Error("first comment should not be visible when scrolled to bottom")
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		t    time.Time
		want string
	}{
		{now.Add(-30 * time.Second), "just now"},
		{now.Add(-5 * time.Minute), "5m ago"},
		{now.Add(-3 * time.Hour), "3h ago"},
		{now.Add(-2 * 24 * time.Hour), "2d ago"},
		{now.Add(-30 * 24 * time.Hour), now.Add(-30 * 24 * time.Hour).Format("Jan 2")},
	}
	for _, tt := range tests {
		got := relativeTime(tt.t)
		if got != tt.want {
			t.Errorf("relativeTime(%v) = %q, want %q", tt.t, got, tt.want)
		}
	}
}
