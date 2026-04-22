package cmd

import (
	"github.com/rogerio/passman/internal/agent"
	"github.com/spf13/cobra"
)

var agentSocketPath string

var agentCmd = &cobra.Command{
	Use:    "_agent",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		sock := agentSocketPath
		if sock == "" {
			sock = app.SocketPath
		}
		srv := agent.NewServer(sock)
		return srv.Start()
	},
}

func init() {
	agentCmd.Flags().StringVar(&agentSocketPath, "socket", "", "socket path")
	rootCmd.AddCommand(agentCmd)
}
