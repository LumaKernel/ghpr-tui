package prlist

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/LumaKernel/ghprq/internal/ghclient"
	"github.com/LumaKernel/ghprq/internal/state"
	"github.com/LumaKernel/ghprq/internal/ui/styles"
)

// SelectMsg is sent when a PR is selected.
type SelectMsg struct {
	PR ghclient.PR
}

// RefreshMsg requests a data refresh.
type RefreshMsg struct{}

// OpenBrowserMsg requests opening a PR in browser.
type OpenBrowserMsg struct {
	Number int
}

// MergeMsg requests merging a PR.
type MergeMsg struct {
	Number  int
	Method  string // "merge", "squash", "rebase"
	Undraft bool   // undraft before merge
}

// ToggleDraftMsg requests toggling draft status.
type ToggleDraftMsg struct {
	Number  int
	IsDraft bool
}

// Model is the PR list screen model.
type Model struct {
	prs          []ghclient.PR
	cursor       int
	width        int
	height       int
	repo         string
	store        *state.Store
	loading      bool
	err          error
	confirmMerge   bool     // showing merge confirmation
	mergeMethod    int      // index into allowedMethods
	merging        bool     // merge in progress
	mergeResult    string   // result message
	allowedMethods []string // from repo settings
}

// New creates a new PR list model.
func New(repo string, store *state.Store) Model {
	return Model{
		repo:    repo,
		store:   store,
		loading: true,
	}
}

// SetSize updates the terminal dimensions.
func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h
	return m
}

// SetPRs updates the PR list.
func (m Model) SetPRs(prs []ghclient.PR) Model {
	m.prs = prs
	m.loading = false
	if m.cursor >= len(prs) {
		m.cursor = max(0, len(prs)-1)
	}
	return m
}

// SetError sets an error state.
func (m Model) SetError(err error) Model {
	m.err = err
	m.loading = false
	return m
}

// SetLoading sets the loading state.
func (m Model) SetLoading(loading bool) Model {
	m.loading = loading
	return m
}

func (m Model) visibleHeight() int {
	h := m.height - 4
	if h < 1 {
		return 10
	}
	return h
}

// SetAllowedMergeMethods sets the allowed merge methods from repo settings.
func (m Model) SetAllowedMergeMethods(methods []string) Model {
	m.allowedMethods = methods
	return m
}

func (m Model) mergeMethods() []string {
	if len(m.allowedMethods) > 0 {
		return m.allowedMethods
	}
	return []string{"squash", "merge", "rebase"}
}

// SetMergeResult sets the merge result message.
func (m Model) SetMergeResult(msg string) Model {
	m.mergeResult = msg
	m.merging = false
	return m
}

// Update handles input for the PR list screen.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Clear merge result on any key
		if m.mergeResult != "" {
			m.mergeResult = ""
		}

		// Merge confirmation mode
		if m.confirmMerge {
			switch msg.String() {
			case "h", "left":
				m.mergeMethod = (m.mergeMethod + len(m.mergeMethods()) - 1) % len(m.mergeMethods())
			case "l", "right":
				m.mergeMethod = (m.mergeMethod + 1) % len(m.mergeMethods())
			case "enter", "y":
				if len(m.prs) > 0 {
					pr := m.prs[m.cursor]
					method := m.mergeMethods()[m.mergeMethod]
					undraft := pr.IsDraft
					m.confirmMerge = false
					m.merging = true
					return m, func() tea.Msg {
						return MergeMsg{Number: pr.Number, Method: method, Undraft: undraft}
					}
				}
			case "esc", "n", "q":
				m.confirmMerge = false
			}
			return m, nil
		}

		switch msg.String() {
		case "j", "down", "ctrl+n":
			if m.cursor < len(m.prs)-1 {
				m.cursor++
			}
		case "k", "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
		case "ctrl+d":
			m.cursor = min(m.cursor+m.visibleHeight()/2, len(m.prs)-1)
		case "ctrl+u":
			m.cursor = max(0, m.cursor-m.visibleHeight()/2)
		case "ctrl+f", "pgdown":
			m.cursor = min(m.cursor+m.visibleHeight(), len(m.prs)-1)
		case "ctrl+b", "pgup":
			m.cursor = max(0, m.cursor-m.visibleHeight())
		case "g", "home":
			m.cursor = 0
		case "G", "end":
			if len(m.prs) > 0 {
				m.cursor = len(m.prs) - 1
			}
		case "enter":
			if len(m.prs) > 0 {
				pr := m.prs[m.cursor]
				m.store.MarkRead(m.repo, pr.Number)
				_ = m.store.Save()
				return m, func() tea.Msg { return SelectMsg{PR: pr} }
			}
		case "m":
			if len(m.prs) > 0 && !m.merging {
				m.confirmMerge = true
				m.mergeMethod = 0 // default squash
			}
		case "r":
			if len(m.prs) > 0 {
				pr := m.prs[m.cursor]
				m.store.ToggleRead(m.repo, pr.Number)
				_ = m.store.Save()
			}
		case "R":
			m.loading = true
			return m, func() tea.Msg { return RefreshMsg{} }
		case "d":
			if len(m.prs) > 0 {
				pr := m.prs[m.cursor]
				return m, func() tea.Msg {
					return ToggleDraftMsg{Number: pr.Number, IsDraft: pr.IsDraft}
				}
			}
		case "o":
			if len(m.prs) > 0 {
				pr := m.prs[m.cursor]
				return m, func() tea.Msg { return OpenBrowserMsg{Number: pr.Number} }
			}
		}
	}
	return m, nil
}

// View renders the PR list screen.
func (m Model) View() string {
	var b strings.Builder

	// Header
	header := styles.Title.Render(fmt.Sprintf(" ghprq — %s ", m.repo))
	b.WriteString(header)
	b.WriteString("\n")

	if m.loading {
		b.WriteString("\n  Loading PRs...")
		return b.String()
	}

	if m.err != nil {
		b.WriteString(fmt.Sprintf("\n  Error: %v", m.err))
		return b.String()
	}

	if len(m.prs) == 0 {
		b.WriteString("\n  No open pull requests.")
		return b.String()
	}

	// PR list
	visibleHeight := m.height - 4 // header + help
	if visibleHeight < 1 {
		visibleHeight = 10
	}

	// Scrolling window
	startIdx := 0
	if m.cursor >= visibleHeight {
		startIdx = m.cursor - visibleHeight + 1
	}
	endIdx := startIdx + visibleHeight
	if endIdx > len(m.prs) {
		endIdx = len(m.prs)
	}

	b.WriteString("\n")
	for i := startIdx; i < endIdx; i++ {
		pr := m.prs[i]
		selected := i == m.cursor
		line := m.renderPR(pr, selected)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Merge confirmation / result
	if m.merging {
		b.WriteString("\n")
		b.WriteString(styles.Subtitle.Render("  Merging..."))
		b.WriteString("\n")
	} else if m.mergeResult != "" {
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s", m.mergeResult))
		b.WriteString("\n")
	} else if m.confirmMerge && len(m.prs) > 0 {
		pr := m.prs[m.cursor]
		b.WriteString("\n")
		action := "Merge"
		if pr.IsDraft {
			action = "Undraft & Merge"
		}
		b.WriteString(styles.Unread.Render(fmt.Sprintf("  %s #%d? ", action, pr.Number)))
		for i, method := range m.mergeMethods() {
			if i == m.mergeMethod {
				b.WriteString(styles.Selected.Render(fmt.Sprintf(" [%s] ", method)))
			} else {
				b.WriteString(styles.Help.Render(fmt.Sprintf("  %s  ", method)))
			}
		}
		b.WriteString(styles.Help.Render("  h/l:method  enter:confirm  esc:cancel"))
		b.WriteString("\n")
	}

	// Status bar
	b.WriteString("\n")
	help := styles.Help.Render("  j/k:navigate  C-d/C-u:half-page  enter:open  m:merge  d:draft  r:read/unread  R:refresh  o:browser  q:quit")
	b.WriteString(help)

	return b.String()
}

func (m Model) renderPR(pr ghclient.PR, selected bool) string {
	isRead := m.store.IsRead(m.repo, pr.Number)

	// Indicator
	indicator := "  "
	if !isRead {
		indicator = styles.Unread.Render("● ")
	}

	// PR number
	num := styles.PRNumber.Render(fmt.Sprintf("#%-4d", pr.Number))

	// Title (truncate if needed)
	titleWidth := m.width - 45
	if titleWidth < 20 {
		titleWidth = 20
	}
	title := pr.Title
	if len(title) > titleWidth {
		title = title[:titleWidth-1] + "…"
	}
	if !isRead {
		title = lipgloss.NewStyle().Bold(true).Render(title)
	}

	// Draft
	draftTag := ""
	if pr.IsDraft {
		draftTag = styles.Draft.Render(" [draft]")
	}

	// Stats
	stats := fmt.Sprintf("%s%s",
		styles.Added.Render(fmt.Sprintf("+%d", pr.Additions)),
		styles.Removed.Render(fmt.Sprintf(" -%d", pr.Deletions)),
	)

	// Author
	author := styles.Author.Render(pr.Author)

	// Time
	timeStr := styles.Subtitle.Render(relativeTime(pr.UpdatedAt))

	line := fmt.Sprintf("%s%s %s%s  %s  %s  %s",
		indicator, num, title, draftTag, stats, author, timeStr)

	if selected {
		// Apply selection highlight across the full width
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
