package cmd

import (
	"fmt"
	"time"

	"github.com/rogerio/passman/internal/totp"
	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
)

var totpCmd = &cobra.Command{
	Use:   "totp <name>",
	Short: "Generate a TOTP code for an entry",
	Long: `Generate a time-based one-time password (TOTP) for an entry that has a
TOTP secret configured (see 'passman add --totp'). Shows the 6-digit
code and seconds until expiry.`,
	Example: `  passman totp github`,
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

		if entry.TotpSecret == "" {
			return fmt.Errorf("entry %q has no TOTP secret configured", name)
		}

		now := time.Now()
		code, err := totp.GenerateAt(entry.TotpSecret, now)
		if err != nil {
			return err
		}

		remaining := totp.TimeRemaining(now)
		fmt.Fprintf(app.Out, "%s (expires in %ds)\n", code, remaining)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(totpCmd)
}
