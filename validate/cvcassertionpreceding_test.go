package validate

import (
	"slices"
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// cpSchemaText is saxonData/Assert/assert005.xsd's shape: an <outer> of
// <inner> children, each asserting `not(a[preceding::a[not(b)]])` over its
// children, which choose between <a> and an element of urn:skip that a skip
// wildcard ·skips·; each a ·skips· its own no-namespace children too, b among
// them.
const cpSchemaText = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="AType">
    <xs:sequence><xs:any namespace="##local" processContents="skip" minOccurs="0" maxOccurs="unbounded"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="IType">
    <xs:choice minOccurs="0" maxOccurs="unbounded">
      <xs:element name="a" type="AType"/>
      <xs:any namespace="urn:skip" processContents="skip"/>
    </xs:choice>
    <xs:assert test="not(a[preceding::a[not(b)]])"/>
  </xs:complexType>
  <xs:complexType name="OType">
    <xs:sequence><xs:element name="inner" type="IType" maxOccurs="unbounded"/></xs:sequence>
  </xs:complexType>
  <xs:element name="outer" type="OType"/>
</xs:schema>`

// cpCharges is the lines of the <inner> elements res charged cvc-assertion
// at, failing the test where it recorded a cvc-assertion decline.
func cpCharges(t *testing.T, res *Result) []int {
	t.Helper()
	for _, u := range res.Unevaluated() {
		if u.Rule() == ruleCvcAssertion {
			t.Errorf("Unevaluated() holds %q, want every assertion evaluated", u.Msg())
		}
	}
	var lines []int
	for _, v := range res.Violations() {
		if v.Rule == ruleCvcAssertion {
			lines = append(lines, v.Loc.Line)
		}
	}
	return lines
}

// cvc-assertion clause 1.3 roots each <inner>'s tree at that <inner>, so
// `preceding::` from its first child selects nothing in an earlier <inner>:
// two sibling <inner>s each holding one a lacking b are both satisfied, where
// one <inner> holding two such a is charged (assert005's n1). A ·skipped· b is
// reported by name and is a child of its a like any other, and so is a
// ·skipped· a nested in an earlier sibling, which precedes a later child a;
// one nested in a later sibling precedes no child a. Every row is declined
// instead, and fails, with xpath's ctaParser.childrenPrecededLength measuring
// nothing.
func TestAssertionPrecedingThroughTheWalk(t *testing.T) {
	schema := parsedSchema(t, map[string]string{"main.xsd": cpSchemaText})
	a, b := local("a"), local("b")
	x := xsd.QName{Space: "urn:skip", Local: "x"}
	inner := func(line int, kids ...Child) Child { return ccNode(local("inner"), line, nil, kids...) }
	outer := func(kids ...Child) *testElement {
		return &testElement{name: local("outer"), kids: kids, loc: loc(1, 1)}
	}
	for _, tc := range []struct {
		why  string
		root *testElement
		want []int
	}{
		{"two inner, each one a lacking b", outer(inner(2, ccNode(a, 2, nil)), inner(3, ccNode(a, 3, nil))), nil},
		{"one inner, two a lacking b", outer(inner(2, ccNode(a, 2, nil), ccNode(a, 2, nil))), []int{2}},
		{"a with a skipped b, then a", outer(inner(2, ccNode(a, 2, nil, ccNode(b, 3, nil)), ccNode(a, 4, nil))), nil},
		{"a skipped a lacking b nested in an earlier sibling, then a",
			outer(inner(2, ccNode(x, 2, nil, ccNode(a, 3, nil)), ccNode(a, 4, nil))), []int{2}},
		{"a skipped a with b nested in an earlier sibling, then a",
			outer(inner(2, ccNode(x, 2, nil, ccNode(a, 3, nil, ccNode(b, 3, nil))), ccNode(a, 4, nil))), nil},
		{"a, then a skipped a nested in a later sibling",
			outer(inner(2, ccNode(a, 2, nil), ccNode(x, 3, nil, ccNode(a, 4, nil)))), nil},
	} {
		t.Run(tc.why, func(t *testing.T) {
			got := cpCharges(t, aAssess(t, schema, tc.root))
			if !slices.Equal(got, tc.want) {
				t.Errorf("cvc-assertion charged at lines %v, want %v (Violations() = %v)", got, tc.want, aAssess(t, schema, tc.root).Violations())
			}
		})
	}
}
