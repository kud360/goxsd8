package main

import (
	"fmt"
	"io"
	"os"
)

// usage is the terminal rendering of the CLI contract in doc.go; the two
// state the same interface and change together.
const usage = `goxsd8 — XSD 1.1 schema compilation, instance validation, and code generation.

Usage (contract; subcommands land with their milestones):

  goxsd8 parse <schema.xsd>...
      Compile one or more schemas into a single set and print a
      summary (target namespaces, global declarations, errors).
      Exit 0 on a valid set, 1 on schema errors.

  goxsd8 validate -schema <schema.xsd>... <instance>...
      Assess instances against the compiled set. Source format by
      extension (.xml, .json, .ber) or forced with -format.
      xsi:schemaLocation hints in XML instances augment the schema set
      (resolved relative to the instance; disable with -no-hints).
      Exit 0 valid, 1 invalid, 2 usage/IO. Each violation prints one
      line: <loc>: [<rule>] <message>.

  goxsd8 gen -schema <schema.xsd> -out <dir> [-schema <s2> -out <d2>]... [-backend strict|native]
      Generate Go types; repeated -schema/-out pairs map schemas to
      output directories (multiple schemas, multiple output dirs).

Flags common to all subcommands: -q (quiet), -v (debug logging via
slog to stderr; scope with GOXSD_DEBUG=parser,validate,codec).
`

const notImplemented = "goxsd8: not yet implemented — see `go doc github.com/kud360/goxsd8/cmd/goxsd8` for the planned interface"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if wantsHelp(args) {
		// 2 is the contract's usage/IO exit code: a truncated write means
		// the user never got the help they asked for.
		if _, err := fmt.Fprint(stdout, usage); err != nil {
			return 2
		}
		return 0
	}
	// Subcommands land with their milestones (parse: M4, validate: M5,
	// gen: M9); see doc.go for the committed CLI contract.
	// A failed stderr write cannot change the outcome — the exit code is 2
	// either way, and stderr is the only channel left to report it on.
	_, _ = fmt.Fprintln(stderr, notImplemented)
	return 2
}

// wantsHelp accepts a help flag in any argument position: no subcommand is
// implemented yet, so nothing downstream can claim -h/-help/--help first.
func wantsHelp(args []string) bool {
	if len(args) == 0 {
		return true
	}
	for _, a := range args {
		if a == "-h" || a == "-help" || a == "--help" {
			return true
		}
	}
	return false
}
