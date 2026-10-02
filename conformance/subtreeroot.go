package conformance

import (
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kud360/goxsd8/internal/xmldecl"
	"github.com/kud360/goxsd8/internal/xmlenc"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/parser/xmltree"
	"github.com/kud360/goxsd8/xsd"
)

// This file holds the instance lane's one shape gate for the "valid"
// observation: an ASSESSED SUBTREE ROOT (#1841), a validation root, with or
// without content (#1855), whose every element the walk strictly assessed
// against a type it really has, through a declaration or, for a wildcard's
// child resolving none, an xsi:type (#1978), or laxly assessed against
// ·xs:anyType· (#1911), and every attribute against its ·governing attribute
// declaration· where it has one (key-sva clause 2), with no clause of
// key-sva (§3.3.4.6), cvc-elt (§3.3.4.3), cvc-type (§3.3.4.4) or
// cvc-complex-type (§3.4.4.2) left undecided unrecorded. instance.go's "Why
// an EMPTY Result is evidence of validity for ONE shape only" states which
// clause each condition discharges; this file is only the conditions.
//
// The gate is computed per case and stored nowhere. It re-reads the instance
// with encoding/xml, decoding a byte-order-marked UTF-16 document and admitting
// a 1.x version label as parser/xmltree does (rawDecoder), and re-derives every
// child's ·attribution· through xsd.Schema.ContentMatcher, independently of the
// walk, so it never rests on what the walk did or did not record for a
// descendant — save for a strict ·wildcard particle·'s child resolving to no
// declaration and carrying no xsi:type, which it admits unread because the walk
// charges that child's parent for it (subtreeGate.resolvedChild).

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
// whose DTD, if any, defaults no attribute (rootStart), against an assembly no
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
	dec := rawDecoder(rc)
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
//   - an xsi:nil it carries has an ·actual value· (nilValue);
//   - d is not abstract and carries no {type table} (assessedDeclaration);
//   - its ·governing type definition· is determined: d.{type definition}
//     resolves, and an xsi:type the element carries meets governingType's
//     conditions, the type it names then standing in for d.{type definition};
//   - the element and its subtree meet governed's conditions against that
//     type, the element being ·nilled· (key-nilled) where d.{nillable} is true
//     and its xsi:nil's ·actual value· is true.
//
// An xsi:nil is otherwise the walk's: validate's nilCheck charges cvc-elt
// clause 3.1 for one on a declaration whose {nillable} is false, whatever its
// value, and clause 3.2.3.2 for a ·nilled· element under a fixed {value
// constraint}; its contentCheck charges clause 3.2.3.1 for a ·nilled· element's
// character or element [[child]]. An xsi:nil whose ·actual value· is false is
// clause 3.2.2, which holds as 3.2.1 does, and the element is read as if it
// carried none.
func (g *subtreeGate) element(start xml.StartElement, d xsd.ElementDeclaration) bool {
	isNil, ok := nilValue(start.Attr)
	if !ok || !assessedDeclaration(d) {
		return false
	}
	defer g.enter(start)()
	td, ok := g.governingType(start, d)
	if !ok {
		return false
	}
	return g.governed(start, td, isNil && d.Nillable())
}

// governed reads through to its end tag an element whose start tag is start and
// whose ·governing type definition· is td, ·nilled· where nilled is true, and
// reports whether:
//
//   - for a Simple Type Definition, the element carries no attribute but the
//     four xsi: ones cvc-type clause 3.1.1 excepts, and no element [[child]];
//   - for a Complex Type Definition, it meets complex's conditions.
func (g *subtreeGate) governed(start xml.StartElement, td xsd.TypeDefinition, nilled bool) bool {
	switch t := td.(type) {
	case *xsd.SimpleType:
		if slices.ContainsFunc(start.Attr, notExcepted) {
			return false
		}
		return g.leaf()
	case xsd.ComplexType:
		return g.complex(start, t, nilled)
	}
	return false
}

// enter pushes the namespace declarations start carries onto g.scope, and
// returns the function that pops them again at start's end tag.
func (g *subtreeGate) enter(start xml.StartElement) func() {
	mark := len(g.scope)
	for _, a := range start.Attr {
		if isNamespaceDeclaration(a) {
			g.scope = append(g.scope, a)
		}
	}
	return func() { g.scope = g.scope[:mark] }
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

// nilValue reports the ·actual value· of the xsi:nil attrs, one start tag's
// attribute list, carries — false where it carries none — and false for ok
// where its lexical has no ·actual value·: after the whiteSpace collapse
// xs:boolean fixes, it is none of boolean-lexical-mapping's four literals
// (Datatypes §3.3.2.2), the type §3.2.7's built-in declaration gives xsi:nil.
//
// The gate refuses such a lexical wherever it sits. Under a ·governing element
// declaration· the walk charges it (validate's nilCheck, cvc-elt clause 3.1 or
// 3.2), so no empty Result reaches the gate with it.
//
// GAP(conformance): on a ·laxly assessed· element (laxlyAssessed), which has no
// declaration to be ·nilled· against, such an xsi:nil is still governed by
// that built-in declaration (key-governing-ad) and so not ·valid· (cvc-attribute
// clause 3), which the walk charges nothing for and records nothing of. The
// refusal leaves execInstanceCase Failing the case: a suite-invalid case of this
// shape scores no pass, and none a false one.
func nilValue(attrs []xml.Attr) (value, ok bool) {
	i := slices.IndexFunc(attrs, func(a xml.Attr) bool { return a.Name == xsiNil })
	if i < 0 {
		return false, true
	}
	switch strings.Trim(attrs[i].Value, " \t\r\n") {
	case "true", "1":
		return true, true
	case "false", "0":
		return false, true
	}
	return false, false
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
// not decide at depth: {abstract} false (clause 2, charged at the root alone)
// and no {type table} (clause 4's ·selected type definition· is then d.{type
// definition}). A {value constraint} of either variety is admitted, at every
// depth: clause 5.1 (an element with no [[children]], default or fixed) and
// clause 5.2.2 (a fixed one over [[children]]: 5.2.2.1 no element children,
// 5.2.2.2.1 a mixed type's lexical match, 5.2.2.2.2 a simple type's value
// equality) are the walk's (validate's contentCheck.defaultValid and
// contentCheck.fixedValue), which records the one comparison it declines.
// {nillable} is admitted, either way: clause 3 is the walk's — 3.1, 3.2 and
// 3.2.3.2 in validate's nilCheck, 3.2.3.1 in its contentCheck — and element
// reads a ·nilled· element's [[children]] as complex says.
// {identity-constraint definitions} are admitted too: clause 6
// (cvc-identity-constraint, §3.11.4) is the walk's, which records every check
// it declines.
func assessedDeclaration(d xsd.ElementDeclaration) bool {
	if d.Abstract() {
		return false
	}
	_, ok := d.TypeTable()
	return !ok
}

// complex reads an element governed by the Complex Type Definition t, ·nilled·
// where nilled is true, through to its end tag, and reports whether:
//
//   - t.{abstract} is false (cvc-type clause 2);
//   - every one of t.{attribute uses} resolves to an {attribute declaration}
//     that recordedAttributeType admits, present on the element or not;
//   - every attribute the element carries that notExcepted names matches one
//     of those uses by ·expanded name· (cvc-complex-type clause 2.1) or meets
//     wildcardAttribute's conditions (clause 2.2);
//   - for a ·nilled· element, there is no element [[child]]: cvc-complex-type
//     clause 1 applies only to an element that is not ·nilled·, so the
//     {content type} is not read, and cvc-elt clause 3.2.3.1, no character or
//     element [[child]], is the walk's (validate's contentCheck);
//   - otherwise, under a simple or an empty {content type}, there is no
//     element [[child]];
//   - otherwise, under an element-only or mixed one, xsd.Schema.ContentMatcher
//     decides it and every element [[child]] meets child's conditions.
func (g *subtreeGate) complex(start xml.StartElement, t xsd.ComplexType, nilled bool) bool {
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
	if nilled {
		return g.leaf()
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
//     5 holds wherever cvc-elt clause 4 does for a declaration with no {type
//     table}, the only kind element admits (assessedDeclaration). Under a {type
//     table} it need not: an xsi:type ·overriding· a selection that is not the
//     declared type can fail it (validate's walk.locallyDeclaredType);
//   - an element particle whose {term} D carries another name: cvc-accept
//     clause 2.3.2 admitted the child as a member of D's ·substitution group·,
//     the Matcher deciding D top-level, D.{disallowed substitutions}, and
//     cos-equiv-derived-ok-rec (§3.3.6.3) clauses 2.1 to 2.3 for the top-level
//     declaration S the child's name ·resolves· to. S is the child's
//     ·context-determined declaration· (key-governing-ed clause 2), and S with
//     the child's subtree must meet element's conditions. The ·locally
//     declared type· (key-ldt-elem case 2, S ·implicitly contained·,
//     key-impl-cont) is S's own {type definition}, so cvc-complex-type clause
//     5 holds wherever cvc-elt clause 4 does, on the first arm's terms. A
//     child carrying D's own name is the first arm's, cvc-accept clause 2.3.1
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
//     {wildcard}: resolvedChild's conditions, and for the {open content} the
//     child's ·locally declared type· within t is ·absent·
//     (xsd.Schema.LocallyDeclaredElementType answers false).
//
// GAP(conformance): a child ·attributed to· the {open content} with a strict or
// lax {wildcard} and a non-·absent· ·locally declared type· is refused,
// resolved name or not. key-governing-ed clause 4.3 gives it no ·governing
// element declaration·, and that type, or an xsi:type ·overriding· it, governs
// it (key-governing-type-elem clauses 6 and 7, validate's
// walk.localGovernance); resolvedChild reads the top-level declaration or
// ·xs:anyType· instead, so it vets a type the walk does not assess against.
// Its one reader, execInstanceCase, then Fails the case: a suite-valid case of
// this shape scores no pass, and none a false one (#2080).
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
		return ok && g.element(start, s)
	case xsd.Wildcard:
		pc := at.ProcessContents()
		if pc == xsd.ProcessSkip {
			return g.dec.Skip() == nil
		}
		return g.resolvedChild(start, pc == xsd.ProcessStrict)
	case *xsd.OpenContent:
		if at.Wildcard().ProcessContents() == xsd.ProcessSkip {
			return g.dec.Skip() == nil
		}
		if _, local := g.schema.LocallyDeclaredElementType(t, name); local {
			return false
		}
		return g.resolvedChild(start, false)
	}
	return false
}

// resolvedChild reads through to its end tag a child whose start tag is start,
// which is ·attributed to· a strict or lax Wildcard or to an {open content}
// with a strict or lax {wildcard}, and reports whether one of these holds:
//
//   - its ·expanded name· ·resolves· to a top-level element declaration d
//     (key-governing-ed clauses 3 and 4), and d and the child's subtree meet
//     element's conditions. cvc-complex-type clause 5 is the walk's: it
//     charges a child whose ·governing type definition· is neither the same
//     as nor ·validly substitutable· for its non-·absent· ·locally declared
//     type· within the parent's type (validate's walk.locallyDeclaredType),
//     so no empty Result reaches the gate with such a child;
//   - its name resolves to none and it carries an xsi:type: instanceTyped's
//     conditions, under a strict and a lax wildcard alike, its ·locally
//     declared type· ·absent· or not (key-governing-type-elem clause 8 or 6,
//     the latter read soundly through the walk's clause 5 charge);
//   - strictParticle holds — the child is ·attributed to· a strict ·wildcard
//     particle· — its name resolves to none, and it carries no xsi:type: the
//     child is ·laxly assessed· with a ·governing type definition· of none, so
//     clause 5 is vacuous, and the walk charges e-validity clause 1.1.3
//     (validate's walk.unresolvedStrictWildcardChild) for the parent, so no
//     empty Result reaches the gate with it and the gate reads nothing below
//     it;
//   - strictParticle does not hold — the child is ·attributed to· a lax
//     Wildcard or {open content} — its name resolves to none, it carries no
//     xsi:type, and laxlyAssessed holds for it (#1911).
func (g *subtreeGate) resolvedChild(start xml.StartElement, strictParticle bool) bool {
	d, ok := g.schema.Element(expandedName(start.Name))
	if ok {
		return g.element(start, d)
	}
	if i := slices.IndexFunc(start.Attr, func(a xml.Attr) bool { return a.Name == xsiType }); i >= 0 {
		return g.instanceTyped(start, start.Attr[i].Value)
	}
	if strictParticle {
		return g.dec.Skip() == nil
	}
	return g.laxlyAssessed(start)
}

// instanceTyped reads through to its end tag a wildcard's child whose start tag
// is start, which has no ·governing element declaration· and whose xsi:type
// carries lexical, and reports whether an xsi:nil it carries has an ·actual
// value· (nilValue), lexical names a top-level type definition T against the
// namespace bindings in scope at start (resolveQName), and the child and its
// subtree meet governed's conditions against T, never ·nilled·: key-nilled is
// relative to a declaration, and it has none.
//
// T is the child's ·governing type definition·, so the child is ·strictly
// assessed· against it (cvc-assess-elt clause 1) under a strict and a lax
// wildcard alike, and e-validity reads its own [validity] (key-sva clause
// 1.2): under strict the walk's e-validity clause 1.1.3 charge does not arise
// for it, and under lax it is not ·laxly assessed· (key-lva clause 1). Where
// the child's ·locally declared type· within the parent's type is ·absent·,
// key-governing-type-elem clause 8 selects T, which is validate's
// walk.childGoverning reading (instanceGovernance). A child ·attributed to· a
// Wildcard reaches here with a non-·absent· one too — child asks
// LocallyDeclaredElementType only on its {open content} arm — and clause 6
// selects T only where T ·overrides· that type, which with no declaration is
// key-overrides clause 2's ·validly substitutable without limitation·. That is
// exactly cvc-complex-type clause 5's condition, which the walk charges for a
// T failing it (validate's walk.locallyDeclaredType), so no empty Result
// reaches the gate with such a child unless clause 6 selects T: the gate's
// reading against T is the walk's. With no declaration there is no ·selected
// type definition· for T to ·override·, so governingType's cvc-elt clause 4
// test has no counterpart here; an ·abstract· T is refused by complex
// (cvc-type clause 2).
//
// An xsi:type naming no type definition leaves the child with no ·governing
// type definition·. The walk charges cvc-attribute clause 5 for it, but the
// gate refuses it too rather than read a subtree it cannot type.
func (g *subtreeGate) instanceTyped(start xml.StartElement, lexical string) bool {
	if _, ok := nilValue(start.Attr); !ok {
		return false
	}
	defer g.enter(start)()
	name, ok := g.resolveQName(lexical)
	if !ok {
		return false
	}
	t, ok := g.schema.Type(name)
	if !ok {
		return false
	}
	return g.governed(start, t, false)
}

// laxlyAssessed reads through to its end tag an element whose start tag is
// start, which has neither a ·governing element declaration· nor a ·governing
// type definition· and is not ·skipped·, and reports whether an xsi:nil it
// carries has an ·actual value· (nilValue) and it and its subtree meet
// complex's conditions against ·xs:anyType·, never ·nilled·: key-nilled is
// relative to a declaration, and it has none. Such an element is ·laxly
// assessed· (cvc-assess-elt clause 3, key-lva): locally validated against
// ·xs:anyType· and its [[attributes]] and [[children]] assessed by key-sva
// clauses 2 and 3, which is validate's walk.child and walk.attribute for a
// laxly assessed parent (#1823, #1891). Its [validity] is notKnown (e-validity
// clause 2), which blocks no ancestor's valid: e-validity clause 1.1.3 counts
// notKnown only under a strict ·wildcard particle·. What it does decide is
// every charge at or below it, which instance.go's "Charges at depth" reads as
// "not valid" (§2.5 key-deep-valid-doc, #1911).
func (g *subtreeGate) laxlyAssessed(start xml.StartElement) bool {
	if _, ok := nilValue(start.Attr); !ok {
		return false
	}
	td, ok := g.schema.Type(anyTypeName)
	if !ok {
		return false
	}
	anyType, ok := td.(xsd.ComplexType)
	if !ok {
		return false
	}
	defer g.enter(start)()
	return g.complex(start, anyType, false)
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

// rootStart reads dec up to the document element's start tag. It answers false
// for a DOCTYPE whose DTD could default an attribute the reader does not see
// (defaultsNoAttribute), and for any other directive. That second refusal also
// covers a DOCTYPE encoding/xml delimits short of its real end, as a quote
// inside a processing instruction can make it: every markup declaration left
// over arrives as a directive of its own, and a parameter-entity reference left
// over needs a declaration, which carries a '%' either inside the DOCTYPE or in
// a directive left over too.
//
// The DOCTYPE that survives declares, in its internal subset alone, general
// entities, notations, element types, comments and processing instructions.
// None of these defaults an attribute (XML 1.0 §3.3.2), so the attributes the
// reader sees are the [attributes] the walk assesses. Its entities still
// change the infoset (§4.4), and neither change is read here: an unparsed
// entity it declares is one an ENTITY value may name, which the walk decides
// through parser/xmltree's own read of the subset (key-vde, cvc-simple-type
// clause 3); a reference to a general entity it declares is included text
// (§4.4.2), which parser/xmltree includes and encoding/xml, knowing no entity
// but the five predefined ones, refuses, so a document holding one reaches no
// verdict through either reader of rootStart (rawDecoder): each answers true
// only once dec has read the whole document, and false on dec's error.
func rootStart(dec *xml.Decoder) (xml.StartElement, bool) {
	for {
		tok, err := dec.Token()
		if err != nil {
			return xml.StartElement{}, false
		}
		switch t := tok.(type) {
		case xml.Directive:
			if !defaultsNoAttribute(t) {
				return xml.StartElement{}, false
			}
		case xml.StartElement:
			return t, true
		}
	}
}

// defaultsNoAttribute reports whether d is a DOCTYPE whose DTD can default no
// attribute: an AttlistDecl is the one declaration that does (XML 1.0 §3.3.2),
// and d answers false wherever one could hide. That is an ExternalID after the
// document type name, whose external subset the reader does not read (§2.8
// doctypedecl), an <!ATTLIST in the internal subset, and any '%' there: a
// parameter-entity reference can expand to an AttlistDecl no literal scan sees
// (§4.4.8). A '%' that is no reference — one in a SystemLiteral — is refused
// with them. The header is the text before the first '[', so a '[' inside an
// ExternalID's literal leaves the literal's opening half in the header, where
// it is a third field.
func defaultsNoAttribute(d xml.Directive) bool {
	header, subset, _ := strings.Cut(string(d), "[")
	fields := strings.Fields(header)
	if len(fields) != 2 || fields[0] != "DOCTYPE" {
		return false
	}
	return !strings.Contains(subset, "<!ATTLIST") && !strings.Contains(subset, "%")
}

// rawDecoder is the one encoding/xml reader the lane's raw re-reads —
// assessedSubtreeRoot and instanceHints — take over a document's bytes. It
// reads the leading byte-order mark through internal/xmlenc, the decoding
// parser/xmltree's reader takes (XML 1.0 §4.3.3, Appendix F.1): a UTF-16
// document, either byte order, is transcoded to UTF-8, a UTF-8 mark is dropped
// as the encoding signature it is, and an encoding declaration that disagrees
// with the mark fails the read through the mark's CharsetReader. The mark is
// thereby consumed before xmldecl.As10 meets the declaration, as As10
// requires, and As10 then admits a 1.x version label as xmltree admits it — so
// a document's label is admitted here exactly when xmltree admits it, in
// either encoding, with the mark or without.
//
// One disagreement xmltree rejects is read here: a UTF-16 mark under
// encoding="UTF-8", which encoding/xml never hands to a CharsetReader and
// xmltree's checkDeclaration catches on its own. No verdict rests on that
// read: assessedSubtreeRoot runs only after assessInstance has read the
// instance through xmltree, and a case whose instance instanceHints read is
// declined by that same assessInstance when xmltree rejects it.
//
// A read failure in the peek for the mark is reported by the decoder's first
// read, as xmlenc.Decode latches it.
func rawDecoder(r io.Reader) *xml.Decoder {
	body, mark := xmlenc.Decode(r)
	dec := xml.NewDecoder(xmldecl.As10(body))
	dec.CharsetReader = mark.CharsetReader
	return dec
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
		if documentCarries(d.Location, isVersioningAttr) {
			return true
		}
	}
	return false
}

// isVersioningAttr reports whether a is in versioningNS.
func isVersioningAttr(a xmltree.Attribute) bool {
	return a.Name().Space() == versioningNS
}

// documentCarries reports whether any element of the document at path carries
// an attribute satisfying is, or the document cannot be read to say: one that
// will not open or read answers true.
//
// It reads through parser/xmltree, the reader the assembly read the document
// with, and not through rawDecoder: an element or attribute that a reference
// to an internal general entity includes (XML 1.0 §4.4.2) is one the assembly
// saw, and encoding/xml, which knows no entity but the five predefined ones,
// would refuse the document instead of reading it.
func documentCarries(path string, is func(xmltree.Attribute) bool) bool {
	f, err := os.Open(path)
	if err != nil {
		return true
	}
	defer func() { _ = f.Close() }() // read-only handle: close error cannot affect the verdict
	r := xmltree.NewReader(path, f)
	for {
		tok, err := r.Token()
		if errors.Is(err, io.EOF) {
			return false
		}
		if err != nil {
			return true
		}
		start, ok := tok.(*xmltree.StartElement)
		if !ok {
			continue
		}
		if slices.ContainsFunc(start.Attributes(), is) {
			return true
		}
	}
}
