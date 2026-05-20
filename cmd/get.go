package cmd

import (
	"fmt"
	"time"

	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <secret-name>",
	Short: "Retrieve a secret from the vault",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		v, _, password, err := app.loadVault()
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(password)

		entry, err := v.Get(name)
		if err != nil {
			return err
		}

		printToStdout, _ := cmd.Flags().GetBool("print")
		if printToStdout {
			fmt.Fprintln(app.Out, entry.Value)
			return nil
		}

		return copyToClipboard(entry.Value)
	},
}

func copyToClipboard(value string) error {
	if app.Clipboard == nil {
		fmt.Fprintln(app.Out, value)
		return nil
	}
	if err := app.Clipboard.CopyWithAutoClear(value, 30*time.Second); err != nil {
		return fmt.Errorf("copy to clipboard: %w", err)
	}
	fmt.Fprintln(app.Out, "Secret copied to clipboard (clears in 30s).")
	return nil
}

func init() {
	getCmd.Flags().BoolP("print", "p", false, "print secret to stdout instead of clipboard")
	rootCmd.AddCommand(getCmd)
}
