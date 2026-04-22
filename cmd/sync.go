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
	Long: `Sync the vault directory with its git remote (commit + pull + push).
The vault must have been initialized with --git.

Use --auto on/off to toggle automatic sync after every write operation.`,
	Example: `  passman sync
  passman sync --auto on
  passman sync --auto off`,
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultDir := filepath.Dir(app.VaultPath)

		autoFlag, _ := cmd.Flags().GetString("auto")
		if autoFlag != "" {
			cfg, err := psync.LoadConfig(vaultDir)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			switch autoFlag {
			case "on":
				cfg.AutoSync = true
			case "off":
				cfg.AutoSync = false
			default:
				return fmt.Errorf("invalid value for --auto: %q (use 'on' or 'off')", autoFlag)
			}

			if err := psync.SaveConfig(vaultDir, cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Fprintf(app.Out, "Auto-sync %s.\n", autoFlag)
			return nil
		}

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
	syncCmd.Flags().String("auto", "", "toggle auto-sync ('on' or 'off')")
	rootCmd.AddCommand(syncCmd)
}
