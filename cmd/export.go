package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:                "export [secret-name] [--eval] [-- command...]",
	Short:              "Export secrets as environment variables",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, rawArgs []string) error {
		secretName, evalMode, subCmd := parseExportArgs(rawArgs)

		// Re-parse known flags that were consumed above
		if err := cmd.Root().PersistentFlags().Parse(extractFlags(rawArgs)); err != nil {
			return err
		}

		v, _, password, err := app.loadVault()
		if err != nil {
			return err
		}
		defer vault.ZeroBytes(password)

		var entries []vault.Entry
		if secretName != "" {
			entries = v.Search(secretName)
		} else {
			entries = v.Search("")
		}

		if evalMode {
			return exportEval(entries, subCmd)
		}

		for _, e := range entries {
			envName := toEnvVar(e.Name)
			if envName == "" {
				continue
			}
			fmt.Fprintf(app.Out, "%s='%s'\n", envName, shellEscape(e.Value))
		}
		return nil
	},
}

func parseExportArgs(args []string) (secretName string, evalMode bool, subCmd []string) {
	dashIdx := -1
	for i, a := range args {
		if a == "--" {
			dashIdx = i
			break
		}
	}

	var beforeDash []string
	if dashIdx >= 0 {
		beforeDash = args[:dashIdx]
		subCmd = args[dashIdx+1:]
	} else {
		beforeDash = args
	}

	for _, a := range beforeDash {
		switch {
		case a == "--eval":
			evalMode = true
		case a == "--help" || a == "-h":
			continue
		case strings.HasPrefix(a, "--vault") || strings.HasPrefix(a, "-"):
			continue
		default:
			if secretName == "" {
				secretName = a
			}
		}
	}
	return
}

func extractFlags(args []string) []string {
	var flags []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			break
		}
		if args[i] == "--vault" && i+1 < len(args) {
			flags = append(flags, args[i], args[i+1])
			i++
		} else if strings.HasPrefix(args[i], "--vault=") {
			flags = append(flags, args[i])
		}
	}
	return flags
}

func exportEval(entries []vault.Entry, command []string) error {
	injected := make([]string, 0, len(entries))
	for _, e := range entries {
		envName := toEnvVar(e.Name)
		if envName == "" {
			continue
		}
		injected = append(injected, envName+"="+e.Value)
	}

	env := os.Environ()
	env = append(env, injected...)

	if len(command) == 0 {
		for _, kv := range injected {
			fmt.Fprintf(app.Out, "export %s\n", shellQuoteKV(kv))
		}
		return nil
	}

	child := exec.Command(command[0], command[1:]...)
	child.Env = env
	child.Stdin = os.Stdin
	child.Stdout = app.Out
	child.Stderr = app.ErrOut

	if err := child.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}

func argsAfterDash(args []string) []string {
	for i, a := range args {
		if a == "--" {
			return args[i+1:]
		}
	}
	return nil
}

var invalidEnvChars = regexp.MustCompile(`[^A-Z0-9_]`)

func toEnvVar(name string) string {
	if name == "" {
		return ""
	}
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

func shellQuoteKV(kv string) string {
	idx := strings.IndexByte(kv, '=')
	if idx < 0 {
		return kv
	}
	return kv[:idx+1] + "'" + shellEscape(kv[idx+1:]) + "'"
}

func init() {
	rootCmd.AddCommand(exportCmd)
}
