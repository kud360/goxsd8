package xpath

import (
	"testing"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// fcValue maps lexical against st, or fails the test.
func fcValue(t *testing.T, st *xsd.SimpleType, lexical string) value.Value {
	t.Helper()
	v, err := value.ValidateLexical(backend(), seededTypes, st, lexical, nil, FacetAssertions())
	if err != nil {
		t.Fatalf("mapping %q against %s: %v", lexical, st.Name(), err)
	}
	return v
}

// FacetAssertions decides an assertions facet's {test} over `$value` alone
// (cvc-assertions-valid clauses 1.1 and 1.4): `$value eq 100` holds for 100
// and fails for 5, `$value = 'x'` holds for "x" and fails for "y". A read of
// the context item, which clause 1.2 makes absent, raises err:XPDY0002 and
// FAILS — `.`, fn:not over it (the error propagates rather than inverting), an
// attribute step, a child-axis step and a rooted path alike. Each such row
// declines with the ctaFacetFacade method that builds its node answering
// false; `not(.)` and `not(@a)` hold with the ctaNoContextItem arm of
// ctaEffectiveBoolean.eval removed, and `not(. = 'x')` with the one of
// ctaItemOf removed.
func TestFacetAssertionsDecideTheValue(t *testing.T) {
	intType, str := asBuiltin(t, "int"), asBuiltin(t, "string")
	for _, tc := range []struct {
		test    string
		st      *xsd.SimpleType
		lexical string
		want    value.AssertionOutcome
	}{
		{"$value eq 100", intType, "100", value.AssertionHolds},
		{"$value eq 100", intType, "5", value.AssertionFails},
		{"$value = 'x'", str, "x", value.AssertionHolds},
		{"$value = 'x'", str, "y", value.AssertionFails},
		{".", str, "x", value.AssertionFails},
		{"not(.)", str, "x", value.AssertionFails},
		{". = 'x'", str, "x", value.AssertionFails},
		{"not(. = 'x')", str, "x", value.AssertionFails},
		{"@a", str, "x", value.AssertionFails},
		{"not(@a)", str, "x", value.AssertionFails},
		{"a = 'x'", str, "x", value.AssertionFails},
		{"/root = 'present'", str, "x", value.AssertionFails},
		{"//root", str, "x", value.AssertionFails},
		{"$value = 'x' or .", str, "x", value.AssertionHolds},
	} {
		got := FacetAssertions().Evaluate(backend(), seededTypes, tc.st, ctaExprRecord(tc.test, ""), fcValue(t, tc.st, tc.lexical))
		if got != tc.want {
			t.Errorf("Evaluate(%q, %q) = %d, want %d", tc.test, tc.lexical, got, tc.want)
		}
	}
}

// FacetAssertions declines a {test} outside the grammar CompileAssertionTest
// admits — arithmetic over a non-numeric `$value`, a call to a function
// outside the string and sequence core, a variable other than `$value`, the
// abbreviated parent step — and every {test} of a union's own assertions
// facet, whose `$value` is typed by an ·active basic member· the evaluator
// is not handed (dt-xdmrep clause 4), even one that reads nothing: the `'a'
// = 'a'` row holds with Evaluate's union check removed.
func TestFacetAssertionsDecline(t *testing.T) {
	str := asBuiltin(t, "string")
	union := asUnion(t)
	types := asTypesWith(union)
	for _, tc := range []struct {
		test string
		st   *xsd.SimpleType
	}{
		{"$value mod 2 = 0", str},
		{"upper-case($value) = 'X'", str},
		{"$other = 'x'", str},
		{".. = 'x'", str},
		{"$value = 'x'", union},
		{"'a' = 'a'", union},
	} {
		v := fcValue(t, str, "x")
		if got := FacetAssertions().Evaluate(backend(), types, tc.st, ctaExprRecord(tc.test, ""), v); got != value.AssertionDeclined {
			t.Errorf("Evaluate(%q over %s) = %d, want declined", tc.test, tc.st.Name(), got)
		}
	}
}

// `.` is the facet façade's alone: a Type Alternative's {test} and an
// assertion's still decline it, where the context item is E.
func TestContextItemDeclinesOutsideTheFacet(t *testing.T) {
	for _, expr := range []string{".", ". = 'x'", "not(.)"} {
		if _, ok := CompileCTATest(ctaExprRecord(expr, ""), seededTypes); ok {
			t.Errorf("CompileCTATest(%q): compiled, want declined", expr)
		}
		if _, ok := CompileAssertionTest(ctaExprRecord(expr, ""), seededTypes, xsd.SimpleContent{SimpleType: asBuiltin(t, "string")}, asUses(t, nil), asNoElems); ok {
			t.Errorf("CompileAssertionTest(%q): compiled, want declined", expr)
		}
	}
}
