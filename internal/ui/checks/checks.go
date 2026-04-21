package checks

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/LumaKernel/ghpr-tui/internal/ghclient"
	"github.com/LumaKernel/ghpr-tui/internal/ui/styles"
)

// BackMsg requests going back.
type BackMsg struct{}

// OpenBrowserMsg requests opening a URL in the browser.
type OpenBrowserMsg struct {
	URL string
}

// RefreshMsg requests refreshing checks.
type RefreshMsg struct {
	Number int
}

// Model is the checks screen model.
type Model struct {
	pr      ghclient.PR
	checks  []ghclient.Check
	cursor  int
	width   int
	height  int
	loading bool
	err     error
}

// New creates a new checks model.
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

// SetChecks sets the check data.
func (m Model) SetChecks(checks []ghclient.Check) Model {
	m.checks = sortChecks(checks)
	m.loading = false
	if m.cursor >= len(m.checks) {
		m.cursor = max(0, len(m.checks)-1)
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

// sortChecks sorts: fail first, then pending, then pass/skip/cancel.
func sortChecks(checks []ghclient.Check) []ghclient.Check {
	sorted := make([]ghclient.Check, len(checks))
	copy(sorted, checks)
	sort.SliceStable(sorted, func(i, j int) bool {
		return bucketOrder(sorted[i].Bucket) < bucketOrder(sorted[j].Bucket)
	})
	return sorted
}

func bucketOrder(bucket string) int {
	switch bucket {
	case "fail":
		return 0
	case "pending":
		return 1
	case "cancel":
		return 2
	case "pass":
		return 3
	case "skipping":
		return 4
	default:
		return 5
	}
}

// Update handles input for the checks screen.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down", "ctrl+n":
			if m.cursor < len(m.checks)-1 {
				m.cursor++
			}
		case "k", "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
		case "ctrl+d":
			m.cursor = min(m.cursor+m.visibleHeight()/2, max(0, len(m.checks)-1))
		case "ctrl+u":
			m.cursor = max(0, m.cursor-m.visibleHeight()/2)
		case "ctrl+f", "pgdown":
			m.cursor = min(m.cursor+m.visibleHeight(), max(0, len(m.checks)-1))
		case "ctrl+b", "pgup":
			m.cursor = max(0, m.cursor-m.visibleHeight())
		case "g", "home":
			m.cursor = 0
		case "G", "end":
			if len(m.checks) > 0 {
				m.cursor = len(m.checks) - 1
			}
		case "enter", "o":
			if len(m.checks) > 0 && m.checks[m.cursor].Link != "" {
				link := m.checks[m.cursor].Link
				return m, func() tea.Msg { return OpenBrowserMsg{URL: link} }
			}
		case "R":
			m.loading = true
			number := m.pr.Number
			return m, func() tea.Msg { return RefreshMsg{Number: number} }
		case "esc", "backspace", "q":
			return m, func() tea.Msg { return BackMsg{} }
		}
	}
	return m, nil
}

// View renders the checks screen.
func (m Model) View() string {
	var b strings.Builder

	// Header
	header := fmt.Sprintf(" #%d Checks ", m.pr.Number)
	b.WriteString(styles.Title.Render(header))
	b.WriteString("\n")

	if m.loading {
		b.WriteString("\n  Loading checks...")
		return b.String()
	}

	if m.err != nil {
		b.WriteString(fmt.Sprintf("\n  Error: %v", m.err))
		return b.String()
	}

	if len(m.checks) == 0 {
		b.WriteString("\n  No checks found.")
		b.WriteString("\n\n")
		b.WriteString(styles.Help.Render("  esc:back"))
		return b.String()
	}

	// Summary line
	summary := m.buildSummary()
	b.WriteString("  ")
	b.WriteString(summary)
	b.WriteString("\n\n")

	// Check list
	vpHeight := m.visibleHeight()
	startIdx := 0
	if m.cursor >= vpHeight {
		startIdx = m.cursor - vpHeight + 1
	}
	endIdx := startIdx + vpHeight
	if endIdx > len(m.checks) {
		endIdx = len(m.checks)
	}

	for i := startIdx; i < endIdx; i++ {
		check := m.checks[i]
		selected := i == m.cursor
		line := m.renderCheck(check, selected)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n")
	b.WriteString(styles.Help.Render("  j/k:navigate  enter/o:open in browser  R:refresh  esc:back"))

	return b.String()
}

func (m Model) buildSummary() string {
	var parts []string
	pass, fail, pending, skip, cancel := 0, 0, 0, 0, 0
	for _, c := range m.checks {
		switch c.Bucket {
		case "pass":
			pass++
		case "fail":
			fail++
		case "pending":
			pending++
		case "skipping":
			skip++
		case "cancel":
			cancel++
		}
	}
	if pass > 0 {
		parts = append(parts, styles.Added.Render(fmt.Sprintf("%d passed", pass)))
	}
	if fail > 0 {
		parts = append(parts, styles.Removed.Render(fmt.Sprintf("%d failed", fail)))
	}
	if pending > 0 {
		parts = append(parts, styles.Unread.Render(fmt.Sprintf("%d pending", pending)))
	}
	if skip > 0 {
		parts = append(parts, styles.Subtitle.Render(fmt.Sprintf("%d skipped", skip)))
	}
	if cancel > 0 {
		parts = append(parts, styles.Subtitle.Render(fmt.Sprintf("%d cancelled", cancel)))
	}
	return strings.Join(parts, "  ")
}

func (m Model) renderCheck(check ghclient.Check, selected bool) string {
	// Status icon
	var icon string
	switch check.Bucket {
	case "pass":
		icon = styles.Added.Render("✓")
	case "fail":
		icon = styles.Removed.Render("✗")
	case "pending":
		icon = styles.Unread.Render("◌")
	case "skipping":
		icon = styles.Subtitle.Render("⊘")
	case "cancel":
		icon = styles.Subtitle.Render("⊘")
	default:
		icon = styles.Subtitle.Render("?")
	}

	// Name and workflow
	name := check.Name
	workflow := ""
	if check.Workflow != "" && check.Workflow != check.Name {
		workflow = styles.Subtitle.Render(fmt.Sprintf(" (%s)", check.Workflow))
	}

	// Duration
	duration := ""
	d := check.Duration()
	if d > 0 {
		if d < time.Minute {
			duration = fmt.Sprintf("%ds", int(d.Seconds()))
		} else {
			duration = fmt.Sprintf("%dm %02ds", int(d.Minutes()), int(d.Seconds())%60)
		}
		duration = styles.Subtitle.Render(duration)
	} else if check.Bucket == "pending" {
		if !check.StartedAt.IsZero() {
			duration = styles.Unread.Render("running")
		} else {
			duration = styles.Subtitle.Render("pending")
		}
	}

	// Status label
	var statusLabel string
	switch check.Bucket {
	case "pass":
		statusLabel = styles.Added.Render("passed")
	case "fail":
		statusLabel = styles.Removed.Render("failed")
	case "pending":
		statusLabel = styles.Unread.Render("pending")
	case "skipping":
		statusLabel = styles.Subtitle.Render("skipped")
	case "cancel":
		statusLabel = styles.Subtitle.Render("cancelled")
	default:
		statusLabel = check.Bucket
	}

	// Compose line
	line := fmt.Sprintf("  %s  %-40s%s  %8s  %s", icon, name, workflow, duration, statusLabel)

	if selected {
		line = styles.Selected.Width(m.width).Render(line)
	}

	return line
}

// CheckIcon returns a styled icon for a CheckSummary (for use in PR list).
func CheckIcon(cs ghclient.CheckSummary) string {
	if !cs.HasChecks() {
		return styles.Subtitle.Render("—")
	}
	if cs.AnyFail() {
		return styles.Removed.Render("✗")
	}
	if cs.Pending > 0 {
		return styles.Unread.Render("◌")
	}
	if cs.AllPass() {
		return styles.Added.Render("✓")
	}
	return styles.Subtitle.Render("—")
}

// CheckSummaryLine returns a formatted summary for file list header.
func CheckSummaryLine(cs ghclient.CheckSummary) string {
	if !cs.HasChecks() {
		return ""
	}
	var parts []string
	if cs.Pass > 0 {
		parts = append(parts, styles.Added.Render(fmt.Sprintf("%d passed", cs.Pass)))
	}
	if cs.Fail > 0 {
		parts = append(parts, styles.Removed.Render(fmt.Sprintf("%d failed", cs.Fail)))
	}
	if cs.Pending > 0 {
		parts = append(parts, styles.Unread.Render(fmt.Sprintf("%d pending", cs.Pending)))
	}
	return fmt.Sprintf("  Checks: %s", strings.Join(parts, "  "))
}
