package xpath

import (
	"testing"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// The fixtures below drive xpath20.md §3.8's IfExpr, which the assertion and
// facet façades admit and a Type Alternative's and a value predicate's
// decline (ctaFacade.conditional).

// aiElems is E's child element types for the IfExpr rows: a, b, c and d are
// xs:int, as ibmData/mixed/assertions/test12.xsd declares them.
func aiElems(t *testing.T) ElementTypes {
	t.Helper()
	i := asBuiltin(t, "int")
	return asElems(map[xsd.QName]xsd.TypeDefinition{uq("a"): i, uq("b"): i, uq("c"): i, uq("d"): i})
}

// aiCompile compiles expr for an element with element-only content whose
// attribute t is an xs:string and whose children aiElems types, or fails the
// test.
func aiCompile(t *testing.T, expr string) AssertionTest {
	t.Helper()
	test, ok := CompileAssertionTest(asRecord(expr), seededTypes, xsd.ElementContent{}, asUses(t, map[string]string{"t": "string"}), aiElems(t))
	if !ok {
		t.Fatalf("CompileAssertionTest(%q): declined, want compiled", expr)
	}
	return test
}

// aiInts is one xs:int child per name=lexical pair of kv, in written order.
func aiInts(kv ...string) []asChild {
	out := make([]asChild, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		out = append(out, asChild{uq(kv[i]), "int", kv[i+1], false})
	}
	return out
}

// `if (@t eq 'x') then a = b else true()` decides a = b where the test is true
// and is true where it is false, whatever a and b hold — a = b false and `a`
// absent included — and an absent @t makes the value comparison the empty
// sequence, whose ·effective boolean value· is false (§2.4.3), so the else
// branch decides. The three test12.xsd {test}s (assert_012) decide the shape
// they guard and pass every other: `not(d)` in a then-branch is the existence
// of d, and so is `a` before an 'else', whatever its value. Every row fails
// with ctaAssertionFacade.conditional answering false, at the compile.
func TestAssertionIfExpr(t *testing.T) {
	x, y := asTyped{uq("t"), "string", "x"}, asTyped{uq("t"), "string", "y"}
	const square = "if (@t eq 'square') then (a = b and b = c and c = d) else true()"
	const rectangle = "if (@t eq 'rectangle') then (a = c and b = d) else true()"
	const triangle = "if (@t eq 'triangle') then not(d) else true()"
	sq := asTyped{uq("t"), "string", "square"}
	rect := asTyped{uq("t"), "string", "rectangle"}
	tri := asTyped{uq("t"), "string", "triangle"}
	for _, tc := range []struct {
		expr     string
		attrs    []asTyped
		children []asChild
		want     bool
	}{
		{"if (@t eq 'x') then a = b else true()", []asTyped{x}, aiInts("a", "1", "b", "1"), true},
		{"if (@t eq 'x') then a = b else true()", []asTyped{x}, aiInts("a", "1", "b", "2"), false},
		{"if (@t eq 'x') then a = b else true()", []asTyped{x}, aiInts("b", "2"), false},
		{"if (@t eq 'x') then a = b else true()", []asTyped{y}, aiInts("a", "1", "b", "1"), true},
		{"if (@t eq 'x') then a = b else true()", []asTyped{y}, aiInts("a", "1", "b", "2"), true},
		{"if (@t eq 'x') then a = b else true()", []asTyped{y}, aiInts("b", "2"), true},
		{"if (@t eq 'x') then a = b else true()", nil, aiInts("a", "1", "b", "2"), true},
		{"if (@t eq 'x') then a = b else false()", []asTyped{y}, aiInts("a", "1", "b", "1"), false},
		{"if (@t eq 'x') then a else false()", []asTyped{x}, aiInts("a", "0"), true},
		{"if (@t eq 'x') then a else false()", []asTyped{x}, nil, false},
		{square, []asTyped{sq}, aiInts("a", "4", "b", "4", "c", "4", "d", "4"), true},
		{square, []asTyped{sq}, aiInts("a", "4", "b", "4", "c", "4", "d", "5"), false},
		{square, []asTyped{rect}, aiInts("a", "4", "b", "4", "c", "4", "d", "5"), true},
		{rectangle, []asTyped{rect}, aiInts("a", "4", "b", "2", "c", "4", "d", "2"), true},
		{rectangle, []asTyped{rect}, aiInts("a", "4", "b", "2", "c", "3", "d", "2"), false},
		{triangle, []asTyped{tri}, aiInts("a", "3", "b", "4", "c", "5"), true},
		{triangle, []asTyped{tri}, aiInts("a", "3", "b", "4", "c", "5", "d", "6"), false},
		{triangle, []asTyped{sq}, aiInts("a", "3", "b", "4", "c", "5", "d", "6"), true},
		{"if (@t eq 'x') then if (a = 1) then b = 2 else b = 3 else false()", []asTyped{x}, aiInts("a", "1", "b", "2"), true},
		{"if (@t eq 'x') then if (a = 1) then b = 2 else b = 3 else false()", []asTyped{x}, aiInts("a", "0", "b", "2"), false},
		{"not(if (@t eq 'x') then a = b else false())", []asTyped{x}, aiInts("a", "1", "b", "2"), true},
	} {
		test := aiCompile(t, tc.expr)
		var nodes []acNode
		for _, c := range tc.children {
			nodes = append(nodes, acPath(c.name.Local))
		}
		got := test.Evaluate(backend(), seededTypes, asValues(t, tc.attrs...), asChildren(t, tc.children...), acTally(test, nodes...), ValueBinding{})
		if got != tc.want {
			t.Errorf("Evaluate(%q) over %v, %v = %v, want %v", tc.expr, tc.attrs, tc.children, got, tc.want)
		}
	}
}

// An element step whose existence alone a then-branch asks ends at the 'else'
// (ctaParser.closesBoolean) and is decided off the [Tally] whatever its type,
// so `p`, of an element-only type no value step reads, compiles there and holds
// exactly where a p child is reported. With 'else' not closing a
// BooleanExpr, `p` is read for its value and the compile declines.
func TestAssertionIfExprBranchAsksExistence(t *testing.T) {
	elems := asElems(map[xsd.QName]xsd.TypeDefinition{uq("p"): asComplex(t, "POnly", asElementContent(t, false))})
	const expr = "if (@t eq 'x') then p else false()"
	test, ok := CompileAssertionTest(asRecord(expr), seededTypes, xsd.ElementContent{}, asUses(t, map[string]string{"t": "string"}), elems)
	if !ok {
		t.Fatalf("CompileAssertionTest(%q): declined, want compiled", expr)
	}
	x := asValues(t, asTyped{uq("t"), "string", "x"})
	if !test.Evaluate(backend(), seededTypes, x, asNoChildren, acTally(test, acPath("p")), ValueBinding{}) {
		t.Errorf("Evaluate(%q) over a p child = false, want true", expr)
	}
	if test.Evaluate(backend(), seededTypes, x, asNoChildren, acTally(test), ValueBinding{}) {
		t.Errorf("Evaluate(%q) with no p child = true, want false", expr)
	}
}

// Only the selected branch is evaluated (xpath20.md §3.8: the expression
// "ignores (does not raise) any dynamic errors encountered in the
// else-expression"), so err:FOAR0001 of an xs:integer `1 div 0` in the other
// branch is never raised and the IfExpr holds. In the selected branch, or in
// the test, the error is the expression's, which fn:not propagates rather than
// inverting: each fn:not row is false, not true. An xs:date test has no
// ·effective boolean value· and raises err:FORG0006 (§2.4.3). With ctaIf.eval
// evaluating both branches and raising for either, the first two rows fail;
// with it reading a raised test as false, the last two rows fail; with it
// reading a raised then-branch as false, the fn:not row over that branch fails.
func TestAssertionIfExprEvaluatesOneBranch(t *testing.T) {
	for _, tc := range []struct {
		expr string
		want bool
	}{
		{"if (true()) then true() else (1 div 0 = 0)", true},
		{"if (false()) then (1 div 0 = 0) else true()", true},
		{"if (true()) then (1 div 0 = 0) else true()", false},
		{"not(if (true()) then (1 div 0 = 0) else true())", false},
		{"not(if (false()) then true() else (1 div 0 = 0))", false},
		{"not(if (1 div 0 = 0) then true() else false())", false},
		{"not(if (xs:date('2000-01-01')) then true() else false())", false},
	} {
		if got := aiCompile(t, tc.expr).Evaluate(backend(), seededTypes, asValues(t), asNoChildren, nil, ValueBinding{}); got != tc.want {
			t.Errorf("Evaluate(%q) = %v, want %v", tc.expr, got, tc.want)
		}
	}
}

// A child either branch or the test reads for its value is reported, and a
// path either branch counts is tallied, whichever branch an evaluation will
// select: with ctaIf.readsChild or ctaIf.counted reading the test alone, the
// branch rows fail.
func TestAssertionIfExprReadsEveryBranch(t *testing.T) {
	for _, tc := range []struct {
		expr string
		read xsd.QName
	}{
		{"if (a = 1) then true() else false()", uq("a")},
		{"if (@t eq 'x') then b = 1 else true()", uq("b")},
		{"if (@t eq 'x') then true() else c = 1", uq("c")},
	} {
		if !aiCompile(t, tc.expr).ReadsChild(tc.read) {
			t.Errorf("ReadsChild(%s) over %q = false, want true", tc.read.Local, tc.expr)
		}
	}
	for _, expr := range []string{
		"if (@t eq 'x') then count(c) = 0 else true()",
		"if (@t eq 'x') then true() else count(c) = 0",
	} {
		if aiCompile(t, expr).Tally() == nil {
			t.Errorf("Tally over %q = nil, want a counter for c", expr)
		}
	}
}

// What an IfExpr's either branch holds is compiled whatever the test will
// select (xpath20.md §2.3.4: an expression is not rewritten to remove a
// static error), so an unknown function, a variable not in the static context
// (err:XPST0017, err:XPST0008) and a construct this engine declines in the
// branch never selected decline the {test} as they would anywhere — never a
// true one. A bare comma between ExprSingles is no branch (XPST0003), nor is a
// missing `then` or `else`. Inside a value predicate an IfExpr declines
// (ctaPredicateFacade.conditional), and so does one read as an item — a
// function argument or a parenthesized comparison operand — or standing as an
// AndExpr operand, which XPath's grammar does not admit. Guard: these decline
// today.
func TestAssertionIfExprDeclines(t *testing.T) {
	for _, expr := range []string{
		"if (true()) then true() else nosuch()",
		"if (false()) then nosuch() else true()",
		"if (true()) then true() else $nosuch = 1",
		"if (true()) then true() else a/b = 1",
		"if (true()) then true(), true() else true()",
		"if (true()) true() else true()",
		"if (true()) then true()",
		"if (true(), true()) then true() else true()",
		"count(a[if (. = 1) then true() else false()]) = 0",
		"a = 1 and if (true()) then true() else true()",
		"contains(if (@t) then 'x' else 'y', 'x')",
		"(if (@t) then 1 else 2) = 1",
	} {
		if _, ok := CompileAssertionTest(asRecord(expr), seededTypes, xsd.ElementContent{}, asUses(t, map[string]string{"t": "string"}), aiElems(t)); ok {
			t.Errorf("CompileAssertionTest(%q): compiled, want declined", expr)
		}
	}
}

// An assertions facet evaluates an IfExpr over `$value`
// (cvc-assertions-valid): `if ($value > 0) then $value mod 2 = 0 else true()`
// holds over 4 and over -3 and fails over 3, and an unknown function in the
// branch not taken declines it. Every evaluated row declines with
// ctaFacetFacade.conditional answering false.
func TestFacetIfExpr(t *testing.T) {
	intType := asBuiltin(t, "int")
	const test = "if ($value > 0) then $value mod 2 = 0 else true()"
	for _, tc := range []struct {
		test    string
		lexical string
		want    value.AssertionOutcome
	}{
		{test, "4", value.AssertionHolds},
		{test, "3", value.AssertionFails},
		{test, "-3", value.AssertionHolds},
		{"if ($value > 0) then true() else nosuch()", "4", value.AssertionDeclined},
	} {
		if got := FacetAssertions().Evaluate(backend(), seededTypes, intType, ctaExprRecord(tc.test, ""), fcValue(t, intType, tc.lexical)); got != tc.want {
			t.Errorf("FacetAssertions(%q) over %s = %d, want %d", tc.test, tc.lexical, got, tc.want)
		}
	}
}

// Guard: a Type Alternative's {test} declines an IfExpr, which ta-props-correct
// clause 2's grammar has no production for, wherever it stands — the
// test12.xsd shape among them, and one that calls no library function, so the
// decline is the IfExpr's own. CTATestStaticError answers nil for each: a
// decline is no static error. With ctaTypeAlternativeFacade.conditional
// answering true, the library-free rows compile.
func TestCompileCTATestDeclinesIfExpr(t *testing.T) {
	for _, expr := range []string{
		"if (@kind = 'square') then (@a = @b) else @c = 'x'",
		"(if (@kind = 'square') then @a = @b else @c = 'x')",
		"not(if (@kind = 'square') then @a = @b else @c = 'x')",
		"if (@kind = 'square') then (a = b and b = c and c = d) else true()",
	} {
		if _, ok := CompileCTATest(ctaExprRecord(expr, ""), seededTypes); ok {
			t.Errorf("CompileCTATest(%q): compiled, want declined", expr)
		}
		if err := CTATestStaticError(ctaExprRecord(expr, ""), seededTypes); err != nil {
			t.Errorf("CTATestStaticError(%q) = %v, want nil", expr, err)
		}
	}
}
