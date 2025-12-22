package session

import (
	"os"
	"path/filepath"

	"github.com/michaelquigley/df/dd"
)

// LoadMainConfig loads the session configuration from the default location.
// Default path: ~/.config/sessionmixer/session.yaml
func LoadMainConfig() (*SessionConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	configPath := filepath.Join(home, ".config", "sessionmixer", "session.yaml")
	return LoadConfig(configPath)
}

// LoadConfig loads a session configuration from the specified path.
func LoadConfig(path string) (*SessionConfig, error) {
	return dd.NewYAMLFile[SessionConfig](path)
}
