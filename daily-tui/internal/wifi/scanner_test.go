package wifi_test

import (
	"testing"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/wifi"
)

var airportOutput = `                            SSID BSSID             RSSI CHANNEL HT CC SECURITY (auth/unicast/group)
                     Agora Public aa:bb:cc:dd:ee:ff  -45       6  Y  US WPA2(PSK/AES/AES)
                   CoffeeShop_2G 11:22:33:44:55:66  -62      11  Y  US WPA2(PSK/AES/AES)
                  iPhone Hotspot 77:88:99:aa:bb:cc  -71       1  Y  US WPA2(PSK/AES/AES)`

func TestParseAirportOutput(t *testing.T) {
	networks := wifi.ParseAirportOutput(airportOutput)
	if len(networks) != 3 {
		t.Fatalf("expected 3 networks, got %d", len(networks))
	}
	if networks[0].SSID != "Agora Public" {
		t.Errorf("expected 'Agora Public', got %q", networks[0].SSID)
	}
	if networks[0].RSSI != -45 {
		t.Errorf("expected RSSI -45, got %d", networks[0].RSSI)
	}
	if networks[1].SSID != "CoffeeShop_2G" {
		t.Errorf("expected 'CoffeeShop_2G', got %q", networks[1].SSID)
	}
	if networks[2].SSID != "iPhone Hotspot" {
		t.Errorf("expected 'iPhone Hotspot', got %q", networks[2].SSID)
	}
}

func TestParseAirportOutputEmpty(t *testing.T) {
	networks := wifi.ParseAirportOutput("")
	if len(networks) != 0 {
		t.Fatalf("expected 0 networks for empty input, got %d", len(networks))
	}
}

var hardwarePortsOutput = `Hardware Port: Wi-Fi
Device: en0
Ethernet Address: aa:bb:cc:dd:ee:ff

Hardware Port: Bluetooth PAN
Device: en1
Ethernet Address: ff:ee:dd:cc:bb:aa`

func TestDetectWiFiInterface(t *testing.T) {
	iface, err := wifi.DetectWiFiInterface(hardwarePortsOutput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if iface != "en0" {
		t.Errorf("expected 'en0', got %q", iface)
	}
}

func TestDetectWiFiInterfaceNotFound(t *testing.T) {
	_, err := wifi.DetectWiFiInterface("Hardware Port: Ethernet\nDevice: en0\n")
	if err == nil {
		t.Error("expected error when no Wi-Fi interface found")
	}
}

var ipconfigRedacted = `<dictionary> {
  BSSID : <redacted>
  ConnectionID : 16
  IPv4 : <array> {
    0 : <dictionary> {
      Addresses : <array> {
        0 : 192.168.1.164
      }
    }
  }
  InterfaceType : WiFi
  LinkStatusActive : TRUE
  SSID : <redacted>
  Security : WPA2_PSK
}`

var ipconfigRealSSID = `<dictionary> {
  BSSID : aa:bb:cc:dd:ee:ff
  InterfaceType : WiFi
  LinkStatusActive : TRUE
  SSID : Agora Public
}`

var ipconfigNotConnected = `<dictionary> {
  InterfaceType : WiFi
  LinkStatusActive : FALSE
}`

func TestParseIpconfigSummaryRedacted(t *testing.T) {
	connected, ssid := wifi.ParseIpconfigSummary(ipconfigRedacted)
	if !connected {
		t.Error("expected connected=true when LinkStatusActive is TRUE")
	}
	if ssid != "Wi-Fi" {
		t.Errorf("expected ssid='Wi-Fi' for redacted SSID, got %q", ssid)
	}
}

func TestParseIpconfigSummaryRealSSID(t *testing.T) {
	connected, ssid := wifi.ParseIpconfigSummary(ipconfigRealSSID)
	if !connected {
		t.Error("expected connected=true")
	}
	if ssid != "Agora Public" {
		t.Errorf("expected ssid='Agora Public', got %q", ssid)
	}
}

func TestParseIpconfigSummaryNotConnected(t *testing.T) {
	connected, ssid := wifi.ParseIpconfigSummary(ipconfigNotConnected)
	if connected {
		t.Error("expected connected=false when LinkStatusActive is FALSE")
	}
	if ssid != "" {
		t.Errorf("expected empty ssid, got %q", ssid)
	}
}

func TestParseRouterFromIpconfig(t *testing.T) {
	got := wifi.ParseRouterFromIpconfig(ipconfigRedacted)
	// ipconfigRedacted test fixture doesn't have a Router line — should return ""
	if got != "" {
		t.Errorf("expected empty router from test fixture, got %q", got)
	}

	withRouter := ipconfigRedacted + "\n  Router : 10.0.0.1\n"
	got = wifi.ParseRouterFromIpconfig(withRouter)
	if got != "10.0.0.1" {
		t.Errorf("expected '10.0.0.1', got %q", got)
	}
}

func TestParseARPMAC(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"? (10.0.0.1) at 1c:93:7c:c3:8f:8e on en0 ifscope [ethernet]\n", "1c:93:7c:c3:8f:8e"},
		{"? (192.168.1.1) at (incomplete)  on en0\n", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := wifi.ParseARPMAC(tt.input)
		if got != tt.want {
			t.Errorf("ParseARPMAC(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSignalBars(t *testing.T) {
	tests := []struct {
		rssi int
		want int
	}{
		{-40, 4},
		{-55, 3},
		{-67, 2},
		{-80, 1},
		{-90, 0},
	}
	for _, tt := range tests {
		got := wifi.SignalBars(tt.rssi)
		if got != tt.want {
			t.Errorf("SignalBars(%d) = %d, want %d", tt.rssi, got, tt.want)
		}
	}
}
