package cmd

import "github.com/spf13/cobra"

var secretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Manage secrets",
	Long:  `Add, read, list, remove, import, and export secrets in a vault.`,
}

func init() {
	rootCmd.AddCommand(secretCmd)
}
