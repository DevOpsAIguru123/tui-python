package theme

import "github.com/charmbracelet/lipgloss"

// Catppuccin Mocha palette
var (
	Crust    = lipgloss.Color("#11111b")
	Mantle   = lipgloss.Color("#181825")
	Base     = lipgloss.Color("#1e1e2e")
	Surface  = lipgloss.Color("#313244")
	Surface1 = lipgloss.Color("#45475a")
	Surface2 = lipgloss.Color("#585b70")
	Overlay  = lipgloss.Color("#6c7086")
	Subtext0 = lipgloss.Color("#a6adc8")
	Text     = lipgloss.Color("#cdd6f4")
	Lavender = lipgloss.Color("#b4befe")
	Blue     = lipgloss.Color("#89b4fa")
	Sapphire = lipgloss.Color("#74c7ec")
	Teal     = lipgloss.Color("#94e2d5")
	Mauve    = lipgloss.Color("#cba6f7")
	Pink     = lipgloss.Color("#f5c2e7")
	Red      = lipgloss.Color("#f38ba8")
	Peach    = lipgloss.Color("#fab387")
	Yellow   = lipgloss.Color("#f9e2af")
	Green    = lipgloss.Color("#a6e3a1")
)

// KeyHint describes a single key/label pair displayed in the bottom help bar.
type KeyHint struct {
	Key   string
	Label string
}

// RenderKeyHints returns a horizontal help bar of "[ key ] label" pairs.
// Each key glyph is drawn inside a rounded keycap; the label sits to its
// right, vertically centred against the 3-line cap block.
//
// Global keys (tab / q) are intentionally NOT rendered here. On narrower
// terminals they were pushing the trailing keycap past the pane's right
// edge, and they're universal TUI conventions users pick up quickly.
func RenderKeyHints(hints []KeyHint) string {
	if len(hints) == 0 {
		return ""
	}
	blocks := make([]string, 0, 2*len(hints)+1)
	blocks = append(blocks, " ")
	for i, h := range hints {
		if i > 0 {
			blocks = append(blocks, "  ")
		}
		blocks = append(blocks, KeyCap.Render(h.Key))
		blocks = append(blocks, " "+KeyDesc.Render(h.Label))
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, blocks...)
}

var (
	// ---- legacy styles (still referenced by non-redesigned call sites) ----

	ActiveTab = lipgloss.NewStyle().
			Foreground(Base).
			Background(Mauve).
			Padding(0, 1)

	InactiveTab = lipgloss.NewStyle().
			Foreground(Overlay).
			Padding(0, 1)

	Panel = lipgloss.NewStyle().Foreground(Text)

	Selected = lipgloss.NewStyle().Foreground(Mauve).Bold(true)

	Normal = lipgloss.NewStyle().Foreground(Text)

	Dimmed = lipgloss.NewStyle().Foreground(Overlay)

	StatusOK  = lipgloss.NewStyle().Foreground(Green)
	StatusErr = lipgloss.NewStyle().Foreground(Red)

	Badge = lipgloss.NewStyle().Foreground(Peach)

	HelpStyle = lipgloss.NewStyle().Foreground(Overlay)

	ProgressFilled = lipgloss.NewStyle().Foreground(Green)
	ProgressEmpty  = lipgloss.NewStyle().Foreground(Overlay)

	SectionHeader = lipgloss.NewStyle().Foreground(Text).Bold(true)

	// ---- redesigned-chrome styles ----

	Brand = lipgloss.NewStyle().
		Foreground(Mauve).
		Bold(true)

	BrandSub = lipgloss.NewStyle().
			Foreground(Overlay)

	SidebarLabel = lipgloss.NewStyle().
			Foreground(Overlay)

	TabRowActive = lipgloss.NewStyle().
			Foreground(Mauve).
			Bold(true)

	TabRowInactive = lipgloss.NewStyle().
			Foreground(Subtext0)

	TabBulletActive = lipgloss.NewStyle().
			Foreground(Mauve).
			Bold(true)

	TabBulletIdle = lipgloss.NewStyle().
			Foreground(Overlay)

	CountPill = lipgloss.NewStyle().
			Foreground(Overlay).
			Background(Surface).
			Padding(0, 1)

	KeyCap = lipgloss.NewStyle().
		Foreground(Text).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Surface1).
		Padding(0, 1)

	KeyDesc = lipgloss.NewStyle().
			Foreground(Overlay)

	Crumb       = lipgloss.NewStyle().Foreground(Subtext0)
	CrumbActive = lipgloss.NewStyle().Foreground(Text).Bold(true)
	CrumbSep    = lipgloss.NewStyle().Foreground(Surface1)
	CrumbCaret  = lipgloss.NewStyle().Foreground(Mauve).Bold(true)

	Clock = lipgloss.NewStyle().Foreground(Overlay)

	MetaLine = lipgloss.NewStyle().Foreground(Overlay)

	SidebarPromptSym = lipgloss.NewStyle().Foreground(Overlay)
	SidebarPromptVal = lipgloss.NewStyle().Foreground(Green)

	RowSelected = lipgloss.NewStyle().
			Foreground(Text).
			Bold(true)

	Tag = lipgloss.NewStyle().
		Foreground(Subtext0).
		Background(Surface).
		Padding(0, 1)

	TagDaily = lipgloss.NewStyle().
			Foreground(Peach).
			Background(Surface).
			Padding(0, 1)

	TagOpen = lipgloss.NewStyle().
		Foreground(Yellow).
		Background(Surface).
		Padding(0, 1)

	ConnectedDot  = lipgloss.NewStyle().Foreground(Green)
	ConnectedText = lipgloss.NewStyle().Foreground(Green)

	CompletedTask = lipgloss.NewStyle().
			Foreground(Overlay).
			Strikethrough(true)

	SignalFilled = lipgloss.NewStyle().Foreground(Green)
	SignalEmpty  = lipgloss.NewStyle().Foreground(Surface1)
)
