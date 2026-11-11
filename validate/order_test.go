package validate_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/validate/xmlsrc"
)

// orderSmall is the namespace urn:t with small, xs:int at most 5, and the
// elements the rows below assess.
const orderSmall = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:simpleType name="small"><xs:restriction base="xs:int"><xs:maxInclusive value="5"/></xs:restriction></xs:simpleType>
  <xs:element name="r">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="e" type="t:small"/>
        <xs:element name="f" type="t:small"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="q">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="e" type="t:small"/>
      </xs:sequence>
      <xs:attribute name="a" type="t:small"/>
    </xs:complexType>
  </xs:element>
  <xs:element name="k">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="e" type="t:small" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="u"><xs:selector xpath="t:e"/><xs:field xpath="."/></xs:unique>
  </xs:element>
  <xs:element name="x" fixed="">
    <xs:complexType mixed="true">
      <xs:sequence>
        <xs:element name="e" type="t:small" minOccurs="0"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

// orderAsserting declares r in urn:t, asserting fewer than three small e
// children.
const orderAsserting = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:element name="r">
    <xs:complexType>
      <xs:sequence><xs:element name="e" maxOccurs="unbounded"><xs:simpleType><xs:restriction base="xs:int"><xs:maxInclusive value="5"/></xs:restriction></xs:simpleType></xs:element></xs:sequence>
      <xs:assert test="count(t:e) lt 3"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

// orderAttributes declares q with an optional int a, a required int b, and a
// QName d whose default "p:x" binds no prefix p, i of the simple type xs:int,
// and s, an xs:int simple-content extension carrying an int a.
const orderAttributes = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="q">
    <xs:complexType>
      <xs:attribute name="a" type="xs:int"/>
      <xs:attribute name="b" type="xs:int" use="required"/>
    </xs:complexType>
  </xs:element>
  <xs:element name="d">
    <xs:complexType>
      <xs:attribute name="a" type="xs:int"/>
      <xs:attribute name="d" type="xs:QName" default="p:x"/>
    </xs:complexType>
  </xs:element>
  <xs:element name="i" type="xs:int"/>
  <xs:element name="s">
    <xs:complexType><xs:simpleContent><xs:extension base="xs:int">
      <xs:attribute name="a" type="xs:int"/>
    </xs:extension></xs:simpleContent></xs:complexType>
  </xs:element>
</xs:schema>`

// orderIDs declares r in urn:t over e children carrying an ID, an IDREF and a
// small n.
const orderIDs = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:element name="r">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="e" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="id" type="xs:ID"/>
            <xs:attribute name="ref" type="xs:IDREF"/>
            <xs:attribute name="n"><xs:simpleType><xs:restriction base="xs:int"><xs:maxInclusive value="5"/></xs:restriction></xs:simpleType></xs:attribute>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

// orderDeclined declares r over c children, each type carrying an assertion
// whose {test} calls fn:unparsed-text, which the engine declines.
const orderDeclined = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="r">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="c">
          <xs:complexType><xs:assert test="unparsed-text('x') eq ''"/></xs:complexType>
        </xs:element>
      </xs:sequence>
      <xs:assert test="unparsed-text('x') eq ''"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

// The lists [validate.Result.Violations] and [validate.Result.Unevaluated]
// return are in the order the walk charges, which that method's doc states and
// which is not document order: each row pins one whole list as "line:col rule"
// entries, and clause, where set, is a substring of the last entry's Msg naming
// which check of its rule it is. Rows 1, 3 to 9 and 11 to 13 each carry a check settled
// once an element's attributes, [[children]] or whole walk are exhausted, after
// a charge at a later location; row 2 is the guard that siblings stay in
// document order, and row 10 that the parent's charge at a child's location
// precedes the child's own.
func TestViolationsAreListedInWalkOrder(t *testing.T) {
	for _, tc := range []struct {
		name, schema, doc string
		unevaluated       bool
		want              []string
		clause            string
	}{
		{"R1 assertion after its element's children", orderAsserting,
			"<r xmlns=\"urn:t\">\n  <e>9</e>\n  <e>1</e>\n  <e>1</e>\n  <e>9</e>\n</r>", false,
			[]string{"2:3 cvc-type", "5:3 cvc-type", "1:1 cvc-assertion"}, ""},
		{"R2 siblings in document order", orderSmall,
			"<k xmlns=\"urn:t\">\n  <e>9</e>\n  <e>8</e>\n</k>", false,
			[]string{"2:3 cvc-type", "3:3 cvc-type"}, ""},
		{"R3 content ending short after a child's charge", orderSmall,
			"<r xmlns=\"urn:t\">\n  <e>9</e>\n</r>", false,
			[]string{"2:3 cvc-type", "1:1 cvc-complex-content"}, ""},
		{"R4 content ending short after an attribute's charge", orderSmall,
			"<q xmlns=\"urn:t\" a=\"9\"/>", false,
			[]string{"1:18 cvc-attribute", "1:1 cvc-complex-content"}, ""},
		{"R5 required attribute absent after an attribute's charge", orderAttributes,
			"<q a=\"x\"/>", false,
			[]string{"1:4 cvc-attribute", "1:1 cvc-complex-type"}, "clause 3"},
		{"R6 simple-content value after an attribute's charge", orderAttributes,
			"<s a=\"x\">notint</s>", false,
			[]string{"1:4 cvc-attribute", "1:1 cvc-complex-type"}, "clause 1.2"},
		{"R13 simple-typed value after an attribute's charge", orderAttributes,
			"<i a=\"x\">notint</i>", false,
			[]string{"1:4 cvc-type", "1:1 cvc-type"}, "clause 3.1.3"},
		{"R7 fixed value after a child's charge", orderSmall,
			"<x xmlns=\"urn:t\">\n  <e>9</e>\n</x>", false,
			[]string{"2:3 cvc-type", "1:1 cvc-elt"}, ""},
		{"R8 identity constraint after the scoping element's subtree", orderSmall,
			"<k xmlns=\"urn:t\">\n  <e>1</e>\n  <e>1</e>\n  <e>9</e>\n</k>", false,
			[]string{"4:3 cvc-type", "3:3 cvc-identity-constraint"}, ""},
		{"R9 cvc-id after the whole walk", orderIDs,
			"<r xmlns=\"urn:t\">\n  <e id=\"a\"/>\n  <e id=\"a\" ref=\"zz\"/>\n  <e n=\"9\"/>\n</r>", false,
			[]string{"4:6 cvc-attribute", "3:6 cvc-id", "3:13 cvc-id"}, ""},
		{"R10 parent's charge at a child before the child's own", orderSmall,
			"<r xmlns=\"urn:t\">\n  <f>9</f>\n  <e>1</e>\n</r>", false,
			[]string{"2:3 cvc-complex-content", "2:3 cvc-type"}, ""},
		{"R11 defaulted attribute after an attribute's charge", orderAttributes,
			"<d a=\"x\"/>", false,
			[]string{"1:4 cvc-attribute", "1:1 cvc-complex-type"}, "clause 4"},
		{"R12 declined assertion after its element's children", orderDeclined,
			"<r>\n  <c/>\n</r>", true,
			[]string{"2:3 cvc-assertion", "1:1 cvc-assertion"}, ""},
	} {
		res := orderAssess(t, tc.schema, tc.doc)
		var got, msgs []string
		if tc.unevaluated {
			for _, u := range res.Unevaluated() {
				got = append(got, fmt.Sprintf("%d:%d %s", u.Loc().Line, u.Loc().Col, u.Rule()))
				msgs = append(msgs, u.Msg())
			}
		}
		if !tc.unevaluated {
			for _, e := range res.Violations() {
				got = append(got, fmt.Sprintf("%d:%d %s", e.Loc.Line, e.Loc.Col, e.Rule))
				msgs = append(msgs, e.Msg)
			}
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s: listed %q, want %q", tc.name, got, tc.want)
			continue
		}
		if tc.clause != "" && !strings.Contains(msgs[len(msgs)-1], tc.clause) {
			t.Errorf("%s: last Msg = %q, want %s", tc.name, msgs[len(msgs)-1], tc.clause)
		}
	}
}

// orderAssess assesses doc against schemaText, failing t unless both read and
// the walk reached the document's end.
func orderAssess(t *testing.T, schemaText, doc string) *validate.Result {
	t.Helper()
	backend := strict.New()
	schema, err := parser.Parse("order.xsd",
		parser.WithBackend(backend),
		parser.WithResolver(loader.Map(map[string]string{"order.xsd": schemaText})))
	if err != nil {
		t.Fatalf("parser.Parse: %v", err)
	}
	v, err := validate.New(schema, backend)
	if err != nil {
		t.Fatalf("validate.New: %v", err)
	}
	res, err := xmlsrc.Validate(v, strings.NewReader(doc), xmlsrc.WithURI("order.xml"))
	if err != nil {
		t.Fatalf("xmlsrc.Validate: %v", err)
	}
	if res.Err() != nil {
		t.Fatalf("Err() = %v, want the walk to reach the end", res.Err())
	}
	return res
}
