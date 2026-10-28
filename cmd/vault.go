package cmd

import "github.com/spf13/cobra"

var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Manage vaults",
	Long:  `Create, list, configure, sync, re-key, and destroy vaults.`,
}

func init() {
	rootCmd.AddCommand(vaultCmd)
}
