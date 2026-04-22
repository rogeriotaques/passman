package sync

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}
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
