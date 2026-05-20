package cmd

import (
	"fmt"
	"path/filepath"

	psync "github.com/rogerio/passman/internal/sync"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync vault with remote git repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultDir := filepath.Dir(app.VaultPath)

		if !psync.IsRepo(vaultDir) {
			return fmt.Errorf("vault is not a git repository (use 'passman init --git' first)")
		}

		if err := psync.Sync(vaultDir); err != nil {
			return fmt.Errorf("sync: %w", err)
		}

		fmt.Fprintln(app.Out, "Vault synced.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
