package cmd

import (
	"fmt"

	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
)

var sshListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the SSH keys stored in the vault",
	Long: `Show the name, type, and fingerprint of every private key in the vault.
Unlocking the vault also loads these keys into the running agent.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		v, _, password, err := app.loadVault()
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(password)

		keys := sshKeyEntries(v)
		if len(keys) == 0 {
			fmt.Fprintln(app.Out, "No SSH keys in the vault. Import one with: passman ssh add <key-file>")
			return nil
		}
		for _, k := range keys {
			fmt.Fprintln(app.Out, describeKey(k.Name, k.signer.PublicKey()))
		}
		return nil
	},
}

func init() {
	sshCmd.AddCommand(sshListCmd)
}
