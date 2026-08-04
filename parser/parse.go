package parser

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/internal/schemaloc"
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

// Parse assembles the schema rooted at location and returns it finalized. It is
// [ParseReport] without the report, for the callers that only want the schema.
func Parse(location string, opts ...Option) (*xsd.Schema, error) {
	schema, _, err := ParseReport(location, opts...)
	return schema, err
}

// ParseReport assembles the schema rooted at location, returns it finalized,
// and reports which schema documents went into it — every document it read, in
// discovery order, and every ·inter-schema-document reference· it could not
// follow to one (see [AssemblyReport]). The report is never nil and is
// populated as far as assembly got even when an error is returned.
//
// It implements §4.2.1's schema(D): the components of the root document plus,
// transitively, those of every schema document reached through an <xs:include>
// child (§4.2.3) — including chameleon inclusion, where a document with no
// targetNamespace of its own contributes its components, and its unqualified
// QName references, to the including namespace (§4.2.3 clause 3.2, §F.1) — or
// through an <xs:import> child (§4.2.6), which brings a DIFFERENT namespace's
// components in unchanged: import never coerces, so an imported document's
// components stay in its own target namespace and a cross-namespace reference
// resolves against them at finalize. Documents are read once each, keyed by
// resolved location AND the namespace they were reached under, so a diamond or a
// (spec-legal) cycle of <include>s, or a namespace imported repeatedly,
// contributes its components once and does not trip sch-props-correct
// (§3.17.6.1) clause 2.
//
// An <xs:override> child (§4.2.5) is followed too: the document it names is
// composed exactly as an <include>d one — §4.2.5 clause 3.1.2 defines an override
// AS an <include> of the transformed document — after ·override pre-processing·
// substitutes the override's own children for the identically-named source
// declarations of that document and of every document it in turn <include>s or
// <override>s (§F.2, parser/override.go).
//
// <redefine> is NOT yet followed: like every other not-yet-produced top-level
// representation it is skipped, not rejected (§3.1.2), so a schema needing it
// assembles short rather than wrongly. Since a skipped <redefine> leaves no trace
// in the returned schema, [WithLogger] at debug level is how a caller discovers
// one: each skipped child <xs:redefine> element is reported there with its
// location.
//
// Errors are schema-validity verdicts as *[xsderr.Error] values (src-include,
// src-import, src-import-noselfimport, src-override, src-resolve,
// sch-props-correct, …)
// carrying the offending construct's location; a failure to resolve the ROOT
// location, by contrast, is a plain I/O error, since the caller named a document
// that must exist. Like [Produce], ParseReport returns only the first error.
func ParseReport(location string, opts ...Option) (*xsd.Schema, *AssemblyReport, error) {
	cfg := newConfig(opts)
	root, resolved, err := readRootDocument(cfg.resolver, location)
	if err != nil {
		// Nothing was assembled: an empty report, not a nil one, so every caller
		// reads the same shape on every path.
		return nil, &AssemblyReport{}, fmt.Errorf("parser: reading root schema document %q: %w", location, err)
	}
	if !root.IsSchema() {
		return nil, &AssemblyReport{}, fmt.Errorf("parser: assembling a schema requires a <schema> document root at %q, got %s", location, root.Root().Name().Local())
	}
	// The root document's effective target namespace is its own: there is no
	// including document to borrow one from (§4.2.3 clause 2.3 needs one).
	rootTNS := attrOr(root.Root(), "targetNamespace")
	a := newAssembly(resolved, rootTNS, cfg)
	// The root is discovered under the nil (identity) override set: no <override>
	// element points at it, so nothing substitutes for its own declarations.
	if err := a.discover(root, resolved, rootTNS, nil); err != nil {
		return nil, a.report(), err
	}
	schema, err := a.compile(cfg.backend)
	return schema, a.report(), err
}

// discovered is one schema document of an assembly paired with the EFFECTIVE
// target namespace its components are minted in. That namespace is established
// at discovery time and differs per document, so it cannot be derived from any
// other assembly state and is not redundant with the document's own
// targetNamespace attribute (STYLE D3):
//
//   - the root document, and any <include>d document that declares a
//     targetNamespace, is minted in its own;
//   - a chameleon <include>d document (none of its own) is minted in the
//     INCLUDER's effective namespace (§4.2.3 clause 2.3, §F.1);
//   - an <import>ed document is ALWAYS minted in its own, coerced never — §F.1's
//     transformation belongs to src-include clause 2.3 alone, so a bare
//     <import> of a no-namespace document leaves its components in no namespace
//     even when the importing document has one.
//
// Do not collapse this back into a single assembly-wide namespace: that field
// was only ever true of an include-only closure.
type discovered struct {
	doc *Document
	tns string

	// resolved is the location the [loader.Resolver] reported for this document —
	// the load-once identity half of docKey, and what [AssemblyReport] hands a
	// consumer as [AssembledDocument.Location]. It is not derivable from doc:
	// Document.URI is the location as REQUESTED, before the resolver
	// canonicalized it (loader.Dir folds on-disk case), so the two differ exactly
	// where dedup depends on them (STYLE D3).
	resolved string

	// ov is the ·override pre-processing· in force over THIS document's own top
	// level (§4.2.5, §F.2), nil when it was reached plainly. Like tns it is
	// established at discovery time and is a property of the PATH the document was
	// reached by, not of the document (STYLE D3): the same document can be reached
	// once plainly and once under an <override>, and each reading contributes its
	// own component set — which is precisely why §4.2.5 says such a schema "will
	// have duplicate and conflicting versions of some components".
	ov *overrideSet
}

// docKey is the load-once identity of one discovered document: its RESOLVED
// location together with the namespace it was reached under and the ·override
// pre-processing· applied to it. The location alone does not suffice once
// <import> exists, because the SAME document can legitimately be reached twice
// under two different namespaces — once as a chameleon <include> coerced into the
// includer's namespace, once as a bare <import> staying in no namespace — and
// each of those contributes its own component set. The override half is there for
// the same reason: one document overridden two different ways yields two
// different component sets (§4.2.5), while overriding it the same way twice, or
// reaching it again around an <include>/<override> cycle, must contribute its
// components once (§4.2.5's note on sch-props-correct clause 2). In an
// include-only closure both extra components are constant, so neither changes any
// include behavior.
type docKey struct {
	resolved  string
	namespace string
	override  string
}

// assembly is the multi-document build context for one [Parse] call: the
// <include>/<import> closure of a root schema document, and the single builder
// every one of those documents produces into.
type assembly struct {
	resolver loader.Resolver
	log      *slog.Logger

	// loaded indexes the documents already read (§4.2.3: "If two <include>
	// elements specify the same schema location (after resolving relative URI
	// references) then they refer to the same schema document"; §4.2.6.2's note
	// wants repeated <import>s of one document not to trip sch-props-correct
	// clause 2 either). It is a LOAD-ONCE index, not a cycle guard (STYLE D4):
	// <include> cycles are spec-legal — §4.2.3 states the same schema corresponds
	// to every document in the cycle — and are not detected, merely loaded once.
	loaded map[docKey]struct{}

	// docs holds every discovered document in discovery order: depth-first,
	// pre-order, over each <schema>'s <include>, <override> and <import> children
	// in a single document-order pass. That order is the order components enter
	// the builder,
	// so it is user-visible in sch-props-correct duplicate reports (STYLE D1/D2)
	// and must not be made to depend on which KIND of directive was seen.
	docs []discovered

	// unfollowed holds every ·inter-schema-document reference· (§4.2.1) that
	// yielded no document, in encounter order, for [AssemblyReport.Unfollowed].
	// It is not redundant with docs (STYLE D3): a directive that named no
	// document contributes nothing TO docs, which is exactly why it needs its own
	// record.
	unfollowed []UnfollowedDirective
}

// report renders the assembly's discovery state as the [AssemblyReport]
// [ParseReport] returns. It is built from the docs SLICE, in append order, never
// from the loaded index — a map iteration would make the reported order
// nondeterministic (STYLE D2).
func (a *assembly) report() *AssemblyReport {
	docs := make([]AssembledDocument, 0, len(a.docs))
	for _, d := range a.docs {
		docs = append(docs, AssembledDocument{Doc: d.doc, Location: d.resolved})
	}
	return &AssemblyReport{documents: docs, unfollowed: a.unfollowed}
}

// unfollowedAt records that the directive at el yielded no document, for reason.
func (a *assembly) unfollowedAt(el *Element, reason UnfollowedReason) {
	a.unfollowed = append(a.unfollowed, UnfollowedDirective{Reason: reason, At: el.Loc()})
}

// newAssembly returns the assembly for a root schema document already read from
// rootResolved under rootTNS, its own effective target namespace, seeding the
// load-once index with it (STYLE T1).
func newAssembly(rootResolved, rootTNS string, cfg config) *assembly {
	return &assembly{
		resolver: cfg.resolver,
		log:      cfg.log,
		loaded:   map[docKey]struct{}{{resolved: rootResolved, namespace: rootTNS}: {}},
	}
}

// discover appends doc to the assembly under resolved — the location the
// resolver reported for it — tns — its effective target
// namespace — and ov — the ·override pre-processing· in force over it — and then,
// in ONE document-order pass, follows each of its top-level <include> (§4.2.3:
// schema(D1) contains immed(D1) plus the components of schema(D2) for each
// <include>d D2), <override> (§4.2.5: plus the components of
// schema(override(E,Dold))) and <import> (§4.2.6.2: plus a set of components
// identical to those of each imported S2) children depth-first. The three
// directive kinds share the pass so that component entry order stays document
// order rather than becoming directive-kind-dependent (STYLE D1).
func (a *assembly) discover(doc *Document, resolved, tns string, ov *overrideSet) error {
	a.docs = append(a.docs, discovered{doc: doc, tns: tns, resolved: resolved, ov: ov})
	for _, child := range doc.Root().Children() {
		el, ok := child.(*Element)
		if !ok {
			continue
		}
		switch {
		case isXSD(el, "include"):
			// §F.2 clause 3: an <include> inside an overridden document becomes an
			// <override> carrying the same children, so ov CASCADES into the included
			// document — that is what makes §4.2.5's ·target set· transitive over
			// inclusion.
			if err := a.compose(el, tns, ov, ruleSrcInclude); err != nil {
				return err
			}
		case isXSD(el, "override"):
			if err := a.override(el, tns, ov); err != nil {
				return err
			}
		case isXSD(el, "import"):
			// §F.2 clause 5: an <import> is copied unchanged, so ov does NOT cascade
			// through it — §4.2.5's ·target set· "does not include schema documents
			// which are pointed to by <import> or <redefine> elements".
			if err := a.importDocument(el, tns); err != nil {
				return err
			}
		case isXSD(el, "redefine"):
			// GAP(xsd): <xs:redefine> (§4.2.4) is not followed. A <redefine> child of a
			// schema document is skipped, not rejected (§3.1.2), so the assembly is short
			// by whatever that document would have contributed. Nothing is resolved:
			// src-redefine clauses 2-4 are conditioned on a resolution never attempted
			// here, and clause 1 — which requires the schemaLocation to resolve whenever
			// the <redefine> has non-<annotation> children — is not enforced either, so a
			// genuine src-redefine violation is silently accepted. The skip leaves no
			// trace in the returned schema; this record is the only way to observe it
			// (Parse's WithLogger, at debug level).
			a.log.Debug("composition skipped: <xs:redefine> is not followed, its schemaLocation hint is never resolved",
				"rule", string(ruleSrcRedefine), "location", attrOr(el, "schemaLocation"), "at", el.Loc().String())
		}
	}
	return nil
}

// override follows one <override> element (§4.2.5). It reads the substitutions
// the element itself declares, composes them with ov — whatever override is
// already in force over the containing document — per §F.2 clause 4, and hands
// the result to compose, which fetches the overridden document and enforces
// src-override clauses 1 and 2 on it.
//
// Those two clauses are src-include clauses 1 and 2 in every particular, which is
// no coincidence: §4.2.5 clause 3.1.2 (and 3.2.2 for the chameleon case) define
// the <override> as "replaced by an <include> element pointing to Dold′", with
// "the inclusion … handled as described in [§4.2.3]". Hence one composer for
// both, told only which rule to charge.
//
// The merged set is handed ONLY to the overridden document; the document that
// CONTAINS this <override> keeps ov, the override in force over it. Substitution
// therefore cannot leak back into the overriding document, whatever cycle of
// mutual overrides the schema documents form (PRINCIPLES 16).
func (a *assembly) override(el *Element, tns string, ov *overrideSet) error {
	own, err := newOverrideSet(el)
	if err != nil {
		return err
	}
	return a.compose(el, tns, own.mergedUnder(ov), ruleSrcOverride)
}

// compose follows one <include> or <override> element: it resolves the
// schemaLocation, reads the document it names, enforces clauses 1 and 2 of rule —
// src-include (§4.2.3) or src-override (§4.2.5), whose clauses coincide — on it,
// and recurses into its own directives under ov. tns is the COMPOSING document's
// effective target namespace, which is both what clause 2 is judged against and
// what the composed document is discovered under (§F.1 coercion for the chameleon
// case).
func (a *assembly) compose(el *Element, tns string, ov *overrideSet, rule xsderr.Rule) error {
	hint, ok := el.Attr("schemaLocation")
	if !ok {
		// schemaLocation is REQUIRED on <include> and <override> by the schema for
		// schema documents, and src-include/src-override govern only what its actual
		// value resolves to. A missing required attribute is therefore a grammar
		// fault with no dedicated Schema Representation Constraint, reported as a
		// plain error — the same treatment a <group> with no ref gets in
		// produce_complex.go. It is ALSO a reference this assembly could not follow,
		// and is recorded as one: the error says the assembly stopped, the report
		// says which document set it stopped short of (§4.2.1 makes these
		// schemaLocations mandatory, "not hints").
		a.unfollowedAt(el, UnfollowedNoSchemaLocation)
		return fmt.Errorf("parser: <%s> at %s has no schemaLocation attribute, which the schema for schema documents requires", el.Name().Local(), el.Loc())
	}
	// §4.3.2 clause 4: the location is a URI REFERENCE, resolved against the base
	// URI in scope at the directive element itself (xml:base-aware), not against
	// the document URI.
	requested := schemaloc.Resolve(el.BaseURI(), hint)

	d2, resolved, err := a.fetch(requested, tns, ov, el, rule)
	if err != nil {
		return err
	}
	if d2 == nil {
		return nil
	}
	if !d2.IsSchema() {
		return xsderr.New(rule, el.Loc(),
			"schemaLocation %q resolves to a <%s> document element, but %s clause 1 requires it to resolve to a <schema> element information item", hint, d2.Root().Name().Local(), rule)
	}
	// D2 is compared against the composing document's EFFECTIVE namespace rather
	// than its raw targetNamespace attribute. Under chameleon inclusion the §F.1
	// transformation is applied to the whole composing document before its own
	// schema is computed, so its <include>s are evaluated as the coerced
	// document's: §4.2.3's recursion note spells out that "if A includes B and B
	// includes C … the effect is as if A included B' and B' included C'", with B'
	// carrying A's targetNamespace.
	own, hasOwn := d2.Root().Attr("targetNamespace")
	if hasOwn && own != tns {
		// src-include clause 2.1 (c-normi) fails (the two targetNamespaces differ,
		// or the composing document has none), 2.2 (c-normi2) fails (D2 has one),
		// 2.3 (c-chami) fails (D2 has one), and 2.4 fails (D2 exists); src-override
		// clause 2.1 (c-o-normir), 2.2 (c-o-normi2r) and 2.3 (c-o-chamir) fail for
		// exactly the same three reasons.
		return xsderr.New(rule, el.Loc(),
			"the schema document %q named by this <%s> has targetNamespace %q, but %s clause 2 requires it to be identical to the composing schema's %q or absent", requested, el.Name().Local(), own, rule, tns)
	}
	// Clause 2.1, 2.2, or — when D2 declares no targetNamespace and the composing
	// document has one — 2.3, whose §F.1 coercion is applied at production time:
	// either way D2 is discovered under the composing document's effective
	// namespace, carrying whatever override is in force over it.
	return a.discover(d2, resolved, tns, ov)
}

// importDocument follows one <import> element (§4.2.6.2). It enforces
// src-import-noselfimport (clause 1) against tns — the IMPORTING document's
// effective target namespace — resolves the schemaLocation hint if there is one,
// enforces src-import clauses 2 and 3 on what comes back, and discovers D2 under
// D2's OWN target namespace, never a coerced one.
//
// The ·absent· namespace is encoded as the empty string throughout, the same
// sentinel [loader.Resolver] documents, so clause 1's and clause 3's namespace
// comparisons are string comparisons; only the PRESENCE of the namespace
// attribute selects which sub-clause applies, since namespace="" is a legal (if
// unusual) xs:anyURI value and is not the same thing as no attribute at all.
func (a *assembly) importDocument(el *Element, tns string) error {
	namespace, hasNamespace := el.Attr("namespace")
	if err := checkNoSelfImport(el, namespace, hasNamespace, tns); err != nil {
		return err
	}
	hint, hasHint := el.Attr("schemaLocation")
	if !hasHint {
		// A bare <import> — namespace but no schemaLocation — is explicitly legal
		// (§4.2.6.2): it declares that references into namespace are expected
		// (src-resolve clause 4.2.2 / 4.1.2) without naming a document to satisfy
		// them from. There is nothing for the reference strategy to succeed at, so
		// clauses 2 and 3 do not apply and there is no D2 to discover. Deliberately
		// NOT routed through schemaloc.Resolve: that returns the IMPORTING
		// document's own base URI for an empty location, which would make a bare
		// import re-read the importer.
		a.unfollowedAt(el, UnfollowedNoLocation)
		a.log.Debug("import declares a namespace with no schemaLocation hint",
			"rule", string(ruleSrcImport), "namespace", namespace, "at", el.Loc().String())
		return nil
	}
	// §4.3.2 clause 4: a URI REFERENCE resolved against the base URI in scope at
	// the <import> element itself (xml:base-aware).
	requested := schemaloc.Resolve(el.BaseURI(), hint)

	// The resolver is asked under the IMPORT's namespace, not the importing
	// document's: that pair — target namespace and location hint — is exactly the
	// "application schema reference strategy" input clause 2 describes.
	d2, resolved, err := a.fetch(requested, namespace, nil, el, ruleSrcImport)
	if err != nil {
		return err
	}
	if d2 == nil {
		return nil
	}
	if !d2.IsSchema() {
		return xsderr.New(ruleSrcImport, el.Loc(),
			"schemaLocation %q resolves to a <%s> document element, but src-import clause 2 requires it to resolve to a <schema> element information item", hint, d2.Root().Name().Local())
	}
	own := attrOr(d2.Root(), "targetNamespace")
	if err := checkImportedNamespace(el, requested, namespace, hasNamespace, own); err != nil {
		return err
	}
	// Clause 3 holds, so own is the namespace the import declared. D2 is
	// discovered under it: import applies NO §F.1 coercion, which is src-include
	// clause 2.3's alone, and no ·override pre-processing· either, since §F.2
	// clause 5 copies an <import> unchanged.
	return a.discover(d2, resolved, own, nil)
}

// checkNoSelfImport enforces src-import clause 1 (src-import-noselfimport,
// §4.2.6.2) on one <import>: a document may not import the namespace it is
// already in.
//
// tns is the importing document's EFFECTIVE target namespace, not its raw
// targetNamespace attribute: §4.2.3's note on chameleon inclusion makes the
// including document's namespace the coerced document's own, so a chameleon
// document importing the includer's namespace is a self-import too. tns is ""
// for the ·absent· namespace — the sentinel [loader.Resolver] documents — so a
// literal targetNamespace="" arrives here as absent and a bare <import> from
// such a document is charged clause 1.2, where a strict-literal reading of "has
// no targetNamespace" would not charge it; tracking literal presence would be a
// second encoding of namespace-absence (STYLE D3).
func checkNoSelfImport(el *Element, namespace string, hasNamespace bool, tns string) error {
	if hasNamespace {
		// Clause 1.1: the namespace attribute's value must NOT match the enclosing
		// <schema>'s effective targetNamespace.
		if namespace != tns {
			return nil
		}
		return xsderr.New(ruleSrcImportNoSelfImport, el.Loc(),
			"<import> names namespace %q, which is the importing schema's own target namespace; src-import clause 1.1 requires them to differ", namespace)
	}
	// Clause 1.2: with no namespace attribute the enclosing <schema> must have a
	// targetNamespace — a bare <import> from a no-namespace document would import
	// the no-namespace it is already in.
	if tns != "" {
		return nil
	}
	return xsderr.New(ruleSrcImportNoSelfImport, el.Loc(),
		"<import> has no namespace attribute, but the importing schema has no target namespace either; src-import clause 1.2 requires one")
}

// checkImportedNamespace enforces src-import clause 3 (§4.2.6.2) on a D2 that
// the reference strategy did produce: the namespace the <import> declared and
// the one D2 actually declares must agree. own is D2's targetNamespace with the
// ·absent· namespace encoded as "".
func checkImportedNamespace(el *Element, requested, namespace string, hasNamespace bool, own string) error {
	if hasNamespace {
		// Clause 3.1: identical to D2's targetNamespace.
		if own == namespace {
			return nil
		}
		return xsderr.New(ruleSrcImport, el.Loc(),
			"<import>ed schema document %q has target namespace %q, but src-import clause 3.1 requires it to be identical to the namespace attribute's %q", requested, own, namespace)
	}
	// Clause 3.2: with no namespace attribute, D2 must have no targetNamespace.
	if own == "" {
		return nil
	}
	return xsderr.New(ruleSrcImport, el.Loc(),
		"<import> has no namespace attribute, but the schema document %q it names has target namespace %q; src-import clause 3.2 requires it to have none", requested, own)
}

// fetch resolves requested through the assembly's resolver, under namespace, and
// reads the schema document it names. It serves <include>, <override> and
// <import> alike, so it is handed the rule the calling directive answers to and
// reads the directive's element name off el rather than carrying a second,
// stringly-typed kind.
//
// namespace is what the resolver is asked under AND the namespace half of the
// load-once key: for <include>/<override> the composing document's effective
// target namespace (which is also what the composed document is discovered
// under), for <import> the import's own namespace attribute (which clause 3 then
// requires D2 to agree with). ov is the override half of that key and is nil for
// every directive but <override> (§F.2 clause 5 leaves an <import> unchanged).
//
// It returns a nil *Document, and no error, for the two outcomes that mean
// "perform no composition here" — outcomes the report must NOT conflate, which
// is why only the first records an [UnfollowedDirective]:
//
//   - the location does not resolve at all — §4.2.3: "It is not an error for the
//     actual value of the schemaLocation attribute to fail to resolve at all, in
//     which case the corresponding inclusion must not be performed" (src-include
//     clause 2.4), and §4.2.6.2 equally: "It is not an error for the application
//     schema component reference strategy to fail". No document was assembled for
//     this directive, so it is UnfollowedLocationUnresolved;
//   - the document was already loaded under this namespace and this override —
//     the same schema location names the same schema document (§4.2.3), §4.2.6.2's
//     note requires repeated <import>s of one document not to trip
//     sch-props-correct clause 2 either, and §4.2.5's note wants the same of
//     "multiple equivalent overrides of the same schema document", so it
//     contributes its components exactly once. This directive WAS followed — to a
//     document already in the report — so nothing is recorded: reporting a dedup
//     hit as unfollowed would mark almost every multi-document assembly.
//
// Any OTHER resolver failure is real: a permission or transport error is not
// silently downgraded to "absent". It still yielded no document, so it is
// recorded too, under the same reason — the report says which references came
// back empty, not why the resolver said so.
//
// The resolved location travels back with the document because it, not the
// requested one, is the assembly's document identity (docKey) and the report's
// [AssembledDocument.Location].
func (a *assembly) fetch(requested, namespace string, ov *overrideSet, el *Element, rule xsderr.Rule) (*Document, string, error) {
	rc, resolved, err := a.resolver.Resolve(namespace, requested)
	if errors.Is(err, loader.ErrNotFound) {
		a.unfollowedAt(el, UnfollowedLocationUnresolved)
		a.log.Debug("composition skipped: schemaLocation does not resolve",
			"rule", string(rule), "directive", el.Name().Local(),
			"namespace", namespace, "location", requested, "at", el.Loc().String())
		return nil, "", nil
	}
	if err != nil {
		a.unfollowedAt(el, UnfollowedLocationUnresolved)
		return nil, "", fmt.Errorf("parser: resolving <%s> schemaLocation %q at %s: %w", el.Name().Local(), requested, el.Loc(), err)
	}
	// A read-only reader: a close failure cannot change what was already read, so
	// it cannot affect the parse verdict (STYLE S3).
	defer func() { _ = rc.Close() }()

	key := docKey{resolved: resolved, namespace: namespace, override: ov.key()}
	if _, done := a.loaded[key]; done {
		a.log.Debug("schema document already loaded", "directive", el.Name().Local(),
			"namespace", namespace, "location", requested, "resolved", resolved, "at", el.Loc().String())
		return nil, resolved, nil
	}
	a.loaded[key] = struct{}{}

	doc, err := ReadDocument(requested, rc)
	if err != nil {
		// src-include clause 1.1 and src-import clause 2 alike require the resolved
		// resource to correspond to a <schema> item "in a well-formed information
		// set", so a well-formedness fault in a document that DID resolve is a
		// composition violation, not a bare XML error. The document is nonetheless
		// one the assembly never read, so the directive is recorded as unfollowed
		// alongside the verdict.
		a.unfollowedAt(el, UnfollowedUnreadable)
		return nil, "", xsderr.Wrap(rule, el.Loc(),
			fmt.Errorf("the schema document %q named by this <%s> is not well-formed, but %s requires a well-formed information set: %w", requested, el.Name().Local(), rule, err))
	}
	return doc, resolved, nil
}

// compile produces every discovered document into ONE builder — each under its
// OWN effective target namespace, so an imported document's components land in
// the namespace it declares rather than the root's — and finalizes it. Every
// document's pre-scan runs before any document is produced, so a base= or
// <attributeGroup ref> in one document reaches a definition contributed by
// another (§4.2.3 clause 3.1.2, c-incl-incl); the builtins are seeded once, by
// newSymbols, for the whole assembly. backend supplies the finalized schema's
// value space too ([value.NewValueSpace]), so the finalize-time {value}
// comparisons (au-props-correct clause 3, loc-testSubP clauses 4.2/5.2.2) and the
// Simple Default Valid checks (a-props-correct clause 2, au-props-correct clause
// 2) decide rather than fail open — see [Produce] for the full statement.
func (a *assembly) compile(backend value.Backend) (*xsd.Schema, error) {
	builder := xsd.NewSchemaBuilder()
	sym, err := newSymbols(builder, backend)
	if err != nil {
		return nil, err
	}
	producers := make([]*producer, 0, len(a.docs))
	for _, d := range a.docs {
		p := newProducer(d.doc, d.tns, d.ov, builder, sym)
		p.prescan()
		producers = append(producers, p)
	}
	for i, p := range producers {
		a.log.Debug("producing schema document",
			"uri", a.docs[i].doc.URI(), "targetNamespace", a.docs[i].tns, "chameleon", p.chameleon())
		if err := p.run(); err != nil {
			return nil, err
		}
	}
	return builder.FinalizeWith(value.NewValueSpace(backend))
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
