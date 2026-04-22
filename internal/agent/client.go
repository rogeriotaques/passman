package agent

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"
)

type Client struct {
	socketPath string
}

func NewClient(socketPath string) *Client {
	return &Client{socketPath: socketPath}
}

func (c *Client) send(req *Request) (*Response, error) {
	conn, err := net.DialTimeout("unix", c.socketPath, 2*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return nil, err
	}

	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Ping() bool {
	resp, err := c.send(&Request{Op: OpPing})
	return err == nil && resp.OK
}

func (c *Client) Store(vaultPath string, password []byte, ttl time.Duration) error {
	resp, err := c.send(&Request{
		Op:        OpStore,
		VaultPath: vaultPath,
		Password:  password,
		TTLMillis: ttl.Milliseconds(),
	})
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("agent store: %s", resp.Error)
	}
	return nil
}

func (c *Client) Retrieve(vaultPath string) ([]byte, error) {
	resp, err := c.send(&Request{
		Op:        OpRetrieve,
		VaultPath: vaultPath,
	})
	if err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("agent retrieve: %s", resp.Error)
	}
	return resp.Password, nil
}

func (c *Client) Lock() error {
	resp, err := c.send(&Request{Op: OpLock})
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("agent lock: %s", resp.Error)
	}
	return nil
}

func EnsureAgent(socketPath string) error {
	c := NewClient(socketPath)
	if c.Ping() {
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find executable: %w", err)
	}

	cmd := exec.Command(exe, "_agent", "--socket", socketPath)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start agent: %w", err)
	}
	cmd.Process.Release()

	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		if c.Ping() {
			return nil
		}
	}
	return fmt.Errorf("agent did not start within 2 seconds")
}
