package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/rogerio/passman/internal/agent"
	psync "github.com/rogerio/passman/internal/sync"
	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
)

var passwdCmd = &cobra.Command{
	Use:   "passwd",
	Short: "Change the master password",
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

		if len(newPassword) > 0 && len(newPassword) < minPasswordLen {
			return fmt.Errorf("new password must be at least %d characters", minPasswordLen)
		}

		confirm, err := readPassword("Confirm new master password: ")
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(confirm)

		if string(newPassword) != string(confirm) {
			return fmt.Errorf("passwords do not match")
		}

		if err := store.Save(v, newPassword); err != nil {
			return fmt.Errorf("save vault: %w", err)
		}

		vaultDir := filepath.Dir(app.VaultPath)
		cfg, _ := psync.LoadConfig(vaultDir)
		cfg.NoPassword = len(newPassword) == 0
		_ = psync.SaveConfig(vaultDir, cfg)

		if app.SocketPath != "" {
			client := agent.NewClient(app.SocketPath)
			if client.Ping() {
				_ = client.Lock()
			}
		}

		cacheInAgent(newPassword)
		fmt.Fprintln(app.Out, "Master password changed.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(passwdCmd)
}
