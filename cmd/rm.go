package cmd

import (
	"fmt"

	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm <secret-name>",
	Short: "Remove a secret from the vault",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		v, store, password, err := app.loadVault()
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(password)

		if err := v.Remove(name); err != nil {
			return err
		}

		if err := app.saveAndSync(store, v, password); err != nil {
			return err
		}

		fmt.Fprintf(app.Out, "Entry %q removed.\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
