package xpath

import (
	"testing"
	"time"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// The fixtures below drive a child step filtered by one `preceding::` step,
// `N[preceding::M]`, `N[preceding::M[P]]` and `N[preceding::M[not(P)]]`, the
// last as saxonData/Assert/assert005.xsd:26 writes it under fn:not. Each
// stands where a child path stands — the whole operand of fn:exists, fn:empty
// or an ·effective boolean value· — and is read off the [Tally]. cvc-assertion
// clause 1.3 roots the tree at E, so the preceding elements of a child of E are
// every element of E's subtree reported before it, at any depth, E itself never
// among them (xpath20.md §3.2.1.1).

// `a[preceding::a]` selects each child a of E with some a element of E's
// subtree before it: every a child but the first where all are children, the
// one a child after an a nested in an earlier sibling, and none where the only
// a children come before every other a. The counts are read off the Tally the
// reports filled. Every row fails with ctaPrecedenceCount.element counting the
// first a child (kept tested after the child opens instead of before); the
// nested and ancestor rows fail with it closing only the elements at depth 1.
func TestPrecedingSelectsEachLaterChild(t *testing.T) {
	key := ctaChildrenPreceded{name: uq("a"), preceding: uq("a"), test: ctaAnyPreceding{}}
	a, x := acPath("a"), acPath("x")
	for _, tc := range []struct {
		why   string
		nodes []acNode
		want  int
	}{
		{"one a", []acNode{a}, 0},
		{"three a", []acNode{a, a, a}, 2},
		{"x, a", []acNode{x, a}, 0},
		{"an a nested in an earlier sibling, then a", []acNode{x, acPath("x", "a"), a}, 1},
		{"an a nested two deep in an earlier sibling, then a", []acNode{x, acPath("x", "y"), acPath("x", "y", "a"), a}, 1},
		{"a, then an a nested in a later sibling", []acNode{a, x, acPath("x", "a")}, 0},
		{"an a whose own child is an a", []acNode{a, acPath("a", "a")}, 0},
		{"an a whose own child is an a, then a", []acNode{a, acPath("a", "a"), a}, 1},
	} {
		t.Run(tc.why, func(t *testing.T) {
			test := awCompile(t, asRecord("a[preceding::a]"))
			got, ok := acTally(test, tc.nodes...).count(key)
			if !ok || got != tc.want {
				t.Errorf("count(a[preceding::a]) over %v = %d, %v; want %d, true", tc.nodes, got, ok, tc.want)
			}
		})
	}
}

// `not(a[preceding::a[not(b)]])` is false exactly where some child a of E has
// an a of E's subtree before it, at any depth and not its ancestor, with no b
// child, and true otherwise — assert005's v1 and v2 hold and n1 does not. A b
// one level too deep is no child, and an a after an earlier a that lacks b
// counts only where it is a child of E. The positive filter `[b]` and the
// unfiltered step run beside it, as an ·effective boolean value·, under
// fn:exists and fn:empty, and as an `and` operand. Every row declines at
// CompileAssertionTest, and so fails, with booleanExpr's and
// presenceArgument's childrenPreceded arms removed.
func TestPrecedingWithAChildFilter(t *testing.T) {
	const assert005 = "not(a[preceding::a[not(b)]])"
	a, ab, x := acPath("a"), acPath("a", "b"), acPath("x")
	for _, tc := range []struct {
		why   string
		expr  string
		nodes []acNode
		holds bool
	}{
		{"assert005.v1's first inner: a/b, a/b", assert005, []acNode{a, ab, a, ab}, true},
		{"assert005.v1's second inner: a", assert005, []acNode{a}, true},
		{"assert005.v2's inner: a/b/b, a/b/b", assert005, []acNode{a, ab, ab, a, ab, ab}, true},
		{"assert005.n1's third inner: a, a", assert005, []acNode{a, a}, false},
		{"no a", assert005, nil, true},
		{"a/b, then a", assert005, []acNode{a, ab, a}, true},
		{"a, then a/b", assert005, []acNode{a, a, ab}, false},
		{"a/b, a, a/b: the third is preceded by the second", assert005, []acNode{a, ab, a, a, ab}, false},
		{"a b one level too deep, then a", assert005, []acNode{a, acPath("a", "c"), acPath("a", "c", "b"), a}, false},
		{"an a nested in an earlier sibling, lacking b, then a", assert005, []acNode{x, acPath("x", "a"), a}, false},
		{"an a nested in an earlier sibling, with b, then a", assert005, []acNode{x, acPath("x", "a"), acPath("x", "a", "b"), a}, true},
		{"a, then an a nested in a later sibling", assert005, []acNode{a, x, acPath("x", "a")}, true},
		{"an a whose own child a lacks b", assert005, []acNode{a, ab, acPath("a", "a")}, true},
		{"the positive filter, satisfied", "a[preceding::a[b]]", []acNode{a, ab, a}, true},
		{"the positive filter, unsatisfied", "a[preceding::a[b]]", []acNode{a, a, ab}, false},
		{"another preceding name", "a[preceding::c]", []acNode{acPath("c"), a}, true},
		{"another preceding name, after", "a[preceding::c]", []acNode{a, acPath("c")}, false},
		{"fn:exists", "exists(a[preceding::a])", []acNode{a, a}, true},
		{"fn:empty", "empty(a[preceding::a[not(b)]])", []acNode{a, ab, a}, true},
		{"an and operand", "a[preceding::a] and " + assert005, []acNode{a, ab, a}, true},
	} {
		t.Run(tc.why, func(t *testing.T) {
			test := awCompile(t, asRecord(tc.expr))
			if got := awEvaluate(t, test, tc.nodes...); got != tc.holds {
				t.Errorf("Evaluate(%q) over %v = %v, want %v", tc.expr, tc.nodes, got, tc.holds)
			}
		})
	}
}

// Every name resolves on elementName's terms, so under a default namespace an
// unprefixed name is in it and a no-namespace element is no match; a prefixed
// name takes its binding. The rows fail with childrenPreceded resolving the
// preceding or filter name on attributeName's terms.
func TestPrecedingResolvesNames(t *testing.T) {
	d := func(local string) xsd.QName { return xsd.QName{Space: "urn:d", Local: local} }
	da := acNode{path: []xsd.QName{d("a")}}
	dab := acNode{path: []xsd.QName{d("a"), d("b")}}
	record := ctaExprRecord("not(a[preceding::a[not(b)]])", "urn:d")
	test := awCompile(t, record)
	if got := awEvaluate(t, test, da, da); got {
		t.Errorf("default namespace: two a lacking b: true, want false")
	}
	if got := awEvaluate(t, test, da, dab, da); !got {
		t.Errorf("default namespace: a/b, then a: false, want true")
	}
	if got := awEvaluate(t, test, acPath("a"), acPath("a")); !got {
		t.Errorf("default namespace: two no-namespace a: false, want true")
	}
	prefixed := awCompile(t, ctaExprRecord("not(p:a[preceding::p:a[not(p:b)]])", "", "p", "urn:d"))
	if got := awEvaluate(t, prefixed, da, da); got {
		t.Errorf("prefixed: two a lacking b: true, want false")
	}
}

// The step reads no child's value, and the same key written twice keeps one
// counter; the positive and the negated filter keep two.
func TestPrecedingCounters(t *testing.T) {
	test := awCompile(t, asRecord("not(a[preceding::a[not(b)]])"))
	for _, name := range []string{"a", "b"} {
		if test.ReadsChild(uq(name)) {
			t.Errorf("ReadsChild(%s) = true, want false", name)
		}
	}
	for _, tc := range []struct {
		expr string
		want int
	}{
		{"a[preceding::a[not(b)]] and exists(a[preceding::a[not(b)]])", 1},
		{"a[preceding::a[b]] and a[preceding::a[not(b)]]", 2},
		{"a[preceding::a] and a[preceding::a[b]]", 2},
	} {
		if c := awCompile(t, asRecord(tc.expr)).Tally(); c == nil || len(c.counters) != tc.want {
			t.Errorf("(%s).Tally() = %v, want %d counters", tc.expr, c, tc.want)
		}
	}
}

// Every shape beside the admitted ones still declines in an assertion — the
// other axes, `ancestor::` and `preceding-sibling::` among them, under
// childrenPreceded's GAP(xpath) (#1042) — and a Type Alternative's {test},
// whose instance has no [children] (key-cta-ta-select clause 1.2), declines
// even the admitted ones. Guard: the assertion rows pass today; the Type
// Alternative rows fail with ctaTypeAlternativeFacade.childrenPreceded
// admitting the node.
func TestPrecedingStillDeclines(t *testing.T) {
	for _, expr := range []string{"not(a[preceding::a[not(b)]])", "a[preceding::a]", "exists(a[preceding::a[b]])"} {
		if _, ok := CompileCTATest(ctaExprRecord(expr, ""), seededTypes); ok {
			t.Errorf("CompileCTATest(%q): compiled, want declined", expr)
		}
	}
	for _, tc := range []struct{ expr, why string }{
		{"a[ancestor::x]", "ancestor::"},
		{"not(a[ancestor::x])", "ancestor:: under fn:not"},
		{"a[preceding-sibling::a]", "preceding-sibling::"},
		{"not(a[preceding-sibling::a[not(b)]])", "preceding-sibling:: with a filter"},
		{"a[following::a]", "following::"},
		{"a[descendant::a]", "descendant::"},
		{"count(a[preceding::a]) = 1", "fn:count over it"},
		{"a[preceding::a] = 1", "read for its value"},
		{"./a[preceding::a]", "on ./N"},
		{".//a[preceding::a]", "on .//N"},
		{"x/a[preceding::a]", "on a longer path"},
		{"a[preceding::a]/b", "a step after it"},
		{"a[preceding::*]", "a wildcard preceding step"},
		{"a[preceding::node()]", "a kind test"},
		{"a[preceding::a/b]", "a step after the preceding one"},
		{"a[preceding::a[*]]", "a wildcard filter"},
		{"a[preceding::a[@b]]", "an attribute filter"},
		{"a[preceding::a[1]]", "a numeric filter"},
		{"a[preceding::a[b/c]]", "a path filter"},
		{"a[preceding::a[not(b) and c]]", "a conjunction filter"},
		{"a[preceding::a[not(b/c)]]", "fn:not over a path"},
		{"a[preceding::a[b][c]]", "a second filter"},
		{"a[preceding::a[exists(b)]]", "another function"},
		{"a[preceding::a and b]", "a conjunct beside the step"},
		{"a[preceding::a][b]", "a second outer predicate"},
		{"a[not(b)]", "fn:not alone"},
		{"preceding::a", "the step off E"},
	} {
		if _, ok := CompileAssertionTest(asRecord(tc.expr), seededTypes, asElementContent(t, false), asUses(t, nil), asNoElems); ok {
			t.Errorf("CompileAssertionTest(%q): compiled, want declined (%s)", tc.expr, tc.why)
		}
	}
}

// An assertions facet's {test} has no context item, so the step reads the
// absent one and Fails, never Holds (ctaFacetFacade.childrenPreceded,
// err:XPDY0002), fn:not over it included. The rows fail with
// ctaFacetFacade.childrenPreceded declining instead.
func TestPrecedingInAFacet(t *testing.T) {
	str := asBuiltin(t, "string")
	for _, test := range []string{"a[preceding::a]", "not(a[preceding::a[not(b)]])", "empty(a[preceding::a[b]])"} {
		if got := FacetAssertions(time.Time{}).Evaluate(backend(), seededTypes, str, ctaExprRecord(test, ""), fcValue(t, str, "x")); got != value.AssertionFails {
			t.Errorf("FacetAssertions().Evaluate(%q) = %d, want Fails (%d)", test, got, value.AssertionFails)
		}
	}
}
