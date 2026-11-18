package cmd

import (
	"fmt"

	"github.com/rogerio/passman/internal/agent"
	"github.com/spf13/cobra"
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Clear cached master passwords and SSH keys from the agent",
	Long: `Immediately clear all cached master passwords and loaded SSH keys from the
background agent. After locking, the next vault operation will prompt for the
password again and ssh will find no keys until a vault is unlocked.`,
	Example: `  passman lock`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if app.SocketPath == "" {
			return fmt.Errorf("agent not configured")
		}
		client := agent.NewClient(app.SocketPath)
		if !client.Ping() {
			fmt.Fprintln(app.Out, "Agent is not running.")
			return nil
		}
		if err := client.Lock(); err != nil {
			return err
		}
		fmt.Fprintln(app.Out, "Agent cache cleared (passwords and SSH keys).")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lockCmd)
}
