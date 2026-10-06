package xpath

import (
	"testing"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// The fixtures below drive the assertion façade's fn:count call
// (xpath-functions.md §15.4.1): an assertion counts nodes of the subtree of E,
// the root of the instance cvc-assertion clause 1.3 builds, through the [Tally]
// its caller fills, and never reads a counted node's value.

// acNode is one node the caller reports to a Tally: an element node, or an
// attribute node where attribute is true, named name and depth levels below E.
// An element node is reported by its chain below E: path where it is non-nil,
// and otherwise depth-1 ancestors named acFiller above name, the empty chain —
// E itself — at a depth below 1; with attrs, the names of its attribute nodes.
// An attribute node is reported alone on an element whose chain is depth
// acFiller names, the empty chain — E itself — at depth 0.
type acNode struct {
	attribute bool
	depth     int
	name      xsd.QName
	path      []xsd.QName
	attrs     []xsd.QName
}

// acFiller names the ancestors acTally invents for an element reported by
// depth, a name no counted path below writes.
var acFiller = uq("anc")

// chain is the element chain n is reported by.
func (n acNode) chain() []xsd.QName {
	if n.path != nil {
		return n.path
	}
	var path []xsd.QName
	if n.attribute {
		for range n.depth {
			path = append(path, acFiller)
		}
		return path
	}
	for range n.depth - 1 {
		path = append(path, acFiller)
	}
	if n.depth >= 1 {
		path = append(path, n.name)
	}
	return path
}

// reported is the attribute names n is reported with.
func (n acNode) reported() []xsd.QName {
	if n.attribute {
		return []xsd.QName{n.name}
	}
	return n.attrs
}

// acPath is an element node reported by the chain of locals below E, each in no
// namespace.
func acPath(locals ...string) acNode {
	path := []xsd.QName{}
	for _, l := range locals {
		path = append(path, uq(l))
	}
	return acNode{path: path}
}

// acEl is an element node named local in no namespace, depth levels below E.
func acEl(depth int, local string) acNode { return acNode{depth: depth, name: uq(local)} }

// acAt is an attribute node named local in no namespace, belonging to the
// element depth levels below E.
func acAt(depth int, local string) acNode {
	return acNode{attribute: true, depth: depth, name: uq(local)}
}

// acTally is test's Tally with nodes reported to it, in the order written.
func acTally(test AssertionTest, nodes ...acNode) *Tally {
	c := test.Tally()
	for _, n := range nodes {
		c.Element(n.chain(), n.reported())
	}
	return c
}

// acCompile compiles expr for an E with element-only content whose attribute
// length is an xs:int and which declares no child element at all: a counted
// step consults neither AttributeTypes nor ElementTypes, so asNoElems types no
// name it counts.
func acCompile(t *testing.T, record xsd.XPathExpression) AssertionTest {
	t.Helper()
	test, ok := CompileAssertionTest(record, seededTypes, xsd.ElementContent{}, asUses(t, map[string]string{"length": "int"}), asNoElems)
	if !ok {
		t.Fatalf("CompileAssertionTest(%q): declined, want compiled", record.Expression())
	}
	return test
}

// fn:count counts the nodes its path selects in E's subtree (xpath20.md
// §3.2.4): `N` and `./N` the element children, `.//N` every element below E
// and never E itself, `@N` and `./@N` E's own attributes, and `.//@N` the
// attributes of E and of every element below it. `@length eq count(./ele)` is
// satisfied where length counts the children and charged where it does not —
// the satisfied row tells a count from a decline read as false. `.//e1` over an
// E itself named e1 counts its e1 child and grandchild and not E (reported at
// depth 0, which an element report selects nothing at). `.//@a` counts E's own
// @a with a grandchild's, so the count is 2. A name a counted step selects is
// typed nowhere: asNoElems declines every child step, and `zz` counts all the
// same. Every row declines at CompileAssertionTest, and fails, with fn:count
// declining (ctaParser.valueExpr handing it to constructorFunction).
func TestAssertionCountsSubtreeNodes(t *testing.T) {
	length := func(lexical string) TypedAttributes {
		return asValues(t, asTyped{uq("length"), "int", lexical})
	}
	for _, tc := range []struct {
		why    string
		expr   string
		attrs  TypedAttributes
		nodes  []acNode
		holds  bool
		record xsd.XPathExpression
	}{
		{why: "length counts the children", expr: "@length eq count(./ele)", attrs: length("2"),
			nodes: []acNode{acEl(1, "ele"), acEl(1, "ele")}, holds: true},
		{why: "length does not count the children", expr: "@length eq count(./ele)", attrs: length("3"),
			nodes: []acNode{acEl(1, "ele"), acEl(1, "ele")}},
		{why: "a grandchild is a descendant, E is not", expr: "count(.//e1) eq 2",
			nodes: []acNode{acEl(0, "e1"), acEl(1, "e1"), acEl(2, "e1")}, holds: true},
		{why: "E named e1 is not its own descendant", expr: "count(.//e1) eq 2",
			nodes: []acNode{acEl(0, "e1"), acEl(1, "e1")}},
		{why: "E's own attribute is in .//@a", expr: "count(.//@a) eq 2",
			nodes: []acNode{acAt(0, "a"), acAt(2, "a")}, holds: true},
		{why: "a grandchild's attribute alone is one", expr: "count(.//@a) eq 2",
			nodes: []acNode{acAt(2, "a")}},
		{why: "a child step counts no grandchild", expr: "count(e1) eq 1",
			nodes: []acNode{acEl(1, "e1"), acEl(2, "e1")}, holds: true},
		{why: "an attribute step counts E's own alone", expr: "count(@a) eq 1",
			nodes: []acNode{acAt(0, "a"), acAt(1, "a")}, holds: true},
		{why: "./@a is @a", expr: "count(./@a) eq 1",
			nodes: []acNode{acAt(0, "a"), acAt(1, "a")}, holds: true},
		{why: "an attribute named e1 is no element", expr: "count(e1) eq 0",
			nodes: []acNode{acAt(1, "e1")}, holds: true},
		{why: "an element named a is no attribute", expr: "count(.//@a) eq 0",
			nodes: []acNode{acEl(1, "a")}, holds: true},
		{why: "two paths, two counters", expr: "count(e1) eq 1 and count(.//e1) eq 2",
			nodes: []acNode{acEl(1, "e1"), acEl(2, "e1")}, holds: true},
		{why: "one path written twice, one counter", expr: "count(e1) eq 1 and count(./e1) = 1",
			nodes: []acNode{acEl(1, "e1")}, holds: true},
		{why: "an untyped name counts", expr: "count(zz) eq 1",
			nodes: []acNode{acEl(1, "zz")}, holds: true},
		{why: "a general comparison", expr: "count(e1) > 1",
			nodes: []acNode{acEl(1, "e1"), acEl(1, "e1")}, holds: true},
		{why: "a count's effective boolean value, nonzero", expr: "count(e1)",
			nodes: []acNode{acEl(1, "e1")}, holds: true},
		{why: "a count's effective boolean value, zero", expr: "count(e1)"},
		{why: "fn:not over a count", expr: "not(count(e1) eq 1)", holds: true},
		{why: "a present default namespace names the qualified child",
			record: ctaExprRecord("count(e1) eq 1", "urn:t"),
			nodes:  []acNode{{depth: 1, name: xsd.QName{Space: "urn:t", Local: "e1"}}}, holds: true},
		{why: "a present default namespace misses the unqualified child",
			record: ctaExprRecord("count(e1) eq 1", "urn:t"), nodes: []acNode{acEl(1, "e1")}},
		{why: "an unprefixed attribute step is in no namespace whatever the default",
			record: ctaExprRecord("count(@a) eq 1", "urn:t"), nodes: []acNode{acAt(0, "a")}, holds: true},
		{why: "a prefix names its binding",
			record: ctaExprRecord("count(.//t:e1) eq 1", "", "t", "urn:t"),
			nodes:  []acNode{{depth: 3, name: xsd.QName{Space: "urn:t", Local: "e1"}}}, holds: true},
	} {
		t.Run(tc.why, func(t *testing.T) {
			record := tc.record
			if tc.expr != "" {
				record = asRecord(tc.expr)
			}
			attrs := tc.attrs
			if attrs == nil {
				attrs = asValues(t)
			}
			test := acCompile(t, record)
			if got := test.Evaluate(backend(), seededTypes, attrs, asNoChildren, acTally(test, tc.nodes...), ValueBinding{}); got != tc.holds {
				t.Errorf("Evaluate(%q) over %v = %v, want %v", record.Expression(), tc.nodes, got, tc.holds)
			}
		})
	}
}

// A rooted fn:count argument raises err:XPDY0050 (xpath20.md §3.2): E has no
// document node above it, so `count(//e1) eq 2` is false over an E with
// exactly two e1 descendants, `count(//@a) eq 1` over one @a, and their fn:not
// too, the error propagating. Such a test counts nothing, so its Tally is nil.
// A rooted attribute step is admitted outside fn:count as well and raises the
// same. Every row declines, and fails, with fn:count declining; the `//@a`
// rows also with ctaParser.rootedPath admitting no attribute step.
func TestAssertionRootedCountRaises(t *testing.T) {
	attrs := asValues(t, asTyped{uq("a"), "string", "a"})
	children := asChildren(t, asChild{uq("e1"), "string", "present", false}, asChild{uq("e1"), "string", "present", false})
	for _, expr := range []string{
		"count(//e1) eq 2", "count(/e1) eq 1", "not(count(//e1) eq 2)", "count(//e1)",
		"count(//@a) eq 1", "count(/@a) eq 1", "not(count(//@a) eq 1)",
		"/@a = 'a'", "//@a", "not(//@a)", "//@a eq 'a'",
	} {
		test, ok := CompileAssertionTest(asRecord(expr), seededTypes, xsd.ElementContent{}, asUses(t, map[string]string{"a": "string"}), asChildTypes(t))
		if !ok {
			t.Errorf("CompileAssertionTest(%q): declined, want compiled", expr)
			continue
		}
		if c := test.Tally(); c != nil {
			t.Errorf("(%q).Tally() = %v, want nil: a rooted path counts nothing", expr, c)
		}
		if test.Evaluate(backend(), seededTypes, attrs, children, nil, ValueBinding{}) {
			t.Errorf("Evaluate(%q) = true, want false: the leading slash raises err:XPDY0050", expr)
		}
	}
}

// Evaluate answers false for a Tally the test did not make, whatever the tree
// holds: `count(e1) eq 0` is satisfied with its own empty Tally and false with
// none, with one counting another path, and with one counting a further path;
// `'a' = 'a'`, which counts nothing, is satisfied with no Tally and false with
// one, a zero-value Tally included. The last three false rows are satisfied
// instead with Tally.fits' check removed from Evaluate, and the zero-value row
// with only fits' empty-paths guard removed; the other two stay false without
// the check, the count node finding no counter for its path (ctaCountItem).
func TestAssertionEvaluateRefusesAMismatchedTally(t *testing.T) {
	counting := acCompile(t, asRecord("count(e1) eq 0"))
	other := acCompile(t, asRecord("count(e2) eq 0"))
	wider := acCompile(t, asRecord("count(e1) eq 0 and count(e2) eq 0"))
	plain := acCompile(t, asRecord("'a' = 'a'"))
	for _, tc := range []struct {
		why    string
		test   AssertionTest
		counts *Tally
		holds  bool
	}{
		{"its own Tally", counting, counting.Tally(), true},
		{"no Tally where it counts", counting, nil, false},
		{"a Tally counting another path", counting, other.Tally(), false},
		{"a Tally counting a further path", counting, wider.Tally(), false},
		{"no Tally where it counts nothing", plain, nil, true},
		{"a Tally where it counts nothing", plain, counting.Tally(), false},
		{"a zero-value Tally where it counts nothing", plain, &Tally{}, false},
	} {
		if got := tc.test.Evaluate(backend(), seededTypes, asValues(t), asNoChildren, tc.counts, ValueBinding{}); got != tc.holds {
			t.Errorf("%s: Evaluate = %v, want %v", tc.why, got, tc.holds)
		}
	}
}

// AssertionTest.Tally is nil for a test that counts nothing and fresh on every
// call otherwise, so filling one Tally leaves the next untouched and the
// compiled test immutable. Tally's methods are total: an empty element chain
// selects no element, and a nil Tally takes any report.
func TestAssertionTallyIsFreshAndTotal(t *testing.T) {
	if c := acCompile(t, asRecord("@length eq 1")).Tally(); c != nil {
		t.Errorf("Tally() of a test with no fn:count = %v, want nil", c)
	}
	test := acCompile(t, asRecord("count(.//e1) eq 0 and count(.//@a) eq 0"))
	acTally(test, acEl(1, "e1"), acAt(0, "a"))
	fresh := acTally(test, acEl(0, "e1"), acEl(-1, "e1"))
	if !test.Evaluate(backend(), seededTypes, asValues(t), asNoChildren, fresh, ValueBinding{}) {
		t.Error("Evaluate over a fresh Tally given only out-of-range depths = false, want true")
	}
	var none *Tally
	none.Element([]xsd.QName{uq("e1")}, []xsd.QName{uq("a")})
}

// CountsAttributesAt answers per counted path: `@a` selects attributes of E
// alone, `.//@a` those of E and of every element below it, and `e1`, `.//e1`
// and the child path `e1/e1` select none at any depth. A test that counts
// nothing has a nil Tally, which answers false, and so does every Tally at a
// depth below 0.
func TestTallyCountsAttributesAt(t *testing.T) {
	for _, tc := range []struct {
		expr string
		want map[int]bool
	}{
		{"count(@a) eq 0", map[int]bool{-1: false, 0: true, 1: false, 2: false}},
		{"count(.//@a) eq 0", map[int]bool{-1: false, 0: true, 1: true, 2: true}},
		{"count(e1) eq 0", map[int]bool{-1: false, 0: false, 1: false, 2: false}},
		{"count(.//e1) eq 0", map[int]bool{-1: false, 0: false, 1: false, 2: false}},
		{"exists(e1/e1)", map[int]bool{-1: false, 0: false, 1: false, 2: false}},
		{"@length eq 1", map[int]bool{-1: false, 0: false, 1: false, 2: false}},
	} {
		c := acCompile(t, asRecord(tc.expr)).Tally()
		for depth := -1; depth <= 2; depth++ {
			if got := c.CountsAttributesAt(depth); got != tc.want[depth] {
				t.Errorf("Tally of %q: CountsAttributesAt(%d) = %v, want %v", tc.expr, depth, got, tc.want[depth])
			}
		}
	}
}

// fn:count's argument is one QName step, or a rooted one: every other argument
// declines, and so does `.//` or `./` in a comparison operand, and a cast from
// a count, which [16] ta-SimpleValue, the operand of both cast spellings, does
// not admit.
func TestCompileAssertionTestDeclinesCounts(t *testing.T) {
	for _, tc := range []struct{ expr, why string }{
		{"count(*) = 0", "a wildcard NameTest"},
		{"count(@*) = 0", "an attribute wildcard"},
		{"count(.//*) = 0", "a descendant wildcard"},
		{"count(//*) = 0", "a rooted wildcard"},
		{"count(//@*) = 0", "a rooted attribute wildcard"},
		{"count(e1/e1) = 0", "a path of two steps"},
		{"count(e1//e1) = 0", "a path through descendant-or-self"},
		{"count(//e1/e1) = 0", "a rooted path of two steps"},
		{"count(.) = 1", "the context item alone"},
		{"count(..) = 1", "the parent step"},
		{"count($value) = 1", "a variable"},
		{"count(e1[1]) = 1", "a predicate"},
		{"count(child::e1) = 1", "the unabbreviated child axis"},
		{"count(attribute::a) = 1", "the unabbreviated attribute axis"},
		{"count() = 0", "no argument"},
		{"count(e1, e1) = 0", "two arguments"},
		{"count(./) = 0", "a slash with no step after it"},
		{"count(.//) = 0", "a double slash with no step after it"},
		{"count(/) = 0", "a bare slash"},
		{"count(@p:a) = 0", "an unbound prefix"},
		{".//e1 = 'a'", "a descendant step in a comparison"},
		{"./@length = 1", "a context-item path in a comparison"},
		{"count(e1) cast as xs:string = '1'", "a cast from an xs:integer count"},
		{"count(count(e1)) = 1", "a count of a count"},
	} {
		if _, ok := CompileAssertionTest(asRecord(tc.expr), seededTypes, xsd.ElementContent{}, asUses(t, map[string]string{"length": "int"}), asChildTypes(t)); ok {
			t.Errorf("CompileAssertionTest(%q): compiled, want declined (%s)", tc.expr, tc.why)
		}
	}
}

// fn:count is the assertion façade's alone: a Type Alternative's {test}
// declines it under §3.12.6 clause 3, and an assertions facet's declines it
// (ctaFacetFacade.count) rather than failing. A rooted attribute step reads the
// facet's absent context item and fails it (err:XPDY0002), and a Type
// Alternative declines it as it declines every rooted path. The relative count
// rows compile, and fail, with either façade's count admitting the call; the
// rooted one is declined there by the rooted path first. The `//@a` facet row
// declines instead, and fails, with ctaParser.rootedPath admitting no
// attribute step.
func TestCountDeclinesOutsideTheAssertionFacade(t *testing.T) {
	str := asBuiltin(t, "string")
	for _, expr := range []string{"count(@a) = 0", "count(.//e1) = 0", "count(//e1) = 0", "//@a = 'x'"} {
		if _, ok := CompileCTATest(ctaExprRecord(expr, ""), seededTypes); ok {
			t.Errorf("CompileCTATest(%q): compiled, want declined", expr)
		}
	}
	for _, tc := range []struct {
		test string
		want value.AssertionOutcome
	}{
		{"count(@a) = 0", value.AssertionDeclined},
		{"count(.//e1) = 0", value.AssertionDeclined},
		{"count(//e1) = 0", value.AssertionDeclined},
		{"//@a = 'x'", value.AssertionFails},
	} {
		if got := FacetAssertions().Evaluate(backend(), seededTypes, str, ctaExprRecord(tc.test, ""), fcValue(t, str, "x")); got != tc.want {
			t.Errorf("FacetAssertions().Evaluate(%q) = %d, want %d", tc.test, got, tc.want)
		}
	}
}
