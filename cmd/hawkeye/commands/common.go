package commands

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// MonitorConfig represents a stored monitor configuration
type MonitorConfig struct {
	URL       string            `json:"url"`
	Interval  string            `json:"interval"`
	Group     string            `json:"group,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Ignore    []string          `json:"ignore,omitempty"`
	CreatedAt string            `json:"created_at,omitempty"`
}

// getConfigDir returns the directory where config files are stored
func getConfigDir() (string, error) {
	// First try to get from viper
	configFile := viper.ConfigFileUsed()
	if configFile != "" {
		return filepath.Dir(configFile), nil
	}

	// Otherwise use home directory
	home, err := getUserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(home, ".hawkeye")
	// Create directory if it doesn't exist
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return "", err
		}
	}

	return configDir, nil
}
