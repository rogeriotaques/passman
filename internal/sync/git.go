package sync

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var ErrNotARepo = errors.New("not a git repository")

// branch is fixed so Sync does not depend on the user's init.defaultBranch.
const branch = "main"

// ignoreRules keeps the advisory lock and atomic-write temp files out of the
// repository; only the encrypted vault and its config belong there.
const ignoreRules = "*.lock\n*.tmp\n"

func Init(dir string) error {
	if err := gitCmd(dir, "init", "-b", branch).Run(); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(ignoreRules), 0600)
}

func IsRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func SetRemote(dir, url string) error {
	if HasRemote(dir) {
		return gitCmd(dir, "remote", "set-url", "origin", url).Run()
	}
	return gitCmd(dir, "remote", "add", "origin", url).Run()
}

func HasRemote(dir string) bool {
	out, err := gitCmd(dir, "remote").Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == "origin" {
			return true
		}
	}
	return false
}

func Sync(dir string) error {
	if !IsRepo(dir) {
		return ErrNotARepo
	}

	if err := gitCmd(dir, "add", "-A").Run(); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	if hasChanges(dir) {
		if err := gitCmd(dir, "commit", "-m", "vault update").Run(); err != nil {
			return fmt.Errorf("git commit: %w", err)
		}
	}

	if !HasRemote(dir) {
		return nil
	}

	// Push first: every connection may cost the user an SSH approval, and the
	// remote is usually either empty or already behind this machine. Only a
	// rejected push (remote moved on) pays for a pull.
	if push(dir) == nil {
		return nil
	}
	if err := gitCmd(dir, "pull", "--rebase", "origin", branch).Run(); err != nil {
		return fmt.Errorf("git pull: %w", err)
	}
	if err := push(dir); err != nil {
		return fmt.Errorf("git push: %w", err)
	}
	return nil
}

func push(dir string) error {
	return gitCmd(dir, "push", "-u", "origin", branch).Run()
}

func hasChanges(dir string) bool {
	out, err := gitCmd(dir, "status", "--porcelain").Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

func gitCmd(dir string, args ...string) *exec.Cmd {
	// Vault commits are machine-made bookkeeping: never sign them (signing
	// would prompt for a key passphrase from a background goroutine) and keep
	// the identity fixed.
	fullArgs := append([]string{"-c", "commit.gpgsign=false"}, args...)
	cmd := exec.Command("git", fullArgs...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=passman",
		"GIT_AUTHOR_EMAIL=passman@localhost",
		"GIT_COMMITTER_NAME=passman",
		"GIT_COMMITTER_EMAIL=passman@localhost",
		"GIT_TERMINAL_PROMPT=0",
	)
	return cmd
}
