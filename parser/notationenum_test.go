package parser_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// undeclaredNotation is the verdict opening, for member resolving to the QName
// name, that it names no notation declaration.
func undeclaredNotation(member, name string) string {
	return `the NOTATION value "` + member + `" resolves to the QName ` + name + `, which names no notation declaration`
}

// wantUndeclaredNotationMember fails unless err is enumeration valid
// restriction (Datatypes §4.3.5.5) charged at the restricting <simpleType> at
// line:col of main.xsd, wrapping a cvc-datatype-valid verdict that opens with
// prefix and carries undeclaredNotation(member, name).
func wantUndeclaredNotationMember(t *testing.T, err error, line, col int, prefix, member, name string) {
	t.Helper()
	var xe *xsderr.Error
	if !errors.As(err, &xe) {
		t.Fatalf("err = %v, want an *xsderr.Error", err)
	}
	if xe.Rule != "enumeration-valid-restriction" {
		t.Fatalf("Rule = %q, want enumeration-valid-restriction: %v", xe.Rule, err)
	}
	if want := (xsderr.Loc{URI: "main.xsd", Line: line, Col: col}); xe.Loc != want {
		t.Errorf("Loc = %s, want the restricting simpleType's %s", xe.Loc, want)
	}
	var cause *xsderr.Error
	if !errors.As(xe.Err, &cause) {
		t.Fatalf("Err = %v, want the wrapped Datatype Valid verdict", xe.Err)
	}
	if cause.Rule != "cvc-datatype-valid" {
		t.Fatalf("wrapped Rule = %q, want cvc-datatype-valid: %v", cause.Rule, err)
	}
	verdict := undeclaredNotation(member, name)
	if !strings.HasPrefix(cause.Msg, prefix) || !strings.Contains(cause.Msg, verdict) {
		t.Fatalf("wrapped Msg = %q, want it to open %q and carry %q", cause.Msg, prefix, verdict)
	}
}

// An enumeration member of a NOTATION-derived type is in the ·value space· of
// its {base type definition} only where its QName names a notation declared in
// the current schema (Datatypes §3.3.19), so a member naming none breaks
// enumeration valid restriction (§4.3.5.5). The base may be xs:NOTATION's
// restriction with or without a facet of its own, a list of xs:NOTATION, or a
// union with an xs:NOTATION member, whose verdict names the NOTATION member's
// among the rest; bez is the undeclared member in each, and bar, declared after
// the types, is not the declaration it needs.
func TestAnUndeclaredNotationMemberIsRejected(t *testing.T) {
	atomic := undeclaredNotation("bez", "bez")
	cases := []struct {
		name, base, prefix string
	}{
		{"atomic", `<xs:restriction base="xs:NOTATION"/>`, atomic},
		{"derived", `<xs:restriction base="xs:NOTATION"><xs:pattern value="\i\c*"/></xs:restriction>`, atomic},
		{"list", `<xs:list itemType="xs:NOTATION"/>`, atomic},
		{"union", `<xs:union memberTypes="xs:NOTATION xs:integer"/>`, `value "bez" is Datatype Valid with respect to no member of the union`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := parseMap(t, "main.xsd", map[string]string{"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="foo" public="pubfoo"/>
  <xs:simpleType name="Base">` + c.base + `</xs:simpleType>
  <xs:simpleType name="Enum">
    <xs:restriction base="Base">
      <xs:enumeration value="foo"/>
      <xs:enumeration value="bez"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:notation name="bar" public="pubbar"/>
</xs:schema>`})
			wantUndeclaredNotationMember(t, err, 4, 3, c.prefix, "bez", "bez")
		})
	}
}

// A member's QName resolves against the namespace bindings in scope at its
// <enumeration> (§3.3.18), the default namespace included, and is compared by
// expanded name: tns:png, png under the default namespace urn:n and n2:png with
// n2 rebound to urn:n all name {urn:n}png, declared AFTER the type, while
// p:png names {urn:p}png, which nothing declares.
func TestANotationMemberResolvesAtItsEnumeration(t *testing.T) {
	doc := func(member string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:n" xmlns:tns="urn:n" xmlns:p="urn:p">
  <xs:simpleType name="Enum">
    <xs:restriction base="xs:NOTATION">
      ` + member + `
    </xs:restriction>
  </xs:simpleType>
  <xs:notation name="png" public="pubpng"/>
</xs:schema>`
	}
	for _, member := range []string{
		`<xs:enumeration value="tns:png"/>`,
		`<xs:enumeration value="png" xmlns="urn:n"/>`,
		`<xs:enumeration value=" n2:png " xmlns:n2="urn:n"/>`,
	} {
		if _, err := parseMap(t, "main.xsd", map[string]string{"main.xsd": doc(member)}); err != nil {
			t.Errorf("%s: parse = %v, want the declared {urn:n}png admitted", member, err)
		}
	}
	_, err := parseMap(t, "main.xsd", map[string]string{"main.xsd": doc(`<xs:enumeration value="p:png"/>`)})
	wantUndeclaredNotationMember(t, err, 2, 3, undeclaredNotation("p:png", "{urn:p}png"), "p:png", "{urn:p}png")
	_, err = parseMap(t, "main.xsd", map[string]string{"main.xsd": doc(`<xs:enumeration value="png"/>`)})
	wantUndeclaredNotationMember(t, err, 2, 3, undeclaredNotation("png", "png"), "png", "png")
}

// The declared set is the assembled schema's {notation declarations}
// (sch-props-correct, Structures §3.17.6.1): a notation declared only in an
// imported document, or only in an included one, is declared for a member
// written in the including document.
func TestANotationDeclaredInAnotherDocumentIsDeclared(t *testing.T) {
	docs := map[string]string{
		"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:n" xmlns:tns="urn:n" xmlns:o="urn:o">
  <xs:import namespace="urn:o" schemaLocation="other.xsd"/>
  <xs:include schemaLocation="part.xsd"/>
  <xs:simpleType name="Enum">
    <xs:restriction base="xs:NOTATION">
      <xs:enumeration value="o:png"/>
      <xs:enumeration value="tns:gif"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`,
		"other.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:o">
  <xs:notation name="png" public="pubpng"/>
</xs:schema>`,
		"part.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:n">
  <xs:notation name="gif" public="pubgif"/>
</xs:schema>`,
	}
	if _, err := parseMap(t, "main.xsd", docs); err != nil {
		t.Fatalf("parse = %v, want o:png and tns:gif admitted as declared elsewhere in the assembly", err)
	}
}

// An ·override· is judged on the overridden document D_old' (src-override,
// Structures §4.2.5), and the host's own top-level notations are in the
// assembled set: Nota, restated inside the <override>, enumerates bez, which
// only the host declares, outside the <override>.
func TestAnOverrideHostsNotationIsAnEnumerableMember(t *testing.T) {
	docs := map[string]string{
		"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:override schemaLocation="base.xsd">
    <xs:notation name="foo" public="pubfoo2"/>
    <xs:simpleType name="Nota">
      <xs:restriction base="xs:NOTATION">
        <xs:enumeration value="foo"/>
        <xs:enumeration value="bez"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:override>
  <xs:notation name="bez" public="pubbez"/>
</xs:schema>`,
		"base.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="Nota">
    <xs:restriction base="xs:NOTATION">
      <xs:enumeration value="foo"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:notation name="foo" public="pubfoo"/>
</xs:schema>`,
	}
	if _, err := parseMap(t, "main.xsd", docs); err != nil {
		t.Fatalf("parse = %v, want bez admitted as the host's declaration", err)
	}
}
