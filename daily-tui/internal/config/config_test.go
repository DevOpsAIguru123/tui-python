package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"daily-tui/internal/config"
)

func TestLoadMissingFile(t *testing.T) {
	cfg, err := config.LoadFrom(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestLoadDailyResetNetworks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("wifi:\n  daily_reset_networks:\n    - \"Agora Public\"\n    - \"CoffeeShop_Guest\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.WiFi.DailyResetNetworks) != 2 {
		t.Fatalf("expected 2 networks, got %d", len(cfg.WiFi.DailyResetNetworks))
	}
}

func TestIsDailyReset(t *testing.T) {
	cfg := &config.Config{
		WiFi: config.WiFiConfig{
			DailyResetNetworks: []string{"Agora Public", "CoffeeShop_Guest"},
		},
	}
	if !cfg.IsDailyReset("Agora Public") {
		t.Error("expected Agora Public to be daily reset")
	}
	if cfg.IsDailyReset("MyHomeWifi") {
		t.Error("expected MyHomeWifi to NOT be daily reset")
	}
}
