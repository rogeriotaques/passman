package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	psync "github.com/rogerio/passman/internal/sync"
	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
)

const minPasswordLen = 8

var initCmd = &cobra.Command{
	Use:   "init [vault-name]",
	Short: "Create a new vault",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 && vaultName == "" {
			vaultName = args[0]
			home, _ := cmd.Root().PersistentFlags().GetString("vault")
			if home == "" {
				// re-resolve vault path with provided name
				resolveVaultPath(args[0])
			}
		}

		store := &vault.Store{Path: app.VaultPath, KDFParams: app.KDFParams}

		if store.Exists() {
			return fmt.Errorf("vault already exists at %s", app.VaultPath)
		}

		password, err := readPassword("Enter master password: ")
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(password)

		if len(password) > 0 {
			if len(password) < minPasswordLen {
				return fmt.Errorf("master password must be at least %d characters", minPasswordLen)
			}

			confirm, err := readPassword("Confirm master password: ")
			if err != nil {
				return err
			}
			defer vault.ZeroBytes(confirm)

			if string(password) != string(confirm) {
				return fmt.Errorf("passwords do not match")
			}
		}

		if err := store.Init(password); err != nil {
			return err
		}

		if len(password) == 0 {
			vaultDir := filepath.Dir(app.VaultPath)
			cfg, _ := psync.LoadConfig(vaultDir)
			cfg.NoPassword = true
			_ = psync.SaveConfig(vaultDir, cfg)
		}

		cacheInAgent(password)
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

func resolveVaultPath(name string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	app.VaultPath = filepath.Join(home, ".passman", "vaults", name, "vault.enc")
}

func init() {
	initCmd.Flags().Bool("git", false, "initialize git repository for vault sync")
	rootCmd.AddCommand(initCmd)
}
