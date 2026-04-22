package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Retrieve a credential from the vault",
	Long: `Retrieve a credential by name. By default the password is copied to the
clipboard and automatically cleared after 30 seconds.

Use -p/--print to display the password on stdout instead of copying.`,
	Example: `  passman get github
  passman get github --print`,
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

		entry, err := v.Get(name)
		if err != nil {
			return err
		}

		printToStdout, _ := cmd.Flags().GetBool("print")
		if printToStdout {
			showEntry(entry, true)
			return nil
		}
		return copyAndShowEntry(entry)
	},
}

func showEntry(entry *vault.Entry, showPassword bool) {
	fmt.Fprintf(app.Out, "Name:     %s\n", entry.Name)
	fmt.Fprintf(app.Out, "Username: %s\n", entry.Username)
	if showPassword {
		fmt.Fprintf(app.Out, "Password: %s\n", entry.Password)
	} else {
		fmt.Fprintln(app.Out, "Password copied to clipboard (clears in 30s).")
	}
	if len(entry.Tags) > 0 {
		fmt.Fprintf(app.Out, "Tags:     %s\n", strings.Join(entry.Tags, ", "))
	}
	if entry.TotpSecret != "" {
		fmt.Fprintln(app.Out, "TOTP:     configured (use 'passman totp' to generate code)")
	}
	if entry.Notes != "" {
		fmt.Fprintf(app.Out, "Notes:    %s\n", entry.Notes)
	}
}

func copyAndShowEntry(entry *vault.Entry) error {
	if app.Clipboard != nil {
		if err := app.Clipboard.CopyWithAutoClear(entry.Password, 30*time.Second); err != nil {
			return fmt.Errorf("copy to clipboard: %w", err)
		}
		showEntry(entry, false)
	} else {
		showEntry(entry, true)
	}
	return nil
}

func init() {
	getCmd.Flags().BoolP("print", "p", false, "print password to stdout instead of clipboard")
	rootCmd.AddCommand(getCmd)
}
