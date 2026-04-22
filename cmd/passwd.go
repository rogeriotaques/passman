package cmd

import (
	"fmt"

	"github.com/rogerio/passman/internal/agent"
	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
)

var passwdCmd = &cobra.Command{
	Use:   "passwd",
	Short: "Change the master password",
	Long: `Change the vault master password. You will be prompted for the current
password, then a new password (minimum 8 characters) with confirmation.

The vault is decrypted with the old password and re-encrypted with the
new one. Any cached password in the agent is cleared.`,
	Example: `  passman passwd`,
	RunE: func(cmd *cobra.Command, args []string) error {
		store := &vault.Store{Path: app.VaultPath, KDFParams: app.KDFParams}

		oldPassword, err := readPassword("Current master password: ")
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(oldPassword)

		v, err := store.Load(oldPassword)
		if err != nil {
			return err
		}

		newPassword, err := readPassword("New master password: ")
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(newPassword)

		confirm, err := readPassword("Confirm new master password: ")
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(confirm)

		const minPasswordLen = 8
		if len(newPassword) < minPasswordLen {
			return fmt.Errorf("new password must be at least %d characters", minPasswordLen)
		}

		if string(newPassword) != string(confirm) {
			return fmt.Errorf("passwords do not match")
		}

		if err := store.Save(v, newPassword); err != nil {
			return fmt.Errorf("save vault: %w", err)
		}

		if app.SocketPath != "" {
			client := agent.NewClient(app.SocketPath)
			if client.Ping() {
				_ = client.Lock()
			}
		}

		storePasswordInAgent(newPassword)
		fmt.Fprintln(app.Out, "Master password changed.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(passwdCmd)
}
