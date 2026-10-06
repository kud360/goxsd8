package xpath

import (
	"testing"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// The fixtures below drive fn:count over an argument that is no path
// (ctaCountedItems) and fn:distinct-values (ctaDistinctValues), each over
// `$value` on both the assertion and the facet façade.

// vcCase is one {test} decided over `$value` bound to lexical under st.
type vcCase struct {
	expr    string
	st      *xsd.SimpleType
	lexical string
	want    bool
}

// vcAssertion compiles tc.expr for an E whose simple {content type} is tc.st,
// failing the test where it declines — a decline would read as false and pass
// every false row vacuously — and evaluates it with `$value` bound to
// tc.lexical. The test counts no path, so it has no Tally to hand over.
func vcAssertion(t *testing.T, types xsd.TypeResolver, tc vcCase) bool {
	t.Helper()
	test, ok := CompileAssertionTest(asRecord(tc.expr), types, xsd.SimpleContent{SimpleType: tc.st}, asUses(t, nil), asNoElems)
	if !ok {
		t.Fatalf("CompileAssertionTest(%q) over %s: declined, want compiled", tc.expr, tc.st.Name().Local)
	}
	if test.Tally() != nil {
		t.Errorf("(%q).Tally() = non-nil, want nil: `$value` is counted off its items", tc.expr)
	}
	v, err := value.ValidateLexical(backend(), types, tc.st, tc.lexical, nil, FacetAssertions())
	if err != nil {
		t.Fatalf("mapping %q against %s: %v", tc.lexical, tc.st.Name().Local, err)
	}
	return test.Evaluate(backend(), types, asValues(t), asNoChildren, nil, BindValue(Typed(v)))
}

// vcFacet decides tc.expr as an assertions facet's {test} over tc.lexical
// under tc.st, failing the test where it declines, on vcAssertion's terms.
func vcFacet(t *testing.T, types xsd.TypeResolver, tc vcCase) bool {
	t.Helper()
	v, err := value.ValidateLexical(backend(), types, tc.st, tc.lexical, nil, FacetAssertions())
	if err != nil {
		t.Fatalf("mapping %q against %s: %v", tc.lexical, tc.st.Name().Local, err)
	}
	got := FacetAssertions().Evaluate(backend(), types, tc.st, asRecord(tc.expr), v)
	if got == value.AssertionDeclined {
		t.Fatalf("FacetAssertions().Evaluate(%q) over %s: declined, want decided", tc.expr, tc.st.Name().Local)
	}
	return got == value.AssertionHolds
}

// vcDecides runs each case on both façades.
func vcDecides(t *testing.T, types xsd.TypeResolver, cases []vcCase) {
	t.Helper()
	for _, tc := range cases {
		if got := vcAssertion(t, types, tc); got != tc.want {
			t.Errorf("assertion %q over %q = %v, want %v", tc.expr, tc.lexical, got, tc.want)
		}
		if got := vcFacet(t, types, tc); got != tc.want {
			t.Errorf("facet %q over %q = %v, want %v", tc.expr, tc.lexical, got, tc.want)
		}
	}
}

// fn:count over `$value` is the number of items Datatypes dt-xdmrep clause 3
// makes a list value's XDM representation (xpath-functions.md §15.4.1): three
// for "1 2 3", two for "1 2", none for the empty list, and one for an atomic
// value. fn:distinct-values drops every item eq to an earlier one (§15.1.6),
// so `1 1 2` has two distinct items of three and `1 2 3` three of three.
// `count(())` is 0. Every row but the `$value = 3` guard declines on both
// façades, and fails, with ctaParser.countsItems answering false.
func TestCountAndDistinctValuesOverValue(t *testing.T) {
	list := asList(t, "IntList", ctaBuiltin("int"))
	intType := asBuiltin(t, "int")
	vcDecides(t, asTypesWith(list), []vcCase{
		{"count($value) le 2", list, "1 2 3", false},
		{"count($value) le 2", list, "1 2", true},
		{"count($value) eq count(distinct-values($value))", list, "1 1 2", false},
		{"count($value) eq count(distinct-values($value))", list, "1 2 3", true},
		{"count(distinct-values($value)) eq 2", list, "1 1 2", true},
		{"count($value) eq 0", list, "", true},
		{"count(distinct-values($value)) eq 0", list, "", true},
		{"count($value) eq 1", intType, "5", true},
		{"count(()) eq 0", intType, "5", true},
		{"count(()) eq 1", intType, "5", false},
		{"$value = 3", list, "1 2 3", true},
	})
}

// fn:distinct-values compares by eq (xpath-functions.md §15.1.6): 0 and -0 are
// one xs:double, 1 and 1.0 one xs:decimal, NaN and NaN one item although
// `NaN eq NaN` is false (the NaN rows fail with ctaNaN answering false), a
// date without a timezone equals the same date at Z, the implicit timezone,
// and no other — over a user restriction of xs:date too, whose rows fail with
// ctaHoldsPair asked in that item type rather than its primitive, a name
// ctaDateTimeFamily does not hold — and strings under the codepoint collation,
// so "a" and "A" are two.
func TestDistinctValuesComparesByEq(t *testing.T) {
	doubles := asList(t, "DoubleList", ctaBuiltin("double"))
	decimals := asList(t, "DecimalList", ctaBuiltin("decimal"))
	dates := asList(t, "DateList", ctaBuiltin("date"))
	strs := asList(t, "StringList", ctaBuiltin("string"))
	myDate := csRestriction(t, "MyDate", ctaBuiltin("date"))
	myDates := asList(t, "MyDateList", myDate.Name())
	vcDecides(t, asTypesWith(doubles, decimals, dates, strs, myDate, myDates), []vcCase{
		{"count(distinct-values($value)) eq 1", doubles, "0 -0", true},
		{"count(distinct-values($value)) eq 1", decimals, "1 1.0", true},
		{"count(distinct-values($value)) eq 1", doubles, "NaN NaN", true},
		{"count(distinct-values($value)) eq 2", doubles, "NaN 1 NaN", true},
		{"count(distinct-values($value)) eq 1", dates, "2008-01-01 2008-01-01Z", true},
		{"count(distinct-values($value)) eq 2", dates, "2008-01-01 2008-01-01+01:00", true},
		{"count(distinct-values($value)) eq 1", myDates, "2008-01-01 2008-01-01Z", true},
		{"count(distinct-values($value)) eq 2", myDates, "2008-01-01 2008-01-01+01:00", true},
		{"count(distinct-values($value)) eq 2", strs, "a A a", true},
	})
}

// fn:distinct-values' result is a sequence of atomic values every operator
// reads: a general comparison is existential over it, its ·effective boolean
// value· is fn:boolean's over the items kept — true for one non-zero number,
// false for zero, err:FORG0006 for two or more, which fn:not propagates — and
// fn:string over one item is that item. The ·effective boolean value· rows
// fail with the ctaDistinctValues arm of ctaEffectiveBoolean.eval removed.
func TestDistinctValuesResult(t *testing.T) {
	list := asList(t, "IntList", ctaBuiltin("int"))
	strs := asList(t, "StringList", ctaBuiltin("string"))
	vcDecides(t, asTypesWith(list, strs), []vcCase{
		{"distinct-values($value) = 2", list, "1 2 2", true},
		{"distinct-values($value) = 3", list, "1 2 2", false},
		{"distinct-values($value)", list, "1 1", true},
		{"distinct-values($value)", list, "0 0", false},
		{"distinct-values($value)", list, "1 2", false},
		{"not(distinct-values($value))", list, "1 2", false},
		{"not(distinct-values($value))", list, "0 0", true},
		{"string(distinct-values($value)) = 'a'", strs, "a a", true},
	})
}

// `$value` over a ·special· {simple type definition} is one xs:untypedAtomic
// value, or none for the zero binding (cvc-assertion clause 2.3.2), and over
// any {content type} that is not simple the empty sequence: fn:count counts 1,
// 0 and 0, and fn:distinct-values keeps the one item, compared as xs:string
// (§15.1.6). The untyped rows fail with ctaSequenceLength's ctaDistinctValues
// arm removed, which counts an untyped operand as statically empty.
func TestCountValueOverSpecialAndNonSimpleContent(t *testing.T) {
	anySimple := xsd.SimpleContent{SimpleType: asBuiltin(t, "anySimpleType")}
	for _, tc := range []struct {
		expr    string
		content xsd.ContentType
		bound   ValueBinding
		want    bool
	}{
		{"count($value) eq 1", anySimple, BindValue(Untyped("x")), true},
		{"count(distinct-values($value)) eq 1", anySimple, BindValue(Untyped("x")), true},
		{"distinct-values($value) = 'x'", anySimple, BindValue(Untyped("x")), true},
		{"distinct-values($value)", anySimple, BindValue(Untyped("")), false},
		{"count($value) eq 0", anySimple, ValueBinding{}, true},
		{"count(distinct-values($value)) eq 0", anySimple, ValueBinding{}, true},
		{"count($value) eq 0", xsd.ElementContent{}, ValueBinding{}, true},
		{"count(distinct-values($value)) eq 0", xsd.EmptyContent{}, ValueBinding{}, true},
	} {
		test, ok := CompileAssertionTest(asRecord(tc.expr), seededTypes, tc.content, asUses(t, nil), asNoElems)
		if !ok {
			t.Fatalf("CompileAssertionTest(%q): declined, want compiled", tc.expr)
		}
		if got := test.Evaluate(backend(), seededTypes, asValues(t), asNoChildren, nil, tc.bound); got != tc.want {
			t.Errorf("Evaluate(%q) = %v, want %v", tc.expr, got, tc.want)
		}
	}
}

// A union facet's `$value` is one value under its ·active basic member·
// (dt-xdmrep clause 4), so fn:count counts 1 whichever member validates it.
func TestCountValueOverAUnionFacet(t *testing.T) {
	date, integer := ctaBuiltin("date"), ctaBuiltin("integer")
	dateOrInt := fcUnion(t, "DateOrInt", date, integer)
	for _, lexical := range []string{"5", "2008-01-01"} {
		st := fcAssertedUnion(t, dateOrInt, "count($value) eq 1 and count(distinct-values($value)) eq 1", date, integer)
		_, err := value.ValidateLexical(backend(), asTypesWith(dateOrInt), st, lexical, nil, FacetAssertions())
		if got := fcOutcome(t, err); got != value.AssertionHolds {
			t.Errorf("count($value) over %q against a union = %d, want holds (err %v)", lexical, got, err)
		}
	}
}

// fn:count over an argument that is no path counts its items whatever they
// are: a count of a count is one item, whose own path the Tally keys. Its
// counted operand reads a child's value where it holds a value step, so
// ReadsChild reports it; `count($value)` reads none.
func TestCountOfAnOperandKeysWhatItCounts(t *testing.T) {
	acDecides(t, "count(count(e1)) eq 1", true)
	acCounters(t, "count(count(e1)) eq 1", 1)
	test, ok := CompileAssertionTest(asRecord("count(string(a)) eq 1"), seededTypes, xsd.ElementContent{}, asUses(t, nil), asChildTypes(t))
	if !ok {
		t.Fatal("CompileAssertionTest(count(string(a)) eq 1): declined, want compiled")
	}
	if !test.ReadsChild(uq("a")) {
		t.Error("(count(string(a)) eq 1).ReadsChild(a) = false, want true")
	}
	list := asList(t, "IntList", ctaBuiltin("int"))
	test, ok = CompileAssertionTest(asRecord("count($value) le 2"), asTypesWith(list), xsd.SimpleContent{SimpleType: list}, asUses(t, nil), asNoElems)
	if !ok {
		t.Fatal("CompileAssertionTest(count($value) le 2): declined, want compiled")
	}
	if test.ReadsChild(uq("value")) || test.Tally() != nil {
		t.Error("count($value) le 2 reads a child or keys a Tally, want neither")
	}
}

// The forms outside this slice still decline: fn:distinct-values with a
// collation argument (§7.3.1) or none, `count(.)`, and each of the new forms
// under a Type Alternative's {test}, whose grammar holds no fn:count
// (§3.12.6 clause 3) — and on the facet, the paths fn:count counts off a
// Tally (TestCountDeclinesOutsideTheAssertionFacade).
func TestCountAndDistinctValuesDecline(t *testing.T) {
	list := asList(t, "IntList", ctaBuiltin("int"))
	types := asTypesWith(list)
	for _, expr := range []string{
		"count(distinct-values($value, 'http://www.w3.org/2005/xpath-functions/collation/codepoint')) eq 1",
		"count(distinct-values()) eq 0",
		"count(.) eq 1",
	} {
		if _, ok := CompileAssertionTest(asRecord(expr), types, xsd.SimpleContent{SimpleType: list}, asUses(t, nil), asNoElems); ok {
			t.Errorf("CompileAssertionTest(%q): compiled, want declined", expr)
		}
		v := fcValue(t, list, "1 2")
		if got := FacetAssertions().Evaluate(backend(), types, list, asRecord(expr), v); got != value.AssertionDeclined {
			t.Errorf("FacetAssertions().Evaluate(%q) = %d, want declined", expr, got)
		}
	}
	for _, expr := range []string{"count(()) = 0", "count('a') = 1", "distinct-values(@a) = 'x'", "count(distinct-values(@a)) = 1"} {
		if _, ok := CompileCTATest(ctaExprRecord(expr, ""), seededTypes); ok {
			t.Errorf("CompileCTATest(%q): compiled, want declined", expr)
		}
	}
}
