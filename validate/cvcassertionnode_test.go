package validate

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// The fixtures below are the suite's ibmData/mixed/assertions/namespace ns3
// and ns4 shapes (assert_027, assert_028): an x in http://www.example.org
// holding one y, whose {test}s read E itself as a node — fn:namespace-uri, on
// x and on y, and fn:in-scope-prefixes, on x — off the Element the walk hands
// xpath.AssertionTest.Evaluate.

// nodeNS is the target namespace both schemas declare.
const nodeNS = "http://www.example.org"

// nodeSchema is ns3.xsd where yTest is non-empty, asserting it on y as well,
// and ns4.xsd's shape otherwise, both over xTest on x.
func nodeSchema(t *testing.T, xTest, yTest string) *xsd.Schema {
	t.Helper()
	yType := `<xs:complexType/>`
	if yTest != "" {
		yType = `<xs:complexType><xs:assert test="` + yTest + `"/></xs:complexType>`
	}
	return parsedSchema(t, map[string]string{"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="` + nodeNS + `" elementFormDefault="qualified">
  <xs:element name="x">
    <xs:complexType>
      <xs:sequence><xs:element name="y">` + yType + `</xs:element></xs:sequence>
      <xs:assert test="` + xTest + `"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`})
}

// nodeDoc is <x> in nodeNS binding bindings, holding one <y> that inherits them.
func nodeDoc(bindings map[string]string) *testElement {
	y := &testElement{name: xsd.QName{Space: nodeNS, Local: "y"}, bindings: bindings, loc: loc(2, 3)}
	return &testElement{name: xsd.QName{Space: nodeNS, Local: "x"}, bindings: bindings, kids: []Child{ElementChild(y)}, loc: loc(1, 1)}
}

// ns3.xml and ns4.xml are decided VALID: `namespace-uri(.) =
// 'http://www.example.org'` holds on x and on y, each in that namespace, and
// `in-scope-prefixes(.) = 'a'` holds on an x binding a. Each is decided the
// other way where the document differs: a binding of b in a's place charges
// ns4's {test}, and ns3's, written against another namespace, is charged on
// both x and y. With xpath's fn:namespace-uri arm removed, ns3.xml is declined
// on x and y; with its fn:in-scope-prefixes comparison unread, ns4.xml is
// declined; with the walk handing Evaluate an element that binds no prefix,
// ns4.xml is charged.
func TestAssertionReadsTheElementsNameAndBindings(t *testing.T) {
	defaultNS := map[string]string{"": nodeNS}
	bindsA := map[string]string{"": nodeNS, "a": "http://test"}
	bindsB := map[string]string{"": nodeNS, "b": "http://test"}

	ns3 := nodeSchema(t, "namespace-uri(.) = '"+nodeNS+"'", "namespace-uri(.) = '"+nodeNS+"'")
	wantSatisfied(t, aAssess(t, ns3, nodeDoc(defaultNS)), "ns3.xml")
	ns4 := nodeSchema(t, "in-scope-prefixes(.) = 'a'", "")
	wantSatisfied(t, aAssess(t, ns4, nodeDoc(bindsA)), "ns4.xml")

	res := aAssess(t, ns4, nodeDoc(bindsB))
	if got := res.Unevaluated(); len(got) != 0 {
		t.Fatalf("ns4 over b: Unevaluated() = %v, want none", messages(got))
	}
	if got := res.Violations(); len(got) != 1 || got[0].Rule != ruleCvcAssertion || got[0].Loc != loc(1, 1) {
		t.Errorf("ns4 over b: Violations() = %v, want one cvc-assertion charge at %s", got, loc(1, 1))
	}

	other := nodeSchema(t, "namespace-uri(.) = 'urn:other'", "namespace-uri() = 'urn:other'")
	res = aAssess(t, other, nodeDoc(defaultNS))
	if got := res.Unevaluated(); len(got) != 0 {
		t.Fatalf("another namespace: Unevaluated() = %v, want none", messages(got))
	}
	got := res.Violations()
	if len(got) != 2 || got[0].Loc != loc(2, 3) || got[1].Loc != loc(1, 1) {
		t.Errorf("another namespace: Violations() = %v, want cvc-assertion at y %s, then at x %s", got, loc(2, 3), loc(1, 1))
	}
}
