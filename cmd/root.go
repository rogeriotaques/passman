package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	gosync "sync"

	"github.com/rogerio/passman/internal/clipboard"
	"github.com/rogerio/passman/internal/crypto"
	psync "github.com/rogerio/passman/internal/sync"
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
 Manage passwords, secrets, and 2FA from your terminal
`

var rootCmd = &cobra.Command{
	Use:   "passman",
	Short: "Manage passwords, secrets, and 2FA from your terminal",
	Long:  banner,
}

var vaultName string

func init() {
	rootCmd.PersistentFlags().StringVar(&app.VaultPath, "vault-path", "", "path to vault file")
	rootCmd.PersistentFlags().StringVar(&vaultName, "vault", "", "vault name (default: \"default\")")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if app.VaultPath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			baseDir := filepath.Join(home, ".passman")

			migrateOldVault(baseDir)

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

func migrateOldVault(baseDir string) {
	oldPath := filepath.Join(baseDir, "vault.enc")
	newDir := filepath.Join(baseDir, "vaults", "default")
	newPath := filepath.Join(newDir, "vault.enc")

	if _, err := os.Stat(oldPath); err != nil {
		return
	}
	if _, err := os.Stat(newPath); err == nil {
		return
	}

	os.MkdirAll(newDir, 0700)
	os.Rename(oldPath, newPath)

	oldConfig := filepath.Join(baseDir, "config.json")
	newConfig := filepath.Join(newDir, "config.json")
	if _, err := os.Stat(oldConfig); err == nil {
		os.Rename(oldConfig, newConfig)
	}

	oldGit := filepath.Join(baseDir, ".git")
	newGit := filepath.Join(newDir, ".git")
	if _, err := os.Stat(oldGit); err == nil {
		os.Rename(oldGit, newGit)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
	app.Wait()
}
