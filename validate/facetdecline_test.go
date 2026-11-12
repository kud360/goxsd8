package validate_test

import (
	"strings"
	"testing"
)

// facetDeclined is the namespace urn:t with Pos, an xs:integer restriction
// whose assertions facet's {test} calls fn:tokenize, which this engine's XPath
// evaluator declines, Five, one whose {test} it decides, and r over an e whose
// type carries an <xs:assert> with Pos's {test} and a p of Pos, and f over a q
// of Five.
const facetDeclined = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:simpleType name="Pos"><xs:restriction base="xs:integer"><xs:assertion test="count(tokenize(string($value),'x')) = 1"/></xs:restriction></xs:simpleType>
  <xs:simpleType name="Five"><xs:restriction base="xs:integer"><xs:assertion test="$value eq 5"/></xs:restriction></xs:simpleType>
  <xs:complexType name="E">
    <xs:simpleContent><xs:extension base="xs:integer"><xs:assert test="count(tokenize(string($value),'x')) = 1"/></xs:extension></xs:simpleContent>
  </xs:complexType>
  <xs:element name="r">
    <xs:complexType><xs:sequence><xs:element name="e" type="t:E"/><xs:element name="p" type="t:Pos"/></xs:sequence></xs:complexType>
  </xs:element>
  <xs:element name="f">
    <xs:complexType><xs:sequence><xs:element name="q" type="t:Five"/></xs:sequence></xs:complexType>
  </xs:element>
</xs:schema>`

// An assertions facet's {test} the evaluator declines is recorded with the
// reason xpath.FacetAssertions states for it, after "was not evaluated
// because" and before the cvc-assertions-valid citation, and that reason is the
// one the <xs:assert> path prints for the same {test} after "declined it
// because": a function decline is the one kind the two façades state alike.
func TestDeclinedFacetAssertionNamesWhy(t *testing.T) {
	const reason = "function fn:tokenize is not supported"
	res := orderAssess(t, facetDeclined, `<r xmlns="urn:t"><e>2</e><p>5</p></r>`)
	if got := res.Violations(); len(got) != 0 {
		t.Fatalf("Violations() = %v, want none: a decline is never a charge", got)
	}
	var assertMsg, facetMsg string
	for _, u := range res.Unevaluated() {
		switch u.Rule() {
		case "cvc-assertion":
			assertMsg = u.Msg()
		case "cvc-assertions-valid":
			facetMsg = u.Msg()
		}
	}
	if !strings.Contains(assertMsg, "declined it because "+reason+", ") {
		t.Errorf("<xs:assert> record = %q, want it to say it was declined because %s", assertMsg, reason)
	}
	want := `whose {test} is "count(tokenize(string($value),'x')) = 1", was not evaluated because ` + reason + `, so it is undecided whether the value is facet-valid with respect to it, which cvc-assertions-valid requires`
	if !strings.Contains(facetMsg, want) {
		t.Errorf("assertions-facet record = %q, want it to carry %q", facetMsg, want)
	}
}

// A {test} the evaluator decides records nothing: `$value eq 5` holds on 5.
func TestDecidedFacetAssertionRecordsNothing(t *testing.T) {
	res := orderAssess(t, facetDeclined, `<f xmlns="urn:t"><q>5</q></f>`)
	if len(res.Violations()) != 0 || len(res.Unevaluated()) != 0 {
		t.Errorf("Violations() = %v, Unevaluated() = %d records, want both empty: the {test} holds", res.Violations(), len(res.Unevaluated()))
	}
}
