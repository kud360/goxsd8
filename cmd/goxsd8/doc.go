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
// The CLI is a thin shell over the library — every capability here is
// reachable through the public packages, and the README documents both
// routes. Error output is stable and line-oriented for scripting.
package main
