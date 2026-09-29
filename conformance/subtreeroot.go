package conformance

import (
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
)

// This file holds the instance lane's one shape gate for the "valid"
// observation: an ASSESSED SUBTREE ROOT (#1841), a validation root, with or
// without content (#1855), whose every element and attribute the walk strictly
// assessed against a declaration and a type it really has, with no clause of
// key-sva (§3.3.4.6), cvc-elt (§3.3.4.3), cvc-type (§3.3.4.4) or
// cvc-complex-type (§3.4.4.2) left undecided unrecorded. instance.go's "Why an
// EMPTY Result is evidence of validity for ONE shape only" states which clause
// each condition discharges; this file is only the conditions.
//
// The gate is computed per case and stored nowhere. It re-reads the instance
// with encoding/xml and re-derives every child's ·attribution· through
// xsd.Schema.ContentMatcher, independently of the walk, so it never rests on
// what the walk did or did not record for a descendant.

// versioningNS is the XML Schema versioning namespace §4.2.2 reads vc:minVersion,
// vc:maxVersion, vc:typeAvailable, vc:typeUnavailable, vc:facetAvailable and
// vc:facetUnavailable from. parser keeps its own copy unexported; the lane
// declares the name here rather than parser exporting it for a gate that exists
// only because of a parser GAP (#1002).
const versioningNS = "http://www.w3.org/2007/XMLSchema-versioning"

// walkUnrecorded are the builtin simple types whose presence anywhere in a
// type's closure puts in play a clause the walk neither decides nor records a
// decline for, at any depth. The gate excludes these alone: the ID family and
// the ENTITY family are the walk's to decide and record at every depth
// (instance.go, the cvc-elt clause 7 bullet).
//
// GAP(validate): an xs:NOTATION value is never checked against the schema's
// notations. NOTATION's ·value space· is "the set of QNames of notations
// declared in the current schema" (Datatypes §3.3.19, with
// enumeration-required-notation), which no validate or backend site checks and
// records no decline for — Override/over027/instance/over027.n01.xml, whose
// NOTATION value names no declared notation, is suite-invalid and walks clean,
// a false accept (#1901).
var walkUnrecorded = []string{"NOTATION"}

// assessedSubtreeRoot reports whether the instance document at doc, against
// schema as assembled into report, has the assessed-subtree-root shape an empty
// validate.Result may be read as "valid" for: a root, with content or without,
// whose subtree meets every condition subtreeGate.element names, in a document
// with no DOCTYPE, against an assembly no version condition touched. Any
// failure to establish a condition — an unreadable document, a decoder error,
// an unresolvable component — is a false, never a guess.
func assessedSubtreeRoot(schema *xsd.Schema, report *parser.AssemblyReport, doc string) bool {
	if closureVersioned(report) {
		return false
	}
	rc, _, err := loader.Dir(filepath.Dir(doc)).Resolve("", filepath.Base(doc))
	if err != nil {
		return false
	}
	defer func() { _ = rc.Close() }() // read-only handle: close error cannot affect the verdict
	dec := xml.NewDecoder(rc)
	root, ok := rootStart(dec)
	if !ok {
		return false
	}
	// cvc-elt clause 1 (§3.3.4.6 ·governing element declaration· clause 4): the
	// root's declaration is the top-level one its ·expanded name· resolves to. A
	// root with none is the untyped element shape, assessed against no
	// declaration.
	d, ok := schema.Element(expandedName(root.Name))
	if !ok {
		return false
	}
	g := subtreeGate{schema: schema, dec: dec}
	if !g.element(root, d, false) {
		return false
	}
	return documentEnd(dec)
}

// documentEnd reads dec past the document element to the end of the document,
// answering false for a directive or a decoder error there.
func documentEnd(dec *xml.Decoder) bool {
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return true
		}
		if err != nil {
			return false
		}
		if _, ok := tok.(xml.Directive); ok {
			return false
		}
	}
}

// subtreeGate is one assessedSubtreeRoot reading: the schema the subtree is
// checked against and the decoder positioned inside it.
type subtreeGate struct {
	schema *xsd.Schema
	dec    *xml.Decoder
}

// element reads the element whose start tag is start, and whose ·governing
// element declaration· is d, through to its end tag, and reports whether every
// condition below holds for it and, recursively, for every element under it:
//
//   - it carries no xsi:type and no xsi:nil (plainAttributes);
//   - d is not abstract and carries no {type table} and no fixed {value
//     constraint} (assessedDeclaration);
//   - d.{type definition} resolves;
//   - for a Simple Type Definition, its closure reaches none of
//     walkUnrecorded, the element carries no attribute but the four
//     xsi: ones cvc-type clause 3.1.1 excepts, and no element [[child]];
//   - for a Complex Type Definition, the conditions complex names.
//
// constrained reports whether the declaration of an element above this one
// carries {identity-constraint definitions}; element passes it on set from d
// down, where complex refuses every ·defaulted attribute·.
func (g *subtreeGate) element(start xml.StartElement, d xsd.ElementDeclaration, constrained bool) bool {
	if !plainAttributes(start.Attr) || !assessedDeclaration(d) {
		return false
	}
	constrained = constrained || len(d.IdentityConstraints()) > 0
	td, ok := g.schema.ResolvedType(d.TypeDefinition())
	if !ok {
		return false
	}
	switch t := td.(type) {
	case *xsd.SimpleType:
		if closureReaches(g.schema, t, walkUnrecorded) || slices.ContainsFunc(start.Attr, notExcepted) {
			return false
		}
		return g.leaf()
	case xsd.ComplexType:
		return g.complex(start, t, constrained)
	}
	return false
}

// plainAttributes reports whether attrs, one start tag's attribute list, holds
// no xsi:type and no xsi:nil.
func plainAttributes(attrs []xml.Attr) bool {
	for _, a := range attrs {
		if a.Name.Space != xsd.XMLSchemaInstanceNS {
			continue
		}
		if a.Name.Local == "type" || a.Name.Local == "nil" {
			return false
		}
	}
	return true
}

// notExcepted reports whether a is an attribute information item that neither
// a namespace declaration is nor one of the four xsi: attributes (§3.2.7)
// cvc-type clause 3.1.1 and cvc-complex-type clause 2 except by name.
func notExcepted(a xml.Attr) bool {
	if isNamespaceDeclaration(a) || isLocationHint(a) {
		return false
	}
	return a.Name.Space != xsd.XMLSchemaInstanceNS || (a.Name.Local != "type" && a.Name.Local != "nil")
}

// assessedDeclaration reports whether d leaves no cvc-elt clause the walk does
// not decide at depth: {abstract} false (clause 2, charged at the root alone),
// no {type table} (clause 4's ·selected type definition· is then d.{type
// definition}), and no fixed {value constraint} (clause 5.2.2). A default one
// is admitted: clause 5.1 is the walk's. So are {identity-constraint
// definitions}: clause 6 (cvc-identity-constraint, §3.11.4) is the walk's,
// which records every check it declines, and the one outcome it leaves
// unrecorded, a field node that is a ·defaulted attribute·, is complex's to
// refuse.
func assessedDeclaration(d xsd.ElementDeclaration) bool {
	if d.Abstract() {
		return false
	}
	if _, ok := d.TypeTable(); ok {
		return false
	}
	vc, ok := d.ValueConstraint()
	return !ok || vc.Kind() != xsd.ValueFixed
}

// complex reads an element governed by the Complex Type Definition t through to
// its end tag, and reports whether:
//
//   - t.{abstract} is false (cvc-type clause 2);
//   - every one of t.{attribute uses} resolves to an {attribute declaration}
//     whose {type definition} resolves to a simple type whose closure reaches
//     none of walkUnrecorded, present on the element or not;
//   - every attribute the element carries that notExcepted names matches one
//     of those uses by ·expanded name· (cvc-complex-type clause 2.1), so none
//     is ·attributed to· the {attribute wildcard};
//   - where constrained, the element has no ·defaulted attribute·
//     (defaultedAttribute). §3.11.4 clause 3's Note has a default or fixed
//     value play a part in a ·key-sequence·, but validate's
//     icCheck.fieldAttributes reads only the attributes the instance carries,
//     so a field selecting a defaulted one leaves the ·key-sequence· short
//     with nothing charged and nothing recorded. The refusal is wider than
//     that shape: it does not ask whether any {fields} path selects the
//     attribute;
//   - under a simple {content type}, its {simple type definition}'s closure
//     reaches none of walkUnrecorded and there is no element [[child]];
//   - under an empty one, there is no element [[child]];
//   - under an element-only or mixed one, xsd.Schema.ContentMatcher decides it
//     and every element [[child]] meets child's conditions.
func (g *subtreeGate) complex(start xml.StartElement, t xsd.ComplexType, constrained bool) bool {
	if t.Abstract() {
		return false
	}
	uses := t.AttributeUses()
	for _, u := range uses {
		ad, ok := g.schema.ResolvedAttributeDeclaration(u)
		if !ok {
			return false
		}
		st, ok := g.schema.ResolvedSimpleType(ad.TypeDefinition())
		if !ok || closureReaches(g.schema, st, walkUnrecorded) {
			return false
		}
		if constrained && g.defaultedAttribute(u, start.Attr) {
			return false
		}
	}
	for _, a := range start.Attr {
		if !notExcepted(a) {
			continue
		}
		n := expandedName(a.Name)
		if !slices.ContainsFunc(uses, func(u xsd.AttributeUse) bool { return u.DeclarationName() == n }) {
			return false
		}
	}
	switch ct := t.ContentType().(type) {
	case xsd.SimpleContent:
		if closureReaches(g.schema, ct.SimpleType, walkUnrecorded) {
			return false
		}
		return g.leaf()
	case xsd.EmptyContent:
		return g.leaf()
	case xsd.ElementContent:
		m, ok := g.schema.ContentMatcher(t)
		if !ok {
			return false
		}
		return g.children(m, constrained)
	}
	return false
}

// defaultedAttribute reports whether u is a ·defaulted attribute· (§3.4.4.2)
// of an element whose start tag carries attrs: u.{required} false, u's
// ·effective value constraint· not ·absent·, and no attribute of attrs
// matching u.{attribute declaration} by ·expanded name·. The definition's
// clause 4, excluding the built-in xsi: declarations, holds for every member
// of {attribute uses}.
func (g *subtreeGate) defaultedAttribute(u xsd.AttributeUse, attrs []xml.Attr) bool {
	if u.Required() {
		return false
	}
	if _, ok := g.schema.EffectiveValueConstraint(u); !ok {
		return false
	}
	n := u.DeclarationName()
	return !slices.ContainsFunc(attrs, func(a xml.Attr) bool { return expandedName(a.Name) == n })
}

// leaf reads an element's content through to its end tag and reports whether
// no element started inside it.
func (g *subtreeGate) leaf() bool {
	for {
		tok, err := g.dec.Token()
		if err != nil {
			return false
		}
		switch tok.(type) {
		case xml.StartElement:
			return false
		case xml.EndElement:
			return true
		}
	}
}

// children reads an element's content through to its end tag, advancing m over
// each element [[child]] in document order, and reports whether every child
// meets child's conditions, constrained as element states, and m accepts the
// whole sequence.
func (g *subtreeGate) children(m *xsd.Matcher, constrained bool) bool {
	for {
		tok, err := g.dec.Token()
		if err != nil {
			return false
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if !g.child(m, t, constrained) {
				return false
			}
		case xml.EndElement:
			return m.Accepting()
		}
	}
}

// child reads one element [[child]] whose start tag is start through to its
// end tag, and reports whether m ·attributes· it (§3.4.4.4) to an element
// particle whose {term} carries the child's own ·expanded name· — its
// ·context-determined declaration· (§3.3.4.6 ·governing element declaration·
// clause 2) — and whether that declaration and the child's subtree meet
// element's conditions in turn. Every other attribution answers false:
//
//   - a skip Wildcard: the child is ·skipped· with its whole subtree
//     (§3.3.4.6 clause 3.2), which no clause of it decides;
//   - a strict or lax Wildcard, or the {open content}: the child is governed by
//     the top-level declaration its name ·resolves· to, if any, and
//     cvc-complex-type clause 5 (§3.4.4.2) then asks that declaration to be
//     consistent with the ·context-determined· one, which the walk never
//     checks; with none, the child is the untyped element shape;
//   - an element particle whose {term} carries another name: cvc-accept
//     clause 2.3.2 admitted the child as a ·substitution group· member, whose
//     {substitution group exclusions} and the head's {disallowed
//     substitutions} the walk never checks.
func (g *subtreeGate) child(m *xsd.Matcher, start xml.StartElement, constrained bool) bool {
	name := expandedName(start.Name)
	a, ok := m.Next(name)
	if !ok {
		return false
	}
	d, ok := a.(xsd.ElementDeclaration)
	if !ok || d.Name() != name {
		return false
	}
	return g.element(start, d, constrained)
}

// rootStart reads dec up to the document element's start tag. A DOCTYPE — any
// xml.Directive — answers false: a DTD can default an attribute onto any
// element, or declare the unparsed entities an ENTITY value is checked against,
// and this reader sees neither.
func rootStart(dec *xml.Decoder) (xml.StartElement, bool) {
	for {
		tok, err := dec.Token()
		if err != nil {
			return xml.StartElement{}, false
		}
		switch t := tok.(type) {
		case xml.Directive:
			return xml.StartElement{}, false
		case xml.StartElement:
			return t, true
		}
	}
}

// expandedName is n as the ·expanded name· the schema indexes by. encoding/xml
// has already translated a bound prefix to its namespace name.
func expandedName(n xml.Name) xsd.QName {
	return xsd.QName{Space: n.Space, Local: n.Local}
}

// isNamespaceDeclaration reports whether a is a namespace declaration, which
// the infoset carries in [namespace attributes] and not in [attributes].
func isNamespaceDeclaration(a xml.Attr) bool {
	return a.Name.Space == "xmlns" || (a.Name.Space == "" && a.Name.Local == "xmlns")
}

// isLocationHint reports whether a is xsi:schemaLocation or
// xsi:noNamespaceSchemaLocation.
func isLocationHint(a xml.Attr) bool {
	return a.Name.Space == xsd.XMLSchemaInstanceNS &&
		(a.Name.Local == "schemaLocation" || a.Name.Local == "noNamespaceSchemaLocation")
}

// closureReaches reports whether st's closure — its {base type definition}
// chain, its {item type definition} and each of its {member type definitions},
// transitively — holds a builtin named in names. A reference the
// resolver cannot follow answers true, so an unreadable closure is excluded
// rather than admitted.
//
// Item and Members are read once, off st itself: both are derived through the
// {base type definition} chain (§3.16.2.1), so every restriction on that chain
// reports the same ones. The recursion needs no visited set: a finalized
// Schema's base chains and union memberships are acyclic (xsd's Phase B,
// checkSimpleBaseAcyclic and checkUnionMembershipAcyclic).
func closureReaches(r xsd.TypeResolver, st *xsd.SimpleType, names []string) bool {
	for t := st; t != nil; {
		if t.Name().Space == xsd.XMLSchemaNS && slices.Contains(names, t.Name().Local) {
			return true
		}
		base, err := t.Base(r)
		if err != nil {
			return true
		}
		t = base
	}
	item, err := st.Item(r)
	if err != nil {
		return true
	}
	if item != nil && closureReaches(r, item, names) {
		return true
	}
	members, err := st.Members(r)
	if err != nil {
		return true
	}
	return slices.ContainsFunc(members, func(m *xsd.SimpleType) bool { return closureReaches(r, m, names) })
}

// closureVersioned reports whether any schema document the assembly read
// carries an attribute in versioningNS, on any element. It is a conservative
// SUPERSET of the documents the #1002 GAP(parser) in parser/conditional.go can
// mis-assemble — vc:maxVersion retained where §4.2.2 excludes it — because the
// assembled components record nothing of which elements a version condition
// touched: VC/vc006.n1 is suite-invalid and walks clean for exactly that reason.
//
// Each document is re-read from its parser.AssembledDocument.Location, which
// for the loader.Dir resolver assembleCase uses is the on-disk path. A document
// that will not open or decode answers true.
func closureVersioned(report *parser.AssemblyReport) bool {
	for _, d := range report.Documents() {
		if documentVersioned(d.Location) {
			return true
		}
	}
	return false
}

// documentVersioned reports whether the document at path carries an attribute
// in versioningNS, or cannot be read to say.
func documentVersioned(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return true
	}
	defer func() { _ = f.Close() }() // read-only handle: close error cannot affect the verdict
	dec := xml.NewDecoder(f)
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return false
		}
		if err != nil {
			return true
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if slices.ContainsFunc(start.Attr, func(a xml.Attr) bool { return a.Name.Space == versioningNS }) {
			return true
		}
	}
}
