package prompt

import "errors"

// Prompter asks the user a question through a graphical dialog. The background
// agent and `passman ssh askpass` run without a terminal, so this is their only
// way to reach the user.
type Prompter interface {
	// Confirm asks a yes/no question. It returns false when the user denies or
	// dismisses the dialog.
	Confirm(message string) (bool, error)
	// Secret asks for a value that must not be echoed.
	Secret(message string) ([]byte, error)
	// Text asks for a visible value.
	Text(message string) (string, error)
}

var ErrUnavailable = errors.New("no graphical prompt available")

// Mock is a Prompter with scripted answers, for tests.
type Mock struct {
	ConfirmAnswer bool
	SecretAnswer  string
	TextAnswer    string
	Err           error
	Messages      []string
}

func (m *Mock) Confirm(message string) (bool, error) {
	m.Messages = append(m.Messages, message)
	return m.ConfirmAnswer, m.Err
}

func (m *Mock) Secret(message string) ([]byte, error) {
	m.Messages = append(m.Messages, message)
	if m.Err != nil {
		return nil, m.Err
	}
	return []byte(m.SecretAnswer), nil
}

func (m *Mock) Text(message string) (string, error) {
	m.Messages = append(m.Messages, message)
	return m.TextAnswer, m.Err
}
