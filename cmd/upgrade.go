package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var repoURL = "https://github.com/rogeriotaques/passman.git"

// override in tests to avoid replacing the test binary
var upgradeDestPath = ""

var upgradeCmd = &cobra.Command{
	Use:     "upgrade",
	Short:   "Upgrade passman to the latest version",
	Long:    `Clones the repository, compiles the latest version, and replaces the current binary.`,
	Example: `  passman upgrade`,
	RunE:    runUpgrade,
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	fmt.Fprintf(app.Out, "• Current version %s\n", version)

	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is required but not found in PATH")
	}
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("go is required but not found in PATH")
	}

	fmt.Fprintln(app.Out, "• Checking new version available...")

	commitHash, err := fetchLatestCommitHash(repoURL)
	if err != nil {
		return fmt.Errorf("failed to check latest version: %w", err)
	}
	if commitHash == "" {
		return fmt.Errorf("could not determine latest version")
	}

	if version == commitHash {
		fmt.Fprintln(app.Out, "• Already up to date.")
		return nil
	}

	fmt.Fprintf(app.Out, "• Updating to %s...\n", commitHash)

	tempDir, err := os.MkdirTemp("", "passman-upgrade-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	cloneDir := filepath.Join(tempDir, "passman")

	c := exec.Command("git", "clone", "--depth", "1", "--branch", "master", repoURL, cloneDir)
	c.Stdout = app.Out
	c.Stderr = app.ErrOut
	if err := c.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	newBinary := filepath.Join(tempDir, "passman-new")
	ldflags := fmt.Sprintf("-X github.com/rogerio/passman/cmd.version=%s", commitHash)
	build := exec.Command("go", "build", "-ldflags", ldflags, "-o", newBinary, ".")
	build.Dir = cloneDir
	build.Stdout = app.Out
	build.Stderr = app.ErrOut
	build.Env = append(os.Environ(), "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH)
	if err := build.Run(); err != nil {
		return fmt.Errorf("failed to build new version: %w", err)
	}

	dest := upgradeDestPath
	if dest == "" {
		dest, err = os.Executable()
		if err != nil {
			return fmt.Errorf("failed to locate current binary: %w", err)
		}
	}

	if err := replaceBinary(newBinary, dest); err != nil {
		return fmt.Errorf("failed to replace current binary: %w", err)
	}

	fmt.Fprintf(app.Out, "• Done. Upgraded to version %s\n", commitHash)
	return nil
}

func fetchLatestCommitHash(repo string) (string, error) {
	// Shallow clone master to temp dir to read HEAD
	tmpDir, err := os.MkdirTemp("", "passman-check-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	cloneDir := filepath.Join(tmpDir, "passman")
	clone := exec.Command("git", "clone", "--depth", "1", "--branch", "master", repo, cloneDir)
	clone.Stderr = os.Stderr
	if err := clone.Run(); err != nil {
		return "", err
	}

	out, err := exec.Command("git", "-C", cloneDir, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func replaceBinary(src, dst string) error {
	// Try direct rename first (works when on same filesystem)
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// Cross-device fallback: copy to temp file in dst dir, then rename
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, ".passman-upgrade-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Chmod(tmpName, 0755); err != nil {
		return err
	}

	return os.Rename(tmpName, dst)
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}
