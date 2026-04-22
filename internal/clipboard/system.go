package clipboard

import (
	"time"

	atotto "github.com/atotto/clipboard"
)

type SystemClipboard struct{}

func (c *SystemClipboard) CopyWithAutoClear(text string, clearAfter time.Duration) error {
	if err := atotto.WriteAll(text); err != nil {
		return err
	}
	go func() {
		time.Sleep(clearAfter)
		current, err := atotto.ReadAll()
		if err == nil && current == text {
			atotto.WriteAll("")
		}
	}()
	return nil
}

type MockClipboard struct {
	Content string
}

func (m *MockClipboard) CopyWithAutoClear(text string, _ time.Duration) error {
	m.Content = text
	return nil
}
