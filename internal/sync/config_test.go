package sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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
	if _, err := os.Stat(path); err != nil {
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
	expected := map[string]bool{"auto-sync": true, "session-timeout": true, "git": true}
	for _, k := range keys {
		delete(expected, k)
	}
	if len(expected) > 0 {
		t.Errorf("missing config keys: %v", expected)
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

func TestConfig_Get_Git_Default(t *testing.T) {
	cfg := &Config{}
	val, err := cfg.Get("git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "" {
		t.Errorf("expected empty default, got %q", val)
	}
}

func TestConfig_Set_Git(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Set("git", "git@github.com:user/vault.git"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	val, _ := cfg.Get("git")
	if val != "git@github.com:user/vault.git" {
		t.Errorf("expected git URL, got %q", val)
	}
}

func TestConfig_Git_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{}
	_ = cfg.Set("git", "https://github.com/user/vault.git")
	_ = SaveConfig(dir, cfg)

	loaded, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	val, _ := loaded.Get("git")
	if val != "https://github.com/user/vault.git" {
		t.Errorf("expected git URL after round trip, got %q", val)
	}
}

func TestConfig_Keys_IncludeSSH(t *testing.T) {
	keys := ConfigKeys()
	expected := map[string]bool{"ssh-timeout": true, "ssh-confirm": true}
	for _, k := range keys {
		delete(expected, k)
	}
	if len(expected) > 0 {
		t.Errorf("missing config keys: %v", expected)
	}
}

func TestConfig_SSHTimeout_Default(t *testing.T) {
	cfg := &Config{}
	val, err := cfg.Get("ssh-timeout")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "480" {
		t.Errorf("expected default '480', got %q", val)
	}
	if cfg.SSHKeyTTL() != 480*time.Minute {
		t.Errorf("expected 480m TTL, got %v", cfg.SSHKeyTTL())
	}
}

func TestConfig_SSHTimeout_SetAndValidate(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Set("ssh-timeout", "60"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SSHKeyTTL() != time.Hour {
		t.Errorf("expected 1h TTL, got %v", cfg.SSHKeyTTL())
	}
	if err := cfg.Set("ssh-timeout", "0"); err != nil {
		t.Fatalf("zero must be accepted (disables SSH keys): %v", err)
	}
	if cfg.SSHKeyTTL() != 0 {
		t.Errorf("expected 0 TTL, got %v", cfg.SSHKeyTTL())
	}
	if err := cfg.Set("ssh-timeout", "-5"); err == nil {
		t.Error("expected error for negative timeout")
	}
	if err := cfg.Set("ssh-timeout", "soon"); err == nil {
		t.Error("expected error for non-numeric timeout")
	}
}

func TestConfig_SSHConfirm_DefaultOn(t *testing.T) {
	cfg := &Config{}
	val, err := cfg.Get("ssh-confirm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "on" || !cfg.SSHConfirmEnabled() {
		t.Errorf("expected confirm on by default, got %q", val)
	}
}

func TestConfig_SSHConfirm_SetOff(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Set("ssh-confirm", "off"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SSHConfirmEnabled() {
		t.Error("expected confirm off")
	}
	if err := cfg.Set("ssh-confirm", "maybe"); err == nil {
		t.Error("expected error for invalid value")
	}
}

func TestConfig_SSH_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{}
	_ = cfg.Set("ssh-timeout", "30")
	_ = cfg.Set("ssh-confirm", "off")
	_ = SaveConfig(dir, cfg)

	loaded, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.SSHKeyTTL() != 30*time.Minute || loaded.SSHConfirmEnabled() {
		t.Errorf("round trip lost ssh settings: ttl=%v confirm=%v", loaded.SSHKeyTTL(), loaded.SSHConfirmEnabled())
	}
}

func TestConfig_SSHUnlockPrompt_DefaultOn(t *testing.T) {
	cfg := &Config{}
	val, err := cfg.Get("ssh-unlock-prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "on" || !cfg.SSHUnlockPromptEnabled() {
		t.Errorf("expected unlock prompt on by default, got %q", val)
	}
	if err := cfg.Set("ssh-unlock-prompt", "off"); err != nil {
		t.Fatal(err)
	}
	if cfg.SSHUnlockPromptEnabled() {
		t.Error("expected unlock prompt off")
	}
	if err := cfg.Set("ssh-unlock-prompt", "sometimes"); err == nil {
		t.Error("expected error for invalid value")
	}
}

func TestConfig_SSHConfirmGrace(t *testing.T) {
	cfg := &Config{}
	val, err := cfg.Get("ssh-confirm-grace")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "60" || cfg.SSHConfirmGrace() != time.Minute {
		t.Errorf("expected default 60s, got %q / %v", val, cfg.SSHConfirmGrace())
	}
	if err := cfg.Set("ssh-confirm-grace", "0"); err != nil {
		t.Fatal(err)
	}
	if cfg.SSHConfirmGrace() != 0 {
		t.Error("expected 0 to disable the grace period")
	}
	if err := cfg.Set("ssh-confirm-grace", "300"); err != nil {
		t.Fatal(err)
	}
	if cfg.SSHConfirmGrace() != 5*time.Minute {
		t.Errorf("expected 5m, got %v", cfg.SSHConfirmGrace())
	}
	if err := cfg.Set("ssh-confirm-grace", "-1"); err == nil {
		t.Error("expected error for negative value")
	}
	if err := cfg.Set("ssh-confirm-grace", "soon"); err == nil {
		t.Error("expected error for non-numeric value")
	}
}
