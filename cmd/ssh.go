package cmd

import (
	"fmt"
	"strings"

	"github.com/rogerio/passman/internal/sshagent"
	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

// sshKeyPrefix groups private keys in the vault; passphrases of file-based
// keys live under sshPassphrasePrefix + <key file name>.
const (
	sshKeyPrefix        = "ssh/"
	sshPassphrasePrefix = "ssh/passphrase/"
)

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Use the vault as an SSH agent",
	Long: `Store SSH private keys in the vault and serve them to ssh through the
passman agent (SSH_AUTH_SOCK). Keys never leave the agent: ssh only receives
signatures, each one approved by you when ssh-confirm is on.

The agent also answers SSH_ASKPASS requests, supplying passphrases of
file-based keys from entries named ` + sshPassphrasePrefix + `<key file name>.`,
}

// sshKeyEntry pairs a vault entry with the signer parsed from it.
type sshKeyEntry struct {
	vault.Entry
	signer ssh.Signer
}

func sshKeyEntries(v *vault.Vault) []sshKeyEntry {
	var keys []sshKeyEntry
	for _, e := range v.Search("") {
		if signer, ok := sshagent.ParseKey(e.Value); ok {
			keys = append(keys, sshKeyEntry{Entry: e, signer: signer})
		}
	}
	return keys
}

func describeKey(name string, pub ssh.PublicKey) string {
	return fmt.Sprintf("%s  %s  %s", name, strings.TrimPrefix(pub.Type(), "ssh-"), ssh.FingerprintSHA256(pub))
}

func init() {
	rootCmd.AddCommand(sshCmd)
}
