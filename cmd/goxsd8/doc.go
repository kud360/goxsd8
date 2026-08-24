// Command goxsd8 is the command-line interface: schema compilation,
// instance validation, and code generation.
//
// # Usage (contract; subcommands land with their milestones)
//
//	goxsd8 parse <schema.xsd>...
//	    Compile one or more schemas into a single set and print a
//	    summary (target namespaces, global declarations, errors).
//	    Exit 0 on a valid set, 1 on schema errors.
//
//	goxsd8 validate -schema <schema.xsd>... <instance>...
//	    Assess instances against the compiled set. Source format by
//	    extension (.xml, .json, .ber) or forced with -format.
//	    xsi:schemaLocation hints in XML instances augment the schema set
//	    (resolved relative to the instance; disable with -no-hints).
//	    Exit 0 valid, 1 invalid, 2 usage/IO. Each violation prints one
//	    line: <loc>: [<rule>] <message>.
//
//	goxsd8 gen -schema <schema.xsd> -out <dir> [-schema <s2> -out <d2>]... [-backend strict|native]
//	    Generate Go types; repeated -schema/-out pairs map schemas to
//	    output directories (multiple schemas, multiple output dirs).
//
// Flags common to all subcommands: -q (quiet), -v (debug logging via
// slog to stderr; scope with GOXSD_DEBUG=parser,validate,codec).
//
// Implemented today: the help path only. With no arguments, or with -h,
// -help or --help in any argument position, goxsd8 prints this usage to
// stdout and exits 0. Every other invocation exits 2, reporting on stderr
// that a subcommand above is reserved but not yet built, that the name is
// not one of them, or that the first argument is a flag and no subcommand
// was given.
//
// # Argument vocabulary
//
// The subcommand vocabulary is exactly parse, validate and gen, matched
// case-sensitively: goxsd8 VALIDATE is an unknown subcommand, not a
// reserved one spelled loudly.
//
// The bareword help is not a contract name. A help request is spelled as a
// flag, so goxsd8 help is an unknown subcommand.
//
// The help flag is spelled -h, -help or --help, and those three bare tokens
// are the whole vocabulary. The flag-package form -help=true is not a help
// request; it is a first argument shaped like a flag.
//
// Help is never scoped to a subcommand: a help flag in any argument position
// prints this whole contract and no other argument is examined, so both
// goxsd8 parse -h and goxsd8 -xyz -help print it and exit 0. Argument
// scanning is deliberately positional-blind until a subcommand is built,
// which is also why -- carries no end-of-options meaning: goxsd8 -- -help is
// a help request. A subcommand that parses its own flags may narrow this
// when it lands.
//
// There is no version entry point and none is planned before 1.0: run
// go version -m $(which goxsd8) for the module version of a tagged build.
// -v is not available for one, being already assigned to debug logging.
//
// The CLI is a thin shell over the library — every capability here is
// reachable through the public packages, and the README documents both
// routes. Error output is stable and line-oriented for scripting.
package main
