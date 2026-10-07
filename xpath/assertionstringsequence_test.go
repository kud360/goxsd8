package xpath

import (
	"slices"
	"testing"
	"time"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// ssElems is E's child element types for the string-sequence rows: r is
// xs:string and n is xs:integer.
func ssElems(t *testing.T) ElementTypes {
	t.Helper()
	return asElems(map[xsd.QName]xsd.TypeDefinition{uq("r"): asBuiltin(t, "string"), uq("n"): asBuiltin(t, "integer")})
}

// A general comparison's operand may be §3.3.1's comma sequence over
// StringLiterals, `('a', 'b', 'c')`, and the comparison is existential over its
// items (xpath20.md §3.5.2): `r = ('a', 'b', 'c')` holds over a child r holding
// b and is false over z, in either operand order. `r != ('a', 'b')` holds over
// a, since a != b, so `!=` is never the negation of `=`. An empty operand — no
// r child — makes both `=` and `!=` false. `('a')` is a one-item sequence, so
// `r != ('a')` is false over a. Against an xs:integer operand the items stay
// xs:string: `$value = ('1', '2')` and `n = ('1', '2')` are err:XPTY0004 (B.2
// has no xs:integer and xs:string row), false under fn:not too, and never a
// match through a cast of the literal; `.` is E's string value, an
// xs:untypedAtomic cast to xs:string (§3.5.2 clause 2.2.4), so `. = ('1', '2')`
// holds over xs:integer content 1 and is false over 3. Every row compiles
// first, so a decline cannot pass a false row. Every row declines with
// sequenceLength answering integerSequenceLength alone; the two type error
// rows without fn:not hold with ctaStaticOf's ctaStringSequence arm removed,
// and every row that holds but `not(r = ('a', 'b'))` fails with the item
// switch's arm removed.
func TestAssertionStringSequences(t *testing.T) {
	elems := ssElems(t)
	r := func(lexical string) []asChild { return []asChild{{uq("r"), "string", lexical, false}} }
	n := []asChild{{uq("n"), "integer", "1", false}}
	for _, tc := range []struct {
		expr     string
		children []asChild
		want     bool
	}{
		{"r = ('a', 'b', 'c')", r("b"), true},
		{"r = ('a', 'b', 'c')", r("z"), false},
		{"('a', 'b') = r", r("b"), true},
		{"('a', 'b') = r", r("z"), false},
		{"r != ('a', 'b')", r("a"), true},
		{"r != ('a')", r("a"), false},
		{"('a') = r", r("a"), true},
		{"('a') = r", r("z"), false},
		{"r = ('a', 'b')", nil, false},
		{"r != ('a', 'b')", nil, false},
		{"r = ('a', 'a')", r("a"), true},
		{"not(r = ('a', 'b'))", r("z"), true},
		{"r = ('black wins', 'white wins', 'draw')", r("stalemate"), false},
		{"n = ('1', '2')", n, false},
		{"not(n = ('1', '2'))", n, false},
	} {
		t.Run(tc.expr, func(t *testing.T) {
			test, ok := CompileAssertionTest(asRecord(tc.expr), seededTypes, xsd.ElementContent{}, asUses(t, nil), elems)
			if !ok {
				t.Fatalf("CompileAssertionTest(%q): declined, want compiled", tc.expr)
			}
			if got := test.Evaluate(backend(), seededTypes, asElem, asValues(t), asChildren(t, tc.children...), nil, ValueBinding{}, time.Time{}); got != tc.want {
				t.Errorf("Evaluate(%q) over %v = %v, want %v", tc.expr, tc.children, got, tc.want)
			}
		})
	}
	integer := asBuiltin(t, "integer")
	for _, tc := range []struct {
		expr, lexical string
		want          bool
	}{
		{"$value = ('1', '2')", "1", false},
		{"not($value = ('1', '2'))", "1", false},
		{". = ('1', '2')", "1", true},
		{". = ('1', '2')", "3", false},
	} {
		if got := cxEval(t, tc.expr, seededTypes, integer, asBind(t, integer, tc.lexical)); got != tc.want {
			t.Errorf("Evaluate(%q) over xs:integer %q = %v, want %v", tc.expr, tc.lexical, got, tc.want)
		}
	}
}

// stringSequence keeps its members in written order, with no dedupe: `r =
// ('b', 'a', 'a')` holds three texts, b, a and a. It fails with the members
// stored in reverse order.
func TestStringSequenceKeepsWrittenOrder(t *testing.T) {
	expr := "r = ('b', 'a', 'a')"
	test, ok := CompileAssertionTest(asRecord(expr), seededTypes, xsd.ElementContent{}, asUses(t, nil), ssElems(t))
	if !ok {
		t.Fatalf("CompileAssertionTest(%q): declined, want compiled", expr)
	}
	cmp, isCompare := test.root.(ctaCompare)
	if !isCompare {
		t.Fatalf("CompileAssertionTest(%q).root = %T, want ctaCompare", expr, test.root)
	}
	seq, isSeq := cmp.right.(ctaStringSequence)
	if !isSeq {
		t.Fatalf("CompileAssertionTest(%q) right operand = %T, want ctaStringSequence", expr, cmp.right)
	}
	if want := []string{"b", "a", "a"}; !slices.Equal(seq.texts, want) {
		t.Errorf("CompileAssertionTest(%q) texts = %q, want %q", expr, seq.texts, want)
	}
}

// What stays declined (ctaParser.integerSequence's GAP(xpath)): a sequence
// mixing a StringLiteral with an IntegerLiteral, a nested parenthesis, a
// trailing comma, a sequence anywhere but a general comparison's operand — a
// function argument, a value comparison's operand, a whole {test} — and the
// empty parenthesis. A Type Alternative's {test} and a value predicate decline
// every sequence. An assertions facet's evaluates one: `$value = ('a', 'b')`
// holds on a and fails on z. The Type Alternative rows compile with
// ctaTypeAlternativeFacade.constructsSequences true, the value-predicate row
// with ctaPredicateFacade.constructsSequences true, and the facet rows decline
// with ctaFacetFacade.constructsSequences false.
func TestStringSequencesDecline(t *testing.T) {
	elems := ssElems(t)
	for _, expr := range []string{
		"r = ('a', 1)",
		"r = (1, 'a')",
		"r = (('a'))",
		"r = ('a',)",
		"r = (, 'a')",
		"exists(('a', 'b'))",
		"r eq ('a', 'b')",
		"('a', 'b') eq r",
		"('a', 'b')",
		"r = ()",
	} {
		if _, ok := CompileAssertionTest(asRecord(expr), seededTypes, xsd.ElementContent{}, asUses(t, nil), elems); ok {
			t.Errorf("CompileAssertionTest(%q): compiled, want declined", expr)
		}
	}
	for _, expr := range []string{"@a = ('x', 'y')", "('x', 'y') = @a"} {
		if _, ok := CompileCTATest(ctaExprRecord(expr, ""), seededTypes); ok {
			t.Errorf("CompileCTATest(%q): compiled, want declined", expr)
		}
	}
	if _, ok := vpCompile(t, "count(white[. = ('a', 'b')]) = 1", asBuiltin(t, "string")); ok {
		t.Error("CompileAssertionTest(count(white[. = ('a', 'b')]) = 1): compiled, want declined")
	}
	str := asBuiltin(t, "string")
	for _, tc := range []struct {
		lexical string
		want    value.AssertionOutcome
	}{
		{"a", value.AssertionHolds},
		{"z", value.AssertionFails},
	} {
		if got := FacetAssertions(time.Time{}).Evaluate(backend(), seededTypes, str, ctaExprRecord("$value = ('a', 'b')", ""), fcValue(t, str, tc.lexical)); got != tc.want {
			t.Errorf("FacetAssertions().Evaluate($value = ('a', 'b')) over %s = %d, want %d", tc.lexical, got, tc.want)
		}
	}
}
