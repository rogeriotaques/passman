package cmd

import (
	"fmt"

	"github.com/rogerio/passman/internal/generate"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a random strong password",
	Long: `Generate a cryptographically random password. By default it includes
uppercase, lowercase, digits, and symbols. The generated password is
guaranteed to contain at least one character from each enabled class.`,
	Example: `  passman generate
  passman generate --length 32
  passman generate --no-symbols`,
	RunE: func(cmd *cobra.Command, args []string) error {
		length, _ := cmd.Flags().GetInt("length")
		noSymbols, _ := cmd.Flags().GetBool("no-symbols")

		opts := generate.DefaultOptions()
		opts.Length = length
		if noSymbols {
			opts.IncludeSymbols = false
		}

		pw, err := generate.Generate(opts)
		if err != nil {
			return err
		}

		fmt.Fprintln(app.Out, pw)
		return nil
	},
}

func init() {
	generateCmd.Flags().Int("length", 20, "password length")
	generateCmd.Flags().Bool("no-symbols", false, "exclude symbols")
	rootCmd.AddCommand(generateCmd)
}
