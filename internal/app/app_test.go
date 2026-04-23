package app

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/LumaKernel/ghpr-tui/internal/ghclient"
	"github.com/LumaKernel/ghpr-tui/internal/state"
	"github.com/LumaKernel/ghpr-tui/internal/ui/checks"
	"github.com/LumaKernel/ghpr-tui/internal/ui/comments"
	"github.com/LumaKernel/ghpr-tui/internal/ui/diffview"
	"github.com/LumaKernel/ghpr-tui/internal/ui/filelist"
	"github.com/LumaKernel/ghpr-tui/internal/ui/prlist"
)

func testStore(t *testing.T) *state.Store {
	t.Helper()
	s, err := state.NewWithPath(t.TempDir() + "/state.json")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func testModel(t *testing.T) Model {
	t.Helper()
	s := testStore(t)
	client := ghclient.NewClient("")
	return New("test/repo", client, s)
}

func testModelWithMockClient(t *testing.T, runFn func(args ...string) (string, error)) Model {
	t.Helper()
	s := testStore(t)
	client := ghclient.NewTestClient("test/repo", runFn)
	return New("test/repo", client, s)
}

func samplePRs() []ghclient.PR {
	return []ghclient.PR{
		{Number: 1, Title: "First PR", Author: "alice", HeadRef: "feat-1", BaseRef: "main"},
		{Number: 2, Title: "Second PR", Author: "bob", IsDraft: true, HeadRef: "feat-2", BaseRef: "main"},
	}
}

func sampleDiff() ghclient.ParsedDiff {
	return ghclient.ParsedDiff{
		Files: []ghclient.FileDiff{
			{
				OldPath: "a.go", NewPath: "a.go",
				Hunks: []ghclient.Hunk{{
					Header: "@@ -1,1 +1,2 @@",
					Lines: []ghclient.DiffLine{
						{Type: ghclient.LineContext, Content: "pkg", OldNum: 1, NewNum: 1},
						{Type: ghclient.LineAdded, Content: "new", NewNum: 2},
					},
				}},
			},
			{
				OldPath: "b.go", NewPath: "b.go",
				Hunks: []ghclient.Hunk{{
					Header: "@@ -1,1 +1,1 @@",
					Lines: []ghclient.DiffLine{
						{Type: ghclient.LineContext, Content: "x", OldNum: 1, NewNum: 1},
					},
				}},
			},
		},
	}
}

// --- New / Init ---

func TestNew(t *testing.T) {
	m := testModel(t)
	if m.screen != ScreenPRList {
		t.Errorf("screen = %d, want ScreenPRList", m.screen)
	}
}

func TestInit(t *testing.T) {
	m := testModel(t)
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init should return a cmd")
	}
}

// --- WindowSizeMsg ---

func TestWindowSizeMsg(t *testing.T) {
	m := testModel(t)
	m2, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if cmd != nil {
		t.Error("WindowSizeMsg should not produce a cmd")
	}
	mm := m2.(Model)
	if mm.width != 120 || mm.height != 40 {
		t.Errorf("size = %dx%d, want 120x40", mm.width, mm.height)
	}
}

// --- PRsLoadedMsg ---

func TestPRsLoadedMsg(t *testing.T) {
	m := testModel(t)
	m2, _ := m.Update(prsLoadedMsg{prs: samplePRs()})
	mm := m2.(Model)
	view := mm.prList.View()
	if !strings.Contains(view, "First PR") {
		t.Error("PR list should show loaded PRs")
	}
}

func TestPRsLoadedMsg_Error(t *testing.T) {
	m := testModel(t)
	m2, _ := m.Update(prsLoadedMsg{err: fmt.Errorf("network error")})
	mm := m2.(Model)
	view := mm.prList.View()
	if !strings.Contains(view, "network error") {
		t.Error("PR list should show error")
	}
}

// --- DiffLoadedMsg ---

func TestDiffLoadedMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenFileList
	m2, _ := m.Update(diffLoadedMsg{diff: sampleDiff()})
	mm := m2.(Model)
	view := mm.fileList.View()
	if !strings.Contains(view, "a.go") {
		t.Error("file list should show loaded files")
	}
}

func TestDiffLoadedMsg_Error(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenFileList
	m2, _ := m.Update(diffLoadedMsg{err: fmt.Errorf("diff error")})
	mm := m2.(Model)
	view := mm.fileList.View()
	if !strings.Contains(view, "diff error") {
		t.Error("file list should show error")
	}
}

// --- ChecksLoadedMsg ---

func TestChecksLoadedMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenChecks
	chks := []ghclient.Check{{Name: "CI", Bucket: "pass", State: "completed"}}
	m2, _ := m.Update(checksLoadedMsg{checks: chks})
	mm := m2.(Model)
	view := mm.checks.View()
	if !strings.Contains(view, "CI") {
		t.Error("checks should show loaded checks")
	}
}

func TestChecksLoadedMsg_Error(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenChecks
	m2, _ := m.Update(checksLoadedMsg{err: fmt.Errorf("checks error")})
	mm := m2.(Model)
	view := mm.checks.View()
	if !strings.Contains(view, "checks error") {
		t.Error("checks should show error")
	}
}

// --- CommentsLoadedMsg ---

func TestCommentsLoadedMsg(t *testing.T) {
	m := testModel(t)
	cs := []ghclient.ReviewComment{
		{ID: 1, Body: "test comment", Path: "a.go", Line: 1,
			User: struct{ Login string `json:"login"` }{Login: "alice"}},
	}
	m2, _ := m.Update(commentsLoadedMsg{comments: cs})
	mm := m2.(Model)
	// Comments should be stored on both comments and diffView models
	_ = mm
}

func TestCommentsLoadedMsg_Error(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenComments
	m2, _ := m.Update(commentsLoadedMsg{err: fmt.Errorf("comments error")})
	mm := m2.(Model)
	view := mm.comments.View()
	if !strings.Contains(view, "comments error") {
		t.Error("comments should show error")
	}
}

func TestCommentsLoadedMsg_ErrorNotOnCommentsScreen(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenPRList // not on comments screen
	m2, _ := m.Update(commentsLoadedMsg{err: fmt.Errorf("comments error")})
	mm := m2.(Model)
	// Should not crash, error is silently ignored when not on comments screen
	_ = mm
}

// --- HeadSHALoadedMsg ---

func TestHeadSHALoadedMsg(t *testing.T) {
	m := testModel(t)
	m2, _ := m.Update(headSHALoadedMsg{sha: "abc123"})
	mm := m2.(Model)
	if mm.diffView.HeadSHA() != "abc123" {
		t.Errorf("HeadSHA = %q, want %q", mm.diffView.HeadSHA(), "abc123")
	}
}

func TestHeadSHALoadedMsg_Error(t *testing.T) {
	m := testModel(t)
	m2, _ := m.Update(headSHALoadedMsg{err: fmt.Errorf("sha error")})
	mm := m2.(Model)
	if mm.diffView.HeadSHA() != "" {
		t.Errorf("HeadSHA should be empty on error, got %q", mm.diffView.HeadSHA())
	}
}

// --- CommentPostedMsg ---

func TestCommentPostedMsg_Success(t *testing.T) {
	m := testModel(t)
	m2, cmd := m.Update(commentPostedMsg{number: 1})
	mm := m2.(Model)
	// Should refresh comments
	if cmd == nil {
		t.Error("should produce a refresh cmd")
	}
	_ = mm
}

func TestCommentPostedMsg_Error(t *testing.T) {
	m := testModel(t)
	m2, _ := m.Update(commentPostedMsg{number: 1, err: fmt.Errorf("post failed")})
	mm := m2.(Model)
	_ = mm
}

// --- ReplyPostedMsg ---

func TestReplyPostedMsg_Success(t *testing.T) {
	m := testModel(t)
	m2, cmd := m.Update(replyPostedMsg{number: 1})
	if cmd == nil {
		t.Error("should produce a refresh cmd")
	}
	_ = m2
}

func TestReplyPostedMsg_Error(t *testing.T) {
	m := testModel(t)
	m2, _ := m.Update(replyPostedMsg{number: 1, err: fmt.Errorf("reply failed")})
	mm := m2.(Model)
	_ = mm
}

// --- MergeSettingsMsg ---

func TestMergeSettingsMsg(t *testing.T) {
	m := testModel(t)
	m2, _ := m.Update(mergeSettingsMsg{methods: []string{"squash", "merge"}})
	mm := m2.(Model)
	_ = mm
}

// --- BrowserOpenedMsg ---

func TestBrowserOpenedMsg(t *testing.T) {
	m := testModel(t)
	m2, cmd := m.Update(browserOpenedMsg{})
	if cmd != nil {
		t.Error("browserOpenedMsg should not produce cmd")
	}
	_ = m2
}

// --- DraftToggledMsg ---

func TestDraftToggledMsg_Success(t *testing.T) {
	m := testModel(t)
	m2, cmd := m.Update(draftToggledMsg{})
	if cmd == nil {
		t.Error("should reload PRs")
	}
	_ = m2
}

func TestDraftToggledMsg_Error(t *testing.T) {
	m := testModel(t)
	m2, cmd := m.Update(draftToggledMsg{err: fmt.Errorf("toggle failed")})
	if cmd == nil {
		t.Error("should still reload PRs")
	}
	_ = m2
}

// --- MergeResultMsg ---

func TestMergeResultMsg_Success(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenFileList
	m2, cmd := m.Update(mergeResultMsg{number: 1})
	mm := m2.(Model)
	if mm.screen != ScreenPRList {
		t.Error("should go to PR list after merge")
	}
	if cmd == nil {
		t.Error("should reload PRs")
	}
}

func TestMergeResultMsg_Error(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenFileList
	m2, cmd := m.Update(mergeResultMsg{number: 1, err: fmt.Errorf("merge failed")})
	mm := m2.(Model)
	if mm.screen != ScreenFileList {
		t.Error("should stay on file list after merge failure")
	}
	if cmd != nil {
		t.Error("should not reload on error")
	}
}

// --- Quit key on different screens ---

func TestQuitFromPRList(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenPRList
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Error("q on PR list should quit")
	}
}

func TestQuitFromFileList(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenFileList
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	mm := m2.(Model)
	if mm.screen != ScreenPRList {
		t.Errorf("q on file list should go to PR list, got screen %d", mm.screen)
	}
}

func TestQuitFromDiffView(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenDiffView
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	mm := m2.(Model)
	if mm.screen != ScreenFileList {
		t.Errorf("q on diff view should go to file list, got screen %d", mm.screen)
	}
}

func TestQuitFromChecks(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenChecks
	m.prevScreen = ScreenPRList
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	mm := m2.(Model)
	if mm.screen != ScreenPRList {
		t.Errorf("q on checks should go to prev screen, got screen %d", mm.screen)
	}
}

func TestQuitFromComments(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenComments
	m.prevScreen = ScreenDiffView
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	mm := m2.(Model)
	if mm.screen != ScreenDiffView {
		t.Errorf("q on comments should go to prev screen, got screen %d", mm.screen)
	}
}

func TestCtrlCFromPRList(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenPRList
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("ctrl+c on PR list should quit")
	}
}

// --- DiffView input mode bypasses quit ---

func TestDiffViewInputModeBypassesQuit(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenDiffView
	// Set up diffView in visual mode
	file := sampleDiff().Files[0]
	m.diffView = m.diffView.SetSize(80, 24).SetFile(file, 0, 2, 1)
	// Enter visual mode
	m.diffView, _ = m.diffView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("V")})
	if !m.diffView.IsInputMode() {
		t.Fatal("diffView should be in input mode")
	}

	// q should NOT quit when in input mode
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	mm := m2.(Model)
	if mm.screen != ScreenDiffView {
		t.Error("q in input mode should not navigate away")
	}
}

// --- Screen delegation: PRList messages ---

func TestPRList_SelectMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenPRList
	pr := ghclient.PR{Number: 1, Title: "Test", HeadRef: "feat", BaseRef: "main"}
	m2, cmd := m.Update(prlist.SelectMsg{PR: pr})
	mm := m2.(Model)
	if mm.screen != ScreenFileList {
		t.Error("select should go to file list")
	}
	if mm.currentPRNumber != 1 {
		t.Errorf("currentPRNumber = %d, want 1", mm.currentPRNumber)
	}
	if cmd == nil {
		t.Error("should produce a batch cmd to load diff/comments/sha")
	}
}

func TestPRList_RefreshMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenPRList
	m2, cmd := m.Update(prlist.RefreshMsg{})
	if cmd == nil {
		t.Error("refresh should produce a cmd")
	}
	_ = m2
}

func TestPRList_OpenBrowserMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenPRList
	_, cmd := m.Update(prlist.OpenBrowserMsg{Number: 1})
	if cmd == nil {
		t.Error("open browser should produce a cmd")
	}
}

func TestPRList_MergeMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenPRList
	_, cmd := m.Update(prlist.MergeMsg{Number: 1, Method: "squash"})
	if cmd == nil {
		t.Error("merge should produce a cmd")
	}
}

func TestPRList_ToggleDraftMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenPRList
	_, cmd := m.Update(prlist.ToggleDraftMsg{Number: 1, IsDraft: true})
	if cmd == nil {
		t.Error("toggle draft should produce a cmd")
	}
}

func TestPRList_OpenChecksMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenPRList
	pr := ghclient.PR{Number: 5}
	m2, cmd := m.Update(prlist.OpenChecksMsg{PR: pr})
	mm := m2.(Model)
	if mm.screen != ScreenChecks {
		t.Error("should go to checks screen")
	}
	if mm.prevScreen != ScreenPRList {
		t.Error("prevScreen should be PRList")
	}
	if cmd == nil {
		t.Error("should load checks")
	}
}

// --- Screen delegation: FileList messages ---

func TestFileList_SelectFileMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenFileList
	// Load diff first
	m2, _ := m.Update(diffLoadedMsg{diff: sampleDiff()})
	m = m2.(Model)
	// Also set PR
	pr := ghclient.PR{Number: 1}
	m.fileList = m.fileList.SetPR(pr)

	m2, _ = m.Update(filelist.SelectFileMsg{Index: 0})
	mm := m2.(Model)
	if mm.screen != ScreenDiffView {
		t.Error("select file should go to diff view")
	}
}

func TestFileList_BackMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenFileList
	m2, _ := m.Update(filelist.BackMsg{})
	mm := m2.(Model)
	if mm.screen != ScreenPRList {
		t.Error("back from file list should go to PR list")
	}
}

func TestFileList_MergeMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenFileList
	_, cmd := m.Update(filelist.MergeMsg{Number: 1, Method: "squash"})
	if cmd == nil {
		t.Error("merge should produce a cmd")
	}
}

func TestFileList_OpenChecksMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenFileList
	pr := ghclient.PR{Number: 3}
	m2, cmd := m.Update(filelist.OpenChecksMsg{PR: pr})
	mm := m2.(Model)
	if mm.screen != ScreenChecks {
		t.Error("should go to checks screen")
	}
	if mm.prevScreen != ScreenFileList {
		t.Error("prevScreen should be FileList")
	}
	if cmd == nil {
		t.Error("should load checks")
	}
}

// --- Screen delegation: DiffView messages ---

func TestDiffView_BackMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenDiffView
	m2, _ := m.Update(diffview.BackMsg{})
	mm := m2.(Model)
	if mm.screen != ScreenFileList {
		t.Error("back from diff view should go to file list")
	}
}

func TestDiffView_NextFileMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenDiffView
	// Load diff
	m2, _ := m.Update(diffLoadedMsg{diff: sampleDiff()})
	m = m2.(Model)
	pr := ghclient.PR{Number: 1}
	m.fileList = m.fileList.SetPR(pr)
	m.diffView = m.diffView.SetSize(80, 24).SetFile(sampleDiff().Files[0], 0, 2, 1)

	m2, _ = m.Update(diffview.NextFileMsg{})
	mm := m2.(Model)
	if mm.screen != ScreenDiffView {
		t.Error("should stay on diff view")
	}
	if mm.diffView.FileIndex() != 1 {
		t.Errorf("FileIndex = %d, want 1", mm.diffView.FileIndex())
	}
}

func TestDiffView_NextFileMsg_Last(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenDiffView
	m2, _ := m.Update(diffLoadedMsg{diff: sampleDiff()})
	m = m2.(Model)
	pr := ghclient.PR{Number: 1}
	m.fileList = m.fileList.SetPR(pr)
	m.diffView = m.diffView.SetSize(80, 24).SetFile(sampleDiff().Files[1], 1, 2, 1)

	m2, _ = m.Update(diffview.NextFileMsg{})
	mm := m2.(Model)
	if mm.screen != ScreenFileList {
		t.Error("next file at last should go to file list")
	}
}

func TestDiffView_PrevFileMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenDiffView
	m2, _ := m.Update(diffLoadedMsg{diff: sampleDiff()})
	m = m2.(Model)
	pr := ghclient.PR{Number: 1}
	m.fileList = m.fileList.SetPR(pr)
	m.diffView = m.diffView.SetSize(80, 24).SetFile(sampleDiff().Files[1], 1, 2, 1)

	m2, _ = m.Update(diffview.PrevFileMsg{})
	mm := m2.(Model)
	if mm.diffView.FileIndex() != 0 {
		t.Errorf("FileIndex = %d, want 0", mm.diffView.FileIndex())
	}
}

func TestDiffView_PrevFileMsg_First(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenDiffView
	m2, _ := m.Update(diffLoadedMsg{diff: sampleDiff()})
	m = m2.(Model)
	pr := ghclient.PR{Number: 1}
	m.fileList = m.fileList.SetPR(pr)
	m.diffView = m.diffView.SetSize(80, 24).SetFile(sampleDiff().Files[0], 0, 2, 1)

	m2, _ = m.Update(diffview.PrevFileMsg{})
	mm := m2.(Model)
	// Should stay on diff view with same file
	if mm.screen != ScreenDiffView {
		t.Error("prev file at first should stay on diff view")
	}
}

func TestDiffView_OpenCommentsMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenDiffView
	pr := ghclient.PR{Number: 10}
	m.fileList = m.fileList.SetPR(pr)

	m2, cmd := m.Update(diffview.OpenCommentsMsg{PRNumber: 10})
	mm := m2.(Model)
	if mm.screen != ScreenComments {
		t.Error("should go to comments screen")
	}
	if mm.prevScreen != ScreenDiffView {
		t.Error("prevScreen should be DiffView")
	}
	if cmd == nil {
		t.Error("should load comments")
	}
}

func TestDiffView_PostCommentMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenDiffView
	m.diffView = m.diffView.SetHeadSHA("abc123")

	_, cmd := m.Update(diffview.PostCommentMsg{
		PRNumber: 1, Body: "test", Path: "a.go", Line: 10, StartLine: 5, Side: "RIGHT",
	})
	if cmd == nil {
		t.Error("post comment should produce a cmd")
	}
}

// --- Screen delegation: Checks messages ---

func TestChecks_BackMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenChecks
	m.prevScreen = ScreenFileList
	m2, _ := m.Update(checks.BackMsg{})
	mm := m2.(Model)
	if mm.screen != ScreenFileList {
		t.Error("back from checks should go to prev screen")
	}
}

func TestChecks_OpenBrowserMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenChecks
	_, cmd := m.Update(checks.OpenBrowserMsg{URL: "https://example.com"})
	if cmd == nil {
		t.Error("open browser should produce a cmd")
	}
}

func TestChecks_RefreshMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenChecks
	_, cmd := m.Update(checks.RefreshMsg{Number: 1})
	if cmd == nil {
		t.Error("refresh should produce a cmd")
	}
}

// --- Screen delegation: Comments messages ---

func TestComments_BackMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenComments
	m.prevScreen = ScreenDiffView
	m2, _ := m.Update(comments.BackMsg{})
	mm := m2.(Model)
	if mm.screen != ScreenDiffView {
		t.Error("back from comments should go to prev screen")
	}
}

func TestComments_RefreshMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenComments
	_, cmd := m.Update(comments.RefreshMsg{Number: 1})
	if cmd == nil {
		t.Error("refresh should produce a cmd")
	}
}

func TestComments_JumpToFileMsg_Found(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenComments
	m.prevScreen = ScreenDiffView
	// Load diff
	m2, _ := m.Update(diffLoadedMsg{diff: sampleDiff()})
	m = m2.(Model)
	pr := ghclient.PR{Number: 1}
	m.fileList = m.fileList.SetPR(pr)
	m.diffView = m.diffView.SetSize(80, 24)

	m2, _ = m.Update(comments.JumpToFileMsg{Path: "a.go", Line: 1})
	mm := m2.(Model)
	if mm.screen != ScreenDiffView {
		t.Error("should jump to diff view")
	}
}

func TestComments_JumpToFileMsg_NotFound(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenComments
	m.prevScreen = ScreenDiffView
	// No diff loaded
	m2, _ := m.Update(comments.JumpToFileMsg{Path: "nonexistent.go", Line: 1})
	mm := m2.(Model)
	if mm.screen != ScreenDiffView {
		t.Error("should go to prev screen when file not found")
	}
}

func TestComments_ReplyMsg(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenComments
	_, cmd := m.Update(comments.ReplyMsg{PRNumber: 1, InReplyToID: 10, Body: "thanks"})
	if cmd == nil {
		t.Error("reply should produce a cmd")
	}
}

// --- View on all screens ---

func TestView_PRList(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenPRList
	view := m.View()
	if view == "" {
		t.Error("PR list view should not be empty")
	}
}

func TestView_FileList(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenFileList
	view := m.View()
	if view == "" {
		t.Error("file list view should not be empty")
	}
}

func TestView_DiffView(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenDiffView
	m.diffView = m.diffView.SetSize(80, 24).SetFile(sampleDiff().Files[0], 0, 1, 1)
	view := m.View()
	if view == "" {
		t.Error("diff view should not be empty")
	}
}

func TestView_Checks(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenChecks
	view := m.View()
	if view == "" {
		t.Error("checks view should not be empty")
	}
}

func TestView_Comments(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenComments
	view := m.View()
	if view == "" {
		t.Error("comments view should not be empty")
	}
}

// --- PRList delegation with non-message key ---

func TestPRList_UnhandledKey(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenPRList
	// Load PRs first so key presses are meaningful
	m2, _ := m.Update(prsLoadedMsg{prs: samplePRs()})
	m = m2.(Model)
	// Press a key that doesn't trigger a known message
	m2, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	mm := m2.(Model)
	if mm.screen != ScreenPRList {
		t.Error("unhandled key should stay on PR list")
	}
}

func TestFileList_UnhandledKey(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenFileList
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	mm := m2.(Model)
	if mm.screen != ScreenFileList {
		t.Error("unhandled key should stay on file list")
	}
}

func TestDiffView_UnhandledKey(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenDiffView
	m.diffView = m.diffView.SetSize(80, 24).SetFile(sampleDiff().Files[0], 0, 2, 1)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	mm := m2.(Model)
	if mm.screen != ScreenDiffView {
		t.Error("j should stay on diff view")
	}
}

func TestChecks_UnhandledKey(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenChecks
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	mm := m2.(Model)
	if mm.screen != ScreenChecks {
		t.Error("j should stay on checks")
	}
}

func TestComments_UnhandledKey(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenComments
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	mm := m2.(Model)
	if mm.screen != ScreenComments {
		t.Error("j should stay on comments")
	}
}

// --- FileList select with out-of-bounds index ---

func TestFileList_SelectFileMsg_OutOfBounds(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenFileList
	// Load diff
	m2, _ := m.Update(diffLoadedMsg{diff: sampleDiff()})
	m = m2.(Model)
	pr := ghclient.PR{Number: 1}
	m.fileList = m.fileList.SetPR(pr)

	// Select with index out of range
	m2, _ = m.Update(filelist.SelectFileMsg{Index: 999})
	mm := m2.(Model)
	if mm.screen != ScreenDiffView {
		t.Error("should still go to diff view")
	}
}

// --- MergeMsg with undraft ---

func TestPRList_MergeMsg_Undraft(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenPRList
	_, cmd := m.Update(prlist.MergeMsg{Number: 1, Method: "squash", Undraft: true})
	if cmd == nil {
		t.Error("merge with undraft should produce a cmd")
	}
}

func TestFileList_MergeMsg_Undraft(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenFileList
	_, cmd := m.Update(filelist.MergeMsg{Number: 1, Method: "rebase", Undraft: true})
	if cmd == nil {
		t.Error("merge with undraft should produce a cmd")
	}
}

// --- Execute load* closures to cover inner code ---
// These closures call subprocess code which will error, but the code paths are exercised.

func TestLoadClosures_Init(t *testing.T) {
	m := testModel(t)
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init should return a cmd")
	}
	// Execute the batch cmd — it returns multiple messages
	// The batch function itself is what we're testing
}

func TestLoadClosures_PRSelect(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenPRList
	pr := ghclient.PR{Number: 1, Title: "Test", HeadRef: "feat", BaseRef: "main"}
	_, cmd := m.Update(prlist.SelectMsg{PR: pr})
	if cmd == nil {
		t.Fatal("should produce a batch cmd")
	}
	// The cmd is a tea.Batch of loadDiff, loadComments, loadHeadSHA
}

func TestLoadClosures_OpenBrowser(t *testing.T) {
	m := testModelWithMockClient(t, func(args ...string) (string, error) {
		return "", nil
	})
	m.screen = ScreenPRList
	_, cmd := m.Update(prlist.OpenBrowserMsg{Number: 1})
	if cmd == nil {
		t.Fatal("should produce cmd")
	}
	msg := cmd()
	if _, ok := msg.(browserOpenedMsg); !ok {
		t.Errorf("expected browserOpenedMsg, got %T", msg)
	}
}

func TestLoadClosures_ToggleDraft(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenPRList
	_, cmd := m.Update(prlist.ToggleDraftMsg{Number: 1, IsDraft: true})
	if cmd == nil {
		t.Fatal("should produce cmd")
	}
	msg := cmd()
	if _, ok := msg.(draftToggledMsg); !ok {
		t.Errorf("expected draftToggledMsg, got %T", msg)
	}
}

func TestLoadClosures_MergePR(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenPRList
	_, cmd := m.Update(prlist.MergeMsg{Number: 1, Method: "squash"})
	if cmd == nil {
		t.Fatal("should produce cmd")
	}
	msg := cmd()
	if r, ok := msg.(mergeResultMsg); !ok {
		t.Errorf("expected mergeResultMsg, got %T", msg)
	} else if r.err == nil {
		t.Error("merge should fail in test (no gh)")
	}
}

func TestLoadClosures_MergePR_FileList(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenFileList
	_, cmd := m.Update(filelist.MergeMsg{Number: 2, Method: "merge"})
	if cmd == nil {
		t.Fatal("should produce cmd")
	}
	msg := cmd()
	if _, ok := msg.(mergeResultMsg); !ok {
		t.Errorf("expected mergeResultMsg, got %T", msg)
	}
}

func TestLoadClosures_PostComment(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenDiffView
	m.diffView = m.diffView.SetHeadSHA("deadbeef")

	_, cmd := m.Update(diffview.PostCommentMsg{
		PRNumber: 1, Body: "test", Path: "a.go", Line: 5, StartLine: 3, Side: "RIGHT",
	})
	if cmd == nil {
		t.Fatal("should produce cmd")
	}
	msg := cmd()
	if r, ok := msg.(commentPostedMsg); !ok {
		t.Errorf("expected commentPostedMsg, got %T", msg)
	} else if r.err == nil {
		t.Error("post should fail in test (no gh)")
	}
}

func TestLoadClosures_ReplyComment(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenComments
	_, cmd := m.Update(comments.ReplyMsg{PRNumber: 1, InReplyToID: 10, Body: "thanks"})
	if cmd == nil {
		t.Fatal("should produce cmd")
	}
	msg := cmd()
	if r, ok := msg.(replyPostedMsg); !ok {
		t.Errorf("expected replyPostedMsg, got %T", msg)
	} else if r.err == nil {
		t.Error("reply should fail in test (no gh)")
	}
}

func TestLoadClosures_ChecksBrowserOpen(t *testing.T) {
	m := testModelWithMockClient(t, func(args ...string) (string, error) {
		return "", nil
	})
	m.screen = ScreenChecks
	_, cmd := m.Update(checks.OpenBrowserMsg{URL: "https://example.com/check"})
	if cmd == nil {
		t.Fatal("should produce cmd")
	}
	// Now safe to execute: OpenURL uses exec.Command("open") but with mock client
	// Note: OpenURL is a method on Client now, so we can test the closure path
	msg := cmd()
	if _, ok := msg.(browserOpenedMsg); !ok {
		t.Errorf("expected browserOpenedMsg, got %T", msg)
	}
}

func TestLoadClosures_Refresh(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenPRList
	_, cmd := m.Update(prlist.RefreshMsg{})
	if cmd == nil {
		t.Fatal("should produce cmd")
	}
	msg := cmd()
	if _, ok := msg.(prsLoadedMsg); !ok {
		t.Errorf("expected prsLoadedMsg, got %T", msg)
	}
}

func TestLoadClosures_RefreshChecks(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenChecks
	_, cmd := m.Update(checks.RefreshMsg{Number: 5})
	if cmd == nil {
		t.Fatal("should produce cmd")
	}
	msg := cmd()
	if _, ok := msg.(checksLoadedMsg); !ok {
		t.Errorf("expected checksLoadedMsg, got %T", msg)
	}
}

func TestLoadClosures_RefreshComments(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenComments
	_, cmd := m.Update(comments.RefreshMsg{Number: 5})
	if cmd == nil {
		t.Fatal("should produce cmd")
	}
	msg := cmd()
	if _, ok := msg.(commentsLoadedMsg); !ok {
		t.Errorf("expected commentsLoadedMsg, got %T", msg)
	}
}

func TestLoadClosures_OpenComments(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenDiffView
	pr := ghclient.PR{Number: 10}
	m.fileList = m.fileList.SetPR(pr)

	_, cmd := m.Update(diffview.OpenCommentsMsg{PRNumber: 10})
	if cmd == nil {
		t.Fatal("should produce cmd")
	}
	msg := cmd()
	if _, ok := msg.(commentsLoadedMsg); !ok {
		t.Errorf("expected commentsLoadedMsg, got %T", msg)
	}
}

func TestLoadClosures_OpenChecksFromPRList(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenPRList
	_, cmd := m.Update(prlist.OpenChecksMsg{PR: ghclient.PR{Number: 3}})
	if cmd == nil {
		t.Fatal("should produce cmd")
	}
	msg := cmd()
	if _, ok := msg.(checksLoadedMsg); !ok {
		t.Errorf("expected checksLoadedMsg, got %T", msg)
	}
}

// --- Quit from invalid screen (default branch) ---

func TestQuitFromInvalidScreen(t *testing.T) {
	m := testModel(t)
	m.screen = Screen(99) // invalid screen value
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	_ = m2
	// The default branch returns tea.Quit
	if cmd == nil {
		t.Error("q on invalid screen should quit")
	}
}

// --- Panic guards for exhaustiveness checks ---

func TestUpdate_InvalidScreen_PanicsOnDelegation(t *testing.T) {
	m := testModel(t)
	m.screen = Screen(99)
	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic on invalid screen delegation")
		}
	}()
	// A string msg is not handled by any typed case, falls through to screen delegation
	m.Update("unhandled-msg-type")
}

func TestView_InvalidScreen_Panics(t *testing.T) {
	m := testModel(t)
	m.screen = Screen(99)
	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic on invalid screen in View")
		}
	}()
	m.View()
}

func TestLoadClosures_OpenChecksFromFileList(t *testing.T) {
	m := testModel(t)
	m.screen = ScreenFileList
	_, cmd := m.Update(filelist.OpenChecksMsg{PR: ghclient.PR{Number: 7}})
	if cmd == nil {
		t.Fatal("should produce cmd")
	}
	msg := cmd()
	if _, ok := msg.(checksLoadedMsg); !ok {
		t.Errorf("expected checksLoadedMsg, got %T", msg)
	}
}

// --- Execute load closures for inner function coverage ---

func TestLoadClosures_LoadMergeSettings(t *testing.T) {
	m := testModel(t)
	cmd := m.loadMergeSettings()
	if cmd == nil {
		t.Fatal("should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(mergeSettingsMsg); !ok {
		t.Errorf("expected mergeSettingsMsg, got %T", msg)
	}
}

func TestLoadClosures_LoadDiff(t *testing.T) {
	m := testModel(t)
	pr := ghclient.PR{Number: 1, HeadRef: "feat", BaseRef: "main"}
	cmd := m.loadDiff(pr)
	if cmd == nil {
		t.Fatal("should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(diffLoadedMsg); !ok {
		t.Errorf("expected diffLoadedMsg, got %T", msg)
	}
}

func TestLoadClosures_LoadHeadSHA(t *testing.T) {
	m := testModel(t)
	cmd := m.loadHeadSHA(1)
	if cmd == nil {
		t.Fatal("should return a cmd")
	}
	msg := cmd()
	// Without gh CLI, this will return an error
	sha, ok := msg.(headSHALoadedMsg)
	if !ok {
		t.Errorf("expected headSHALoadedMsg, got %T", msg)
	}
	// The closure has err != nil branch and err == nil branch
	// In test environment (no gh CLI), err should be non-nil
	if sha.err == nil {
		t.Error("expected error without gh CLI")
	}
}

func TestLoadClosures_LoadComments(t *testing.T) {
	m := testModel(t)
	cmd := m.loadComments(1)
	if cmd == nil {
		t.Fatal("should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(commentsLoadedMsg); !ok {
		t.Errorf("expected commentsLoadedMsg, got %T", msg)
	}
}

func TestLoadClosures_LoadPRs(t *testing.T) {
	m := testModel(t)
	cmd := m.loadPRs()
	if cmd == nil {
		t.Fatal("should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(prsLoadedMsg); !ok {
		t.Errorf("expected prsLoadedMsg, got %T", msg)
	}
}

func TestLoadClosures_LoadChecks(t *testing.T) {
	m := testModel(t)
	cmd := m.loadChecks(1)
	if cmd == nil {
		t.Fatal("should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(checksLoadedMsg); !ok {
		t.Errorf("expected checksLoadedMsg, got %T", msg)
	}
}

// --- Tests with mock client to cover success paths in closures ---

func TestLoadClosures_LoadHeadSHA_Success(t *testing.T) {
	m := testModelWithMockClient(t, func(args ...string) (string, error) {
		// pr view N --json headRefOid -q .headRefOid
		return "abc123def456\n", nil
	})
	cmd := m.loadHeadSHA(1)
	msg := cmd()
	sha, ok := msg.(headSHALoadedMsg)
	if !ok {
		t.Fatalf("expected headSHALoadedMsg, got %T", msg)
	}
	if sha.err != nil {
		t.Errorf("unexpected error: %v", sha.err)
	}
	if sha.sha != "abc123def456" {
		t.Errorf("sha = %q, want %q", sha.sha, "abc123def456")
	}
}

func TestLoadClosures_LoadDiff_Success(t *testing.T) {
	diffText := "diff --git a/f.go b/f.go\nindex 1234..5678 100644\n--- a/f.go\n+++ b/f.go\n@@ -1,1 +1,2 @@\n line1\n+added\n"
	m := testModelWithMockClient(t, func(args ...string) (string, error) {
		return diffText, nil
	})
	pr := ghclient.PR{Number: 1, HeadRef: "feat", BaseRef: "main"}
	cmd := m.loadDiff(pr)
	msg := cmd()
	d, ok := msg.(diffLoadedMsg)
	if !ok {
		t.Fatalf("expected diffLoadedMsg, got %T", msg)
	}
	if d.err != nil {
		t.Errorf("unexpected error: %v", d.err)
	}
	if len(d.diff.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(d.diff.Files))
	}
}

func TestLoadClosures_LoadPRs_Success(t *testing.T) {
	m := testModelWithMockClient(t, func(args ...string) (string, error) {
		return `[{"number":1,"title":"Test PR","author":{"login":"user"},"state":"OPEN","headRefName":"feat","baseRefName":"main"}]`, nil
	})
	cmd := m.loadPRs()
	msg := cmd()
	p, ok := msg.(prsLoadedMsg)
	if !ok {
		t.Fatalf("expected prsLoadedMsg, got %T", msg)
	}
	if p.err != nil {
		t.Errorf("unexpected error: %v", p.err)
	}
	if len(p.prs) != 1 {
		t.Errorf("expected 1 PR, got %d", len(p.prs))
	}
}

func TestLoadClosures_LoadChecks_Success(t *testing.T) {
	m := testModelWithMockClient(t, func(args ...string) (string, error) {
		return `[{"name":"CI","state":"completed","bucket":"pass"}]`, nil
	})
	cmd := m.loadChecks(1)
	msg := cmd()
	c, ok := msg.(checksLoadedMsg)
	if !ok {
		t.Fatalf("expected checksLoadedMsg, got %T", msg)
	}
	if c.err != nil {
		t.Errorf("unexpected error: %v", c.err)
	}
	if len(c.checks) != 1 {
		t.Errorf("expected 1 check, got %d", len(c.checks))
	}
}
