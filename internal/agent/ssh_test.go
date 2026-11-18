package agent

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rogerio/passman/internal/crypto"
	"github.com/rogerio/passman/internal/sshagent"
	psync "github.com/rogerio/passman/internal/sync"
	"github.com/rogerio/passman/internal/vault"
	"golang.org/x/crypto/ssh"
	sshagentproto "golang.org/x/crypto/ssh/agent"
)

func testKeyPEM(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(block))
}

// writeTestVault creates an encrypted vault holding the given entries and
// returns its path.
func writeTestVault(t *testing.T, password string, entries ...vault.Entry) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "vault.enc")
	store := &vault.Store{Path: path, KDFParams: crypto.FastKDFParams()}
	v := &vault.Vault{Entries: entries}
	if err := store.Save(v, []byte(password)); err != nil {
		t.Fatal(err)
	}
	return path
}

func startSSHServer(t *testing.T, cfg SSHConfig) (*Server, *Client, string) {
	t.Helper()
	sock := shortSocketPath(t)
	cfg.SocketPath = filepath.Join(filepath.Dir(sock), "ssh.sock")
	srv := NewServer(sock)
	srv.EnableSSH(cfg)
	go srv.Start()

	client := NewClient(sock)
	for i := 0; i < 20; i++ {
		if client.Ping() {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Cleanup(func() { srv.Stop() })
	return srv, client, cfg.SocketPath
}

func sshKeys(t *testing.T, sshSock string) []*sshagentproto.Key {
	t.Helper()
	conn, err := net.Dial("unix", sshSock)
	if err != nil {
		t.Fatalf("dial ssh socket: %v", err)
	}
	defer conn.Close()
	keys, err := sshagentproto.NewClient(conn).List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	return keys
}

func TestServer_StoreLoadsSSHKeys(t *testing.T) {
	vaultPath := writeTestVault(t, "pw",
		vault.Entry{Name: "ssh/id", Value: testKeyPEM(t)},
		vault.Entry{Name: "token", Value: "not a key"},
	)
	_, client, sshSock := startSSHServer(t, SSHConfig{Confirm: func(sshagent.SignRequest) bool { return true }})

	if err := client.Store(vaultPath, []byte("pw"), time.Minute); err != nil {
		t.Fatalf("store: %v", err)
	}
	keys := sshKeys(t, sshSock)
	if len(keys) != 1 || keys[0].Comment != "ssh/id" {
		t.Fatalf("expected the vault's key to be served, got %+v", keys)
	}

	if err := client.Lock(); err != nil {
		t.Fatalf("lock: %v", err)
	}
	if keys := sshKeys(t, sshSock); len(keys) != 0 {
		t.Errorf("expected no keys after lock, got %d", len(keys))
	}
	if !client.Ping() {
		t.Error("control socket should still answer after lock")
	}
}

func TestServer_StoreReloadsWhenVaultChanges(t *testing.T) {
	vaultPath := writeTestVault(t, "pw", vault.Entry{Name: "ssh/one", Value: testKeyPEM(t)})
	_, client, sshSock := startSSHServer(t, SSHConfig{})

	_ = client.Store(vaultPath, []byte("pw"), time.Minute)
	_ = client.Store(vaultPath, []byte("pw"), time.Minute)
	if keys := sshKeys(t, sshSock); len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}

	store := &vault.Store{Path: vaultPath, KDFParams: crypto.FastKDFParams()}
	v, err := store.Load([]byte("pw"))
	if err != nil {
		t.Fatal(err)
	}
	_ = v.Add(vault.Entry{Name: "ssh/two", Value: testKeyPEM(t)})
	time.Sleep(10 * time.Millisecond)
	if err := store.Save(v, []byte("pw")); err != nil {
		t.Fatal(err)
	}

	_ = client.Store(vaultPath, []byte("pw"), time.Minute)
	if keys := sshKeys(t, sshSock); len(keys) != 2 {
		t.Errorf("expected 2 keys after the vault changed, got %d", len(keys))
	}
}

func TestServer_SSHDisabledForVault(t *testing.T) {
	vaultPath := writeTestVault(t, "pw", vault.Entry{Name: "ssh/id", Value: testKeyPEM(t)})
	cfg := &psync.Config{}
	_ = cfg.Set("ssh-timeout", "0")
	if err := psync.SaveConfig(filepath.Dir(vaultPath), cfg); err != nil {
		t.Fatal(err)
	}
	_, client, sshSock := startSSHServer(t, SSHConfig{})

	_ = client.Store(vaultPath, []byte("pw"), time.Minute)
	if keys := sshKeys(t, sshSock); len(keys) != 0 {
		t.Errorf("ssh-timeout 0 must keep keys out of the agent, got %d", len(keys))
	}
}

func TestServer_StoreWithMissingVaultIsHarmless(t *testing.T) {
	_, client, sshSock := startSSHServer(t, SSHConfig{})

	if err := client.Store("/vault/missing.enc", []byte("pw"), time.Minute); err != nil {
		t.Fatalf("store: %v", err)
	}
	if keys := sshKeys(t, sshSock); len(keys) != 0 {
		t.Errorf("expected no keys, got %d", len(keys))
	}
}

func TestServer_SSHOnDemandUnlock(t *testing.T) {
	vaultPath := writeTestVault(t, "pw", vault.Entry{Name: "ssh/id", Value: testKeyPEM(t)})
	prompts := 0
	_, client, sshSock := startSSHServer(t, SSHConfig{
		UnlockVault: vaultPath,
		PromptPassword: func(string) ([]byte, error) {
			prompts++
			return []byte("pw"), nil
		},
	})

	keys := sshKeys(t, sshSock)
	if len(keys) != 1 || prompts != 1 {
		t.Fatalf("expected on-demand unlock to load 1 key with 1 prompt, got %d keys, %d prompts", len(keys), prompts)
	}
	if _, err := client.Retrieve(vaultPath); err == nil {
		t.Error("on-demand unlock must not cache the master password")
	}

	sshKeys(t, sshSock)
	if prompts != 1 {
		t.Error("no prompt expected while keys are loaded")
	}
}

func TestServer_SSHOnDemandUnlock_WrongPasswordAndCooldown(t *testing.T) {
	vaultPath := writeTestVault(t, "pw", vault.Entry{Name: "ssh/id", Value: testKeyPEM(t)})
	prompts := 0
	_, _, sshSock := startSSHServer(t, SSHConfig{
		UnlockVault: vaultPath,
		PromptPassword: func(string) ([]byte, error) {
			prompts++
			return []byte("wrong"), nil
		},
	})

	if keys := sshKeys(t, sshSock); len(keys) != 0 {
		t.Fatalf("wrong password must not load keys, got %d", len(keys))
	}
	sshKeys(t, sshSock)
	if prompts != 1 {
		t.Errorf("expected a single prompt within the cooldown, got %d", prompts)
	}
}

func TestServer_SSHOnDemandUnlock_NoPasswordVault(t *testing.T) {
	vaultPath := writeTestVault(t, "", vault.Entry{Name: "ssh/id", Value: testKeyPEM(t)})
	cfg := &psync.Config{NoPassword: true}
	if err := psync.SaveConfig(filepath.Dir(vaultPath), cfg); err != nil {
		t.Fatal(err)
	}
	_, _, sshSock := startSSHServer(t, SSHConfig{
		UnlockVault: vaultPath,
		PromptPassword: func(string) ([]byte, error) {
			t.Fatal("must not prompt for a passwordless vault")
			return nil, nil
		},
	})

	if keys := sshKeys(t, sshSock); len(keys) != 1 {
		t.Errorf("expected 1 key, got %d", len(keys))
	}
}

func TestServer_Idle(t *testing.T) {
	srv := NewServer(shortSocketPath(t))
	if !srv.idle() {
		t.Error("a fresh server with no SSH support is idle")
	}

	srv.EnableSSH(SSHConfig{SocketPath: "x", UnlockVault: "/vault.enc"})
	if srv.idle() {
		t.Error("a server that can unlock on demand must stay alive for ssh clients")
	}

	srv = NewServer(shortSocketPath(t))
	srv.EnableSSH(SSHConfig{SocketPath: "x"})
	srv.keyring.Load("v", []vault.Entry{{Name: "k", Value: testKeyPEM(t)}}, sshagent.LoadOptions{TTL: time.Hour})
	if srv.idle() {
		t.Error("a server holding ssh keys is not idle")
	}
}

func TestServer_StopRemovesSSHSocket(t *testing.T) {
	srv, _, sshSock := startSSHServer(t, SSHConfig{})
	if _, err := os.Stat(sshSock); err != nil {
		t.Fatalf("ssh socket should exist while running: %v", err)
	}
	srv.Stop()
	if _, err := os.Stat(sshSock); !os.IsNotExist(err) {
		t.Error("ssh socket should be removed on stop")
	}
}

func TestServer_SSHOnDemandUnlock_BacksOffAfterCancel(t *testing.T) {
	vaultPath := writeTestVault(t, "pw", vault.Entry{Name: "ssh/id", Value: testKeyPEM(t)})
	prompts := 0
	srv, _, sshSock := startSSHServer(t, SSHConfig{
		UnlockVault: vaultPath,
		PromptPassword: func(string) ([]byte, error) {
			prompts++
			return nil, nil // cancelled
		},
	})

	sshKeys(t, sshSock) // 1st prompt, cancelled: cooldown doubles to 20s
	srv.mu.Lock()
	srv.ssh.lastUnlockPrompt = time.Now().Add(-25 * time.Second)
	srv.mu.Unlock()
	sshKeys(t, sshSock) // 2nd prompt, cancelled: cooldown doubles to 40s
	if prompts != 2 {
		t.Fatalf("expected a second prompt after the doubled cooldown, got %d", prompts)
	}

	srv.mu.Lock()
	srv.ssh.lastUnlockPrompt = time.Now().Add(-25 * time.Second)
	srv.mu.Unlock()
	sshKeys(t, sshSock)
	if prompts != 2 {
		t.Errorf("25s is inside the 40s cooldown after two cancellations, got %d prompts", prompts)
	}
}

func TestServer_SSHOnDemandUnlock_Disabled(t *testing.T) {
	vaultPath := writeTestVault(t, "pw", vault.Entry{Name: "ssh/id", Value: testKeyPEM(t)})
	cfg := &psync.Config{}
	_ = cfg.Set("ssh-unlock-prompt", "off")
	if err := psync.SaveConfig(filepath.Dir(vaultPath), cfg); err != nil {
		t.Fatal(err)
	}
	_, _, sshSock := startSSHServer(t, SSHConfig{
		UnlockVault: vaultPath,
		PromptPassword: func(string) ([]byte, error) {
			t.Fatal("must not prompt when ssh-unlock-prompt is off")
			return nil, nil
		},
	})

	if keys := sshKeys(t, sshSock); len(keys) != 0 {
		t.Errorf("expected no keys, got %d", len(keys))
	}
}

func TestServer_StartReturnsOnlyAfterCleanup(t *testing.T) {
	sock := shortSocketPath(t)
	sshSock := filepath.Join(filepath.Dir(sock), "ssh.sock")
	srv := NewServer(sock)
	srv.EnableSSH(SSHConfig{SocketPath: sshSock})
	returned := make(chan error, 1)
	go func() { returned <- srv.Start() }()
	client := NewClient(sock)
	for i := 0; i < 20 && !client.Ping(); i++ {
		time.Sleep(50 * time.Millisecond)
	}

	go srv.Stop()
	select {
	case <-returned:
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after Stop")
	}
	// The agent process exits as soon as Start returns.
	for _, p := range []string{sock, sshSock} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s still on disk when Start returned", filepath.Base(p))
		}
	}
}
