package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/internal/schemaloc"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/parser/xmltree"
	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/validate/xmlsrc"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// stdinArg is the instance argument naming standard input. It is an INSTANCE
// spelling alone: -schema - is not supported, because a schema document's
// location is the base URI its own relative <xs:include> and <xs:import>
// references resolve against (doc.go's argument vocabulary).
const stdinArg = "-"

// stringList accumulates the values of a repeatable flag in argument order.
// It is what makes `-schema a.xsd -schema b.xsd` the contract's spelling of a
// two-document schema set, and `-schema a.xsd b.xsd` a one-document set
// followed by an instance argument (doc.go).
type stringList []string

// String renders the accumulated values for the flag package's own reporting.
// It is not a spelling the flag reads back: Set appends one whole value per
// occurrence and never splits one.
func (l *stringList) String() string { return strings.Join(*l, " ") }

// Set records one occurrence. Every string is a location, so nothing here
// fails.
func (l *stringList) Set(v string) error {
	*l = append(*l, v)
	return nil
}

// sourceFormat is one instance encoding of the contract's -format vocabulary.
type sourceFormat string

const (
	formatXML  sourceFormat = "xml"
	formatJSON sourceFormat = "json"
	formatBER  sourceFormat = "ber"
)

// sourceFormats is the whole -format vocabulary, in the order the contract
// spells it and a diagnosis lists it. Both the flag's validation and the
// extension mapping read this one encoding rather than restating the tokens
// (STYLE D3/T4).
var sourceFormats = []sourceFormat{formatXML, formatJSON, formatBER}

// formatVocabulary renders the valid -format values for a diagnosis, in
// sourceFormats order.
func formatVocabulary() string {
	spellings := make([]string, 0, len(sourceFormats))
	for _, f := range sourceFormats {
		spellings = append(spellings, string(f))
	}
	return strings.Join(spellings, ", ")
}

// runValidate implements `goxsd8 validate`: it compiles the -schema arguments
// into ONE schema set and assesses every positional argument against it,
// writing each violation to stdout as the contract's "<loc>: [<rule>]
// <message>". args excludes the subcommand name itself.
//
// Every instance is assessed, in argument order: a violation in one never
// stops the next, so a script gets the whole report from one run. The exit
// code is the worst outcome over the instances (main.go's exit codes), the way
// runParse's is over its schemas.
//
// The flags are parsed here rather than by run, for the reason runParse states.
func runValidate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	// Discarding the flag set's output silences its own usage rendering, which
	// would print a second, narrower contract next to doc.go's; helpPointer
	// names the real one instead.
	flags.SetOutput(io.Discard)
	var schemas stringList
	flags.Var(&schemas, "schema", "a schema document of the set; repeat it for each")
	format := flags.String("format", "", "force the instance source format: xml, json or ber")
	noHints := flags.Bool("no-hints", false, "ignore the xsi:schemaLocation hints of XML instances")
	verbose := flags.Bool("v", false, "log assembly and assessment at debug level to stderr")
	// -q is defined because the contract makes it common to every subcommand,
	// and is read by nothing: validate writes no informational output, its
	// stdout carrying the violation report doc.go forbids -q to silence.
	_ = flags.Bool("q", false, "accepted; validate has no informational output to suppress")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return usageError(stderr, fmt.Sprintf(helpNotAFlagValueFmt, "validate"))
		}
		return usageError(stderr, fmt.Sprintf("goxsd8: validate: %v", err))
	}
	forced, err := forcedFormat(*format)
	if err != nil {
		return usageError(stderr, fmt.Sprintf("goxsd8: validate: %v", err))
	}
	instances := flags.Args()
	if len(schemas) == 0 {
		return usageError(stderr, "goxsd8: validate: no schema given")
	}
	if len(instances) == 0 {
		return usageError(stderr, "goxsd8: validate: no instance given")
	}

	// A nil logger selects the parser's and the engine's silent defaults, so -v
	// is the whole of the injection and quiet is the default (STYLE L1). -v is
	// all or nothing here too, for the reason parse.go's GAP(cmd) marker
	// records (#1185).
	var log *slog.Logger
	if *verbose {
		log = slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	docs := make([]schemaDoc, 0, len(schemas))
	for _, location := range schemas {
		path, err := rootLocation(location)
		if err != nil {
			// An argument that cannot be read is a usage/IO fault, never a
			// verdict about a schema — rootLocation's own reasoning, and
			// parse's contract for the same argument shape.
			return usageError(stderr, fmt.Sprintf("goxsd8: validate: %v", err))
		}
		doc, err := readSchemaDoc(path)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, violationLine(err))
			return exitSchema
		}
		docs = append(docs, doc)
	}

	// The backend the set is COMPILED with is the backend it is ASSESSED with:
	// validate.New requires them to be one value, or instance lexicals are read
	// in a value space no facet on the schema was ever checked against.
	backend := strict.New()
	base, err := compileSet(docs, backend, log)
	if err != nil {
		// Reported once, before any instance is read: with no schema set there
		// is no assessment to run, and one line beats the same line per
		// instance.
		_, _ = fmt.Fprintln(stderr, violationLine(err))
		return exitSchema
	}
	v, err := validate.New(base, backend, validate.WithLogger(log))
	if err != nil {
		return usageError(stderr, fmt.Sprintf("goxsd8: validate: %v", err))
	}

	job := validation{docs: docs, base: v, backend: backend, forced: forced, hints: !*noHints, log: log}
	code := exitOK
	for _, instance := range instances {
		code = max(code, job.one(instance, stdout, stderr))
	}
	return code
}

// validation is one invocation's resolved configuration: the schema set every
// instance is assessed against, and the policy decisions the flags settled for
// all of them at once.
type validation struct {
	// docs are the -schema documents, in argument order. They are kept beside
	// base because an instance whose hints augment the set is compiled from
	// them again, with the hinted documents appended.
	docs []schemaDoc
	// base assesses against the -schema documents alone.
	base *validate.Validator
	// backend is the value space docs were compiled in, and the one every
	// hint-augmented recompilation must reuse (validate.New).
	backend value.Backend
	// forced is the -format value, empty when the flag was not given and the
	// format is derived per instance from its extension.
	forced sourceFormat
	// hints reports whether xsi:schemaLocation hints augment the set, which
	// -no-hints turns off.
	hints bool
	// log is the debug logger -v installed, nil when it was not given.
	log *slog.Logger
}

// one assesses a single instance argument and reports its exit code.
func (vn *validation) one(instance string, stdout, stderr io.Writer) int {
	format, err := formatOf(instance, vn.forced)
	if err != nil {
		return usageError(stderr, fmt.Sprintf("goxsd8: validate: %v", err))
	}
	if format != formatXML {
		// A token the contract reserves and no milestone has built: validate/
		// jsonsrc and validate/bersrc carry a doc.go apiece and no Validate, so
		// the honest answer is the usage code and this diagnosis, never a
		// silent pass.
		return usageError(stderr, fmt.Sprintf("goxsd8: validate: %s: -format %s is reserved by the contract and not yet implemented; only %s instances are assessed today", instance, format, formatXML))
	}
	src, closeSrc, err := openInstance(instance)
	if err != nil {
		return usageError(stderr, fmt.Sprintf("goxsd8: validate: %v", err))
	}
	defer closeSrc()

	v, src, code := vn.validatorFor(instance, src, stderr)
	if code != exitOK {
		return code
	}

	result, err := xmlsrc.Validate(v, src, xmlsrc.WithURI(instance))
	if err != nil {
		// The assessment never ran: the document is malformed before its
		// document element, or holds none at all. That is a verdict about the
		// instance in the same rendering a violation gets, so it lands on
		// stdout with them and counts as invalid — nothing in the document was
		// shown valid.
		return reportLines(stdout, stderr, instance, []string{violationLine(err)})
	}
	return reportLines(stdout, stderr, instance, violationLines(result))
}

// validatorFor returns the validator instance is assessed against and the
// reader to assess it from, or a non-exitOK code that is the instance's whole
// outcome.
//
// The reader coming back is not always the one going in: reading an instance's
// hints consumes its prefix, so what the assessment reads is the replay
// instanceHints returns. The hinted documents augment THIS instance's set and
// no other's — a hint is a property of the document that carries it, and the
// -schema arguments are the only set every instance shares. Hints that will
// not compose with that set are unusable rather than fatal: they are reported
// against the instance and dropped, and exitSchema stays the answer to a
// -schema set that does not compile.
func (vn *validation) validatorFor(instance string, src io.Reader, stderr io.Writer) (*validate.Validator, io.Reader, int) {
	if !vn.hints {
		return vn.base, src, exitOK
	}
	base, err := filepath.Abs(instance)
	if err != nil {
		return nil, nil, usageError(stderr, fmt.Sprintf("goxsd8: validate: resolving %s: %v", instance, err))
	}
	found, replay := instanceHints(instance, base, src)
	if len(found) == 0 {
		return vn.base, replay, exitOK
	}
	augmented, err := compileSet(append(slices.Clone(vn.docs), found...), vn.backend, vn.log)
	if err != nil {
		// A set that stops compiling only once THIS instance's hints are folded
		// in is a fault of the instance, not of the -schema set the invocation
		// named: exitSchema would send a script to a schema set that compiles,
		// and the fault is charged inside the wrapper root, a document the
		// reader cannot open (STYLE E3). §4.3.2 clause 3 obliges a processor to
		// dereference no hint at all, so the honest degradation is the one a
		// hint naming a MISSING document already gets — the -schema set alone
		// decides this instance, which charges cvc-assess-elt where it declares
		// nothing for the root — with the hints reported as unusable rather
		// than silently dropped.
		_, _ = fmt.Fprintf(stderr, "goxsd8: validate: %s: ignoring its schema location hints, which do not compose with the -schema set: %s\n", instance, hintFault(err))
		return vn.base, replay, exitOK
	}
	v, err := validate.New(augmented, vn.backend, validate.WithLogger(vn.log))
	if err != nil {
		return nil, nil, usageError(stderr, fmt.Sprintf("goxsd8: validate: %v", err))
	}
	return v, replay, exitOK
}

// hintFault renders a compile fault of a hint-augmented set for the diagnosis
// that names the instance which carried the hints.
//
// A fault the wrapper root itself carries — src-import clause 3.1 against a
// mis-paired hint, src-include clause 2 against a hinted document the wrapper
// cannot compose — is rendered without its location: schemaSetLocation is this
// process's own synthesis and names no document the reader can open, so the
// rule and the message, which name the hinted document, are the whole of what
// is left to say. A fault charged at a real document keeps its location, which
// is the file the reader must edit.
func hintFault(err error) string {
	var e *xsderr.Error
	if errors.As(err, &e) && e.Loc.URI == schemaSetLocation {
		return fmt.Sprintf("[%s] %s", e.Rule, e.Msg)
	}
	return violationLine(err)
}

// violationLines renders one assessment's report, in document order: every
// violation it charged, then the source fault that stopped the walk if one
// did. A stopped walk is not a violation, but it is a verdict about the
// instance in the same shape — the document was not read to its end, so
// nothing past the fault was assessed — and it is rendered beside them rather
// than dropped.
func violationLines(result *validate.Result) []string {
	violations := result.Violations()
	lines := make([]string, 0, len(violations)+1)
	for _, v := range violations {
		lines = append(lines, violationLine(v))
	}
	if result.Err() != nil {
		lines = append(lines, violationLine(result.Err()))
	}
	return lines
}

// reportLines writes one instance's violation report to stdout and reports the
// instance's exit code: no line is a clean assessment, and any line is a
// verdict of invalid.
//
// A failed write is charged the usage/IO code, on parseOne's reasoning: the
// report the user asked for never arrived, which is a fault in this run rather
// than a verdict about the document.
func reportLines(stdout, stderr io.Writer, instance string, lines []string) int {
	if len(lines) == 0 {
		return exitOK
	}
	if _, err := io.WriteString(stdout, strings.Join(lines, "\n")+"\n"); err != nil {
		return usageError(stderr, fmt.Sprintf("goxsd8: validate: writing the violations of %s: %v", instance, err))
	}
	return exitInvalid
}

// openInstance opens one instance argument for reading and returns the reader
// together with the close its caller owes. stdinArg names standard input,
// which this process does not own and therefore does not close.
func openInstance(instance string) (io.Reader, func(), error) {
	if instance == stdinArg {
		return os.Stdin, func() {}, nil
	}
	f, err := os.Open(instance)
	if err != nil {
		return nil, nil, err
	}
	// A close error on a read-only handle cannot change what was read.
	return f, func() { _ = f.Close() }, nil
}

// forcedFormat reads the -format value. The empty string is the flag's absence,
// not a token, and leaves the format to each instance's own extension.
//
// Matching is case-sensitive, as the contract states: -format XML is an
// unrecognized token and not a loudly spelled one.
func forcedFormat(token string) (sourceFormat, error) {
	if token == "" {
		return "", nil
	}
	if slices.Contains(sourceFormats, sourceFormat(token)) {
		return sourceFormat(token), nil
	}
	return "", fmt.Errorf("-format %q is not a source format; the values are %s", token, formatVocabulary())
}

// formatOf reports the source format of one instance argument: the -format
// value where the flag was given, and otherwise the format its extension names.
//
// An argument whose extension names none of them — including stdinArg, which
// has no extension at all — is a usage error rather than a guess, so that no
// document is ever read in a format nothing in the invocation asked for.
func formatOf(instance string, forced sourceFormat) (sourceFormat, error) {
	if forced != "" {
		return forced, nil
	}
	if instance == stdinArg {
		return "", fmt.Errorf("%s names standard input, which carries no extension to name a source format; pass -format %s", stdinArg, formatVocabulary())
	}
	ext := filepath.Ext(instance)
	for _, f := range sourceFormats {
		if ext == "."+string(f) {
			return f, nil
		}
	}
	return "", fmt.Errorf("%s: the extension %q names no source format; pass -format %s", instance, ext, formatVocabulary())
}

// schemaDoc is one document of the schema set as the wrapper root names it: an
// absolute filesystem path, and the target namespace under which its
// components enter the set.
type schemaDoc struct {
	// location is the document's absolute path, which the wrapper root carries
	// as a schemaLocation and the filesystem resolver serves.
	location string
	// namespace is the namespace the document's components are minted in,
	// ·absent· encoded as "". For a -schema argument it is the document's own
	// targetNamespace; for an xsi:schemaLocation hint it is the namespace the
	// hint PAIRED with the location, which src-import clause 3.1 then requires
	// the document to agree with.
	namespace string
}

// readSchemaDoc reads the schema document at path — already proved readable by
// rootLocation — for the one fact the wrapper root needs about it: its own
// targetNamespace, which decides whether the wrapper <import>s it or
// <include>s it. A failure here is therefore about the document's content
// rather than about the argument.
func readSchemaDoc(path string) (schemaDoc, error) {
	f, err := os.Open(path)
	if err != nil {
		return schemaDoc{}, fmt.Errorf("reading schema document %s: %w", path, err)
	}
	defer func() { _ = f.Close() }() // read-only handle: a close error cannot change what was read
	doc, err := parser.ReadDocument(path, f)
	if err != nil {
		return schemaDoc{}, err
	}
	// An ·absent· targetNamespace and an empty one are the same state here, as
	// they are everywhere else in the module.
	namespace, _ := doc.Root().Attr("targetNamespace")
	return schemaDoc{location: path, namespace: namespace}, nil
}

// schemaSetLocation is the location the synthesized wrapper root is served
// under. It is a bare name with no directory part, so that internal/schemaloc's
// resolution of the absolute schemaLocations inside it leaves them absolute.
const schemaSetLocation = "goxsd8-schema-set.xsd"

// compileSet assembles docs into ONE schema set and returns it finalized.
//
// There is no multi-root entry point to call — parser.Parse takes a single root
// location, which is §4.2.1's schema(D) — so the set is expressed as a schema
// document in the spec's own terms: a wrapper <schema> with no targetNamespace
// of its own that <import>s or <include>s each document, served in memory and
// composed by the ordinary assembly. Composing the set this way rather than
// merging several finalized schemas is what keeps every cross-document rule the
// parser already enforces — src-import clause 3, sch-props-correct clause 2,
// src-resolve at finalize — enforced over the CLI's set too.
func compileSet(docs []schemaDoc, backend value.Backend, log *slog.Logger) (*xsd.Schema, error) {
	// The wrapper's schemaLocations are absolute paths, so the filesystem
	// resolver is rooted at the filesystem root: parseOne's reasoning, that this
	// process reads the documents its own user named, with that user's
	// privileges. docs is never empty — runValidate requires a -schema — so the
	// first document names the volume the whole set resolves under, which on a
	// platform with more than one is the volume they must share.
	resolver := loader.Chain(
		loader.Map(map[string]string{schemaSetLocation: schemaSetSource(docs)}),
		loader.Dir(filesystemRoot(docs[0].location)),
	)
	return parser.Parse(schemaSetLocation,
		parser.WithResolver(resolver),
		parser.WithBackend(backend),
		parser.WithLogger(log))
}

// schemaSetSource renders the wrapper root document for docs, in argument
// order, so that one invocation always composes its set the same way (STYLE
// D1).
//
// A document with a target namespace of its own is <import>ed, which brings its
// components in unchanged (§4.2.6). One with none is <include>d instead: the
// wrapper has no targetNamespace either, which is src-include clause 2.2, and
// src-import clause 1.2 forbids a namespace-less <import> from a wrapper that
// has no target namespace to declare it in.
func schemaSetSource(docs []schemaDoc) string {
	var b strings.Builder
	b.WriteString(`<xs:schema xmlns:xs="` + xsd.XMLSchemaNS + `">`)
	for _, d := range docs {
		if d.namespace == "" {
			b.WriteString(`<xs:include schemaLocation="` + escapeAttr(d.location) + `"/>`)
			continue
		}
		b.WriteString(`<xs:import namespace="` + escapeAttr(d.namespace) + `" schemaLocation="` + escapeAttr(d.location) + `"/>`)
	}
	b.WriteString(`</xs:schema>`)
	return b.String()
}

// escapeAttr renders s as an XML attribute value's content. A filesystem path
// and a namespace name are both arbitrary strings, and either can carry a
// character the wrapper's markup would otherwise take as its own.
func escapeAttr(s string) string {
	var b strings.Builder
	// strings.Builder writes never fail, so the only error channel here cannot
	// carry one.
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// filesystemRoot is the resolver root under which an absolute path resolves:
// the filesystem root of the volume location names. Every location the CLI
// hands a resolver is absolute, so this is the root that serves all of them.
func filesystemRoot(location string) string {
	return filepath.VolumeName(location) + string(filepath.Separator)
}

// instanceHints reads the schema location hints off the DOCUMENT ELEMENT of
// the XML instance in r, resolved against base, and returns them with a reader
// replaying every byte the scan consumed.
//
// The replay is what lets a hint be read from standard input, which cannot be
// reopened, and it is used for a file too so that both are read exactly once
// and identically.
//
// Reading only the document element is a policy, not a shortfall: §4.3.2
// clause 5 admits a hint on any element and makes its effect global either way,
// clause 3 requires no processor to dereference any of them, and doc.go states
// the scope this one follows.
func instanceHints(uri, base string, r io.Reader) ([]schemaDoc, io.Reader) {
	var consumed bytes.Buffer
	reader := xmltree.NewReader(uri, io.TeeReader(r, &consumed))
	replay := func() io.Reader { return io.MultiReader(&consumed, r) }
	for {
		node, err := reader.Token()
		if err != nil {
			// The prefix is malformed, or ends before a document element:
			// there are no hints to read, and no diagnosis to make here. The
			// assessment reads the same bytes off the replay reader and charges
			// the fault at its own location under its own rule, so reporting it
			// here too would be two lines for one fault.
			return nil, replay()
		}
		start, ok := node.(*xmltree.StartElement)
		if !ok {
			continue
		}
		return hintsOf(start, base), replay()
	}
}

// hintsOf reads the §2.7 schema location hints off one element, in the order
// its attributes carry them (STYLE D1/D2), resolving each location against
// base through internal/schemaloc — the same resolution the parser gives an
// <xs:include>, so that a hint and a directive naming one document agree on
// which document that is.
//
// xsi:schemaLocation pairs a namespace with a location; xsi:noNamespaceSchema-
// Location names a location whose document has no target namespace, which the
// wrapper root <include>s.
func hintsOf(start *xmltree.StartElement, base string) []schemaDoc {
	var hints []schemaDoc
	for _, a := range start.Attributes() {
		if a.Name().Space() != xsd.XMLSchemaInstanceNS {
			continue
		}
		fields := strings.Fields(a.Value())
		switch a.Name().Local() {
		case "noNamespaceSchemaLocation":
			for _, location := range fields {
				hints = append(hints, schemaDoc{location: schemaloc.Resolve(base, location)})
			}
		case "schemaLocation":
			// An odd trailing member pairs with nothing and names no document,
			// so it is dropped rather than resolved against an empty namespace.
			for i := 0; i+1 < len(fields); i += 2 {
				hints = append(hints, schemaDoc{
					namespace: fields[i],
					location:  schemaloc.Resolve(base, fields[i+1]),
				})
			}
		}
	}
	return hints
}
