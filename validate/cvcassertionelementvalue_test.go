package validate

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// The fixtures below drive an assertion {test} reading, with fn:data, the typed
// values of the d grandchildren of <temp> (xpath-functions.md §2.4), which the
// walk reports to the test's xpath.Tally once each d's own assessment is over
// (walk.tallyValue): saxonData's assert022.xsd:12, `data(event/d) instance of
// xs:date*`. <event> is D, whose d is a nillable xs:date; DR restricts D,
// narrowing d to Day, a restriction of xs:date, which an xsi:type on <event>
// selects. <odd> is Odd, admitting one local xs:date d and then any
// no-namespace element strictly, which the global d, an xs:string, governs;
// <skips> is Skips, the same with the wildcard ·skipped·; <lax> is Lax, whose
// local e is an xs:date and whose lax wildcard admits an e no global
// declaration governs, ·laxly assessed·; <und>'s d takes its type from an
// alternative whose {test} xpath declines, so its ·governing type definition·
// is undetermined.
const evSchemaText = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="Day">
    <xs:restriction base="xs:date"/>
  </xs:simpleType>
  <xs:complexType name="D">
    <xs:sequence>
      <xs:element name="d" type="xs:date" nillable="true" minOccurs="0" maxOccurs="unbounded"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="DR">
    <xs:complexContent>
      <xs:restriction base="D">
        <xs:sequence>
          <xs:element name="d" type="Day" nillable="true" minOccurs="0" maxOccurs="unbounded"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:complexType name="Odd">
    <xs:sequence>
      <xs:element name="d" type="xs:date" minOccurs="0"/>
      <xs:any namespace="##local" processContents="strict" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="Skips">
    <xs:sequence>
      <xs:element name="d" type="xs:date" minOccurs="0"/>
      <xs:any namespace="##local" processContents="skip" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="Lax">
    <xs:sequence>
      <xs:element name="e" type="xs:date" minOccurs="0"/>
      <xs:any namespace="##local" processContents="lax" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="Und">
    <xs:sequence>
      <xs:element name="d" type="xs:date" minOccurs="0">
        <xs:alternative test="string-length(@k) gt 0" type="Day"/>
      </xs:element>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="Temp">
    <xs:sequence>
      <xs:element name="event" type="D" minOccurs="0"/>
      <xs:element name="odd" type="Odd" minOccurs="0"/>
      <xs:element name="skips" type="Skips" minOccurs="0"/>
      <xs:element name="und" type="Und" minOccurs="0"/>
      <xs:element name="lax" type="Lax" minOccurs="0"/>
    </xs:sequence>
    <xs:assert test="TEST"/>
  </xs:complexType>
  <xs:element name="temp" type="Temp"/>
  <xs:element name="d" type="xs:string"/>
</xs:schema>`

// evSchema is evSchemaText with test as Temp's one assertion.
func evSchema(t *testing.T, test string) *xsd.Schema {
	t.Helper()
	escaped := strings.NewReplacer("&", "&amp;", "<", "&lt;", `"`, "&quot;").Replace(test)
	return parsedSchema(t, map[string]string{"main.xsd": strings.Replace(evSchemaText, "TEST", escaped, 1)})
}

// evD is a <d> at line holding text, carrying attrs, with the xsi prefix's
// namespace usable for xsi:nil.
func evD(line int, text string, attrs ...Attribute) Child {
	e := &testElement{name: local("d"), attrs: attrs, loc: loc(line, 5)}
	if text != "" {
		e.kids = []Child{TextChild(&testText{data: text, loc: loc(line, 8)})}
	}
	return ElementChild(e)
}

// evE is an <e> at line holding text.
func evE(line int, text string) Child {
	return ElementChild(&testElement{name: local("e"), loc: loc(line, 5),
		kids: []Child{TextChild(&testText{data: text, loc: loc(line, 8)})}})
}

// evNil is xsi:nil = true on the element at line.
func evNil(line int) Attribute {
	return ccAttr(xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "nil"}, "true", line)
}

// evTemp is the validation root <temp> at 1:1 over one container named
// container, at 2:3, carrying attrs and holding ds; the xs prefix is bound
// for an xsi:type.
func evTemp(container string, attrs []Attribute, ds ...Child) *testElement {
	c := &testElement{name: local(container), attrs: attrs, kids: ds, loc: loc(2, 3),
		bindings: map[string]string{"xs": xsd.XMLSchemaNS}}
	return &testElement{name: local("temp"), loc: loc(1, 1), kids: []Child{ElementChild(c)}}
}

// evDeclined fails unless res recorded a cvc-assertion decline at <temp>
// naming want, and charged no cvc-assertion — whatever else res charged.
func evDeclined(t *testing.T, res *Result, want string) {
	t.Helper()
	found := false
	for _, u := range res.Unevaluated() {
		found = found || u.Rule() == ruleCvcAssertion && u.Loc() == loc(1, 1) && strings.Contains(u.Msg(), want)
	}
	if !found {
		t.Errorf("Unevaluated() = %v, want a cvc-assertion decline at %s naming %q", messages(res.Unevaluated()), loc(1, 1), want)
	}
	for _, v := range res.Violations() {
		if v.Rule == ruleCvcAssertion {
			t.Errorf("Violations() holds %v: a declined assertion is never charged", v)
		}
	}
}

// fn:data reads each d of <event> TYPED, as the xs:date the {test} was
// compiled to read off D, the ·locally declared type· (key-ldt-elem): over one
// d, as assert022.v1.xml has it, over two, over none, and over a ·nilled· d,
// whose typed value is the empty sequence, `instance of xs:date*` is
// satisfied, while `instance of xs:date` over two is charged and over a
// ·nilled· d and a dated one satisfied. An <event> whose xsi:type selects DR
// governs its d by Day, a restriction of xs:date, which is read against the
// static xs:date and satisfied. With walk.tallyValue not called, every row but
// the two over no d is charged — each Tally left owing a value — and the
// no-d rows still hold.
func TestAssertionReadsGrandchildValues(t *testing.T) {
	drType := []Attribute{ccAttr(xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "type"}, "DR", 2)}
	for _, tc := range []struct {
		why, test string
		root      *testElement
		charged   bool
	}{
		{"one d", "data(event/d) instance of xs:date*", evTemp("event", nil, evD(3, "2008-07-01")), false},
		{"two d", "data(event/d) instance of xs:date*", evTemp("event", nil, evD(3, "2008-07-01"), evD(4, "2010-11-17Z")), false},
		{"no d", "data(event/d) instance of xs:date*", evTemp("event", nil), false},
		{"no event", "data(event/d) instance of xs:date*", evTemp("odd", nil), false},
		{"exactly one over two", "data(event/d) instance of xs:date", evTemp("event", nil, evD(3, "2008-07-01"), evD(4, "2010-11-17Z")), true},
		{"a ·nilled· d beside a dated one", "data(event/d) instance of xs:date", evTemp("event", nil, evD(3, "", evNil(3)), evD(4, "2010-11-17Z")), false},
		{"a d under an xsi:type DR event", "data(event/d) instance of xs:date*", evTemp("event", drType, evD(3, "2008-07-01")), false},
	} {
		t.Run(tc.why, func(t *testing.T) {
			res := aAssess(t, evSchema(t, tc.test), tc.root)
			if !tc.charged {
				wantSatisfied(t, res, tc.test)
				return
			}
			wantAssertionCharge(t, res, "the element temp is not ·valid· with respect to assertion 1 of 1 in the {assertions} of the ·governing type definition· Temp, whose {test} is ")
		})
	}
}

// A d the {test} reads that has no typed value under the type it was compiled
// to read declines the assertion, which is never charged: one the skip
// wildcard of <skips> ·skipped·, whose typed value is xs:untypedAtomic; one
// whose ·governing type definition· was not determined; one the strict
// wildcard of <odd> admits, governed by the global d's xs:string, not derived
// from xs:date; and one with no ·actual value· under xs:date ("abc", which
// cvc-datatype-valid charges). The undetermined row is a guard — the
// assertion was declined before this slice whatever d was — and so is the
// ·skipped· one. With walk.tallyValue not called, the xs:string and abc rows
// are charged and the rest still decline; with assertionAncestry.tallySkipped
// taking no lack, the ·skipped· row is charged; with walk.elementBelow's
// derivation test removed, the xs:string row is charged.
func TestAssertionDeclinesAnUnreadGrandchild(t *testing.T) {
	for _, tc := range []struct {
		why, test, want string
		root            *testElement
	}{
		{"a ·skipped· d", "data(skips/d) instance of xs:date*", "it is ·skipped·",
			evTemp("skips", nil, evD(3, "2008-07-01"), evD(4, "x"))},
		{"a d whose type was not determined", "data(und/d) instance of xs:date*", "a violation or an unevaluated check was recorded for it",
			evTemp("und", nil, evD(3, "2008-07-01"))},
		{"a ·laxly assessed· e", "data(lax/e) instance of xs:date*", "it has no ·governing type definition·",
			evTemp("lax", nil, evE(3, "2008-07-01"), evE(4, "2010-11-17"))},
		{"a d of a type not derived from xs:date", "data(odd/d) instance of xs:date*", "the type the {test} was compiled to read, nor validly derived from it",
			evTemp("odd", nil, evD(3, "2008-07-01"), evD(4, "x"))},
		{"a d with no ·actual value·", "data(event/d) instance of xs:date*", "a violation or an unevaluated check was recorded for it",
			evTemp("event", nil, evD(3, "abc"))},
	} {
		t.Run(tc.why, func(t *testing.T) {
			evDeclined(t, aAssess(t, evSchema(t, tc.test), tc.root), tc.want)
		})
	}
}

// Guard: the forms beyond fn:data's argument under `instance of` — a general
// comparison and fn:count over the same path — are still declined by the
// evaluator, as they were before this slice.
func TestAssertionGrandchildPathOtherFormsDecline(t *testing.T) {
	for _, test := range []string{"event/d = xs:date('2008-07-01')", "count(event/d) eq 1"} {
		evDeclined(t, aAssess(t, evSchema(t, test), evTemp("event", nil, evD(3, "2008-07-01"))), "this engine's XPath evaluator declined it")
	}
}

// The closure answers a one-step path exactly as
// [xsd.Schema.LocallyDeclaredElementType] answers its name within the
// element's type; a two-step path is the last step's ·locally declared type·
// within the static type of the first, a simple one included; a simple middle
// step and the empty path answer false. With assertionElementTypes looking the
// last step up within the element's own type, the [event, d] row fails.
func TestAssertionElementTypesFollowsTheChain(t *testing.T) {
	schema := evSchema(t, "true()")
	td, _ := schema.Type(local("Temp"))
	temp, isComplex := td.(xsd.ComplexType)
	if !isComplex {
		t.Fatalf("Temp is %T, want a complex type", td)
	}
	types := (&walk{schema: schema}).assertionElementTypes(temp)
	for _, name := range []string{"event", "odd", "d", "none"} {
		want, wantOK := schema.LocallyDeclaredElementType(temp, local(name))
		got, ok := types([]xsd.QName{local(name)})
		if ok != wantOK || ok && !sameType(got, want) {
			t.Errorf("[%s]: (%v, %v), want (%v, %v)", name, got, ok, want, wantOK)
		}
	}
	if got, ok := types([]xsd.QName{local("event"), local("d")}); !ok || got.Name() != (xsd.QName{Space: xsd.XMLSchemaNS, Local: "date"}) {
		t.Errorf("[event, d] = (%v, %v), want (xs:date, true)", got, ok)
	}
	for _, path := range [][]xsd.QName{nil, {local("event"), local("d"), local("x")}, {local("event"), local("q")}} {
		if _, ok := types(path); ok {
			t.Errorf("%v: answered, want false", path)
		}
	}
}
