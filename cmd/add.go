package cmd

import (
	"fmt"

	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a credential to the vault",
	Long: `Add a new credential to the vault. You will be prompted for a username,
password, and optional notes. Use --tag to categorize the entry (can be
repeated). Use --totp to attach a TOTP secret for two-factor auth.`,
	Example: `  passman add github
  passman add aws-prod --tag aws --tag production
  passman add gitlab --totp`,
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

		username, err := readInput("Username: ")
		if err != nil {
			return err
		}

		secret, err := readPassword("Password: ")
		if err != nil {
			return err
		}

		notes, err := readInput("Notes (optional): ")
		if err != nil {
			return err
		}

		tags, _ := cmd.Flags().GetStringArray("tag")

		var totpSecret string
		useTotp, _ := cmd.Flags().GetBool("totp")
		if useTotp {
			totpInput, err := readInput("TOTP secret (base32): ")
			if err != nil {
				return err
			}
			totpSecret = totpInput
		}

		entry := vault.Entry{
			Name:       name,
			Username:   username,
			Password:   string(secret),
			Notes:      notes,
			Tags:       tags,
			TotpSecret: totpSecret,
		}

		if err := v.Add(entry); err != nil {
			return err
		}

		if err := store.Save(v, password); err != nil {
			return err
		}

		fmt.Fprintf(app.Out, "Entry %q added.\n", name)
		app.backgroundSync()
		return nil
	},
}

func init() {
	addCmd.Flags().StringArray("tag", nil, "tag for the entry (can be repeated)")
	addCmd.Flags().Bool("totp", false, "prompt for TOTP secret")
	rootCmd.AddCommand(addCmd)
}
