package xpath

import (
	"testing"
	"time"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// The fixtures below drive the two existence operands the purchase-order
// assertion of ibmData/mixed/assertions/po_sample/po.xsd:33 writes: a child
// path whose last step is the wildcard `*` (xpath20.md §3.2.1.2), and a child
// step filtered by a conjunction of child steps (§3.2.2). Both stand where a
// child path stands — the whole operand of fn:exists, fn:empty or an
// ·effective boolean value· — and are read off the [Tally].

// awCompile compiles record for an E of element-only content declaring no
// attribute and no child type, or fails the test: neither operand consults
// AttributeTypes or ElementTypes.
func awCompile(t *testing.T, record xsd.XPathExpression) AssertionTest {
	t.Helper()
	test, ok := CompileAssertionTest(record, seededTypes, asElementContent(t, false), asUses(t, nil), asNoElems)
	if !ok {
		t.Fatalf("CompileAssertionTest(%q): declined, want compiled", record.Expression())
	}
	return test
}

// awEvaluate is test evaluated over a Tally nodes are reported to (acTally).
func awEvaluate(t *testing.T, test AssertionTest, nodes ...acNode) bool {
	t.Helper()
	return test.Evaluate(backend(), seededTypes, asElem, asValues(t), asNoChildren, acTally(test, nodes...), ValueBinding{}, time.Time{})
}

// `N/*` selects every element child of an N child of E, whatever its name or
// namespace, so its ·effective boolean value· is true where some N has an
// element child and false where every N is empty or there is none (§2.4.3
// rule 2); fn:not over it is the inverse, true for an empty or absent N. The
// rows pin both polarities as the whole {test}, an `and` operand and an
// IfExpr's test, and under fn:exists and fn:empty. Every row declines at
// CompileAssertionTest, and so fails, with childPathLength measuring no `*`
// step.
func TestAssertionWildcardChildPath(t *testing.T) {
	ba := acPath("billing-address")
	street := acPath("billing-address", "street1")
	foreign := acNode{path: []xsd.QName{uq("billing-address"), {Space: "urn:x", Local: "any"}}}
	sa := acPath("shipping-address")
	saCity := acPath("shipping-address", "city")
	for _, tc := range []struct {
		why   string
		expr  string
		nodes []acNode
		holds bool
	}{
		{why: "a child of N", expr: "billing-address/*", nodes: []acNode{ba, street}, holds: true},
		{why: "a namespaced child of N", expr: "billing-address/*", nodes: []acNode{ba, foreign}, holds: true},
		{why: "an empty N", expr: "billing-address/*", nodes: []acNode{ba}},
		{why: "no N", expr: "billing-address/*", nodes: []acNode{sa, saCity}},
		{why: "a child of N with a child of its own",
			expr: "billing-address/*", nodes: []acNode{ba, street, acPath("billing-address", "street1", "x")}, holds: true},
		{why: "the empty N first, then a populated one",
			expr: "billing-address/*", nodes: []acNode{ba, ba, street}, holds: true},
		{why: "a child of N named N", expr: "billing-address/*",
			nodes: []acNode{ba, acPath("billing-address", "billing-address")}, holds: true},
		{why: "fn:not over an empty N", expr: "not(shipping-address/*)", nodes: []acNode{sa}, holds: true},
		{why: "fn:not over no N", expr: "not(shipping-address/*)", holds: true},
		{why: "fn:not over a populated N", expr: "not(shipping-address/*)", nodes: []acNode{sa, saCity}},
		{why: "an and operand, true", expr: "billing-address/* and not(shipping-address/*)",
			nodes: []acNode{ba, street, sa}, holds: true},
		{why: "an and operand, false", expr: "billing-address/* and not(shipping-address/*)",
			nodes: []acNode{ba, street, sa, saCity}},
		{why: "an IfExpr test selecting then", expr: "if (billing-address/*) then true() else false()",
			nodes: []acNode{ba, street}, holds: true},
		{why: "an IfExpr test selecting else", expr: "if (billing-address/*) then true() else false()",
			nodes: []acNode{ba}},
		{why: "a three-step path", expr: "a/b/*", nodes: []acNode{acPath("a", "b", "c")}, holds: true},
		{why: "a three-step path one level short", expr: "a/b/*", nodes: []acNode{acPath("a", "b")}},
		{why: "fn:exists", expr: "exists(billing-address/*)", nodes: []acNode{ba, street}, holds: true},
		{why: "fn:empty", expr: "empty(billing-address/*)", nodes: []acNode{ba}, holds: true},
	} {
		t.Run(tc.why, func(t *testing.T) {
			test := awCompile(t, asRecord(tc.expr))
			if got := awEvaluate(t, test, tc.nodes...); got != tc.holds {
				t.Errorf("Evaluate(%q) over %v = %v, want %v", tc.expr, tc.nodes, got, tc.holds)
			}
		})
	}
}

// `N[a and b]` selects each child N of E that has, among ITS children, one
// named each conjunct (§3.2.2 rule 2: the predicate is evaluated once per N,
// that N the context item), so two N children that split the names between
// them select nothing — where the hoisted `N/a and N/b`, which the split row
// also pins, is true. The rows run the operand as an ·effective boolean
// value·, as an IfExpr branch and under fn:exists and fn:empty. Every row but
// the hoisted one declines at CompileAssertionTest, and so fails, with
// booleanExpr's and presenceArgument's childrenHaving arms removed; the split
// row and the fn:empty row fail with ctaInstanceCount.element keeping the
// shown names across N children.
func TestAssertionChildrenHaving(t *testing.T) {
	const all = "billing-address[street1 and city and country]"
	ba := acPath("billing-address")
	street := acPath("billing-address", "street1")
	city := acPath("billing-address", "city")
	country := acPath("billing-address", "country")
	other := acPath("billing-address", "zipcode")
	for _, tc := range []struct {
		why   string
		expr  string
		nodes []acNode
		holds bool
	}{
		{why: "one N with all three", expr: all, nodes: []acNode{ba, street, city, country}, holds: true},
		{why: "one N with all three, in another order, among others", expr: all,
			nodes: []acNode{ba, country, other, street, city, other}, holds: true},
		{why: "one N lacking one", expr: all, nodes: []acNode{ba, street, city, other}},
		{why: "no N", expr: all},
		{why: "an empty N", expr: all, nodes: []acNode{ba}},
		{why: "two N splitting the three", expr: all, nodes: []acNode{ba, street, ba, city, country}},
		{why: "the hoisted conjunction is true on the split row",
			expr:  "billing-address/street1 and billing-address/city and billing-address/country",
			nodes: []acNode{ba, street, ba, city, country}, holds: true},
		{why: "an empty N, then one with all three", expr: all,
			nodes: []acNode{ba, ba, street, city, country}, holds: true},
		{why: "one with all three, then an empty N", expr: all,
			nodes: []acNode{ba, street, city, country, ba}, holds: true},
		{why: "one with all three, then a sibling of another name", expr: all,
			nodes: []acNode{ba, street, city, country, acPath("email")}, holds: true},
		{why: "the names under a sibling of another name", expr: all,
			nodes: []acNode{ba, acPath("x"), acPath("x", "street1"), acPath("x", "city"), acPath("x", "country")}},
		{why: "a required name one level too deep", expr: all,
			nodes: []acNode{ba, street, city, other, acPath("billing-address", "zipcode", "country")}},
		{why: "one conjunct", expr: "billing-address[city]", nodes: []acNode{ba, city}, holds: true},
		{why: "a name written twice", expr: "billing-address[city and city]", nodes: []acNode{ba, city}, holds: true},
		{why: "fn:not", expr: "not(" + all + ")", nodes: []acNode{ba, street}, holds: true},
		{why: "an IfExpr branch, parenthesized", expr: "if (billing-address/*) then (" + all + ") else true()",
			nodes: []acNode{ba, street, city}},
		{why: "an IfExpr branch, satisfied", expr: "if (billing-address/*) then " + all + " else false()",
			nodes: []acNode{ba, street, city, country}, holds: true},
		{why: "an and operand", expr: all + " and shipping-address[street1]",
			nodes: []acNode{ba, street, city, country, acPath("shipping-address"), acPath("shipping-address", "street1")}, holds: true},
		{why: "fn:exists", expr: "exists(" + all + ")", nodes: []acNode{ba, street, city, country}, holds: true},
		{why: "fn:empty", expr: "empty(" + all + ")", nodes: []acNode{ba, street, ba, city, country}, holds: true},
	} {
		t.Run(tc.why, func(t *testing.T) {
			test := awCompile(t, asRecord(tc.expr))
			if got := awEvaluate(t, test, tc.nodes...); got != tc.holds {
				t.Errorf("Evaluate(%q) over %v = %v, want %v", tc.expr, tc.nodes, got, tc.holds)
			}
		})
	}
}

// Neither operand reads a child's value, so ReadsChild is false for every name
// it writes, while Tally is non-nil. `N[a and b]` and `N[b and a]` select the
// same nodes and keep one counter, as `a/*` written twice does; `a/*` and
// `a/b` keep two.
func TestAssertionWildcardAndHavingCounters(t *testing.T) {
	test := awCompile(t, asRecord("billing-address/* and billing-address[street1 and city]"))
	for _, name := range []string{"billing-address", "street1", "city"} {
		if test.ReadsChild(uq(name)) {
			t.Errorf("ReadsChild(%s) = true, want false", name)
		}
	}
	for _, tc := range []struct {
		expr string
		want int
	}{
		{"billing-address[street1 and city] and billing-address[city and street1]", 1},
		{"a/* and exists(a/*)", 1},
		{"a/* and a/b", 2},
		{"a[b] and a/b", 2},
	} {
		if c := awCompile(t, asRecord(tc.expr)).Tally(); c == nil || len(c.counters) != tc.want {
			t.Errorf("(%s).Tally() = %v, want %d counters", tc.expr, c, tc.want)
		}
	}
}

// awPurchaseOrder is the first assertion of the PO_BUSINESS_RULES type,
// ibmData/mixed/assertions/po_sample/po.xsd:33, as written there.
const awPurchaseOrder = `if (billing-address/* and not(shipping-address/*)) then (billing-address[street1 and city and country])
                         else
                       if (shipping-address/* and not(billing-address/*)) then (shipping-address[street1 and city and country])
                         else
                       if (shipping-address/* and billing-address/*) then (billing-address[street1 and city and country] and
                         shipping-address[street1 and city and country]) else
                       true()`

// po.xsd's first assertion compiles and decides the three shapes its
// documentation names — (a) billing-address empty, so shipping-address must
// have street1, city and country; (b) shipping-address empty, so
// billing-address must; (c) both populated, so both must — and the fourth it
// leaves to `true()`, both empty. Row (b), satisfied, is po.xml's own order.
func TestAssertionPurchaseOrderRules(t *testing.T) {
	test := awCompile(t, asRecord(awPurchaseOrder))
	address := func(name string, children ...string) []acNode {
		nodes := []acNode{acPath(name)}
		for _, c := range children {
			nodes = append(nodes, acPath(name, c))
		}
		return nodes
	}
	order := func(billing, shipping []acNode) []acNode {
		nodes := []acNode{acPath("buyer"), acPath("buyer", "fName")}
		nodes = append(nodes, billing...)
		nodes = append(nodes, shipping...)
		return append(nodes, acPath("email"), acPath("items"), acPath("items", "item"))
	}
	full := []string{"street1", "street2", "city", "zipcode", "state", "country"}
	partial := []string{"street1", "city", "state"}
	for _, tc := range []struct {
		why               string
		billing, shipping []acNode
		holds             bool
	}{
		{"(a) shipping complete", address("billing-address"), address("shipping-address", full...), true},
		{"(a) shipping lacking country", address("billing-address"), address("shipping-address", partial...), false},
		{"(b) billing complete, po.xml", address("billing-address", full...), address("shipping-address"), true},
		{"(b) billing lacking country", address("billing-address", partial...), address("shipping-address"), false},
		{"(c) both complete", address("billing-address", full...), address("shipping-address", full...), true},
		{"(c) billing lacking country", address("billing-address", partial...), address("shipping-address", full...), false},
		{"(c) shipping lacking country", address("billing-address", full...), address("shipping-address", partial...), false},
		{"both empty", address("billing-address"), address("shipping-address"), true},
	} {
		if got := awEvaluate(t, test, order(tc.billing, tc.shipping)...); got != tc.holds {
			t.Errorf("%s: Evaluate = %v, want %v", tc.why, got, tc.holds)
		}
	}
}

// A report breaking the pre-order [Tally] obliges — a grandchild before its
// parent, or a grandchild whose parent was never reported —
// drops the Tally's counters, and Evaluate answers false whatever the reports
// after it hold. The control rows report the same nodes in pre-order and
// hold. With Tally.Element's ctaNests check removed both breach rows hold.
func TestTallyRefusesAnOutOfOrderReport(t *testing.T) {
	for _, tc := range []struct {
		why   string
		expr  string
		paths [][]string
		holds bool
	}{
		{"a/b in pre-order", "a/b", [][]string{{}, {"a"}, {"a", "b"}}, true},
		{"a/b before a", "a/b", [][]string{{}, {"a", "b"}, {"a"}}, false},
		{"a[b] in pre-order", "a[b]", [][]string{{}, {"a"}, {"a", "b"}, {"c"}, {"c", "d"}}, true},
		{"a[b], then a grandchild under an unreported c", "a[b]", [][]string{{}, {"a"}, {"a", "b"}, {"c", "d"}}, false},
	} {
		test := awCompile(t, asRecord(tc.expr))
		c := test.Tally()
		for _, locals := range tc.paths {
			path := []xsd.QName{}
			for _, l := range locals {
				path = append(path, uq(l))
			}
			c.Element(path, nil)
		}
		if got := test.Evaluate(backend(), seededTypes, asElem, asValues(t), asNoChildren, c, ValueBinding{}, time.Time{}); got != tc.holds {
			t.Errorf("%s: Evaluate(%q) = %v, want %v", tc.why, tc.expr, got, tc.holds)
		}
	}
}

// Each shape beside the two operands still declines: a child-axis wildcard or
// a filtered child step in a Type Alternative's {test}, whose instance has no
// [children] (key-cta-ta-select clause 1.2); and, in an assertion's, fn:count
// over either, a wildcard step read for its value, `*:N` and `a:*` (a prefix
// asRecord binds), a wildcard before the last step, a numeric predicate in an
// ·effective boolean value· and in an fn:count argument, and a predicate of any
// other shape. The Type Alternative rows fail with
// ctaTypeAlternativeFacade.childPath and childrenHaving admitting the path.
func TestWildcardAndHavingStillDecline(t *testing.T) {
	for _, expr := range []string{"a/*", "not(a/*)", "a[b and c]", "a[b] and @x"} {
		if _, ok := CompileCTATest(ctaExprRecord(expr, ""), seededTypes); ok {
			t.Errorf("CompileCTATest(%q): compiled, want declined", expr)
		}
	}
	for _, tc := range []struct{ expr, why string }{
		{"count(a/*) = 1", "fn:count over a wildcard path"},
		{"count(a[b]) = 1", "fn:count over a filtered step"},
		{"a/* = 1", "a wildcard step read for its value"},
		{"a[b] = 1", "a filtered step read for its value"},
		{"a/*:b", "a *:N step"},
		{"a/a:*", "an a:* step"},
		{"*:a", "a *:N step alone"},
		{"a/*/b", "a wildcard before the last step"},
		{"*/a", "a wildcard first step"},
		{"a[1]", "a numeric predicate as an effective boolean value"},
		{"exists(a[1])", "a numeric predicate under fn:exists"},
		{"count(a[1]) = 1", "a numeric predicate in an fn:count argument"},
		{"a[b or c]", "a disjunction"},
		{"a[not(b)]", "fn:not in the predicate"},
		{"a[b/c]", "a path in the predicate"},
		{"a[@b and c]", "an attribute test beside a child step"},
		{"a[b][c]", "a second predicate"},
		{"a[b]/c", "a step after the predicate"},
		{"a/b[c]", "a predicate on a later step"},
		{"a[*]", "a wildcard in the predicate"},
	} {
		if _, ok := CompileAssertionTest(asRecord(tc.expr), seededTypes, asElementContent(t, false), asUses(t, nil), asNoElems); ok {
			t.Errorf("CompileAssertionTest(%q): compiled, want declined (%s)", tc.expr, tc.why)
		}
	}
}

// An assertions facet's {test} has no context item, so either operand reads
// the absent one and Fails, never Holds (ctaFacetFacade.childPath,
// ctaFacetFacade.childrenHaving, err:XPDY0002), fn:not over it included.
func TestWildcardAndHavingInAFacet(t *testing.T) {
	str := asBuiltin(t, "string")
	for _, test := range []string{"a/*", "not(a/*)", "a[b]", "not(a[b and c])", "exists(a[b])"} {
		if got := FacetAssertions(time.Time{}).Evaluate(backend(), seededTypes, str, ctaExprRecord(test, ""), fcValue(t, str, "x")); got != value.AssertionFails {
			t.Errorf("FacetAssertions().Evaluate(%q) = %d, want Fails (%d)", test, got, value.AssertionFails)
		}
	}
}
