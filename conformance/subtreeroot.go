package conformance

import (
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
)

// This file holds the instance lane's one shape gate for the "valid"
// observation: an ASSESSED SUBTREE ROOT (#1841), a validation root, with or
// without content (#1855), whose every element the walk strictly assessed
// against a declaration and a type it really has, and every attribute against
// its ·governing attribute declaration· where it has one (key-sva clause 2),
// with no clause of key-sva (§3.3.4.6), cvc-elt (§3.3.4.3), cvc-type
// (§3.3.4.4) or cvc-complex-type (§3.4.4.2) left undecided unrecorded.
// instance.go's "Why an EMPTY Result is evidence of validity for ONE shape
// only" states which clause each condition discharges; this file is only the
// conditions.
//
// The gate is computed per case and stored nowhere. It re-reads the instance
// with encoding/xml and re-derives every child's ·attribution· through
// xsd.Schema.ContentMatcher, independently of the walk, so it never rests on
// what the walk did or did not record for a descendant — save for a strict
// ·wildcard particle·'s child resolving to no declaration and carrying no
// xsi:type, which it admits unread because the walk charges that child's parent
// for it (subtreeGate.resolvedChild).

// versioningNS is the XML Schema versioning namespace §4.2.2 reads vc:minVersion,
// vc:maxVersion, vc:typeAvailable, vc:typeUnavailable, vc:facetAvailable and
// vc:facetUnavailable from. parser keeps its own copy unexported; the lane
// declares the name here rather than parser exporting it for a gate that exists
// only because of a parser GAP (#1002).
const versioningNS = "http://www.w3.org/2007/XMLSchema-versioning"

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
	if !g.element(root, d) {
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
// checked against, the decoder positioned inside it, and the namespace
// declarations of every element open above that position, outermost first,
// which resolveQName reads.
type subtreeGate struct {
	schema *xsd.Schema
	dec    *xml.Decoder
	scope  []xml.Attr
}

// element reads the element whose start tag is start, and whose ·governing
// element declaration· is d, through to its end tag, and reports whether every
// condition below holds for it and, recursively, for every element under it:
//
//   - it carries no xsi:nil (plainAttributes);
//   - d is not abstract and carries no {type table} and no fixed {value
//     constraint} (assessedDeclaration);
//   - its ·governing type definition· is determined: d.{type definition}
//     resolves, and an xsi:type the element carries meets governingType's
//     conditions, the type it names then standing in for d.{type definition}
//     in the two conditions below;
//   - for a Simple Type Definition, the element carries no attribute but the
//     four xsi: ones cvc-type clause 3.1.1 excepts, and no element [[child]];
//   - for a Complex Type Definition, the conditions complex names.
func (g *subtreeGate) element(start xml.StartElement, d xsd.ElementDeclaration) bool {
	if !plainAttributes(start.Attr) || !assessedDeclaration(d) {
		return false
	}
	mark := len(g.scope)
	for _, a := range start.Attr {
		if isNamespaceDeclaration(a) {
			g.scope = append(g.scope, a)
		}
	}
	defer func() { g.scope = g.scope[:mark] }()
	td, ok := g.governingType(start, d)
	if !ok {
		return false
	}
	switch t := td.(type) {
	case *xsd.SimpleType:
		if slices.ContainsFunc(start.Attr, notExcepted) {
			return false
		}
		return g.leaf()
	case xsd.ComplexType:
		return g.complex(start, t)
	}
	return false
}

// governingType is the ·governing type definition· (key-governing-type-elem) of
// the element whose start tag is start and whose ·governing element
// declaration· is d, determined here independently of the walk, and false
// wherever the gate does not determine it. d carries no {type table}
// (assessedDeclaration), so its ·selected type definition· is d.{type
// definition}, which must resolve.
//
// With no xsi:type the selected type governs (clause 4). With one, the gate
// follows its ·instance-specified type definition· T (clause 3) where all of
// these hold, and answers false otherwise:
//
//   - the lexical names a type: it resolves as a QName against the namespace
//     bindings in scope (§3.17.6.3, cvc-resolve-instance) and the schema has a
//     top-level type definition of that name. A lexical that does not is the
//     walk's, charged under cvc-attribute clause 3 or 5 or, where the walk
//     withholds clause 3, recorded in Result.Unevaluated, so no empty Result
//     reaches the gate with it;
//   - xsd.Schema.ValidlySubstitutable answers that T ·overrides· the selected
//     type under d.{disallowed substitutions} (§3.3.4.2, key-overrides), which
//     is cvc-elt clause 4. A false is the walk's cvc-elt charge. An error is
//     validate's instanceOverride decline, which leaves the governing type
//     undetermined and records nothing, so the gate refuses it itself.
func (g *subtreeGate) governingType(start xml.StartElement, d xsd.ElementDeclaration) (xsd.TypeDefinition, bool) {
	selected, ok := g.schema.ResolvedType(d.TypeDefinition())
	if !ok {
		return nil, false
	}
	i := slices.IndexFunc(start.Attr, func(a xml.Attr) bool { return a.Name == xsiType })
	if i < 0 {
		return selected, true
	}
	name, ok := g.resolveQName(start.Attr[i].Value)
	if !ok {
		return nil, false
	}
	t, ok := g.schema.Type(name)
	if !ok {
		return nil, false
	}
	overrides, err := g.schema.ValidlySubstitutable(t, selected, d.DisallowedSubstitutions())
	if err != nil || !overrides {
		return nil, false
	}
	return t, true
}

// resolveQName maps lexical, an xs:QName lexical, to the ·expanded name· the
// namespace declarations in g.scope bind it to after whiteSpace collapse: an
// unprefixed name takes the default namespace, or none where no default
// namespace declaration is in scope. A lexical with an empty prefix or local
// part, or whose prefix no declaration in scope binds, answers false; any
// other malformation yields a name no type definition carries.
func (g *subtreeGate) resolveQName(lexical string) (xsd.QName, bool) {
	prefix, local, prefixed := strings.Cut(strings.Trim(lexical, " \t\r\n"), ":")
	decl := xml.Name{Space: "xmlns", Local: prefix}
	if !prefixed {
		prefix, local = "", prefix
		decl = xml.Name{Local: "xmlns"}
	}
	if local == "" || (prefixed && prefix == "") {
		return xsd.QName{}, false
	}
	for i := len(g.scope) - 1; i >= 0; i-- {
		if g.scope[i].Name == decl {
			return xsd.QName{Space: g.scope[i].Value, Local: local}, true
		}
	}
	return xsd.QName{Local: local}, !prefixed
}

// xsiType and xsiNil are the ·expanded names· of the built-in xsi:type and
// xsi:nil attribute declarations (§3.2.7).
var (
	xsiType = xml.Name{Space: xsd.XMLSchemaInstanceNS, Local: "type"}
	xsiNil  = xml.Name{Space: xsd.XMLSchemaInstanceNS, Local: "nil"}
)

// anyTypeName is the ·expanded name· of ·xs:anyType·.
var anyTypeName = xsd.QName{Space: xsd.XMLSchemaNS, Local: "anyType"}

// plainAttributes reports whether attrs, one start tag's attribute list, holds
// no xsi:nil.
func plainAttributes(attrs []xml.Attr) bool {
	return !slices.ContainsFunc(attrs, func(a xml.Attr) bool { return a.Name == xsiNil })
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
// which records every check it declines.
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
//     that recordedAttributeType admits, present on the element or not;
//   - every attribute the element carries that notExcepted names matches one
//     of those uses by ·expanded name· (cvc-complex-type clause 2.1) or meets
//     wildcardAttribute's conditions (clause 2.2);
//   - under a simple or an empty {content type}, there is no element
//     [[child]];
//   - under an element-only or mixed one, xsd.Schema.ContentMatcher decides it
//     and every element [[child]] meets child's conditions.
func (g *subtreeGate) complex(start xml.StartElement, t xsd.ComplexType) bool {
	if t.Abstract() {
		return false
	}
	uses := t.AttributeUses()
	for _, u := range uses {
		ad, ok := g.schema.ResolvedAttributeDeclaration(u)
		if !ok || !g.recordedAttributeType(ad) {
			return false
		}
	}
	for _, a := range start.Attr {
		if !notExcepted(a) {
			continue
		}
		n := expandedName(a.Name)
		if slices.ContainsFunc(uses, func(u xsd.AttributeUse) bool { return u.DeclarationName() == n }) {
			continue
		}
		if !g.wildcardAttribute(t, n) {
			return false
		}
	}
	switch t.ContentType().(type) {
	case xsd.SimpleContent, xsd.EmptyContent:
		return g.leaf()
	case xsd.ElementContent:
		m, ok := g.schema.ContentMatcher(t)
		if !ok {
			return false
		}
		return g.children(t, m)
	}
	return false
}

// recordedAttributeType reports whether ad.{type definition} resolves to a
// simple type: the one condition the gate puts on an attribute declaration the
// walk assesses an attribute against, an {attribute uses} member's or a
// wildcard-resolved one's.
func (g *subtreeGate) recordedAttributeType(ad xsd.AttributeDeclaration) bool {
	_, ok := g.schema.ResolvedSimpleType(ad.TypeDefinition())
	return ok
}

// wildcardAttribute reports whether an attribute named n that matches none of
// t.{attribute uses} is one whose assessment the walk decides or records
// (validate's walk.unmatchedAttribute and walk.wildcardAttribute): t has an
// {attribute wildcard} that admits n (cvc-complex-type clause 2.2, cvc-wildcard
// §3.10.4.1, xsd.Schema.AllowsAttributeWildcardName), and one of these holds:
//
//   - its {process contents} is skip: the attribute is ·skipped·
//     (key-skipped), assessed against nothing, and cvc-assess-elt clause 2.2
//     leaves it unassessed with nothing to decide;
//   - n ·resolves· to a top-level attribute declaration (key-governing-ad
//     clause 3) that recordedAttributeType admits, and the attribute's
//     ·locally declared type· within t is ·absent· (locallyDeclaredAttribute
//     answers false), which makes cvc-complex-type clause 5 vacuous for it:
//     the walk charges or records cvc-attribute clauses 3 and 4 against the
//     declaration, under lax and strict alike;
//   - n resolves to none under lax: the attribute has no ·governing attribute
//     declaration· and is not assessed, which charges nothing (sic-e-outcome
//     clause 1.1.3 reaches only a ·wildcard particle·), and clause 5 is
//     vacuous for want of a ·governing type definition·.
//
// It refuses n resolving to a declaration where the ·locally declared type· is
// not ·absent· — a base's attribute use a restriction prohibited (key-ldt-att
// case 3): clause 5 then asks the declaration's {type definition} to be
// ·validly substitutable· for that use's, which the walk never checks. It
// refuses n resolving to none under strict. The walk charges nothing there
// either, on the same reading, and records nothing, a reading the suite does
// not share for seven of its invalid cases (#1912). A name the wildcard does
// not admit, or a type with no {attribute wildcard}, is refused too, though the
// walk charges clause 2 for both.
func (g *subtreeGate) wildcardAttribute(t xsd.ComplexType, n xsd.QName) bool {
	wild, ok := t.AttributeWildcard()
	if !ok || !g.schema.AllowsAttributeWildcardName(wild, n) {
		return false
	}
	pc := wild.ProcessContents()
	if pc == xsd.ProcessSkip {
		return true
	}
	ad, ok := g.schema.Attribute(n)
	if !ok {
		return pc == xsd.ProcessLax
	}
	return g.recordedAttributeType(ad) && !g.locallyDeclaredAttribute(t, n)
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

// children reads the content of an element governed by t through to its end
// tag, advancing m, t's ContentMatcher, over each element [[child]] in document
// order, and reports whether every child meets child's conditions and m
// accepts the whole sequence.
func (g *subtreeGate) children(t xsd.ComplexType, m *xsd.Matcher) bool {
	for {
		tok, err := g.dec.Token()
		if err != nil {
			return false
		}
		switch s := tok.(type) {
		case xml.StartElement:
			if !g.child(t, m, s) {
				return false
			}
		case xml.EndElement:
			return m.Accepting()
		}
	}
}

// child reads one element [[child]] whose start tag is start, of an element
// governed by t, through to its end tag, and reports whether m ·attributes· it
// (§3.4.4.4) to one of these, and the child meets that arm's conditions:
//
//   - an element particle whose {term} carries the child's own ·expanded
//     name·: that {term} is its ·context-determined declaration· (§3.3.4.6
//     key-governing-ed clause 2), which with the child's subtree must meet
//     element's conditions. The ·locally declared type· (key-ldt-elem case 2)
//     is that declaration's own {type definition}, so cvc-complex-type clause
//     5 holds wherever cvc-elt clause 4 does;
//   - an element particle whose {term} D carries another name: cvc-accept
//     clause 2.3.2 admitted the child as a member of D's ·substitution group·,
//     the Matcher deciding D top-level, D.{disallowed substitutions}, and
//     cos-equiv-derived-ok-rec (§3.3.6.3) clauses 2.1 to 2.3 for the top-level
//     declaration S the child's name ·resolves· to. S is the child's
//     ·context-determined declaration· (key-governing-ed clause 2), and S with
//     the child's subtree must meet element's conditions where substitutable
//     holds. The ·locally declared type· (key-ldt-elem case 2, S ·implicitly
//     contained·, key-impl-cont) is S's own {type definition}, so
//     cvc-complex-type clause 5 holds wherever cvc-elt clause 4 does. A child
//     carrying D's own name is the first arm's, cvc-accept clause 2.3.1
//     attributing it to D, where element refuses an ·abstract· D;
//   - a skip Wildcard, or the {open content} with a skip {wildcard}: the child
//     is ·skipped· with its whole subtree (key-sva clause 3.2, cvc-assess-elt
//     clause 2), which is read past unchecked. A skipped child has no
//     [validity], so it cannot block its parent's `valid` (sic-e-outcome
//     clauses 1.1.2 and 1.1.3, the latter naming a strict particle alone), and
//     with no ·governing type definition· it binds no ID or IDREF (key-eas
//     clause 3). Its NAME is still checked: m.Next admits it only where the
//     wildcard's namespace constraint does (cvc-wildcard clause 1, and
//     cvc-complex-content clause 2.4 or 3.4 for the {open content}). The
//     {open content} half is validate's reading, not the spec's words
//     (validate's walk.childGoverning, #1969);
//   - a strict or lax Wildcard, or the {open content} with a strict or lax
//     {wildcard}: resolvedChild's conditions.
func (g *subtreeGate) child(t xsd.ComplexType, m *xsd.Matcher, start xml.StartElement) bool {
	name := expandedName(start.Name)
	a, ok := m.Next(name)
	if !ok {
		return false
	}
	switch at := a.(type) {
	case xsd.ElementDeclaration:
		if at.Name() == name {
			return g.element(start, at)
		}
		s, ok := g.schema.Element(name)
		return ok && g.substitutable(s, at) && g.element(start, s)
	case xsd.Wildcard:
		pc := at.ProcessContents()
		if pc == xsd.ProcessSkip {
			return g.dec.Skip() == nil
		}
		return g.resolvedChild(t, start, pc == xsd.ProcessStrict)
	case *xsd.OpenContent:
		if at.Wildcard().ProcessContents() == xsd.ProcessSkip {
			return g.dec.Skip() == nil
		}
		return g.resolvedChild(t, start, false)
	}
	return false
}

// substitutable reports whether the gate takes the Matcher's cvc-accept clause
// 2.3.2 admission of the member s for the head d as the spec's answer. It
// refuses one shape: s.{type definition}'s {base type definition} chain takes a
// step from a Simple Type Definition before it reaches d.{type definition}, and
// restriction is in cos-equiv-derived-ok-rec clause 2.3's blocking union —
// d.{disallowed substitutions}, or the {prohibited substitutions} of a
// Complex Type Definition strictly between the two. The union's second member,
// d.{type definition}'s own {prohibited substitutions}, is empty wherever such a
// chain reaches it: a simple type's chain meets no complex type but
// ·xs:anyType· (§3.4.7). An unresolvable type on either side is refused too.
//
// GAP(xsd): xsd's derivationAdmitsSubstitution counts no {derivation method}
// for a Simple Type Definition step, so the Matcher admits a member whose
// ·derivation· takes a simple restriction step while the blocking union holds
// restriction. The suite reads that step as restriction: ElemDecl
// disallowedsubst00501m2 Negative and MS-Element elemT063.i are invalid for it
// and walk clean (#1942).
//
// The walk stops at d.{type definition}, by name, or by identity for an
// anonymous simple type s inherits from d; an anonymous complex one is walked
// past, which only refuses more. A complex chain reaching ·xs:anyType· short of
// d.{type definition} involves no {derivation method}; it takes no simple step
// and answers true. s.{type definition}'s own {prohibited substitutions} is not
// in the union.
func (g *subtreeGate) substitutable(s, d xsd.ElementDeclaration) bool {
	head, ok := g.schema.ResolvedType(d.TypeDefinition())
	if !ok {
		return false
	}
	cur, ok := g.schema.ResolvedType(s.TypeDefinition())
	if !ok {
		return false
	}
	blocked := slices.Contains(d.DisallowedSubstitutions(), xsd.DerivationRestriction)
	for step := 0; !sameType(cur, head); step++ {
		c, complex := cur.(xsd.ComplexType)
		if !complex {
			return !blocked
		}
		if c.Name() == anyTypeName {
			return true
		}
		prohibits := slices.Contains(c.ProhibitedSubstitutions(), xsd.DerivationRestriction)
		blocked = blocked || (step > 0 && prohibits)
		cur, ok = g.schema.ResolvedType(c.Base())
		if !ok {
			return false
		}
	}
	return true
}

// sameType reports whether a and b are one type definition: the same non-empty
// ·expanded name·, or the same anonymous Simple Type Definition. Two anonymous
// Complex Type Definitions are reported distinct.
func sameType(a, b xsd.TypeDefinition) bool {
	if a.Name() != (xsd.QName{}) {
		return a.Name() == b.Name()
	}
	sa, ok := a.(*xsd.SimpleType)
	if !ok {
		return false
	}
	sb, ok := b.(*xsd.SimpleType)
	return ok && sa == sb
}

// resolvedChild reads through to its end tag a child whose start tag is start,
// of an element governed by t, which is ·attributed to· a strict or lax
// Wildcard or to t's {open content} with a strict or lax {wildcard}, and
// reports whether one of these holds:
//
//   - its ·expanded name· ·resolves· to a top-level element declaration d
//     (key-governing-ed clauses 3 and 4), its ·locally declared type· within
//     t is ·absent· (locallyDeclared answers false for the names
//     substitutionHeads gives), and d and the child's subtree meet element's
//     conditions. The ldt being ·absent·, cvc-complex-type clause 5 is
//     vacuous for the child; where it is not, clause 5 asks the child's
//     ·governing type definition· to be ·validly substitutable· for it, which
//     the walk never checks;
//   - strictParticle holds — the child is ·attributed to· a strict ·wildcard
//     particle· — its name resolves to none, and it carries no xsi:type: the
//     child is ·laxly assessed· with a ·governing type definition· of none, so
//     clause 5 is vacuous, and the walk charges e-validity clause 1.1.3
//     (validate's walk.unresolvedStrictWildcardChild) for the parent, so no
//     empty Result reaches the gate with it and the gate reads nothing below
//     it.
//
// It refuses a name that resolves to none under a lax Wildcard or the {open
// content}: the walk charges nothing for the child, e-validity clause 1.1.3
// naming a strict ·wildcard particle· alone, and what its ·laxly assessed·
// subtree costs its ancestors is #1911's.
//
// GAP(conformance): an unresolved strict child with an xsi:type is refused,
// whatever type the xsi:type names. Where it resolves, that type is the
// child's ·governing type definition· (key-governing-type-elem clause 8), so
// its parent's cvc-assess-elt clause 3.1 has it ·strictly assessed· against
// that type (key-sva clause 1.2) and the spec decides it; the walk's e-validity
// clause 1.1.3 charge is gone, and the gate does not read a subtree governed by
// a type with no declaration. Its one reader, execInstanceCase, then Fails the
// case: a suite-valid case of this shape scores no pass, and none a false one.
func (g *subtreeGate) resolvedChild(t xsd.ComplexType, start xml.StartElement, strictParticle bool) bool {
	d, ok := g.schema.Element(expandedName(start.Name))
	if !ok {
		if !strictParticle || slices.ContainsFunc(start.Attr, func(a xml.Attr) bool { return a.Name == xsiType }) {
			return false
		}
		return g.dec.Skip() == nil
	}
	if g.locallyDeclared(t, g.substitutionHeads(d)) {
		return false
	}
	return g.element(start, d)
}

// substitutionHeads is d's ·expanded name· followed by the name of every
// declaration its {substitution group affiliations} chain reaches, transitively
// (cos-equiv-derived-ok-rec clause 2.2): the names of every declaration whose
// ·substitution group· (key-eq) d may be a member of. It is a SUPERSET of that
// group's heads: clauses 2.1 and 2.3, the {disallowed substitutions} and
// derivation-method conditions, are not read, so a head d is not
// ·substitutable· for still names itself here, which only makes locallyDeclared
// refuse more.
//
// A name already listed is not listed again, which also ends the walk. An
// affiliation that resolves to no top-level declaration is listed and not
// followed.
func (g *subtreeGate) substitutionHeads(d xsd.ElementDeclaration) []xsd.QName {
	names := []xsd.QName{d.Name()}
	for i := 0; i < len(names); i++ {
		h, ok := g.schema.Element(names[i])
		if !ok {
			continue
		}
		for _, a := range h.SubstitutionGroupAffiliationNames() {
			if !slices.Contains(names, a) {
				names = append(names, a)
			}
		}
	}
	return names
}

// locallyDeclared reports whether the ·locally declared type· (key-ldt-elem)
// within t of an element whose name, with its ·substitution group· heads, is
// names (substitutionHeads) may be non-·absent·: some Complex Type Definition
// on onChain's walk from t has a content model that ·contains· (key-contain-xpx,
// termContains) an element declaration named in names. A declaration named by a
// head is one whose ·substitution group· the element's declaration may be in,
// which key-impl-cont makes ·implicitly contained·. A local declaration sharing
// a head's name counts too, though no local declaration has a ·substitution
// group· member but itself: the superset only refuses more.
func (g *subtreeGate) locallyDeclared(t xsd.ComplexType, names []xsd.QName) bool {
	return g.onChain(t, func(c xsd.ComplexType) bool {
		ec, ok := c.ContentType().(xsd.ElementContent)
		return ok && g.termContains(ec.Particle.Term(), names)
	})
}

// locallyDeclaredAttribute reports whether the ·locally declared type·
// (key-ldt-att) within t of an attribute named n may be non-·absent·: some
// Complex Type Definition on onChain's walk from t has an {attribute uses}
// member whose {attribute declaration} is named n (case 2). For an attribute
// matching none of t.{attribute uses}, that is a base's use the restriction
// t, or a restriction between t and that base, prohibited.
func (g *subtreeGate) locallyDeclaredAttribute(t xsd.ComplexType, n xsd.QName) bool {
	return g.onChain(t, func(c xsd.ComplexType) bool {
		return slices.ContainsFunc(c.AttributeUses(), func(u xsd.AttributeUse) bool { return u.DeclarationName() == n })
	})
}

// onChain reports whether holds answers true for t or for a Complex Type
// Definition on t's {base type definition} chain, which the ·locally declared
// type· recursion (key-ldtype case 3) walks, stopping at ·xs:anyType· (case 1,
// never offered to holds) or at a Simple Type Definition, which declares no
// element and no attribute. A {base type definition} that does not resolve
// answers true, so an unreadable chain is refused rather than read as ·absent·.
// The walk needs no visited set: a finalized Schema's base chains are acyclic
// but for ·xs:anyType·'s own, where it stops.
func (g *subtreeGate) onChain(t xsd.ComplexType, holds func(xsd.ComplexType) bool) bool {
	for c := t; c.Name() != anyTypeName; {
		if holds(c) {
			return true
		}
		base, ok := g.schema.ResolvedType(c.Base())
		if !ok {
			return true
		}
		next, complex := base.(xsd.ComplexType)
		if !complex {
			return false
		}
		c = next
	}
	return false
}

// termContains reports whether term, a particle's {term}, is or ·contains·,
// directly or indirectly (key-contain-dpt, key-contain-ipx), an element
// declaration named in names: an <element ref> by the name it references, a
// <group ref> through the referenced definition's {model group}. A <group ref>
// that resolves to no definition answers true. The recursion needs no visited
// set: finalize rejects a circular group reference (mg-props-correct clause
// 2).
func (g *subtreeGate) termContains(term xsd.TermOrRef, names []xsd.QName) bool {
	switch t := term.(type) {
	case xsd.ElementDeclarationRef:
		return slices.Contains(names, t.Name)
	case xsd.ModelGroupRef:
		def, ok := g.schema.ModelGroup(t.Name)
		return !ok || g.groupContains(def.ModelGroup(), names)
	case xsd.ResolvedTerm:
		switch r := t.Term.(type) {
		case xsd.ElementDeclaration:
			return slices.Contains(names, r.Name())
		case xsd.ModelGroup:
			return g.groupContains(r, names)
		case xsd.Wildcard:
			return false
		}
	}
	return true
}

// groupContains reports whether a member of mg.{particles} is or contains an
// element declaration named in names (termContains).
func (g *subtreeGate) groupContains(mg xsd.ModelGroup, names []xsd.QName) bool {
	return slices.ContainsFunc(mg.Particles(), func(p xsd.Particle) bool { return g.termContains(p.Term(), names) })
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

// closureVersioned reports whether any schema document the assembly read
// carries an attribute in versioningNS, on any element. It is a conservative
// SUPERSET of the documents the #1002 GAP(parser) in parser/conditional.go can
// mis-assemble — vc:maxVersion retained where §4.2.2 excludes it — because the
// assembled components record nothing of which elements a version condition
// touched: VC/vc006.n1 is suite-invalid and walks clean for exactly that reason.
//
// Each document is re-read from its parser.AssembledDocument.Location, which
// for the pinnedResolver assembleCase uses is an on-disk path. A document
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
