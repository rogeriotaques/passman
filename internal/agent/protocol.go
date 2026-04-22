package agent

import (
	"os"
	"path/filepath"
	"time"
)

const DefaultTTL = 15 * time.Minute

type OpCode string

const (
	OpPing     OpCode = "ping"
	OpStore    OpCode = "store"
	OpRetrieve OpCode = "retrieve"
	OpLock     OpCode = "lock"
)

type Request struct {
	Op        OpCode        `json:"op"`
	VaultPath string        `json:"vault_path,omitempty"`
	Password  []byte        `json:"password,omitempty"`
	TTLMillis int64         `json:"ttl_millis,omitempty"`
}

func (r *Request) TTL() time.Duration {
	if r.TTLMillis <= 0 {
		return DefaultTTL
	}
	return time.Duration(r.TTLMillis) * time.Millisecond
}

type Response struct {
	OK       bool   `json:"ok"`
	Password []byte `json:"password,omitempty"`
	Error    string `json:"error,omitempty"`
}

func DefaultSocketPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".passman", "agent.sock")
}
