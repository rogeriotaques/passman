//go:build darwin

package prompt

import "errors"

const osascriptPath = "/usr/bin/osascript"

type dialog struct{}

// System returns the macOS dialog prompter.
func System() Prompter {
	return dialog{}
}

func (dialog) Confirm(message string) (bool, error) {
	out, err := runScript(osascriptPath, confirmScript, message)
	if errors.Is(err, errCancelled) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return out == "Allow", nil
}

func (dialog) Secret(message string) ([]byte, error) {
	out, err := runScript(osascriptPath, secretScript, message)
	if err != nil {
		return nil, err
	}
	return []byte(out), nil
}

func (dialog) Text(message string) (string, error) {
	return runScript(osascriptPath, textScript, message)
}
