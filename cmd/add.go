package cmd

import (
	"fmt"

	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <secret-name>",
	Short: "Add a secret to the vault",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		v, store, password, err := app.loadVault()
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(password)

		value, err := readPassword("Secret value: ")
		if err != nil {
			return err
		}

		entry := vault.Entry{
			Name:  name,
			Value: string(value),
		}

		if err := v.Add(entry); err != nil {
			return err
		}

		if err := app.saveAndSync(store, v, password); err != nil {
			return err
		}

		fmt.Fprintf(app.Out, "Entry %q added.\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
