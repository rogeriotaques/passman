package cmd

import (
	"fmt"
	"os"
	"testing"
)

// TestMain refuses to act as the background agent. agent.EnsureAgent
// re-executes os.Executable(), which is the test binary during tests. Without
// this guard the child parses "_agent" as a test argument and runs the whole
// suite again, spawning more children on each pass.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "_agent" {
		fmt.Fprintln(os.Stderr, "refusing to start agent from test binary")
		os.Exit(1)
	}
	os.Exit(m.Run())
}
