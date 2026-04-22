package cmd

import (
	"fmt"
	"path/filepath"

	psync "github.com/rogerio/passman/internal/sync"
	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a new vault",
	Long: `Create a new encrypted vault. You will be prompted for a master password
(minimum 8 characters) with confirmation.

The vault is stored at ~/.passman/vaults/<name>/vault.enc by default.
Use --vault to choose a name, or --vault-path for a custom location.

With --git, a git repository is initialized in the vault directory and
you are optionally prompted for a remote URL for sync.`,
	Example: `  passman init
  passman init --vault work
  passman init --git`,
	RunE: func(cmd *cobra.Command, args []string) error {
		store := &vault.Store{Path: app.VaultPath, KDFParams: app.KDFParams}

		if store.Exists() {
			return fmt.Errorf("vault already exists at %s", app.VaultPath)
		}

		password, err := readPassword("Enter master password: ")
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(password)

		confirm, err := readPassword("Confirm master password: ")
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(confirm)

		const minPasswordLen = 8
		if len(password) < minPasswordLen {
			return fmt.Errorf("master password must be at least %d characters", minPasswordLen)
		}

		if string(password) != string(confirm) {
			return fmt.Errorf("passwords do not match")
		}

		if err := store.Init(password); err != nil {
			return err
		}

		storePasswordInAgent(password)
		fmt.Fprintln(app.Out, "Vault created successfully.")

		useGit, _ := cmd.Flags().GetBool("git")
		if useGit {
			vaultDir := filepath.Dir(app.VaultPath)

			if err := psync.Init(vaultDir); err != nil {
				return fmt.Errorf("git init: %w", err)
			}
			fmt.Fprintln(app.Out, "Git repository initialized.")

			remote, err := readInput("Remote repository URL (leave empty to skip): ")
			if err != nil {
				return err
			}

			if remote != "" {
				if err := psync.SetRemote(vaultDir, remote); err != nil {
					return fmt.Errorf("git remote: %w", err)
				}
				fmt.Fprintf(app.Out, "Remote set to %s\n", remote)
			}

			if err := psync.Sync(vaultDir); err != nil {
				return fmt.Errorf("initial sync: %w", err)
			}
			fmt.Fprintln(app.Out, "Initial commit created.")
		}

		return nil
	},
}

func init() {
	initCmd.Flags().Bool("git", false, "initialize git repository for vault sync")
	rootCmd.AddCommand(initCmd)
}
