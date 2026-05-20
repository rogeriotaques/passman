package cmd

import (
	"fmt"
	"path/filepath"

	psync "github.com/rogerio/passman/internal/sync"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config [[key] value]",
	Short: "View or update vault configuration",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultDir := filepath.Dir(app.VaultPath)

		cfg, err := psync.LoadConfig(vaultDir)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		switch len(args) {
		case 0:
			for _, key := range psync.ConfigKeys() {
				val, _ := cfg.Get(key)
				fmt.Fprintf(app.Out, "%s = %s\n", key, val)
			}
		case 1:
			val, err := cfg.Get(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(app.Out, val)
		case 2:
			if err := cfg.Set(args[0], args[1]); err != nil {
				return err
			}
			if err := psync.SaveConfig(vaultDir, cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			if args[0] == "git" {
				if psync.IsRepo(vaultDir) {
					if err := psync.SetRemote(vaultDir, args[1]); err != nil {
						return fmt.Errorf("set git remote: %w", err)
					}
				}
			}

			fmt.Fprintf(app.Out, "%s = %s\n", args[0], args[1])
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
