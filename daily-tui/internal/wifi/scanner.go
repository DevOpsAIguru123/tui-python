package wifi

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Network represents a nearby WiFi network.
type Network struct {
	SSID string
	RSSI int // signal strength in dBm, e.g. -45
}

var macRegex = regexp.MustCompile(`[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}:[0-9a-f]{2}`)

// ParseAirportOutput parses the output of `airport -s` into Networks.
// The first line (header) is skipped.
func ParseAirportOutput(output string) []Network {
	var networks []Network
	lines := strings.Split(output, "\n")
	if len(lines) <= 1 {
		return networks
	}
	for _, line := range lines[1:] {
		n, ok := parseAirportLine(line)
		if ok {
			networks = append(networks, n)
		}
	}
	return networks
}

func parseAirportLine(line string) (Network, bool) {
	idx := macRegex.FindStringIndex(line)
	if idx == nil {
		return Network{}, false
	}
	ssid := strings.TrimSpace(line[:idx[0]])
	if ssid == "" {
		return Network{}, false
	}
	rest := strings.TrimSpace(line[idx[1]:])
	fields := strings.Fields(rest)
	if len(fields) < 1 {
		return Network{}, false
	}
	rssi, err := strconv.Atoi(fields[0])
	if err != nil {
		return Network{}, false
	}
	return Network{SSID: ssid, RSSI: rssi}, true
}

// DetectWiFiInterface parses `networksetup -listallhardwareports` output
// and returns the device name for the Wi-Fi interface (e.g. "en0").
func DetectWiFiInterface(output string) (string, error) {
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "Wi-Fi") || strings.Contains(trimmed, "AirPort") {
			if i+1 < len(lines) {
				deviceLine := strings.TrimSpace(lines[i+1])
				if strings.HasPrefix(deviceLine, "Device: ") {
					return strings.TrimPrefix(deviceLine, "Device: "), nil
				}
			}
		}
	}
	return "", fmt.Errorf("no Wi-Fi interface found in hardware ports output")
}

// SignalBars converts RSSI (dBm) to a 0–4 bar rating.
func SignalBars(rssi int) int {
	switch {
	case rssi >= -50:
		return 4
	case rssi >= -60:
		return 3
	case rssi >= -70:
		return 2
	case rssi >= -80:
		return 1
	default:
		return 0
	}
}

// BarString returns a 4-char block string for the given bar count (0–4).
func BarString(bars int) string {
	blocks := []string{"    ", "█   ", "██  ", "███ ", "████"}
	if bars < 0 {
		bars = 0
	}
	if bars > 4 {
		bars = 4
	}
	return blocks[bars]
}

// ParseIpconfigSummary parses `ipconfig getsummary <iface>` output.
// Returns (connected bool, ssid string). On macOS 13+, the SSID may be the
// literal string "<redacted>" when Location Services are not granted to the
// terminal; in that case ssid is returned as "Wi-Fi" so the UI can still
// show a connected state rather than "Not connected".
func ParseIpconfigSummary(output string) (bool, string) {
	if !strings.Contains(output, "LinkStatusActive : TRUE") {
		return false, ""
	}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SSID : ") {
			ssid := strings.TrimSpace(strings.TrimPrefix(line, "SSID : "))
			if ssid == "<redacted>" || ssid == "" {
				return true, "Wi-Fi"
			}
			return true, ssid
		}
	}
	return true, "Wi-Fi"
}

// ParseRouterFromIpconfig extracts the router (gateway) IP from `ipconfig getsummary` output.
// Returns "" if not found.
func ParseRouterFromIpconfig(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Router : ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Router : "))
		}
	}
	return ""
}

// ParsePreferredNetworks parses `networksetup -listpreferredwirelessnetworks` output.
// macOS 15+ redacts SSIDs in airport/system_profiler, but this command returns real names.
// The first line is a header ("Preferred networks on <iface>:") and is skipped.
func ParsePreferredNetworks(output string) []Network {
	var networks []Network
	lines := strings.Split(output, "\n")
	if len(lines) <= 1 {
		return networks
	}
	for _, line := range lines[1:] {
		ssid := strings.TrimSpace(line)
		if ssid != "" {
			networks = append(networks, Network{SSID: ssid, RSSI: 0})
		}
	}
	return networks
}
