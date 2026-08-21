package parser

import (
	"github.com/kud360/goxsd8/parser/xmltree"
	"github.com/kud360/goxsd8/xsderr"
)

// Node is one item in a parsed schema document's element tree: an element or a
// run of character data. It is a closed set (STYLE T2 sealed sum) — consumers
// type-switch over *Element and *Text — sealed by the unexported node method so
// no other package can add a case. Every Node answers Loc.
type Node interface {
	// Loc reports where the node begins: the document URI plus the 1-based
	// line and column of its first byte.
	Loc() xsderr.Loc
	node()
}

// Element is one element in a schema document's tree: a start tag together with
// its retained content. Unlike parser/xmltree, which streams and keeps only
// bounded state, the tree holds the whole document so a later resolution phase
// can walk parents and children (§3.17.6.2 src-resolve walks up to a <schema>
// ancestor). Its resolved name, ordered attributes, in-scope namespace
// bindings, and location are those of the underlying xmltree.StartElement, to
// which the accessors delegate — the single canonical QName and scope
// implementation, never re-derived here.
//
// Name, Attributes, Attr, Children, BaseURI and Loc are exported for a single
// consumer, this module's own conformance harness, which walks raw schema
// documents; Loc alone is exported because the Node interface carries it rather
// than because anything calls it. No use case outside this module is known, so
// pre-1.0 these names may be unexported without a deprecation path — the
// ratchet and the gate are the only compatibility promises (docs/PLAN.md, "v1.0
// — the stability line").
type Element struct {
	src *xmltree.StartElement
	// parent is the enclosing element, nil for the tree's root. The link is set
	// once at build time (STYLE D3: a structural edge, not derived-redundant
	// state) so a later src-resolve phase (§3.17.6.2) can walk up to a <schema>
	// ancestor without re-searching the tree.
	parent   *Element
	children []Node
	baseURI  string
}

// Name returns the resolved (namespace, local) name of the element, delegating
// to the underlying start tag.
func (e *Element) Name() xmltree.Name { return e.src.Name() }

// Attributes returns the element's attributes in document order, excluding
// namespace declarations, delegating to the underlying start tag. The slice is
// the reader's own; its Attribute values are immutable.
func (e *Element) Attributes() []xmltree.Attribute { return e.src.Attributes() }

// Attr returns the value of the element's UNPREFIXED (no-namespace) attribute
// named local, reporting whether it is present at all. Presence is a distinct
// fact from an empty value throughout XSD — a bare <import> (no schemaLocation)
// is not an <import schemaLocation=""> (§4.2.6.2), and namespace="" is a legal
// xs:anyURI (§4.2.6.2 clause 1.2) — so both come back rather than one collapsed
// string.
//
// The no-namespace restriction is the whole point: every attribute the schema
// for schema documents declares on an XSD element (name, ref, type,
// schemaLocation, …) is unqualified, so a prefixed attribute of the same local
// name is foreign markup and must not answer here. That rule is subtle enough
// that a consumer walking a schema document tree through [Element.Attributes]
// would otherwise re-derive it — as this module's own conformance harness did,
// until the two implementations were collapsed into this one (issue #272).
func (e *Element) Attr(local string) (string, bool) {
	for _, a := range e.Attributes() {
		if a.Name().Space() == "" && a.Name().Local() == local {
			return a.Value(), true
		}
	}
	return "", false
}

// lookupPrefix resolves prefix to a namespace URI using the bindings in scope
// at this element, so a consumer can later resolve a QName-valued lexical that
// occurred here (Datatypes §3.3.18). It delegates to the underlying start tag;
// the empty prefix yields the default namespace, "xml" always resolves, and ok
// is false for an unbound prefix.
func (e *Element) lookupPrefix(prefix string) (uri string, ok bool) {
	return e.src.LookupPrefix(prefix)
}

// inScopePrefixes returns every prefixed namespace binding in scope at this
// element, sorted by prefix, so a consumer can materialize the whole in-scope
// namespace context an XPath expression written here must be interpreted
// against (Structures §3.13.2's {namespace bindings}). It delegates to the
// underlying start tag; the default namespace is not among them — it is
// obtained from lookupPrefix("").
func (e *Element) inScopePrefixes() []xmltree.Namespace {
	return e.src.InScopePrefixes()
}

// Loc reports the position of the element's opening "<", delegating to the
// underlying start tag.
func (e *Element) Loc() xsderr.Loc { return e.src.Loc() }

// Children returns the element's child nodes — elements and character-data runs
// — in document order (STYLE D2: a slice, never a map). Character data is
// retained as first-class Text nodes so mixed content (annotations) is not
// lost: the streaming reader discards the document as it advances, so text not
// captured here is unrecoverable for later phases.
func (e *Element) Children() []Node { return e.children }

// BaseURI returns the element's base URI: its xml:base attribute resolved as an
// RFC 3986 relative reference against its parent's base URI, or the parent's
// base URI inherited unchanged when it declares no xml:base. See ReadDocument
// for the composition rule and its (external, not locally spec-grounded)
// provenance.
func (e *Element) BaseURI() string { return e.baseURI }

func (e *Element) node() {}

// Text is a run of character data (text or CDATA) retained as a first-class
// child node, so mixed content — notably <xs:documentation> text inside
// <xs:annotation> — round-trips through the tree. Whitespace is preserved
// verbatim; stripping or normalization is a later phase's decision, not this
// raw layer's (STYLE D3: no concatenating Text() convenience until a real
// consumer needs it).
type Text struct {
	data string
	loc  xsderr.Loc
}

// Data returns the decoded character content, with entity and character
// references already expanded by the underlying reader.
func (t *Text) Data() string { return t.data }

// Loc reports the position of the content's first byte.
func (t *Text) Loc() xsderr.Loc { return t.loc }

func (t *Text) node() {}
