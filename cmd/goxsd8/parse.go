package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The exit codes parse answers with, ordered by severity so that a
// multi-argument invocation reports the worst outcome over its runs by taking
// their maximum: a clean compilation, a schema the parser rejected, and a
// document that yielded no verdict at all.
const (
	exitOK      = 0
	exitInvalid = 1
	exitUsage   = 2
)

// helpNotAFlagValue answers the flag-package spellings -h=…/-help=…, which
// wantsHelp deliberately does not accept: the help vocabulary is the three
// bare tokens and nothing else (doc.go), so this is a usage error naming the
// spelling that would have worked.
const helpNotAFlagValue = "goxsd8: parse: a help request is spelled -h, -help or --help, with no value"

// runParse implements `goxsd8 parse`: it compiles each schema argument and
// writes its summary to stdout, one run per argument in argument order.
// args excludes the subcommand name itself.
//
// The flags are parsed here rather than by run, which is what lets the
// contract state that the common flags qualify a subcommand and follow it
// (doc.go's Argument vocabulary). Nothing before the subcommand name reaches
// this function, and a help request never does: run answers one before it
// dispatches, so wiring a flag set here cannot take -h away from the help
// path.
func runParse(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("parse", flag.ContinueOnError)
	// Discarding the flag set's output silences its own usage rendering, which
	// would print a second, narrower contract next to doc.go's; helpPointer
	// names the real one instead.
	flags.SetOutput(io.Discard)
	quiet := flags.Bool("q", false, "suppress the summary")
	verbose := flags.Bool("v", false, "log the parser's assembly at debug level to stderr")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return usageError(stderr, helpNotAFlagValue)
		}
		return usageError(stderr, fmt.Sprintf("goxsd8: parse: %v", err))
	}
	locations := flags.Args()
	if len(locations) == 0 {
		return usageError(stderr, "goxsd8: parse: no schema given")
	}

	// A nil logger selects parser's silent default, so -v is the whole of the
	// injection and quiet is the default (STYLE L1).
	var log *slog.Logger
	if *verbose {
		log = slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	code := exitOK
	for _, location := range locations {
		code = max(code, parseOne(location, *quiet, log, stdout, stderr))
	}
	return code
}

// parseOne compiles one schema argument and reports its exit code. It never
// stops the run: every argument is compiled and reported, so one broken schema
// does not hide the next argument's verdict.
func parseOne(location string, quiet bool, log *slog.Logger, stdout, stderr io.Writer) int {
	root, err := rootLocation(location)
	if err != nil {
		return usageError(stderr, fmt.Sprintf("goxsd8: parse: %v", err))
	}
	// The resolver is rooted at the filesystem root and the document is named
	// by its absolute path, so that an argument spelled absolutely or through
	// "../" resolves at all, a relative <xs:include> in it resolves against
	// its own directory, and the location an error cites is a path the reader
	// can open. loader.Dir's confinement to a subtree is the library default
	// for an embedding that treats a schemaLocation hint as attacker-supplied;
	// this process reads the documents its own user named, with that user's
	// privileges.
	schema, err := parser.Parse(root,
		parser.WithResolver(loader.Dir(filepath.VolumeName(root)+string(filepath.Separator))),
		parser.WithLogger(log))
	if err != nil {
		// A schema verdict, not an IO fault: rootLocation already opened the
		// document. Errors reach stderr whatever -q says, so a script can
		// grep them.
		_, _ = fmt.Fprintln(stderr, violationLine(err))
		return exitInvalid
	}
	if quiet {
		return exitOK
	}
	if _, err := io.WriteString(stdout, summarize(location, schema)); err != nil {
		// The summary the user asked for never arrived; that is an IO fault,
		// not a verdict about the schema.
		return usageError(stderr, fmt.Sprintf("goxsd8: parse: writing the summary for %s: %v", location, err))
	}
	return exitOK
}

// rootLocation opens location to prove it is a readable file and returns its
// absolute path. Opening it here is what lets an unreadable argument be
// reported in the operating system's own words and charged exit 2, rather than
// reaching parser.Parse and coming back as an assembly error indistinguishable
// in shape from a verdict about a schema's content.
func rootLocation(location string) (string, error) {
	f, err := os.Open(location)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }() // read-only handle: a close error cannot change what was read
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("open %s: is a directory", location)
	}
	return filepath.Abs(location)
}

// violationLine renders one schema error as the contract's
// "<loc>: [<rule>] <message>". The rendering is (*xsderr.Error).Error()'s own
// and is never re-derived here; the error is unwrapped to reach it, because
// the parser wraps some rejections in assembly context that would otherwise
// prefix the line.
//
// An error carrying no *xsderr.Error prints its own message instead: a
// document whose root is not <xs:schema>, and the s4s-grammar class the spec
// catalogs no rule for (xsderr/doc.go), are real rejections with no rule ID to
// cite, and inventing one for them would read as a citation (STYLE E2).
func violationLine(err error) string {
	var e *xsderr.Error
	if errors.As(err, &e) {
		return e.Error()
	}
	return err.Error()
}

// usageError reports a usage or IO fault: the message, then the remedy, on
// stderr, and exit 2. Nothing reaches stdout, which carries verdicts alone.
func usageError(stderr io.Writer, msg string) int {
	// A failed stderr write cannot change the outcome — the exit code is 2
	// either way, and stderr is the only channel left to report it on.
	_, _ = fmt.Fprintf(stderr, "%s\n%s\n", msg, helpPointer)
	return exitUsage
}

// absentNamespace renders the ·absent· target namespace (§2.2), which a QName
// carries as an empty Space. It is spelled rather than left blank so that a
// no-namespace schema produces a namespace line like any other.
const absentNamespace = "(absent)"

// summarize renders the parse summary of one compiled schema: the namespaces
// its components are in, then the count of each §3.17.1 property, then their
// total. location is echoed as the argument spelled it.
//
// Every fact comes from *xsd.Schema's published enumeration accessors, each of
// which returns document order, and the buckets are visited in a fixed order,
// so the whole block is byte-identical across runs on identical input (STYLE
// D1/D2).
//
// The namespaces are the ones the compiled set's own components are in, in
// first-appearance order. That is not quite "the target namespace of each
// schema document": §3.17.1 gives the Schema component no {target namespace}
// property, so a document's targetNamespace is observable only through the
// components it declares, and one that declares nothing contributes no line.
func summarize(location string, s *xsd.Schema) string {
	buckets := []bucket{
		{"types", declaredNames(s.Types())},
		{"elements", declaredNames(s.Elements())},
		{"attributes", declaredNames(s.Attributes())},
		{"attribute groups", declaredNames(s.AttributeGroups())},
		{"model groups", declaredNames(s.ModelGroups())},
		{"notations", declaredNames(s.Notations())},
		{"identity constraints", declaredNames(s.IdentityConstraints())},
	}

	var b strings.Builder
	b.WriteString(location + "\n")
	for _, ns := range namespacesOf(buckets) {
		b.WriteString("  namespace: " + ns + "\n")
	}
	total := 0
	for _, bk := range buckets {
		total += len(bk.names)
		fmt.Fprintf(&b, "  %s: %d\n", bk.label, len(bk.names))
	}
	fmt.Fprintf(&b, "  components: %d\n", total)
	return b.String()
}

// bucket is one §3.17.1 property as the summary reports it: the label it
// prints under, and the names of the components a schema document declared
// into it.
type bucket struct {
	label string
	names []xsd.QName
}

// namespacesOf collects the distinct namespaces the named components are in,
// in first-appearance order over the buckets. The map decides membership only;
// the slice decides the order, so no output order comes from map iteration
// (STYLE D2).
//
// An anonymous component contributes nothing: its {name} is the zero QName, so
// it carries no target namespace to report — reading its empty Space as the
// ·absent· namespace would invent a namespace line for a schema that has none.
func namespacesOf(buckets []bucket) []string {
	var order []string
	seen := make(map[string]bool)
	for _, bk := range buckets {
		for _, name := range bk.names {
			if name == (xsd.QName{}) || seen[name.Space] {
				continue
			}
			seen[name.Space] = true
			order = append(order, namespaceLabel(name.Space))
		}
	}
	return order
}

// namespaceLabel renders one namespace name for the summary.
func namespaceLabel(space string) string {
	if space == "" {
		return absentNamespace
	}
	return space
}

// component is the shape every §3.17.1 property's members share: an expanded
// name and a source position. Both are on the eight component kinds already,
// so the summary reads all seven buckets it counts through one constraint
// rather than seven near-copies (STYLE T4).
type component interface {
	Name() xsd.QName
	Loc() xsderr.Loc
}

// declaredNames reduces one §3.17.1 property to the names of the components a
// SCHEMA DOCUMENT declared.
//
// The zero Loc is the discriminator, and xsd's package doc fixes its meaning:
// it "is the correct value for a component with no schema document behind it —
// parser.Produce's synthesized xs:anyType and package builtin's seeded
// built-in datatypes are the legitimate zero-Loc producers". Those are exactly
// what every parse seeds into {type definitions} before reading a document, so
// counting them would report the same fifty-odd types for every schema and
// bury what the schema itself declares.
func declaredNames[T component](components []T) []xsd.QName {
	names := make([]xsd.QName, 0, len(components))
	for _, c := range components {
		if c.Loc() == (xsderr.Loc{}) {
			continue
		}
		names = append(names, c.Name())
	}
	return names
}
