package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

const configFile = "config.json"

const DefaultSessionTimeout = 15

type Config struct {
	AutoSync       bool `json:"auto_sync"`
	SessionTimeout *int `json:"session_timeout,omitempty"`
}

func LoadConfig(dir string) (*Config, error) {
	path := filepath.Join(dir, configFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func SaveConfig(dir string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, configFile), data, 0600)
}

func (c *Config) Get(key string) (string, error) {
	switch key {
	case "auto-sync":
		if c.AutoSync {
			return "on", nil
		}
		return "off", nil
	case "session-timeout":
		if c.SessionTimeout == nil {
			return strconv.Itoa(DefaultSessionTimeout), nil
		}
		return strconv.Itoa(*c.SessionTimeout), nil
	default:
		return "", fmt.Errorf("unknown config key: %q (available: %v)", key, ConfigKeys())
	}
}

func (c *Config) Set(key, value string) error {
	switch key {
	case "auto-sync":
		switch value {
		case "on":
			c.AutoSync = true
		case "off":
			c.AutoSync = false
		default:
			return fmt.Errorf("invalid value for %q: %q (use 'on' or 'off')", key, value)
		}
		return nil
	case "session-timeout":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid value for %q: %q (must be a number of minutes)", key, value)
		}
		if n < 0 {
			return fmt.Errorf("invalid value for %q: %q (must be >= 0)", key, value)
		}
		c.SessionTimeout = &n
		return nil
	default:
		return fmt.Errorf("unknown config key: %q (available: %v)", key, ConfigKeys())
	}
}

func ConfigKeys() []string {
	return []string{"auto-sync", "session-timeout"}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
