// Command goxsd8 is the command-line interface: schema compilation,
// instance validation, and code generation.
//
// # Usage (contract; subcommands land with their milestones)
//
//	goxsd8 parse [-q] [-v] <schema.xsd>...
//	    Compile each schema argument and print its summary on stdout:
//	    the distinct namespaces of the components the compilation
//	    declares (the argument document and every one it includes,
//	    imports, overrides or redefines), in first-appearance order
//	    and none when it declares nothing, then a count of each kind
//	    of declaration the schema documents make. Each argument is its
//	    own root document and its own run, in argument order — several
//	    arguments are several compilations, not one set.
//	    Exit 0 when every one compiles; 1 when any is rejected, its
//	    first error on stderr as <loc>: [<rule>] <message> (assembly
//	    stops there, so a rejected schema is one line); 2 when an
//	    argument cannot be read, which is never a verdict about a
//	    schema. The exit code is the worst of those outcomes.
//
//	goxsd8 validate -schema <schema.xsd> [-schema <s2>]... <instance>...
//	    Assess instances against the compiled set; every schema needs
//	    its own -schema, and every positional argument is an instance.
//	    The -schema documents compose into ONE set — several of them are
//	    one compilation, not one each — and - names standard input as an
//	    instance, never as a schema.
//	    Source format by extension (.xml, .json, .ber) or forced with
//	    -format xml|json|ber, matched case-sensitively and applying to
//	    every instance of the invocation (there is no per-instance
//	    spelling); an unrecognized token, and an instance whose
//	    extension names none of the three, are usage errors listing the
//	    values. Only xml is assessed today: json and ber are reserved,
//	    and an instance in either exits 2 saying so.
//	    xsi:schemaLocation hints on the document element of an XML
//	    instance augment the schema set for that instance (resolved
//	    relative to the instance; disable with -no-hints).
//	    Exit 0 when no instance was charged a violation, 1 invalid,
//	    2 usage/IO, 3 when the schema set does not compile, aggregated
//	    over the instances: 1 if any one of them is invalid. Every
//	    instance is assessed — the run never stops at the first invalid
//	    one — and each violation prints one line on stdout:
//	    <loc>: [<rule>] <message>.
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
// Implemented today: the help path, parse and validate. With no arguments,
// or with -h, -help or --help in any argument position, goxsd8 prints this
// usage to stdout and exits 0. goxsd8 parse compiles its arguments as above
// and honours -q and -v; goxsd8 validate assesses XML instances as above and
// honours -v, -q silencing nothing there because it writes no informational
// output. GOXSD_DEBUG scopes -v for neither. Every other invocation exits 2,
// reporting on stderr that gen is reserved but not yet built, that the name
// is not one of the three, that a flag stands before the subcommand it
// qualifies, or that the first argument is a flag and no subcommand was
// given.
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
// document's own directory. That resolution is confined to no subtree: a
// schema document may name any path the invoking user can read, so an
// <xs:include schemaLocation="../../../etc/passwd"> is served like any other.
//
// validate composes its -schema documents into ONE schema set, through a
// synthesized wrapper schema document that <import>s each document declaring a
// target namespace of its own and <include>s each declaring none. Every
// cross-document rule an ordinary assembly enforces therefore holds over the
// set: two documents colliding on a name are sch-props-correct clause 2, and a
// reference no document of the set supplies is src-resolve. A set that does not
// compile exits 3, which is neither a verdict about an instance (1) nor an
// argument this process could not read (2), so a script tells "your schema is
// wrong" from "your data is wrong" by exit code alone.
//
// An instance argument spelled - is standard input. -schema - is not
// supported: a schema document's location is the base URI its own relative
// <xs:include>, <xs:import> and <xs:override> references resolve against, and
// standard input has none.
//
// validate follows an xsi:schemaLocation or xsi:noNamespaceSchemaLocation hint
// carried by the DOCUMENT ELEMENT of an XML instance, and no other element's.
// §4.3.2 clause 5 admits one on any element and makes its effect global to the
// assessment either way, and clause 3 obliges a processor to dereference none
// of them, so the scope above is the strategy this CLI publishes rather than a
// shortfall. Each hinted location is resolved against the instance's own path
// (clause 4) and joins that instance's schema set alone.
//
// -no-hints turns all of it off — clause 3's "Schema processors should provide
// an option to control whether they do so" — and turning it off is what makes
// an insufficient -schema set fail: a validation root the set declares nothing
// for is charged cvc-assess-elt (§3.3.4.6) instead of the run quietly
// succeeding on a schema document the instance itself named.
//
// There is no version entry point and none is planned before 1.0: run
// go version -m $(which goxsd8) for the module version of a tagged build.
// -v is not available for one, being already assigned to debug logging.
//
// The CLI is a thin shell over the library — every capability here is
// reachable through the public packages, and the README documents both
// routes. Error output is stable and line-oriented for scripting.
package main
