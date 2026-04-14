package wifi

import (
	"strings"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/config"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/theme"

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
func (m Model) Cursor() int            { return m.cursor }
func (m Model) Inputting() bool        { return m.state == stateInputting }
func (m Model) Connecting() bool       { return m.state == stateConnecting }
func (m Model) Connected() string      { return m.connected }
func (m Model) HasInternet() bool      { return m.hasInternet }
func (m Model) CheckingInternet() bool { return m.checkingInternet }

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

// View renders the WiFi tab content.
func (m Model) View() string {
	var sb strings.Builder

	if m.connected != "" {
		status := "● Connected: " + m.connected
		if m.checkingInternet {
			sb.WriteString(theme.StatusOK.Render(status) + " " + theme.Dimmed.Render("(checking internet...)") + "\n")
		} else if m.hasInternet {
			sb.WriteString(theme.StatusOK.Render(status) + "\n")
		} else {
			sb.WriteString(theme.StatusOK.Render(status) + " " + theme.StatusErr.Render("(no internet)") + "\n")
		}
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
	sb.WriteString(theme.HelpStyle.Render("↑↓ navigate • enter connect/reset pw • r refresh • esc cancel"))
	return sb.String()
}
