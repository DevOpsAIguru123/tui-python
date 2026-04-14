package wifi

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

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

type InternetCheckMsg struct {
	Reachable bool
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
// On macOS 13+ (Ventura and later), networksetup -getairportnetwork may return
// "not associated" even when connected because SSID access requires Location
// Services. We fall back to `ipconfig getsummary <iface>` which still exposes
// LinkStatusActive and the SSID (or the literal "<redacted>" placeholder).
// If the SSID is redacted, we resolve it via a gateway IP+MAC cache lookup.
func GetCurrentNetworkCmd(iface string) tea.Cmd {
	return func() tea.Msg {
		// Primary: networksetup (works on macOS < 13 without Location Services)
		out, err := exec.Command("networksetup", "-getairportnetwork", iface).Output()
		if err == nil {
			line := strings.TrimSpace(string(out))
			const prefix = "Current Wi-Fi Network: "
			if strings.HasPrefix(line, prefix) {
				return ConnectDoneMsg{SSID: strings.TrimPrefix(line, prefix)}
			}
		}

		// Fallback: ipconfig getsummary (macOS 13+ with redacted SSID)
		out, err = exec.Command("ipconfig", "getsummary", iface).Output()
		if err != nil {
			return ConnectDoneMsg{} // not connected
		}
		summary := string(out)
		connected, ssid := ParseIpconfigSummary(summary)
		if !connected {
			return ConnectDoneMsg{}
		}

		// SSID is redacted — resolve via gateway IP+MAC cache.
		// IP is free from ipconfig output; MAC (via ARP) makes the key unique
		// across networks that share the same gateway IP (e.g. 192.168.1.1).
		if ssid == "Wi-Fi" {
			routerIP := ParseRouterFromIpconfig(summary)
			if routerIP != "" {
				var mac string
				if arpOut, arpErr := exec.Command("arp", "-n", routerIP).Output(); arpErr == nil {
					mac = ParseARPMAC(string(arpOut))
				}
				if cached := LookupCachedSSID(routerIP, mac); cached != "" {
					ssid = cached
				}
			}
		}
		return ConnectDoneMsg{SSID: ssid}
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

// CacheNetworkCmd resolves the current gateway IP+MAC and saves the SSID
// to the local network cache. Fire-and-forget: returns nil Msg.
func CacheNetworkCmd(iface, ssid string) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("ipconfig", "getsummary", iface).Output()
		if err != nil {
			return nil
		}
		routerIP := ParseRouterFromIpconfig(string(out))
		if routerIP == "" {
			return nil
		}
		var mac string
		if arpOut, arpErr := exec.Command("arp", "-n", routerIP).Output(); arpErr == nil {
			mac = ParseARPMAC(string(arpOut))
		}
		SaveCachedSSID(routerIP, mac, ssid)
		return nil
	}
}

// PingInternetCmd checks internet connectivity by attempting a TCP connection
// to Google's public DNS (8.8.8.8:53). This works without root privileges
// unlike ICMP ping, and reliably indicates internet reachability.
func PingInternetCmd() tea.Cmd {
	return func() tea.Msg {
		conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 3*time.Second)
		if err != nil {
			return InternetCheckMsg{Reachable: false}
		}
		conn.Close()
		return InternetCheckMsg{Reachable: true}
	}
}
