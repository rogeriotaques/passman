package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec [tag] -- <command> [args...]",
	Short: "Run a command with vault secrets injected as environment variables",
	Long: `Run a command with vault secrets injected as environment variables.
All current environment variables are preserved; vault entries are added
on top. Entry names are converted to SCREAMING_SNAKE_CASE.

Use a tag before -- to inject only matching entries. Everything after --
is the command and its arguments.`,
	Example: `  passman exec -- ./deploy.sh
  passman exec aws -- terraform apply
  passman exec -- env | grep AWS`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		tag, cmdArgs := parseExecArgs(args)
		if len(cmdArgs) == 0 {
			return fmt.Errorf("no command specified (usage: passman exec [tag] -- <command>)")
		}

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
		if tag != "" {
			entries = v.ListByTag(tag)
		} else {
			entries = v.Entries
		}

		env := os.Environ()
		for _, e := range entries {
			env = append(env, fmt.Sprintf("%s=%s", toEnvVar(e.Name), e.Password))
		}

		child := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		child.Env = env
		child.Stdin = os.Stdin
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr

		return child.Run()
	},
}

func parseExecArgs(args []string) (tag string, cmdArgs []string) {
	for i, arg := range args {
		if arg == "--" {
			if i > 0 {
				tag = args[0]
			}
			cmdArgs = args[i+1:]
			return
		}
	}
	return "", args
}

func init() {
	rootCmd.AddCommand(execCmd)
}
