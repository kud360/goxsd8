// Command goxsd8 is the command-line interface: schema compilation,
// instance validation, and code generation.
//
// # Usage (contract; subcommands land with their milestones)
//
//	goxsd8 parse [-q] [-v] <schema.xsd>...
//	    Compile each schema argument and print its summary on stdout:
//	    the argument as it was spelled, then a block of lines indented
//	    two spaces and spelled "<label>: <value>". A namespace: line
//	    for each distinct namespace of the components the compilation
//	    declares (the argument document and every one it includes,
//	    imports, overrides or redefines) comes first, in
//	    first-appearance order and none when it declares nothing.
//	    Then one count per kind of declaration those documents make,
//	    all seven kinds always and always in this order: types,
//	    elements, attributes, attribute groups, model groups,
//	    notations, identity constraints. types counts the simple and
//	    the complex definitions together on the one line; model groups
//	    counts the top-level <xs:group> definitions; and the
//	    components: line closing the block is the sum of those seven,
//	    which no namespace line is counted into. Each argument is its
//	    own root document and its own run, in argument order — several
//	    arguments are several compilations, not one set.
//	    Exit 0 when every one compiles; 1 when any is rejected, its
//	    first error on stderr as <loc>: [<rule>] <message> (assembly
//	    stops there, so a rejected schema is one error line); 2 when
//	    an argument cannot be read, which is never a verdict about a
//	    schema. The exit code is the worst of those outcomes.
//	    A document whose root is not <xs:schema>, and a rejection
//	    in the s4s-grammar class (xsderr/doc.go), have no rule to
//	    cite: each prints the bare <message> instead, carrying
//	    what location it has inside the sentence rather than as
//	    the <loc>: prefix.
//	    An <xs:include>, <xs:import>, <xs:override> or empty
//	    <xs:redefine> whose schemaLocation resolves to no document is
//	    named on stderr at its own position, with no rule ID and no
//	    change of exit code: the skip is legal, so the summary above
//	    is printed off a schema short of whatever that document
//	    declares. -q does not silence it. A bare <xs:import>, which
//	    names no document, is not reported. A REJECTED schema is
//	    named the same way, for the directives assembly reached
//	    before it stopped, in a line saying the assembly was rejected
//	    rather than compiled and saying nothing about whether the
//	    unread document had a part in that.
//
//	goxsd8 validate -schema <schema.xsd> [-schema <s2>]... <instance>...
//	    Assess instances against the compiled set; every schema needs
//	    its own -schema, and every positional argument is an instance.
//	    The -schema documents compose into ONE set — several of them are
//	    one compilation, not one each — and - names standard input as an
//	    instance, never as a schema.
//	    A -schema document's own unresolved directive is named on
//	    stderr the same way, once for the set, before any instance is
//	    assessed — and when the set does not compile, for whichever
//	    documents the assembly reached, so one document's rejection
//	    does not silence another's shortfall.
//	    Source format by extension (.xml, .json, .ber) or forced with
//	    -format xml|json|ber, matched case-sensitively and applying to
//	    every instance of the invocation (there is no per-instance
//	    spelling); an unrecognized token, and an instance whose
//	    extension names none of the three, are usage errors listing the
//	    values. Only xml is assessed today: json and ber are reserved,
//	    and an instance in either exits 2 saying so.
//	    xsi:schemaLocation hints on the document element of an XML
//	    instance augment the schema set for that instance (resolved
//	    relative to the instance; disable with -no-hints). A hint the
//	    set will not compose with is the instance's own fault, not
//	    the schema set's: it is reported on stderr naming that
//	    instance, whose hints are then dropped, and it is assessed
//	    against the -schema documents alone.
//	    Exit 0 when no instance was charged a violation and none left
//	    a check undecided, 1 invalid, 2 usage/IO, and
//	    3 when the schema set does not compile. 4 is an instance the
//	    assessment declined to decide: no violation charged, and a
//	    check it reached not performed, so the instance stands
//	    undecided against the rule that check answers to rather than
//	    clean. The code is the worst outcome over the instances, by
//	    severity and not by number: undecided is less severe than
//	    invalid, so a run holding one of each exits 1, and 4 answers a
//	    run where something was left undecided and nothing was worse.
//	    Every instance is assessed — the run never stops at the first
//	    invalid one — and each violation, and each check the assessment
//	    declined, prints one line on stdout:
//	    <loc>: [<rule>] <message>.
//
//	goxsd8 gen -schema <schema.xsd> -out <dir> [-schema <s2> -out <d2>]... [-backend strict|native]
//	    Generate Go types; repeated -schema/-out pairs map schemas to
//	    output directories (multiple schemas, multiple output dirs).
//	    Exit 0 when every pair is generated; 1 when a schema is
//	    rejected, its first error on stderr as <loc>: [<rule>]
//	    <message>; 2 when an argument cannot be read, an output
//	    directory cannot be written, or a -schema stands without its
//	    -out. The exit code is the worst of those outcomes. Those are
//	    the codes gen answers with once M9 builds it: until then every
//	    gen invocation exits 2, reporting that gen is not yet
//	    implemented.
//
// Flags common to all subcommands: -q (quiet), -v (debug logging via
// slog to stderr; scope with GOXSD_DEBUG=parser,validate,codec). They
// qualify a subcommand and follow its name — goxsd8 parse -q a.xsd, not
// goxsd8 -q parse a.xsd. -q suppresses a subcommand's informational
// output, which is parse's summary, and never a diagnosis: neither the
// error lines above nor validate's violations and undecided checks are
// silenced by it.
//
// Implemented today: the help path, parse and validate. With no arguments,
// or with -h, -help or --help in any argument position, goxsd8 prints this
// usage to stdout and exits 0. goxsd8 parse compiles its arguments as above
// and honours -q and -v; goxsd8 validate assesses XML instances as above and
// honours -v, -q silencing nothing there because it writes no informational
// output. GOXSD_DEBUG scopes -v for neither. Every other invocation exits 2,
// reporting on stderr that gen is reserved but not yet implemented, that the
// name is not one of the three, that a help request carries a value, that a
// flag stands before the subcommand it qualifies, or that the first argument
// is a flag and no subcommand was given.
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
// accept. Both answers are usage errors naming the three spellings, and
// neither reports the flag as one to move: no argument position accepts a
// valued help spelling, so there is nowhere to move it to.
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
// its positional arguments, the flag package stopping at the first of them, so
// a flag-shaped token standing after one is not read as a flag at all: it is
// reported as the misplacement it is, exit 2, rather than taken for a path or
// silently dropped, so in goxsd8 parse a.xsd -q neither is -q a schema
// location nor is the summary it asked to suppress printed. The exact
// spelling - is never a misplaced flag — it is validate's standard-input
// instance argument — and a positional genuinely named -q is spelled ./-q, as
// a file named - is.
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
// An instance the assessment DECLINED to decide exits 4: nothing was charged
// against it, and a check the assessment reached was not performed — an
// <xs:assert> whose {test} this engine does not evaluate, or a {type table}
// that withheld its ·conditionally selected· type — so the document stands
// undecided against exactly the rules those checks answer to, which is what
// [validate.Result.Unevaluated] records and validate's own doc.go calls not a
// pass. Each such check prints the "<loc>: [<rule>] <message>" line a
// violation prints, on stdout, so a script scanning for a rule ID reads them
// with the charges. 4 is the least severe outcome after 0 and the codes
// aggregate by severity rather than by number, so a run whose instances mix
// invalid with undecided exits 1: a gate acts on the verdict it has.
//
// An instance argument spelled - is standard input. -schema - is not
// supported: a schema document's location is the base URI its own relative
// <xs:include>, <xs:import> and <xs:override> references resolve against, and
// standard input has none. That spelling is refused, exit 2, rather than
// opened, so a file which happens to be named - is never compiled as the
// schema set behind it; ./- is what names that file.
//
// validate follows an xsi:schemaLocation or xsi:noNamespaceSchemaLocation hint
// carried by the DOCUMENT ELEMENT of an XML instance, and no other element's.
// §4.3.2 clause 5 admits one on any element and makes its effect global to the
// assessment either way, and clause 3 obliges a processor to dereference none
// of them, so the scope above is the strategy this CLI publishes rather than a
// shortfall. Each hinted location is resolved against the instance's own path
// (clause 4) and joins that instance's schema set alone.
//
// A hint that instance's set will not compose with — one pairing a namespace
// with a document declaring another (src-import clause 3.1), or naming a
// document that is not well-formed — is a fault of the INSTANCE that carried
// it and never of the -schema set: clause 3 obliges a processor to dereference
// no hint at all, so the hints of that instance are reported unusable on
// stderr, naming it, and it is assessed against the -schema documents alone.
// Exit 3 answers a -schema set that does not compile and nothing else.
//
// A hint naming a document that is NOT THERE is named on stderr too, against
// the instance that carried it and by the location it resolved to, carrying no
// rule ID and moving no exit code: src-import and src-include alike make a
// schemaLocation that resolves to nothing legal to skip, so that hint's
// siblings still apply and the set still composes — short of whatever the
// document would have declared, which is the fact the line carries and clause 3
// calls "less than complete ·assessment· outcomes". When the instance's hints
// instead fail to compose with the -schema set, the same shortfall is named
// first, in a line saying the augmented set was rejected rather than compiled
// and saying nothing about whether the unread document had a part in that;
// the hints are then reported unusable and the instance falls back to the
// -schema set alone.
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
// The CLI is a thin shell over the library for the capabilities it runs
// today: schema compilation and XML instance assessment are reachable through
// parser, xsd, validate and validate/xmlsrc, and the README documents both
// routes for them. The capabilities this page reserves have no library route
// either — validate/jsonsrc (M8), codegen (M9) with codec (M10) and
// validate/bersrc (M11) each export nothing yet, so for JSON instances, code
// generation and BER instances there is no second route to document until
// those milestones land. Error output is stable and line-oriented for
// scripting.
package main
