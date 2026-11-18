package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var sshEnvCmd = &cobra.Command{
	Use:   "env",
	Short: "Print the shell exports that route ssh through passman",
	Long: `Print export statements for SSH_AUTH_SOCK, SSH_ASKPASS, and
SSH_ASKPASS_REQUIRE, and (re)write the small SSH_ASKPASS helper they point at.

SSH_ASKPASS_REQUIRE is set to "force" because ssh only consults SSH_ASKPASS
when DISPLAY is set otherwise, which macOS terminals do not do.

It also starts the agent when it is not running (after a reboot, for
example), so SSH_AUTH_SOCK always points at a live socket. The agent then
offers to unlock the vault the first time ssh needs a key.`,
	Example: `  eval "$(passman ssh env)"      # add this line to ~/.zshrc`,
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := writeAskpassHelper(app.AskpassPath); err != nil {
			return err
		}
		// Runs from shell startup files: a failure must not break the shell.
		if app.AutoStartAgent && app.SocketPath != "" {
			if err := ensureAgent(app.SocketPath, app.SSHSocketPath, app.SSHUnlockVault); err != nil {
				fmt.Fprintf(app.ErrOut, "Warning: could not start the passman agent: %v\n", err)
			}
		}
		for _, kv := range [][2]string{
			{"SSH_AUTH_SOCK", app.SSHSocketPath},
			{"SSH_ASKPASS", app.AskpassPath},
			{"SSH_ASKPASS_REQUIRE", "force"},
		} {
			fmt.Fprintf(app.Out, "export %s='%s'\n", kv[0], shellEscape(kv[1]))
		}
		return nil
	},
}

// writeAskpassHelper installs a wrapper that ssh can exec as SSH_ASKPASS
// (ssh accepts a program path only, no arguments).
func writeAskpassHelper(path string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate passman binary: %w", err)
	}
	content := []byte("#!/bin/sh\nexec '" + shellEscape(exe) + "' ssh askpass \"$@\"\n")

	if current, err := os.ReadFile(path); err == nil && bytes.Equal(current, content) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create helper directory: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, content, 0700); err != nil {
		return fmt.Errorf("write askpass helper: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("install askpass helper: %w", err)
	}
	return nil
}

func init() {
	sshCmd.AddCommand(sshEnvCmd)
}
