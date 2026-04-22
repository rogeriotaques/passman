package cmd

import (
	"fmt"

	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Remove a credential from the vault",
	Long:    `Permanently remove a credential from the vault by name.`,
	Example: `  passman rm old-service`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		store := &vault.Store{Path: app.VaultPath, KDFParams: app.KDFParams}

		password, err := getPassword()
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(password)

		v, err := store.Load(password)
		if err != nil {
			return err
		}

		if err := v.Remove(name); err != nil {
			return err
		}

		if err := store.Save(v, password); err != nil {
			return err
		}

		fmt.Fprintf(app.Out, "Entry %q removed.\n", name)
		app.backgroundSync()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
