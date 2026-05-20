package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rogerio/passman/internal/agent"
	psync "github.com/rogerio/passman/internal/sync"
	"golang.org/x/term"
)

var inputReader *bufio.Reader

func getInputReader() *bufio.Reader {
	if inputReader == nil {
		inputReader = bufio.NewReader(app.In)
	}
	return inputReader
}

func resetInputReader() {
	inputReader = nil
}

func readPassword(prompt string) ([]byte, error) {
	fmt.Fprint(app.ErrOut, prompt)

	if f, ok := app.In.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		pw, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(app.ErrOut)
		return pw, err
	}

	line, err := readLineFromReader()
	if err != nil {
		return nil, err
	}
	return []byte(line), nil
}

func readLineFromReader() (string, error) {
	r := getInputReader()
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" && err == io.EOF {
		return "", io.ErrUnexpectedEOF
	}
	return line, nil
}

func readInput(prompt string) (string, error) {
	fmt.Fprint(app.ErrOut, prompt)
	return readLineFromReader()
}

func getPassword() ([]byte, error) {
	if isNoPassword() {
		return []byte(""), nil
	}

	if app.SocketPath != "" {
		client := agent.NewClient(app.SocketPath)
		if pw, err := client.Retrieve(app.VaultPath); err == nil {
			return pw, nil
		}
	}

	pw, err := readPassword("Master password: ")
	if err != nil {
		return nil, err
	}

	cacheInAgent(pw)
	return pw, nil
}

func isNoPassword() bool {
	vaultDir := filepath.Dir(app.VaultPath)
	cfg, err := psync.LoadConfig(vaultDir)
	if err != nil {
		return false
	}
	return cfg.NoPassword
}

func cacheInAgent(pw []byte) {
	if app.SocketPath == "" || len(pw) == 0 {
		return
	}
	ttl := loadSessionTimeout()
	if ttl <= 0 {
		return
	}
	client := agent.NewClient(app.SocketPath)
	if err := client.Store(app.VaultPath, pw, ttl); err == nil {
		return
	}
	if agent.EnsureAgent(app.SocketPath) == nil {
		client = agent.NewClient(app.SocketPath)
		_ = client.Store(app.VaultPath, pw, ttl)
	}
}

func loadSessionTimeout() time.Duration {
	vaultDir := filepath.Dir(app.VaultPath)
	cfg, err := psync.LoadConfig(vaultDir)
	if err != nil {
		return agent.DefaultTTL
	}
	val, err := cfg.Get("session-timeout")
	if err != nil {
		return agent.DefaultTTL
	}
	minutes, err := strconv.Atoi(val)
	if err != nil {
		return agent.DefaultTTL
	}
	return time.Duration(minutes) * time.Minute
}
