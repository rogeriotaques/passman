package prompt

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

// runScript executes an osascript program and returns the dialog result on
// stdout. The message travels as an argument to the script's `on run argv`
// handler rather than being interpolated into the source, which rules out
// script injection via key names or process names.
func runScript(osascript, script, message string) (string, error) {
	if strings.HasPrefix(message, "-") {
		// osascript would read it as an option.
		message = " " + message
	}
	cmd := exec.Command(osascript, "-", message)
	cmd.Stdin = strings.NewReader(script)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if isUserCancel(stderr.String()) {
			return "", errCancelled
		}
		return "", errors.New("prompt failed: " + strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSuffix(stdout.String(), "\n"), nil
}

var errCancelled = errors.New("prompt cancelled")

// isUserCancel recognises osascript's exit when the cancel button is pressed.
func isUserCancel(stderr string) bool {
	return strings.Contains(stderr, "User canceled") || strings.Contains(stderr, "(-128)")
}

const confirmScript = `on run argv
return button returned of (display dialog (item 1 of argv) with title "passman" buttons {"Deny", "Allow"} default button "Deny" cancel button "Deny" with icon caution)
end run`

const secretScript = `on run argv
return text returned of (display dialog (item 1 of argv) with title "passman" default answer "" with hidden answer buttons {"Cancel", "OK"} default button "OK" cancel button "Cancel" with icon caution)
end run`

const textScript = `on run argv
return text returned of (display dialog (item 1 of argv) with title "passman" default answer "" buttons {"Cancel", "OK"} default button "OK" cancel button "Cancel" with icon note)
end run`
