package conformance

import (
	"net/url"
	"path"
	"strings"

	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
)

// This file holds the schema lane's OWN <xs:include>/<xs:override>/<xs:import>
// closure walk (issues #242, #182, #183).
// It exists for one reason: schema.go's false-accept guard (schemaShapeDecidable)
// must hold for EVERY document parser.Parse assembles, not just the root, and
// parser.Parse cannot be asked which documents those were.
//
// # Why the harness re-walks what the parser walks
//
// parser.Parse's discovery is internal (assembly.discover/.include/.fetch in
// parser/parse.go, all unexported): the *xsd.Schema it returns carries components,
// not the list of documents they came from. A single-document lane could gate on
// the one document it read; a multi-document lane cannot, because an <include>d
// document holding a representation the producer silently SKIPS (§3.1.2) — an
// inline anonymous type, a list/union simpleType — or one it builds without any
// rule yet judging it (a <complexContent> <extension>, pending cos-ct-extends,
// #264) would let a schema-INVALID assembly "Parse" cleanly, a FALSE ACCEPT of exactly
// the kind schema.go's step-3 allowlist exists to prevent.
//
// So the harness discovers the closure itself, BEFORE calling parser.Parse, and
// runs schemaShapeDecidable on every document in it. Only when the whole closure
// is decidable does the case proceed; otherwise it DECLINES (Fail) as an honest
// recorded gap.
//
// # Why running the shape check on the RAW document is correct
//
// Chameleon inclusion (§4.2.3 clause 2.3, §F.1) is purely a NAMESPACE-level
// transformation: per the c-chamvalidi note it "(a) adds a targetNamespace
// [attribute] to D2 ... and (b) updates all unqualified QName references so that
// their namespace names become the ·actual value· of the targetNamespace
// [attribute]" (oracle grounding, issue #242). It restructures no elements, so the
// decidability verdict on the raw document is the verdict on the coerced one and
// the walk needs no chameleon awareness of its own. src-include itself likewise
// imposes no shape constraint on D2 — only existence and targetNamespace agreement
// — so shape decidability is a property this walk establishes independently of
// anything parser.Parse checks.
//
// # Why this walk must resolve locations EXACTLY as parser.Parse does
//
// Under-discovering is not a harmless conservatism here: a document this walk
// misses but parser.Parse reads is a document whose shape was never gated, which
// is the false accept back again. §4.3.2 clause 4 resolves a schemaLocation
// against the base URI in scope AT THE <include> ELEMENT, so the walk uses
// Element.BaseURI (already xml:base-aware) and the same resolution algorithm the
// parser uses — see resolveSchemaLocation below — over the SAME loader.Resolver
// the parser will be handed, reading the root under the same location string
// parser.Parse itself will (see execSchemaCase).
//
// Over-discovering, by contrast, is safe: the walk reads a superset of the
// documents parser.Parse reads (it keeps walking past conditions on which Parse
// gives up), in the same depth-first document order, so every document Parse reads
// has been shape-checked. A document only this walk reaches can cost a decidable
// case (declined on a shape Parse would never have seen) — a lost win, never a
// fabricated verdict.

// scanKey is the walk's load-once document identity, mirroring parser's docKey
// exactly: the resolved location paired with the namespace the document was
// reached under. The pair, not the location alone, because one document can be
// reached both as a chameleon <include> (coerced into the includer's namespace)
// and as a bare <import> (staying in no namespace), and the parser walks both.
type scanKey struct {
	resolved  string
	namespace string
}

// closureScan is one schema case's <xs:include>/<xs:override>/<xs:import> closure
// walk. It mirrors parser.assembly's discovery state and nothing else: the resolver every
// document is fetched through, and the load-once index. The effective target
// namespace is NOT a field, exactly as it is not one on parser.assembly: it is
// per-document (a chameleon include borrows the includer's, an import keeps its
// own) and is threaded as a parameter.
type closureScan struct {
	resolver loader.Resolver

	// loaded indexes the documents already walked. It is spec-mandated document
	// identity, not a defensive hack: §4.2.3 states that two <include> elements
	// specifying the same schema location (after resolving relative URI
	// references) refer to the same schema document, and <include> CYCLES are
	// legal — "processors should guard against infinite loops", not reject them;
	// §4.2.6.2's note wants repeated <import>s of one document treated the same.
	// It is scan-scoped: created per case, never threaded anywhere else
	// (STYLE D4, like xsd/resolve.go's acyclic-scan sets).
	loaded map[scanKey]struct{}

	// unresolved records that SOME directive in the closure yielded no D2 — an
	// <include>/<override>/<import> whose schemaLocation did not resolve, or a bare
	// <import> with none at all. Each is a spec non-error (§4.2.3 clause 2.4,
	// §4.3.2, §4.2.6.2), so it cannot decline the walk on its own; it is the
	// PRECONDITION of the fabricated-rejection hazard, which execSchemaCase
	// completes by pairing it with the parse outcome (#276). Recording it is not
	// redundant state (STYLE D3): nothing else in the scan witnesses a directive
	// that named no document, precisely because no document was loaded for it.
	unresolved bool
}

// newClosureScan returns the scan for a root schema document already read from
// rootResolved under rootTNS, its own target namespace. Seeding loaded with that
// pair mirrors parser.newAssembly, so a legal <include> cycle pointing back at
// the root is recognized as already-walked rather than re-read.
func newClosureScan(resolver loader.Resolver, rootResolved, rootTNS string) *closureScan {
	return &closureScan{
		resolver: resolver,
		loaded:   map[scanKey]struct{}{{resolved: rootResolved, namespace: rootTNS}: {}},
	}
}

// visited reports whether resolved was reached, under ANY namespace, during the
// walk — i.e. whether decidable has already shape-checked that filesystem
// location. It exists for the multi-document schemaTest, whose extra declared
// documents may only be decided when the walk from the first one provably
// reached them.
//
// The namespace half of the key is deliberately ignored: one document can be
// reached under several namespaces (a chameleon <include> and a bare <import>),
// and the question here is only whether THIS location was gated, not under which
// namespace. The linear scan is the whole implementation on purpose — loaded
// holds one case's closure and is scanned once per extra document, so a second
// index would be redundant state (STYLE D3) for no measured hot path (STYLE D4).
func (s *closureScan) visited(resolved string) bool {
	for k := range s.loaded {
		if k.resolved == resolved {
			return true
		}
	}
	return false
}

// decidable reports whether doc — reached under the effective target namespace
// tns — and every document transitively reachable from it through a top-level
// <xs:include> or <xs:import> lies within the producer's decidable subset. It is
// the root's entry point too: the root is checked through the same code path as
// every composed document, so no shape is gated differently for being the root.
//
// A directive that names no document at all is NOT what this reports on: there is
// no shape to gate, so the walk keeps going and merely records the fact on the
// scan (closureScan.unresolved), which execSchemaCase reads once the parse
// outcome is known (#276).
func (s *closureScan) decidable(doc *parser.Document, tns string) bool {
	if !schemaShapeDecidable(doc) {
		return false
	}
	// ONE document-order pass, depth-first, pre-order — the same single pass
	// parser.assembly.discover makes, so the load-once index dedups the same
	// documents in the same order (STYLE D1).
	for _, child := range doc.Root().Children() {
		el, ok := child.(*parser.Element)
		if !ok || el.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		switch el.Name().Local() {
		case "include", "override":
			if !s.compose(el, tns) {
				return false
			}
		case "import":
			if !s.importDirective(el) {
				return false
			}
		}
	}
	return true
}

// compose follows one <include> or <override> element and reports whether what it
// names is decidable. The two share this walk because §4.2.5 clause 3.1.2 defines
// an <override> as "replaced by an <include> element pointing to Dold′", with the
// inclusion "handled as described in [§4.2.3]" — the parser composes both through
// one method for the same reason (parser.assembly.compose).
//
// The walk deliberately carries NO override state. §F.2's transformation
// substitutes declarations inside the documents of the ·target set·; it never
// changes which schemaLocations they name, so the DOCUMENT SET is the same
// whether or not an override is in force, and it is the document set this walk
// exists to enumerate. The parser may read one document several times (once per
// distinct override applied to it, see parser's docKey) where this walk reads it
// once; that is not under-discovery, because every one of those readings has the
// same raw top-level shape, which was gated on the single visit — and the
// substituted declarations, which come from the <override> elements themselves,
// are gated by overrideDecidable where those elements occur (schema.go).
//
// Its outcomes:
//
//   - no schemaLocation attribute: DECLINE outright. The attribute is REQUIRED by
//     the schema for schema documents, so this is a malformed directive, not the
//     spec's "fails to resolve" non-error; parser.Parse reports its absence as a
//     plain grammar fault, and the walk cannot verify the decidability of a target
//     it cannot name.
//   - the location does not resolve, for ANY resolver reason: NOT a decline, and
//     the siblings keep being walked — the fact is RECORDED on the scan
//     (s.unresolved) instead, symmetrically with importDirective (#276). §4.2.3
//     clause 2.4 is explicit that "it is not an error for the ·actual value· of the
//     schemaLocation [attribute] to fail to resolve at all, in which case the
//     corresponding inclusion must not be performed", and §4.3.2 says the same of
//     <override> ("the attempt must be made but it is not an error for it to
//     fail"). When parser.Parse then SUCCEEDS, that non-error rule is simply
//     working — nothing was fabricated, and the case must still be decided; the
//     suite tests the rule head-on (MS-Schema schD8 names a target
//     "must%20not%20resolve.xyzzy"), so declining here would refuse the very cases
//     clause 2.4 is about.
//     The hazard is the OTHER outcome. Precisely because the missing D2 is a
//     non-error, parser.Parse may assemble a schema lacking every component D2
//     would have contributed, and each reference that would have been discharged
//     from it instead fails src-resolve clauses 1-3 at finalize (the §5.3 Missing
//     Sub-components pathway src-include's own prose points at). The lane would
//     then agree with a suite-invalid case for a reason the spec does not attach to
//     the missing document: a FABRICATED "invalid" verdict, the one direction that
//     can corrupt the ratchet. That failure is visible only AFTER the parse, so
//     execSchemaCase declines exactly when this recorded flag coincides with
//     perr != nil.
//   - a ReadDocument error on the fetched document: DECLINE, and ambiguous on top
//     of that — a genuine well-formedness fault or a parser encoding LIMITATION
//     (well-formed UTF-16 read as invalid UTF-8) — so it may not be turned into a
//     verdict (the same reasoning as schema.go step 1). Unlike a location that
//     never resolved, a D2 that exists but could not be read leaves a document the
//     parser does read UNGATED.
//   - the document is already loaded: NOT a decline, and not re-walked. It was
//     shape-checked when first reached.
//   - the document is not a <schema>: NOT a decline, and not recursed into. There
//     is no top-level content to shape-check, and parser.Parse rejects this
//     independently as a genuine src-include (or src-override) clause 1 violation
//     — a real decided rejection, not a skipped representation.
func (s *closureScan) compose(el *parser.Element, tns string) bool {
	hint, ok := elementAttr(el, "schemaLocation")
	if !ok {
		return false
	}
	// §4.3.2 clause 4: a URI reference resolved against the base URI in scope at
	// the directive element itself (xml:base-aware), not the document URI.
	requested := resolveSchemaLocation(el.BaseURI(), hint)

	rc, resolved, err := s.resolver.Resolve(tns, requested)
	if err != nil {
		s.unresolved = true
		return true
	}
	// Read-only reader: a close failure cannot change what was already read, so it
	// cannot affect the verdict (STYLE S3).
	defer func() { _ = rc.Close() }()

	key := scanKey{resolved: resolved, namespace: tns}
	if _, done := s.loaded[key]; done {
		return true
	}
	s.loaded[key] = struct{}{}

	d2, err := parser.ReadDocument(requested, rc)
	if err != nil {
		return false
	}
	if !d2.IsSchema() {
		return true
	}
	// Whether D2 declares tns itself or is coerced into it (§4.2.3 clause 2.3,
	// §4.2.5 clause 2.3), the parser discovers it under the COMPOSING document's
	// effective namespace.
	return s.decidable(d2, tns)
}

// importDirective follows one <import> element and reports whether what it names
// is decidable. Its outcomes are compose's, for the same reasons: a directive that
// yields no D2 is RECORDED, not declined, whichever directive it is.
//
//   - no schemaLocation attribute (the bare <import> §4.2.6.2 calls legal), or a
//     schemaLocation that does not resolve for any resolver reason: NOT a decline,
//     RECORDED on the scan (s.unresolved). Both are non-errors for the parser
//     (§4.2.6.2: "It is not an error for the application schema component
//     reference strategy to fail"), so the imported namespace contributes NO
//     components — and every reference into it then fails src-resolve at finalize.
//     THAT is the fabricated "invalid" this records against, and it is observable
//     only as a failed parse, so execSchemaCase pairs the flag with perr != nil
//     (#276); an import that yields no D2 while the parse succeeds broke nothing
//     and is decided normally.
//     §4.2.6.2's "not an error" text is the exact parallel of src-include clause
//     2.4, and src-resolve draws no line between the two origins either: clause 4
//     (cl.qnr.nsdeclared) licenses a same-namespace reference unconditionally
//     (4.2.1) and a foreign-namespace one on the mere PRESENCE of a matching
//     <import> element (4.2.2), whether or not that import's schemaLocation
//     resolved — so a missing included D2 and a missing imported D2 both fail at
//     clauses 1-3, in the same §5.3 bucket. They are one hazard, and compose
//     records it identically (#276). The one asymmetry §4.2.6.1 does draw — a
//     foreign namespace referenced with no <import> element at ALL is not
//     §5.3-eligible — keys on the ABSENCE of a directive, not on a resolver
//     outcome, and is not what this branch sees.
//   - already loaded under this namespace: NOT a decline, and not re-walked. It
//     was shape-checked when first reached.
//   - a ReadDocument error on the fetched document: DECLINE — ambiguous between a
//     well-formedness fault and a parser encoding LIMITATION, exactly as for
//     include, and the document it leaves ungated is one the parser does read.
//   - the document is not a <schema>: NOT a decline. src-import clause 2 makes
//     that a genuine rejection parser.Parse emits, and there is no top-level
//     content to shape-check.
//
// A src-import clause 1 or clause 3 violation needs no special handling here:
// those are genuine, implemented rejections on documents this walk has gated.
func (s *closureScan) importDirective(el *parser.Element) bool {
	hint, hasHint := elementAttr(el, "schemaLocation")
	if !hasHint {
		s.unresolved = true
		return true
	}
	// The resolver is asked under the IMPORT's namespace, as parser.assembly does:
	// that (namespace, location) pair is the reference strategy's input.
	namespace, _ := elementAttr(el, "namespace")
	requested := resolveSchemaLocation(el.BaseURI(), hint)

	rc, resolved, err := s.resolver.Resolve(namespace, requested)
	if err != nil {
		s.unresolved = true
		return true
	}
	// Read-only reader (STYLE S3).
	defer func() { _ = rc.Close() }()

	key := scanKey{resolved: resolved, namespace: namespace}
	if _, done := s.loaded[key]; done {
		return true
	}
	s.loaded[key] = struct{}{}

	d2, err := parser.ReadDocument(requested, rc)
	if err != nil {
		return false
	}
	if !d2.IsSchema() {
		return true
	}
	// No §F.1 coercion on an import: D2 is walked under its OWN target namespace,
	// the namespace the parser discovers it under.
	own, _ := elementAttr(d2.Root(), "targetNamespace")
	return s.decidable(d2, own)
}

// resolveSchemaLocation resolves a schemaLocation URI reference against the base
// URI in scope at the element carrying it (§4.3.2 clause 4).
//
// It is a VERBATIM copy of parser.resolveSchemaLocation (parser/parse.go) and MUST
// be kept in sync with it. The duplication is deliberate: the function has no
// consumer outside the parser's own assembly, so exporting it would add public
// surface no library user needs (STYLE T5), yet this harness needs the IDENTICAL
// algorithm — a walk that resolved a hint even slightly differently would discover
// a different document set than parser.Parse actually reads, and a document it
// under-discovered would be one whose shape was never gated, reopening the
// false-accept gap this whole walk exists to close.
//
// An absolute reference wins outright. Otherwise, when the base is itself an
// absolute URI, standard RFC 3986 reference resolution applies. When it is not — a
// bare relative path such as "sub/child.xsd", which is exactly what a resolver
// rooted at a directory or an in-memory map is keyed by — url.URL.ResolveReference
// would root the result at "/" and turn a resolver-relative location into an
// absolute one that no such resolver can serve; path-wise resolution against the
// base's directory is used instead.
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

// elementAttr returns the value of el's unprefixed (no-namespace) attribute local,
// as XSD schema-element attributes (name, ref, schemaLocation, …) carry no
// namespace. ok is false when the attribute is absent.
//
// It is named for its receiver type rather than reusing the parser's attrValue
// spelling because conformance/datatypes.go already owns attrValue over
// encoding/xml.StartElement, a different type entirely.
func elementAttr(el *parser.Element, local string) (string, bool) {
	for _, a := range el.Attributes() {
		if a.Name().Space() == "" && a.Name().Local() == local {
			return a.Value(), true
		}
	}
	return "", false
}
