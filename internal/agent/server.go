package agent

import (
	"encoding/json"
	"net"
	"os"
	"sync"
	"syscall"
	"time"
)

type cacheEntry struct {
	password  []byte
	expiresAt time.Time
}

type Server struct {
	mu         sync.Mutex
	entries    map[string]*cacheEntry
	socketPath string
	listener   net.Listener
	done       chan struct{}
}

func NewServer(socketPath string) *Server {
	return &Server{
		entries:    make(map[string]*cacheEntry),
		socketPath: socketPath,
		done:       make(chan struct{}),
	}
}

func (s *Server) Start() error {
	os.Remove(s.socketPath)

	oldMask := syscall.Umask(0077)
	ln, err := net.Listen("unix", s.socketPath)
	syscall.Umask(oldMask)
	if err != nil {
		return err
	}
	s.listener = ln

	go s.reaper()
	s.serve()
	return nil
}

func (s *Server) Done() <-chan struct{} {
	return s.done
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
	defer s.mu.Unlock()

	if old, ok := s.entries[req.VaultPath]; ok {
		zeroBytes(old.password)
	}

	pw := make([]byte, len(req.Password))
	copy(pw, req.Password)

	s.entries[req.VaultPath] = &cacheEntry{
		password:  pw,
		expiresAt: time.Now().Add(req.TTL()),
	}
	return &Response{OK: true}
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
	defer s.mu.Unlock()

	for k, e := range s.entries {
		zeroBytes(e.password)
		delete(s.entries, k)
	}
	return &Response{OK: true}
}

func (s *Server) reaper() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for k, e := range s.entries {
				if now.After(e.expiresAt) {
					zeroBytes(e.password)
					delete(s.entries, k)
				}
			}
			empty := len(s.entries) == 0
			s.mu.Unlock()

			if empty {
				s.Stop()
				return
			}
		}
	}
}

func (s *Server) Stop() {
	select {
	case <-s.done:
		return
	default:
		close(s.done)
	}

	if s.listener != nil {
		s.listener.Close()
	}

	s.mu.Lock()
	for k, e := range s.entries {
		zeroBytes(e.password)
		delete(s.entries, k)
	}
	s.mu.Unlock()

	os.Remove(s.socketPath)
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
