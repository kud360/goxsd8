package parser

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"path"
	"strings"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// Option configures [Parse]. The zero set of options is a complete, usable
// configuration: options only replace defaults, never combine into an invalid
// one (STYLE T1).
type Option func(*config)

// config is the resolved [Parse] configuration. It is always valid — every
// field is non-nil from construction onward, since each Option either installs
// a usable replacement or panics.
type config struct {
	resolver loader.Resolver
	backend  value.Backend
	log      *slog.Logger
}

// newConfig applies opts over the defaults: schema documents are resolved as
// paths under the current directory, values use the spec-exact strict backend,
// and nothing is logged.
func newConfig(opts []Option) config {
	cfg := config{
		resolver: loader.Dir("."),
		backend:  strict.New(),
		log:      slog.New(slog.DiscardHandler),
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// WithResolver sets the [loader.Resolver] through which every schema document —
// the root location and every <include>d location — is fetched (§4.3.2: how
// schema definitions are located is the processor's, hence the caller's,
// business).
//
// It panics if r is nil: a nil resolver is a caller bug, not a schema-validity
// condition, so it is guarded exactly like [xsd.SchemaBuilder.AddType]'s nil
// component rather than deferred into a nil dereference mid-parse.
func WithResolver(r loader.Resolver) Option {
	if r == nil {
		panic("parser: WithResolver: nil loader.Resolver")
	}
	return func(c *config) { c.resolver = r }
}

// WithBackend sets the [value.Backend] supplying the builtin datatypes' value
// spaces, replacing the default spec-exact builtin/strict backend.
//
// It panics if b is nil, on the same grounds as [WithResolver].
func WithBackend(b value.Backend) Option {
	if b == nil {
		panic("parser: WithBackend: nil value.Backend")
	}
	return func(c *config) { c.backend = b }
}

// WithLogger sets the logger the parser reports assembly progress on, under the
// "parser" group at debug level. A nil logger selects the silent default
// ([slog.DiscardHandler]), so parsing is quiet unless a logger is asked for
// (STYLE L1).
func WithLogger(l *slog.Logger) Option {
	if l == nil {
		l = slog.New(slog.DiscardHandler)
	}
	return func(c *config) { c.log = l }
}

// Parse assembles the schema rooted at location and returns it finalized.
//
// It implements §4.2.1's schema(D): the components of the root document plus,
// transitively, those of every schema document reached through an <xs:include>
// child (§4.2.3) — including chameleon inclusion, where a document with no
// targetNamespace of its own contributes its components, and its unqualified
// QName references, to the including namespace (§4.2.3 clause 3.2, §F.1).
// Documents are read once each, keyed by resolved location, so a diamond or a
// (spec-legal) cycle of <include>s contributes its components once and does not
// trip sch-props-correct (§3.17.6.1) clause 2.
//
// <import>, <redefine> and <override> are NOT yet followed: like every other
// not-yet-produced top-level representation they are skipped, not rejected
// (§3.1.2), so a schema needing them assembles short rather than wrongly.
//
// Errors are schema-validity verdicts as *[xsderr.Error] values (src-include,
// src-resolve, sch-props-correct, …) carrying the offending construct's
// location; a failure to resolve the ROOT location, by contrast, is a plain I/O
// error, since the caller named a document that must exist. Like [Produce],
// Parse returns only the first error.
func Parse(location string, opts ...Option) (*xsd.Schema, error) {
	cfg := newConfig(opts)
	root, resolved, err := readRootDocument(cfg.resolver, location)
	if err != nil {
		return nil, fmt.Errorf("parser: reading root schema document %q: %w", location, err)
	}
	if !root.IsSchema() {
		return nil, fmt.Errorf("parser: Parse requires a <schema> document root at %q, got %s", location, root.Root().Name().Local())
	}
	a := newAssembly(root, resolved, cfg)
	if err := a.discover(root); err != nil {
		return nil, err
	}
	return a.compile(cfg.backend)
}

// assembly is the multi-document build context for one [Parse] call: the
// <include> closure of a root schema document, and the single builder every one
// of those documents produces into.
type assembly struct {
	resolver loader.Resolver
	log      *slog.Logger

	// tns is the assembly's effective target namespace — the ROOT document's
	// targetNamespace. Every document of the closure ends up in it, by induction
	// over src-include clause 2: 2.1 and 2.2 require the included document to
	// agree, and 2.3 coerces it (§F.1). That is why one namespace, and one
	// location-keyed dedup index, suffice for the whole assembly (STYLE D3).
	tns string

	// loaded indexes the RESOLVED locations already read (§4.2.3: "If two
	// <include> elements specify the same schema location (after resolving
	// relative URI references) then they refer to the same schema document"). It
	// is a LOAD-ONCE index, not a cycle guard (STYLE D4): <include> cycles are
	// spec-legal — §4.2.3 states the same schema corresponds to every document in
	// the cycle — and are not detected, merely loaded once.
	loaded map[string]struct{}

	// docs holds every discovered document in discovery order: depth-first,
	// pre-order, over each <schema>'s <include> children in document order. That
	// order is the order components enter the builder, so it is user-visible in
	// sch-props-correct duplicate reports (STYLE D1/D2).
	docs []*Document
}

// newAssembly returns the assembly for a root schema document already read from
// rootResolved, with its effective target namespace fixed at construction
// (STYLE T1). root must be a <schema> document.
func newAssembly(root *Document, rootResolved string, cfg config) *assembly {
	return &assembly{
		resolver: cfg.resolver,
		log:      cfg.log,
		tns:      attrOr(root.Root(), "targetNamespace"),
		loaded:   map[string]struct{}{rootResolved: {}},
	}
}

// discover appends doc to the assembly and then, in document order, follows each
// of its top-level <include> children depth-first (§4.2.3: schema(D1) contains
// immed(D1) plus the components of schema(D2) for each <include>d D2).
func (a *assembly) discover(doc *Document) error {
	a.docs = append(a.docs, doc)
	for _, child := range doc.Root().Children() {
		el, ok := child.(*Element)
		if !ok || !isXSD(el, "include") {
			continue
		}
		if err := a.include(el); err != nil {
			return err
		}
	}
	return nil
}

// include follows one <include> element: it resolves the schemaLocation, reads
// the document it names, enforces src-include (§4.2.3) clauses 1 and 2 on it,
// and recurses into its own <include>s.
func (a *assembly) include(el *Element) error {
	hint, ok := attrValue(el, "schemaLocation")
	if !ok {
		// schemaLocation is REQUIRED on <include> by the schema for schema
		// documents, and src-include governs only what its actual value resolves
		// to. A missing required attribute is therefore a grammar fault with no
		// dedicated Schema Representation Constraint, reported as a plain error —
		// the same treatment a <group> with no ref gets in produce_complex.go.
		return fmt.Errorf("parser: <include> at %s has no schemaLocation attribute, which the schema for schema documents requires", el.Loc())
	}
	// §4.3.2 clause 4: the location is a URI REFERENCE, resolved against the base
	// URI in scope at the <include> element itself (xml:base-aware), not against
	// the document URI.
	requested := resolveSchemaLocation(el.BaseURI(), hint)

	d2, err := a.fetch(requested, el)
	if err != nil {
		return err
	}
	if d2 == nil {
		return nil
	}
	if !d2.IsSchema() {
		return xsderr.New(ruleSrcInclude, el.Loc(),
			"schemaLocation %q resolves to a <%s> document element, but src-include clause 1 requires it to resolve to a <schema> element information item", hint, d2.Root().Name().Local())
	}
	// D2 is compared against the ASSEMBLY's namespace rather than the raw
	// targetNamespace of the document physically containing this <include>. Under
	// chameleon inclusion the §F.1 transformation is applied to the whole
	// including document before its own schema is computed, so its <include>s are
	// evaluated as the coerced document's: §4.2.3's recursion note spells out that
	// "if A includes B and B includes C … the effect is as if A included B' and B'
	// included C'", with B' carrying A's targetNamespace.
	own, hasOwn := attrValue(d2.Root(), "targetNamespace")
	if hasOwn && own != a.tns {
		// Clause 2.1 (c-normi) fails (the two targetNamespaces differ, or the
		// including document has none), 2.2 (c-normi2) fails (D2 has one), 2.3
		// (c-chami) fails (D2 has one), and 2.4 fails (D2 exists).
		return xsderr.New(ruleSrcInclude, el.Loc(),
			"<include>d schema document %q has targetNamespace %q, but src-include clause 2 requires it to be identical to the including schema's %q or absent", requested, own, a.tns)
	}
	// Clause 2.1, 2.2, or — when D2 declares no targetNamespace and the assembly
	// has one — 2.3, whose §F.1 coercion is applied at production time.
	return a.discover(d2)
}

// fetch resolves requested through the assembly's resolver and reads the schema
// document it names. It returns a nil *Document, and no error, for the two
// outcomes that mean "perform no inclusion here":
//
//   - the location does not resolve at all — §4.2.3: "It is not an error for the
//     actual value of the schemaLocation attribute to fail to resolve at all, in
//     which case the corresponding inclusion must not be performed" (src-include
//     clause 2.4);
//   - the resolved location was already loaded — the same schema location names
//     the same schema document (§4.2.3), which therefore contributes its
//     components exactly once.
//
// Any OTHER resolver failure is real: a permission or transport error is not
// silently downgraded to "absent".
func (a *assembly) fetch(requested string, el *Element) (*Document, error) {
	rc, resolved, err := a.resolver.Resolve(a.tns, requested)
	if errors.Is(err, loader.ErrNotFound) {
		a.log.Debug("include skipped: schemaLocation does not resolve",
			"rule", string(ruleSrcInclude), "location", requested, "at", el.Loc().String())
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("parser: resolving <include> schemaLocation %q at %s: %w", requested, el.Loc(), err)
	}
	// A read-only reader: a close failure cannot change what was already read, so
	// it cannot affect the parse verdict (STYLE S3).
	defer func() { _ = rc.Close() }()

	if _, done := a.loaded[resolved]; done {
		a.log.Debug("include already loaded", "location", requested, "resolved", resolved, "at", el.Loc().String())
		return nil, nil
	}
	a.loaded[resolved] = struct{}{}

	doc, err := ReadDocument(requested, rc)
	if err != nil {
		// Clause 1.1 requires the resolved resource to correspond to a <schema>
		// item "in a well-formed information set"; a well-formedness fault in an
		// <include>d document is thus an inclusion violation, not a bare XML error.
		return nil, xsderr.Wrap(ruleSrcInclude, el.Loc(),
			fmt.Errorf("<include>d schema document %q is not well-formed, but src-include clause 1.1 requires a well-formed information set: %w", requested, err))
	}
	return doc, nil
}

// compile produces every discovered document into ONE builder and finalizes it.
// Every document's pre-scan runs before any document is produced, so a base= or
// <attributeGroup ref> in one document reaches a definition contributed by
// another (§4.2.3 clause 3.1.2, c-incl-incl); the builtins are seeded once, by
// newSymbols, for the whole assembly.
func (a *assembly) compile(backend value.Backend) (*xsd.Schema, error) {
	builder := xsd.NewSchemaBuilder()
	sym, err := newSymbols(builder, backend)
	if err != nil {
		return nil, err
	}
	producers := make([]*producer, 0, len(a.docs))
	for _, doc := range a.docs {
		p := newProducer(doc, a.tns, builder, sym)
		p.prescan()
		producers = append(producers, p)
	}
	for i, p := range producers {
		a.log.Debug("producing schema document",
			"uri", a.docs[i].URI(), "targetNamespace", a.tns, "chameleon", p.chameleon())
		if err := p.run(); err != nil {
			return nil, err
		}
	}
	return builder.Finalize()
}

// resolveSchemaLocation resolves a schemaLocation URI reference against the base
// URI in scope at the element carrying it (§4.3.2 clause 4). The result is what
// the [loader.Resolver] is asked for, and — for a document that has its own
// <include>s — the base those are in turn resolved against, so it must stay in
// the same space as the location the caller handed [Parse].
//
// An absolute reference wins outright. Otherwise, when the base is itself an
// absolute URI, standard RFC 3986 reference resolution applies. When it is not —
// a bare relative path such as "schemas/main.xsd", which is exactly what a
// resolver rooted at a directory or an in-memory map is keyed by —
// [net/url.URL.ResolveReference] would root the result at "/" and turn a
// resolver-relative location into an absolute one that no such resolver can
// serve; path-wise resolution against the base's directory is used instead.
func resolveSchemaLocation(base, location string) string {
	if location == "" {
		return base
	}
	ref, refErr := url.Parse(location)
	if refErr == nil && ref.IsAbs() {
		return location
	}
	b, baseErr := url.Parse(base)
	if refErr == nil && baseErr == nil && b.IsAbs() {
		return b.ResolveReference(ref).String()
	}
	if strings.HasPrefix(location, "/") {
		return path.Clean(location)
	}
	return path.Join(path.Dir(base), location)
}

// readRootDocument resolves the location [Parse] was handed and reads the schema
// document it names, returning the document and its RESOLVED location — the
// load-once dedup key the [loader.Resolver] contract defines, which seeds the
// assembly's index so an <include> pointing back at the root does not re-load
// it. It is the ROOT counterpart of assembly.fetch and deliberately does not
// share its error policy: every failure here is real, because the caller named a
// document that must exist, whereas §4.2.3 makes an unresolvable <include>
// location a no-op.
//
// The resolver is asked under the no-namespace sentinel "": the document's own
// targetNamespace is only knowable after it has been read, so there is nothing
// truthful to pass.
func readRootDocument(r loader.Resolver, location string) (*Document, string, error) {
	rc, resolved, err := r.Resolve("", location)
	if err != nil {
		return nil, "", err
	}
	// Read-only reader; a close failure cannot change what was read (STYLE S3).
	defer func() { _ = rc.Close() }()
	doc, err := ReadDocument(location, rc)
	if err != nil {
		return nil, "", err
	}
	return doc, resolved, nil
}
