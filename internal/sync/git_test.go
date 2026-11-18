package sync

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}
	// Keep the developer's global git config (signing, default branch, hooks)
	// out of the tests.
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

// bareRemote creates an empty bare repository to stand in for a hosted remote
// (GitHub-style: default branch main, no commits).
func bareRemote(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "remote.git")
	if out, err := exec.Command("git", "init", "--bare", "-b", "main", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
	return dir
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func setupGitDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "vault.enc"), []byte("encrypted"), 0600)
	return dir
}

func TestInit_CreatesGitRepo(t *testing.T) {
	requireGit(t)
	dir := setupGitDir(t)

	if err := Init(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !IsRepo(dir) {
		t.Error("expected directory to be a git repo")
	}
}

func TestInit_AlreadyRepo_NoError(t *testing.T) {
	requireGit(t)
	dir := setupGitDir(t)
	_ = Init(dir)

	if err := Init(dir); err != nil {
		t.Fatalf("expected no error for re-init, got: %v", err)
	}
}

func TestIsRepo_False(t *testing.T) {
	dir := t.TempDir()
	if IsRepo(dir) {
		t.Error("expected false for non-repo dir")
	}
}

func TestSetRemote(t *testing.T) {
	requireGit(t)
	dir := setupGitDir(t)
	_ = Init(dir)

	err := SetRemote(dir, "https://github.com/test/vault.git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !HasRemote(dir) {
		t.Error("expected remote to be set")
	}
}

func TestSetRemote_UpdatesExisting(t *testing.T) {
	requireGit(t)
	dir := setupGitDir(t)
	_ = Init(dir)
	_ = SetRemote(dir, "https://github.com/test/old.git")

	err := SetRemote(dir, "https://github.com/test/new.git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHasRemote_False(t *testing.T) {
	requireGit(t)
	dir := setupGitDir(t)
	_ = Init(dir)

	if HasRemote(dir) {
		t.Error("expected no remote before SetRemote")
	}
}

func TestSync_CommitsChanges(t *testing.T) {
	requireGit(t)
	dir := setupGitDir(t)
	_ = Init(dir)

	err := Sync(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out, _ := exec.Command("git", "-C", dir, "log", "--oneline").Output()
	if len(out) == 0 {
		t.Error("expected at least one commit after sync")
	}
}

func TestSync_NothingToCommit_NoError(t *testing.T) {
	requireGit(t)
	dir := setupGitDir(t)
	_ = Init(dir)
	_ = Sync(dir)

	err := Sync(dir)
	if err != nil {
		t.Fatalf("expected no error for nothing-to-commit, got: %v", err)
	}
}

func TestSync_NotARepo_Error(t *testing.T) {
	dir := t.TempDir()
	err := Sync(dir)
	if err == nil {
		t.Error("expected error for non-repo directory")
	}
}

func TestInit_UsesMainBranchRegardlessOfGlobalDefault(t *testing.T) {
	requireGit(t)
	_ = os.WriteFile(os.Getenv("GIT_CONFIG_GLOBAL"), []byte("[init]\n\tdefaultBranch = master\n"), 0600)
	dir := setupGitDir(t)

	if err := Init(dir); err != nil {
		t.Fatal(err)
	}
	if branch := gitOutput(t, dir, "symbolic-ref", "--short", "HEAD"); branch != "main" {
		t.Errorf("branch %q, want main", branch)
	}
}

func TestInit_IgnoresLockAndTempFiles(t *testing.T) {
	requireGit(t)
	dir := setupGitDir(t)
	_ = os.WriteFile(filepath.Join(dir, "vault.enc.lock"), nil, 0600)
	_ = os.WriteFile(filepath.Join(dir, "vault.enc.tmp"), []byte("partial"), 0600)
	_ = Init(dir)

	if err := Sync(dir); err != nil {
		t.Fatal(err)
	}
	files := gitOutput(t, dir, "ls-files")
	if strings.Contains(files, ".lock") || strings.Contains(files, ".tmp") {
		t.Errorf("lock/temp files must not be committed, got:\n%s", files)
	}
	if !strings.Contains(files, "vault.enc") || !strings.Contains(files, ".gitignore") {
		t.Errorf("expected vault.enc and .gitignore to be tracked, got:\n%s", files)
	}
}

func TestSync_CommitsAreNotSigned(t *testing.T) {
	requireGit(t)
	// A signing setup that would fail loudly if honoured.
	_ = os.WriteFile(os.Getenv("GIT_CONFIG_GLOBAL"), []byte("[commit]\n\tgpgsign = true\n[gpg]\n\tprogram = /nonexistent/gpg\n"), 0600)
	dir := setupGitDir(t)
	_ = Init(dir)

	if err := Sync(dir); err != nil {
		t.Fatalf("sync must not depend on the user's signing setup: %v", err)
	}
}

func TestSync_EmptyRemote_PushesFirstCommit(t *testing.T) {
	requireGit(t)
	dir := setupGitDir(t)
	_ = Init(dir)
	remote := bareRemote(t)
	if err := SetRemote(dir, remote); err != nil {
		t.Fatal(err)
	}

	if err := Sync(dir); err != nil {
		t.Fatalf("sync to an empty remote: %v", err)
	}
	if n := gitOutput(t, remote, "rev-list", "--count", "main"); n != "1" {
		t.Errorf("remote has %s commits, want 1", n)
	}
}

func TestSync_PullsRemoteChangesBeforePushing(t *testing.T) {
	requireGit(t)
	remote := bareRemote(t)

	first := setupGitDir(t)
	_ = Init(first)
	_ = SetRemote(first, remote)
	if err := Sync(first); err != nil {
		t.Fatal(err)
	}

	second := t.TempDir()
	if out, err := exec.Command("git", "clone", "-q", remote, second).CombinedOutput(); err != nil {
		t.Fatalf("clone: %v\n%s", err, out)
	}
	_ = os.WriteFile(filepath.Join(second, "config.json"), []byte("{}"), 0600)
	if err := Sync(second); err != nil {
		t.Fatalf("sync from second clone: %v", err)
	}

	_ = os.WriteFile(filepath.Join(first, "vault.enc"), []byte("re-encrypted"), 0600)
	if err := Sync(first); err != nil {
		t.Fatalf("sync with remote ahead: %v", err)
	}
	if n := gitOutput(t, remote, "rev-list", "--count", "main"); n != "3" {
		t.Errorf("remote has %s commits, want 3", n)
	}
	if _, err := os.Stat(filepath.Join(first, "config.json")); err != nil {
		t.Error("expected the second clone's file to be pulled into the first")
	}
}

func TestSync_UnreachableRemote_ReportsError(t *testing.T) {
	requireGit(t)
	dir := setupGitDir(t)
	_ = Init(dir)
	_ = SetRemote(dir, filepath.Join(t.TempDir(), "missing.git"))

	if err := Sync(dir); err == nil {
		t.Error("expected an error when the remote cannot be reached")
	}
}
