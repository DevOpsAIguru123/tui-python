package theme

import "github.com/charmbracelet/lipgloss"

// Catppuccin Mocha palette
var (
	Base    = lipgloss.Color("#1e1e2e")
	Surface = lipgloss.Color("#313244")
	Mauve   = lipgloss.Color("#cba6f7")
	Green   = lipgloss.Color("#a6e3a1")
	Red     = lipgloss.Color("#f38ba8")
	Text    = lipgloss.Color("#cdd6f4")
	Overlay = lipgloss.Color("#6c7086")
	Peach   = lipgloss.Color("#fab387")
)

var (
	ActiveTab = lipgloss.NewStyle().
			Foreground(Base).
			Background(Mauve).
			Padding(0, 1)

	InactiveTab = lipgloss.NewStyle().
			Foreground(Overlay).
			Padding(0, 1)

	Panel = lipgloss.NewStyle().
		Foreground(Text)

	Selected = lipgloss.NewStyle().
			Foreground(Mauve).
			Bold(true)

	Normal = lipgloss.NewStyle().
		Foreground(Text)

	Dimmed = lipgloss.NewStyle().
		Foreground(Overlay)

	StatusOK = lipgloss.NewStyle().
			Foreground(Green)

	StatusErr = lipgloss.NewStyle().
			Foreground(Red)

	Badge = lipgloss.NewStyle().
		Foreground(Peach)

	HelpStyle = lipgloss.NewStyle().
			Foreground(Overlay)
)
