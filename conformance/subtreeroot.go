package conformance

import (
	"encoding/xml"
	"errors"
	"io"
	"path/filepath"
	"slices"

	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
)

// This file holds the instance lane's third shape gate for the "valid"
// observation (#1841): an ASSESSED SUBTREE ROOT, a validation root WITH content
// whose every element and attribute the walk strictly assessed against a
// declaration and a type it really has, with no clause of key-sva (§3.3.4.6),
// cvc-elt (§3.3.4.3), cvc-type (§3.3.4.4) or cvc-complex-type (§3.4.4.2) left
// undecided unrecorded. instance.go's "Why an EMPTY Result is evidence of
// validity for THREE shapes only" states which clause each condition
// discharges; this file is only the conditions, and simpleleaf.go holds the
// helpers it shares with the two leaf-root gates.
//
// The gate is computed per case and stored nowhere. It re-reads the instance
// with encoding/xml and re-derives every child's ·attribution· through
// xsd.Schema.ContentMatcher, independently of the walk, so it never rests on
// what the walk did or did not record for a descendant.

// assessedSubtreeRoot reports whether the instance document at doc, against
// schema as assembled into report, has the assessed-subtree-root shape an empty
// validate.Result may be read as "valid" for: a root with at least one element
// or character information item [[child]] whose subtree meets every condition
// subtreeGate.element names, in a document with no DOCTYPE, against an assembly no
// version condition touched. Any failure to establish a condition — an
// unreadable document, a decoder error, an unresolvable component — is a false,
// never a guess.
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
	if !g.element(root, d) || !g.rootHadContent {
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
// checked against and the decoder positioned inside it. rootHadContent is set
// by the root's own content reading alone, which is the only depth that asks.
type subtreeGate struct {
	schema         *xsd.Schema
	dec            *xml.Decoder
	depth          int
	rootHadContent bool
}

// element reads the element whose start tag is start, and whose ·governing
// element declaration· is d, through to its end tag, and reports whether every
// condition below holds for it and, recursively, for every element under it:
//
//   - it carries no xsi:type and no xsi:nil (plainAttributes);
//   - d is not abstract and carries no {type table}, no {identity-constraint
//     definitions} and no fixed {value constraint} (assessedDeclaration);
//   - d.{type definition} resolves;
//   - for a Simple Type Definition, its closure reaches none of
//     idOrEntityOrNotation, the element carries no attribute but the four
//     xsi: ones cvc-type clause 3.1.1 excepts, and no element [[child]];
//   - for a Complex Type Definition, the conditions complex names.
func (g *subtreeGate) element(start xml.StartElement, d xsd.ElementDeclaration) bool {
	if !plainAttributes(start.Attr) || !assessedDeclaration(d) {
		return false
	}
	td, ok := g.schema.ResolvedType(d.TypeDefinition())
	if !ok {
		return false
	}
	switch t := td.(type) {
	case *xsd.SimpleType:
		if closureReaches(g.schema, t) || slices.ContainsFunc(start.Attr, notExcepted) {
			return false
		}
		return g.leaf()
	case xsd.ComplexType:
		return g.complex(start, t)
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
// definition}), no {identity-constraint definitions} (clause 6), and no fixed
// {value constraint} (clause 5.2.2). A default one is admitted: clause 5.1 is
// the walk's.
func assessedDeclaration(d xsd.ElementDeclaration) bool {
	if d.Abstract() {
		return false
	}
	if _, ok := d.TypeTable(); ok {
		return false
	}
	if len(d.IdentityConstraints()) > 0 {
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
//     none of idOrEntityOrNotation, present on the element or not;
//   - every attribute the element carries that notExcepted names matches one
//     of those uses by ·expanded name· (cvc-complex-type clause 2.1), so none
//     is ·attributed to· the {attribute wildcard};
//   - under a simple {content type}, its {simple type definition}'s closure
//     reaches none of idOrEntityOrNotation and there is no element [[child]];
//   - under an empty one, there is no element [[child]];
//   - under an element-only or mixed one, xsd.Schema.ContentMatcher decides it
//     and every element [[child]] meets child's conditions.
func (g *subtreeGate) complex(start xml.StartElement, t xsd.ComplexType) bool {
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
		if !ok || closureReaches(g.schema, st) {
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
		if closureReaches(g.schema, ct.SimpleType) {
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
		return g.children(m)
	}
	return false
}

// leaf reads an element's content through to its end tag and reports whether
// no element started inside it, noting on the way whether the root held any
// character data.
func (g *subtreeGate) leaf() bool {
	for {
		tok, err := g.dec.Token()
		if err != nil {
			return false
		}
		switch tok.(type) {
		case xml.StartElement:
			return false
		case xml.CharData:
			g.noteContent()
		case xml.EndElement:
			return true
		}
	}
}

// children reads an element's content through to its end tag, advancing m over
// each element [[child]] in document order, and reports whether every child
// meets assessedChild and m accepts the whole sequence.
func (g *subtreeGate) children(m *xsd.Matcher) bool {
	for {
		tok, err := g.dec.Token()
		if err != nil {
			return false
		}
		switch t := tok.(type) {
		case xml.CharData:
			g.noteContent()
		case xml.StartElement:
			g.noteContent()
			if !g.child(m, t) {
				return false
			}
		case xml.EndElement:
			return m.Accepting()
		}
	}
}

// noteContent records a character or element information item [[child]] of
// the root; below the root it records nothing.
func (g *subtreeGate) noteContent() {
	if g.depth == 0 {
		g.rootHadContent = true
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
func (g *subtreeGate) child(m *xsd.Matcher, start xml.StartElement) bool {
	name := expandedName(start.Name)
	a, ok := m.Next(name)
	if !ok {
		return false
	}
	d, ok := a.(xsd.ElementDeclaration)
	if !ok || d.Name() != name {
		return false
	}
	g.depth++
	ok = g.element(start, d)
	g.depth--
	return ok
}
