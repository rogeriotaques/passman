package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/rogerio/passman/cmd"
)

//go:embed VERSION
var version string

func main() {
	if err := cmd.SetVersion(strings.TrimSpace(version)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cmd.Execute()
}
