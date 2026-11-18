package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
)

// keyPathPattern extracts the key file from OpenSSH passphrase prompts:
//
//	Enter passphrase for key '/home/u/.ssh/id_ed25519':                    (ssh)
//	Enter passphrase for /home/u/.ssh/id_ed25519:                          (ssh-add)
//	Enter passphrase for /home/u/.ssh/id_ed25519 (will confirm each use):  (ssh-add -c)
//	Enter passphrase for "id_ed25519":                                     (ssh-keygen)
var keyPathPattern = regexp.MustCompile(`^Enter passphrase for (?:key )?["']?([^"':]+?)["']?(?: \(will confirm each use\))?:\s*$`)

var sshAskpassCmd = &cobra.Command{
	Use:   "askpass [prompt]",
	Short: "Answer an SSH_ASKPASS request",
	Long: `Implements the SSH_ASKPASS protocol: ssh passes the prompt as the only
argument and reads the answer from stdout. Set up through "passman ssh env".

Passphrase prompts for a key file are answered from the vault entry named
` + sshPassphrasePrefix + `<key file name> when it exists. Everything else is
asked in the terminal, or through a dialog when there is none. Confirmation
prompts (SSH_ASKPASS_PROMPT=confirm) exit 0 for yes and 1 for no.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		message := strings.TrimSpace(strings.Join(args, " "))
		if message == "" {
			message = "ssh prompt"
		}
		attachTTY()
		app.GUIPassword = true

		switch os.Getenv("SSH_ASKPASS_PROMPT") {
		case "confirm":
			ok, err := app.Prompter.Confirm(message)
			if err != nil {
				return err
			}
			if !ok {
				return errors.New("denied")
			}
			return nil
		case "none":
			answer, err := readVisible(message)
			if err != nil {
				return err
			}
			fmt.Fprintln(app.Out, answer)
			return nil
		}

		if m := keyPathPattern.FindStringSubmatch(message); m != nil {
			if passphrase, ok := vaultPassphrase(filepath.Base(m[1])); ok {
				fmt.Fprintln(app.Out, passphrase)
				return nil
			}
		}

		answer, err := readPassword(message + " ")
		if err != nil {
			return err
		}
		fmt.Fprintln(app.Out, string(answer))
		return nil
	},
}

// vaultPassphrase looks up the passphrase stored for a key file. Any failure
// (locked vault, cancelled prompt, missing entry) falls back to asking.
func vaultPassphrase(keyFile string) (string, bool) {
	v, _, password, err := app.loadVault()
	if err != nil {
		return "", false
	}
	defer vault.ZeroBytes(password)

	entry, err := v.Get(sshPassphrasePrefix + keyFile)
	if err != nil {
		return "", false
	}
	return entry.Value, true
}

// attachTTY prompts on the controlling terminal when there is one, even if
// ssh was started with stdin redirected (git, pipelines). Without a terminal
// the prompts fall back to dialogs.
func attachTTY() {
	if f, ok := app.In.(*os.File); ok && isTerminal(f) {
		return
	}
	tty, err := os.OpenFile(app.TTYPath, os.O_RDWR, 0)
	if err != nil {
		return
	}
	if !isTerminal(tty) {
		tty.Close()
		return
	}
	app.In = tty
	app.ErrOut = tty
	resetInputReader()
}

// readVisible answers prompts whose input need not be hidden, such as host
// key confirmations, preferring the terminal when there is one.
func readVisible(message string) (string, error) {
	if f, ok := app.In.(*os.File); ok && isTerminal(f) {
		return readInput(message + " ")
	}
	return app.Prompter.Text(message)
}

func init() {
	sshCmd.AddCommand(sshAskpassCmd)
}
