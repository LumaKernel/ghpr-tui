package diffview

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/LumaKernel/ghprq/internal/ghclient"
	"github.com/LumaKernel/ghprq/internal/state"
	"github.com/LumaKernel/ghprq/internal/ui/styles"
)

// BackMsg requests going back to file list.
type BackMsg struct{}

// NextFileMsg requests the next file.
type NextFileMsg struct{}

// PrevFileMsg requests the previous file.
type PrevFileMsg struct{}

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
}

type renderedLine struct {
	content string
}

// FileIndex returns the current file index.
func (m Model) FileIndex() int {
	return m.fileIndex
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
	m.lines = renderDiffLines(file, m.width)
	m.hunkStarts = findHunkStarts(m.lines)
	return m
}

// SetSize updates terminal dimensions.
func (m Model) SetSize(w, h int) Model {
	m.width = w
	m.height = h
	if m.file.NewPath != "" || m.file.OldPath != "" {
		m.lines = renderDiffLines(m.file, w)
		m.hunkStarts = findHunkStarts(m.lines)
	}
	return m
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
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
		case "esc", "backspace":
			return m, func() tea.Msg { return BackMsg{} }
		}
	}
	return m, nil
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
	header := fmt.Sprintf(" %s%s  (%d/%d) ", path, reviewMark, m.fileIndex+1, m.fileCount)
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
	endIdx := m.scroll + vpHeight
	if endIdx > len(m.lines) {
		endIdx = len(m.lines)
	}

	for i := m.scroll; i < endIdx; i++ {
		b.WriteString(m.lines[i].content)
		b.WriteString("\n")
	}

	// Scroll indicator
	scrollPct := ""
	if len(m.lines) > vpHeight {
		pct := float64(m.scroll) / float64(len(m.lines)-vpHeight) * 100
		scrollPct = fmt.Sprintf(" %d%%", int(pct))
	}

	// Help
	help := styles.Help.Render(fmt.Sprintf(
		"  j/k:scroll  d/u:half-page  n/N:hunk  [/]:file  space:reviewed+next  esc:back%s", scrollPct))
	b.WriteString(help)

	return b.String()
}

func renderDiffLines(file ghclient.FileDiff, width int) []renderedLine {
	var lines []renderedLine

	for _, hunk := range file.Hunks {
		// Hunk header
		lines = append(lines, renderedLine{
			content: styles.HunkHeader.Render(hunk.Header),
		})

		for _, dl := range hunk.Lines {
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
				content: gutter + styled,
			})
		}
	}

	return lines
}

func findHunkStarts(lines []renderedLine) []int {
	var starts []int
	for i, l := range lines {
		if strings.HasPrefix(l.content, "\x1b") && strings.Contains(l.content, "@@") {
			starts = append(starts, i)
		}
	}
	// Fallback: look for @@ in raw content
	if len(starts) == 0 {
		for i, l := range lines {
			if strings.Contains(l.content, "@@") {
				starts = append(starts, i)
			}
		}
	}
	return starts
}
