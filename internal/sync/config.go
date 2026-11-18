package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const configFile = "config.json"

const (
	DefaultSessionTimeout = 15
	// DefaultSSHTimeout keeps SSH keys loaded for a working day. Signing still
	// requires confirmation, so this is less exposed than the password cache.
	DefaultSSHTimeout = 480
	// DefaultSSHConfirmGrace (seconds) covers the few connections one git
	// operation opens without re-asking for each of them.
	DefaultSSHConfirmGrace = 60
)

type Config struct {
	AutoSync               bool   `json:"auto_sync"`
	SessionTimeout         *int   `json:"session_timeout,omitempty"`
	GitRemote              string `json:"git_remote,omitempty"`
	NoPassword             bool   `json:"no_password,omitempty"`
	SSHTimeout             *int   `json:"ssh_timeout,omitempty"`
	SSHConfirm             *bool  `json:"ssh_confirm,omitempty"`
	SSHConfirmGraceSeconds *int   `json:"ssh_confirm_grace,omitempty"`
	SSHUnlockPrompt        *bool  `json:"ssh_unlock_prompt,omitempty"`
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

// SSHKeyTTL is how long the agent serves this vault's SSH keys after an
// unlock. Zero disables SSH keys for the vault.
func (c *Config) SSHKeyTTL() time.Duration {
	if c.SSHTimeout == nil {
		return DefaultSSHTimeout * time.Minute
	}
	return time.Duration(*c.SSHTimeout) * time.Minute
}

// SSHConfirmEnabled reports whether each SSH signature needs user approval.
func (c *Config) SSHConfirmEnabled() bool {
	return c.SSHConfirm == nil || *c.SSHConfirm
}

// SSHConfirmGrace is how long an approval for a program and key stays valid.
func (c *Config) SSHConfirmGrace() time.Duration {
	if c.SSHConfirmGraceSeconds == nil {
		return DefaultSSHConfirmGrace * time.Second
	}
	return time.Duration(*c.SSHConfirmGraceSeconds) * time.Second
}

// SSHUnlockPromptEnabled reports whether the agent may ask to unlock this
// vault when an SSH client finds no keys loaded.
func (c *Config) SSHUnlockPromptEnabled() bool {
	return c.SSHUnlockPrompt == nil || *c.SSHUnlockPrompt
}

func (c *Config) Get(key string) (string, error) {
	switch key {
	case "auto-sync":
		return onOff(c.AutoSync), nil
	case "session-timeout":
		if c.SessionTimeout == nil {
			return strconv.Itoa(DefaultSessionTimeout), nil
		}
		return strconv.Itoa(*c.SessionTimeout), nil
	case "git":
		return c.GitRemote, nil
	case "ssh-timeout":
		return strconv.Itoa(int(c.SSHKeyTTL() / time.Minute)), nil
	case "ssh-confirm":
		return onOff(c.SSHConfirmEnabled()), nil
	case "ssh-confirm-grace":
		return strconv.Itoa(int(c.SSHConfirmGrace() / time.Second)), nil
	case "ssh-unlock-prompt":
		return onOff(c.SSHUnlockPromptEnabled()), nil
	default:
		return "", fmt.Errorf("unknown config key: %q (available: %v)", key, ConfigKeys())
	}
}

func (c *Config) Set(key, value string) error {
	switch key {
	case "auto-sync":
		on, err := parseOnOff(key, value)
		if err != nil {
			return err
		}
		c.AutoSync = on
	case "session-timeout":
		n, err := parseMinutes(key, value)
		if err != nil {
			return err
		}
		c.SessionTimeout = &n
	case "git":
		c.GitRemote = value
	case "ssh-timeout":
		n, err := parseMinutes(key, value)
		if err != nil {
			return err
		}
		c.SSHTimeout = &n
	case "ssh-confirm":
		on, err := parseOnOff(key, value)
		if err != nil {
			return err
		}
		c.SSHConfirm = &on
	case "ssh-confirm-grace":
		n, err := parseNonNegative(key, value, "seconds")
		if err != nil {
			return err
		}
		c.SSHConfirmGraceSeconds = &n
	case "ssh-unlock-prompt":
		on, err := parseOnOff(key, value)
		if err != nil {
			return err
		}
		c.SSHUnlockPrompt = &on
	default:
		return fmt.Errorf("unknown config key: %q (available: %v)", key, ConfigKeys())
	}
	return nil
}

func onOff(on bool) string {
	if on {
		return "on"
	}
	return "off"
}

func parseOnOff(key, value string) (bool, error) {
	switch value {
	case "on":
		return true, nil
	case "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid value for %q: %q (use 'on' or 'off')", key, value)
	}
}

func parseMinutes(key, value string) (int, error) {
	return parseNonNegative(key, value, "minutes")
}

func parseNonNegative(key, value, unit string) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid value for %q: %q (must be a number of %s)", key, value, unit)
	}
	if n < 0 {
		return 0, fmt.Errorf("invalid value for %q: %q (must be >= 0)", key, value)
	}
	return n, nil
}

func ConfigKeys() []string {
	return []string{"auto-sync", "session-timeout", "git", "ssh-timeout", "ssh-confirm", "ssh-confirm-grace", "ssh-unlock-prompt"}
}
