package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	WiFi       WiFiConfig       `yaml:"wifi"`
	ClaudeCode ClaudeCodeConfig `yaml:"claude_code"`
}

type WiFiConfig struct {
	DailyResetNetworks []string `yaml:"daily_reset_networks"`
}

// ClaudeCodeConfig holds user-defined entries for the Claude tab's command
// launcher. Each entry is fed straight into the existing Command struct so
// the tab can execute it the same way as builtins and on-disk markdown
// commands. See daily-tui/README.md for the YAML shape.
type ClaudeCodeConfig struct {
	Commands []ClaudeCodeCommand `yaml:"commands"`
}

// ClaudeCodeCommand mirrors claudecode.Command but without importing that
// package (config is imported by many tabs and shouldn't pull in the tab's
// internals). The claudecode package maps these into its own Command type.
type ClaudeCodeCommand struct {
	Name      string   `yaml:"name"`
	Prompt    string   `yaml:"prompt"`
	ExtraArgs []string `yaml:"extra_args,omitempty"`
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
