package validate

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These fixtures drive cvc-elt (§3.3.4.3) clause 2 ([walk.abstractDeclaration]):
// an element whose ·governing element declaration· has {abstract} true is
// charged once, at the element, at whatever depth and by whatever route
// key-governing-ed settled that declaration.

// abstractDeclSchema declares the abstract head of a substitution group with
// the concrete member and the abstract absMember, an abstract absRoot over head
// particles, holder over head particles, and strictHolder, laxHolder and
// skipHolder over a wildcard of each {process contents}.
const abstractDeclSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" abstract="true" type="xs:string"/>
  <xs:element name="member" substitutionGroup="head" type="xs:string"/>
  <xs:element name="absMember" substitutionGroup="head" abstract="true" type="xs:string"/>
  <xs:element name="absRoot" abstract="true">
    <xs:complexType><xs:sequence><xs:element ref="head" maxOccurs="unbounded"/></xs:sequence></xs:complexType>
  </xs:element>
  <xs:element name="holder">
    <xs:complexType><xs:sequence><xs:element ref="head" maxOccurs="unbounded"/></xs:sequence></xs:complexType>
  </xs:element>
  <xs:element name="strictHolder">
    <xs:complexType><xs:sequence><xs:any processContents="strict" maxOccurs="unbounded"/></xs:sequence></xs:complexType>
  </xs:element>
  <xs:element name="laxHolder">
    <xs:complexType><xs:sequence><xs:any processContents="lax" maxOccurs="unbounded"/></xs:sequence></xs:complexType>
  </xs:element>
  <xs:element name="skipHolder">
    <xs:complexType><xs:sequence><xs:any processContents="skip" maxOccurs="unbounded"/></xs:sequence></xs:complexType>
  </xs:element>
</xs:schema>`

// abstractDeclCharge is one expected cvc-elt clause 2 charge, at the element
// named element. The message names no declaration: cvc-elt clause 1 gives it
// the element's own ·expanded name·.
type abstractDeclCharge struct {
	at      xsderr.Loc
	element string
}

// wantAbstractDeclCharges fails unless got is exactly want, in walk order: each
// a cvc-elt charge at its element whose message opens on that element.
func wantAbstractDeclCharges(t *testing.T, got []*xsderr.Error, want ...abstractDeclCharge) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Violations() = %v, want exactly %d cvc-elt clause 2 charge(s)", got, len(want))
	}
	for i, w := range want {
		if got[i].Rule != ruleCvcElt {
			t.Errorf("violation %d: Rule = %q, want cvc-elt", i, got[i].Rule)
		}
		if got[i].Loc != w.at {
			t.Errorf("violation %d: Loc = %s, want the element's %s", i, got[i].Loc, w.at)
		}
		prefix := "the element " + w.element + " is governed by an element declaration whose {abstract} is true, but cvc-elt clause 2 requires"
		if !strings.HasPrefix(got[i].Msg, prefix) {
			t.Errorf("violation %d: Msg = %q, want the prefix %q", i, got[i].Msg, prefix)
		}
	}
}

// Each route to an abstract ·governing element declaration· is charged once per
// element and nothing else is. Every row but the skip one fails with
// walk.element's abstractDeclaration call deleted and the root-only arm it
// replaced restored in Validator.Assess: the rows below the root then see no
// charge at all, and the abstract root row sees one where it wants two. The
// abstract root row also fails with that root arm restored beside the call,
// which charges the root twice.
func TestAbstractGoverningDeclarationIsCharged(t *testing.T) {
	schema := parsedSchema(t, map[string]string{"main.xsd": abstractDeclSchema})
	kid := func(name string, line int) Child { return ElementChild(abstractElem(name, line, "")) }
	for _, tc := range []struct {
		why  string
		root Element
		want []abstractDeclCharge
	}{
		{
			"an abstract head used directly in content (cvc-accept clause 2.3.1)",
			abstractElem("holder", 1, "", kid("head", 2)),
			[]abstractDeclCharge{{loc(2, 1), "head"}},
		},
		{
			"a concrete member substituting for the head is not charged, the head used directly beside it is (cvc-accept clause 2.3.2)",
			abstractElem("holder", 1, "", kid("member", 2), kid("head", 3), kid("member", 4)),
			[]abstractDeclCharge{{loc(3, 1), "head"}},
		},
		{
			"an abstract member substituting for the head is charged as itself",
			abstractElem("holder", 1, "", kid("member", 2), kid("absMember", 3)),
			[]abstractDeclCharge{{loc(3, 1), "absMember"}},
		},
		{
			"an abstract root over an abstract child, each once",
			abstractElem("absRoot", 1, "", kid("head", 2)),
			[]abstractDeclCharge{{loc(1, 1), "absRoot"}, {loc(2, 1), "head"}},
		},
		{
			"a strict wildcard's child resolving to an abstract declaration (key-governing-ed clause 3)",
			abstractElem("strictHolder", 1, "", kid("member", 2), kid("head", 3)),
			[]abstractDeclCharge{{loc(3, 1), "head"}},
		},
		{
			"a lax wildcard's child resolving to an abstract declaration (key-governing-ed clause 3)",
			abstractElem("laxHolder", 1, "", kid("absMember", 2), kid("member", 3)),
			[]abstractDeclCharge{{loc(2, 1), "absMember"}},
		},
		{
			"a skip wildcard's child has no governing declaration (cvc-assess-elt clause 2)",
			abstractElem("skipHolder", 1, "", kid("head", 2)),
			nil,
		},
	} {
		t.Run(tc.why, func(t *testing.T) {
			wantAbstractDeclCharges(t, cAssess(t, schema, tc.root), tc.want...)
		})
	}
}
