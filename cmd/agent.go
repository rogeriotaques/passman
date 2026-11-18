package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/rogerio/passman/internal/agent"
	"github.com/rogerio/passman/internal/sshagent"
	"github.com/spf13/cobra"
)

var (
	agentSocketPath    string
	agentSSHSocketPath string
	agentSSHVault      string
)

var agentCmd = &cobra.Command{
	Use:    "_agent",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		sock := agentSocketPath
		if sock == "" {
			sock = app.SocketPath
		}
		srv := agent.NewServer(sock)
		if agentSSHSocketPath != "" {
			ui := app.Prompter
			srv.EnableSSH(agent.SSHConfig{
				SocketPath: agentSSHSocketPath,
				Confirm: func(req sshagent.SignRequest) bool {
					ok, err := ui.Confirm(req.Message())
					return err == nil && ok
				},
				UnlockVault:    agentSSHVault,
				PromptPassword: ui.Secret,
			})
		}
		stopOnSignal(srv)
		return srv.Start()
	},
}

// stopOnSignal makes termination zero cached secrets and remove the sockets.
func stopOnSignal(srv *agent.Server) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		<-sigs
		srv.Stop()
	}()
}

func init() {
	agentCmd.Flags().StringVar(&agentSocketPath, "socket", "", "socket path")
	agentCmd.Flags().StringVar(&agentSSHSocketPath, "ssh-socket", "", "serve the SSH agent protocol on this socket")
	agentCmd.Flags().StringVar(&agentSSHVault, "ssh-vault", "", "vault to unlock on demand for SSH clients")
	rootCmd.AddCommand(agentCmd)
}
