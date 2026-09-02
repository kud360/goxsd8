// Command goxsd8 is the command-line interface: schema compilation,
// instance validation, and code generation.
//
// # Usage (contract; subcommands land with their milestones)
//
//	goxsd8 parse [-q] [-v] <schema.xsd>...
//	    Compile each schema argument and print its summary on stdout:
//	    the namespaces its components are in, then a count of each kind
//	    of declaration the schema documents make. Each argument is its
//	    own root document and its own run, in argument order — several
//	    arguments are several compilations, not one set.
//	    Exit 0 when every one compiles; 1 when any is rejected, one
//	    line per error on stderr as <loc>: [<rule>] <message>; 2 when
//	    an argument cannot be read, which is never a verdict about a
//	    schema. The exit code is the worst of those outcomes.
//
//	goxsd8 validate -schema <schema.xsd> [-schema <s2>]... <instance>...
//	    Assess instances against the compiled set; every schema needs
//	    its own -schema, and every positional argument is an instance.
//	    Source format by extension (.xml, .json, .ber) or forced with
//	    -format xml|json|ber, matched case-sensitively and applying to
//	    every instance of the invocation (there is no per-instance
//	    spelling).
//	    xsi:schemaLocation hints in XML instances augment the schema set
//	    (resolved relative to the instance; disable with -no-hints).
//	    Exit 0 valid, 1 invalid, 2 usage/IO, aggregated over the
//	    instances: 1 if any one of them is invalid. Each violation
//	    prints one line on stdout: <loc>: [<rule>] <message>.
//
//	goxsd8 gen -schema <schema.xsd> -out <dir> [-schema <s2> -out <d2>]... [-backend strict|native]
//	    Generate Go types; repeated -schema/-out pairs map schemas to
//	    output directories (multiple schemas, multiple output dirs).
//
// Flags common to all subcommands: -q (quiet), -v (debug logging via
// slog to stderr; scope with GOXSD_DEBUG=parser,validate,codec). They
// qualify a subcommand and follow its name — goxsd8 parse -q a.xsd, not
// goxsd8 -q parse a.xsd. -q suppresses a subcommand's informational
// output, which is parse's summary, and never a diagnosis: neither the
// error lines above nor validate's violations are silenced by it.
//
// Implemented today: the help path and parse. With no arguments, or with
// -h, -help or --help in any argument position, goxsd8 prints this usage
// to stdout and exits 0. goxsd8 parse compiles its arguments as above and
// honours -q and -v, though GOXSD_DEBUG does not scope -v yet. Every other
// invocation exits 2, reporting on stderr that validate or gen is reserved
// but not yet built, that the name is not one of the three, that a flag
// stands before the subcommand it qualifies, or that the first argument is
// a flag and no subcommand was given.
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
// are the whole vocabulary. The flag-package forms -help=true and -h=1 are not
// help requests wherever they stand: before a subcommand one is a flag where a
// name belongs, and after one it is a flag whose value the subcommand does not
// accept. Both answers are usage errors naming the three spellings.
//
// Help is never scoped to a subcommand: a help flag in any argument position
// prints this whole contract and no other argument is examined, so both
// goxsd8 parse -h and goxsd8 -xyz -help print it and exit 0. That scan is
// positional-blind, which is also why -- carries no end-of-options meaning:
// goxsd8 -- -help is a help request.
//
// Everything else is positional. The subcommand name comes first and the
// common flags are its own, following it: goxsd8 parse -q a.xsd, never
// goxsd8 -q parse a.xsd, which is the usage error that says so rather than
// one claiming no subcommand was given. A subcommand's flags in turn precede
// its positional arguments, the flag package stopping at the first of them,
// so in goxsd8 parse a.xsd -q the -q is a schema location.
//
// A schema argument is a filesystem path, resolved to an absolute one before
// the document is read: an argument spelled absolutely or through "../" works,
// and an error cites a path the reader can open. Relative <xs:include>,
// <xs:import> and <xs:override> locations inside it resolve against that
// document's own directory.
//
// There is no version entry point and none is planned before 1.0: run
// go version -m $(which goxsd8) for the module version of a tagged build.
// -v is not available for one, being already assigned to debug logging.
//
// The CLI is a thin shell over the library — every capability here is
// reachable through the public packages, and the README documents both
// routes. Error output is stable and line-oriented for scripting.
package main
