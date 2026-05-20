package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/rogerio/passman/internal/selector"
	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
)

const listDefaultLimit = 20

var listCmd = &cobra.Command{
	Use:   "list [query...]",
	Short: "List and search vault entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		v, _, password, err := app.loadVault()
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(password)

		showAll, _ := cmd.Flags().GetBool("all")
		query := strings.Join(args, " ")
		entries := v.Search(query)

		if len(entries) == 0 {
			if query != "" {
				fmt.Fprintln(app.Out, "No matching entries.")
			} else {
				fmt.Fprintln(app.Out, "Vault is empty.")
			}
			return nil
		}

		if app.Interactive {
			return listInteractive(v, entries)
		}
		return listPlain(entries, showAll, query)
	},
}

func listPlain(entries []vault.Entry, showAll bool, query string) error {
	limit := len(entries)
	truncated := false
	if !showAll && query == "" && limit > listDefaultLimit {
		limit = listDefaultLimit
		truncated = true
	}

	for i := 0; i < limit; i++ {
		fmt.Fprintln(app.Out, entries[i].Name)
	}
	if truncated {
		fmt.Fprintf(app.Out, "... and %d more entries (use --all to show all)\n", len(entries)-listDefaultLimit)
	}
	return nil
}

func listInteractive(v *vault.Vault, entries []vault.Entry) error {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}

	inFile, ok := app.In.(*os.File)
	if !ok {
		return listPlain(entries, true, "")
	}

	idx, err := selector.Run(selector.Options{
		Label: "Select an entry:",
		Items: names,
		Size:  15,
	}, int(inFile.Fd()), app.Out)

	if err != nil || idx < 0 {
		return nil
	}

	return copyToClipboard(entries[idx].Value)
}

func init() {
	listCmd.Flags().BoolP("all", "a", false, "show all entries without truncation")
	rootCmd.AddCommand(listCmd)
}
