package agent

import (
	"crypto/subtle"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/rogerio/passman/internal/sshagent"
	psync "github.com/rogerio/passman/internal/sync"
	"github.com/rogerio/passman/internal/vault"
)

type cacheEntry struct {
	password  []byte
	expiresAt time.Time
}

// SSHConfig turns the agent into an SSH agent serving the keys of unlocked
// vaults.
type SSHConfig struct {
	// SocketPath is where SSH clients connect (SSH_AUTH_SOCK).
	SocketPath string
	// Confirm approves signature requests; nil denies them all when a vault
	// asks for confirmation.
	Confirm sshagent.Confirmer
	// UnlockVault, when set, is the vault offered for unlocking when an SSH
	// client asks for keys and none are loaded. PromptPassword asks the user
	// for its master password.
	UnlockVault    string
	PromptPassword func(message string) ([]byte, error)
}

// Unlock prompts are rate-limited so a client that keeps asking for keys
// (automation reconnecting through ssh) cannot flood the user with dialogs.
// Each cancelled or failed prompt doubles the wait, up to unlockMaxCooldown; a
// successful unlock resets it.
const (
	unlockCooldown    = 10 * time.Second
	unlockMaxCooldown = 15 * time.Minute
)

type sshState struct {
	cfg      SSHConfig
	listener net.Listener
	// loaded remembers which vault file produced the current keys so that a
	// Store for an unchanged vault does not re-derive the key.
	loaded           map[string]vaultStamp
	lastUnlockPrompt time.Time
	unlockFailures   uint
}

type vaultStamp struct {
	modTime  time.Time
	size     int64
	password []byte
}

type Server struct {
	mu         sync.Mutex
	entries    map[string]*cacheEntry
	socketPath string
	listener   net.Listener
	// done is closed when shutdown begins (accept loops exit); stopped when
	// secrets are zeroed and the sockets removed. Start waits for stopped,
	// because the agent process exits as soon as Start returns.
	done     chan struct{}
	stopped  chan struct{}
	stopOnce sync.Once

	keyring *sshagent.Keyring
	ssh     *sshState
}

func NewServer(socketPath string) *Server {
	return &Server{
		entries:    make(map[string]*cacheEntry),
		socketPath: socketPath,
		done:       make(chan struct{}),
		stopped:    make(chan struct{}),
	}
}

// EnableSSH must be called before Start.
func (s *Server) EnableSSH(cfg SSHConfig) {
	s.keyring = sshagent.NewKeyring(cfg.Confirm)
	s.ssh = &sshState{cfg: cfg, loaded: make(map[string]vaultStamp)}
	if cfg.UnlockVault != "" {
		s.keyring.OnEmpty = s.unlockOnDemand
	}
}

func (s *Server) Start() error {
	os.Remove(s.socketPath)

	oldMask := syscall.Umask(0077)
	ln, err := net.Listen("unix", s.socketPath)
	if err == nil && s.ssh != nil {
		os.Remove(s.ssh.cfg.SocketPath)
		s.ssh.listener, err = net.Listen("unix", s.ssh.cfg.SocketPath)
	}
	syscall.Umask(oldMask)
	if err != nil {
		if ln != nil {
			ln.Close()
		}
		return err
	}
	s.listener = ln

	go s.reaper()
	if s.ssh != nil {
		go s.serveSSH()
	}
	s.serve()
	<-s.stopped
	return nil
}

// Done is closed once the agent has shut down and cleaned up.
func (s *Server) Done() <-chan struct{} {
	return s.stopped
}

func (s *Server) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				continue
			}
		}
		go s.handleConn(conn)
	}
}

func (s *Server) serveSSH() {
	for {
		conn, err := s.ssh.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				continue
			}
		}
		go s.keyring.ServeConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	var req Request
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		json.NewEncoder(conn).Encode(&Response{Error: "invalid request"})
		return
	}

	var resp *Response
	switch req.Op {
	case OpPing:
		resp = &Response{OK: true}
	case OpStore:
		resp = s.handleStore(&req)
	case OpRetrieve:
		resp = s.handleRetrieve(&req)
	case OpLock:
		resp = s.handleLock()
	default:
		resp = &Response{Error: "unknown operation"}
	}

	json.NewEncoder(conn).Encode(resp)
}

func (s *Server) handleStore(req *Request) *Response {
	s.mu.Lock()
	if old, ok := s.entries[req.VaultPath]; ok {
		zeroBytes(old.password)
	}

	pw := make([]byte, len(req.Password))
	copy(pw, req.Password)

	s.entries[req.VaultPath] = &cacheEntry{
		password:  pw,
		expiresAt: time.Now().Add(req.TTL()),
	}
	s.mu.Unlock()

	if s.ssh != nil {
		s.loadSSHKeys(req.VaultPath, req.Password)
	}
	return &Response{OK: true}
}

// loadSSHKeys serves the private keys of an unlocked vault. The vault is read
// again only when its file or password changed since the last load.
func (s *Server) loadSSHKeys(vaultPath string, password []byte) {
	info, err := os.Stat(vaultPath)
	if err != nil {
		s.forgetVault(vaultPath)
		return
	}

	s.mu.Lock()
	prev, seen := s.ssh.loaded[vaultPath]
	s.mu.Unlock()
	if seen && prev.modTime.Equal(info.ModTime()) && prev.size == info.Size() &&
		subtle.ConstantTimeCompare(prev.password, password) == 1 {
		return
	}

	cfg, err := psync.LoadConfig(filepath.Dir(vaultPath))
	if err != nil || cfg.SSHKeyTTL() == 0 {
		s.forgetVault(vaultPath)
		return
	}

	store := &vault.Store{Path: vaultPath}
	v, err := store.Load(password)
	if err != nil {
		s.forgetVault(vaultPath)
		return
	}

	s.keyring.Load(vaultPath, v.Entries, sshagent.LoadOptions{
		TTL:          cfg.SSHKeyTTL(),
		Confirm:      cfg.SSHConfirmEnabled(),
		ConfirmGrace: cfg.SSHConfirmGrace(),
	})

	pw := make([]byte, len(password))
	copy(pw, password)
	s.mu.Lock()
	if old, ok := s.ssh.loaded[vaultPath]; ok {
		zeroBytes(old.password)
	}
	s.ssh.loaded[vaultPath] = vaultStamp{modTime: info.ModTime(), size: info.Size(), password: pw}
	s.mu.Unlock()
}

func (s *Server) forgetVault(vaultPath string) {
	s.keyring.Unload(vaultPath)
	s.mu.Lock()
	if old, ok := s.ssh.loaded[vaultPath]; ok {
		zeroBytes(old.password)
		delete(s.ssh.loaded, vaultPath)
	}
	s.mu.Unlock()
}

func (s *Server) forgetAllVaults() {
	s.keyring.Clear()
	s.mu.Lock()
	for path, stamp := range s.ssh.loaded {
		zeroBytes(stamp.password)
		delete(s.ssh.loaded, path)
	}
	s.mu.Unlock()
}

// unlockOnDemand runs when an SSH client asks for keys while none are loaded.
// It unlocks the configured vault for SSH only: the master password is not
// cached for other commands, so a client cannot widen its access by
// triggering this prompt.
func (s *Server) unlockOnDemand() {
	s.mu.Lock()
	cooldown := unlockCooldown << s.ssh.unlockFailures
	if cooldown > unlockMaxCooldown || cooldown <= 0 {
		cooldown = unlockMaxCooldown
	}
	tooSoon := time.Since(s.ssh.lastUnlockPrompt) < cooldown
	if !tooSoon {
		s.ssh.lastUnlockPrompt = time.Now()
	}
	s.mu.Unlock()
	if tooSoon {
		return
	}

	vaultPath := s.ssh.cfg.UnlockVault
	cfg, err := psync.LoadConfig(filepath.Dir(vaultPath))
	if err != nil || !cfg.SSHUnlockPromptEnabled() {
		return
	}

	var password []byte
	if !cfg.NoPassword {
		if s.ssh.cfg.PromptPassword == nil {
			return
		}
		password, err = s.ssh.cfg.PromptPassword("Unlock the passman vault to use your SSH keys.\n\nMaster password:")
		if err != nil || len(password) == 0 {
			s.recordUnlockAttempt(false)
			return
		}
		defer zeroBytes(password)
	}
	s.loadSSHKeys(vaultPath, password)
	s.recordUnlockAttempt(!s.keyring.Empty())
}

func (s *Server) recordUnlockAttempt(ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ok {
		s.ssh.unlockFailures = 0
		return
	}
	s.ssh.unlockFailures++
}

func (s *Server) handleRetrieve(req *Request) *Response {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.entries[req.VaultPath]
	if !ok || time.Now().After(entry.expiresAt) {
		if ok {
			zeroBytes(entry.password)
			delete(s.entries, req.VaultPath)
		}
		return &Response{OK: false, Error: "not found"}
	}

	pw := make([]byte, len(entry.password))
	copy(pw, entry.password)
	return &Response{OK: true, Password: pw}
}

func (s *Server) handleLock() *Response {
	s.mu.Lock()
	for k, e := range s.entries {
		zeroBytes(e.password)
		delete(s.entries, k)
	}
	s.mu.Unlock()

	if s.ssh != nil {
		s.forgetAllVaults()
	}
	return &Response{OK: true}
}

// idle reports whether the agent has nothing left to serve. An agent that can
// unlock on demand is never idle: ssh clients must always find its socket.
func (s *Server) idle() bool {
	s.mu.Lock()
	noEntries := len(s.entries) == 0
	s.mu.Unlock()
	if s.ssh == nil {
		return noEntries
	}
	return noEntries && s.ssh.cfg.UnlockVault == "" && s.keyring.Empty()
}

func (s *Server) reaper() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			now := time.Now()
			s.mu.Lock()
			for k, e := range s.entries {
				if now.After(e.expiresAt) {
					zeroBytes(e.password)
					delete(s.entries, k)
				}
			}
			s.mu.Unlock()
			if s.ssh != nil {
				s.keyring.Expire(now)
			}

			if s.idle() {
				s.Stop()
				return
			}
		}
	}
}

// Stop shuts the agent down. It is safe to call more than once and from any
// goroutine (reaper, signal handler, tests).
func (s *Server) Stop() {
	s.stopOnce.Do(func() {
		close(s.done)

		if s.listener != nil {
			s.listener.Close()
		}
		if s.ssh != nil && s.ssh.listener != nil {
			s.ssh.listener.Close()
		}

		s.mu.Lock()
		for k, e := range s.entries {
			zeroBytes(e.password)
			delete(s.entries, k)
		}
		s.mu.Unlock()

		if s.ssh != nil {
			s.forgetAllVaults()
			os.Remove(s.ssh.cfg.SocketPath)
		}
		os.Remove(s.socketPath)
		close(s.stopped)
	})
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
