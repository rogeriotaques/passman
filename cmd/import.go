package cmd

import (
	"fmt"
	"os"

	"github.com/rogerio/passman/internal/importer"
	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import credentials from external sources",
	Long:  `Import credentials into the vault from .env files or CSV exports (1Password, Bitwarden). Duplicate names are skipped.`,
}

var importEnvCmd = &cobra.Command{
	Use:   "env <file>",
	Short: "Import from a .env file",
	Long: `Import entries from a .env file. Each KEY=VALUE line becomes a vault
entry where the key is the name and the value is the password.`,
	Example: `  passman import env .env
  passman import env /path/to/secrets.env`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runImport(args[0], func(f *os.File) ([]vault.Entry, error) {
			return importer.ParseEnv(f)
		})
	},
}

var importCSVCmd = &cobra.Command{
	Use:   "csv <file>",
	Short: "Import from a 1Password or Bitwarden CSV export",
	Long: `Import entries from a CSV file exported by 1Password or Bitwarden.
Expected columns: Name/Title, Username, Password, Notes.`,
	Example: `  passman import csv ~/Downloads/1password-export.csv
  passman import csv bitwarden.csv`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runImport(args[0], func(f *os.File) ([]vault.Entry, error) {
			return importer.ParseCSV(f)
		})
	},
}

func runImport(filePath string, parse func(*os.File) ([]vault.Entry, error)) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	entries, err := parse(f)
	if err != nil {
		return fmt.Errorf("parse file: %w", err)
	}

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

	added, skipped := 0, 0
	for _, entry := range entries {
		if err := v.Add(entry); err != nil {
			skipped++
			continue
		}
		added++
	}

	if err := store.Save(v, password); err != nil {
		return err
	}

	fmt.Fprintf(app.Out, "Imported %d entries (%d skipped).\n", added, skipped)
	app.backgroundSync()
	return nil
}

func init() {
	importCmd.AddCommand(importEnvCmd)
	importCmd.AddCommand(importCSVCmd)
	rootCmd.AddCommand(importCmd)
}
