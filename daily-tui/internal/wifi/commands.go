package wifi

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Msg types returned by async WiFi commands

type InterfaceDetectedMsg struct {
	Iface string
	Err   error
}

type ScanDoneMsg struct {
	Networks []Network
	Err      error
}

type ConnectDoneMsg struct {
	SSID string
	Err  error
}

// DetectInterfaceCmd detects the WiFi interface name (e.g. "en0").
func DetectInterfaceCmd() tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("networksetup", "-listallhardwareports").Output()
		if err != nil {
			return InterfaceDetectedMsg{Err: err}
		}
		iface, err := DetectWiFiInterface(string(out))
		return InterfaceDetectedMsg{Iface: iface, Err: err}
	}
}

// ScanCmd lists saved (preferred) WiFi networks via networksetup.
// macOS 15+ redacts SSIDs in airport/system_profiler output, so we use
// the preferred networks list which always returns real SSID names.
// The iface is detected inline since ScanCmd is also called on 'r' keypress
// before iface is guaranteed to be stored in the model.
func ScanCmd() tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("networksetup", "-listallhardwareports").Output()
		if err != nil {
			return ScanDoneMsg{Err: fmt.Errorf("interface detection failed: %w", err)}
		}
		iface, err := DetectWiFiInterface(string(out))
		if err != nil {
			return ScanDoneMsg{Err: err}
		}
		out, err = exec.Command("networksetup", "-listpreferredwirelessnetworks", iface).Output()
		if err != nil {
			return ScanDoneMsg{Err: fmt.Errorf("list preferred networks failed: %w", err)}
		}
		return ScanDoneMsg{Networks: ParsePreferredNetworks(string(out))}
	}
}

// GetCurrentNetworkCmd detects which SSID the machine is currently connected to.
func GetCurrentNetworkCmd(iface string) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("networksetup", "-getairportnetwork", iface).Output()
		if err != nil {
			return ConnectDoneMsg{Err: fmt.Errorf("could not get current network: %w", err)}
		}
		// Output format: "Current Wi-Fi Network: Agora Public"
		line := strings.TrimSpace(string(out))
		const prefix = "Current Wi-Fi Network: "
		if strings.HasPrefix(line, prefix) {
			ssid := strings.TrimPrefix(line, prefix)
			return ConnectDoneMsg{SSID: ssid}
		}
		return ConnectDoneMsg{} // not connected
	}
}

// ForgetAndConnectCmd forgets the network then reconnects with a new password.
// Used for daily-reset networks (e.g. "Agora Public").
func ForgetAndConnectCmd(iface, ssid, password string) tea.Cmd {
	return func() tea.Msg {
		_ = exec.Command("networksetup", "-removepreferredwirelessnetwork", iface, ssid).Run()
		out, err := exec.Command("networksetup", "-setairportnetwork", iface, ssid, password).CombinedOutput()
		if err != nil {
			return ConnectDoneMsg{SSID: ssid, Err: fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))}
		}
		return ConnectDoneMsg{SSID: ssid}
	}
}

// ConnectCmd connects to a new network with a password.
func ConnectCmd(iface, ssid, password string) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("networksetup", "-setairportnetwork", iface, ssid, password).CombinedOutput()
		if err != nil {
			return ConnectDoneMsg{SSID: ssid, Err: fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))}
		}
		return ConnectDoneMsg{SSID: ssid}
	}
}

// ConnectSavedCmd connects to an already-saved network (no password needed).
func ConnectSavedCmd(iface, ssid string) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("networksetup", "-setairportnetwork", iface, ssid).CombinedOutput()
		if err != nil {
			return ConnectDoneMsg{SSID: ssid, Err: fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))}
		}
		return ConnectDoneMsg{SSID: ssid}
	}
}

