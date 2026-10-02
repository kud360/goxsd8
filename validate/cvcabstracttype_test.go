package validate

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// These fixtures drive cvc-type (§3.3.4.4) clause 2 ([walk.abstractType]): an
// element whose ·governing type definition· is a complex type with {abstract}
// true is charged, at the element, whichever clause of key-governing-type-elem
// settled that type.

// abstractSchema declares a concrete Base, Abs (abstract, extending Base),
// Concrete (extending Base), Other (abstract, unrelated to Base), the roots
// absRoot of Abs and baseRoot of Base, and holder, whose content is a local kid
// of Base followed by an optional lax wildcard.
func abstractSchema(t *testing.T) *xsd.Schema {
	t.Helper()
	return parsedSchema(t, map[string]string{"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base"><xs:sequence><xs:element name="c" type="xs:int" minOccurs="0"/></xs:sequence></xs:complexType>
  <xs:complexType name="Abs" abstract="true"><xs:complexContent><xs:extension base="Base"/></xs:complexContent></xs:complexType>
  <xs:complexType name="Concrete"><xs:complexContent><xs:extension base="Base"/></xs:complexContent></xs:complexType>
  <xs:complexType name="Other" abstract="true"/>
  <xs:element name="absRoot" type="Abs"/>
  <xs:element name="baseRoot" type="Base"/>
  <xs:element name="holder">
    <xs:complexType><xs:sequence>
      <xs:element name="kid" type="Base"/>
      <xs:any processContents="lax" minOccurs="0"/>
    </xs:sequence></xs:complexType>
  </xs:element>
</xs:schema>`})
}

// abstractElem is an element named name at line, carrying an xsi:type of
// xsiType where it is non-empty, and holding kids.
func abstractElem(name string, line int, xsiType string, kids ...Child) *testElement {
	e := &testElement{name: local(name), loc: loc(line, 1), kids: kids}
	if xsiType != "" {
		e.attrs = []Attribute{&testAttribute{
			name:  xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "type"},
			value: xsiType, loc: loc(line, 10)}}
	}
	return e
}

// abstractHolder is <holder> over a <kid> at line 2 carrying kidType as its
// xsi:type, then, where wildType is non-empty, a <u> at line 3 carrying it.
func abstractHolder(kidType, wildType string) *testElement {
	h := abstractElem("holder", 1, "", ElementChild(abstractElem("kid", 2, kidType)))
	if wildType != "" {
		h.kids = append(h.kids, ElementChild(abstractElem("u", 3, wildType)))
	}
	return h
}

// wantAbstractCharge fails unless got is exactly one cvc-type clause 2 charge
// at at, whose message opens on the element and then the type, so swapping the
// two fails it.
func wantAbstractCharge(t *testing.T, got []*xsderr.Error, at xsderr.Loc, element, typ string) {
	t.Helper()
	tWantCharge(t, got, "2", at)
	want := "the element " + element + " is governed by the complex type definition " + typ + ", whose {abstract} is true"
	if !strings.HasPrefix(got[0].Msg, want) {
		t.Errorf("Msg = %q, want the prefix %q", got[0].Msg, want)
	}
}

// Each governance shape that reaches an abstract complex type is charged
// cvc-type clause 2 at the element, and nothing else is: a root declared with
// it (key-governing-type-elem clause 4), a root and a declared child whose
// xsi:type names it and ·overrides· the declared Base (clause 3), an undeclared
// root whose xsi:type names it (clause 8), and a child the lax wildcard admits,
// resolving no declaration, whose xsi:type names it (clause 8). Each fails with
// walk.element's abstractType call deleted (no violation at all).
func TestAbstractGoverningTypeIsCharged(t *testing.T) {
	schema := abstractSchema(t)
	for _, tc := range []struct {
		why     string
		root    Element
		at      xsderr.Loc
		element string
	}{
		{"a root declared with the abstract type", abstractElem("absRoot", 1, ""), loc(1, 1), "absRoot"},
		{"a declared root whose xsi:type overrides with it", abstractElem("baseRoot", 1, "Abs"), loc(1, 1), "baseRoot"},
		{"an undeclared root whose xsi:type names it", abstractElem("loose", 1, "Abs"), loc(1, 1), "loose"},
		{"a declared child whose xsi:type overrides with it", abstractHolder("Abs", ""), loc(2, 1), "kid"},
		{"a lax wildcard's unresolved child whose xsi:type names it", abstractHolder("", "Abs"), loc(3, 1), "u"},
	} {
		t.Run(tc.why, func(t *testing.T) {
			wantAbstractCharge(t, cAssess(t, schema, tc.root), tc.at, tc.element, "Abs")
		})
	}
}

// The concrete controls for the rows above: the same shapes under Concrete,
// which extends Base exactly as Abs does, charge nothing, so the charge turns on
// {abstract} and not on the derivation or the shape of the document.
func TestConcreteGoverningTypeIsNotCharged(t *testing.T) {
	schema := abstractSchema(t)
	for _, tc := range []struct {
		why  string
		root Element
	}{
		{"a declared root whose xsi:type overrides with Concrete", abstractElem("baseRoot", 1, "Concrete")},
		{"an undeclared root whose xsi:type names Concrete", abstractElem("loose", 1, "Concrete")},
		{"a declared child whose xsi:type overrides with Concrete", abstractHolder("Concrete", "")},
		{"a lax wildcard's unresolved child whose xsi:type names Concrete", abstractHolder("", "Concrete")},
	} {
		wantSilence(t, cAssess(t, schema, tc.root), tc.why)
	}
}

// An xsi:type that fails cvc-elt clause 4 does not govern: the selected type
// does (the Note under cvc-elt), so an abstract Other that does not ·override·
// Base is charged cvc-elt clause 4 and NOT cvc-type clause 2. It fails with
// abstractType reading the xsi:type's type in place of the governing one.
func TestFailedOverrideAbstractTypeIsNotCharged(t *testing.T) {
	got := cAssess(t, abstractSchema(t), abstractHolder("Other", ""))
	eWantClause(t, got, "4")
	if got[0].Loc != loc(2, 1) {
		t.Errorf("Loc = %s, want the kid's %s", got[0].Loc, loc(2, 1))
	}
}

// The charge does not halt the walk: key-sva (§3.3.4.6) clause 1.1.3 gates
// ·strict assessment· on cvc-type clause 1 alone, so the abstract-typed root's
// [[children]] are still assessed against Abs, and an xs:int child holding a
// non-integer is charged beside it, in walk order.
func TestAbstractGoverningTypeStillAssessesContent(t *testing.T) {
	c := abstractElem("c", 2, "", TextChild(&testText{data: "x", loc: loc(2, 4)}))
	got := cAssess(t, abstractSchema(t), abstractElem("absRoot", 1, "", ElementChild(c)))
	if len(got) != 2 {
		t.Fatalf("Violations() = %v, want the clause 2 charge and the child's", got)
	}
	wantAbstractCharge(t, got[:1], loc(1, 1), "absRoot", "Abs")
	tWantCharge(t, got[1:], "3.1.3", loc(2, 1))
}
