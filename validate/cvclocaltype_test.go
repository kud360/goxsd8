package validate

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// These fixtures drive cvc-complex-type clause 5 for element [[children]]
// ([walk.locallyDeclaredType]): a child ·attributed to· a wildcard whose
// ·governing type definition· is neither the same as nor ·validly
// substitutable· ·without limitation· for its ·locally declared type·
// (key-ldt-elem) is charged. The schemas are saxonData/Wild's shapes, parsed
// from documents.

// ldtSchema is wild061's shape: zing declares a local e of localType and a
// local f of xs:string, then a wildcard of the given {process contents};
// a top-level e of topType is what the wildcard's child resolves to.
func ldtSchema(t *testing.T, localType, topType, processContents string) *xsd.Schema {
	t.Helper()
	return parsedSchema(t, map[string]string{"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="zing">
    <xs:sequence>
      <xs:element name="e" type="` + localType + `"/>
      <xs:element name="f" type="xs:string"/>
      <xs:any namespace="##local" processContents="` + processContents + `"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="doc" type="zing"/>
  <xs:element name="e" type="` + topType + `"/>
</xs:schema>`})
}

// ldtUnionSchema is wild066's shape: the local e is an anonymous union of
// xs:date and xs:time, and the top-level e is xs:date, one of its members.
func ldtUnionSchema(t *testing.T) *xsd.Schema {
	t.Helper()
	return parsedSchema(t, map[string]string{"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="zing">
    <xs:sequence>
      <xs:element name="e">
        <xs:simpleType><xs:union memberTypes="xs:date xs:time"/></xs:simpleType>
      </xs:element>
      <xs:element name="f" type="xs:integer"/>
      <xs:any namespace="##local" processContents="lax"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="doc" type="zing"/>
  <xs:element name="e" type="xs:date"/>
</xs:schema>`})
}

// ldtDoc is <doc> holding three children, each a text-only element: the first
// named e, the second f, and the wildcard's child named last, which sits at 4:3
// and carries attrs.
func ldtDoc(eText, fText, last, lastText string, attrs ...Attribute) *testElement {
	kid := func(name string, text string, line int, attrs ...Attribute) Child {
		return ElementChild(&testElement{
			name:     local(name),
			attrs:    attrs,
			kids:     []Child{TextChild(&testText{data: text, loc: loc(line, 6)})},
			bindings: map[string]string{"xs": xsd.XMLSchemaNS},
			loc:      loc(line, 3),
		})
	}
	return &testElement{
		name: local("doc"),
		kids: []Child{kid("e", eText, 2), kid("f", fText, 3), kid(last, lastText, 4, attrs...)},
		loc:  loc(1, 1),
	}
}

// xsiTypeAttr is an xsi:type attribute naming lexical, at 4:6.
func xsiTypeAttr(lexical string) Attribute {
	return &testAttribute{name: xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "type"}, value: lexical, loc: loc(4, 6)}
}

// TestLocallyDeclaredTypeChargesAMismatchedWildcardChild pins the charge on
// wild061.n1's shape: the strict wildcard's e resolves to the top-level e of
// xs:time, whose type is not ·validly substitutable· for the local e's
// xs:date. It fails with walk.child's locallyDeclaredType call deleted (no
// violation at all). The message is pinned on the governing type before the
// ·locally declared type·, so swapping the two arguments fails it too.
func TestLocallyDeclaredTypeChargesAMismatchedWildcardChild(t *testing.T) {
	for _, pc := range []string{"strict", "lax"} {
		t.Run(pc, func(t *testing.T) {
			got, unevaluated := assessRecorded(t, ldtSchema(t, "xs:date", "xs:time", pc),
				ldtDoc("2008-11-03", "", "e", "12:20:02"))
			viol := onlyCharge(t, got, ruleCvcComplexType)
			if viol.Loc != loc(4, 3) {
				t.Errorf("Loc = %s, want the wildcard child's %s", viol.Loc, loc(4, 3))
			}
			want := "the element e is a child of doc, and its ·governing type definition· {" + xsd.XMLSchemaNS + "}time is neither"
			if !strings.HasPrefix(viol.Msg, want) {
				t.Errorf("Msg = %q, want the prefix %q", viol.Msg, want)
			}
			if !strings.Contains(viol.Msg, "·locally declared type· {"+xsd.XMLSchemaNS+"}date within") ||
				!strings.Contains(viol.Msg, "cvc-complex-type clause 5") {
				t.Errorf("Msg = %q, want the ·locally declared type· xs:date and clause 5 named", viol.Msg)
			}
			if len(unevaluated) != 0 {
				t.Errorf("Unevaluated() = %v, want none", unevaluated)
			}
		})
	}
}

// TestLocallyDeclaredTypeChargesAnXSITypeWildcardChild pins the charge on
// wild062.n3's shape: the lax wildcard's f resolves to no top-level
// declaration, so its xsi:type alone governs it (key-governing-type-elem
// clause 8), and xs:time is not ·validly substitutable· for the local f's
// xs:string.
func TestLocallyDeclaredTypeChargesAnXSITypeWildcardChild(t *testing.T) {
	got, _ := assessRecorded(t, ldtSchema(t, "xs:date", "xs:time", "lax"),
		ldtDoc("2008-11-03", "", "f", "12:20:02", xsiTypeAttr("xs:time")))
	viol := onlyCharge(t, got, ruleCvcComplexType)
	if viol.Loc != loc(4, 3) {
		t.Errorf("Loc = %s, want the wildcard child's %s", viol.Loc, loc(4, 3))
	}
	if !strings.Contains(viol.Msg, "{"+xsd.XMLSchemaNS+"}time is neither") ||
		!strings.Contains(viol.Msg, "·locally declared type· {"+xsd.XMLSchemaNS+"}string within") {
		t.Errorf("Msg = %q, want xs:time against the ·locally declared type· xs:string", viol.Msg)
	}
}

// TestLocallyDeclaredTypeAdmits pins clause 5's satisfied and vacuous arms:
// the same type, a type ·validly substitutable· for the ·locally declared
// type· by restriction and as a union member, an anonymous complex type
// against itself (sameType's {context} arm: without it the first child is
// charged), and a child whose name the parent's type declares nothing for,
// whose ·locally declared type· is ·absent·.
func TestLocallyDeclaredTypeAdmits(t *testing.T) {
	for _, tc := range []struct {
		why    string
		schema *xsd.Schema
		doc    *testElement
	}{
		{"the same type (wild061's shape, both xs:date)",
			ldtSchema(t, "xs:date", "xs:date", "strict"), ldtDoc("2008-11-03", "", "e", "2008-11-04")},
		{"xs:positiveInteger is validly substitutable for xs:integer (wild063.v1)",
			ldtSchema(t, "xs:integer", "xs:positiveInteger", "strict"), ldtDoc("-1", "", "e", "12")},
		{"xs:date is validly substitutable for a union of xs:date and xs:time (wild066.v1)",
			ldtUnionSchema(t), ldtDoc("12:12:00", "42", "e", "2008-11-02")},
		{"an xsi:type that is the ·locally declared type· itself",
			ldtSchema(t, "xs:date", "xs:time", "lax"), ldtDoc("2008-11-03", "", "f", "x", xsiTypeAttr("xs:string"))},
		{"no declaration for the name in the parent's type: the ·locally declared type· is ·absent·",
			parsedSchema(t, map[string]string{"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="zing">
    <xs:sequence>
      <xs:element name="e" type="xs:date"/>
      <xs:element name="f" type="xs:string"/>
      <xs:any namespace="##local" processContents="strict"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="doc" type="zing"/>
  <xs:element name="g" type="xs:time"/>
</xs:schema>`}), ldtDoc("2008-11-03", "", "g", "12:20:02")},
		{"a child governed by the local declaration whose type is an anonymous complex type",
			parsedSchema(t, map[string]string{"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="zing">
    <xs:sequence>
      <xs:element name="e"><xs:complexType mixed="true"/></xs:element>
      <xs:element name="f" type="xs:string"/>
      <xs:any namespace="##local" processContents="lax"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="doc" type="zing"/>
</xs:schema>`}), ldtDoc("x", "", "g", "")},
	} {
		t.Run(tc.why, func(t *testing.T) {
			got, unevaluated := assessRecorded(t, tc.schema, tc.doc)
			if len(got) != 0 {
				t.Errorf("Violations() = %v, want none", got)
			}
			if len(unevaluated) != 0 {
				t.Errorf("Unevaluated() = %v, want none", unevaluated)
			}
		})
	}
}
