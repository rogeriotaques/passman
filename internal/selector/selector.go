package selector

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

const (
	escUp1    = "\x1b[A"
	escUp2    = "\x1bOA"
	escDown1  = "\x1b[B"
	escDown2  = "\x1bOB"
	hideCursor = "\x1b[?25l"
	showCursor = "\x1b[?25h"
	clearLine  = "\x1b[2K\r"
)

type Options struct {
	Label string
	Items []string
	Size  int
}

func Run(opts Options, inFd int, out io.Writer) (int, error) {
	if len(opts.Items) == 0 {
		return -1, fmt.Errorf("no items to select from")
	}

	pageSize := opts.Size
	if pageSize <= 0 {
		pageSize = 15
	}

	oldState, err := term.MakeRaw(inFd)
	if err != nil {
		return -1, fmt.Errorf("raw mode: %w", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	restore := func() {
		fmt.Fprint(out, showCursor)
		term.Restore(inFd, oldState)
	}

	go func() {
		<-sigCh
		restore()
		os.Exit(130)
	}()

	defer restore()

	cursor := 0
	render(out, opts, cursor, pageSize)

	buf := make([]byte, 3)
	in := os.NewFile(uintptr(inFd), "/dev/stdin")

	for {
		n, err := in.Read(buf)
		if err != nil {
			return -1, err
		}

		input := string(buf[:n])
		switch {
		case input == "\r" || input == "\n":
			clearDisplay(out, opts, pageSize)
			return cursor, nil
		case input == "\x03" || input == "q":
			clearDisplay(out, opts, pageSize)
			return -1, nil
		case input == escUp1 || input == escUp2:
			if cursor > 0 {
				cursor--
			}
		case input == escDown1 || input == escDown2:
			if cursor < len(opts.Items)-1 {
				cursor++
			}
		}

		clearDisplay(out, opts, pageSize)
		render(out, opts, cursor, pageSize)
	}
}

func render(out io.Writer, opts Options, cursor, pageSize int) {
	fmt.Fprint(out, hideCursor)

	if opts.Label != "" {
		fmt.Fprintf(out, "%s\r\n", opts.Label)
	}

	start, end := visibleRange(cursor, len(opts.Items), pageSize)
	for i := start; i < end; i++ {
		if i == cursor {
			fmt.Fprintf(out, "  \x1b[7m %s \x1b[0m\r\n", opts.Items[i])
		} else {
			fmt.Fprintf(out, "    %s\r\n", opts.Items[i])
		}
	}
}

func clearDisplay(out io.Writer, opts Options, pageSize int) {
	lines := min(len(opts.Items), pageSize)
	if opts.Label != "" {
		lines++
	}
	for i := 0; i < lines; i++ {
		fmt.Fprintf(out, "\x1b[A%s", clearLine)
	}
}

func visibleRange(cursor, total, pageSize int) (int, int) {
	if total <= pageSize {
		return 0, total
	}

	start := cursor - pageSize/2
	if start < 0 {
		start = 0
	}
	end := start + pageSize
	if end > total {
		end = total
		start = end - pageSize
	}
	return start, end
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
