package comments

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/LumaKernel/ghpr-tui/internal/ghclient"
	"github.com/LumaKernel/ghpr-tui/internal/ui/styles"
)

// BackMsg requests going back.
type BackMsg struct{}

// RefreshMsg requests refreshing comments.
type RefreshMsg struct {
	Number int
}

// JumpToFileMsg requests navigating to a specific file and line in the diff.
type JumpToFileMsg struct {
	Path string
	Line int
}

// ReplyMsg requests posting a reply to a comment thread.
type ReplyMsg struct {
	PRNumber    int
	InReplyToID int
	Body        string
}

// Model is the comments list screen model.
type Model struct {
	pr       ghclient.PR
	threads  []ghclient.CommentThread
	cursor   int
	width    int
	height   int
	loading  bool
	err      error
	// Reply mode
	replyMode bool
	replyText []string
	replyRow  int
	replyCol  int
}

// New creates a new comments model.
func New() Model {
	return Model{}
}

// SetPR sets the current PR.
func (m Model) SetPR(pr ghclient.PR) Model {
	m.pr = pr
	m.loading = true
	m.cursor = 0
	m.err = nil
	return m
}

// SetComments sets the comment threads.
func (m Model) SetComments(threads []ghclient.CommentThread) Model {
	m.threads = threads
	m.loading = false
	if m.cursor >= len(m.threads) {
		m.cursor = max(0, len(m.threads)-1)
	}
	return m
}

// SetError sets the error state.
func (m Model) SetError(err error) Model {
	m.err = err
	m.loading = false
	return m
}

// SetSize updates terminal dimensions.
func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h
	return m
}

func (m Model) visibleHeight() int {
	h := m.height - 7
	if h < 1 {
		return 10
	}
	return h
}

// Update handles input for the comments screen.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.replyMode {
			return m.updateReply(msg)
		}
		return m.updateNormal(msg)
	}
	return m, nil
}

func (m Model) updateNormal(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.cursor < len(m.threads)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "ctrl+d":
		m.cursor = min(m.cursor+m.visibleHeight()/2, max(0, len(m.threads)-1))
	case "ctrl+u":
		m.cursor = max(0, m.cursor-m.visibleHeight()/2)
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		if len(m.threads) > 0 {
			m.cursor = len(m.threads) - 1
		}
	case "enter":
		if len(m.threads) > 0 {
			t := m.threads[m.cursor]
			return m, func() tea.Msg {
				return JumpToFileMsg{Path: t.Root.Path, Line: t.Root.Line}
			}
		}
	case "r":
		if len(m.threads) > 0 {
			m.replyMode = true
			m.replyText = []string{""}
			m.replyRow = 0
			m.replyCol = 0
		}
	case "R":
		m.loading = true
		number := m.pr.Number
		return m, func() tea.Msg { return RefreshMsg{Number: number} }
	case "esc", "backspace", "q":
		return m, func() tea.Msg { return BackMsg{} }
	}
	return m, nil
}

func (m Model) updateReply(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.replyMode = false
		m.replyText = nil
	case "ctrl+s":
		body := strings.Join(m.replyText, "\n")
		body = strings.TrimSpace(body)
		if body == "" {
			return m, nil
		}
		t := m.threads[m.cursor]
		m.replyMode = false
		m.loading = true
		number := m.pr.Number
		rootID := t.Root.ID
		return m, func() tea.Msg {
			return ReplyMsg{PRNumber: number, InReplyToID: rootID, Body: body}
		}
	case "enter":
		tail := m.replyText[m.replyRow][m.replyCol:]
		m.replyText[m.replyRow] = m.replyText[m.replyRow][:m.replyCol]
		newLines := make([]string, 0, len(m.replyText)+1)
		newLines = append(newLines, m.replyText[:m.replyRow+1]...)
		newLines = append(newLines, tail)
		newLines = append(newLines, m.replyText[m.replyRow+1:]...)
		m.replyText = newLines
		m.replyRow++
		m.replyCol = 0
	case "backspace":
		if m.replyCol > 0 {
			line := m.replyText[m.replyRow]
			m.replyText[m.replyRow] = line[:m.replyCol-1] + line[m.replyCol:]
			m.replyCol--
		} else if m.replyRow > 0 {
			prevLen := len(m.replyText[m.replyRow-1])
			m.replyText[m.replyRow-1] += m.replyText[m.replyRow]
			m.replyText = append(m.replyText[:m.replyRow], m.replyText[m.replyRow+1:]...)
			m.replyRow--
			m.replyCol = prevLen
		}
	default:
		ch := msg.String()
		if len(ch) == 1 && ch[0] >= 32 {
			line := m.replyText[m.replyRow]
			m.replyText[m.replyRow] = line[:m.replyCol] + ch + line[m.replyCol:]
			m.replyCol++
		}
	}
	return m, nil
}

// View renders the comments screen.
func (m Model) View() string {
	var b strings.Builder

	header := fmt.Sprintf(" #%d Comments ", m.pr.Number)
	b.WriteString(styles.Title.Render(header))
	b.WriteString("\n")

	if m.loading {
		b.WriteString("\n  Loading comments...")
		return b.String()
	}

	if m.err != nil {
		b.WriteString(fmt.Sprintf("\n  Error: %v", m.err))
		return b.String()
	}

	if len(m.threads) == 0 {
		b.WriteString("\n  No review comments.")
		b.WriteString("\n\n")
		b.WriteString(styles.Help.Render("  esc:back"))
		return b.String()
	}

	// Summary
	b.WriteString(styles.Subtitle.Render(fmt.Sprintf("  %d comment threads", len(m.threads))))
	b.WriteString("\n\n")

	// Thread list
	vpHeight := m.visibleHeight()
	startIdx := 0
	if m.cursor >= vpHeight {
		startIdx = m.cursor - vpHeight + 1
	}
	endIdx := startIdx + vpHeight
	if endIdx > len(m.threads) {
		endIdx = len(m.threads)
	}

	for i := startIdx; i < endIdx; i++ {
		t := m.threads[i]
		selected := i == m.cursor
		line := m.renderThread(t, selected)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Reply mode
	if m.replyMode {
		b.WriteString("\n")
		b.WriteString(styles.CommentMarker.Render("  Reply:"))
		b.WriteString("\n")
		for i, line := range m.replyText {
			cursor := " "
			if i == m.replyRow {
				cursor = ">"
			}
			b.WriteString(fmt.Sprintf("  %s %s\n", cursor, line))
		}
		b.WriteString(styles.Help.Render("  ctrl+s:submit  esc:cancel"))
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n")
	if m.replyMode {
		b.WriteString(styles.Help.Render("  ctrl+s:submit  esc:cancel  enter:newline"))
	} else {
		b.WriteString(styles.Help.Render("  j/k:navigate  enter:jump to code  r:reply  R:refresh  esc:back"))
	}

	return b.String()
}

func (m Model) renderThread(thread ghclient.CommentThread, selected bool) string {
	root := thread.Root

	// Path and line info
	lineInfo := fmt.Sprintf("%s:%d", root.Path, root.Line)
	if root.StartLine > 0 && root.StartLine != root.Line {
		lineInfo = fmt.Sprintf("%s:%d-%d", root.Path, root.StartLine, root.Line)
	}

	// Body preview
	body := strings.ReplaceAll(root.Body, "\n", " ")
	maxLen := m.width - 40
	if maxLen < 20 {
		maxLen = 20
	}
	if len(body) > maxLen {
		body = body[:maxLen-1] + "…"
	}

	// Reply count
	replyInfo := ""
	if len(thread.Replies) > 0 {
		replyInfo = styles.Subtitle.Render(fmt.Sprintf(" +%d replies", len(thread.Replies)))
	}

	// Time
	timeStr := relativeTime(root.CreatedAt)

	line := fmt.Sprintf("  %s  %s  %s  %s%s",
		styles.CommentAuthor.Render("@"+root.User.Login),
		styles.Subtitle.Render(lineInfo),
		body,
		styles.Subtitle.Render(timeStr),
		replyInfo,
	)

	if selected {
		line = styles.Selected.Width(m.width).Render(line)
	}

	return line
}

func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}
