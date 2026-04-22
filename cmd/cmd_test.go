package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rogerio/passman/internal/agent"
	"github.com/rogerio/passman/internal/clipboard"
	"github.com/rogerio/passman/internal/crypto"
)

var testVaultPath string

func setupTestApp(t *testing.T, input string) (*bytes.Buffer, *bytes.Buffer, *clipboard.MockClipboard) {
	t.Helper()
	dir := t.TempDir()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	mock := &clipboard.MockClipboard{}

	testVaultPath = filepath.Join(dir, "vault.enc")
	app.VaultPath = testVaultPath
	app.SocketPath = ""
	app.Out = out
	app.ErrOut = errOut
	app.Clipboard = mock
	app.KDFParams = crypto.FastKDFParams()

	setInput(input)
	resetFlags()
	return out, errOut, mock
}

func setInput(input string) {
	app.In = strings.NewReader(input)
	resetInputReader()
}

func resetFlags() {
	getCmd.Flags().Set("print", "false")
	generateCmd.Flags().Set("length", "20")
	generateCmd.Flags().Set("no-symbols", "false")
	initCmd.Flags().Set("git", "false")
	syncCmd.Flags().Set("auto", "")
	addCmd.Flags().Lookup("tag").Value.(interface{ Replace([]string) error }).Replace(nil)
	addCmd.Flags().Lookup("tag").Changed = false
	addCmd.Flags().Set("totp", "false")
	listCmd.Flags().Set("all", "false")
	listCmd.Flags().Set("tag", "")
	rootCmd.PersistentFlags().Set("vault", "")
}

func runCmd(args ...string) error {
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

func TestInit_Success(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	err := runCmd("init")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "Vault created successfully") {
		t.Errorf("unexpected output: %s", out.String())
	}
}

func TestInit_PasswordTooShort(t *testing.T) {
	setupTestApp(t, "short\nshort\n")
	err := runCmd("init")
	if err == nil {
		t.Error("expected error for short password")
	}
}

func TestInit_EmptyPassword(t *testing.T) {
	setupTestApp(t, "\n\n")
	err := runCmd("init")
	if err == nil {
		t.Error("expected error for empty password")
	}
}

func TestInit_PasswordMismatch(t *testing.T) {
	setupTestApp(t, "masterpass\nwrongpass\n")
	err := runCmd("init")
	if err == nil {
		t.Error("expected error for password mismatch")
	}
}

func TestInit_AlreadyExists(t *testing.T) {
	setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nmasterpass\n")
	err := runCmd("init")
	if err == nil {
		t.Error("expected error when vault already exists")
	}
}

func TestAdd_And_List(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nmyuser\nmypass\nsome notes\n")
	out.Reset()
	err := runCmd("add", "github")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if !strings.Contains(out.String(), `"github" added`) {
		t.Errorf("unexpected add output: %s", out.String())
	}

	setInput("masterpass\n")
	out.Reset()
	err = runCmd("list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out.String(), "github") {
		t.Errorf("expected 'github' in list output: %s", out.String())
	}
}

func TestAdd_Duplicate(t *testing.T) {
	setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\npass\nnotes\n")
	_ = runCmd("add", "github")

	setInput("masterpass\nuser2\npass2\nnotes2\n")
	err := runCmd("add", "github")
	if err == nil {
		t.Error("expected error for duplicate entry")
	}
}

func TestGet_Print(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nmyuser\nsecretpass\n\n")
	_ = runCmd("add", "github")

	setInput("masterpass\n")
	out.Reset()
	err := runCmd("get", "--print", "github")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "myuser") {
		t.Errorf("expected username in output: %s", output)
	}
	if !strings.Contains(output, "secretpass") {
		t.Errorf("expected password in output: %s", output)
	}
}

func TestGet_Clipboard(t *testing.T) {
	out, _, mock := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nmyuser\nsecretpass\n\n")
	_ = runCmd("add", "github")

	setInput("masterpass\n")
	out.Reset()
	err := runCmd("get", "github")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if mock.Content != "secretpass" {
		t.Errorf("expected clipboard content 'secretpass', got %q", mock.Content)
	}
	if !strings.Contains(out.String(), "copied to clipboard") {
		t.Errorf("expected clipboard message in output: %s", out.String())
	}
}

func TestGet_NotFound(t *testing.T) {
	setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\n")
	err := runCmd("get", "nonexistent")
	if err == nil {
		t.Error("expected error for missing entry")
	}
}

func TestRm_Success(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\npass\n\n")
	_ = runCmd("add", "github")

	setInput("masterpass\n")
	out.Reset()
	err := runCmd("rm", "github")
	if err != nil {
		t.Fatalf("rm: %v", err)
	}
	if !strings.Contains(out.String(), `"github" removed`) {
		t.Errorf("unexpected rm output: %s", out.String())
	}

	setInput("masterpass\n")
	out.Reset()
	err = runCmd("list")
	if err != nil {
		t.Fatalf("list after rm: %v", err)
	}
	if !strings.Contains(out.String(), "empty") {
		t.Errorf("expected empty vault after rm: %s", out.String())
	}
}

func TestRm_NotFound(t *testing.T) {
	setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\n")
	err := runCmd("rm", "nonexistent")
	if err == nil {
		t.Error("expected error for removing nonexistent entry")
	}
}

func TestGenerate_Default(t *testing.T) {
	out, _, _ := setupTestApp(t, "")
	err := runCmd("generate")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	pw := strings.TrimSpace(out.String())
	if len(pw) != 20 {
		t.Errorf("expected 20-char password, got %d: %q", len(pw), pw)
	}
}

func TestGenerate_CustomLength(t *testing.T) {
	out, _, _ := setupTestApp(t, "")
	err := runCmd("generate", "--length", "32")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	pw := strings.TrimSpace(out.String())
	if len(pw) != 32 {
		t.Errorf("expected 32-char password, got %d: %q", len(pw), pw)
	}
}

func TestList_EmptyVault(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\n")
	out.Reset()
	err := runCmd("list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out.String(), "empty") {
		t.Errorf("expected empty message: %s", out.String())
	}
}

func TestList_WrongPassword(t *testing.T) {
	setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("wrongpassword\n")
	err := runCmd("list")
	if err == nil {
		t.Error("expected error for wrong password")
	}
}

func TestList_SearchByName(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\npass\n\n")
	_ = runCmd("add", "github-work")
	setInput("masterpass\nuser\npass\n\n")
	resetFlags()
	_ = runCmd("add", "github-personal")
	setInput("masterpass\nuser\npass\n\n")
	resetFlags()
	_ = runCmd("add", "aws-prod")

	setInput("masterpass\n")
	out.Reset()
	err := runCmd("list", "github")
	if err != nil {
		t.Fatalf("list search: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "github-work") {
		t.Errorf("expected github-work: %s", output)
	}
	if !strings.Contains(output, "github-personal") {
		t.Errorf("expected github-personal: %s", output)
	}
	if strings.Contains(output, "aws-prod") {
		t.Errorf("did not expect aws-prod: %s", output)
	}
}

func TestList_SearchMultipleTokens(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nalice@corp.com\npass\n\n")
	_ = runCmd("add", "github-work")
	setInput("masterpass\nalice@gmail.com\npass\n\n")
	resetFlags()
	_ = runCmd("add", "github-personal")

	setInput("masterpass\n")
	out.Reset()
	err := runCmd("list", "github", "corp")
	if err != nil {
		t.Fatalf("list search: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "github-work") {
		t.Errorf("expected github-work: %s", output)
	}
	if strings.Contains(output, "github-personal") {
		t.Errorf("did not expect github-personal: %s", output)
	}
}

func TestList_SearchNoMatch(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\npass\n\n")
	_ = runCmd("add", "github")

	setInput("masterpass\n")
	out.Reset()
	err := runCmd("list", "nonexistent")
	if err != nil {
		t.Fatalf("list search: %v", err)
	}
	if !strings.Contains(out.String(), "No matching entries") {
		t.Errorf("expected no matching message: %s", out.String())
	}
}

func TestList_TruncatesLongList(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	for i := 0; i < 25; i++ {
		setInput(fmt.Sprintf("masterpass\nuser\npass\n\n"))
		resetFlags()
		_ = runCmd("add", fmt.Sprintf("entry-%03d", i))
	}

	setInput("masterpass\n")
	out.Reset()
	err := runCmd("list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	output := out.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	lastLine := lines[len(lines)-1]
	if !strings.Contains(lastLine, "more") {
		t.Errorf("expected truncation message, got: %s", lastLine)
	}
}

func TestList_AllFlag(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	for i := 0; i < 25; i++ {
		setInput(fmt.Sprintf("masterpass\nuser\npass\n\n"))
		resetFlags()
		_ = runCmd("add", fmt.Sprintf("entry-%03d", i))
	}

	setInput("masterpass\n")
	out.Reset()
	resetFlags()
	err := runCmd("list", "--all")
	if err != nil {
		t.Fatalf("list --all: %v", err)
	}
	output := out.String()
	if strings.Contains(output, "more") {
		t.Errorf("--all should not truncate: %s", output)
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 25 {
		t.Errorf("expected 25 lines, got %d", len(lines))
	}
}

func TestList_TagFilter(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\npass\n\n")
	_ = runCmd("add", "aws-key", "--tag", "prod")
	setInput("masterpass\nuser\npass\n\n")
	resetFlags()
	_ = runCmd("add", "dev-key", "--tag", "dev")

	setInput("masterpass\n")
	out.Reset()
	resetFlags()
	err := runCmd("list", "--tag", "prod")
	if err != nil {
		t.Fatalf("list --tag: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "aws-key") {
		t.Errorf("expected aws-key: %s", output)
	}
	if strings.Contains(output, "dev-key") {
		t.Errorf("did not expect dev-key: %s", output)
	}
}

// --- passwd tests ---

func TestPasswd_Success(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nnewpass123\nnewpass123\n")
	out.Reset()
	err := runCmd("passwd")
	if err != nil {
		t.Fatalf("passwd: %v", err)
	}
	if !strings.Contains(out.String(), "Master password changed") {
		t.Errorf("expected success message: %s", out.String())
	}

	// Verify the new password works
	setInput("newpass123\n")
	out.Reset()
	err = runCmd("list")
	if err != nil {
		t.Fatalf("list with new password: %v", err)
	}
	if !strings.Contains(out.String(), "empty") {
		t.Errorf("expected empty vault: %s", out.String())
	}
}

func TestPasswd_OldPasswordWrong(t *testing.T) {
	setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("wrongpass\nnewpass123\nnewpass123\n")
	err := runCmd("passwd")
	if err == nil {
		t.Error("expected error for wrong current password")
	}
}

func TestPasswd_NewPasswordTooShort(t *testing.T) {
	setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nshort\nshort\n")
	err := runCmd("passwd")
	if err == nil {
		t.Error("expected error for short new password")
	}
}

func TestPasswd_NewPasswordMismatch(t *testing.T) {
	setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nnewpass123\ndifferent1\n")
	err := runCmd("passwd")
	if err == nil {
		t.Error("expected error for password mismatch")
	}
}

func TestPasswd_PreservesEntries(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\npass\n\n")
	_ = runCmd("add", "github")

	setInput("masterpass\nnewpass123\nnewpass123\n")
	resetFlags()
	err := runCmd("passwd")
	if err != nil {
		t.Fatalf("passwd: %v", err)
	}

	setInput("newpass123\n")
	out.Reset()
	err = runCmd("list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out.String(), "github") {
		t.Errorf("expected github in list after passwd: %s", out.String())
	}
}

func TestPasswd_OldPasswordNoLongerWorks(t *testing.T) {
	setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nnewpass123\nnewpass123\n")
	_ = runCmd("passwd")

	setInput("masterpass\n")
	err := runCmd("list")
	if err == nil {
		t.Error("expected error when using old password after passwd")
	}
}

// --- Interactive list tests ---

func TestList_Interactive_SingleResult_AutoSelects(t *testing.T) {
	out, _, mock := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\nsecretpass\n\n")
	_ = runCmd("add", "github")

	setInput("masterpass\n")
	out.Reset()
	resetFlags()
	app.Interactive = true
	defer func() { app.Interactive = false }()

	err := runCmd("list", "github")
	if err != nil {
		t.Fatalf("list interactive: %v", err)
	}
	if mock.Content != "secretpass" {
		t.Errorf("expected clipboard content 'secretpass', got %q", mock.Content)
	}
	if !strings.Contains(out.String(), "copied to clipboard") {
		t.Errorf("expected clipboard message: %s", out.String())
	}
}

func TestList_Interactive_NoResults_NoSelector(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\npass\n\n")
	_ = runCmd("add", "github")

	setInput("masterpass\n")
	out.Reset()
	resetFlags()
	app.Interactive = true
	defer func() { app.Interactive = false }()

	err := runCmd("list", "nonexistent")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out.String(), "No matching entries") {
		t.Errorf("expected no matching message: %s", out.String())
	}
}

func TestList_NonInteractive_MultipleResults_PlainOutput(t *testing.T) {
	out, _, mock := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\npass\n\n")
	_ = runCmd("add", "github-work")
	setInput("masterpass\nuser\npass\n\n")
	resetFlags()
	_ = runCmd("add", "github-personal")

	setInput("masterpass\n")
	out.Reset()
	resetFlags()
	app.Interactive = false

	err := runCmd("list", "github")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "github-personal") || !strings.Contains(output, "github-work") {
		t.Errorf("expected both entries in plain output: %s", output)
	}
	if mock.Content != "" {
		t.Errorf("non-interactive should not copy to clipboard, got %q", mock.Content)
	}
}

// --- Phase 1.5: Git sync tests ---

func TestInit_WithGit(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n\n")
	err := runCmd("init", "--git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "Vault created successfully") {
		t.Errorf("expected vault created message: %s", output)
	}
	if !strings.Contains(output, "Git repository initialized") {
		t.Errorf("expected git init message: %s", output)
	}
	if !strings.Contains(output, "Initial commit created") {
		t.Errorf("expected initial commit message: %s", output)
	}
}

func TestInit_WithGit_AndRemote(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\nhttps://github.com/test/vault.git\n")
	// Push will fail because the remote is fake, but remote should be configured
	_ = runCmd("init", "--git")
	output := out.String()
	if !strings.Contains(output, "Remote set to") {
		t.Errorf("expected remote set message: %s", output)
	}
}

func TestSync_ManualSync(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n\n")
	_ = runCmd("init", "--git")

	setInput("masterpass\nuser\npass\n\n")
	_ = runCmd("add", "github")
	app.Wait()

	setInput("")
	out.Reset()
	err := runCmd("sync")
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if !strings.Contains(out.String(), "Vault synced") {
		t.Errorf("expected sync message: %s", out.String())
	}
}

func TestSync_NotARepo(t *testing.T) {
	setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("")
	err := runCmd("sync")
	if err == nil {
		t.Error("expected error when vault is not a git repo")
	}
}

func TestSync_AutoOn(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n\n")
	_ = runCmd("init", "--git")

	out.Reset()
	err := runCmd("sync", "--auto", "on")
	if err != nil {
		t.Fatalf("sync --auto on: %v", err)
	}
	if !strings.Contains(out.String(), "Auto-sync on") {
		t.Errorf("expected auto-sync on message: %s", out.String())
	}
}

func TestSync_AutoOff(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n\n")
	_ = runCmd("init", "--git")

	_ = runCmd("sync", "--auto", "on")
	out.Reset()
	resetFlags()
	err := runCmd("sync", "--auto", "off")
	if err != nil {
		t.Fatalf("sync --auto off: %v", err)
	}
	if !strings.Contains(out.String(), "Auto-sync off") {
		t.Errorf("expected auto-sync off message: %s", out.String())
	}
}

func TestSync_AutoInvalid(t *testing.T) {
	setupTestApp(t, "masterpass\nmasterpass\n\n")
	_ = runCmd("init", "--git")

	err := runCmd("sync", "--auto", "maybe")
	if err == nil {
		t.Error("expected error for invalid --auto value")
	}
}

func TestGitRemote(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n\n")
	_ = runCmd("init", "--git")

	out.Reset()
	err := runCmd("git", "remote", "https://github.com/test/vault.git")
	if err != nil {
		t.Fatalf("git remote: %v", err)
	}
	if !strings.Contains(out.String(), "Remote set to") {
		t.Errorf("expected remote set message: %s", out.String())
	}
}

func TestGitRemote_NotARepo(t *testing.T) {
	setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	err := runCmd("git", "remote", "https://github.com/test/vault.git")
	if err == nil {
		t.Error("expected error when vault is not a git repo")
	}
}

func TestAutoSync_TriggeredOnAdd(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n\n")
	_ = runCmd("init", "--git")
	_ = runCmd("sync", "--auto", "on")

	setInput("masterpass\nuser\npass\n\n")
	resetFlags()
	out.Reset()
	err := runCmd("add", "github")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	app.Wait()
}

func TestAutoSync_TriggeredOnRm(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n\n")
	_ = runCmd("init", "--git")
	_ = runCmd("sync", "--auto", "on")

	setInput("masterpass\nuser\npass\n\n")
	resetFlags()
	_ = runCmd("add", "github")
	app.Wait()

	setInput("masterpass\n")
	resetFlags()
	out.Reset()
	err := runCmd("rm", "github")
	if err != nil {
		t.Fatalf("rm: %v", err)
	}
	app.Wait()
}

// --- Phase 2: Shell integration tests ---

func TestAdd_WithTags(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\npass\n\n")
	out.Reset()
	err := runCmd("add", "aws-key", "--tag", "prod", "--tag", "infra")
	if err != nil {
		t.Fatalf("add with tags: %v", err)
	}
	if !strings.Contains(out.String(), `"aws-key" added`) {
		t.Errorf("unexpected output: %s", out.String())
	}

	setInput("masterpass\n")
	out.Reset()
	resetFlags()
	err = runCmd("get", "--print", "aws-key")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "prod") || !strings.Contains(output, "infra") {
		t.Errorf("expected tags in output: %s", output)
	}
}

func TestEnv_AllEntries(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\nsecret123\n\n")
	_ = runCmd("add", "aws-key")

	setInput("masterpass\ndbuser\ndbpass456\n\n")
	resetFlags()
	_ = runCmd("add", "db-password")

	setInput("masterpass\n")
	out.Reset()
	err := runCmd("env")
	if err != nil {
		t.Fatalf("env: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "export AWS_KEY='secret123'") {
		t.Errorf("expected AWS_KEY export: %s", output)
	}
	if !strings.Contains(output, "export DB_PASSWORD='dbpass456'") {
		t.Errorf("expected DB_PASSWORD export: %s", output)
	}
}

func TestEnv_FilterByTag(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\nprodpass\n\n")
	_ = runCmd("add", "aws-key", "--tag", "prod")

	setInput("masterpass\nuser\ndevpass\n\n")
	resetFlags()
	_ = runCmd("add", "dev-key", "--tag", "dev")

	setInput("masterpass\n")
	out.Reset()
	err := runCmd("env", "prod")
	if err != nil {
		t.Fatalf("env: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "AWS_KEY") {
		t.Errorf("expected AWS_KEY in output: %s", output)
	}
	if strings.Contains(output, "DEV_KEY") {
		t.Errorf("did not expect DEV_KEY in output: %s", output)
	}
}

func TestEnv_EmptyVault(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\n")
	out.Reset()
	err := runCmd("env")
	if err != nil {
		t.Fatalf("env: %v", err)
	}
	if out.String() != "" {
		t.Errorf("expected empty output, got: %s", out.String())
	}
}

func TestEnv_ShellEscape(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\nit's a \"test\"\n\n")
	_ = runCmd("add", "tricky")

	setInput("masterpass\n")
	out.Reset()
	err := runCmd("env")
	if err != nil {
		t.Fatalf("env: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "TRICKY=") {
		t.Errorf("expected TRICKY export: %s", output)
	}
	if strings.Contains(output, "it's") {
		t.Error("single quote should be escaped in output")
	}
}

func TestExec_NoCommand_Error(t *testing.T) {
	setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\n")
	err := runCmd("exec", "--")
	if err == nil {
		t.Error("expected error when no command specified")
	}
}

// --- Phase 2: Import/Export tests ---

func TestImportEnv(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	envFile := filepath.Join(t.TempDir(), "test.env")
	os.WriteFile(envFile, []byte("DB_HOST=localhost\nDB_PORT=5432\n"), 0600)

	setInput("masterpass\n")
	out.Reset()
	err := runCmd("import", "env", envFile)
	if err != nil {
		t.Fatalf("import env: %v", err)
	}
	if !strings.Contains(out.String(), "Imported 2 entries") {
		t.Errorf("unexpected output: %s", out.String())
	}

	setInput("masterpass\n")
	out.Reset()
	err = runCmd("list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "DB_HOST") || !strings.Contains(output, "DB_PORT") {
		t.Errorf("expected imported entries in list: %s", output)
	}
}

func TestImportCSV(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	csvFile := filepath.Join(t.TempDir(), "export.csv")
	os.WriteFile(csvFile, []byte("Title,Username,Password,Notes,URL\nGitHub,octocat,secret,,https://github.com\n"), 0600)

	setInput("masterpass\n")
	out.Reset()
	err := runCmd("import", "csv", csvFile)
	if err != nil {
		t.Fatalf("import csv: %v", err)
	}
	if !strings.Contains(out.String(), "Imported 1 entries") {
		t.Errorf("unexpected output: %s", out.String())
	}
}

func TestImportEnv_SkipsDuplicates(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\npass\n\n")
	_ = runCmd("add", "DB_HOST")

	envFile := filepath.Join(t.TempDir(), "test.env")
	os.WriteFile(envFile, []byte("DB_HOST=localhost\nDB_PORT=5432\n"), 0600)

	setInput("masterpass\n")
	resetFlags()
	out.Reset()
	err := runCmd("import", "env", envFile)
	if err != nil {
		t.Fatalf("import env: %v", err)
	}
	if !strings.Contains(out.String(), "1 skipped") {
		t.Errorf("expected 1 skipped: %s", out.String())
	}
}

func TestExportEnv(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\nsecret123\n\n")
	_ = runCmd("add", "aws-key")

	setInput("masterpass\n")
	out.Reset()
	err := runCmd("export", "env")
	if err != nil {
		t.Fatalf("export env: %v", err)
	}
	if !strings.Contains(out.String(), "AWS_KEY='secret123'") {
		t.Errorf("expected env export: %s", out.String())
	}
}

func TestExportCSV(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\nsecret123\nnotes here\n")
	_ = runCmd("add", "github")

	setInput("masterpass\n")
	resetFlags()
	out.Reset()
	err := runCmd("export", "csv")
	if err != nil {
		t.Fatalf("export csv: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "Name,Username,Password") {
		t.Errorf("expected CSV header: %s", output)
	}
	if !strings.Contains(output, "github,user,secret123") {
		t.Errorf("expected github entry: %s", output)
	}
}

// --- Phase 2: TOTP tests ---

func TestAdd_WithTotp(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\npass\n\nGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ\n")
	out.Reset()
	err := runCmd("add", "github", "--totp")
	if err != nil {
		t.Fatalf("add with totp: %v", err)
	}
	if !strings.Contains(out.String(), `"github" added`) {
		t.Errorf("unexpected output: %s", out.String())
	}
}

func TestTotp_GenerateCode(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\npass\n\nGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ\n")
	_ = runCmd("add", "github", "--totp")

	setInput("masterpass\n")
	resetFlags()
	out.Reset()
	err := runCmd("totp", "github")
	if err != nil {
		t.Fatalf("totp: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "expires in") {
		t.Errorf("expected TOTP code with expiry: %s", output)
	}
	if len(strings.TrimSpace(output)) < 6 {
		t.Errorf("expected at least 6 chars for code + message: %s", output)
	}
}

func TestTotp_NoSecret(t *testing.T) {
	setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\npass\n\n")
	_ = runCmd("add", "github")

	setInput("masterpass\n")
	resetFlags()
	err := runCmd("totp", "github")
	if err == nil {
		t.Error("expected error for entry without TOTP secret")
	}
}

func TestTotp_NotFound(t *testing.T) {
	setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\n")
	err := runCmd("totp", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent entry")
	}
}

func TestGet_ShowsTotpIndicator(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\npass\n\nGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ\n")
	_ = runCmd("add", "github", "--totp")

	setInput("masterpass\n")
	resetFlags()
	out.Reset()
	err := runCmd("get", "--print", "github")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(out.String(), "TOTP:     configured") {
		t.Errorf("expected TOTP indicator: %s", out.String())
	}
}

// --- Phase 2: Multiple vaults tests ---

func TestMigrateOldVault(t *testing.T) {
	dir := t.TempDir()
	oldVault := filepath.Join(dir, "vault.enc")
	os.WriteFile(oldVault, []byte("encrypted-data"), 0600)
	os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"auto_sync":true}`), 0600)

	migrateOldVault(dir)

	newPath := filepath.Join(dir, "vaults", "default", "vault.enc")
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("expected vault to be migrated to %s", newPath)
	}
	if _, err := os.Stat(oldVault); err == nil {
		t.Error("old vault should be moved, not copied")
	}

	newConfig := filepath.Join(dir, "vaults", "default", "config.json")
	if _, err := os.Stat(newConfig); err != nil {
		t.Errorf("expected config to be migrated to %s", newConfig)
	}
}

func TestMigrateOldVault_NoOldVault(t *testing.T) {
	dir := t.TempDir()
	migrateOldVault(dir)

	newPath := filepath.Join(dir, "vaults", "default", "vault.enc")
	if _, err := os.Stat(newPath); err == nil {
		t.Error("should not create vault if none exists")
	}
}

func TestMigrateOldVault_AlreadyMigrated(t *testing.T) {
	dir := t.TempDir()
	oldVault := filepath.Join(dir, "vault.enc")
	os.WriteFile(oldVault, []byte("old-data"), 0600)

	newDir := filepath.Join(dir, "vaults", "default")
	os.MkdirAll(newDir, 0700)
	os.WriteFile(filepath.Join(newDir, "vault.enc"), []byte("new-data"), 0600)

	migrateOldVault(dir)

	data, _ := os.ReadFile(filepath.Join(newDir, "vault.enc"))
	if string(data) != "new-data" {
		t.Error("should not overwrite already-migrated vault")
	}
}

// --- Config command tests ---

func TestConfig_ShowAll(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	out.Reset()
	err := runCmd("config")
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if !strings.Contains(out.String(), "auto-sync = off") {
		t.Errorf("expected default config output: %s", out.String())
	}
}

func TestConfig_GetKey(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	out.Reset()
	err := runCmd("config", "auto-sync")
	if err != nil {
		t.Fatalf("config get: %v", err)
	}
	if strings.TrimSpace(out.String()) != "off" {
		t.Errorf("expected 'off', got %q", out.String())
	}
}

func TestConfig_SetKey(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	out.Reset()
	err := runCmd("config", "auto-sync", "on")
	if err != nil {
		t.Fatalf("config set: %v", err)
	}
	if !strings.Contains(out.String(), "auto-sync = on") {
		t.Errorf("expected confirmation: %s", out.String())
	}

	out.Reset()
	err = runCmd("config", "auto-sync")
	if err != nil {
		t.Fatalf("config get after set: %v", err)
	}
	if strings.TrimSpace(out.String()) != "on" {
		t.Errorf("expected 'on' after set, got %q", out.String())
	}
}

func TestConfig_UnknownKey(t *testing.T) {
	setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	err := runCmd("config", "nonexistent")
	if err == nil {
		t.Error("expected error for unknown config key")
	}
}

func TestConfig_InvalidValue(t *testing.T) {
	setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	err := runCmd("config", "auto-sync", "maybe")
	if err == nil {
		t.Error("expected error for invalid value")
	}
}

// --- Security: env var name sanitization ---

func TestToEnvVar_StripsInvalidChars(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"aws-key", "AWS_KEY"},
		{"db password", "DB_PASSWORD"},
		{"normal_name", "NORMAL_NAME"},
		{"has.dots.here", "HAS_DOTS_HERE"},
		{"special!@#chars", "SPECIAL___CHARS"},
		{"123numeric", "_123NUMERIC"},
	}
	for _, tt := range tests {
		got := toEnvVar(tt.input)
		if got != tt.want {
			t.Errorf("toEnvVar(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToEnvVar_RejectsEmpty(t *testing.T) {
	got := toEnvVar("")
	if got != "" {
		t.Errorf("toEnvVar(\"\") = %q, want empty", got)
	}
}

func TestEnv_SkipsUnsafeNames(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\npass\n\n")
	_ = runCmd("add", "good-name")

	setInput("masterpass\n")
	out.Reset()
	err := runCmd("env")
	if err != nil {
		t.Fatalf("env: %v", err)
	}
	if !strings.Contains(out.String(), "GOOD_NAME") {
		t.Errorf("expected GOOD_NAME in output: %s", out.String())
	}
}

func TestExportEnv_QuotesValues(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\nhas spaces\n\n")
	_ = runCmd("add", "mykey")

	setInput("masterpass\n")
	resetFlags()
	out.Reset()
	err := runCmd("export", "env")
	if err != nil {
		t.Fatalf("export env: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "MYKEY='has spaces'") {
		t.Errorf("expected quoted value in export env: %s", output)
	}
}

func TestExportEnv_EscapesSingleQuotes(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nuser\nit's secret\n\n")
	_ = runCmd("add", "mykey")

	setInput("masterpass\n")
	resetFlags()
	out.Reset()
	err := runCmd("export", "env")
	if err != nil {
		t.Fatalf("export env: %v", err)
	}
	output := out.String()
	if strings.Contains(output, "it's") {
		t.Errorf("single quote should be escaped: %s", output)
	}
}

// --- Agent session caching tests ---

func TestGetPassword_AgentDisabled_FallsThrough(t *testing.T) {
	setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\n")
	pw, err := getPassword()
	if err != nil {
		t.Fatalf("getPassword: %v", err)
	}
	if string(pw) != "masterpass" {
		t.Errorf("expected 'masterpass', got %q", pw)
	}
}

func TestGetPassword_AgentStoreAndRetrieve(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	sock := testSocketPath(t)
	app.SocketPath = sock

	srv := startTestAgent(t, sock)
	defer srv.Stop()

	_ = runCmd("init")

	client := agent.NewClient(sock)
	err := client.Store(app.VaultPath, []byte("masterpass"), 5*time.Minute)
	if err != nil {
		t.Fatalf("direct store: %v", err)
	}

	setInput("")
	out.Reset()
	pw, err := getPassword()
	if err != nil {
		t.Fatalf("getPassword (from agent): %v", err)
	}
	if string(pw) != "masterpass" {
		t.Errorf("expected 'masterpass', got %q", pw)
	}
}

func TestLock_ClearsCache(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	sock := testSocketPath(t)
	app.SocketPath = sock

	srv := startTestAgent(t, sock)
	defer srv.Stop()

	_ = runCmd("init")

	client := agent.NewClient(sock)
	_ = client.Store(app.VaultPath, []byte("masterpass"), 5*time.Minute)

	out.Reset()
	err := runCmd("lock")
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	if !strings.Contains(out.String(), "Agent cache cleared") {
		t.Errorf("expected cleared message: %s", out.String())
	}

	setInput("")
	_, err = getPassword()
	if err == nil {
		t.Error("expected error when agent cache is cleared and no input available")
	}
}

func TestLock_NoAgent(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	sock := testSocketPath(t)
	app.SocketPath = sock
	_ = runCmd("init")

	out.Reset()
	err := runCmd("lock")
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	if !strings.Contains(out.String(), "Agent is not running") {
		t.Errorf("expected not running message: %s", out.String())
	}
}

func testSocketPath(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "pm-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, "a.sock")
}

func startTestAgent(t *testing.T, sock string) *agent.Server {
	t.Helper()
	srv := agent.NewServer(sock)
	go srv.Start()
	client := agent.NewClient(sock)
	for i := 0; i < 20; i++ {
		if client.Ping() {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	return srv
}
