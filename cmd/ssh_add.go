package cmd

import (
	"crypto"
	"crypto/ed25519"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

// maxKeyFileSize bounds what `ssh add` reads; real private keys are a few KiB.
const maxKeyFileSize = 64 * 1024

var sshAddCmd = &cobra.Command{
	Use:   "add <private-key-file>",
	Short: "Import a private key into the vault",
	Long: `Read an SSH private key file, remove its passphrase if it has one, and store
it in the vault so the agent can serve it. The entry is named ` + sshKeyPrefix + `<file name>
unless --name is given. The file itself is left untouched.`,
	Example: `  passman ssh add ~/.ssh/id_ed25519
  passman ssh add ~/.ssh/aws.pem --name ssh/aws-prod`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		keyPath := args[0]
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			name = sshKeyPrefix + filepath.Base(keyPath)
		}

		v, store, password, err := app.loadVault()
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(password)

		if _, err := v.Get(name); err == nil {
			return fmt.Errorf("entry %q already exists", name)
		}

		key, err := readPrivateKeyFile(keyPath)
		if err != nil {
			return err
		}

		block, err := ssh.MarshalPrivateKey(key, name)
		if err != nil {
			return fmt.Errorf("unsupported key type: %w", err)
		}
		signer, err := ssh.NewSignerFromKey(key)
		if err != nil {
			return fmt.Errorf("unsupported key type: %w", err)
		}

		if err := v.Add(vault.Entry{Name: name, Value: string(pem.EncodeToMemory(block))}); err != nil {
			return err
		}
		if err := app.saveAndSync(store, v, password); err != nil {
			return err
		}

		pub := signer.PublicKey()
		fmt.Fprintf(app.Out, "Added %s\n", describeKey(name, pub))
		fmt.Fprintf(app.Out, "Public key: %s", ssh.MarshalAuthorizedKey(pub))
		fmt.Fprintln(app.Out, "The agent serves this key once the vault is unlocked. Keep a backup of the file before deleting it.")
		return nil
	},
}

// readPrivateKeyFile parses the key at path, asking for its passphrase when
// the file is encrypted.
func readPrivateKeyFile(path string) (crypto.PrivateKey, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}
	if info.Size() > maxKeyFileSize {
		return nil, fmt.Errorf("%s is too large to be a private key (%d bytes)", path, info.Size())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}
	defer vault.ZeroBytes(data)

	key, err := ssh.ParseRawPrivateKey(data)
	var missing *ssh.PassphraseMissingError
	if errors.As(err, &missing) {
		passphrase, perr := readPassword(fmt.Sprintf("Passphrase for %s: ", filepath.Base(path)))
		if perr != nil {
			return nil, perr
		}
		defer vault.ZeroBytes(passphrase)
		key, err = ssh.ParseRawPrivateKeyWithPassphrase(data, passphrase)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot use %s: %w", filepath.Base(path), err)
	}
	return normalizeKey(key), nil
}

// normalizeKey aligns parser output with what ssh.MarshalPrivateKey accepts:
// OpenSSH-format ed25519 keys parse to a pointer, but marshal as a value.
func normalizeKey(key crypto.PrivateKey) crypto.PrivateKey {
	if k, ok := key.(*ed25519.PrivateKey); ok {
		return *k
	}
	return key
}

func init() {
	sshAddCmd.Flags().String("name", "", "entry name (default: "+sshKeyPrefix+"<file name>)")
	sshCmd.AddCommand(sshAddCmd)
}
