package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	WiFi WiFiConfig `yaml:"wifi"`
}

type WiFiConfig struct {
	DailyResetNetworks []string `yaml:"daily_reset_networks"`
}

// Load reads config from ~/.config/daily-tui/config.yaml.
// Returns an empty Config if the file does not exist.
func Load() (*Config, error) {
	return LoadFrom(filepath.Join(os.Getenv("HOME"), ".config", "daily-tui", "config.yaml"))
}

// LoadFrom reads config from an explicit path.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// IsDailyReset returns true if ssid is in the daily_reset_networks list.
func (c *Config) IsDailyReset(ssid string) bool {
	for _, n := range c.WiFi.DailyResetNetworks {
		if n == ssid {
			return true
		}
	}
	return false
}
