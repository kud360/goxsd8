package main

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
)

// subcommands is the contract's whole subcommand vocabulary, in the order
// usage documents it. Dispatch reads this one encoding rather than restating
// the names (STYLE D3/T4), and TestUsageCoversContract pins usage's own text
// to it. help and version are deliberately not members; doc.go states why.
var subcommands = []string{"parse", "validate", "gen"}

// usage is the terminal rendering of the CLI contract in doc.go: the
// subcommand syntax, the common flags and the implementation status, which
// the two state identically and change together. doc.go carries two things
// the help path does not print — the argument vocabulary and the CLI's
// relationship to the library.
const usage = `goxsd8 — XSD 1.1 schema compilation, instance validation, and code generation.

Usage (contract; subcommands land with their milestones):

  goxsd8 parse <schema.xsd>...
      Compile one or more schemas into a single set and print a
      summary (target namespaces, global declarations) on stdout.
      Exit 0 on a valid set; 1 on schema errors, one line per error
      on stderr.

  goxsd8 validate -schema <schema.xsd> [-schema <s2>]... <instance>...
      Assess instances against the compiled set; every schema needs
      its own -schema, and every positional argument is an instance.
      Source format by extension (.xml, .json, .ber) or forced with
      -format xml|json|ber, matched case-sensitively and applying to
      every instance of the invocation (there is no per-instance
      spelling).
      xsi:schemaLocation hints in XML instances augment the schema set
      (resolved relative to the instance; disable with -no-hints).
      Exit 0 valid, 1 invalid, 2 usage/IO, aggregated over the
      instances: 1 if any one of them is invalid. Each violation
      prints one line on stdout: <loc>: [<rule>] <message>.

  goxsd8 gen -schema <schema.xsd> -out <dir> [-schema <s2> -out <d2>]... [-backend strict|native]
      Generate Go types; repeated -schema/-out pairs map schemas to
      output directories (multiple schemas, multiple output dirs).

Flags common to all subcommands: -q (quiet), -v (debug logging via
slog to stderr; scope with GOXSD_DEBUG=parser,validate,codec).

Implemented today: the help path only. With no arguments, or with -h,
-help or --help in any argument position, goxsd8 prints this usage to
stdout and exits 0. Every other invocation exits 2, reporting on stderr
that a subcommand above is reserved but not yet built, that the name is
not one of them, or that the first argument is a flag and no subcommand
was given.
`

const (
	// helpPointer is the remedy line under every usage error. It names the
	// binary's own help path, which resolves wherever the binary runs; a
	// `go doc <import path>` invocation needs the module tree and fails for
	// an installed binary (#870).
	helpPointer = "run `goxsd8 -help` for the usage contract, or see https://github.com/kud360/goxsd8"

	// notImplementedFmt answers a name the contract reserves and no milestone
	// has built yet. Reserved to those names: its promise that a planned
	// interface documents the name is false for anything else (#514).
	notImplementedFmt = "goxsd8: %s is not yet implemented"

	// unknownSubcommandFmt answers a name outside the vocabulary.
	unknownSubcommandFmt = "goxsd8: unknown subcommand %q"

	// noSubcommand answers a first argument shaped like a flag. The common
	// flags qualify a subcommand and never stand alone, so -q is neither
	// unknown nor unimplemented — it is a subcommand short of an invocation.
	noSubcommand = "goxsd8: no subcommand given"
)

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
	// args is non-empty here: wantsHelp reports the bare invocation as a help
	// request. A failed stderr write cannot change the outcome — the exit code
	// is 2 either way, and stderr is the only channel left to report it on.
	_, _ = fmt.Fprintf(stderr, "%s\n%s\n", diagnose(args[0]), helpPointer)
	return 2
}

// diagnose names what is wrong with a first argument that is not a help
// request. Matching is case-sensitive per Go CLI convention, so goxsd8
// VALIDATE is an unknown subcommand rather than a reserved one.
func diagnose(arg string) string {
	if strings.HasPrefix(arg, "-") {
		return noSubcommand
	}
	if !slices.Contains(subcommands, arg) {
		return fmt.Sprintf(unknownSubcommandFmt, arg)
	}
	return fmt.Sprintf(notImplementedFmt, arg)
}

// wantsHelp accepts a help flag in any argument position, and only in the
// three bare spellings: the scan is deliberately positional-blind, gives --
// no end-of-options meaning, and does not parse -help=true (doc.go).
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
