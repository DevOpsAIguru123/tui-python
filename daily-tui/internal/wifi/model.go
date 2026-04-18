package wifi

import (
	"fmt"
	"strings"
	"time"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/config"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type wifiState int

const (
	stateBrowsing wifiState = iota
	stateInputting
	stateConnecting
)

// Model is the Bubble Tea model for the WiFi tab.
type Model struct {
	cfg              *config.Config
	networks         []Network
	cursor           int
	connected        string
	iface            string
	state            wifiState
	input            textinput.Model
	status           string
	err              string
	hasInternet      bool
	checkingInternet bool
	lastScanAt       time.Time
	rowWidth         int
}

// SetRowWidth lets the app tell the wifi view how wide the content area is.
// The view uses it to right-align the signal/dBm/security cluster.
func (m *Model) SetRowWidth(w int) { m.rowWidth = w }

// New creates a WifiModel with the given config.
func New(cfg *config.Config) Model {
	ti := textinput.New()
	ti.Placeholder = "Password"
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	return Model{cfg: cfg, input: ti}
}

// Exported accessors for tests
func (m Model) Cursor() int            { return m.cursor }
func (m Model) Inputting() bool        { return m.state == stateInputting }
func (m Model) Connecting() bool       { return m.state == stateConnecting }
func (m Model) Connected() string      { return m.connected }
func (m Model) Iface() string          { return m.iface }
func (m Model) HasInternet() bool      { return m.hasInternet }
func (m Model) CheckingInternet() bool { return m.checkingInternet }

// Count returns the number of known networks, used by the sidebar count pill.
func (m Model) Count() int { return len(m.networks) }

// Title is the subtitle shown in the breadcrumb header.
func (m Model) Title() string { return "Network Manager" }

// Help returns the key hints shown in the bottom help bar.
func (m Model) Help() []theme.KeyHint {
	switch m.state {
	case stateInputting:
		return []theme.KeyHint{
			{Key: "enter", Label: "submit"},
			{Key: "esc", Label: "cancel"},
		}
	case stateConnecting:
		return []theme.KeyHint{{Key: "esc", Label: "cancel"}}
	default:
		return []theme.KeyHint{
			{Key: "↑↓", Label: "navigate"},
			{Key: "enter", Label: "connect"},
			{Key: "r", Label: "refresh"},
			{Key: "esc", Label: "cancel"},
		}
	}
}

// SetNetworks sets the network list (used in tests and from ScanDoneMsg).
func (m *Model) SetNetworks(n []Network) { m.networks = n }

// SetIface sets the WiFi interface (used in tests and from InterfaceDetectedMsg).
func (m *Model) SetIface(iface string) { m.iface = iface }

// Init starts interface detection; scanning and current-network detection
// are triggered once the interface name is known (see InterfaceDetectedMsg handler).
func (m Model) Init() tea.Cmd {
	return DetectInterfaceCmd()
}

// Update handles messages and keypresses.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case InterfaceDetectedMsg:
		if msg.Err == nil {
			m.iface = msg.Iface
		}
		return m, tea.Batch(ScanCmd(), GetCurrentNetworkCmd(m.iface))

	case ScanDoneMsg:
		if msg.Err != nil {
			m.err = msg.Err.Error()
		} else {
			m.networks = msg.Networks
			m.lastScanAt = time.Now()
			m.err = ""
		}
		return m, nil

	case ConnectDoneMsg:
		m.state = stateBrowsing
		if msg.Err != nil {
			m.err = msg.Err.Error()
			m.status = ""
		} else if msg.SSID != "" {
			m.connected = msg.SSID
			m.status = "Connected"
			m.err = ""
			m.checkingInternet = true
			// Cache the gateway MAC → SSID mapping so future startups can
			// resolve the name even when macOS redacts the SSID.
			return m, tea.Batch(PingInternetCmd(), CacheNetworkCmd(m.iface, msg.SSID))
		} else {
			// Not connected (empty SSID from GetCurrentNetworkCmd)
			m.connected = ""
			m.hasInternet = false
			m.checkingInternet = false
			m.status = ""
		}
		return m, nil

	case InternetCheckMsg:
		m.checkingInternet = false
		m.hasInternet = msg.Reachable
		if !msg.Reachable && m.connected != "" {
			return m, m.promptForNoInternet()
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	if m.state == stateInputting {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {

	case stateBrowsing:
		switch msg.Type {
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown:
			if m.cursor < len(m.networks)-1 {
				m.cursor++
			}
		case tea.KeyEnter:
			if len(m.networks) == 0 {
				return m, nil
			}
			ssid := m.networks[m.cursor].SSID
			if ssid == m.connected {
				// Already-connected network: prompt for password reset
				m.state = stateInputting
				m.input.SetValue("")
				m.input.Focus()
				return m, nil
			}
			if m.cfg.IsDailyReset(ssid) {
				// Daily-reset networks always prompt for a fresh password
				m.state = stateInputting
				m.input.SetValue("")
				m.input.Focus()
				return m, nil
			}
			// All listed networks are already saved — connect directly
			m.state = stateConnecting
			m.status = "Connecting..."
			m.hasInternet = false
			m.checkingInternet = false
			return m, ConnectSavedCmd(m.iface, ssid)
		case tea.KeyRunes:
			if string(msg.Runes) == "r" {
				return m, tea.Batch(ScanCmd(), GetCurrentNetworkCmd(m.iface))
			}
		}

	case stateInputting:
		switch msg.Type {
		case tea.KeyEsc:
			m.state = stateBrowsing
			m.input.Blur()
		case tea.KeyEnter:
			if len(m.networks) == 0 {
				return m, nil
			}
			ssid := m.networks[m.cursor].SSID
			password := m.input.Value()
			m.input.Blur()
			m.state = stateConnecting
			m.status = "Connecting..."
			m.hasInternet = false
			m.checkingInternet = false
			if ssid == m.connected || m.cfg.IsDailyReset(ssid) {
				return m, ForgetAndConnectCmd(m.iface, ssid, password)
			}
			return m, ConnectCmd(m.iface, ssid, password)
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// promptForNoInternet is called when connected but no internet is detected.
// It finds the right network to prompt a password reset for:
//   - Real SSID known → find it in the list and prompt directly.
//   - SSID was redacted ("Wi-Fi") → fall back to the first [daily] network,
//     since daily-reset networks are exactly the scenario where
//     "connected but no internet" means the password changed.
func (m *Model) promptForNoInternet() tea.Cmd {
	target := m.connected

	if target == "Wi-Fi" {
		for _, n := range m.networks {
			if m.cfg.IsDailyReset(n.SSID) {
				target = n.SSID
				break
			}
		}
	}

	if target == "" || target == "Wi-Fi" {
		return nil
	}

	for i, n := range m.networks {
		if n.SSID == target {
			m.connected = target
			m.cursor = i
			m.state = stateInputting
			m.input.SetValue("")
			m.input.Focus()
			return nil
		}
	}
	return nil
}

// ---- View ----

// View renders the WiFi tab content (no help bar — the app frame draws that).
func (m Model) View() string {
	var sb strings.Builder

	status := m.renderStatusLine()
	if meta := m.MetaRight(); meta != "" {
		w := m.rowWidth
		if w <= 0 {
			w = 84
		}
		status = padBetween(status, theme.MetaLine.Render(meta), w)
	}
	sb.WriteString(status)
	sb.WriteString("\n")
	if m.err != "" {
		sb.WriteString(theme.StatusErr.Render("✗ "+m.err) + "\n")
	}
	sb.WriteString("\n")

	sb.WriteString(renderSectionDivider("NEARBY NETWORKS"))
	sb.WriteString("\n")

	if len(m.networks) == 0 {
		sb.WriteString(theme.Dimmed.Render("  Loading saved networks...") + "\n")
	} else {
		for i, n := range m.networks {
			sb.WriteString(m.renderRow(n, i == m.cursor))
			sb.WriteString("\n")
		}
	}

	if m.state == stateInputting && len(m.networks) > 0 {
		ssid := m.networks[m.cursor].SSID
		label := "Password for " + ssid + ":"
		if !m.hasInternet && m.connected != "" {
			label = "No internet — enter new password for " + ssid + ":"
		}
		sb.WriteString("\n" + theme.Dimmed.Render(label) + "\n")
		sb.WriteString(m.input.View() + "\n")
	}

	if m.state == stateConnecting {
		sb.WriteString("\n" + theme.Dimmed.Render(m.status) + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(theme.RenderKeyHints(m.Help()))
	return sb.String()
}

// StatusLine returns just the left-side status line (for tests and for the
// app chrome, which renders it alongside the "last scan" meta).
func (m Model) renderStatusLine() string {
	if m.connected == "" {
		return theme.Dimmed.Render("○ Not connected")
	}
	status := theme.ConnectedDot.Render("● ") + theme.Dimmed.Render("Connected: ") +
		theme.ConnectedText.Render(m.connected)
	if m.checkingInternet {
		status += " " + theme.Dimmed.Render("(checking internet...)")
	} else if !m.hasInternet {
		status += " " + theme.StatusErr.Render("(no internet)")
	}
	return status
}

// MetaRight returns the "last scan Xs ago" hint that sits on the right edge
// of the status row when a scan has completed at least once.
func (m Model) MetaRight() string {
	if m.lastScanAt.IsZero() {
		return ""
	}
	age := time.Since(m.lastScanAt)
	return "last scan " + humanDuration(age) + " ago"
}

func (m Model) renderRow(n Network, selected bool) string {
	caret := "  "
	if selected {
		caret = theme.CrumbCaret.Render("▸ ")
	}

	name := n.SSID
	if selected {
		name = theme.RowSelected.Render(name)
	} else {
		name = theme.Normal.Render(name)
	}

	var tags []string
	if m.cfg != nil && m.cfg.IsDailyReset(n.SSID) {
		tags = append(tags, theme.TagDaily.Render("daily"))
	}
	if n.SSID == m.connected {
		tags = append(tags, theme.ConnectedDot.Render("● ")+theme.ConnectedText.Render("connected"))
	} else if n.Security == "" && n.RSSI != 0 {
		tags = append(tags, theme.TagOpen.Render("open"))
	} else if n.SSID != m.connected && !(m.cfg != nil && m.cfg.IsDailyReset(n.SSID)) {
		tags = append(tags, theme.Tag.Render("saved"))
	}

	left := caret + name
	if len(tags) > 0 {
		left += "  " + strings.Join(tags, " ")
	}

	// Right cluster: signal bars · dBm · security
	right := renderSignal(n.RSSI) + "  " + renderDBm(n.RSSI) + "  " + renderSecurity(n.Security, n.RSSI)
	w := m.rowWidth
	if w <= 0 {
		w = 84
	}
	return padBetween(left, right, w)
}

func renderSignal(rssi int) string {
	glyphs := BarGlyphs()
	bars := SignalBars(rssi)
	if rssi == 0 {
		bars = 0
	}
	var sb strings.Builder
	for i, g := range glyphs {
		if i < bars {
			sb.WriteString(theme.SignalFilled.Render(string(g)))
		} else {
			sb.WriteString(theme.SignalEmpty.Render(string(g)))
		}
	}
	return sb.String()
}

func renderDBm(rssi int) string {
	if rssi == 0 {
		return theme.Dimmed.Render("   —  ")
	}
	s := fmt.Sprintf("%4d dBm", rssi)
	return theme.Dimmed.Render(s)
}

func renderSecurity(sec string, rssi int) string {
	if sec == "" {
		if rssi == 0 {
			return theme.Dimmed.Render("—   ")
		}
		return theme.Dimmed.Render("--  ")
	}
	return theme.Dimmed.Render(fmt.Sprintf("%-4s", sec))
}

func renderSectionDivider(label string) string {
	return theme.Dimmed.Render("── ") + theme.SectionHeader.Render(label) + " " +
		theme.Dimmed.Render(strings.Repeat("─", 60))
}

// padBetween joins a left and right string with spaces so the rendered width
// is at least `width`. Width is approximated by raw rune count because the
// caller's styles don't widen the content (only colour).
func padBetween(left, right string, width int) string {
	leftW := visibleWidth(left)
	rightW := visibleWidth(right)
	pad := width - leftW - rightW
	if pad < 2 {
		pad = 2
	}
	return left + strings.Repeat(" ", pad) + right
}

// visibleWidth strips ANSI escape sequences and returns the rune count.
func visibleWidth(s string) int {
	out := make([]rune, 0, len(s))
	inEsc := false
	for _, r := range s {
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		if r == 0x1b {
			inEsc = true
			continue
		}
		out = append(out, r)
	}
	return len(out)
}

func humanDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
}
