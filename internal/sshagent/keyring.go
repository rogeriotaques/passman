// Package sshagent serves the SSH agent protocol from private keys stored in a
// vault. Keys never leave the agent process: clients can only ask for
// signatures, and each request may require user confirmation.
package sshagent

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rogerio/passman/internal/vault"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

var (
	ErrDenied      = errors.New("signature request denied")
	ErrUnsupported = errors.New("keys are managed by passman: use `passman ssh add` and `passman secret remove`")
	errKeyNotFound = errors.New("key not found")
)

// SignRequest describes a pending signature so the user can decide on it.
type SignRequest struct {
	KeyName     string
	Fingerprint string
	// PeerPID and PeerName identify the connecting process when the platform
	// exposes them; PeerPID is 0 otherwise.
	PeerPID  int
	PeerName string
}

// Message renders the request for a confirmation dialog.
func (r SignRequest) Message() string {
	who := "A process"
	if r.PeerPID > 0 {
		who = fmt.Sprintf("%s (pid %d)", r.PeerName, r.PeerPID)
		if r.PeerName == "" {
			who = fmt.Sprintf("Process %d", r.PeerPID)
		}
	}
	return fmt.Sprintf("%s wants to sign with your SSH key.\n\nKey: %s\n%s", who, r.KeyName, r.Fingerprint)
}

// Confirmer decides whether a signature request may proceed. It runs on the
// agent's request path, so it blocks the client until the user answers.
type Confirmer func(req SignRequest) bool

// LoadOptions control how long a vault's keys stay loaded and whether each
// signature needs confirmation. ConfirmGrace lets a program that was just
// allowed to use a key sign again without a new prompt for that long; zero
// asks every time.
type LoadOptions struct {
	TTL          time.Duration
	Confirm      bool
	ConfirmGrace time.Duration
}

type loadedKey struct {
	name    string
	signer  ssh.Signer
	confirm bool
	grace   time.Duration
}

// approvalKey identifies "this program may use this key": the peer's
// executable path rather than its pid, since every ssh connection is a new
// process.
type approvalKey struct {
	fingerprint string
	peer        string
}

type source struct {
	keys      []loadedKey
	expiresAt time.Time
}

// Keyring holds the keys of one or more vaults, grouped by source so a vault
// can be unloaded or expire independently.
type Keyring struct {
	mu        sync.Mutex
	sources   map[string]*source
	approvals map[approvalKey]time.Time
	confirm   Confirmer

	// OnEmpty, when set, runs before List returns an empty result. The agent
	// uses it to offer unlocking a vault on demand.
	OnEmpty func()
}

func NewKeyring(confirm Confirmer) *Keyring {
	return &Keyring{
		sources:   make(map[string]*source),
		approvals: make(map[approvalKey]time.Time),
		confirm:   confirm,
	}
}

// Load replaces the keys of source with the private keys found in entries and
// returns how many were loaded. Entries that are not parseable, unencrypted
// private keys are ignored.
func (k *Keyring) Load(src string, entries []vault.Entry, opts LoadOptions) int {
	var keys []loadedKey
	for _, e := range entries {
		signer, ok := ParseKey(e.Value)
		if !ok {
			continue
		}
		keys = append(keys, loadedKey{name: e.Name, signer: signer, confirm: opts.Confirm, grace: opts.ConfirmGrace})
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].name < keys[j].name })

	k.mu.Lock()
	defer k.mu.Unlock()
	if len(keys) == 0 {
		delete(k.sources, src)
		return 0
	}
	k.sources[src] = &source{keys: keys, expiresAt: time.Now().Add(opts.TTL)}
	return len(keys)
}

// ParseKey returns a signer for value when it holds an unencrypted private
// key in a format OpenSSH understands.
func ParseKey(value string) (ssh.Signer, bool) {
	if !strings.Contains(value, "PRIVATE KEY-----") {
		return nil, false
	}
	signer, err := ssh.ParsePrivateKey([]byte(value))
	if err != nil {
		return nil, false
	}
	return signer, true
}

// Unload drops the keys of one source.
func (k *Keyring) Unload(src string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.dropSource(src)
}

// Expire drops every source whose TTL passed at the given time.
func (k *Keyring) Expire(now time.Time) {
	k.mu.Lock()
	defer k.mu.Unlock()
	for name, s := range k.sources {
		if now.After(s.expiresAt) {
			k.dropSource(name)
		}
	}
}

// dropSource removes a source and the approvals granted for its keys.
// Callers hold k.mu.
func (k *Keyring) dropSource(src string) {
	s, ok := k.sources[src]
	if !ok {
		return
	}
	for _, lk := range s.keys {
		fp := ssh.FingerprintSHA256(lk.signer.PublicKey())
		for a := range k.approvals {
			if a.fingerprint == fp {
				delete(k.approvals, a)
			}
		}
	}
	delete(k.sources, src)
}

// Clear drops all keys and approvals.
func (k *Keyring) Clear() {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.sources = make(map[string]*source)
	k.approvals = make(map[approvalKey]time.Time)
}

// Empty reports whether no keys are loaded.
func (k *Keyring) Empty() bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	return len(k.sources) == 0
}

// snapshot returns the loaded keys in a stable order.
func (k *Keyring) snapshot() []loadedKey {
	k.mu.Lock()
	defer k.mu.Unlock()
	names := make([]string, 0, len(k.sources))
	for name := range k.sources {
		names = append(names, name)
	}
	sort.Strings(names)

	var keys []loadedKey
	for _, name := range names {
		keys = append(keys, k.sources[name].keys...)
	}
	return keys
}

// ServeConn answers agent protocol requests on conn until it closes.
func (k *Keyring) ServeConn(conn net.Conn) {
	defer conn.Close()
	pid, name := peerProcess(conn)
	agent.ServeAgent(&connAgent{keyring: k, pid: pid, name: name}, conn)
}

// List implements agent.Agent.
func (k *Keyring) List() ([]*agent.Key, error) {
	return k.list()
}

func (k *Keyring) list() ([]*agent.Key, error) {
	keys := k.snapshot()
	if len(keys) == 0 && k.OnEmpty != nil {
		k.OnEmpty()
		keys = k.snapshot()
	}

	out := make([]*agent.Key, 0, len(keys))
	for _, lk := range keys {
		pub := lk.signer.PublicKey()
		out = append(out, &agent.Key{Format: pub.Type(), Blob: pub.Marshal(), Comment: lk.name})
	}
	return out, nil
}

// Sign implements agent.Agent.
func (k *Keyring) Sign(key ssh.PublicKey, data []byte) (*ssh.Signature, error) {
	return k.SignWithFlags(key, data, 0)
}

// SignWithFlags implements agent.ExtendedAgent.
func (k *Keyring) SignWithFlags(key ssh.PublicKey, data []byte, flags agent.SignatureFlags) (*ssh.Signature, error) {
	return k.sign(key, data, flags, SignRequest{})
}

func (k *Keyring) sign(key ssh.PublicKey, data []byte, flags agent.SignatureFlags, req SignRequest) (*ssh.Signature, error) {
	want := key.Marshal()
	for _, lk := range k.snapshot() {
		pub := lk.signer.PublicKey()
		if !bytes.Equal(pub.Marshal(), want) {
			continue
		}
		if lk.confirm {
			req.KeyName = lk.name
			req.Fingerprint = ssh.FingerprintSHA256(pub)
			if !k.approved(req, lk.grace) {
				return nil, ErrDenied
			}
		}
		return signWithFlags(lk.signer, data, flags)
	}
	return nil, errKeyNotFound
}

// approved asks the confirmer, honouring a still-valid approval for the same
// program and key when a grace period is configured. A denial revokes any
// earlier approval.
func (k *Keyring) approved(req SignRequest, grace time.Duration) bool {
	key := approvalKey{fingerprint: req.Fingerprint, peer: req.PeerName}
	useGrace := grace > 0 && req.PeerName != ""

	if useGrace {
		k.mu.Lock()
		granted, ok := k.approvals[key]
		k.mu.Unlock()
		if ok && time.Since(granted) < grace {
			return true
		}
	}

	ok := k.confirm != nil && k.confirm(req)

	k.mu.Lock()
	defer k.mu.Unlock()
	if ok && useGrace {
		k.approvals[key] = time.Now()
	} else {
		delete(k.approvals, key)
	}
	return ok
}

func signWithFlags(signer ssh.Signer, data []byte, flags agent.SignatureFlags) (*ssh.Signature, error) {
	if flags == 0 {
		return signer.Sign(rand.Reader, data)
	}
	algorithmSigner, ok := signer.(ssh.AlgorithmSigner)
	if !ok {
		return nil, fmt.Errorf("key does not support algorithm selection")
	}
	var algorithm string
	switch flags {
	case agent.SignatureFlagRsaSha256:
		algorithm = ssh.KeyAlgoRSASHA256
	case agent.SignatureFlagRsaSha512:
		algorithm = ssh.KeyAlgoRSASHA512
	default:
		return nil, fmt.Errorf("unsupported signature flags: %d", flags)
	}
	return algorithmSigner.SignWithAlgorithm(rand.Reader, data, algorithm)
}

// Signers implements agent.Agent.
func (k *Keyring) Signers() ([]ssh.Signer, error) {
	keys := k.snapshot()
	out := make([]ssh.Signer, 0, len(keys))
	for _, lk := range keys {
		out = append(out, lk.signer)
	}
	return out, nil
}

// The vault is the only way in or out, so the protocol's key management
// operations are rejected.

func (k *Keyring) Add(agent.AddedKey) error   { return ErrUnsupported }
func (k *Keyring) Remove(ssh.PublicKey) error { return ErrUnsupported }
func (k *Keyring) RemoveAll() error           { return ErrUnsupported }
func (k *Keyring) Lock([]byte) error          { return ErrUnsupported }
func (k *Keyring) Unlock([]byte) error        { return ErrUnsupported }

// Extension implements agent.ExtendedAgent.
func (k *Keyring) Extension(string, []byte) ([]byte, error) {
	return nil, agent.ErrExtensionUnsupported
}

// connAgent binds one connection's peer identity to the keyring so that
// confirmation dialogs can name the requesting process.
type connAgent struct {
	keyring *Keyring
	pid     int
	name    string
}

func (c *connAgent) List() ([]*agent.Key, error) { return c.keyring.list() }

func (c *connAgent) Sign(key ssh.PublicKey, data []byte) (*ssh.Signature, error) {
	return c.SignWithFlags(key, data, 0)
}

func (c *connAgent) SignWithFlags(key ssh.PublicKey, data []byte, flags agent.SignatureFlags) (*ssh.Signature, error) {
	return c.keyring.sign(key, data, flags, SignRequest{PeerPID: c.pid, PeerName: c.name})
}

func (c *connAgent) Signers() ([]ssh.Signer, error)               { return c.keyring.Signers() }
func (c *connAgent) Add(key agent.AddedKey) error                 { return c.keyring.Add(key) }
func (c *connAgent) Remove(key ssh.PublicKey) error               { return c.keyring.Remove(key) }
func (c *connAgent) RemoveAll() error                             { return c.keyring.RemoveAll() }
func (c *connAgent) Lock(passphrase []byte) error                 { return c.keyring.Lock(passphrase) }
func (c *connAgent) Unlock(passphrase []byte) error               { return c.keyring.Unlock(passphrase) }
func (c *connAgent) Extension(t string, d []byte) ([]byte, error) { return c.keyring.Extension(t, d) }
