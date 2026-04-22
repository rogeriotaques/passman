package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
)

const listDefaultLimit = 20

var listCmd = &cobra.Command{
	Use:   "list [query...]",
	Short: "List and search vault entries",
	Long: `List credentials stored in the vault. Without arguments, shows the
first 20 entries (use --all to show everything).

With arguments, performs a multi-token search across entry names,
usernames, notes, and tags. All tokens must match (case-insensitive)
for an entry to appear.

Use --tag to filter by tag. Search and --tag can be combined.

In a terminal, results are shown as an interactive selector — use
arrow keys to navigate and Enter to retrieve the selected entry.
When piped, output is plain text (one name per line).`,
	Example: `  passman list
  passman list --all
  passman list github
  passman list alice@ github
  passman list --tag prod`,
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

		tag, _ := cmd.Flags().GetString("tag")
		showAll, _ := cmd.Flags().GetBool("all")

		entries := v.Entries
		if tag != "" {
			entries = v.ListByTag(tag)
		}

		query := strings.Join(args, " ")
		filtered := (&vault.Vault{Entries: entries}).Search(query)

		if len(filtered) == 0 {
			if query != "" || tag != "" {
				fmt.Fprintln(app.Out, "No matching entries.")
			} else {
				fmt.Fprintln(app.Out, "Vault is empty.")
			}
			return nil
		}

		if app.Interactive {
			return listInteractive(v, filtered)
		}
		return listPlain(filtered, showAll, query)
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
	if len(entries) == 1 {
		entry, err := v.Get(entries[0].Name)
		if err != nil {
			return err
		}
		return copyAndShowEntry(entry)
	}

	names := make([]string, len(entries))
	for i, e := range entries {
		label := e.Name
		if e.Username != "" {
			label += "  (" + e.Username + ")"
		}
		names[i] = label
	}

	prompt := promptui.Select{
		Label: "Select an entry",
		Items: names,
		Size:  15,
		Stdin: io.NopCloser(app.In),
	}

	idx, _, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt || err == promptui.ErrEOF {
			return nil
		}
		return err
	}

	entry, err := v.Get(entries[idx].Name)
	if err != nil {
		return err
	}
	return copyAndShowEntry(entry)
}

func init() {
	listCmd.Flags().BoolP("all", "a", false, "show all entries without truncation")
	listCmd.Flags().String("tag", "", "filter by tag")
	rootCmd.AddCommand(listCmd)
}
