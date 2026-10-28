package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFetchLatestVersion(t *testing.T) {
	repo := createTestRepo(t, "0.2.0")
	got, err := fetchLatestVersion("file://" + repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "0.2.0" {
		t.Errorf("expected 0.2.0, got %q", got)
	}
}

func TestFetchLatestVersion_MissingFile(t *testing.T) {
	repo := createTestRepo(t, "")
	_, err := fetchLatestVersion("file://" + repo)
	if err == nil {
		t.Fatal("expected error when the repository has no VERSION file")
	}
}

func TestFetchLatestVersion_Invalid(t *testing.T) {
	repo := createTestRepo(t, "not-semver")
	_, err := fetchLatestVersion("file://" + repo)
	if err == nil {
		t.Fatal("expected error for a VERSION file that is not semver")
	}
}

func TestUpgrade_AlreadyUpToDate(t *testing.T) {
	repo := createTestRepo(t, "0.2.0")

	oldRepo := repoURL
	repoURL = "file://" + repo
	defer func() { repoURL = oldRepo }()

	oldVersion := version
	version = "0.2.0"
	defer func() { version = oldVersion }()

	out, errOut, _ := setupTestApp(t, "")
	err := runUpgrade(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, errOut.String())
	}
	output := out.String()
	if !strings.Contains(output, "Already up to date") {
		t.Errorf("expected 'Already up to date' in output, got: %s", output)
	}
}

func TestUpgrade_MissingGit(t *testing.T) {
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "/nonexistent")
	defer os.Setenv("PATH", oldPath)

	out, _, _ := setupTestApp(t, "")
	err := runUpgrade(nil, nil)
	if err == nil {
		t.Fatal("expected error when git is missing")
	}
	if !strings.Contains(err.Error(), "git is required") {
		t.Errorf("expected 'git is required' error, got: %v", err)
	}
	if !strings.Contains(out.String(), "Current version") {
		t.Errorf("expected current version printed, got: %s", out.String())
	}
}

func TestUpgrade_MissingGo(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not found")
	}
	gitDir := filepath.Dir(gitPath)

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", gitDir)
	defer os.Setenv("PATH", oldPath)

	err = runUpgrade(nil, nil)
	if err == nil {
		t.Fatal("expected error when go is missing")
	}
	if !strings.Contains(err.Error(), "go is required") {
		t.Errorf("expected 'go is required' error, got: %v", err)
	}
}

func TestUpgrade_Success(t *testing.T) {
	repo := createTestRepo(t, "0.2.0")

	oldRepo := repoURL
	repoURL = "file://" + repo
	defer func() { repoURL = oldRepo }()

	oldVersion := version
	version = "0.1.0"
	defer func() { version = oldVersion }()

	// Create a fake destination binary
	dest := filepath.Join(t.TempDir(), "passman")
	if err := os.WriteFile(dest, []byte("old binary"), 0755); err != nil {
		t.Fatal(err)
	}
	oldUpgradeDest := upgradeDestPath
	upgradeDestPath = dest
	defer func() { upgradeDestPath = oldUpgradeDest }()

	out, errOut, _ := setupTestApp(t, "")
	err := runUpgrade(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, errOut.String())
	}

	output := out.String()
	if !strings.Contains(output, "Current version 0.1.0") {
		t.Errorf("expected current version in output, got: %s", output)
	}

	// The new version comes from the cloned repository's VERSION file
	if !strings.Contains(output, "Updating to 0.2.0") {
		t.Errorf("expected 'Updating to 0.2.0' message, got: %s", output)
	}
	if !strings.Contains(output, "Upgraded to version 0.2.0") {
		t.Errorf("expected 'Upgraded to version 0.2.0' message, got: %s", output)
	}

	// Verify the binary was replaced
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "old binary" {
		t.Error("expected binary to be replaced")
	}
}

func createTestRepo(t *testing.T, version string) string {
	t.Helper()
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}
	run := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
		}
	}
	run("git", "init")
	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "Test")

	// Ensure branch is named master so clone --branch master works
	currentBranch, err := exec.Command("git", "-C", repo, "branch", "--show-current").Output()
	if err == nil && strings.TrimSpace(string(currentBranch)) != "master" {
		run("git", "branch", "-m", "master")
	}

	// Create a simple Go module so `go build` works
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module test\n\ngo 1.25\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "main.go"), []byte("package main\nfunc main(){}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if version != "" {
		if err := os.WriteFile(filepath.Join(repo, "VERSION"), []byte(version+"\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	run("git", "add", ".")
	run("git", "commit", "-m", "init")
	return repo
}
