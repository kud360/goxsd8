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

// This file holds two of the instance lane's three shape gates for the "valid"
// observation it makes, one per shape: a SIMPLE LEAF ROOT (#1738) and a COMPLEX
// EMPTY LEAF ROOT (#1808) — each a validation root whose ·governing element
// declaration· and instance shape leave no clause of cvc-elt (§3.3.4.3) and
// key-sva (§3.3.4.6) to decide beyond what an empty validate.Result has already
// shown decided — and the helpers the third, subtreeroot.go's assessed subtree
// root (#1841), shares with them. instance.go's "Why an EMPTY Result is evidence
// of validity for THREE shapes only" states which clause each condition below
// discharges; this file is only the conditions.
//
// Each gate is computed per case and stored nowhere. It reads the schema and
// the documents independently of the walk, so the walk's own charges on the
// attribute and children conditions — cvc-type clauses 3.1.1/3.1.2 for the
// simple leaf root, cvc-complex-type clauses 1.1 and 2 for the complex empty
// one, cvc-elt clause 3.1 for both — are a SECOND check, never the only one.

// versioningNS is the XML Schema versioning namespace §4.2.2 reads vc:minVersion,
// vc:maxVersion, vc:typeAvailable, vc:typeUnavailable, vc:facetAvailable and
// vc:facetUnavailable from. parser keeps its own copy unexported; the lane
// declares the name here rather than parser exporting it for a gate that exists
// only because of a parser GAP (#1002).
const versioningNS = "http://www.w3.org/2007/XMLSchema-versioning"

// idOrEntityOrNotation are the builtin simple types the two LEAF-root gates
// (simpleLeafRoot, complexEmptyLeafRoot) exclude from the root type's closure.
// Those gates discharge cvc-elt clause 7 (cvc-id, §3.3.4.5) by an EMPTY
// [ID/IDREF table], which holds only with ID, IDREF and IDREFS absent; they keep
// String Valid clause 3's ·declared entity name· check (key-vde) out of play by
// excluding ENTITY and ENTITIES; and they exclude NOTATION on walkUnrecorded's
// terms. Each is excluded by NAME, which covers every type that restricts,
// lists or unions over one: closureReaches visits the {base type definition}
// chain, the {item type definition} and every {member type definitions} entry.
var idOrEntityOrNotation = []string{"ID", "IDREF", "IDREFS", "ENTITY", "ENTITIES", "NOTATION"}

// walkUnrecorded are the builtin simple types whose presence anywhere in a
// type's closure puts in play a clause the walk neither decides nor records a
// decline for, at any depth. The assessed-subtree-root gate excludes these
// alone: the ID family and the ENTITY family are the walk's to decide and
// record at every depth (instance.go, the third shape's cvc-elt clause 7
// bullet).
//
// GAP(validate): an xs:NOTATION value is never checked against the schema's
// notations. NOTATION's ·value space· is "the set of QNames of notations
// declared in the current schema" (Datatypes §3.3.19, with
// enumeration-required-notation), which no validate or backend site checks and
// records no decline for — Override/over027/instance/over027.n01.xml, whose
// NOTATION value names no declared notation, is suite-invalid and walks clean,
// a false accept (#1901).
var walkUnrecorded = []string{"NOTATION"}

// simpleLeafRoot reports whether the instance document at doc, against schema
// as assembled into report, has the simple-leaf-root shape an empty
// validate.Result may be read as "valid" for. Any failure to establish a
// condition — an unreadable document, a peek error, an unresolvable type — is a
// false, never a guess: the gate only ever narrows what the lane decides.
func simpleLeafRoot(schema *xsd.Schema, report *parser.AssemblyReport, doc string) bool {
	root, ok := peekRoot(doc, leafContent)
	if !ok {
		return false
	}
	d, ok := schema.Element(root)
	if !ok {
		return false
	}
	if !simpleLeafDeclaration(schema, d) {
		return false
	}
	return !closureVersioned(report)
}

// complexEmptyLeafRoot reports whether the instance document at doc, against
// schema as assembled into report, has the complex-empty-leaf-root shape an
// empty validate.Result may be read as "valid" for. Like simpleLeafRoot, any
// failure to establish a condition is a false, never a guess.
func complexEmptyLeafRoot(schema *xsd.Schema, report *parser.AssemblyReport, doc string) bool {
	root, ok := peekRoot(doc, emptyContent)
	if !ok {
		return false
	}
	d, ok := schema.Element(root)
	if !ok {
		return false
	}
	if !complexEmptyDeclaration(schema, d) {
		return false
	}
	return !closureVersioned(report)
}

// peekRoot reads the instance document at doc with encoding/xml, independently
// of the walk, and returns its root's expanded name where content — which reads
// the root's content up to its end tag, leafContent or emptyContent — admits
// that content, and where the root meets both gates' common conditions:
//
//   - no DOCTYPE at all, since a DTD can default an attribute onto the root that
//     this peek would not see;
//   - no [[attributes]] other than xsi:schemaLocation and
//     xsi:noNamespaceSchemaLocation. Namespace declarations are skipped: the
//     infoset carries them in [namespace attributes], not [attributes].
//     xsi:type and xsi:nil are refused along with everything else.
//
// A decoder error of any kind — including an encoding encoding/xml does not
// read — answers false.
func peekRoot(doc string, content func(*xml.Decoder) bool) (xsd.QName, bool) {
	rc, _, err := loader.Dir(filepath.Dir(doc)).Resolve("", filepath.Base(doc))
	if err != nil {
		return xsd.QName{}, false
	}
	defer func() { _ = rc.Close() }() // read-only handle: close error cannot affect the verdict
	dec := xml.NewDecoder(rc)
	root, ok := rootStart(dec)
	if !ok || !leafAttributes(root.Attr) || !content(dec) {
		return xsd.QName{}, false
	}
	return expandedName(root.Name), true
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

// leafAttributes reports whether attrs, a root start tag's attribute list, holds
// nothing beyond namespace declarations and the two xsi: location hints.
func leafAttributes(attrs []xml.Attr) bool {
	for _, a := range attrs {
		if !isNamespaceDeclaration(a) && !isLocationHint(a) {
			return false
		}
	}
	return true
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

// leafContent reads the root's content up to its end tag and reports whether no
// element started inside it. Character data, comments and processing
// instructions are the only content it admits.
func leafContent(dec *xml.Decoder) bool {
	for {
		tok, err := dec.Token()
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

// emptyContent reads the root's content up to its end tag and reports whether it
// holds no element and no character information item [[children]] at all,
// white space included: comments and processing instructions are the only
// content it admits. It is leafContent less character data, which
// cvc-complex-type clause 1.1 admits none of under an empty {content type}.
func emptyContent(dec *xml.Decoder) bool {
	for {
		tok, err := dec.Token()
		if err != nil {
			return false
		}
		switch tok.(type) {
		case xml.StartElement, xml.CharData:
			return false
		case xml.EndElement:
			return true
		}
	}
}

// simpleLeafDeclaration reports whether d, the top-level declaration the root
// resolves to ([xsd.Schema.Element] indexes top-level declarations alone), is
// one whose every cvc-elt clause a complete walk has decided: not abstract,
// not nillable, no fixed {value constraint} (a default one is decided by the
// walk's cvc-elt clause 5.1 arm), no {identity-constraint definitions}, no
// {type table}, and a {type definition} that resolves to a Simple Type
// Definition whose closure reaches none of idOrEntityOrNotation.
func simpleLeafDeclaration(schema *xsd.Schema, d xsd.ElementDeclaration) bool {
	if d.Abstract() || d.Nillable() {
		return false
	}
	if vc, ok := d.ValueConstraint(); ok && vc.Kind() == xsd.ValueFixed {
		return false
	}
	if len(d.IdentityConstraints()) > 0 {
		return false
	}
	if _, ok := d.TypeTable(); ok {
		return false
	}
	st, ok := schema.ResolvedSimpleType(d.TypeDefinition())
	if !ok {
		return false
	}
	return !closureReaches(schema, st, idOrEntityOrNotation)
}

// complexEmptyDeclaration reports whether d, the top-level declaration the root
// resolves to, is one whose every cvc-elt clause is either vacuous for a root
// with no attributes and no content or decided by the walk: not abstract, not
// nillable, no {value constraint} of either kind, no {identity-constraint
// definitions}, no {type table}, and a {type definition} that resolves to a
// Complex Type Definition that is not abstract, whose {content type}.{variety}
// is empty, and whose {attribute uses}, {attribute wildcard} and {assertions}
// are all empty or absent.
//
// Unlike simpleLeafDeclaration it refuses a default {value constraint} too:
// with none at all, cvc-elt clause 5.2 applies and clause 5.1's substitution of
// a {lexical form} into an empty {content type} never arises. An assembled
// schema carries no such declaration anyway (e-props-correct clause 2 through
// cos-valid-default clause 2.1, which admits a default on mixed content only);
// the gate refuses it on its own rather than resting on that check. The type's
// properties are read off the component as Finalize leaves them, already folded
// over its {base type definition} chain (cos-ct-extends, §3.4.2.4, §3.4.2.5), so
// no base walk is needed here the way closureReaches needs one.
func complexEmptyDeclaration(schema *xsd.Schema, d xsd.ElementDeclaration) bool {
	if d.Abstract() || d.Nillable() {
		return false
	}
	if _, ok := d.ValueConstraint(); ok {
		return false
	}
	if len(d.IdentityConstraints()) > 0 {
		return false
	}
	if _, ok := d.TypeTable(); ok {
		return false
	}
	td, ok := schema.ResolvedType(d.TypeDefinition())
	if !ok {
		return false
	}
	ct, ok := td.(xsd.ComplexType)
	if !ok {
		return false
	}
	if ct.Abstract() {
		return false
	}
	if _, ok := ct.ContentType().(xsd.EmptyContent); !ok {
		return false
	}
	if len(ct.AttributeUses()) > 0 {
		return false
	}
	if _, ok := ct.AttributeWildcard(); ok {
		return false
	}
	return len(ct.Assertions()) == 0
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
