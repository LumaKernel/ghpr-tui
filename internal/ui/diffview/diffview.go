package diffview

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/LumaKernel/ghpr-tui/internal/ghclient"
	"github.com/LumaKernel/ghpr-tui/internal/state"
	"github.com/LumaKernel/ghpr-tui/internal/ui/styles"
)

// Mode represents the diffview interaction mode.
type Mode int

const (
	ModeNormal  Mode = iota
	ModeVisual       // visual line selection
	ModeComment      // typing a comment
	modeSentinel     // exhaustive guard
)

// BackMsg requests going back to file list.
type BackMsg struct{}

// NextFileMsg requests the next file.
type NextFileMsg struct{}

// PrevFileMsg requests the previous file.
type PrevFileMsg struct{}

// OpenCommentsMsg requests opening the comments list screen.
type OpenCommentsMsg struct {
	PRNumber int
}

// PostCommentMsg requests posting a comment via the app layer.
type PostCommentMsg struct {
	PRNumber  int
	Body      string
	Path      string
	Line      int
	StartLine int
	Side      string
}

// Model is the diff view screen model.
type Model struct {
	file       ghclient.FileDiff
	fileIndex  int
	fileCount  int
	prNumber   int
	repo       string
	store      *state.Store
	lines      []renderedLine
	scroll     int
	width      int
	height     int
	hunkStarts []int // indices into lines where hunks begin

	// Visual selection mode
	mode     Mode
	cursor   int // line index for visual mode cursor
	selStart int // line index where visual selection started

	// Comment input mode
	commentText []string // multi-line comment buffer
	commentRow  int      // cursor row in comment text
	commentCol  int      // cursor col in comment text

	// Inline comments
	comments     []ghclient.CommentThread
	showComments bool
	headSHA      string

	// Status message
	statusMsg string
}

type renderedLine struct {
	content  string
	diffLine *ghclient.DiffLine // nil for hunk headers and comment lines
	isHunk   bool
	isComment bool // inline comment display line
}

// FileIndex returns the current file index.
func (m Model) FileIndex() int {
	return m.fileIndex
}

// IsInputMode returns true when the diffview is in a mode that captures all keys.
func (m Model) IsInputMode() bool {
	return m.mode == ModeVisual || m.mode == ModeComment
}

// CurrentMode returns the current interaction mode.
func (m Model) CurrentMode() Mode {
	return m.mode
}

// New creates a new diff view model.
func New(repo string, store *state.Store) Model {
	return Model{
		repo:  repo,
		store: store,
	}
}

// SetFile sets the file to display.
func (m Model) SetFile(file ghclient.FileDiff, index, count, prNumber int) Model {
	m.file = file
	m.fileIndex = index
	m.fileCount = count
	m.prNumber = prNumber
	m.scroll = 0
	m.mode = ModeNormal
	m.statusMsg = ""
	m.rebuildLines()
	return m
}

// SetSize updates terminal dimensions.
func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h
	if m.file.NewPath != "" || m.file.OldPath != "" {
		m.rebuildLines()
	}
	return m
}

// SetComments sets the inline comment threads.
func (m Model) SetComments(threads []ghclient.CommentThread) Model {
	m.comments = threads
	m.rebuildLines()
	return m
}

// SetHeadSHA sets the PR head commit SHA for posting comments.
func (m Model) SetHeadSHA(sha string) Model {
	m.headSHA = sha
	return m
}

// HeadSHA returns the PR head commit SHA.
func (m Model) HeadSHA() string {
	return m.headSHA
}

// SetStatusMsg sets a temporary status message.
func (m Model) SetStatusMsg(msg string) Model {
	m.statusMsg = msg
	return m
}

func (m *Model) rebuildLines() {
	fileThreads := ghclient.CommentsForFile(m.comments, m.file.FilePath())
	m.lines = renderDiffLines(m.file, m.width, fileThreads, m.showComments)
	m.hunkStarts = findHunkStarts(m.lines)
}

func (m Model) viewportHeight() int {
	h := m.height - 5 // header + status
	if h < 1 {
		return 10
	}
	return h
}

// Update handles input for diff view.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch m.mode {
	case ModeNormal:
		return m.updateNormal(msg)
	case ModeVisual:
		return m.updateVisual(msg)
	case ModeComment:
		return m.updateComment(msg)
	default:
		panic(fmt.Sprintf("unhandled diffview mode: %d", m.mode))
	}
}

func (m Model) updateNormal(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.statusMsg = ""
		maxScroll := max(0, len(m.lines)-m.viewportHeight())
		switch msg.String() {
		case "j", "down", "ctrl+n":
			if m.scroll < maxScroll {
				m.scroll++
			}
		case "k", "up", "ctrl+p":
			if m.scroll > 0 {
				m.scroll--
			}
		case "d", "ctrl+d":
			m.scroll = min(m.scroll+m.viewportHeight()/2, maxScroll)
		case "u", "ctrl+u":
			m.scroll = max(0, m.scroll-m.viewportHeight()/2)
		case "f", "ctrl+f", "pgdown":
			m.scroll = min(m.scroll+m.viewportHeight(), maxScroll)
		case "b", "ctrl+b", "pgup":
			m.scroll = max(0, m.scroll-m.viewportHeight())
		case "ctrl+e":
			if m.scroll < maxScroll {
				m.scroll++
			}
		case "ctrl+y":
			if m.scroll > 0 {
				m.scroll--
			}
		case "g", "home":
			m.scroll = 0
		case "G", "end":
			m.scroll = maxScroll
		case "n":
			m.scroll = m.nextHunk()
		case "N":
			m.scroll = m.prevHunk()
		case "]", "tab":
			return m, func() tea.Msg { return NextFileMsg{} }
		case "[", "shift+tab":
			return m, func() tea.Msg { return PrevFileMsg{} }
		case "space", " ":
			m.store.MarkFileReviewed(m.repo, m.prNumber, m.file.FilePath())
			_ = m.store.Save()
			return m, func() tea.Msg { return NextFileMsg{} }
		case "V":
			// Enter visual line selection mode
			m.mode = ModeVisual
			m.cursor = m.scroll
			m.selStart = m.cursor
		case "c":
			// Toggle inline comments
			m.showComments = !m.showComments
			m.rebuildLines()
		case "C":
			// Open comments list screen
			number := m.prNumber
			return m, func() tea.Msg { return OpenCommentsMsg{PRNumber: number} }
		case "esc", "backspace":
			return m, func() tea.Msg { return BackMsg{} }
		}
	}
	return m, nil
}

func (m Model) updateVisual(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		maxLine := max(0, len(m.lines)-1)
		switch msg.String() {
		case "j", "down":
			if m.cursor < maxLine {
				m.cursor++
				// Scroll to follow cursor
				if m.cursor >= m.scroll+m.viewportHeight() {
					m.scroll = m.cursor - m.viewportHeight() + 1
				}
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.scroll {
					m.scroll = m.cursor
				}
			}
		case "g", "home":
			m.cursor = 0
			m.scroll = 0
		case "G", "end":
			m.cursor = maxLine
			maxScroll := max(0, len(m.lines)-m.viewportHeight())
			m.scroll = maxScroll
		case "V", "esc":
			// Cancel visual selection
			m.mode = ModeNormal
		case "enter", "c":
			// Open comment input for selected range
			startLine, endLine, side := m.selectionLineRange()
			if startLine > 0 {
				m.mode = ModeComment
				m.commentText = []string{""}
				m.commentRow = 0
				m.commentCol = 0
				// Store the computed range (selStart/cursor already set)
				_ = startLine
				_ = endLine
				_ = side
			} else {
				m.statusMsg = "Cannot comment on this selection"
				m.mode = ModeNormal
			}
		}
	}
	return m, nil
}

func (m Model) updateComment(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.mode = ModeNormal
			m.commentText = nil
		case "ctrl+s":
			// Submit comment
			body := strings.Join(m.commentText, "\n")
			body = strings.TrimSpace(body)
			if body == "" {
				m.statusMsg = "Comment body is empty"
				return m, nil
			}
			startLine, endLine, side := m.selectionLineRange()
			if startLine <= 0 {
				m.statusMsg = "Invalid line range"
				m.mode = ModeNormal
				return m, nil
			}
			m.mode = ModeNormal
			m.statusMsg = "Posting comment..."
			path := m.file.FilePath()
			number := m.prNumber
			return m, func() tea.Msg {
				return PostCommentMsg{
					PRNumber:  number,
					Body:      body,
					Path:      path,
					Line:      endLine,
					StartLine: startLine,
					Side:      side,
				}
			}
		case "enter":
			// Insert newline
			tail := m.commentText[m.commentRow][m.commentCol:]
			m.commentText[m.commentRow] = m.commentText[m.commentRow][:m.commentCol]
			newLines := make([]string, 0, len(m.commentText)+1)
			newLines = append(newLines, m.commentText[:m.commentRow+1]...)
			newLines = append(newLines, tail)
			newLines = append(newLines, m.commentText[m.commentRow+1:]...)
			m.commentText = newLines
			m.commentRow++
			m.commentCol = 0
		case "backspace":
			if m.commentCol > 0 {
				line := m.commentText[m.commentRow]
				m.commentText[m.commentRow] = line[:m.commentCol-1] + line[m.commentCol:]
				m.commentCol--
			} else if m.commentRow > 0 {
				// Join with previous line
				prevLen := len(m.commentText[m.commentRow-1])
				m.commentText[m.commentRow-1] += m.commentText[m.commentRow]
				m.commentText = append(m.commentText[:m.commentRow], m.commentText[m.commentRow+1:]...)
				m.commentRow--
				m.commentCol = prevLen
			}
		case "left":
			if m.commentCol > 0 {
				m.commentCol--
			}
		case "right":
			if m.commentCol < len(m.commentText[m.commentRow]) {
				m.commentCol++
			}
		case "up":
			if m.commentRow > 0 {
				m.commentRow--
				m.commentCol = min(m.commentCol, len(m.commentText[m.commentRow]))
			}
		case "down":
			if m.commentRow < len(m.commentText)-1 {
				m.commentRow++
				m.commentCol = min(m.commentCol, len(m.commentText[m.commentRow]))
			}
		default:
			// Insert printable characters
			ch := msg.String()
			if len(ch) == 1 && ch[0] >= 32 {
				line := m.commentText[m.commentRow]
				m.commentText[m.commentRow] = line[:m.commentCol] + ch + line[m.commentCol:]
				m.commentCol++
			}
		}
	}
	return m, nil
}

// selectionLineRange returns the GitHub API line numbers and side for the current visual selection.
// Returns (startLine, endLine, side). startLine=0 means invalid selection.
func (m Model) selectionLineRange() (int, int, string) {
	lo := min(m.selStart, m.cursor)
	hi := max(m.selStart, m.cursor)

	var startNum, endNum int
	side := "RIGHT"

	for i := lo; i <= hi; i++ {
		if i >= len(m.lines) {
			break
		}
		dl := m.lines[i].diffLine
		if dl == nil {
			continue
		}
		num := dl.NewNum
		if dl.Type == ghclient.LineRemoved {
			num = dl.OldNum
			side = "LEFT"
		}
		if num > 0 {
			if startNum == 0 {
				startNum = num
			}
			endNum = num
		}
	}

	return startNum, endNum, side
}

func (m Model) nextHunk() int {
	for _, hs := range m.hunkStarts {
		if hs > m.scroll {
			return hs
		}
	}
	return m.scroll
}

func (m Model) prevHunk() int {
	prev := 0
	for _, hs := range m.hunkStarts {
		if hs >= m.scroll {
			break
		}
		prev = hs
	}
	return prev
}

// View renders the diff view screen.
func (m Model) View() string {
	var b strings.Builder

	path := m.file.FilePath()
	isReviewed := m.store.IsFileReviewed(m.repo, m.prNumber, path)

	// Header
	reviewMark := ""
	if isReviewed {
		reviewMark = styles.Reviewed.Render(" ✓")
	}

	modeIndicator := ""
	switch m.mode {
	case ModeVisual:
		modeIndicator = styles.Unread.Render(" [VISUAL]")
	case ModeComment:
		modeIndicator = styles.CommentMarker.Render(" [COMMENT]")
	case ModeNormal:
		// no indicator
	default:
		panic(fmt.Sprintf("unhandled mode in View: %d", m.mode))
	}

	commentsIndicator := ""
	if m.showComments && len(m.comments) > 0 {
		commentsIndicator = styles.CommentMarker.Render(fmt.Sprintf(" 💬%d", len(m.comments)))
	}

	header := fmt.Sprintf(" %s%s  (%d/%d)%s%s ", path, reviewMark, m.fileIndex+1, m.fileCount, modeIndicator, commentsIndicator)
	b.WriteString(styles.Title.Render(header))
	b.WriteString("\n")

	stats := fmt.Sprintf("  %s %s",
		styles.Added.Render(fmt.Sprintf("+%d", m.file.Additions())),
		styles.Removed.Render(fmt.Sprintf("-%d", m.file.Deletions())),
	)
	b.WriteString(stats)
	b.WriteString("\n")

	if m.file.IsBinary {
		b.WriteString("\n  Binary file")
		return b.String()
	}

	// Diff content
	vpHeight := m.viewportHeight()

	// Reserve space for comment input if in comment mode
	commentInputHeight := 0
	if m.mode == ModeComment {
		commentInputHeight = len(m.commentText) + 4 // border + help
		vpHeight -= commentInputHeight
		if vpHeight < 3 {
			vpHeight = 3
		}
	}

	endIdx := m.scroll + vpHeight
	if endIdx > len(m.lines) {
		endIdx = len(m.lines)
	}

	// Visual selection range
	selLo, selHi := -1, -1
	if m.mode == ModeVisual || m.mode == ModeComment {
		selLo = min(m.selStart, m.cursor)
		selHi = max(m.selStart, m.cursor)
	}

	for i := m.scroll; i < endIdx; i++ {
		line := m.lines[i].content
		if selLo >= 0 && i >= selLo && i <= selHi {
			// Highlight selected line
			line = styles.VisualSelect.Width(m.width).Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Comment input overlay
	if m.mode == ModeComment {
		b.WriteString("\n")
		startLine, endLine, _ := m.selectionLineRange()
		b.WriteString(styles.CommentMarker.Render(fmt.Sprintf("  Comment on lines %d-%d:", startLine, endLine)))
		b.WriteString("\n")
		for i, line := range m.commentText {
			cursor := " "
			if i == m.commentRow {
				cursor = ">"
			}
			b.WriteString(fmt.Sprintf("  %s %s\n", cursor, line))
		}
		b.WriteString(styles.Help.Render("  ctrl+s:submit  esc:cancel  enter:newline"))
		b.WriteString("\n")
	}

	// Status message
	if m.statusMsg != "" {
		b.WriteString(fmt.Sprintf("  %s\n", m.statusMsg))
	}

	// Scroll indicator
	scrollPct := ""
	totalVP := m.viewportHeight()
	if len(m.lines) > totalVP {
		pct := float64(m.scroll) / float64(len(m.lines)-totalVP) * 100
		scrollPct = fmt.Sprintf(" %d%%", int(pct))
	}

	// Help — mode is already validated by the modeIndicator switch above,
	// so no default panic needed here.
	var helpText string
	switch m.mode {
	case ModeNormal:
		helpText = fmt.Sprintf(
			"  j/k:scroll  d/u:half-page  n/N:hunk  [/]:file  V:visual  c:comments  C:list  space:reviewed  esc:back%s", scrollPct)
	case ModeVisual:
		helpText = fmt.Sprintf(
			"  j/k:extend  enter/c:comment  V/esc:cancel  g/G:jump%s", scrollPct)
	case ModeComment:
		helpText = "  ctrl+s:submit  esc:cancel"
	}
	b.WriteString(styles.Help.Render(helpText))

	return b.String()
}

func renderDiffLines(file ghclient.FileDiff, width int, threads []ghclient.CommentThread, showComments bool) []renderedLine {
	var lines []renderedLine

	// Build line-to-comment map for inline display
	type commentKey struct {
		line int
		side string
	}
	commentMap := make(map[commentKey][]ghclient.CommentThread)
	if showComments {
		for _, t := range threads {
			key := commentKey{line: t.Root.Line, side: t.Root.Side}
			commentMap[key] = append(commentMap[key], t)
		}
	}

	for _, hunk := range file.Hunks {
		// Hunk header
		lines = append(lines, renderedLine{
			content: styles.HunkHeader.Render(hunk.Header),
			isHunk:  true,
		})

		for idx := range hunk.Lines {
			dl := &hunk.Lines[idx]
			oldNum := "     "
			newNum := "     "
			prefix := " "

			switch dl.Type {
			case ghclient.LineAdded:
				newNum = fmt.Sprintf("%4d ", dl.NewNum)
				prefix = "+"
			case ghclient.LineRemoved:
				oldNum = fmt.Sprintf("%4d ", dl.OldNum)
				prefix = "-"
			case ghclient.LineContext:
				if dl.OldNum > 0 {
					oldNum = fmt.Sprintf("%4d ", dl.OldNum)
				}
				if dl.NewNum > 0 {
					newNum = fmt.Sprintf("%4d ", dl.NewNum)
				}
			}

			gutter := styles.LineNum.Render(oldNum) + styles.LineNum.Render(newNum)
			content := prefix + dl.Content

			var styled string
			switch dl.Type {
			case ghclient.LineAdded:
				styled = styles.AddedBg.Render(content)
			case ghclient.LineRemoved:
				styled = styles.RemovedBg.Render(content)
			case ghclient.LineContext:
				styled = styles.DiffContext.Render(content)
			}

			lines = append(lines, renderedLine{
				content:  gutter + styled,
				diffLine: dl,
			})

			// Insert inline comments after this line
			if showComments {
				lineNum := dl.NewNum
				side := "RIGHT"
				if dl.Type == ghclient.LineRemoved {
					lineNum = dl.OldNum
					side = "LEFT"
				}
				key := commentKey{line: lineNum, side: side}
				if threads, ok := commentMap[key]; ok {
					for _, t := range threads {
						lines = append(lines, renderCommentThread(t, width)...)
					}
					delete(commentMap, key)
				}
			}
		}
	}

	return lines
}

func renderCommentThread(thread ghclient.CommentThread, width int) []renderedLine {
	var lines []renderedLine

	renderOneComment := func(c ghclient.ReviewComment, prefix string) {
		body := strings.ReplaceAll(c.Body, "\n", " ")
		maxLen := width - 20
		if maxLen < 30 {
			maxLen = 30
		}
		if len(body) > maxLen {
			body = body[:maxLen-1] + "…"
		}
		content := fmt.Sprintf("  %s %s %s",
			styles.CommentMarker.Render(prefix+"💬"),
			styles.CommentAuthor.Render("@"+c.User.Login),
			styles.CommentBody.Render(body),
		)
		lines = append(lines, renderedLine{content: content, isComment: true})
	}

	renderOneComment(thread.Root, "")
	for _, reply := range thread.Replies {
		renderOneComment(reply, "  ↳ ")
	}
	return lines
}

func findHunkStarts(lines []renderedLine) []int {
	var starts []int
	for i, l := range lines {
		if l.isHunk {
			starts = append(starts, i)
		}
	}
	// Fallback: look for @@ in content (for legacy compat)
	if len(starts) == 0 {
		for i, l := range lines {
			if strings.Contains(l.content, "@@") {
				starts = append(starts, i)
			}
		}
	}
	return starts
}
