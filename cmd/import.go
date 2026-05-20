package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rogerio/passman/internal/importer"
	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
)

const maxImportFileSize = 10 * 1024 * 1024 // 10 MB

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import secrets from a CSV or .env file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		replace, _ := cmd.Flags().GetBool("replace")

		info, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}
		if info.Size() > maxImportFileSize {
			return fmt.Errorf("file too large (max %d MB)", maxImportFileSize/(1024*1024))
		}

		ext := strings.ToLower(filepath.Ext(filePath))
		var parse func(*os.File) ([]vault.Entry, error)
		switch ext {
		case ".env":
			parse = func(f *os.File) ([]vault.Entry, error) { return importer.ParseEnv(f) }
		case ".csv":
			parse = func(f *os.File) ([]vault.Entry, error) { return importer.ParseCSV(f) }
		default:
			return fmt.Errorf("unsupported file format %q (use .csv or .env)", ext)
		}

		f, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}
		defer f.Close()

		entries, err := parse(f)
		if err != nil {
			return fmt.Errorf("parse file: %w", err)
		}

		v, store, password, err := app.loadVault()
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(password)

		added, skipped, replaced := 0, 0, 0
		for _, entry := range entries {
			if replace {
				if _, err := v.Get(entry.Name); err == nil {
					v.Upsert(entry)
					replaced++
					continue
				}
			}
			if err := v.Add(entry); err != nil {
				skipped++
				continue
			}
			added++
		}

		if err := app.saveAndSync(store, v, password); err != nil {
			return err
		}

		parts := []string{fmt.Sprintf("Imported %d entries", added)}
		if replaced > 0 {
			parts = append(parts, fmt.Sprintf("%d replaced", replaced))
		}
		if skipped > 0 {
			parts = append(parts, fmt.Sprintf("%d skipped", skipped))
		}
		fmt.Fprintf(app.Out, "%s.\n", strings.Join(parts, ", "))
		return nil
	},
}

func init() {
	importCmd.Flags().Bool("replace", false, "replace existing entries on name collision")
	rootCmd.AddCommand(importCmd)
}
