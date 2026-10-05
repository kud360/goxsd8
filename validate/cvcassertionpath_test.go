package validate

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// The fixtures below drive an assertion {test} asking whether a relative path
// of child steps, `a/b`, selects a node of the element's subtree, which the walk
// reports to the test's xpath.Tally by each element's chain of names below the
// asserting element (walk.tallyElement).

// apSchema003 is Saxon's assert003.xsd: an <a> whose type is element-only, over
// an optional sequence holding a <b> of xs:anyType, and an untyped (so
// xs:anySimpleType) optional y.
const apSchema003 = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" elementFormDefault="qualified" attributeFormDefault="unqualified">
  <xs:element name="temp">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a">
          <xs:complexType>
            <xs:sequence minOccurs="0">
              <xs:element name="b"/>
            </xs:sequence>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
      <xs:attribute name="x" use="required"/>
      <xs:attribute name="y" use="optional"/>
      <xs:assert test="exists(@y) ne exists(a/b)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

// assert003's four instances: `exists(@y) ne exists(a/b)` is charged with both
// y and a/b (n1) and with neither (n2), and holds with exactly one (v1, v2) —
// at the ·validation root·, which is the assessed subtree's root the four
// valid cases need. Neither <a>, whose type is element-only and so has no value
// a {test} could read, nor <b> is read for its value
// (xpath.AssertionTest.ReadsChild), so no lackingChild declines the
// assertion: the satisfied rows record an Unevaluated instead with
// ctaChildPath.readsChild answering true. Every row is declined instead with
// xpath's child path declining.
func TestAssertionChildPathAssert003(t *testing.T) {
	schema := parsedSchema(t, map[string]string{"main.xsd": apSchema003})
	x, y := ccAttr(local("x"), "205", 1), ccAttr(local("y"), "204", 1)
	a, b := local("a"), local("b")
	temp := func(attrs []Attribute, kids ...Child) *testElement {
		return &testElement{name: local("temp"), attrs: attrs, kids: kids, loc: loc(1, 1)}
	}
	const charge = `the element temp is not ·valid· with respect to assertion 1 of 1 in the {assertions} of the ·governing type definition· `
	wantAssertionCharge(t, aAssess(t, schema, temp([]Attribute{x, y}, ccNode(a, 2, nil, ccNode(b, 3, nil)))), charge)
	wantAssertionCharge(t, aAssess(t, schema, temp([]Attribute{x}, ccNode(a, 2, nil))), charge)
	wantSatisfied(t, aAssess(t, schema, temp([]Attribute{x, y}, ccNode(a, 2, nil))), "v1: y and no a/b")
	wantSatisfied(t, aAssess(t, schema, temp([]Attribute{x}, ccNode(a, 2, nil, ccNode(b, 3, nil)))), "v2: a/b and no y")
}

// A ·nilled· element is a node of the instance (xpath-datamodel §6.2.4), and
// the walk reports it whatever its validity: a nilled <b> is the a/b
// `exists(a/b)` asks for, and an <a> that is nilled yet holds a <b> — invalid
// for that, and charged cvc-elt on its own — still has the b below it in the
// subtree, so the assertion holds over both.
func TestAssertionChildPathCountsNilledNodes(t *testing.T) {
	schema := parsedSchema(t, map[string]string{"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="AType">
    <xs:sequence><xs:element name="b" minOccurs="0" nillable="true"/></xs:sequence>
  </xs:complexType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence><xs:element name="a" type="AType" nillable="true"/></xs:sequence>
      <xs:assert test="exists(a/b)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`})
	nilled := func(line int) Attribute {
		return ccAttr(xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "nil"}, "true", line)
	}
	a, b := local("a"), local("b")
	root := func(kids ...Child) *testElement {
		return &testElement{name: local("root"), kids: kids, loc: loc(1, 1)}
	}
	wantSatisfied(t, aAssess(t, schema, root(ccNode(a, 2, nil, ccNode(b, 3, []Attribute{nilled(3)})))), "a nilled leaf")
	ccNotCharged(t, aAssess(t, schema, root(ccNode(a, 2, []Attribute{nilled(2)}, ccNode(b, 3, nil)))), "a nilled intermediate")
}

// Each counting element reads a child path from its own children down: <root>
// asks `exists(x/y)` and its child <x> `empty(y/z)`, so root/x/y/z holds at
// <root> and is charged at <x>. Siblings and their subtrees reuse one array of
// names: `exists(a/b) and exists(c/d)` holds over <a><b/></a><c><d/></c> and is
// charged over <a><d/></a><c><b/></c>, whose chains are a/d and c/b. The <x>
// row is satisfied instead with walk.tallyElement handing every counting
// ancestor the chain from the ·validation root·'s child.
func TestNestedChildPathsTakeTheirOwnChain(t *testing.T) {
	schema := parsedSchema(t, map[string]string{"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="XType">
    <xs:sequence><xs:element name="y" type="xs:anyType" minOccurs="0"/></xs:sequence>
    <xs:assert test="empty(y/z)"/>
  </xs:complexType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence><xs:element name="x" type="XType" minOccurs="0"/><xs:any namespace="##local" processContents="lax" minOccurs="0" maxOccurs="unbounded"/></xs:sequence>
      <xs:assert test="exists(x/y) or (exists(a/b) and exists(c/d))"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`})
	x, y, z := local("x"), local("y"), local("z")
	a, b, c, d := local("a"), local("b"), local("c"), local("d")
	root := func(kids ...Child) *testElement {
		return &testElement{name: local("root"), kids: kids, loc: loc(1, 1)}
	}
	wantSatisfied(t, aAssess(t, schema, root(ccNode(x, 2, nil, ccNode(y, 3, nil)))), "root/x/y without z")
	res := aAssess(t, schema, root(ccNode(x, 2, nil, ccNode(y, 3, nil, ccNode(z, 4, nil)))))
	if got := res.Unevaluated(); len(got) != 0 {
		t.Fatalf("root/x/y/z: Unevaluated() = %v, want none", messages(got))
	}
	if got := res.Violations(); len(got) != 1 || got[0].Rule != ruleCvcAssertion || got[0].Loc != loc(2, 3) {
		t.Errorf("root/x/y/z: Violations() = %v, want one cvc-assertion at %s, <x>'s", got, loc(2, 3))
	}
	wantSatisfied(t, aAssess(t, schema, root(ccNode(a, 2, nil, ccNode(b, 3, nil)), ccNode(c, 4, nil, ccNode(d, 5, nil)))), "a/b and c/d")
	wantAssertionCharge(t, aAssess(t, schema, root(ccNode(a, 2, nil, ccNode(d, 3, nil)), ccNode(c, 4, nil, ccNode(b, 5, nil)))),
		`the element root is not ·valid· with respect to assertion 1 of 1 in the {assertions} of the ·governing type definition· `)
}

// Saxon's assert020: a chameleon-<include>d document with
// xpathDefaultNamespace="##targetNamespace" asserts `empty(temp/temp/temp)`,
// whose unprefixed steps take the INCLUDER's namespace (§F.1), the namespace
// the instance's <temp>s are in. Three levels below the outer <temp> are none
// too many; four are, charged at the outer <temp> alone. The four-level row
// holds instead with the parser resolving ##targetNamespace from the raw
// targetNamespace attribute, which leaves the steps in no namespace.
func TestAssertionChildPathUnderAChameleonDefaultNamespace(t *testing.T) {
	const ns = "http://assert020.ns/"
	schema := parsedSchema(t, map[string]string{
		"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" elementFormDefault="qualified" targetNamespace="` + ns + `" xmlns:t="` + ns + `">
  <xs:include schemaLocation="lib.xsd"/>
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence><xs:element ref="t:temp" minOccurs="0" maxOccurs="unbounded"/></xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
		"lib.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" elementFormDefault="qualified" xpathDefaultNamespace="##targetNamespace">
  <xs:element name="temp">
    <xs:complexType>
      <xs:sequence><xs:element ref="temp" minOccurs="0" maxOccurs="unbounded"/></xs:sequence>
      <xs:assert test="empty(temp/temp/temp)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
	})
	temp := xsd.QName{Space: ns, Local: "temp"}
	nest := func(levels int) Child {
		var inner []Child
		for line := levels + 1; line >= 2; line-- {
			inner = []Child{ccNode(temp, line, nil, inner...)}
		}
		return inner[0]
	}
	doc := func(levels int) *testElement {
		return &testElement{name: xsd.QName{Space: ns, Local: "doc"}, kids: []Child{nest(levels)}, loc: loc(1, 1)}
	}
	wantSatisfied(t, aAssess(t, schema, doc(3)), "assert020.v1: three levels of temp")
	res := aAssess(t, schema, doc(4))
	if got := res.Unevaluated(); len(got) != 0 {
		t.Fatalf("assert020.n1: Unevaluated() = %v, want none", messages(got))
	}
	if got := res.Violations(); len(got) != 1 || got[0].Rule != ruleCvcAssertion || got[0].Loc != loc(2, 3) {
		t.Errorf("assert020.n1: Violations() = %v, want one cvc-assertion at %s, the outer temp's", got, loc(2, 3))
	}
}
