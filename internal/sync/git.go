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

func Init(dir string) error {
	return gitCmd(dir, "init").Run()
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

	if HasRemote(dir) {
		if err := gitCmd(dir, "pull", "--rebase", "origin", "main").Run(); err != nil {
			return fmt.Errorf("git pull: %w", err)
		}

		if err := gitCmd(dir, "push", "-u", "origin", "main").Run(); err != nil {
			return fmt.Errorf("git push: %w", err)
		}
	}

	return nil
}

func hasChanges(dir string) bool {
	out, err := gitCmd(dir, "status", "--porcelain").Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

func gitCmd(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=passman",
		"GIT_AUTHOR_EMAIL=passman@localhost",
		"GIT_COMMITTER_NAME=passman",
		"GIT_COMMITTER_EMAIL=passman@localhost",
	)
	return cmd
}
