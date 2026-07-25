package conformance

import (
	"errors"
	"net/url"
	"path"
	"strings"

	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
)

// This file holds the schema lane's OWN <xs:include> closure walk (issue #242).
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
// inline anonymous type, a list/union simpleType, a <complexContent> <extension> —
// would let a schema-INVALID assembly "Parse" cleanly, a FALSE ACCEPT of exactly
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

// closureScan is one schema case's <xs:include> closure walk. It mirrors
// parser.assembly's discovery state and nothing else: the resolver every document
// is fetched through, the assembly's effective target namespace (the ROOT
// document's — every document of the closure ends up in it, §4.2.3 clause 2), and
// the load-once index.
type closureScan struct {
	resolver loader.Resolver

	// tns is what the resolver is asked under for an <include>d location, exactly
	// as parser.assembly.fetch asks. It is the root document's targetNamespace,
	// since the parser fixes the assembly's namespace there.
	tns string

	// loaded indexes the RESOLVED locations already walked. It is spec-mandated
	// document identity, not a defensive hack: §4.2.3 states that two <include>
	// elements specifying the same schema location (after resolving relative URI
	// references) refer to the same schema document, and <include> CYCLES are
	// legal — "processors should guard against infinite loops", not reject them.
	// It is scan-scoped: created per case, never threaded anywhere else
	// (STYLE D4, like xsd/resolve.go's acyclic-scan sets).
	loaded map[string]struct{}
}

// newClosureScan returns the scan for a root schema document already read from
// rootResolved. Seeding loaded with the root's own resolved location mirrors
// parser.newAssembly, so a legal <include> cycle pointing back at the root is
// recognized as already-walked rather than re-read.
func newClosureScan(resolver loader.Resolver, root *parser.Document, rootResolved string) *closureScan {
	tns, _ := elementAttr(root.Root(), "targetNamespace")
	return &closureScan{
		resolver: resolver,
		tns:      tns,
		loaded:   map[string]struct{}{rootResolved: {}},
	}
}

// decidable reports whether doc and every document transitively reachable from it
// through a top-level <xs:include> lies within the producer's decidable subset. It
// is the root's entry point too: the root is checked through the same code path as
// every included document, so no shape is gated differently for being the root.
func (s *closureScan) decidable(doc *parser.Document) bool {
	if !schemaShapeDecidable(doc) {
		return false
	}
	// Document order, depth-first, pre-order — the same order parser.assembly
	// discovers in, so the load-once index dedups the same documents (STYLE D1).
	for _, child := range doc.Root().Children() {
		el, ok := child.(*parser.Element)
		if !ok {
			continue
		}
		name := el.Name()
		if name.Space() != xsd.XMLSchemaNS || name.Local() != "include" {
			continue
		}
		if !s.include(el) {
			return false
		}
	}
	return true
}

// include follows one <include> element and reports whether what it names is
// decidable. Its outcomes:
//
//   - no schemaLocation attribute: DECLINE. The attribute is required by the
//     schema for schema documents and parser.Parse reports its absence as a plain
//     grammar fault, but the walk cannot verify the decidability of a target it
//     cannot name, so being conservative is the only honest answer.
//   - the location does not resolve: NOT a decline. §4.2.3 clause 2.4 is explicit
//     that "it is not an error for the ·actual value· of the schemaLocation
//     [attribute] to fail to resolve at all, in which case the corresponding
//     inclusion must not be performed" — parser.Parse performs no inclusion
//     either, so there is nothing left to shape-check. Keep walking the siblings.
//   - any OTHER resolver error, or a ReadDocument error on the fetched document:
//     DECLINE. Both are ambiguous — a permission or transport failure, or a parser
//     encoding LIMITATION (well-formed UTF-16 read as invalid UTF-8) — and neither
//     may be turned into a verdict (the same reasoning as schema.go step 1).
//   - the document is already loaded: NOT a decline, and not re-walked. It was
//     shape-checked when first reached.
//   - the document is not a <schema>: NOT a decline, and not recursed into. There
//     is no top-level content to shape-check, and parser.Parse rejects this
//     independently as a genuine src-include clause 1 violation — a real decided
//     rejection, not a skipped representation.
func (s *closureScan) include(el *parser.Element) bool {
	hint, ok := elementAttr(el, "schemaLocation")
	if !ok {
		return false
	}
	// §4.3.2 clause 4: a URI reference resolved against the base URI in scope at
	// the <include> element itself (xml:base-aware), not the document URI.
	requested := resolveSchemaLocation(el.BaseURI(), hint)

	rc, resolved, err := s.resolver.Resolve(s.tns, requested)
	if errors.Is(err, loader.ErrNotFound) {
		return true
	}
	if err != nil {
		return false
	}
	// Read-only reader: a close failure cannot change what was already read, so it
	// cannot affect the verdict (STYLE S3).
	defer func() { _ = rc.Close() }()

	if _, done := s.loaded[resolved]; done {
		return true
	}
	s.loaded[resolved] = struct{}{}

	d2, err := parser.ReadDocument(requested, rc)
	if err != nil {
		return false
	}
	if !d2.IsSchema() {
		return true
	}
	return s.decidable(d2)
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
