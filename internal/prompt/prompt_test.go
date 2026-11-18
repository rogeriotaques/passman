package prompt

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// fakeOSAScript writes a shell script that mimics osascript: it echoes the
// message argument, or fails like a cancelled dialog.
func fakeOSAScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "osascript")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunScript_PassesMessageAsArgument(t *testing.T) {
	fake := fakeOSAScript(t, `printf '%s\n' "$2"`)
	msg := `he said "hi" \ and; rm -rf / 鍵`

	out, err := runScript(fake, "ignored", msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != msg {
		t.Errorf("got %q, want %q", out, msg)
	}
}

func TestRunScript_ShieldsLeadingDash(t *testing.T) {
	fake := fakeOSAScript(t, `printf '%s\n' "$2"`)

	out, err := runScript(fake, "ignored", "-rf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != " -rf" {
		t.Errorf("got %q, want leading space", out)
	}
}

func TestRunScript_FeedsScriptOnStdin(t *testing.T) {
	fake := fakeOSAScript(t, `cat`)

	out, err := runScript(fake, "the script", "msg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "the script" {
		t.Errorf("got %q", out)
	}
}

func TestRunScript_UserCancel(t *testing.T) {
	fake := fakeOSAScript(t, `echo "execution error: User canceled. (-128)" >&2; exit 1`)

	_, err := runScript(fake, "s", "m")
	if !errors.Is(err, errCancelled) {
		t.Errorf("expected errCancelled, got %v", err)
	}
}

func TestRunScript_OtherFailure(t *testing.T) {
	fake := fakeOSAScript(t, `echo "boom" >&2; exit 2`)

	_, err := runScript(fake, "s", "m")
	if err == nil || errors.Is(err, errCancelled) {
		t.Errorf("expected generic failure, got %v", err)
	}
	if err != nil && err.Error() != "prompt failed: boom" {
		t.Errorf("unexpected message: %v", err)
	}
}

func TestMock_RecordsMessages(t *testing.T) {
	m := &Mock{ConfirmAnswer: true, SecretAnswer: "pw", TextAnswer: "yes"}

	ok, _ := m.Confirm("c")
	secret, _ := m.Secret("s")
	text, _ := m.Text("t")

	if !ok || string(secret) != "pw" || text != "yes" {
		t.Errorf("unexpected answers: %v %q %q", ok, secret, text)
	}
	if len(m.Messages) != 3 || m.Messages[1] != "s" {
		t.Errorf("messages not recorded: %v", m.Messages)
	}
}
