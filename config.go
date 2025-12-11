package sessionmixer

import (
	"os"
	"path/filepath"

	"github.com/michaelquigley/df/dd"
)

type Config struct {
	Card           int `dd:"+required"`
	LevelSmoothing int // Number of samples to average for level meters (0 = disabled)
	GangControls   []GangControl
}

type GangControl struct {
	Name           string   `dd:"+required"`
	Controls       []string `dd:"+required"`
	Unit           string
	TaperDb        float32  // If > 0, use DecibelTaper(TaperDb); otherwise LinearTaper
	Levels         []string // Optional level control names for signal indication
	ShowVUMeter    bool     // Enable VUMeter display (requires Levels)
	ShowTrackColor bool     // Enable track coloring based on level (requires Levels)
}

func LoadMainConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	configPath := filepath.Join(home, ".config", "sessionmixer", "session.yaml")
	return LoadConfig(configPath)
}

func LoadConfig(path string) (*Config, error) {
	return dd.NewYAMLFile[Config](path)
}
