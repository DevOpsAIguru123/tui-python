package wifi_test

import (
	"testing"

	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/config"
	"github.com/DevOpsAIguru123/productivity-tools/daily-tui/internal/wifi"

	tea "github.com/charmbracelet/bubbletea"
)

func makeCfg(dailyReset ...string) *config.Config {
	return &config.Config{WiFi: config.WiFiConfig{DailyResetNetworks: dailyReset}}
}

func TestWifiModelInitialState(t *testing.T) {
	m := wifi.New(makeCfg("Agora Public"))
	if m.Connecting() {
		t.Error("should not be connecting on init")
	}
	if m.Inputting() {
		t.Error("should not be in input mode on init")
	}
}

func TestWifiModelEnterOnDailyResetOpensInput(t *testing.T) {
	m := wifi.New(makeCfg("Agora Public"))
	m.SetNetworks([]wifi.Network{{SSID: "Agora Public", RSSI: -45}})
	m.SetIface("en0")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	wm := updated.(wifi.Model)
	if !wm.Inputting() {
		t.Error("expected input mode after enter on daily-reset network")
	}
}

func TestWifiModelEscCancelsInput(t *testing.T) {
	m := wifi.New(makeCfg("Agora Public"))
	m.SetNetworks([]wifi.Network{{SSID: "Agora Public", RSSI: -45}})
	m.SetIface("en0")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	wm := updated.(wifi.Model)
	updated, _ = wm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	wm2 := updated.(wifi.Model)
	if wm2.Inputting() {
		t.Error("expected input mode cancelled after esc")
	}
}

func TestWifiModelCursorNavigation(t *testing.T) {
	m := wifi.New(makeCfg())
	m.SetNetworks([]wifi.Network{
		{SSID: "Net1", RSSI: -40},
		{SSID: "Net2", RSSI: -60},
	})
	if m.Cursor() != 0 {
		t.Errorf("expected cursor at 0, got %d", m.Cursor())
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	wm := updated.(wifi.Model)
	if wm.Cursor() != 1 {
		t.Errorf("expected cursor at 1, got %d", wm.Cursor())
	}
}

func TestConnectDoneTriggersPingAndSetsCheckingInternet(t *testing.T) {
	m := wifi.New(makeCfg())
	m.SetIface("en0")

	// Simulate a successful connection
	updated, cmd := m.Update(wifi.ConnectDoneMsg{SSID: "TestNet"})
	wm := updated.(wifi.Model)
	if wm.Connected() != "TestNet" {
		t.Errorf("expected connected to TestNet, got %q", wm.Connected())
	}
	if !wm.CheckingInternet() {
		t.Error("expected checkingInternet to be true after connect")
	}
	if cmd == nil {
		t.Error("expected a command (PingInternetCmd) after successful connect")
	}
}

func TestInternetCheckMsgUpdatesHasInternet(t *testing.T) {
	m := wifi.New(makeCfg())
	m.SetIface("en0")

	// First connect to set checkingInternet
	updated, _ := m.Update(wifi.ConnectDoneMsg{SSID: "TestNet"})
	wm := updated.(wifi.Model)

	// Simulate internet reachable
	updated, _ = wm.Update(wifi.InternetCheckMsg{Reachable: true})
	wm = updated.(wifi.Model)
	if !wm.HasInternet() {
		t.Error("expected hasInternet to be true")
	}
	if wm.CheckingInternet() {
		t.Error("expected checkingInternet to be false after check completes")
	}

	// Simulate internet unreachable
	updated, _ = wm.Update(wifi.InternetCheckMsg{Reachable: false})
	wm = updated.(wifi.Model)
	if wm.HasInternet() {
		t.Error("expected hasInternet to be false")
	}
}

func TestEmptyConnectDoneClearsConnected(t *testing.T) {
	m := wifi.New(makeCfg())
	m.SetIface("en0")

	// First connect
	updated, _ := m.Update(wifi.ConnectDoneMsg{SSID: "TestNet"})
	wm := updated.(wifi.Model)
	if wm.Connected() != "TestNet" {
		t.Fatalf("expected connected to TestNet, got %q", wm.Connected())
	}

	// Simulate GetCurrentNetworkCmd returning empty (not connected)
	updated, _ = wm.Update(wifi.ConnectDoneMsg{})
	wm = updated.(wifi.Model)
	if wm.Connected() != "" {
		t.Errorf("expected connected to be empty, got %q", wm.Connected())
	}
	if wm.HasInternet() {
		t.Error("expected hasInternet to be false when not connected")
	}
}

func TestEnterOnConnectedNetworkOpensPasswordInput(t *testing.T) {
	m := wifi.New(makeCfg())
	m.SetNetworks([]wifi.Network{
		{SSID: "HomeNet", RSSI: -40},
		{SSID: "CoffeeShop", RSSI: -60},
	})
	m.SetIface("en0")

	// Simulate being connected to HomeNet
	updated, _ := m.Update(wifi.ConnectDoneMsg{SSID: "HomeNet"})
	wm := updated.(wifi.Model)

	// Simulate internet check completing
	updated, _ = wm.Update(wifi.InternetCheckMsg{Reachable: true})
	wm = updated.(wifi.Model)

	// Press Enter on connected network (cursor=0, which is HomeNet)
	updated, _ = wm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	wm = updated.(wifi.Model)
	if !wm.Inputting() {
		t.Error("expected input mode when pressing enter on connected network")
	}
}
