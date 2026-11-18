package sshagent

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rogerio/passman/internal/vault"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func ed25519PEM(t *testing.T) (string, ssh.PublicKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(block)), sshPub
}

func rsaPEM(t *testing.T) (string, ssh.PublicKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	sshPub, err := ssh.NewPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(block)), sshPub
}

func encryptedPEM(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKeyWithPassphrase(priv, "", []byte("locked"))
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(block))
}

func allow(req SignRequest) bool { return true }
func deny(req SignRequest) bool  { return false }

func TestKeyring_LoadParsesOnlyPrivateKeys(t *testing.T) {
	edKey, edPub := ed25519PEM(t)
	rsaKey, rsaPub := rsaPEM(t)
	kr := NewKeyring(allow)

	n := kr.Load("v1", []vault.Entry{
		{Name: "ssh/rsa", Value: rsaKey},
		{Name: "github-token", Value: "ghp_notakey"},
		{Name: "ssh/ed25519", Value: edKey},
		{Name: "ssh/locked", Value: encryptedPEM(t)},
	}, LoadOptions{TTL: time.Hour, Confirm: true})

	if n != 2 {
		t.Fatalf("loaded %d keys, want 2", n)
	}
	keys, err := kr.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 {
		t.Fatalf("listed %d keys, want 2", len(keys))
	}
	if keys[0].Comment != "ssh/ed25519" || keys[1].Comment != "ssh/rsa" {
		t.Errorf("unexpected order/comments: %q %q", keys[0].Comment, keys[1].Comment)
	}
	if !bytes.Equal(keys[0].Marshal(), edPub.Marshal()) || !bytes.Equal(keys[1].Marshal(), rsaPub.Marshal()) {
		t.Error("public keys do not match the loaded private keys")
	}
}

func TestKeyring_SignAsksForConfirmation(t *testing.T) {
	edKey, edPub := ed25519PEM(t)
	var seen SignRequest
	kr := NewKeyring(func(req SignRequest) bool {
		seen = req
		return true
	})
	kr.Load("v1", []vault.Entry{{Name: "ssh/id", Value: edKey}}, LoadOptions{TTL: time.Hour, Confirm: true})

	data := []byte("payload")
	sig, err := kr.Sign(edPub, data)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if err := edPub.Verify(data, sig); err != nil {
		t.Errorf("signature does not verify: %v", err)
	}
	if seen.KeyName != "ssh/id" || seen.Fingerprint != ssh.FingerprintSHA256(edPub) {
		t.Errorf("confirmer got %+v", seen)
	}
}

func TestKeyring_SignDenied(t *testing.T) {
	edKey, edPub := ed25519PEM(t)
	kr := NewKeyring(deny)
	kr.Load("v1", []vault.Entry{{Name: "ssh/id", Value: edKey}}, LoadOptions{TTL: time.Hour, Confirm: true})

	sig, err := kr.Sign(edPub, []byte("payload"))
	if !errors.Is(err, ErrDenied) {
		t.Errorf("expected ErrDenied, got %v", err)
	}
	if sig != nil {
		t.Error("expected no signature when denied")
	}
}

func TestKeyring_SignWithoutConfirmation(t *testing.T) {
	edKey, edPub := ed25519PEM(t)
	called := false
	kr := NewKeyring(func(SignRequest) bool { called = true; return false })
	kr.Load("v1", []vault.Entry{{Name: "ssh/id", Value: edKey}}, LoadOptions{TTL: time.Hour, Confirm: false})

	if _, err := kr.Sign(edPub, []byte("payload")); err != nil {
		t.Fatalf("sign: %v", err)
	}
	if called {
		t.Error("confirmer must not be called when confirm is off")
	}
}

func TestKeyring_SignWithFlagsRSA(t *testing.T) {
	rsaKey, rsaPub := rsaPEM(t)
	kr := NewKeyring(allow)
	kr.Load("v1", []vault.Entry{{Name: "ssh/rsa", Value: rsaKey}}, LoadOptions{TTL: time.Hour})

	data := []byte("payload")
	sig, err := kr.SignWithFlags(rsaPub, data, agent.SignatureFlagRsaSha512)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if sig.Format != ssh.KeyAlgoRSASHA512 {
		t.Errorf("format %q, want %q", sig.Format, ssh.KeyAlgoRSASHA512)
	}
	if err := rsaPub.Verify(data, sig); err != nil {
		t.Errorf("signature does not verify: %v", err)
	}
}

func TestKeyring_SignUnknownKey(t *testing.T) {
	edKey, _ := ed25519PEM(t)
	_, otherPub := ed25519PEM(t)
	kr := NewKeyring(allow)
	kr.Load("v1", []vault.Entry{{Name: "ssh/id", Value: edKey}}, LoadOptions{TTL: time.Hour})

	if _, err := kr.Sign(otherPub, []byte("x")); err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestKeyring_LoadReplacesSource(t *testing.T) {
	first, _ := ed25519PEM(t)
	second, secondPub := ed25519PEM(t)
	kr := NewKeyring(allow)
	kr.Load("v1", []vault.Entry{{Name: "old", Value: first}}, LoadOptions{TTL: time.Hour})
	kr.Load("v1", []vault.Entry{{Name: "new", Value: second}}, LoadOptions{TTL: time.Hour})

	keys, _ := kr.List()
	if len(keys) != 1 || !bytes.Equal(keys[0].Marshal(), secondPub.Marshal()) {
		t.Errorf("expected only the second key, got %d keys", len(keys))
	}
}

func TestKeyring_UnloadAndExpire(t *testing.T) {
	k1, _ := ed25519PEM(t)
	k2, _ := ed25519PEM(t)
	kr := NewKeyring(allow)
	now := time.Now()
	kr.Load("short", []vault.Entry{{Name: "a", Value: k1}}, LoadOptions{TTL: time.Minute})
	kr.Load("long", []vault.Entry{{Name: "b", Value: k2}}, LoadOptions{TTL: time.Hour})

	kr.Expire(now.Add(2 * time.Minute))
	keys, _ := kr.List()
	if len(keys) != 1 || keys[0].Comment != "b" {
		t.Fatalf("expected only 'b' after expiry, got %d keys", len(keys))
	}

	kr.Unload("long")
	if !kr.Empty() {
		t.Error("expected empty keyring after unload")
	}
}

func TestKeyring_ClearRemovesEverything(t *testing.T) {
	k1, _ := ed25519PEM(t)
	kr := NewKeyring(allow)
	kr.Load("v1", []vault.Entry{{Name: "a", Value: k1}}, LoadOptions{TTL: time.Hour})

	kr.Clear()
	keys, _ := kr.List()
	if len(keys) != 0 || !kr.Empty() {
		t.Error("expected no keys after clear")
	}
}

func TestKeyring_UnsupportedOperations(t *testing.T) {
	kr := NewKeyring(allow)

	if err := kr.Add(agent.AddedKey{}); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Add: %v", err)
	}
	if err := kr.Remove(nil); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Remove: %v", err)
	}
	if err := kr.RemoveAll(); !errors.Is(err, ErrUnsupported) {
		t.Errorf("RemoveAll: %v", err)
	}
	if err := kr.Lock(nil); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Lock: %v", err)
	}
	if err := kr.Unlock(nil); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Unlock: %v", err)
	}
	if _, err := kr.Extension("x", nil); !errors.Is(err, agent.ErrExtensionUnsupported) {
		t.Errorf("Extension: %v", err)
	}
}

func TestKeyring_OnEmptyHookLoadsKeys(t *testing.T) {
	k1, _ := ed25519PEM(t)
	kr := NewKeyring(allow)
	calls := 0
	kr.OnEmpty = func() {
		calls++
		kr.Load("v1", []vault.Entry{{Name: "a", Value: k1}}, LoadOptions{TTL: time.Hour})
	}

	keys, err := kr.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || calls != 1 {
		t.Errorf("hook not used: %d keys, %d calls", len(keys), calls)
	}

	kr.List()
	if calls != 1 {
		t.Error("hook must not run while keys are loaded")
	}
}

func TestSignRequest_Message(t *testing.T) {
	msg := SignRequest{KeyName: "ssh/id", Fingerprint: "SHA256:abc", PeerPID: 42, PeerName: "/usr/bin/ssh"}.Message()
	for _, want := range []string{"ssh/id", "SHA256:abc", "42", "/usr/bin/ssh"} {
		if !bytes.Contains([]byte(msg), []byte(want)) {
			t.Errorf("message %q lacks %q", msg, want)
		}
	}

	anon := SignRequest{KeyName: "k", Fingerprint: "f"}.Message()
	if bytes.Contains([]byte(anon), []byte("pid")) {
		t.Errorf("message %q should not mention a pid when unknown", anon)
	}
}

func TestServeConn_AgentProtocol(t *testing.T) {
	edKey, edPub := ed25519PEM(t)
	var seen SignRequest
	kr := NewKeyring(func(req SignRequest) bool {
		seen = req
		return true
	})
	kr.Load("v1", []vault.Entry{{Name: "ssh/id", Value: edKey}}, LoadOptions{TTL: time.Hour, Confirm: true})

	dir, err := os.MkdirTemp("/tmp", "pm-ssh-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	sock := filepath.Join(dir, "s.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		kr.ServeConn(conn)
	}()

	conn, err := net.Dial("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := agent.NewClient(conn)

	keys, err := client.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 1 || keys[0].Comment != "ssh/id" {
		t.Fatalf("unexpected keys: %+v", keys)
	}

	data := []byte("challenge")
	sig, err := client.Sign(edPub, data)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if err := edPub.Verify(data, sig); err != nil {
		t.Errorf("signature does not verify: %v", err)
	}
	if seen.PeerPID != os.Getpid() {
		t.Errorf("peer pid %d, want %d", seen.PeerPID, os.Getpid())
	}
	if seen.PeerName == "" {
		t.Error("expected the peer process name to be resolved")
	}

	if err := client.RemoveAll(); err == nil {
		t.Error("expected RemoveAll to be rejected over the protocol")
	}
}

func TestKeyring_ConfirmGraceSkipsRepeatPromptsForSamePeer(t *testing.T) {
	edKey, edPub := ed25519PEM(t)
	prompts := 0
	kr := NewKeyring(func(SignRequest) bool { prompts++; return true })
	kr.Load("v1", []vault.Entry{{Name: "ssh/id", Value: edKey}}, LoadOptions{TTL: time.Hour, Confirm: true, ConfirmGrace: time.Minute})
	git := SignRequest{PeerPID: 10, PeerName: "/usr/bin/ssh"}
	data := []byte("x")

	if _, err := kr.sign(edPub, data, 0, git); err != nil {
		t.Fatal(err)
	}
	git.PeerPID = 11 // a new ssh process started by the same program
	if _, err := kr.sign(edPub, data, 0, git); err != nil {
		t.Fatal(err)
	}
	if prompts != 1 {
		t.Fatalf("expected one prompt within the grace period, got %d", prompts)
	}

	other := SignRequest{PeerPID: 12, PeerName: "/tmp/other"}
	if _, err := kr.sign(edPub, data, 0, other); err != nil {
		t.Fatal(err)
	}
	if prompts != 2 {
		t.Errorf("a different program must be asked, got %d prompts", prompts)
	}
}

func TestKeyring_ConfirmGraceExpires(t *testing.T) {
	edKey, edPub := ed25519PEM(t)
	prompts := 0
	kr := NewKeyring(func(SignRequest) bool { prompts++; return true })
	kr.Load("v1", []vault.Entry{{Name: "ssh/id", Value: edKey}}, LoadOptions{TTL: time.Hour, Confirm: true, ConfirmGrace: time.Minute})
	req := SignRequest{PeerPID: 10, PeerName: "/usr/bin/ssh"}

	kr.sign(edPub, []byte("x"), 0, req)
	kr.mu.Lock()
	for k := range kr.approvals {
		kr.approvals[k] = time.Now().Add(-2 * time.Minute)
	}
	kr.mu.Unlock()
	kr.sign(edPub, []byte("x"), 0, req)

	if prompts != 2 {
		t.Errorf("expected a new prompt after the grace expired, got %d", prompts)
	}
}

func TestKeyring_ConfirmGraceNotAppliedWithoutPeerOrWhenZero(t *testing.T) {
	edKey, edPub := ed25519PEM(t)
	prompts := 0
	kr := NewKeyring(func(SignRequest) bool { prompts++; return true })
	kr.Load("v1", []vault.Entry{{Name: "ssh/id", Value: edKey}}, LoadOptions{TTL: time.Hour, Confirm: true, ConfirmGrace: time.Minute})

	anon := SignRequest{}
	kr.sign(edPub, []byte("x"), 0, anon)
	kr.sign(edPub, []byte("x"), 0, anon)
	if prompts != 2 {
		t.Errorf("unknown peers must always be asked, got %d prompts", prompts)
	}

	kr.Load("v1", []vault.Entry{{Name: "ssh/id", Value: edKey}}, LoadOptions{TTL: time.Hour, Confirm: true})
	prompts = 0
	req := SignRequest{PeerPID: 10, PeerName: "/usr/bin/ssh"}
	kr.sign(edPub, []byte("x"), 0, req)
	kr.sign(edPub, []byte("x"), 0, req)
	if prompts != 2 {
		t.Errorf("grace 0 must ask every time, got %d prompts", prompts)
	}
}

func TestKeyring_DenyRevokesGraceAndLockClearsIt(t *testing.T) {
	edKey, edPub := ed25519PEM(t)
	allowNext := true
	prompts := 0
	kr := NewKeyring(func(SignRequest) bool { prompts++; return allowNext })
	kr.Load("v1", []vault.Entry{{Name: "ssh/id", Value: edKey}}, LoadOptions{TTL: time.Hour, Confirm: true, ConfirmGrace: time.Minute})
	req := SignRequest{PeerPID: 10, PeerName: "/usr/bin/ssh"}

	kr.sign(edPub, []byte("x"), 0, req) // allowed → grace
	kr.mu.Lock()
	for k := range kr.approvals {
		kr.approvals[k] = time.Now().Add(-2 * time.Minute)
	}
	kr.mu.Unlock()
	allowNext = false
	if _, err := kr.sign(edPub, []byte("x"), 0, req); !errors.Is(err, ErrDenied) {
		t.Fatalf("expected denial, got %v", err)
	}
	allowNext = true
	kr.sign(edPub, []byte("x"), 0, req)
	if prompts != 3 {
		t.Errorf("a denial must not leave an approval behind, got %d prompts", prompts)
	}

	kr.Clear()
	kr.Load("v1", []vault.Entry{{Name: "ssh/id", Value: edKey}}, LoadOptions{TTL: time.Hour, Confirm: true, ConfirmGrace: time.Minute})
	kr.sign(edPub, []byte("x"), 0, req)
	if prompts != 4 {
		t.Errorf("Clear must drop approvals, got %d prompts", prompts)
	}
}
