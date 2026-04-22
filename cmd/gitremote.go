package cmd

import (
	"fmt"
	"path/filepath"

	psync "github.com/rogerio/passman/internal/sync"
	"github.com/spf13/cobra"
)

var gitCmd_ = &cobra.Command{
	Use:   "git",
	Short: "Git-related vault commands",
}

var gitRemoteCmd = &cobra.Command{
	Use:   "remote <url>",
	Short: "Set or change the remote repository URL",
	Long: `Set or change the git remote origin URL for the vault repository.
The vault must have been initialized with --git.`,
	Example: `  passman git remote git@github.com:user/vault.git`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]
		vaultDir := filepath.Dir(app.VaultPath)

		if !psync.IsRepo(vaultDir) {
			return fmt.Errorf("vault is not a git repository (use 'passman init --git' first)")
		}

		if err := psync.SetRemote(vaultDir, url); err != nil {
			return fmt.Errorf("set remote: %w", err)
		}

		fmt.Fprintf(app.Out, "Remote set to %s\n", url)
		return nil
	},
}

func init() {
	gitCmd_.AddCommand(gitRemoteCmd)
	rootCmd.AddCommand(gitCmd_)
}
