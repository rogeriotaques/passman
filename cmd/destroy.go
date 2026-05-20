package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Permanently delete a vault or all vaults",
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultDir := filepath.Dir(app.VaultPath)
		vaultsDir := filepath.Dir(vaultDir)

		name := vaultName
		if name != "" {
			return destroyVault(vaultsDir, name)
		}

		confirm, err := readInput("This will delete ALL vaults. Type 'yes' to confirm: ")
		if err != nil {
			return err
		}
		if confirm != "yes" {
			fmt.Fprintln(app.Out, "Aborted.")
			return nil
		}

		entries, err := os.ReadDir(vaultsDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintln(app.Out, "No vaults found.")
				return nil
			}
			return err
		}

		count := 0
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			dir := filepath.Join(vaultsDir, entry.Name())
			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("remove vault %q: %w", entry.Name(), err)
			}
			count++
		}

		fmt.Fprintf(app.Out, "Destroyed %d vault(s).\n", count)
		return nil
	},
}

func destroyVault(vaultsDir, name string) error {
	dir := filepath.Join(vaultsDir, name)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("vault %q not found", name)
	}

	confirm, err := readInput(fmt.Sprintf("This will delete vault %q. Type 'yes' to confirm: ", name))
	if err != nil {
		return err
	}
	if confirm != "yes" {
		fmt.Fprintln(app.Out, "Aborted.")
		return nil
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove vault: %w", err)
	}

	fmt.Fprintf(app.Out, "Vault %q destroyed.\n", name)
	return nil
}

func init() {
	rootCmd.AddCommand(destroyCmd)
}
