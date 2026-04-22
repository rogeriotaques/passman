package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var vaultsCmd = &cobra.Command{
	Use:   "vaults",
	Short: "List all vaults",
	Long:    `List all vaults found under ~/.passman/vaults/.`,
	Example: `  passman vaults`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		vaultsDir := filepath.Join(home, ".passman", "vaults")

		entries, err := os.ReadDir(vaultsDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintln(app.Out, "No vaults found.")
				return nil
			}
			return err
		}

		found := false
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			vaultFile := filepath.Join(vaultsDir, entry.Name(), "vault.enc")
			if _, err := os.Stat(vaultFile); err == nil {
				fmt.Fprintln(app.Out, entry.Name())
				found = true
			}
		}

		if !found {
			fmt.Fprintln(app.Out, "No vaults found.")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(vaultsCmd)
}
