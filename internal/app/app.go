package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/LumaKernel/ghpr-tui/internal/ghclient"
	"github.com/LumaKernel/ghpr-tui/internal/state"
	"github.com/LumaKernel/ghpr-tui/internal/ui/checks"
	"github.com/LumaKernel/ghpr-tui/internal/ui/comments"
	"github.com/LumaKernel/ghpr-tui/internal/ui/diffview"
	"github.com/LumaKernel/ghpr-tui/internal/ui/filelist"
	"github.com/LumaKernel/ghpr-tui/internal/ui/prlist"
	"github.com/LumaKernel/ghpr-tui/internal/ui/styles"
)

// Screen represents the current active screen.
type Screen int

const (
	ScreenPRList Screen = iota
	ScreenFileList
	ScreenDiffView
	ScreenChecks
	ScreenComments
	screenSentinel // exhaustive check guard
)

// Messages for async data loading
type prsLoadedMsg struct {
	prs []ghclient.PR
	err error
}

type diffLoadedMsg struct {
	diff ghclient.ParsedDiff
	err  error
}

type checksLoadedMsg struct {
	checks []ghclient.Check
	err    error
}

type browserOpenedMsg struct {
	err error
}

type mergeResultMsg struct {
	number int
	err    error
}

type draftToggledMsg struct {
	err error
}

type mergeSettingsMsg struct {
	methods []string
}

type commentsLoadedMsg struct {
	comments []ghclient.ReviewComment
	err      error
}

type headSHALoadedMsg struct {
	sha string
	err error
}

type commentPostedMsg struct {
	number int
	err    error
}

type replyPostedMsg struct {
	number int
	err    error
}

// Model is the top-level app model.
type Model struct {
	screen     Screen
	prevScreen Screen // for back navigation from checks/comments
	client     *ghclient.Client
	repo       string
	store      *state.Store
	prList     prlist.Model
	fileList   filelist.Model
	diffView   diffview.Model
	checks     checks.Model
	comments   comments.Model
	width      int
	height     int
	// Cached data for passing between screens
	currentPRNumber int
}

// New creates a new app model.
func New(repo string, client *ghclient.Client, store *state.Store) Model {
	return Model{
		screen:   ScreenPRList,
		client:   client,
		repo:     repo,
		store:    store,
		prList:   prlist.New(repo, store),
		fileList: filelist.New(repo, store),
		diffView: diffview.New(repo, store),
		checks:   checks.New(),
		comments: comments.New(),
	}
}

// Init starts the app by loading PRs and merge settings concurrently.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadPRs(), m.loadMergeSettings())
}

func (m Model) loadMergeSettings() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		settings, _ := client.GetMergeSettings()
		return mergeSettingsMsg{methods: settings.AllowedMethods()}
	}
}

func (m Model) loadPRs() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		prs, err := client.ListPRs(50)
		return prsLoadedMsg{prs: prs, err: err}
	}
}

func (m Model) loadDiff(pr ghclient.PR) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		diff, err := client.GetParsedDiff(pr)
		return diffLoadedMsg{diff: diff, err: err}
	}
}

func (m Model) loadChecks(number int) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		chks, err := client.GetChecks(number)
		return checksLoadedMsg{checks: chks, err: err}
	}
}

func (m Model) loadComments(number int) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		cs, err := client.GetReviewComments(number)
		return commentsLoadedMsg{comments: cs, err: err}
	}
}

func (m Model) loadHeadSHA(number int) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		sha, err := client.GetPRHeadSHA(number)
		if err != nil {
			return headSHALoadedMsg{err: err}
		}
		return headSHALoadedMsg{sha: sha}
	}
}

// Update is the main update loop.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.prList = m.prList.SetSize(msg.Width, msg.Height)
		m.fileList = m.fileList.SetSize(msg.Width, msg.Height)
		m.diffView = m.diffView.SetSize(msg.Width, msg.Height)
		m.checks = m.checks.SetSize(msg.Width, msg.Height)
		m.comments = m.comments.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		// Don't intercept q/ctrl+c when diffview is in input mode
		if m.screen == ScreenDiffView && m.diffView.IsInputMode() {
			break // fall through to screen delegation
		}
		switch msg.String() {
		case "q", "ctrl+c":
			if m.screen == ScreenPRList {
				return m, tea.Quit
			}
			switch m.screen {
			case ScreenDiffView:
				m.screen = ScreenFileList
				return m, nil
			case ScreenFileList:
				m.screen = ScreenPRList
				return m, nil
			case ScreenChecks:
				m.screen = m.prevScreen
				return m, nil
			case ScreenComments:
				m.screen = m.prevScreen
				return m, nil
			default:
				return m, tea.Quit
			}
		}

	case prsLoadedMsg:
		if msg.err != nil {
			m.prList = m.prList.SetError(msg.err)
		} else {
			m.prList = m.prList.SetPRs(msg.prs)
		}
		return m, nil

	case diffLoadedMsg:
		if msg.err != nil {
			m.fileList = m.fileList.SetError(msg.err)
		} else {
			m.fileList = m.fileList.SetDiff(msg.diff)
		}
		return m, nil

	case checksLoadedMsg:
		if msg.err != nil {
			m.checks = m.checks.SetError(msg.err)
		} else {
			m.checks = m.checks.SetChecks(msg.checks)
		}
		return m, nil

	case commentsLoadedMsg:
		if msg.err != nil {
			if m.screen == ScreenComments {
				m.comments = m.comments.SetError(msg.err)
			}
		} else {
			threads := ghclient.GroupCommentThreads(msg.comments)
			m.comments = m.comments.SetComments(threads)
			m.diffView = m.diffView.SetComments(threads)
		}
		return m, nil

	case headSHALoadedMsg:
		if msg.err == nil {
			m.diffView = m.diffView.SetHeadSHA(msg.sha)
		}
		return m, nil

	case commentPostedMsg:
		if msg.err != nil {
			m.diffView = m.diffView.SetStatusMsg(styles.Removed.Render(fmt.Sprintf("Comment failed: %v", msg.err)))
		} else {
			m.diffView = m.diffView.SetStatusMsg(styles.Reviewed.Render("Comment posted!"))
			// Refresh comments
			return m, m.loadComments(msg.number)
		}
		return m, nil

	case replyPostedMsg:
		if msg.err != nil {
			m.comments = m.comments.SetError(msg.err)
		} else {
			// Refresh comments
			return m, m.loadComments(msg.number)
		}
		return m, nil

	case mergeSettingsMsg:
		m.prList = m.prList.SetAllowedMergeMethods(msg.methods)
		m.fileList = m.fileList.SetAllowedMergeMethods(msg.methods)
		return m, nil

	case browserOpenedMsg:
		return m, nil

	case draftToggledMsg:
		if msg.err != nil {
			m.prList = m.prList.SetMergeResult(styles.Removed.Render(fmt.Sprintf("Draft toggle failed: %v", msg.err)))
		}
		return m, m.loadPRs()

	case mergeResultMsg:
		resultMsg := ""
		if msg.err != nil {
			resultMsg = styles.Removed.Render(fmt.Sprintf("Merge failed: %v", msg.err))
		} else {
			resultMsg = styles.Reviewed.Render(fmt.Sprintf("Merged #%d successfully!", msg.number))
		}
		m.prList = m.prList.SetMergeResult(resultMsg)
		m.fileList = m.fileList.SetMergeResult(resultMsg)
		if msg.err == nil {
			m.screen = ScreenPRList
			return m, m.loadPRs()
		}
		return m, nil
	}

	// Delegate to active screen
	switch m.screen {
	case ScreenPRList:
		return m.updatePRList(msg)
	case ScreenFileList:
		return m.updateFileList(msg)
	case ScreenDiffView:
		return m.updateDiffView(msg)
	case ScreenChecks:
		return m.updateChecks(msg)
	case ScreenComments:
		return m.updateComments(msg)
	default:
		panic(fmt.Sprintf("unhandled screen: %d", m.screen))
	}
}

func (m Model) updatePRList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.prList, cmd = m.prList.Update(msg)

	switch msg.(type) {
	case prlist.SelectMsg:
		selectMsg := msg.(prlist.SelectMsg)
		m.screen = ScreenFileList
		m.fileList = m.fileList.SetPR(selectMsg.PR)
		m.currentPRNumber = selectMsg.PR.Number
		return m, tea.Batch(
			m.loadDiff(selectMsg.PR),
			m.loadComments(selectMsg.PR.Number),
			m.loadHeadSHA(selectMsg.PR.Number),
		)
	case prlist.RefreshMsg:
		m.prList = m.prList.SetLoading(true)
		return m, m.loadPRs()
	case prlist.OpenBrowserMsg:
		openMsg := msg.(prlist.OpenBrowserMsg)
		client := m.client
		number := openMsg.Number
		return m, func() tea.Msg {
			err := client.OpenInBrowser(number)
			return browserOpenedMsg{err: err}
		}
	case prlist.MergeMsg:
		mergeMsg := msg.(prlist.MergeMsg)
		client := m.client
		number := mergeMsg.Number
		method := mergeMsg.Method
		undraft := mergeMsg.Undraft
		return m, func() tea.Msg {
			err := client.MergePR(number, method, undraft)
			return mergeResultMsg{number: number, err: err}
		}
	case prlist.ToggleDraftMsg:
		toggleMsg := msg.(prlist.ToggleDraftMsg)
		client := m.client
		number := toggleMsg.Number
		isDraft := toggleMsg.IsDraft
		return m, func() tea.Msg {
			err := client.ToggleDraft(number, isDraft)
			return draftToggledMsg{err: err}
		}
	case prlist.OpenChecksMsg:
		checksMsg := msg.(prlist.OpenChecksMsg)
		m.screen = ScreenChecks
		m.prevScreen = ScreenPRList
		m.checks = m.checks.SetPR(checksMsg.PR)
		return m, m.loadChecks(checksMsg.PR.Number)
	}

	return m, cmd
}

func (m Model) updateFileList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.fileList, cmd = m.fileList.Update(msg)

	switch msg.(type) {
	case filelist.SelectFileMsg:
		selectMsg := msg.(filelist.SelectFileMsg)
		m.screen = ScreenDiffView
		diff := m.fileList.Diff()
		if selectMsg.Index < len(diff.Files) {
			m.diffView = m.diffView.SetFile(
				diff.Files[selectMsg.Index],
				selectMsg.Index,
				len(diff.Files),
				m.fileList.PR().Number,
			)
		}
		return m, nil
	case filelist.BackMsg:
		m.screen = ScreenPRList
		return m, nil
	case filelist.MergeMsg:
		mergeMsg := msg.(filelist.MergeMsg)
		client := m.client
		number := mergeMsg.Number
		method := mergeMsg.Method
		undraft := mergeMsg.Undraft
		return m, func() tea.Msg {
			err := client.MergePR(number, method, undraft)
			return mergeResultMsg{number: number, err: err}
		}
	case filelist.OpenChecksMsg:
		checksMsg := msg.(filelist.OpenChecksMsg)
		m.screen = ScreenChecks
		m.prevScreen = ScreenFileList
		m.checks = m.checks.SetPR(checksMsg.PR)
		return m, m.loadChecks(checksMsg.PR.Number)
	}

	return m, cmd
}

func (m Model) updateDiffView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.diffView, cmd = m.diffView.Update(msg)

	switch msg.(type) {
	case diffview.BackMsg:
		m.screen = ScreenFileList
		return m, nil
	case diffview.NextFileMsg:
		diff := m.fileList.Diff()
		nextIdx := m.diffView.FileIndex() + 1
		if nextIdx >= len(diff.Files) {
			m.screen = ScreenFileList
			return m, nil
		}
		m.diffView = m.diffView.SetFile(
			diff.Files[nextIdx],
			nextIdx,
			len(diff.Files),
			m.fileList.PR().Number,
		)
		return m, nil

	case diffview.PrevFileMsg:
		diff := m.fileList.Diff()
		prevIdx := m.diffView.FileIndex() - 1
		if prevIdx < 0 {
			return m, nil
		}
		m.diffView = m.diffView.SetFile(
			diff.Files[prevIdx],
			prevIdx,
			len(diff.Files),
			m.fileList.PR().Number,
		)
		return m, nil

	case diffview.OpenCommentsMsg:
		openMsg := msg.(diffview.OpenCommentsMsg)
		m.screen = ScreenComments
		m.prevScreen = ScreenDiffView
		m.comments = m.comments.SetPR(m.fileList.PR())
		return m, m.loadComments(openMsg.PRNumber)

	case diffview.PostCommentMsg:
		postMsg := msg.(diffview.PostCommentMsg)
		client := m.client
		dv := m.diffView
		return m, func() tea.Msg {
			err := client.CreateReviewComment(
				postMsg.PRNumber,
				postMsg.Body,
				postMsg.Path,
				dv.HeadSHA(),
				postMsg.Line,
				postMsg.StartLine,
				postMsg.Side,
			)
			return commentPostedMsg{number: postMsg.PRNumber, err: err}
		}
	}

	return m, cmd
}

func (m Model) updateChecks(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.checks, cmd = m.checks.Update(msg)

	switch msg.(type) {
	case checks.BackMsg:
		m.screen = m.prevScreen
		return m, nil
	case checks.OpenBrowserMsg:
		openMsg := msg.(checks.OpenBrowserMsg)
		client := m.client
		url := openMsg.URL
		return m, func() tea.Msg {
			_ = client.OpenURL(url)
			return browserOpenedMsg{}
		}
	case checks.RefreshMsg:
		refreshMsg := msg.(checks.RefreshMsg)
		return m, m.loadChecks(refreshMsg.Number)
	}

	return m, cmd
}

func (m Model) updateComments(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.comments, cmd = m.comments.Update(msg)

	switch msg.(type) {
	case comments.BackMsg:
		m.screen = m.prevScreen
		return m, nil
	case comments.RefreshMsg:
		refreshMsg := msg.(comments.RefreshMsg)
		return m, m.loadComments(refreshMsg.Number)
	case comments.JumpToFileMsg:
		// Navigate back to diffview at the specified file
		jumpMsg := msg.(comments.JumpToFileMsg)
		diff := m.fileList.Diff()
		for i, f := range diff.Files {
			if f.FilePath() == jumpMsg.Path || f.NewPath == jumpMsg.Path || f.OldPath == jumpMsg.Path {
				m.screen = ScreenDiffView
				m.diffView = m.diffView.SetFile(f, i, len(diff.Files), m.fileList.PR().Number)
				return m, nil
			}
		}
		// File not found, just go back
		m.screen = m.prevScreen
		return m, nil
	case comments.ReplyMsg:
		replyMsg := msg.(comments.ReplyMsg)
		client := m.client
		return m, func() tea.Msg {
			err := client.ReplyToComment(replyMsg.PRNumber, replyMsg.InReplyToID, replyMsg.Body)
			return replyPostedMsg{number: replyMsg.PRNumber, err: err}
		}
	}

	return m, cmd
}

// View renders the current screen.
func (m Model) View() string {
	switch m.screen {
	case ScreenPRList:
		return m.prList.View()
	case ScreenFileList:
		return m.fileList.View()
	case ScreenDiffView:
		return m.diffView.View()
	case ScreenChecks:
		return m.checks.View()
	case ScreenComments:
		return m.comments.View()
	default:
		panic(fmt.Sprintf("unhandled screen: %d", m.screen))
	}
}
