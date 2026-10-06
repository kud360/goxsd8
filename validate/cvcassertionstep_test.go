package validate

import "testing"

// The fixtures below drive an assertion {test} asking whether one element
// step, `N` or `.//N`, selects a node of the element's subtree, whatever N's
// type: the walk reports the subtree to the test's xpath.Tally
// (walk.tallyElement, assertionAncestry.tallySkipped) and keeps no child's
// value for it (xpath.AssertionTest.ReadsChild).

// esSchema18 is IBM's test18.xsd (ibmData/mixed/assertions), annotations
// dropped: X asserts `a and b and d` and Y `a and b and c and d` over a group
// whose a is element-only, asserting `count(a1) eq @aCount` itself, and whose
// b, c and d are xs:string.
const esSchema18 = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="test">
    <xs:complexType>
      <xs:sequence><xs:element name="x" type="X"/><xs:element name="y" type="Y"/></xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:complexType name="X"><xs:group ref="List1"/><xs:assert test="a and b and d"/></xs:complexType>
  <xs:complexType name="Y"><xs:group ref="List1"/><xs:assert test="a and b and c and d"/></xs:complexType>
  <xs:group name="List1">
    <xs:sequence>
      <xs:element name="a" minOccurs="0">
        <xs:complexType>
          <xs:sequence><xs:element name="a1" type="xs:string" maxOccurs="unbounded"/></xs:sequence>
          <xs:attribute name="aCount" type="xs:nonNegativeInteger"/>
          <xs:assert test="count(a1) eq @aCount"/>
        </xs:complexType>
      </xs:element>
      <xs:element name="b" type="xs:string" minOccurs="0"/>
      <xs:element name="c" type="xs:string" minOccurs="0"/>
      <xs:element name="d" type="xs:string" minOccurs="0"/>
    </xs:sequence>
  </xs:group>
</xs:schema>`

// esTest18 is a test18 instance: <x> at line 2 and <y> at line 10, each with
// an <a> one line below whose aCount is xCount and yCount over two <a1>s, then
// <b>, an optional <c> under <y> alone, and <d>.
func esTest18(xCount, yCount string, withC bool) *testElement {
	a := func(line int, count string) Child {
		return ccNode(local("a"), line, []Attribute{ccAttr(local("aCount"), count, line)},
			acKid("a1", line+1, "aaa"), acKid("a1", line+2, "aaa.."))
	}
	yKids := []Child{a(11, yCount), acKid("b", 14, "world")}
	if withC {
		yKids = append(yKids, acKid("c", 15, "hello.."))
	}
	yKids = append(yKids, acKid("d", 16, "world.."))
	return &testElement{name: local("test"), loc: loc(1, 1), kids: []Child{
		ccNode(local("x"), 2, nil, a(3, xCount), acKid("b", 6, "world"), acKid("d", 7, "world..")),
		ccNode(local("y"), 10, nil, yKids...),
	}}
}

// assert_018_2: test18_1.xml is valid, every assertion evaluated — `a` over
// the element-only a decided by the Tally, and b, c, d read as nodes and not as
// values — and test18_2.xml is charged cvc-assertion three times and recorded
// unevaluated nowhere: each <a>'s count against its aCount of 4 and 3, and <y>
// for its missing <c>, while <x>, holding a, b and d, is not charged. Both
// rows record a decline for <x> and <y> instead with
// ctaParser.selectedElements' booleanExpr call site removed.
func TestAssertionElementStepAssert018(t *testing.T) {
	schema := parsedSchema(t, map[string]string{"main.xsd": esSchema18})
	wantSatisfied(t, aAssess(t, schema, esTest18("2", "2", true)), "test18_1")
	res := aAssess(t, schema, esTest18("4", "3", false))
	if got := res.Unevaluated(); len(got) != 0 {
		t.Fatalf("test18_2: Unevaluated() = %v, want none", messages(got))
	}
	var at []string
	for _, v := range res.Violations() {
		if v.Rule != ruleCvcAssertion {
			t.Errorf("test18_2: Violations() holds %v, want cvc-assertion alone", v)
			continue
		}
		at = append(at, v.Loc.String())
	}
	want := []string{loc(3, 3).String(), loc(11, 3).String(), loc(10, 3).String()}
	if len(at) != len(want) || !esSameSet(at, want) {
		t.Errorf("test18_2: cvc-assertion charged at %v, want %v: both <a>s and <y>, never <x>", at, want)
	}
}

// esSameSet reports whether got and want hold the same strings, each once.
func esSameSet(got, want []string) bool {
	for _, w := range want {
		n := 0
		for _, g := range got {
			if g == w {
				n++
			}
		}
		if n != 1 {
			return false
		}
	}
	return true
}

// esSchema009 is Saxon's assert009.xsd: <doc> admits anything under a skip
// wildcard and asserts `not(.//disallowed)`.
const esSchema009 = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" elementFormDefault="qualified" attributeFormDefault="unqualified">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence><xs:any processContents="skip" maxOccurs="unbounded"/></xs:sequence>
      <xs:assert test="not(.//disallowed)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

// assert009: `.//disallowed` selects a <disallowed> three levels inside a
// ·skipped· <a> (assertionAncestry.tallySkipped), so n1, holding one, is
// charged at <doc>, and v1, holding <allowed> there instead, is valid. Both
// rows record a decline instead with ctaParser.selectedElements' booleanExpr
// call site removed; the n1 row is satisfied instead with tallySkipped
// reporting nothing below the ·skipped· child.
func TestAssertionElementStepAssert009(t *testing.T) {
	schema := parsedSchema(t, map[string]string{"main.xsd": esSchema009})
	doc := func(leaf string) *testElement {
		row := func(line int, inner Child) Child {
			return ccNode(local("a"), line, nil, ccNode(local("b"), line, nil, ccNode(local("c"), line, nil, inner)))
		}
		return &testElement{name: local("doc"), loc: loc(1, 1), kids: []Child{
			row(2, acKid("c", 2, "asdasas")),
			row(3, ccNode(local(leaf), 3, nil)),
			row(4, acKid("c", 4, "asdasas")),
		}}
	}
	wantAssertionCharge(t, aAssess(t, schema, doc("disallowed")),
		"the element doc is not ·valid· with respect to assertion 1 of 1 in the {assertions} of the ·governing type definition· ")
	wantSatisfied(t, aAssess(t, schema, doc("allowed")), "assert009.v1")
}

// A child whose existence alone is asked is a node whatever its own
// assessment found: `exists(n)` holds and `empty(n)` is charged over an <n>
// whose "abc" is no xs:int, which cvc-datatype charges on its own, where a
// {test} reading n's value, `n = 1`, still declines on the lackingChild it
// cannot read (guard). The exists and empty rows record that decline instead
// with ctaParser.selectedElements' presenceArgument call site removed.
func TestAssertionElementStepReadsNoValue(t *testing.T) {
	bad := acKid("n", 2, "abc")
	ccNotCharged(t, aAssess(t, acSchema(t, "exists(n)"), acRoot(bad)), "exists(n) over an invalid n")
	res := aAssess(t, acSchema(t, "empty(n)"), acRoot(bad))
	if got := res.Unevaluated(); len(got) != 0 {
		t.Fatalf("empty(n): Unevaluated() = %v, want none", messages(got))
	}
	charged := false
	for _, v := range res.Violations() {
		charged = charged || v.Rule == ruleCvcAssertion && v.Loc == loc(1, 1)
	}
	if !charged {
		t.Errorf("empty(n): Violations() = %v, want a cvc-assertion at <root>", res.Violations())
	}
	acDeclined(t, aAssess(t, acSchema(t, "n = 1"), acRoot(bad)), "the child element n of the element root")
}
