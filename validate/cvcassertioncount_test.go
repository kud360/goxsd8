package validate

import (
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// The fixtures below drive an assertion {test} that counts nodes of <e1>'s
// subtree with fn:count, which the walk reports to the test's xpath.Tally as
// it passes (walk.tallyElement). The validation root is an <e1> of RootType,
// whose content is
//
//	sequence( e1*, u?, any(urn:lax, lax)*, any(urn:skip, skip)* )
//
// each local <e1> of E1Type, which nests further e1s, carries an optional a and
// a d that defaults to "x"; <u> takes its type from an alternative whose
// {test} xpath declines, so its ·governing type definition· is undetermined.
// skip.xsd declares a global {urn:skip}y whose d defaults to "x" too, which
// the skip wildcard's ·skipped· y never takes.
const ccSchemaText = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:l="urn:lax" xmlns:s="urn:skip">
  <xs:import namespace="urn:skip" schemaLocation="skip.xsd"/>
  <xs:complexType name="E1Type">
    <xs:sequence>
      <xs:element name="e1" type="E1Type" minOccurs="0" maxOccurs="unbounded" nillable="true"/>
    </xs:sequence>
    <xs:attribute name="a" type="xs:string"/>
    <xs:attribute name="d" type="xs:string" default="x"/>
  </xs:complexType>
  <xs:complexType name="RootType">
    <xs:sequence>
      <xs:element name="e1" type="E1Type" minOccurs="0" maxOccurs="unbounded" nillable="true"/>
      <xs:element name="u" minOccurs="0">
        <xs:alternative test="string-length(@k) gt 0" type="xs:string"/>
      </xs:element>
      <xs:any namespace="urn:lax" processContents="lax" minOccurs="0" maxOccurs="unbounded"/>
      <xs:any namespace="urn:skip" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
    </xs:sequence>
    <xs:attribute name="a" type="xs:string"/>
    <xs:assert test="TEST"/>
  </xs:complexType>
  <xs:element name="e1" type="RootType"/>
</xs:schema>`

// ccSkipText is skip.xsd: the global {urn:skip}y, with a defaulted d.
const ccSkipText = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:skip">
  <xs:element name="y">
    <xs:complexType>
      <xs:sequence><xs:any processContents="lax" minOccurs="0" maxOccurs="unbounded"/></xs:sequence>
      <xs:attribute name="d" type="xs:string" default="x"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

// ccSchema is ccSchemaText with test as RootType's one assertion.
func ccSchema(t *testing.T, test string) *xsd.Schema {
	t.Helper()
	escaped := strings.NewReplacer("&", "&amp;", "<", "&lt;", `"`, "&quot;").Replace(test)
	return parsedSchema(t, map[string]string{"main.xsd": strings.Replace(ccSchemaText, "TEST", escaped, 1), "skip.xsd": ccSkipText})
}

// ccNode is an element named name at line, carrying attrs, over kids.
func ccNode(name xsd.QName, line int, attrs []Attribute, kids ...Child) Child {
	return ElementChild(&testElement{name: name, attrs: attrs, kids: kids, loc: loc(line, 3)})
}

// ccAttr is an attribute named name with value, on the element at line.
func ccAttr(name xsd.QName, value string, line int) Attribute {
	return &testAttribute{name: name, value: value, loc: loc(line, 6)}
}

// ccRoot is the validation root <e1> at 1:1, carrying attrs, over kids.
func ccRoot(attrs []Attribute, kids ...Child) *testElement {
	return &testElement{name: local("e1"), attrs: attrs, kids: kids, loc: loc(1, 1)}
}

// ccNotCharged fails if res charged cvc-assertion or recorded any Unevaluated:
// the assertion was evaluated and holds, whatever else res charged.
func ccNotCharged(t *testing.T, res *Result, why string) {
	t.Helper()
	if got := res.Unevaluated(); len(got) != 0 {
		t.Errorf("%s: Unevaluated() = %v, want none", why, messages(got))
	}
	for _, v := range res.Violations() {
		if v.Rule == ruleCvcAssertion {
			t.Errorf("%s: Violations() holds %v, want the assertion satisfied", why, v)
		}
	}
}

// ccDeclinedAmong fails unless res recorded, among its Unevaluated, a
// cvc-assertion decline at the root naming want, and charged no cvc-assertion.
func ccDeclinedAmong(t *testing.T, res *Result, want string) {
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

// fn:count counts the nodes of <e1>'s subtree the walk reports, in the data
// model instance cvc-assertion clause 1 builds. `count(e1) eq 1` counts the
// child and not the grandchild; `count(.//e1) eq 2` counts both and not the
// root <e1> itself, and is charged over a third. `count(.//@a) eq 2` counts the
// root's own @a with a grandchild's, and is charged without the root's. Each
// E1Type element's d is ·defaulted· (key-dflt-att) and counted as carried, so
// `count(.//@d) eq 2` holds over two of them, one carrying d or neither; the
// two rows are charged instead with walk.attributeNodes adding no ·defaulted
// attribute·. An xsi:nil attribute is a node of the instance and its ·nilled·
// element is walked. Every row is declined instead, and fails, with xpath's
// fn:count declining.
func TestAssertionCountsTheWalkedSubtree(t *testing.T) {
	e1, a, d := local("e1"), local("a"), local("d")
	nilled := ccAttr(xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "nil"}, "true", 2)
	for _, tc := range []struct {
		why, test string
		root      *testElement
		charged   bool
	}{
		{"a child, not a grandchild", "count(e1) eq 1",
			ccRoot(nil, ccNode(e1, 2, nil, ccNode(e1, 3, nil))), false},
		{"a child and a grandchild, not the root", "count(.//e1) eq 2",
			ccRoot(nil, ccNode(e1, 2, nil, ccNode(e1, 3, nil))), false},
		{"three descendants", "count(.//e1) eq 2",
			ccRoot(nil, ccNode(e1, 2, nil, ccNode(e1, 3, nil)), ccNode(e1, 4, nil)), true},
		{"the root's own @a and a grandchild's", "count(.//@a) eq 2",
			ccRoot([]Attribute{ccAttr(a, "r", 1)}, ccNode(e1, 2, nil, ccNode(e1, 3, []Attribute{ccAttr(a, "g", 3)}))), false},
		{"a grandchild's @a alone", "count(.//@a) eq 2",
			ccRoot(nil, ccNode(e1, 2, nil, ccNode(e1, 3, []Attribute{ccAttr(a, "g", 3)}))), true},
		{"the root's own @a, not a child's", "count(@a) eq 1",
			ccRoot([]Attribute{ccAttr(a, "r", 1)}, ccNode(e1, 2, []Attribute{ccAttr(a, "c", 2)})), false},
		{"two ·defaulted· d", "count(.//@d) eq 2",
			ccRoot(nil, ccNode(e1, 2, nil, ccNode(e1, 3, nil))), false},
		{"one carried d, one ·defaulted·", "count(.//@d) eq 2",
			ccRoot(nil, ccNode(e1, 2, []Attribute{ccAttr(d, "y", 2)}, ccNode(e1, 3, nil))), false},
		{"an xsi:nil on a ·nilled· child", "count(.//@xsi:nil) eq 1 and count(e1) eq 1",
			ccRoot(nil, ccNode(e1, 2, []Attribute{nilled})), false},
	} {
		t.Run(tc.why, func(t *testing.T) {
			res := aAssess(t, ccSchema(t, tc.test), tc.root)
			if !tc.charged {
				wantSatisfied(t, res, tc.test)
				return
			}
			wantAssertionCharge(t, res, "the element e1 is not ·valid· with respect to assertion 1 of 1 in the {assertions} of the ·governing type definition· RootType, whose {test} is ")
		})
	}
}

// Every element the walk enters is counted whatever its assessment: an <e1>
// invalid for an undeclared attribute, which its own charge reports, and a
// ·laxly assessed· {urn:lax}x and the lax x below it, whose @a counts too —
// no ·governing type definition· means no ·defaulted attribute· to miss. The
// lax row is declined instead, and fails, with walk.attributeNodes treating a
// ·laxly assessed· element as undetermined.
func TestAssertionCountsInvalidAndLaxElements(t *testing.T) {
	e1, x := local("e1"), xsd.QName{Space: "urn:lax", Local: "x"}
	stray := ccAttr(local("stray"), "1", 2)
	res := aAssess(t, ccSchema(t, "count(e1) eq 2"), ccRoot(nil, ccNode(e1, 2, []Attribute{stray}), ccNode(e1, 3, nil)))
	ccNotCharged(t, res, "an invalid child")
	if len(res.Violations()) == 0 {
		t.Error("Violations() is empty, want the stray attribute charged")
	}
	res = aAssess(t, ccSchema(t, "count(.//l:x) eq 2 and count(.//@a) eq 1"),
		ccRoot(nil, ccNode(x, 2, []Attribute{ccAttr(local("a"), "v", 2)}, ccNode(x, 3, nil))))
	wantSatisfied(t, res, "lax elements")
}

// A ·skipped· subtree is counted by name (assertionAncestry.tallySkipped): a
// {urn:skip}y the skip wildcard ·skipped· and every element below it, each at
// its own depth, so `count(e1)` does not count the e1 inside y, with the
// attributes each carries — xsi:type among them, never assessed, so its bogus
// QName charges nothing — and no ·defaulted attribute·, so y's declared d,
// which ·skipped· y is not governed by, counts only where carried; `count(s:y)`
// counts a ·skipped· y child and not the y inside it. Every table row is
// declined instead, and fails, with assertionAncestry.skipped recording a
// lackingCount for each counting ancestor; the first two charged rows are
// satisfied instead with tallySkipped reporting nothing below y. With
// tallySkipped counting every element one level deeper than it stands, the
// y-child row is charged; one level shallower, its charged twin is satisfied.
func TestAssertionCountsAcrossASkippedElement(t *testing.T) {
	e1, y, a, d := local("e1"), xsd.QName{Space: "urn:skip", Local: "y"}, local("a"), local("d")
	xsiType := ccAttr(xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "type"}, "no:such", 4)
	for _, tc := range []struct {
		why, test string
		root      *testElement
		charged   bool
	}{
		{"an e1 inside y and one outside", "count(.//e1) eq 2",
			ccRoot(nil, ccNode(e1, 2, nil), ccNode(y, 3, nil, ccNode(e1, 4, nil))), false},
		{"three e1, two outside y", "count(.//e1) eq 2",
			ccRoot(nil, ccNode(e1, 2, nil), ccNode(e1, 3, nil), ccNode(y, 4, nil, ccNode(e1, 5, nil))), true},
		{"y itself and a y inside it", "count(.//s:y) eq 2",
			ccRoot(nil, ccNode(y, 2, nil, ccNode(y, 3, nil))), false},
		{"the e1 inside y is no child", "count(e1) eq 1",
			ccRoot(nil, ccNode(e1, 2, nil), ccNode(y, 3, nil, ccNode(e1, 4, nil))), false},
		{"a y child, not the y inside it", "count(s:y) eq 1",
			ccRoot(nil, ccNode(y, 2, nil, ccNode(y, 3, nil))), false},
		{"two y children, not the y inside one", "count(s:y) eq 1",
			ccRoot(nil, ccNode(y, 2, nil), ccNode(y, 3, nil, ccNode(y, 4, nil))), true},
		{"y's own @a and its child's", "count(.//@a) eq 2",
			ccRoot(nil, ccNode(y, 2, []Attribute{ccAttr(a, "1", 2)}, ccNode(e1, 3, []Attribute{ccAttr(a, "2", 3)}))), false},
		{"y's own @a and its child's, one too many", "count(.//@a) eq 1",
			ccRoot(nil, ccNode(y, 2, []Attribute{ccAttr(a, "1", 2)}, ccNode(e1, 3, []Attribute{ccAttr(a, "2", 3)}))), true},
		{"an xsi:type inside y", "count(.//@xsi:type) eq 1",
			ccRoot(nil, ccNode(y, 2, nil, ccNode(y, 3, nil, ccNode(e1, 4, []Attribute{xsiType})))), false},
		{"a carried d and no ·defaulted· one", "count(.//@d) eq 1",
			ccRoot(nil, ccNode(y, 2, []Attribute{ccAttr(d, "c", 2)}, ccNode(y, 3, nil))), false},
	} {
		t.Run(tc.why, func(t *testing.T) {
			res := aAssess(t, ccSchema(t, tc.test), tc.root)
			if !tc.charged {
				wantSatisfied(t, res, tc.test)
				return
			}
			wantAssertionCharge(t, res, "the element e1 is not ·valid· with respect to assertion 1 of 1 in the {assertions} of the ·governing type definition· RootType, whose {test} is ")
		})
	}
	wantSatisfied(t, aAssess(t, ccSchema(t, "@a = 'v'"), ccRoot([]Attribute{ccAttr(a, "v", 1)}, ccNode(y, 2, nil))),
		"a non-counting assertion beside a ·skipped· child")
}

// A ·skipped· child whose value a {test} reads still DECLINES its parent's
// assertions through lackingChild, whatever else the same {test} counts
// (guard): the ·skipped· <e1> is read by name for `count(.//x)`, and its value
// is not read.
func TestAssertionReadingASkippedCountedChildIsDeclined(t *testing.T) {
	acDeclined(t, aAssess(t, acSchema(t, "e1 = 'present' and count(.//x) ge 0"), acRoot(acKid("e1", 2, "present"), acKid("e1", 3, "present"))),
		"the child element e1 of the element root at instance.xml:3:3, which a {test} reads, has no typed value for the data model instance cvc-assertion clause 1 builds: it is ·skipped·")
}

// A fault in the source inside a ·skipped· subtree stops the walk where an
// ancestor counts across it, since the subtree is read for the Tally: Err()
// wraps the fault with the element whose [[children]] were being read, two
// levels below the skip, and nothing is decided at the root. It is Err() nil
// instead, with tallySkipped dropping its cursor's Err or a child call's
// error. Where no ancestor counts the subtree is never opened, and a fault a
// cursor over it would report goes unread (guard).
func TestSourceFaultInASkippedCountedSubtreeStopsTheWalk(t *testing.T) {
	y := xsd.QName{Space: "urn:skip", Local: "y"}
	fault := errors.New("truncated")
	tree := func() *testElement {
		inner := &testElement{name: local("z"), kidsErr: fault, loc: loc(4, 3)}
		return ccRoot([]Attribute{ccAttr(local("a"), "v", 1)}, ccNode(y, 2, nil, ccNode(y, 3, nil, ElementChild(inner))))
	}
	v, err := New(ccSchema(t, "count(.//s:y) eq 2"), testBackend())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	res := v.Assess(tree())
	if !errors.Is(res.Err(), fault) {
		t.Fatalf("Err() = %v, want it to wrap %v", res.Err(), fault)
	}
	if want := "reading the children of z at instance.xml:4:3: "; !strings.HasPrefix(res.Err().Error(), want) {
		t.Errorf("Err() = %q, want it to open %q", res.Err(), want)
	}
	if len(res.Violations()) != 0 || len(res.Unevaluated()) != 0 {
		t.Errorf("Violations() = %v, Unevaluated() = %v, want both empty: the walk stopped", res.Violations(), messages(res.Unevaluated()))
	}
	wantSatisfied(t, aAssess(t, ccSchema(t, "@a = 'v'"), tree()), "a non-counting assertion over a faulting ·skipped· subtree")
}

// A ·skipped· subtree below a counting element that is not the ·validation
// root· is counted at its depth below THAT element: <inner> counts the y child
// a skip wildcard ·skipped· and not the y inside it, while <root> above it
// counts both. Two y children of <inner> are charged at <inner> alone. The
// satisfied row is charged at <inner> instead with tallySkipped measuring
// depth from the ·validation root· rather than from each counting ancestor.
func TestNestedCountsAcrossASkippedElement(t *testing.T) {
	schema := parsedSchema(t, map[string]string{"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:s="urn:skip">
  <xs:complexType name="IType">
    <xs:sequence><xs:any namespace="urn:skip" processContents="skip" minOccurs="0" maxOccurs="unbounded"/></xs:sequence>
    <xs:assert test="count(s:y) eq 1"/>
  </xs:complexType>
  <xs:complexType name="RType">
    <xs:sequence><xs:element name="inner" type="IType"/></xs:sequence>
    <xs:assert test="count(.//s:y) eq 2"/>
  </xs:complexType>
  <xs:element name="root" type="RType"/>
</xs:schema>`})
	y, inner := xsd.QName{Space: "urn:skip", Local: "y"}, local("inner")
	root := func(kids ...Child) *testElement {
		return &testElement{name: local("root"), kids: []Child{ccNode(inner, 2, nil, kids...)}, loc: loc(1, 1)}
	}
	wantSatisfied(t, aAssess(t, schema, root(ccNode(y, 3, nil, ccNode(y, 4, nil)))), "a y child of inner and the y inside it")
	res := aAssess(t, schema, root(ccNode(y, 3, nil), ccNode(y, 4, nil)))
	if got := res.Unevaluated(); len(got) != 0 {
		t.Fatalf("Unevaluated() = %v, want none", messages(got))
	}
	got := res.Violations()
	if len(got) != 1 || got[0].Rule != ruleCvcAssertion || got[0].Loc != loc(2, 3) {
		t.Fatalf("Violations() = %v, want one cvc-assertion charge at %s", got, loc(2, 3))
	}
	if want := `the element inner is not ·valid· with respect to assertion 1 of 1 in the {assertions} of the ·governing type definition· IType, whose {test} is "count(s:y) eq 1",`; !strings.HasPrefix(got[0].Msg, want) {
		t.Errorf("Msg = %q, want it to open %q", got[0].Msg, want)
	}
}

// ccDecided fails unless res recorded no cvc-assertion decline, and charged
// cvc-assertion at the root exactly where charged: the assertion was
// evaluated, whatever else res declined.
func ccDecided(t *testing.T, res *Result, charged bool, why string) {
	t.Helper()
	for _, u := range res.Unevaluated() {
		if u.Rule() == ruleCvcAssertion {
			t.Errorf("%s: Unevaluated() holds %q, want the assertion evaluated", why, u.Msg())
		}
	}
	got := false
	for _, v := range res.Violations() {
		got = got || v.Rule == ruleCvcAssertion && v.Loc == loc(1, 1)
	}
	if got != charged {
		t.Errorf("%s: charged = %v, want %v (Violations() = %v)", why, got, charged, res.Violations())
	}
}

// An element whose ·governing type definition· is undetermined — <u>, whose
// alternative's {test} xpath declines, and the <e1> below it, walked against
// nothing — has ·defaulted attributes· the walk cannot know, but its element
// node is counted wherever no {test} counts attribute nodes at its depth: the
// element counts are decided, and so is `count(@a)`, which selects the root's
// attributes alone. Every decided row is declined instead, and fails, with
// walk.tallyElement declining an undetermined element whatever the Tally
// counts; with its element node unreported, the charged row is satisfied
// instead and the first two are charged. `count(.//@a)` selects <u>'s
// attributes, so it is declined still (guard).
func TestAssertionCountsAcrossAnUndeterminedElement(t *testing.T) {
	e1, u, a := local("e1"), local("u"), local("a")
	k := ccAttr(local("k"), "1", 2)
	for _, tc := range []struct {
		why, test string
		root      *testElement
		charged   bool
	}{
		{"u among the descendants", "count(.//u) eq 1 and count(.//e1) eq 1",
			ccRoot(nil, ccNode(e1, 2, nil), ccNode(u, 3, []Attribute{k})), false},
		{"an e1 below u", "count(.//e1) eq 2",
			ccRoot(nil, ccNode(e1, 2, nil), ccNode(u, 3, []Attribute{k}, ccNode(e1, 4, nil))), false},
		{"an e1 below u, counted", "count(.//e1) eq 1",
			ccRoot(nil, ccNode(e1, 2, nil), ccNode(u, 3, []Attribute{k}, ccNode(e1, 4, nil))), true},
		{"the root's own @a beside u", "count(@a) eq 1",
			ccRoot([]Attribute{ccAttr(a, "r", 1)}, ccNode(u, 2, []Attribute{k, ccAttr(a, "u", 2)})), false},
	} {
		t.Run(tc.why, func(t *testing.T) {
			ccDecided(t, aAssess(t, ccSchema(t, tc.test), tc.root), tc.charged, tc.test)
		})
	}
	ccDeclinedAmong(t, aAssess(t, ccSchema(t, "count(.//@a) eq 0"), ccRoot(nil, ccNode(u, 2, []Attribute{k}))),
		"the element u at instance.xml:2:3, in the subtree of the element e1 whose nodes a {test} counts, cannot be counted exactly for the data model instance cvc-assertion clause 1 builds: its ·governing type definition· was not determined")
}

// An element below another counting one is counted by both, each at its own
// depth: <root> counts three b below it and one child b, <inner> two below it
// and one child b. A fourth b is charged at <root> alone. Every row is charged
// instead with walk.tallyElement passing the depth below the ·validation root·
// rather than below each counting ancestor.
func TestNestedCountsTakeTheirOwnDepth(t *testing.T) {
	schema := parsedSchema(t, map[string]string{"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="CType">
    <xs:sequence><xs:element name="b" minOccurs="0" maxOccurs="unbounded"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="IType">
    <xs:sequence>
      <xs:element name="b" minOccurs="0" maxOccurs="unbounded"/>
      <xs:element name="c" type="CType" minOccurs="0"/>
    </xs:sequence>
    <xs:assert test="count(.//b) eq 2 and count(b) eq 1"/>
  </xs:complexType>
  <xs:complexType name="RType">
    <xs:sequence>
      <xs:element name="inner" type="IType"/>
      <xs:element name="b" minOccurs="0" maxOccurs="unbounded"/>
    </xs:sequence>
    <xs:assert test="count(.//b) eq 3 and count(b) eq 1"/>
  </xs:complexType>
  <xs:element name="root" type="RType"/>
</xs:schema>`})
	b, c, inner := local("b"), local("c"), local("inner")
	tree := func(extra ...Child) *testElement {
		kids := append([]Child{ccNode(inner, 2, nil, ccNode(b, 3, nil), ccNode(c, 4, nil, ccNode(b, 5, nil))), ccNode(b, 6, nil)}, extra...)
		return &testElement{name: local("root"), kids: kids, loc: loc(1, 1)}
	}
	wantSatisfied(t, aAssess(t, schema, tree()), "three b below root, two below inner")
	wantAssertionCharge(t, aAssess(t, schema, tree(ccNode(b, 7, nil))),
		`the element root is not ·valid· with respect to assertion 1 of 1 in the {assertions} of the ·governing type definition· RType, whose {test} is "count(.//b) eq 3 and count(b) eq 1",`)
}

// A child step filtered by attribute existence counts the children whose
// attribute nodes, carried or ·defaulted·, include each name (xpath20.md
// §3.2.2): every E1Type <e1> has a ·defaulted· d, so `count(e1[@d]) eq 2`
// holds over two <e1> carrying none, and `count(e1[@a and @d]) eq 1` over one
// carrying an EMPTY a — the predicate asks for the node, never its value — and
// one carrying none; a grandchild <e1> carrying a is no child. A ·skipped· y
// is counted by the attributes it carries and no ·defaulted· one: `s:y[@a and
// @b]` counts a y carrying both and not one carrying a alone, and `s:y[@d]`
// none, its declared default never taken. A union counts each node once. The
// first three rows are charged instead with walk.attributeNodes adding no
// ·defaulted attribute·; the "a and b" row with assertionAncestry.tallySkipped
// reporting no attribute names; the union row with the Tally summing a union's
// operands.
func TestAssertionCountsChildrenFilteredByAttributes(t *testing.T) {
	e1, y, a, b, d := local("e1"), xsd.QName{Space: "urn:skip", Local: "y"}, local("a"), local("b"), local("d")
	for _, tc := range []struct {
		why, test string
		root      *testElement
		charged   bool
	}{
		{"two ·defaulted· d", "count(e1[@d]) eq 2",
			ccRoot(nil, ccNode(e1, 2, nil), ccNode(e1, 3, nil)), false},
		{"an empty a and a ·defaulted· d", "count(e1[@a and @d]) eq 1",
			ccRoot(nil, ccNode(e1, 2, []Attribute{ccAttr(a, "", 2)}), ccNode(e1, 3, nil)), false},
		{"a carried d too", "count(e1[@a and @d]) eq 2",
			ccRoot(nil, ccNode(e1, 2, []Attribute{ccAttr(a, "", 2)}), ccNode(e1, 3, []Attribute{ccAttr(a, "1", 3), ccAttr(d, "y", 3)})), false},
		{"a grandchild is no child", "count(e1[@a]) eq 1",
			ccRoot(nil, ccNode(e1, 2, []Attribute{ccAttr(a, "1", 2)}, ccNode(e1, 3, []Attribute{ccAttr(a, "2", 3)}))), false},
		{"a ·skipped· y carrying a and b", "count(s:y[@a and @b]) eq 1",
			ccRoot(nil, ccNode(y, 2, []Attribute{ccAttr(a, "1", 2), ccAttr(b, "1", 2)}, ccNode(y, 3, []Attribute{ccAttr(a, "1", 3)}))), false},
		{"a ·skipped· y carrying a alone", "count(s:y[@a and @b]) eq 1",
			ccRoot(nil, ccNode(y, 2, []Attribute{ccAttr(a, "1", 2)}, ccNode(y, 3, []Attribute{ccAttr(b, "1", 3)}))), true},
		{"a ·skipped· y has no ·defaulted· d", "count(s:y[@d]) eq 0",
			ccRoot(nil, ccNode(y, 2, nil)), false},
		{"a union counts each node once", "count(@a | e1 | e1[@a]) eq 3",
			ccRoot([]Attribute{ccAttr(a, "r", 1)}, ccNode(e1, 2, []Attribute{ccAttr(a, "1", 2)}), ccNode(e1, 3, nil)), false},
	} {
		t.Run(tc.why, func(t *testing.T) {
			res := aAssess(t, ccSchema(t, tc.test), tc.root)
			if !tc.charged {
				wantSatisfied(t, res, tc.test)
				return
			}
			wantAssertionCharge(t, res, "the element e1 is not ·valid· with respect to assertion 1 of 1 in the {assertions} of the ·governing type definition· RootType, whose {test} is ")
		})
	}
}

// An element whose ·governing type definition· is undetermined has ·defaulted
// attributes· the walk cannot know, so a filtered child step naming it, which
// reads the attribute names of E's children, DECLINES — `count(u[@k])` — where
// the bare step still counts it: `count(u) eq 1` is decided. The decline row
// is decided instead, and charged, with ctaFilteredChildren.selectsAttributesAt
// false at depth 1.
func TestAssertionFilteredCountOverAnUndeterminedChild(t *testing.T) {
	u, k := local("u"), ccAttr(local("k"), "1", 2)
	ccDeclinedAmong(t, aAssess(t, ccSchema(t, "count(u[@k]) eq 1"), ccRoot(nil, ccNode(u, 2, []Attribute{k}))),
		"the element u at instance.xml:2:3, in the subtree of the element e1 whose nodes a {test} counts, cannot be counted exactly for the data model instance cvc-assertion clause 1 builds: its ·governing type definition· was not determined")
	ccDecided(t, aAssess(t, ccSchema(t, "count(u) eq 1"), ccRoot(nil, ccNode(u, 2, []Attribute{k}))), false, "count(u) eq 1")
}
