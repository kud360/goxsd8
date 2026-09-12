package parser_test

import (
	"testing"

	"github.com/kud360/goxsd8/parser/xmltree"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// importsXML is a schema document whose <schema> carries the bare <import> that
// licenses the XML namespace — src-resolve (§3.17.6.2) clause 4.2.2, the only
// clause that licenses it, since clauses 4.2.3 and 4.2.4 hardcode the XSD and
// XSI namespaces alone — wrapped around body.
func importsXML(body string) string {
	return `<xs:schema xmlns:xs="` + xsdNS + `" xmlns:xml="` + xmltree.XMLNamespaceURI + `">` +
		`<xs:import namespace="` + xmltree.XMLNamespaceURI + `"/>` + body + `</xs:schema>`
}

// TestProduceSuppliesXMLNamespace pins the four Attribute Declarations the
// schema document for the XML namespace declares (§1.3.2's first ·namespace
// with special status·, xmlschema11-1.md:217), supplied for an <import> naming
// that namespace under the §4.2.6.1 license to "construct components using
// means of its own choosing" (xmlschema11-1.md:4199).
//
// The document declares NOTHING but the import, so every declaration asserted
// here is supplied rather than mapped.
//
// THE REVISION ASSERTED IS THE 2009 ONE, and two rows are where that shows:
// xml:lang is the UNION of xs:language with an anonymous empty-string type that
// Datatypes §3.4.3's note quotes off http://www.w3.org/2001/xml.xsd
// (xmlschema11-2.md:1742), not the plain xs:language of the 2001 original; and
// xml:space carries NO {value constraint}, where the 2001 original wrote
// default="preserve" on it. A supply of the older revision satisfies neither.
func TestProduceSuppliesXMLNamespace(t *testing.T) {
	s, err := produce(t, importsXML(""))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}

	// A slice, not a map: subtest order is output (STYLE D2). wantType is the
	// by-name {type definition} each row expects, and is the zero QName for the
	// two rows whose slot is an inline anonymous type instead.
	cases := []struct {
		local    string
		wantType xsd.QName
	}{
		{local: "lang"},
		{local: "space"},
		{local: "base", wantType: xsd.QName{Space: xsdNS, Local: "anyURI"}},
		{local: "id", wantType: xsd.QName{Space: xsdNS, Local: "ID"}},
	}
	for _, tc := range cases {
		t.Run(tc.local, func(t *testing.T) {
			name := xsd.QName{Space: xmltree.XMLNamespaceURI, Local: tc.local}
			d, found := s.Attribute(name)
			if !found {
				t.Fatalf("Schema.Attribute(%s) reports no declaration, want the supplied built-in one", name)
			}
			if got := d.Name(); got != name {
				t.Fatalf("{name} = %s, want %s", got, name)
			}
			if got := d.ScopeVariety(); got != xsd.ScopeGlobal {
				t.Fatalf("{scope}.{variety} = %v, want global", got)
			}
			if vc, has := d.ValueConstraint(); has {
				t.Fatalf("{value constraint} = %v, want ·absent· (the 2009 revision writes no default)", vc)
			}
			if got := d.Loc(); got != (xsderr.Loc{}) {
				t.Fatalf("Loc() = %v, want the zero Loc: no schema document declares this component", got)
			}
			if tc.wantType != (xsd.QName{}) {
				if got := declaredTypeName(t, d.TypeDefinition()); got != tc.wantType {
					t.Fatalf("{type definition} = %s, want %s", got, tc.wantType)
				}
				return
			}
			// The inline arm is itself an assertion about the revision: the 2001
			// original's xml:lang is type="xs:language", a by-name reference, which
			// inlineSimpleType refuses.
			st := inlineSimpleType(t, d.TypeDefinition())
			switch tc.local {
			case "lang":
				assertLangType(t, s, st)
			case "space":
				assertSpaceType(t, s, st)
			}
		})
	}
}

// hasAttributeGroup reports whether the schema's {attribute group definitions}
// hold one with this expanded name. The property has no by-name query on the
// published surface, so the document-order slice is scanned (STYLE D2).
func hasAttributeGroup(s *xsd.Schema, name xsd.QName) bool {
	for _, g := range s.AttributeGroups() {
		if g.Name() == name {
			return true
		}
	}
	return false
}

// assertLangType pins xml:lang's {member type definitions}: xs:language first,
// then the anonymous type whose only value is the empty string — the pair
// xmlschema11-2.md:1742 quotes, in that order (§3.16.2.4 map.std.union case 1
// fixes it).
func assertLangType(t *testing.T, s *xsd.Schema, st *xsd.SimpleType) {
	t.Helper()
	members, err := st.Members(s)
	if err != nil {
		t.Fatalf("Members: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("{member type definitions} = %d members, want 2", len(members))
	}
	if got, want := members[0].Name(), (xsd.QName{Space: xsdNS, Local: "language"}); got != want {
		t.Fatalf("{member type definitions}[0] = %s, want %s", got, want)
	}
	if got := members[1].Name(); got != (xsd.QName{}) {
		t.Fatalf("{member type definitions}[1] {name} = %s, want ·absent· (an anonymous type)", got)
	}
	base, err := members[1].Base(s)
	if err != nil {
		t.Fatalf("Base: %v", err)
	}
	if got, want := base.Name(), (xsd.QName{Space: xsdNS, Local: "string"}); got != want {
		t.Fatalf("{member type definitions}[1] {base type definition} = %s, want %s", got, want)
	}
	assertEnumeration(t, members[1], []string{""})
}

// assertSpaceType pins xml:space's {type definition}: an anonymous restriction
// of xs:NCName enumerating the two values XML §2.10 gives the attribute.
func assertSpaceType(t *testing.T, s *xsd.Schema, st *xsd.SimpleType) {
	t.Helper()
	base, err := st.Base(s)
	if err != nil {
		t.Fatalf("Base: %v", err)
	}
	if got, want := base.Name(), (xsd.QName{Space: xsdNS, Local: "NCName"}); got != want {
		t.Fatalf("{base type definition} = %s, want %s", got, want)
	}
	assertEnumeration(t, st, []string{"default", "preserve"})
}

// assertEnumeration pins a type's OWN enumeration facet {value}, in the order
// declared: the lexical members and nothing else among its own facets.
func assertEnumeration(t *testing.T, st *xsd.SimpleType, want []string) {
	t.Helper()
	facets := st.OwnFacets()
	if len(facets) != 1 {
		t.Fatalf("{facets} = %d own facets, want 1 (the enumeration)", len(facets))
	}
	if got := facets[0].Kind(); got != xsd.FacetEnumeration {
		t.Fatalf("{facets}[0] kind = %v, want enumeration", got)
	}
	got := facets[0].Values()
	if len(got) != len(want) {
		t.Fatalf("enumeration {value} = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("enumeration {value} = %q, want %q", got, want)
		}
	}
}

// TestProduceXMLNamespaceRefResolves pins the defect the supply closes: an
// <xs:attribute ref="xml:base"/> in a document that imports the namespace
// resolves at finalize instead of being charged src-resolve clause 1.2, which
// is where the addC001 and isDefault078 schema cases sat. The attribute use is
// asserted, not merely the absence of an error: a reference that resolved to
// nothing would still have to fail here.
func TestProduceXMLNamespaceRefResolves(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, local := range []string{"lang", "space", "base", "id"} {
		t.Run(local, func(t *testing.T) {
			s, err := produce(t, importsXML(
				`<xs:complexType name="ct"><xs:attribute ref="xml:`+local+`"/></xs:complexType>`))
			if err != nil {
				t.Fatalf("Produce: %v, want the supplied built-in declaration to resolve the ref (src-resolve clause 1.2)", err)
			}
			ct, found := s.Type(xsd.QName{Local: "ct"})
			if !found {
				t.Fatalf("Schema.Type reports no ct")
			}
			complexType, ok := ct.(xsd.ComplexType)
			if !ok {
				t.Fatalf("ct = %T, want an xsd.ComplexType", ct)
			}
			uses := complexType.AttributeUses()
			if len(uses) != 1 {
				t.Fatalf("{attribute uses} = %d, want 1", len(uses))
			}
			want := xsd.QName{Space: xmltree.XMLNamespaceURI, Local: local}
			if got := uses[0].DeclarationName(); got != want {
				t.Fatalf("{attribute uses}[0] declaration name = %s, want %s", got, want)
			}
			if _, ok := s.ResolvedAttributeDeclaration(uses[0]); !ok {
				t.Fatalf("{attribute uses}[0] resolves to no declaration, want the supplied built-in %s", want)
			}
		})
	}
}

// TestProduceXMLNamespaceGroupWithheld pins addXMLNamespace's GAP(parser): the
// xml:specialAttrs attribute group definition is NOT supplied, so an
// <attributeGroup ref="xml:specialAttrs"/> is still charged src-resolve clause
// 1.4 and the property holds no such definition. The gap is pinned rather than
// left to be noticed, so the landing that closes it has to come through here.
// Owned by #1458.
func TestProduceXMLNamespaceGroupWithheld(t *testing.T) {
	_, err := produce(t, importsXML(
		`<xs:complexType name="ct"><xs:attributeGroup ref="xml:specialAttrs"/></xs:complexType>`))
	if err == nil {
		t.Fatalf("Produce accepted an xml:specialAttrs reference, which nothing supplies (src-resolve clause 1.4)")
	}
	s, err := produce(t, importsXML(""))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	if hasAttributeGroup(s, xsd.QName{Space: xmltree.XMLNamespaceURI, Local: "specialAttrs"}) {
		t.Errorf("{attribute group definitions} holds xml:specialAttrs, which addXMLNamespace does not supply")
	}
}

// TestProduceWithoutImportWithholdsXMLNamespace pins the IMPORT half of the
// supply gate: the four are NOT "present in every schema by definition" the way
// §3.2.7's xsi: four are, so a document that never asks for the namespace is
// handed nothing, and a reference in it is charged src-resolve clause 4.2 —
// which hardcodes the XSD and XSI namespaces (clauses 4.2.3, 4.2.4) and not
// this one.
func TestProduceWithoutImportWithholdsXMLNamespace(t *testing.T) {
	doc := `<xs:schema xmlns:xs="` + xsdNS + `" xmlns:xml="` + xmltree.XMLNamespaceURI + `">` +
		`<xs:complexType name="ct"><xs:attribute ref="xml:base"/></xs:complexType></xs:schema>`
	if _, err := produce(t, doc); err == nil {
		t.Fatalf("Produce accepted an xml:base reference from a document that imports nothing, want src-resolve clause 4.2")
	}
	s, err := produce(t, `<xs:schema xmlns:xs="`+xsdNS+`"/>`)
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	for _, local := range []string{"lang", "space", "base", "id"} {
		name := xsd.QName{Space: xmltree.XMLNamespaceURI, Local: local}
		if _, found := s.Attribute(name); found {
			t.Errorf("Schema.Attribute(%s) reports a declaration for a schema that imports nothing", name)
		}
	}
	if hasAttributeGroup(s, xsd.QName{Space: xmltree.XMLNamespaceURI, Local: "specialAttrs"}) {
		t.Errorf("{attribute group definitions} holds xml:specialAttrs for a schema that imports nothing")
	}
}

// TestParseComposedXMLNamespaceWinsOverBuiltin pins the COMPOSED half of the
// gate, the half that keeps the built-in a SUBSTITUTE: when the <import>'s
// schemaLocation DOES resolve, the composed document's own declarations are the
// schema's and nothing is supplied beside them. Supplying both is two
// components of one kind sharing an expanded name, charged sch-props-correct
// (§3.17.6.1) clause 2 at finalize — the shape the W3C suite's
// saxonData/Override/over030 (banked pass) and msData/additional/test264908_1
// carry.
//
// The composed document gives xml:lang a by-NAME xs:token, which the built-in
// never does (its slot is an anonymous union), so the assertion distinguishes
// "the composed declaration survived" from "either declaration is present".
func TestParseComposedXMLNamespaceWinsOverBuiltin(t *testing.T) {
	docs := map[string]string{
		"main.xsd": `<xs:schema xmlns:xs="` + xsdNS + `" xmlns:xml="` + xmltree.XMLNamespaceURI + `">` +
			`<xs:import namespace="` + xmltree.XMLNamespaceURI + `" schemaLocation="xml.xsd"/>` +
			`<xs:complexType name="ct"><xs:attribute ref="xml:lang"/></xs:complexType></xs:schema>`,
		"xml.xsd": `<xs:schema xmlns:xs="` + xsdNS + `" targetNamespace="` + xmltree.XMLNamespaceURI + `">` +
			`<xs:attribute name="lang" type="xs:token"/>` +
			`<xs:attribute name="space" type="xs:token"/>` +
			`<xs:attribute name="base" type="xs:token"/>` +
			`<xs:attribute name="id" type="xs:ID"/>` +
			`</xs:schema>`,
	}
	s, err := parseMap(t, "main.xsd", docs)
	if err != nil {
		t.Fatalf("Parse: %v, want the composed document's own declarations and no built-in beside them (sch-props-correct clause 2)", err)
	}
	name := xsd.QName{Space: xmltree.XMLNamespaceURI, Local: "lang"}
	d, found := s.Attribute(name)
	if !found {
		t.Fatalf("Schema.Attribute(%s) reports no declaration", name)
	}
	if got, want := declaredTypeName(t, d.TypeDefinition()), (xsd.QName{Space: xsdNS, Local: "token"}); got != want {
		t.Fatalf("{type definition} = %s, want the composed document's %s", got, want)
	}
	if got := d.Loc(); got == (xsderr.Loc{}) {
		t.Fatalf("Loc() is the zero Loc, want xml.xsd's own position: this declaration was composed, not supplied")
	}
	if hasAttributeGroup(s, xsd.QName{Space: xmltree.XMLNamespaceURI, Local: "specialAttrs"}) {
		t.Errorf("{attribute group definitions} holds xml:specialAttrs, which no composed document here declares")
	}
}

// TestParseUnresolvedXMLNamespaceLocationSupplies pins the other side of that
// coin: a schemaLocation the resolver cannot read leaves nothing composed under
// the namespace, and §4.2.6.1's license is explicitly indifferent to the hint —
// "whether or not a schemaLocation hint is provided" (xmlschema11-1.md:4199) —
// so the built-in is supplied exactly as for a bare <import>. src-import makes
// the failed fetch no error of its own ("It is not an error for the application
// schema component reference strategy to fail").
func TestParseUnresolvedXMLNamespaceLocationSupplies(t *testing.T) {
	docs := map[string]string{
		"main.xsd": `<xs:schema xmlns:xs="` + xsdNS + `" xmlns:xml="` + xmltree.XMLNamespaceURI + `">` +
			`<xs:import namespace="` + xmltree.XMLNamespaceURI + `" schemaLocation="http://www.w3.org/2001/xml.xsd"/>` +
			`<xs:complexType name="ct"><xs:attribute ref="xml:base"/></xs:complexType></xs:schema>`,
	}
	s, err := parseMap(t, "main.xsd", docs)
	if err != nil {
		t.Fatalf("Parse: %v, want the built-in supplied for an unreadable schemaLocation", err)
	}
	name := xsd.QName{Space: xmltree.XMLNamespaceURI, Local: "base"}
	d, found := s.Attribute(name)
	if !found {
		t.Fatalf("Schema.Attribute(%s) reports no declaration", name)
	}
	if got, want := declaredTypeName(t, d.TypeDefinition()), (xsd.QName{Space: xsdNS, Local: "anyURI"}); got != want {
		t.Fatalf("{type definition} = %s, want %s", got, want)
	}
}
