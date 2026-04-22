package clipboard

import "time"

type Clipboard interface {
	CopyWithAutoClear(text string, clearAfter time.Duration) error
}
