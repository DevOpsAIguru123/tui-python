package wifi_test

import (
	"testing"

	"daily-tui/internal/config"
	"daily-tui/internal/wifi"

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
