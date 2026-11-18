//go:build !darwin

package prompt

import (
	"os/exec"
	"strings"
)

// askpass delegates to an SSH_ASKPASS-compatible helper (ssh-askpass, zenity
// wrappers, ...), the de facto standard for graphical SSH prompts on Linux.
type askpass struct {
	path string
}

// System returns a prompter backed by ssh-askpass when one is installed.
func System() Prompter {
	path, err := exec.LookPath("ssh-askpass")
	if err != nil {
		return unavailable{}
	}
	return askpass{path: path}
}

func (a askpass) run(message, promptType string) (string, error) {
	cmd := exec.Command(a.path, message)
	cmd.Env = append(cmd.Environ(), "SSH_ASKPASS_PROMPT="+promptType)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(out), "\n"), nil
}

func (a askpass) Confirm(message string) (bool, error) {
	// By convention the helper exits non-zero when the user declines.
	_, err := a.run(message, "confirm")
	if _, declined := err.(*exec.ExitError); declined {
		return false, nil
	}
	return err == nil, err
}

func (a askpass) Secret(message string) ([]byte, error) {
	out, err := a.run(message, "")
	if err != nil {
		return nil, err
	}
	return []byte(out), nil
}

func (a askpass) Text(message string) (string, error) {
	return a.run(message, "none")
}

type unavailable struct{}

func (unavailable) Confirm(string) (bool, error)  { return false, ErrUnavailable }
func (unavailable) Secret(string) ([]byte, error) { return nil, ErrUnavailable }
func (unavailable) Text(string) (string, error)   { return "", ErrUnavailable }
