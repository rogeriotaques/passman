package cmd

import (
	"encoding/csv"
	"fmt"

	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export credentials to external formats",
	Long:  `Export vault credentials to .env or CSV format. Use a subcommand to choose the format.`,
}

var exportEnvCmd = &cobra.Command{
	Use:   "env [tag]",
	Short: "Export as .env format",
	Long: `Export vault entries as KEY='value' pairs (without export prefix).
Optionally filter by tag. Output is suitable for writing to a .env file.`,
	Example: `  passman export env > .env
  passman export env aws > .env.aws`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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

		var entries []vault.Entry
		if len(args) == 1 {
			entries = v.ListByTag(args[0])
		} else {
			entries = v.Entries
		}

		for _, e := range entries {
			fmt.Fprintf(app.Out, "%s='%s'\n", toEnvVar(e.Name), shellEscape(e.Password))
		}
		return nil
	},
}

var exportCSVCmd = &cobra.Command{
	Use:   "csv",
	Short: "Export as CSV",
	Long: `Export all vault entries as CSV with columns:
Name, Username, Password, Notes, Tags.`,
	Example: `  passman export csv > backup.csv`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		w := csv.NewWriter(app.Out)
		w.Write([]string{"Name", "Username", "Password", "Notes", "Tags"})

		for _, e := range v.Entries {
			tags := ""
			if len(e.Tags) > 0 {
				tags = fmt.Sprintf("%v", e.Tags)
			}
			w.Write([]string{e.Name, e.Username, e.Password, e.Notes, tags})
		}

		w.Flush()
		return w.Error()
	},
}

func init() {
	exportCmd.AddCommand(exportEnvCmd)
	exportCmd.AddCommand(exportCSVCmd)
	rootCmd.AddCommand(exportCmd)
}
