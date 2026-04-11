package wifi

import (
	"strings"

	"daily-tui/internal/config"
	"daily-tui/internal/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type wifiState int

const (
	stateBrowsing  wifiState = iota
	stateInputting
	stateConnecting
)

// Model is the Bubble Tea model for the WiFi tab.
type Model struct {
	cfg       *config.Config
	networks  []Network
	cursor    int
	connected string
	iface     string
	state     wifiState
	input     textinput.Model
	status    string
	err       string
}

// New creates a WifiModel with the given config.
func New(cfg *config.Config) Model {
	ti := textinput.New()
	ti.Placeholder = "Password"
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	return Model{cfg: cfg, input: ti}
}

// Exported accessors for tests
func (m Model) Cursor() int       { return m.cursor }
func (m Model) Inputting() bool   { return m.state == stateInputting }
func (m Model) Connecting() bool  { return m.state == stateConnecting }
func (m Model) Connected() string { return m.connected }

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
			return m, ConnectSavedCmd(m.iface, ssid)
		case tea.KeyRunes:
			if string(msg.Runes) == "r" {
				return m, ScanCmd()
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
			if m.cfg.IsDailyReset(ssid) {
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

// View renders the WiFi tab content.
func (m Model) View() string {
	var sb strings.Builder

	if m.connected != "" {
		sb.WriteString(theme.StatusOK.Render("● Connected: "+m.connected) + "\n")
	} else {
		sb.WriteString(theme.Dimmed.Render("○ Not connected") + "\n")
	}
	if m.err != "" {
		sb.WriteString(theme.StatusErr.Render("✗ "+m.err) + "\n")
	}
	sb.WriteString("\n")

	if len(m.networks) == 0 {
		sb.WriteString(theme.Dimmed.Render("Loading saved networks...") + "\n")
	} else {
		sb.WriteString(theme.Dimmed.Render("Saved Networks") + "\n")
		for i, n := range m.networks {
			badge := ""
			if m.cfg.IsDailyReset(n.SSID) {
				badge = " " + theme.Badge.Render("[daily]")
			}
			line := n.SSID + badge
			if i == m.cursor {
				sb.WriteString(theme.Selected.Render("▸ "+line) + "\n")
			} else {
				sb.WriteString(theme.Normal.Render("  "+line) + "\n")
			}
		}
	}

	if m.state == stateInputting && len(m.networks) > 0 {
		ssid := m.networks[m.cursor].SSID
		sb.WriteString("\n" + theme.Dimmed.Render("Password for "+ssid+":") + "\n")
		sb.WriteString(m.input.View() + "\n")
	}

	if m.state == stateConnecting {
		sb.WriteString("\n" + theme.Dimmed.Render(m.status) + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(theme.HelpStyle.Render("↑↓ navigate • enter connect • r refresh • esc cancel"))
	return sb.String()
}
