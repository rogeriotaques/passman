package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	gosync "sync"

	"github.com/rogerio/passman/internal/clipboard"
	"github.com/rogerio/passman/internal/crypto"
	psync "github.com/rogerio/passman/internal/sync"
	"github.com/rogerio/passman/internal/vault"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type App struct {
	VaultPath   string
	SocketPath  string
	In          io.Reader
	Out         io.Writer
	ErrOut      io.Writer
	Clipboard   clipboard.Clipboard
	KDFParams   crypto.KDFParams
	Interactive bool
	wg          gosync.WaitGroup
}

func (a *App) Wait() {
	a.wg.Wait()
}

func (a *App) backgroundSync() {
	vaultDir := filepath.Dir(a.VaultPath)

	cfg, err := psync.LoadConfig(vaultDir)
	if err != nil || !cfg.AutoSync {
		return
	}

	if !psync.IsRepo(vaultDir) {
		return
	}

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		if err := psync.Sync(vaultDir); err != nil {
			fmt.Fprintf(a.ErrOut, "Warning: auto-sync failed: %v\n", err)
		}
	}()
}

func (a *App) loadVault() (*vault.Vault, *vault.Store, []byte, error) {
	store := &vault.Store{Path: a.VaultPath, KDFParams: a.KDFParams}
	password, err := getPassword()
	if err != nil {
		return nil, nil, nil, err
	}
	v, err := store.Load(password)
	if err != nil {
		vault.ZeroBytes(password)
		return nil, nil, nil, err
	}
	return v, store, password, nil
}

func (a *App) saveAndSync(store *vault.Store, v *vault.Vault, password []byte) error {
	if err := store.Save(v, password); err != nil {
		return err
	}
	a.backgroundSync()
	return nil
}

var app = &App{
	In:     os.Stdin,
	Out:    os.Stdout,
	ErrOut: os.Stderr,
}

const banner = `
 ██████╗  █████╗ ███████╗███████╗███╗   ███╗ █████╗ ███╗   ██╗
 ██╔══██╗██╔══██╗██╔════╝██╔════╝████╗ ████║██╔══██╗████╗  ██║
 ██████╔╝███████║███████╗███████╗██╔████╔██║███████║██╔██╗ ██║
 ██╔═══╝ ██╔══██║╚════██║╚════██║██║╚██╔╝██║██╔══██║██║╚██╗██║
 ██║     ██║  ██║███████║███████║██║ ╚═╝ ██║██║  ██║██║ ╚████║
 ╚═╝     ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝
 Manage secrets from your terminal
`

var rootCmd = &cobra.Command{
	Use:               "passman",
	Short:             "Manage secrets from your terminal",
	Long:              banner,
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	SilenceUsage:      true,
	SilenceErrors:     true,
}

var vaultName string

func init() {
	rootCmd.PersistentFlags().StringVar(&vaultName, "vault", "", "vault name (default: \"default\")")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if app.VaultPath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			baseDir := filepath.Join(home, ".passman")

			name := vaultName
			if name == "" {
				name = "default"
			}
			app.VaultPath = filepath.Join(baseDir, "vaults", name, "vault.enc")
			if app.SocketPath == "" {
				app.SocketPath = filepath.Join(baseDir, "agent.sock")
			}
		}
		if app.Clipboard == nil {
			app.Clipboard = &clipboard.SystemClipboard{}
		}
		if f, ok := app.Out.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
			app.Interactive = true
		}
		return nil
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(app.ErrOut, formatError(err))
		os.Exit(1)
	}
	app.Wait()
}

func formatError(err error) string {
	msg := err.Error()
	if len(msg) == 0 {
		return msg
	}
	return strings.ToUpper(msg[:1]) + msg[1:]
}
