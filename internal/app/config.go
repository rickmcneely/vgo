package app

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// IDEConfig stores the IDE layout configuration
type IDEConfig struct {
	WindowX      int `json:"window_x"`
	WindowY      int `json:"window_y"`
	WindowWidth  int `json:"window_width"`
	WindowHeight int `json:"window_height"`
	Maximized    bool `json:"maximized"`

	// Splitter positions (widths/heights in pixels)
	LeftPanelWidth     int `json:"left_panel_width"`
	RightPanelWidth    int `json:"right_panel_width"`
	ProjectPanelHeight int `json:"project_panel_height"`
	DebugPanelHeight   int `json:"debug_panel_height"`
}

// DefaultConfig returns the default IDE configuration
func DefaultConfig() *IDEConfig {
	return &IDEConfig{
		WindowX:            100,
		WindowY:            100,
		WindowWidth:        1280,
		WindowHeight:       800,
		Maximized:          false,
		LeftPanelWidth:     180,
		RightPanelWidth:    220,
		ProjectPanelHeight: 200,
		DebugPanelHeight:   150,
	}
}

// ConfigPath returns the path to the config file
func ConfigPath() string {
	exePath, err := os.Executable()
	if err != nil {
		return "vgeditor.config"
	}
	return filepath.Join(filepath.Dir(exePath), "vgeditor.config")
}

// LoadConfig loads the IDE configuration from file
func LoadConfig() *IDEConfig {
	config := DefaultConfig()
	configPath := ConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return config
	}

	if err := json.Unmarshal(data, config); err != nil {
		return DefaultConfig()
	}

	// Validate values
	if config.DebugPanelHeight < 50 {
		config.DebugPanelHeight = 50
	}
	if config.DebugPanelHeight > 500 {
		config.DebugPanelHeight = 500
	}

	return config
}

// Save saves the IDE configuration to file
func (c *IDEConfig) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), data, 0644)
}
