package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env [tag]",
	Short: "Output vault entries as export statements",
	Long: `Print vault entries as shell export statements (export KEY='value').
Entry names are converted to SCREAMING_SNAKE_CASE environment variable
names. Optionally filter by tag.

Designed for eval in a shell: eval "$(passman env)"`,
	Example: `  passman env
  passman env aws
  eval "$(passman env production)"`,
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
			envVar := toEnvVar(e.Name)
			fmt.Fprintf(app.Out, "export %s='%s'\n", envVar, shellEscape(e.Password))
		}
		return nil
	},
}

var invalidEnvChars = regexp.MustCompile(`[^A-Z0-9_]`)

func toEnvVar(name string) string {
	s := strings.ToUpper(name)
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	s = invalidEnvChars.ReplaceAllString(s, "_")
	if len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		s = "_" + s
	}
	return s
}

func shellEscape(s string) string {
	return strings.ReplaceAll(s, "'", "'\"'\"'")
}

func init() {
	rootCmd.AddCommand(envCmd)
}
