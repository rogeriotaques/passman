package cmd

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rogerio/passman/internal/agent"
	"github.com/rogerio/passman/internal/prompt"
	"golang.org/x/crypto/ssh"
	sshagentproto "golang.org/x/crypto/ssh/agent"
)

func writeKeyFile(t *testing.T, name string, block *pem.Block) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func ed25519KeyFile(t *testing.T, name, passphrase string) (string, ssh.PublicKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	var block *pem.Block
	if passphrase == "" {
		block, err = ssh.MarshalPrivateKey(priv, "")
	} else {
		block, err = ssh.MarshalPrivateKeyWithPassphrase(priv, "", []byte(passphrase))
	}
	if err != nil {
		t.Fatal(err)
	}
	sshPub, _ := ssh.NewPublicKey(pub)
	return writeKeyFile(t, name, block), sshPub
}

func rsaKeyFile(t *testing.T, name string) string {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	return writeKeyFile(t, name, block)
}

func initVault(t *testing.T) {
	t.Helper()
	setupTestApp(t, "masterpass\nmasterpass\n")
	if err := runCmd("vault", "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
}

// --- ssh add ---

func TestSSHAdd_ImportsUnencryptedKey(t *testing.T) {
	initVault(t)
	keyPath, pub := ed25519KeyFile(t, "id_ed25519", "")
	out, _, _ := setupOutput(t, "masterpass\n")

	if err := runCmd("ssh", "add", keyPath); err != nil {
		t.Fatalf("ssh add: %v", err)
	}
	if !strings.Contains(out.String(), "ssh/id_ed25519") || !strings.Contains(out.String(), ssh.FingerprintSHA256(pub)) {
		t.Errorf("expected name and fingerprint in output: %s", out.String())
	}
	authorized := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub)))
	if !strings.Contains(out.String(), authorized) {
		t.Errorf("expected the public key line in output: %s", out.String())
	}

	setInput("masterpass\n")
	out.Reset()
	resetFlags()
	if err := runCmd("secret", "get", "--print", "ssh/id_ed25519"); err != nil {
		t.Fatalf("get: %v", err)
	}
	signer, err := ssh.ParsePrivateKey([]byte(out.String()))
	if err != nil {
		t.Fatalf("stored value is not a usable private key: %v", err)
	}
	if string(signer.PublicKey().Marshal()) != string(pub.Marshal()) {
		t.Error("stored key does not match the imported key")
	}
}

func TestSSHAdd_DecryptsPassphraseProtectedKey(t *testing.T) {
	initVault(t)
	keyPath, pub := ed25519KeyFile(t, "id_locked", "s3cret")
	out, _, _ := setupOutput(t, "masterpass\ns3cret\n")

	if err := runCmd("ssh", "add", keyPath); err != nil {
		t.Fatalf("ssh add: %v", err)
	}

	setInput("masterpass\n")
	out.Reset()
	resetFlags()
	_ = runCmd("secret", "get", "--print", "ssh/id_locked")
	signer, err := ssh.ParsePrivateKey([]byte(out.String()))
	if err != nil {
		t.Fatalf("stored key must be unencrypted: %v", err)
	}
	if string(signer.PublicKey().Marshal()) != string(pub.Marshal()) {
		t.Error("stored key does not match the imported key")
	}
}

func TestSSHAdd_WrongPassphrase(t *testing.T) {
	initVault(t)
	keyPath, _ := ed25519KeyFile(t, "id_locked", "s3cret")
	setupOutput(t, "masterpass\nnope\n")

	if err := runCmd("ssh", "add", keyPath); err == nil {
		t.Error("expected error for wrong passphrase")
	}
}

func TestSSHAdd_CustomName(t *testing.T) {
	initVault(t)
	keyPath, _ := ed25519KeyFile(t, "id_ed25519", "")
	out, _, _ := setupOutput(t, "masterpass\n")

	if err := runCmd("ssh", "add", keyPath, "--name", "ssh/github"); err != nil {
		t.Fatalf("ssh add: %v", err)
	}
	if !strings.Contains(out.String(), "ssh/github") {
		t.Errorf("expected custom name in output: %s", out.String())
	}
}

func TestSSHAdd_RSAKey(t *testing.T) {
	initVault(t)
	keyPath := rsaKeyFile(t, "aws.pem")
	out, _, _ := setupOutput(t, "masterpass\n")

	if err := runCmd("ssh", "add", keyPath); err != nil {
		t.Fatalf("ssh add: %v", err)
	}
	if !strings.Contains(out.String(), "ssh/aws.pem") || !strings.Contains(out.String(), "ssh-rsa") {
		t.Errorf("unexpected output: %s", out.String())
	}
}

func TestSSHAdd_RejectsNonKeyFile(t *testing.T) {
	initVault(t)
	path := filepath.Join(t.TempDir(), "notes.txt")
	_ = os.WriteFile(path, []byte("hello"), 0600)
	setupOutput(t, "masterpass\n")

	if err := runCmd("ssh", "add", path); err == nil {
		t.Error("expected error for a file that is not a private key")
	}
}

func TestSSHAdd_RejectsOversizedFile(t *testing.T) {
	initVault(t)
	path := filepath.Join(t.TempDir(), "huge")
	_ = os.WriteFile(path, make([]byte, maxKeyFileSize+1), 0600)
	setupOutput(t, "masterpass\n")

	if err := runCmd("ssh", "add", path); err == nil {
		t.Error("expected error for oversized file")
	}
}

func TestSSHAdd_DuplicateName(t *testing.T) {
	initVault(t)
	keyPath, _ := ed25519KeyFile(t, "id_ed25519", "")
	setupOutput(t, "masterpass\n")
	_ = runCmd("ssh", "add", keyPath)

	setInput("masterpass\n")
	resetFlags()
	if err := runCmd("ssh", "add", keyPath); err == nil {
		t.Error("expected error when the entry already exists")
	}
}

// --- ssh list ---

func TestSSHList(t *testing.T) {
	initVault(t)
	edPath, pub := ed25519KeyFile(t, "id_ed25519", "")
	setupOutput(t, "masterpass\n")
	_ = runCmd("ssh", "add", edPath)
	setInput("tok\n")
	resetFlags()
	_ = runCmd("secret", "add", "github-token")

	out, _, _ := setupOutput(t, "masterpass\n")
	if err := runCmd("ssh", "list"); err != nil {
		t.Fatalf("ssh list: %v", err)
	}
	if !strings.Contains(out.String(), "ssh/id_ed25519") || !strings.Contains(out.String(), ssh.FingerprintSHA256(pub)) {
		t.Errorf("expected key name and fingerprint: %s", out.String())
	}
	if strings.Contains(out.String(), "github-token") {
		t.Errorf("non-key entries must not be listed: %s", out.String())
	}
}

func TestSSHList_Empty(t *testing.T) {
	initVault(t)
	out, _, _ := setupOutput(t, "masterpass\n")

	if err := runCmd("ssh", "list"); err != nil {
		t.Fatalf("ssh list: %v", err)
	}
	if !strings.Contains(out.String(), "No SSH keys") {
		t.Errorf("unexpected output: %s", out.String())
	}
}

// --- ssh env ---

func TestSSHEnv_PrintsExportsAndWritesHelper(t *testing.T) {
	out, _, _ := setupTestApp(t, "")
	dir := t.TempDir()
	app.SSHSocketPath = filepath.Join(dir, "ssh-agent.sock")
	app.AskpassPath = filepath.Join(dir, "askpass")

	if err := runCmd("ssh", "env"); err != nil {
		t.Fatalf("ssh env: %v", err)
	}
	for _, want := range []string{
		"export SSH_AUTH_SOCK='" + app.SSHSocketPath + "'",
		"export SSH_ASKPASS='" + app.AskpassPath + "'",
		"export SSH_ASKPASS_REQUIRE='force'",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, out.String())
		}
	}

	info, err := os.Stat(app.AskpassPath)
	if err != nil {
		t.Fatalf("helper not written: %v", err)
	}
	if info.Mode().Perm() != 0700 {
		t.Errorf("helper mode %o, want 0700", info.Mode().Perm())
	}
	exe, _ := os.Executable()
	content, _ := os.ReadFile(app.AskpassPath)
	if !strings.HasPrefix(string(content), "#!/bin/sh\n") || !strings.Contains(string(content), exe) || !strings.Contains(string(content), "ssh askpass") {
		t.Errorf("unexpected helper content:\n%s", content)
	}
}

func TestSSHEnv_RewritesStaleHelper(t *testing.T) {
	setupTestApp(t, "")
	dir := t.TempDir()
	app.SSHSocketPath = filepath.Join(dir, "ssh-agent.sock")
	app.AskpassPath = filepath.Join(dir, "askpass")
	_ = os.WriteFile(app.AskpassPath, []byte("#!/bin/sh\nexec /old/passman ssh askpass \"$@\"\n"), 0700)

	if err := runCmd("ssh", "env"); err != nil {
		t.Fatalf("ssh env: %v", err)
	}
	content, _ := os.ReadFile(app.AskpassPath)
	if strings.Contains(string(content), "/old/passman") {
		t.Error("stale helper should be rewritten")
	}
}

// --- ssh askpass ---

func TestSSHAskpass_ConfirmAllow(t *testing.T) {
	out, _, _ := setupTestApp(t, "")
	mock := &prompt.Mock{ConfirmAnswer: true}
	app.Prompter = mock
	t.Setenv("SSH_ASKPASS_PROMPT", "confirm")

	if err := runCmd("ssh", "askpass", "Allow use of key?"); err != nil {
		t.Fatalf("expected success when allowed: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("confirm must not print anything, got %q", out.String())
	}
	if len(mock.Messages) != 1 || mock.Messages[0] != "Allow use of key?" {
		t.Errorf("unexpected prompts: %v", mock.Messages)
	}
}

func TestSSHAskpass_ConfirmDeny(t *testing.T) {
	setupTestApp(t, "")
	app.Prompter = &prompt.Mock{ConfirmAnswer: false}
	t.Setenv("SSH_ASKPASS_PROMPT", "confirm")

	if err := runCmd("ssh", "askpass", "Allow use of key?"); err == nil {
		t.Error("expected non-zero exit when denied")
	}
}

func TestSSHAskpass_VisiblePrompt(t *testing.T) {
	out, _, _ := setupTestApp(t, "")
	app.Prompter = &prompt.Mock{TextAnswer: "yes"}
	t.Setenv("SSH_ASKPASS_PROMPT", "none")

	if err := runCmd("ssh", "askpass", "Continue connecting (yes/no)?"); err != nil {
		t.Fatalf("askpass: %v", err)
	}
	if out.String() != "yes\n" {
		t.Errorf("got %q, want %q", out.String(), "yes\n")
	}
}

func TestSSHAskpass_PassphraseFromVault(t *testing.T) {
	initVault(t)
	setupOutput(t, "masterpass\nkey-pass\n")
	_ = runCmd("secret", "add", "ssh/passphrase/id_ed25519")

	out, _, _ := setupOutput(t, "")
	mock := &prompt.Mock{SecretAnswer: "masterpass"}
	app.Prompter = mock
	t.Setenv("SSH_ASKPASS_PROMPT", "")

	if err := runCmd("ssh", "askpass", "Enter passphrase for key '/home/u/.ssh/id_ed25519': "); err != nil {
		t.Fatalf("askpass: %v", err)
	}
	if out.String() != "key-pass\n" {
		t.Errorf("got %q, want the vault passphrase", out.String())
	}
	if len(mock.Messages) != 1 {
		t.Errorf("expected exactly one prompt (master password), got %v", mock.Messages)
	}
}

func TestSSHAskpass_PassphraseFromVault_SSHAddWording(t *testing.T) {
	initVault(t)
	setupOutput(t, "masterpass\nkey-pass\n")
	_ = runCmd("secret", "add", "ssh/passphrase/aws.pem")

	out, _, _ := setupOutput(t, "")
	app.Prompter = &prompt.Mock{SecretAnswer: "masterpass"}
	t.Setenv("SSH_ASKPASS_PROMPT", "")

	if err := runCmd("ssh", "askpass", "Enter passphrase for /home/u/.ssh/aws.pem: "); err != nil {
		t.Fatalf("askpass: %v", err)
	}
	if out.String() != "key-pass\n" {
		t.Errorf("got %q, want the vault passphrase", out.String())
	}
}

func TestSSHAskpass_UnknownKeyFallsBackToPrompt(t *testing.T) {
	initVault(t)
	out, _, _ := setupOutput(t, "")
	mock := &prompt.Mock{SecretAnswer: "typed"}
	app.Prompter = mock
	t.Setenv("SSH_ASKPASS_PROMPT", "")

	if err := runCmd("ssh", "askpass", "Enter passphrase for key '/home/u/.ssh/other': "); err != nil {
		t.Fatalf("askpass: %v", err)
	}
	if out.String() != "typed\n" {
		t.Errorf("got %q, want the prompted passphrase", out.String())
	}
	// master password, then the passphrase itself
	if len(mock.Messages) != 2 {
		t.Errorf("expected 2 prompts, got %v", mock.Messages)
	}
}

func TestSSHAskpass_BadPassphraseRetrySkipsVault(t *testing.T) {
	initVault(t)
	setupOutput(t, "masterpass\nstale\n")
	_ = runCmd("secret", "add", "ssh/passphrase/id_ed25519")

	out, _, _ := setupOutput(t, "")
	mock := &prompt.Mock{SecretAnswer: "fresh"}
	app.Prompter = mock
	t.Setenv("SSH_ASKPASS_PROMPT", "")

	if err := runCmd("ssh", "askpass", "Bad passphrase, try again for /home/u/.ssh/id_ed25519: "); err != nil {
		t.Fatalf("askpass: %v", err)
	}
	if out.String() != "fresh\n" {
		t.Errorf("a retry must ask the user instead of replaying the vault value, got %q", out.String())
	}
	if len(mock.Messages) != 1 {
		t.Errorf("expected only the passphrase prompt, got %v", mock.Messages)
	}
}

func TestSSHAskpass_PromptWithoutKeyPath(t *testing.T) {
	out, _, _ := setupTestApp(t, "")
	mock := &prompt.Mock{SecretAnswer: "pw"}
	app.Prompter = mock
	t.Setenv("SSH_ASKPASS_PROMPT", "")

	if err := runCmd("ssh", "askpass", "user@host's password: "); err != nil {
		t.Fatalf("askpass: %v", err)
	}
	if out.String() != "pw\n" || len(mock.Messages) != 1 {
		t.Errorf("expected a single secret prompt, got %q / %v", out.String(), mock.Messages)
	}
}

func TestSSHAskpass_CancelledPrompt(t *testing.T) {
	setupTestApp(t, "")
	app.Prompter = &prompt.Mock{Err: prompt.ErrUnavailable}
	t.Setenv("SSH_ASKPASS_PROMPT", "")

	if err := runCmd("ssh", "askpass", "user@host's password: "); err == nil {
		t.Error("expected failure when no answer can be obtained")
	}
}

// --- agent wiring ---

func TestCacheInAgent_PassesSSHSettingsToSpawner(t *testing.T) {
	setupTestApp(t, "")
	app.SocketPath = testSocketPath(t)
	app.SSHSocketPath = app.SocketPath + ".ssh"
	app.SSHUnlockVault = "/vaults/default/vault.enc"
	app.AutoStartAgent = true

	var gotSock, gotSSH, gotVault string
	stubEnsureAgent(t, func(sock, sshSock, sshVault string) error {
		gotSock, gotSSH, gotVault = sock, sshSock, sshVault
		return nil
	})

	cacheInAgent([]byte("secret"))

	if gotSock != app.SocketPath || gotSSH != app.SSHSocketPath || gotVault != app.SSHUnlockVault {
		t.Errorf("spawner got %q %q %q", gotSock, gotSSH, gotVault)
	}
}

func TestSSHAdd_ReloadsKeysInRunningAgent(t *testing.T) {
	initVault(t)
	sock := testSocketPath(t)
	sshSock := sock + ".ssh"
	srv := agent.NewServer(sock)
	srv.EnableSSH(agent.SSHConfig{SocketPath: sshSock})
	go srv.Start()
	t.Cleanup(func() { srv.Stop() })
	client := agent.NewClient(sock)
	for i := 0; i < 20 && !client.Ping(); i++ {
		time.Sleep(50 * time.Millisecond)
	}

	keyPath, pub := ed25519KeyFile(t, "id_ed25519", "")
	setupOutput(t, "masterpass\n")
	app.SocketPath = sock
	if err := runCmd("ssh", "add", keyPath); err != nil {
		t.Fatalf("ssh add: %v", err)
	}

	conn, err := net.Dial("unix", sshSock)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	keys, err := sshagentproto.NewClient(conn).List()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || string(keys[0].Marshal()) != string(pub.Marshal()) {
		t.Fatalf("expected the new key to be served right after add, got %d keys", len(keys))
	}
}

// setupOutput resets I/O for a follow-up command against the current vault.
func setupOutput(t *testing.T, input string) (*strings.Builder, *strings.Builder, *prompt.Mock) {
	t.Helper()
	out := &strings.Builder{}
	errOut := &strings.Builder{}
	mock := &prompt.Mock{}
	app.Out = out
	app.ErrOut = errOut
	app.Prompter = mock
	setInput(input)
	resetFlags()
	return out, errOut, mock
}

func TestAttachTTY_IgnoresNonTerminal(t *testing.T) {
	setupTestApp(t, "")
	in := app.In

	attachTTY()

	if app.In != in {
		t.Error("stdin must stay untouched when the TTY path is not a terminal")
	}
}

func TestKeyPathPattern_OpenSSHWordings(t *testing.T) {
	cases := map[string]string{
		"Enter passphrase for key '/home/u/.ssh/id_ed25519':":                "/home/u/.ssh/id_ed25519",
		"Enter passphrase for /home/u/.ssh/aws.pem:":                         "/home/u/.ssh/aws.pem",
		"Enter passphrase for /home/u/.ssh/aws.pem (will confirm each use):": "/home/u/.ssh/aws.pem",
		`Enter passphrase for "e2e_rsa":`:                                    "e2e_rsa",
		"Enter passphrase for key '/Users/r/.ssh/with space/id': ":           "/Users/r/.ssh/with space/id",
	}
	for prompt, want := range cases {
		m := keyPathPattern.FindStringSubmatch(strings.TrimSpace(prompt))
		if m == nil || m[1] != want {
			t.Errorf("%q: got %v, want %q", prompt, m, want)
		}
	}
	for _, prompt := range []string{
		"Bad passphrase, try again for /home/u/.ssh/id_ed25519:",
		"Enter passphrase (empty for no passphrase):",
		"user@host's password:",
	} {
		if keyPathPattern.MatchString(prompt) {
			t.Errorf("%q must not be treated as a key prompt", prompt)
		}
	}
}

func TestSSHEnv_StartsAgent(t *testing.T) {
	setupTestApp(t, "")
	dir := t.TempDir()
	app.SocketPath = filepath.Join(dir, "agent.sock")
	app.SSHSocketPath = filepath.Join(dir, "ssh-agent.sock")
	app.SSHUnlockVault = "/vaults/default/vault.enc"
	app.AutoStartAgent = true

	var got []string
	stubEnsureAgent(t, func(sock, sshSock, sshVault string) error {
		got = []string{sock, sshSock, sshVault}
		return nil
	})

	if err := runCmd("ssh", "env"); err != nil {
		t.Fatalf("ssh env: %v", err)
	}
	want := []string{app.SocketPath, app.SSHSocketPath, app.SSHUnlockVault}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("ssh env must make sure the agent runs: got %v, want %v", got, want)
	}
}

func TestSSHEnv_AgentStartFailureIsOnlyAWarning(t *testing.T) {
	out, errOut, _ := setupTestApp(t, "")
	app.SocketPath = filepath.Join(t.TempDir(), "agent.sock")
	app.AutoStartAgent = true
	stubEnsureAgent(t, func(string, string, string) error { return errors.New("boom") })

	if err := runCmd("ssh", "env"); err != nil {
		t.Fatalf("a failed agent start must not break shell startup: %v", err)
	}
	if !strings.Contains(out.String(), "export SSH_AUTH_SOCK=") {
		t.Errorf("exports missing: %s", out.String())
	}
	if !strings.Contains(errOut.String(), "boom") {
		t.Errorf("expected a warning on stderr, got %q", errOut.String())
	}
}
