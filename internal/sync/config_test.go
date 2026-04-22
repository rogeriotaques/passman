package sync

import (
	"path/filepath"
	"testing"
)

func TestLoadConfig_Default(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AutoSync {
		t.Error("expected AutoSync to default to false")
	}
}

func TestSaveAndLoadConfig_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{AutoSync: true}
	if err := SaveConfig(dir, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !loaded.AutoSync {
		t.Error("expected AutoSync to be true after save")
	}
}

func TestSaveConfig_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{AutoSync: false}
	if err := SaveConfig(dir, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	path := filepath.Join(dir, "config.json")
	if !fileExists(path) {
		t.Error("expected config.json to exist")
	}
}

func TestSaveConfig_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()

	_ = SaveConfig(dir, &Config{AutoSync: true})
	_ = SaveConfig(dir, &Config{AutoSync: false})

	loaded, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.AutoSync {
		t.Error("expected AutoSync to be false after overwrite")
	}
}

func TestConfig_Get_AutoSync(t *testing.T) {
	cfg := &Config{AutoSync: true}
	val, err := cfg.Get("auto-sync")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "on" {
		t.Errorf("expected 'on', got %q", val)
	}
}

func TestConfig_Get_AutoSync_Off(t *testing.T) {
	cfg := &Config{}
	val, err := cfg.Get("auto-sync")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "off" {
		t.Errorf("expected 'off', got %q", val)
	}
}

func TestConfig_Get_Unknown(t *testing.T) {
	cfg := &Config{}
	_, err := cfg.Get("nonexistent")
	if err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestConfig_Set_AutoSync_On(t *testing.T) {
	cfg := &Config{}
	err := cfg.Set("auto-sync", "on")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.AutoSync {
		t.Error("expected AutoSync to be true")
	}
}

func TestConfig_Set_AutoSync_Off(t *testing.T) {
	cfg := &Config{AutoSync: true}
	err := cfg.Set("auto-sync", "off")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AutoSync {
		t.Error("expected AutoSync to be false")
	}
}

func TestConfig_Set_InvalidValue(t *testing.T) {
	cfg := &Config{}
	err := cfg.Set("auto-sync", "maybe")
	if err == nil {
		t.Error("expected error for invalid value")
	}
}

func TestConfig_Set_UnknownKey(t *testing.T) {
	cfg := &Config{}
	err := cfg.Set("nonexistent", "value")
	if err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestConfig_Keys(t *testing.T) {
	keys := ConfigKeys()
	if len(keys) == 0 {
		t.Error("expected at least one config key")
	}
	found := false
	for _, k := range keys {
		if k == "auto-sync" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'auto-sync' in config keys")
	}
}

func TestConfig_Get_SessionTimeout_Default(t *testing.T) {
	cfg := &Config{}
	val, err := cfg.Get("session-timeout")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "15" {
		t.Errorf("expected default '15', got %q", val)
	}
}

func TestConfig_Set_SessionTimeout(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Set("session-timeout", "10"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	val, _ := cfg.Get("session-timeout")
	if val != "10" {
		t.Errorf("expected '10', got %q", val)
	}
}

func TestConfig_Set_SessionTimeout_Zero(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Set("session-timeout", "0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	val, _ := cfg.Get("session-timeout")
	if val != "0" {
		t.Errorf("expected '0', got %q", val)
	}
}

func TestConfig_Set_SessionTimeout_Negative(t *testing.T) {
	cfg := &Config{}
	err := cfg.Set("session-timeout", "-1")
	if err == nil {
		t.Error("expected error for negative timeout")
	}
}

func TestConfig_Set_SessionTimeout_NonNumeric(t *testing.T) {
	cfg := &Config{}
	err := cfg.Set("session-timeout", "abc")
	if err == nil {
		t.Error("expected error for non-numeric timeout")
	}
}

func TestConfig_SessionTimeout_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{}
	_ = cfg.Set("session-timeout", "10")
	_ = SaveConfig(dir, cfg)

	loaded, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	val, _ := loaded.Get("session-timeout")
	if val != "10" {
		t.Errorf("expected '10' after round trip, got %q", val)
	}
}

func TestConfig_Keys_IncludesSessionTimeout(t *testing.T) {
	keys := ConfigKeys()
	found := false
	for _, k := range keys {
		if k == "session-timeout" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'session-timeout' in config keys")
	}
}
