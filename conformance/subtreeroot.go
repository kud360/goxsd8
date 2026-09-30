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
	mark := len(g.scope)
	for _, a := range start.Attr {
		if isNamespaceDeclaration(a) {
			g.scope = append(g.scope, a)
		}
	}
	defer func() { g.scope = g.scope[:mark] }()
	constrained = constrained || len(d.IdentityConstraints()) > 0
	td, ok := g.governingType(start, d)
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
//   - blockingUnread does not hold;
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
	blocked := d.DisallowedSubstitutions()
	if blockingUnread(t, selected, blocked) {
		return nil, false
	}
	overrides, err := g.schema.ValidlySubstitutable(t, selected, blocked)
	if err != nil || !overrides {
		return nil, false
	}
	return t, true
}

// blockingUnread reports whether xsd.Schema.ValidlySubstitutable may answer
// TRUE for the ·instance-specified type definition· t against the ·selected
// type definition· selected where the blocking keywords in blocked make the
// answer FALSE. The walk reads that answer as cvc-elt clause 4 decided and
// records nothing, so the gate refuses both shapes xsd decides without reading
// blocked:
//
//   - selected is ·xs:anyType·, which validlyDerived answers true for before
//     reading blocked (the GAP(xsd) at its xs:anyType shortcut). For a complex
//     t, extension or restriction in blocked can fail cos-ct-derived-ok clause
//     1 on a step of t's {base type definition} chain, which the gate does not
//     walk (xs:anyType's own {prohibited substitutions} is the empty set,
//     §3.4.7, and adds nothing); for a simple t, restriction in blocked can
//     fail cos-st-derived-ok clause 2.1, the one keyword that constraint
//     reads.
//   - selected is a Simple Type Definition and restriction is in blocked:
//     derivedOKSimple runs cos-st-derived-ok under the empty blocking set (the
//     GAP(xsd) on ValidlySubstitutable), reached for a simple t directly and
//     for a complex t through cos-ct-derived-ok clause 2.3.2.2. This arm is
//     conservative for a complex t: derivedOKComplex decides clause 1 and
//     clause 2.2 exactly where t's {base type definition} chain reaches
//     selected before any other simple type, and the gate refuses those too
//     rather than walk the chain. MS-Element elemT058.v, a simpleContent
//     extension of the declared simple type itself, is such a valid case the
//     gate declines.
//
// Neither applies where t is selected itself, which clause 1 of both
// constraints admits whatever blocked holds; t is a top-level type, so a name
// equal to selected's is that identity.
func blockingUnread(t, selected xsd.TypeDefinition, blocked []xsd.DerivationMethod) bool {
	if t.Name() == selected.Name() {
		return false
	}
	restriction := slices.Contains(blocked, xsd.DerivationRestriction)
	if _, simple := selected.(*xsd.SimpleType); simple {
		return restriction
	}
	if selected.Name() != anyTypeName {
		return false
	}
	if _, simple := t.(*xsd.SimpleType); simple {
		return restriction
	}
	return restriction || slices.Contains(blocked, xsd.DerivationExtension)
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
//     that recordedAttributeType admits, present on the element or not;
//   - every attribute the element carries that notExcepted names matches one
//     of those uses by ·expanded name· (cvc-complex-type clause 2.1) or meets
//     wildcardAttribute's conditions (clause 2.2);
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
		if !ok || !g.recordedAttributeType(ad) {
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
		if slices.ContainsFunc(uses, func(u xsd.AttributeUse) bool { return u.DeclarationName() == n }) {
			continue
		}
		if !g.wildcardAttribute(t, n) {
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

// recordedAttributeType reports whether ad.{type definition} resolves to a
// simple type whose closure reaches none of walkUnrecorded: the one condition
// the gate puts on an attribute declaration the walk assesses an attribute
// against, an {attribute uses} member's or a wildcard-resolved one's.
func (g *subtreeGate) recordedAttributeType(ad xsd.AttributeDeclaration) bool {
	st, ok := g.schema.ResolvedSimpleType(ad.TypeDefinition())
	return ok && !closureReaches(g.schema, st, walkUnrecorded)
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
//     clause 3) that recordedAttributeType admits: the walk charges or records
//     cvc-attribute clauses 3 and 4 against it, under lax and strict alike;
//   - n resolves to none under lax: the attribute has no ·governing attribute
//     declaration· and is not assessed, which charges nothing (sic-e-outcome
//     clause 1.1.3 reaches only a ·wildcard particle·).
//
// It refuses n resolving to none under strict. The walk charges nothing there
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
	return g.recordedAttributeType(ad)
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
