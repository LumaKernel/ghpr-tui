package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Green   = lipgloss.Color("#22c55e")
	Red     = lipgloss.Color("#ef4444")
	Yellow  = lipgloss.Color("#eab308")
	Blue    = lipgloss.Color("#3b82f6")
	Cyan    = lipgloss.Color("#06b6d4")
	Gray    = lipgloss.Color("#6b7280")
	DimGray = lipgloss.Color("#374151")
	White   = lipgloss.Color("#f9fafb")

	// Styles
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(White).
		Background(Blue).
		Padding(0, 1)

	Subtitle = lipgloss.NewStyle().
			Foreground(Gray).
			Italic(true)

	Selected = lipgloss.NewStyle().
			Background(lipgloss.Color("#1e3a5f")).
			Bold(true)

	StatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("#1f2937")).
			Foreground(Gray).
			Padding(0, 1)

	Help = lipgloss.NewStyle().
		Foreground(Gray)

	Added = lipgloss.NewStyle().
		Foreground(Green)

	Removed = lipgloss.NewStyle().
			Foreground(Red)

	HunkHeader = lipgloss.NewStyle().
			Foreground(Cyan).
			Bold(true)

	DiffContext = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9ca3af"))

	AddedBg = lipgloss.NewStyle().
		Foreground(Green).
		Background(lipgloss.Color("#052e16"))

	RemovedBg = lipgloss.NewStyle().
			Foreground(Red).
			Background(lipgloss.Color("#350a0a"))

	LineNum = lipgloss.NewStyle().
		Foreground(DimGray).
		Width(5).
		Align(lipgloss.Right)

	Reviewed = lipgloss.NewStyle().
			Foreground(Green).
			Bold(true)

	Unread = lipgloss.NewStyle().
		Foreground(Yellow).
		Bold(true)

	Draft = lipgloss.NewStyle().
		Foreground(Gray).
		Italic(true)

	Author = lipgloss.NewStyle().
		Foreground(Cyan)

	PRNumber = lipgloss.NewStyle().
			Foreground(Blue).
			Bold(true)

	VisualSelect = lipgloss.NewStyle().
			Background(lipgloss.Color("#2d1b69"))

	CommentMarker = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a78bfa")).
			Italic(true)

	CommentBody = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9ca3af")).
			Background(lipgloss.Color("#1c1c2e"))

	CommentAuthor = lipgloss.NewStyle().
			Foreground(Cyan).
			Bold(true)

	CommentInput = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Blue).
			Padding(0, 1)
)
