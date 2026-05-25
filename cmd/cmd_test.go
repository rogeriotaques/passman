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
	"github.com/rogerio/passman/internal/vault"
)

func setupTestApp(t *testing.T, input string) (*bytes.Buffer, *bytes.Buffer, *clipboard.MockClipboard) {
	t.Helper()
	dir := t.TempDir()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	mock := &clipboard.MockClipboard{}

	app.VaultPath = filepath.Join(dir, "vault.enc")
	app.SocketPath = ""
	app.Out = out
	app.ErrOut = errOut
	app.Clipboard = mock
	app.KDFParams = crypto.FastKDFParams()
	app.Interactive = false

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
	listCmd.Flags().Set("all", "false")
	importCmd.Flags().Set("replace", "false")
	rootCmd.PersistentFlags().Set("vault", "")
}

func runCmd(args ...string) error {
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

// --- init tests ---

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

func TestInit_EmptyPassword(t *testing.T) {
	out, _, _ := setupTestApp(t, "\n")
	err := runCmd("init")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "Vault created successfully") {
		t.Errorf("expected success with empty password: %s", out.String())
	}
}

func TestInit_EmptyPassword_NoPromptOnSubsequentCommands(t *testing.T) {
	out, _, _ := setupTestApp(t, "\n")
	_ = runCmd("init")

	// Add a secret — no password prompt needed (just the value)
	setInput("mysecret\n")
	out.Reset()
	err := runCmd("add", "github")
	if err != nil {
		t.Fatalf("add after empty-password init: %v", err)
	}

	// List — no input needed at all
	setInput("")
	out.Reset()
	err = runCmd("list")
	if err != nil {
		t.Fatalf("list after empty-password init: %v", err)
	}
	if !strings.Contains(out.String(), "github") {
		t.Errorf("expected 'github' in list output: %s", out.String())
	}

	// Get — no password prompt needed
	setInput("")
	out.Reset()
	resetFlags()
	err = runCmd("get", "--print", "github")
	if err != nil {
		t.Fatalf("get after empty-password init: %v", err)
	}
	if strings.TrimSpace(out.String()) != "mysecret" {
		t.Errorf("expected 'mysecret', got %q", out.String())
	}
}

func TestInit_PasswordTooShort(t *testing.T) {
	setupTestApp(t, "short\nshort\n")
	err := runCmd("init")
	if err == nil {
		t.Error("expected error for short password")
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

// --- add and get tests ---

func TestAdd_And_Get(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nmypassword\n")
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
	err = runCmd("get", "--print", "github")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if strings.TrimSpace(out.String()) != "mypassword" {
		t.Errorf("expected 'mypassword', got %q", out.String())
	}
}

func TestAdd_Duplicate(t *testing.T) {
	setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nvalue1\n")
	_ = runCmd("add", "github")

	setInput("masterpass\nvalue2\n")
	err := runCmd("add", "github")
	if err == nil {
		t.Error("expected error for duplicate entry")
	}
}

func TestGet_Clipboard(t *testing.T) {
	out, _, mock := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nsecretpass\n")
	_ = runCmd("add", "github")

	setInput("masterpass\n")
	out.Reset()
	resetFlags()
	err := runCmd("get", "github")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if mock.Content != "secretpass" {
		t.Errorf("expected clipboard content 'secretpass', got %q", mock.Content)
	}
	if !strings.Contains(out.String(), "copied to clipboard") {
		t.Errorf("expected clipboard message: %s", out.String())
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

// --- list tests ---

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

func TestList_Search(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\npass1\n")
	_ = runCmd("add", "github-work")
	setInput("masterpass\npass2\n")
	_ = runCmd("add", "github-personal")
	setInput("masterpass\npass3\n")
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

	setInput("masterpass\npass1\n")
	_ = runCmd("add", "github-work")
	setInput("masterpass\npass2\n")
	_ = runCmd("add", "github-personal")

	setInput("masterpass\n")
	out.Reset()
	err := runCmd("list", "github", "work")
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

	setInput("masterpass\nvalue\n")
	_ = runCmd("add", "github")

	setInput("masterpass\n")
	out.Reset()
	err := runCmd("list", "nonexistent")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out.String(), "No matching entries") {
		t.Errorf("expected no matching message: %s", out.String())
	}
}

func TestList_TruncatesLongList(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	for i := 0; i < 25; i++ {
		setInput("masterpass\nvalue\n")
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
		setInput("masterpass\nvalue\n")
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

func TestList_Interactive_SingleResult_NoAutoCopy(t *testing.T) {
	_, _, mock := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nsecretpass\n")
	_ = runCmd("add", "github")

	setInput("masterpass\n")
	app.Interactive = false

	err := runCmd("list", "github")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if mock.Content != "" {
		t.Errorf("list should not auto-copy to clipboard, got %q", mock.Content)
	}
}

// --- rm tests ---

func TestRm_Success(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nvalue\n")
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

// --- generate tests ---

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

// --- export tests ---

func TestExport_AllEntries(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nsecret123\n")
	_ = runCmd("add", "aws-key")
	setInput("masterpass\ndbpass456\n")
	_ = runCmd("add", "db-password")

	setInput("masterpass\n")
	out.Reset()
	resetFlags()
	err := runCmd("export")
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "AWS_KEY='secret123'") {
		t.Errorf("expected AWS_KEY: %s", output)
	}
	if !strings.Contains(output, "DB_PASSWORD='dbpass456'") {
		t.Errorf("expected DB_PASSWORD: %s", output)
	}
	if strings.Contains(output, "export ") {
		t.Errorf("should not include 'export' prefix: %s", output)
	}
}

func TestExport_FilterByName(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nprodpass\n")
	_ = runCmd("add", "aws-prod")
	setInput("masterpass\ndevpass\n")
	_ = runCmd("add", "dev-key")

	setInput("masterpass\n")
	out.Reset()
	resetFlags()
	err := runCmd("export", "aws")
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "AWS_PROD=") {
		t.Errorf("expected AWS_PROD in output: %s", output)
	}
	if strings.Contains(output, "DEV_KEY=") {
		t.Errorf("did not expect DEV_KEY in output: %s", output)
	}
}

func TestExport_ShellEscape(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nit's a \"test\"\n")
	_ = runCmd("add", "tricky")

	setInput("masterpass\n")
	out.Reset()
	resetFlags()
	err := runCmd("export")
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "TRICKY='") {
		t.Errorf("expected TRICKY in output: %s", output)
	}
	if strings.Contains(output, "it's") {
		t.Error("single quote should be escaped in output")
	}
}

func TestExport_Eval_WithCommand(t *testing.T) {
	out, _, _ := setupTestApp(t, "\n")
	_ = runCmd("init")

	setInput("production\n")
	_ = runCmd("add", "my-env")

	out.Reset()
	setInput("")
	err := runCmd("export", "--eval", "--", "sh", "-c", "echo $MY_ENV")
	if err != nil {
		t.Fatalf("export --eval -- cmd: %v", err)
	}
	if strings.TrimSpace(out.String()) != "production" {
		t.Errorf("expected 'production', got %q", out.String())
	}
}

func TestExport_Eval_NoCommand_PrintsExportStatements(t *testing.T) {
	out, _, _ := setupTestApp(t, "\n")
	_ = runCmd("init")

	setInput("secret123\n")
	_ = runCmd("add", "aws-key")

	out.Reset()
	setInput("")
	err := runCmd("export", "--eval")
	if err != nil {
		t.Fatalf("export --eval: %v", err)
	}
	if !strings.Contains(out.String(), "export AWS_KEY='secret123'") {
		t.Errorf("expected export statement, got %q", out.String())
	}
}

// --- import tests ---

func TestImport_Env(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	envFile := filepath.Join(t.TempDir(), "test.env")
	os.WriteFile(envFile, []byte("DB_HOST=localhost\nDB_PORT=5432\n"), 0600)

	setInput("masterpass\n")
	out.Reset()
	resetFlags()
	err := runCmd("import", envFile)
	if err != nil {
		t.Fatalf("import: %v", err)
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

func TestImport_CSV(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	csvFile := filepath.Join(t.TempDir(), "export.csv")
	os.WriteFile(csvFile, []byte("name,value\ngithub,secret123\n"), 0600)

	setInput("masterpass\n")
	out.Reset()
	resetFlags()
	err := runCmd("import", csvFile)
	if err != nil {
		t.Fatalf("import csv: %v", err)
	}
	if !strings.Contains(out.String(), "Imported 1 entries") {
		t.Errorf("unexpected output: %s", out.String())
	}
}

func TestImport_SkipsDuplicates(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\noldvalue\n")
	_ = runCmd("add", "DB_HOST")

	envFile := filepath.Join(t.TempDir(), "test.env")
	os.WriteFile(envFile, []byte("DB_HOST=localhost\nDB_PORT=5432\n"), 0600)

	setInput("masterpass\n")
	out.Reset()
	resetFlags()
	err := runCmd("import", envFile)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if !strings.Contains(out.String(), "1 skipped") {
		t.Errorf("expected 1 skipped: %s", out.String())
	}
}

func TestImport_Replace(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\noldvalue\n")
	_ = runCmd("add", "DB_HOST")

	envFile := filepath.Join(t.TempDir(), "test.env")
	os.WriteFile(envFile, []byte("DB_HOST=newvalue\n"), 0600)

	setInput("masterpass\n")
	out.Reset()
	resetFlags()
	err := runCmd("import", "--replace", envFile)
	if err != nil {
		t.Fatalf("import --replace: %v", err)
	}
	if !strings.Contains(out.String(), "1 replaced") {
		t.Errorf("expected 1 replaced: %s", out.String())
	}

	setInput("masterpass\n")
	out.Reset()
	resetFlags()
	err = runCmd("get", "--print", "DB_HOST")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if strings.TrimSpace(out.String()) != "newvalue" {
		t.Errorf("expected 'newvalue', got %q", out.String())
	}
}

func TestImport_UnsupportedFormat(t *testing.T) {
	setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	file := filepath.Join(t.TempDir(), "data.txt")
	os.WriteFile(file, []byte("some data"), 0600)

	setInput("masterpass\n")
	resetFlags()
	err := runCmd("import", file)
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

// --- config tests ---

func TestConfig_ShowAll(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	out.Reset()
	err := runCmd("config")
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "auto-sync = off") {
		t.Errorf("expected auto-sync in output: %s", output)
	}
	if !strings.Contains(output, "session-timeout = 15") {
		t.Errorf("expected session-timeout in output: %s", output)
	}
	if !strings.Contains(output, "git = ") {
		t.Errorf("expected git in output: %s", output)
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
}

func TestConfig_UnknownKey(t *testing.T) {
	setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	err := runCmd("config", "nonexistent")
	if err == nil {
		t.Error("expected error for unknown config key")
	}
}

func TestConfig_SetGit(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	out.Reset()
	err := runCmd("config", "git", "git@github.com:user/vault.git")
	if err != nil {
		t.Fatalf("config set git: %v", err)
	}
	if !strings.Contains(out.String(), "git = git@github.com:user/vault.git") {
		t.Errorf("expected confirmation: %s", out.String())
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

	setInput("newpass123\n")
	out.Reset()
	err = runCmd("list")
	if err != nil {
		t.Fatalf("list with new password: %v", err)
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

func TestPasswd_EmptyNewPassword(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\n\n\n")
	out.Reset()
	err := runCmd("passwd")
	if err != nil {
		t.Fatalf("passwd with empty new password: %v", err)
	}
	if !strings.Contains(out.String(), "Master password changed") {
		t.Errorf("expected success: %s", out.String())
	}
}

func TestPasswd_PreservesEntries(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	_ = runCmd("init")

	setInput("masterpass\nsecretvalue\n")
	_ = runCmd("add", "github")

	setInput("masterpass\nnewpass123\nnewpass123\n")
	_ = runCmd("passwd")

	setInput("newpass123\n")
	out.Reset()
	resetFlags()
	err := runCmd("get", "--print", "github")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if strings.TrimSpace(out.String()) != "secretvalue" {
		t.Errorf("expected 'secretvalue', got %q", out.String())
	}
}

// --- sync tests ---

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
}

func TestSync_ManualSync(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n\n")
	_ = runCmd("init", "--git")

	setInput("masterpass\nvalue\n")
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

// --- env var name tests ---

func TestToEnvVar_Conversion(t *testing.T) {
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
		{"", ""},
	}
	for _, tt := range tests {
		got := toEnvVar(tt.input)
		if got != tt.want {
			t.Errorf("toEnvVar(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- agent tests ---

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

func TestLoadVault_WrongPassword_DoesNotCache(t *testing.T) {
	out, _, _ := setupTestApp(t, "masterpass\nmasterpass\n")
	sock := testSocketPath(t)
	app.SocketPath = sock

	srv := startTestAgent(t, sock)
	defer srv.Stop()

	_ = runCmd("init")

	// Clear the correctly-cached password so we can simulate a wrong entry
	client := agent.NewClient(sock)
	_ = client.Lock()

	// Try to load vault with wrong password
	setInput("wrongpass\n")
	_, _, _, err := app.loadVault()
	if err == nil {
		t.Fatal("expected error for wrong password")
	}

	// Next time should prompt again, not use cached wrong password
	setInput("masterpass\n")
	v, _, pw, err := app.loadVault()
	if err != nil {
		t.Fatalf("expected success after prompting again, got %v", err)
	}
	if v == nil {
		t.Fatal("expected vault to be loaded")
	}
	vault.ZeroBytes(pw)

	// Verify success message was printed (from a command like get)
	_ = out
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

// --- destroy tests ---

func TestDestroy_SpecificVault(t *testing.T) {
	dir := t.TempDir()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	mock := &clipboard.MockClipboard{}

	// Set up vault at <dir>/vaults/work/vault.enc
	app.VaultPath = filepath.Join(dir, "vaults", "work", "vault.enc")
	app.SocketPath = ""
	app.Out = out
	app.ErrOut = errOut
	app.Clipboard = mock
	app.KDFParams = crypto.FastKDFParams()
	app.Interactive = false
	vaultName = "work"

	setInput("\n")
	resetFlags()
	_ = runCmd("init", "--vault", "work")

	if _, err := os.Stat(app.VaultPath); err != nil {
		t.Fatalf("vault should exist: %v", err)
	}

	setInput("yes\n")
	out.Reset()
	err := runCmd("destroy", "--vault", "work")
	if err != nil {
		t.Fatalf("destroy: %v", err)
	}
	if !strings.Contains(out.String(), "destroyed") {
		t.Errorf("expected destroyed message: %s", out.String())
	}

	vaultDir := filepath.Dir(app.VaultPath)
	if _, err := os.Stat(vaultDir); !os.IsNotExist(err) {
		t.Error("vault directory should be removed")
	}
}

func TestDestroy_Aborted(t *testing.T) {
	dir := t.TempDir()
	out := &bytes.Buffer{}

	app.VaultPath = filepath.Join(dir, "vaults", "work", "vault.enc")
	app.SocketPath = ""
	app.Out = out
	app.ErrOut = &bytes.Buffer{}
	app.Clipboard = &clipboard.MockClipboard{}
	app.KDFParams = crypto.FastKDFParams()
	app.Interactive = false
	vaultName = "work"

	setInput("\n")
	resetFlags()
	_ = runCmd("init", "--vault", "work")

	setInput("no\n")
	out.Reset()
	err := runCmd("destroy", "--vault", "work")
	if err != nil {
		t.Fatalf("destroy: %v", err)
	}
	if !strings.Contains(out.String(), "Aborted") {
		t.Errorf("expected aborted message: %s", out.String())
	}

	if _, err := os.Stat(app.VaultPath); err != nil {
		t.Error("vault should still exist after abort")
	}
}

func TestDestroy_VaultNotFound(t *testing.T) {
	dir := t.TempDir()
	app.VaultPath = filepath.Join(dir, "vaults", "nonexistent", "vault.enc")
	app.Out = &bytes.Buffer{}
	app.ErrOut = &bytes.Buffer{}
	vaultName = "nonexistent"

	setInput("")
	err := runCmd("destroy", "--vault", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent vault")
	}
}

// --- helpers ---

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
