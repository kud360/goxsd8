package main

import (
	"fmt"
	"os"
)

func main() {
	// Subcommands land with their milestones (parse: M4, validate: M5,
	// gen: M9); see doc.go for the committed CLI contract.
	fmt.Fprintln(os.Stderr, "goxsd8: not yet implemented — see `go doc github.com/kud360/goxsd8/cmd/goxsd8` for the planned interface")
	os.Exit(2)
}
