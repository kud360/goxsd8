package main

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
)

// subcommands is the contract's whole subcommand vocabulary, in the order
// usage documents it. Diagnosis reads this one encoding rather than restating
// the names (STYLE D3/T4) — which name is built is answered where that name is
// dispatched, in run — and TestUsageCoversContract pins usage's own text to it.
// help and version are deliberately not members; doc.go states why.
var subcommands = []string{"parse", "validate", "gen"}

// usage is the terminal rendering of the CLI contract in doc.go: the
// subcommand syntax, the common flags and the implementation status, which
// the two state identically and change together. doc.go carries two things
// the help path does not print — the argument vocabulary and the CLI's
// relationship to the library.
const usage = `goxsd8 — XSD 1.1 schema compilation, instance validation, and code generation.

Usage (contract; subcommands land with their milestones):

  goxsd8 parse [-q] [-v] <schema.xsd>...
      Compile each schema argument and print its summary on stdout:
      the namespaces its components are in, then a count of each kind
      of declaration the schema documents make. Each argument is its
      own root document and its own run, in argument order — several
      arguments are several compilations, not one set.
      Exit 0 when every one compiles; 1 when any is rejected, its
      first error on stderr as <loc>: [<rule>] <message> (assembly
      stops there, so a rejected schema is one line); 2 when an
      argument cannot be read, which is never a verdict about a
      schema. The exit code is the worst of those outcomes.

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
slog to stderr; scope with GOXSD_DEBUG=parser,validate,codec). They
qualify a subcommand and follow its name — goxsd8 parse -q a.xsd, not
goxsd8 -q parse a.xsd. -q suppresses a subcommand's informational
output, which is parse's summary, and never a diagnosis: neither the
error lines above nor validate's violations are silenced by it.

Implemented today: the help path and parse. With no arguments, or with
-h, -help or --help in any argument position, goxsd8 prints this usage
to stdout and exits 0. goxsd8 parse compiles its arguments as above and
honours -q and -v, though GOXSD_DEBUG does not scope -v yet. Every other
invocation exits 2, reporting on stderr that validate or gen is reserved
but not yet built, that the name is not one of the three, that a flag
stands before the subcommand it qualifies, or that the first argument is
a flag and no subcommand was given.
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

	// noSubcommand answers a first argument shaped like a flag with no
	// subcommand anywhere after it. The common flags qualify a subcommand and
	// never stand alone, so -q is neither unknown nor unimplemented — it is a
	// subcommand short of an invocation.
	noSubcommand = "goxsd8: no subcommand given"

	// leadingFlagFmt answers a flag placed BEFORE the subcommand it qualifies.
	// The flags are the subcommand's own and follow its name (doc.go's
	// Argument vocabulary), so this invocation names a real subcommand and is
	// still a usage error — one that must not be reported as no subcommand at
	// all, which is what a scan of the first argument alone concluded (#472).
	leadingFlagFmt = "goxsd8: %s must follow the subcommand: goxsd8 %s %s ..."
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if wantsHelp(args) {
		// A truncated write means the user never got the help they asked for,
		// which is an IO fault and not a success.
		if _, err := fmt.Fprint(stdout, usage); err != nil {
			return exitUsage
		}
		return exitOK
	}
	// args is non-empty here: wantsHelp reports the bare invocation as a help
	// request.
	if args[0] == "parse" {
		return runParse(args[1:], stdout, stderr)
	}
	return usageError(stderr, diagnose(args))
}

// diagnose names what is wrong with an invocation that neither requests help
// nor names a built subcommand. Matching is case-sensitive per Go CLI
// convention, so goxsd8 VALIDATE is an unknown subcommand rather than a
// reserved one.
//
// It reads past the first argument for one purpose only: a flag standing where
// a subcommand belongs is a different fault depending on whether a subcommand
// follows it, and reporting `goxsd8 -q parse a.xsd` as no subcommand at all
// contradicts the argument list (#472). args is non-empty.
func diagnose(args []string) string {
	arg := args[0]
	if strings.HasPrefix(arg, "-") {
		if name, ok := subcommandIn(args[1:]); ok {
			return fmt.Sprintf(leadingFlagFmt, arg, name, arg)
		}
		return noSubcommand
	}
	if !slices.Contains(subcommands, arg) {
		return fmt.Sprintf(unknownSubcommandFmt, arg)
	}
	return fmt.Sprintf(notImplementedFmt, arg)
}

// subcommandIn returns the first argument that is a contract subcommand name.
// It reads the same one encoding of the vocabulary dispatch does (STYLE D3).
func subcommandIn(args []string) (string, bool) {
	for _, a := range args {
		if slices.Contains(subcommands, a) {
			return a, true
		}
	}
	return "", false
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
