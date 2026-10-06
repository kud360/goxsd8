package xpath

import (
	"testing"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// The fixtures below drive the context item `.` of an assertion over simple
// content (ctaContextAtom) and the integer sequences a general comparison's
// operand may be (ctaIntegerRanges).

// cxEval compiles expr for an E with simple content of type st, read through
// types, or fails the test, and evaluates it under bound.
func cxEval(t *testing.T, expr string, types xsd.TypeResolver, st *xsd.SimpleType, bound ValueBinding) bool {
	t.Helper()
	test, ok := CompileAssertionTest(asRecord(expr), types, xsd.SimpleContent{SimpleType: st}, asUses(t, nil), asNoElems)
	if !ok {
		t.Fatalf("CompileAssertionTest(%q) over %s content: declined, want compiled", expr, st.Name().Local)
	}
	return test.Evaluate(backend(), types, asValues(t), asNoChildren, nil, bound)
}

// `.` over simple content atomizes to ONE xs:untypedAtomic value, E's string
// value (cvc-assertion clause 2.3.1's Note), and never to `$value`'s typed
// value: over xs:integer content "0030" `.` is "0030" — four characters,
// equal to the string '0030' and to the number 30 (cast to xs:double against a
// numeric, xpath20.md §3.5.2 clause 2.1) — while `$value` is 30. A value
// comparison casts `.` to xs:string (§3.5.1 step 4), so `. eq 4` is the
// err:XPTY0004 of xs:string against xs:decimal, false under fn:not too, where
// `$value eq 4` holds. Arithmetic casts it to xs:double (§3.4), so
// `. mod 2 = 0` holds on 4 and fails on 5. The zero-argument string functions
// read the same string (xpath-functions.md §2.3, §7.4.4, §7.4.5). Every row
// reading `.` declines with ctaAssertionFacade.contextItem declining. With
// contextItem answering `$value`'s ctaValueVar, `. eq 4` holds, `. eq '4'`,
// `string-length(.) = 4` and `. = '0030'` fail, and `string-length() = 4`
// declines.
func TestAssertionContextItemIsTheStringValue(t *testing.T) {
	integer := asBuiltin(t, "integer")
	for _, tc := range []struct {
		expr, lexical string
		want          bool
	}{
		{". mod 2 = 0", "4", true},
		{". mod 2 = 0", "5", false},
		{"$value eq 4", "4", true},
		{". eq 4", "4", false},
		{"not(. eq 4)", "4", false},
		{". eq '4'", "4", true},
		{"string-length(.) = 4", "0030", true},
		{". = '0030'", "0030", true},
		{". = 30", "0030", true},
		{"$value = 30", "0030", true},
		{"string-length() = 4", "0030", true},
		{"string() = '0030'", "0030", true},
		{"normalize-space() = '0030'", " 0030 ", true},
		{"string() = ' 0030 '", " 0030 ", true},
		{"contains(., '03')", "0030", true},
		{". cast as xs:integer = 30", "0030", true},
		{"xs:integer(.) + 1 = 31", "0030", true},
		{"distinct-values(.) = '0030'", "0030", true},
	} {
		if got := cxEval(t, tc.expr, seededTypes, integer, asBind(t, integer, tc.lexical)); got != tc.want {
			t.Errorf("Evaluate(%q) over %q = %v, want %v", tc.expr, tc.lexical, got, tc.want)
		}
	}
}

// `.` reads the binding's string value whatever `$value` is: an invalid E,
// whose `$value` is the empty sequence (cvc-assertion clause 2.3.2), still has
// its string value, so `. = 5` holds where `$value = 5` fails; a ·nilled· E,
// the zero ValueBinding, has the zero-length string, which is one item and
// never the empty sequence (xpath-datamodel §6.2.4), so `.` equals the
// zero-length string literal and `$value`'s general comparison against it,
// over no item, fails. Over a list `.` is one item, the whole string "1 2",
// where `$value` is two; over a union, whose `$value` declines, `.` compiles
// and reads the string. With ctaContextAtomItem reading `$value`'s binding
// instead, `. = 5` and `not(. = 5)` over the invalid E, `.` against the
// zero-length string over the ·nilled· one, the list's `. = '1 2'` and
// string-length rows and the union row fail.
func TestAssertionContextItemOverEveryBinding(t *testing.T) {
	integer := asBuiltin(t, "integer")
	for _, tc := range []struct {
		expr  string
		bound ValueBinding
		want  bool
	}{
		{". = 5", BindValue("5", nil), true},
		{"$value = 5", BindValue("5", nil), false},
		{". = ''", ValueBinding{}, true},
		{"string-length() = 0", ValueBinding{}, true},
		{"$value = ''", ValueBinding{}, false},
		{". = 5", BindValue("five", nil), false},
		{"not(. = 5)", BindValue("five", nil), false},
	} {
		if got := cxEval(t, tc.expr, seededTypes, integer, tc.bound); got != tc.want {
			t.Errorf("Evaluate(%q) = %v, want %v", tc.expr, got, tc.want)
		}
	}
	list := asList(t, "IntList", ctaBuiltin("int"))
	types := asTypesWith(list)
	lv, err := value.ValidateLexical(backend(), types, list, "1 2", nil, FacetAssertions())
	if err != nil {
		t.Fatalf("mapping 1 2 against the list: %v", err)
	}
	listed := BindValue("1 2", Typed(lv))
	for _, tc := range []struct {
		expr string
		want bool
	}{
		{". = '1 2'", true},
		{"count($value) = 2", true},
		{"string-length(.) = 3", true},
		{". = 1", false},
	} {
		if got := cxEval(t, tc.expr, types, list, listed); got != tc.want {
			t.Errorf("Evaluate(%q) over the list %q = %v, want %v", tc.expr, "1 2", got, tc.want)
		}
	}
	union := asUnion(t)
	if !cxEval(t, ". = 'abc'", asTypesWith(union), union, BindValue("abc", nil)) {
		t.Error("Evaluate(. = 'abc') over union content = false, want true")
	}
}

// `.` declines where it is a node rather than an atom — the whole operand of
// an ·effective boolean value·, fn:not over one, fn:exists, fn:empty and
// fn:count (the last a path, ctaParser.countPath) — and under every {content
// type} that is not simple, whose string value is its descendants' text. A
// Type Alternative's {test} declines it whatever its position. The
// non-simple-content rows compile with ctaAssertionFacade.contextItem
// answering ctaContextAtom under any content; the node-position rows but
// count(.)'s compile with the ctaContextAtom checks in ctaParser.booleanExpr
// and ctaParser.presenceCall removed.
func TestAssertionContextItemDeclines(t *testing.T) {
	simple := xsd.SimpleContent{SimpleType: asBuiltin(t, "string")}
	for _, expr := range []string{".", "not(.)", "exists(.)", "empty(.)", "not(exists(.))", "count(.) eq 1", ". and true()"} {
		if _, ok := CompileAssertionTest(asRecord(expr), seededTypes, simple, asUses(t, nil), asNoElems); ok {
			t.Errorf("CompileAssertionTest(%q) over simple content: compiled, want declined", expr)
		}
	}
	for _, tc := range []struct {
		content xsd.ContentType
		why     string
	}{
		{nil, "no"},
		{xsd.EmptyContent{}, "empty"},
		{asElementContent(t, false), "element-only"},
		{asElementContent(t, true), "mixed"},
	} {
		for _, expr := range []string{". = 'x'", "string-length() = 0", "string(.) = ''"} {
			if _, ok := CompileAssertionTest(asRecord(expr), seededTypes, tc.content, asUses(t, nil), asNoElems); ok {
				t.Errorf("CompileAssertionTest(%q) over %s content: compiled, want declined", expr, tc.why)
			}
		}
	}
	for _, expr := range []string{". = 'x'", "string-length(.) = 1"} {
		if _, ok := CompileCTATest(ctaExprRecord(expr, ""), seededTypes); ok {
			t.Errorf("CompileCTATest(%q): compiled, want declined", expr)
		}
	}
}

// A general comparison's operand may be xpath20.md §3.3.1's integer sequence:
// `1 to 10`, bare or parenthesized, and the comma sequence `(1 to 10, 20, 30)`,
// either operand. The comparison is existential over its items (§3.5.2), so
// `. = (1 to 10, 20, 30)` holds on 20 and fails on 80, and `!=` holds where
// any item differs. `1 to 0` and `3 to 1` are the empty sequence, which forms
// no pair, so the comparison is false and its negation true — against an
// xs:string operand too, `not(string(.) = (1 to 0))`, where a non-empty
// xs:integer sequence would be err:XPTY0004, which fn:not propagates. Each row
// declines with ctaFacade.constructsSequences false on the assertion façade,
// and the xs:string row fails with ctaParser.integerSequence building an
// itemless ctaIntegerRanges in place of ctaEmptyValue.
func TestAssertionIntegerSequences(t *testing.T) {
	integer := asBuiltin(t, "integer")
	for _, tc := range []struct {
		expr, lexical string
		want          bool
	}{
		{". = (1 to 10, 20, 30)", "20", true},
		{". = (1 to 10, 20, 30)", "80", false},
		{". = (1 to 10, 20, 30)", "7", true},
		{"$value = (1 to 10, 20, 30)", "30", true},
		{"$value = 1 to 10", "4", true},
		{"$value = 1 to 10", "11", false},
		{"(1 to 10, 20, 30) = .", "20", true},
		{"1 to 3 = $value", "3", true},
		{"(5) = .", "5", true},
		{". = (1 to 0)", "0", false},
		{"not(. = (1 to 0))", "0", true},
		{"not($value = (3 to 1))", "2", true},
		{"not(string(.) = (1 to 0))", "2", true},
		{"not(string(.) = (1 to 3))", "2", false},
		{". != (1 to 3)", "2", true},
		{". != (2)", "2", false},
		{". > (1 to 3)", "2", true},
		{"$value = 1 to 3 and . = '2'", "2", true},
		{". mod 2 = 0 and . = (1 to 10, 20, 30)", "20", true},
	} {
		if got := cxEval(t, tc.expr, seededTypes, integer, asBind(t, integer, tc.lexical)); got != tc.want {
			t.Errorf("Evaluate(%q) over %q = %v, want %v", tc.expr, tc.lexical, got, tc.want)
		}
	}
}

// What stays declined (ctaParser.integerSequence's GAP(xpath)): a sequence in
// any position but a general comparison's operand, a range whose operand is
// not an IntegerLiteral, a nested or empty parenthesis, a member of another
// kind, an IntegerLiteral beyond int64, and a sequence longer than
// ctaMaxSequenceLength. A Type Alternative's {test} declines every sequence,
// and an assertions facet's evaluates one, `$value = (1 to 10, 20)` holding on
// 20 and failing on 15. The two rows longer than ctaMaxSequenceLength compile
// with the length bound removed, the Type Alternative rows with
// ctaTypeAlternativeFacade.constructsSequences true, and the facet rows
// decline with ctaFacetFacade.constructsSequences false.
func TestIntegerSequencesDecline(t *testing.T) {
	simple := xsd.SimpleContent{SimpleType: asBuiltin(t, "integer")}
	for _, expr := range []string{
		". eq (1 to 3)",
		"$value eq 1 to 3",
		". = 1 to .",
		". = 1 to $value",
		". = 1 + 1 to 3",
		". = (1 to 3) + 1",
		"(1 to 3)",
		"exists(1 to 3)",
		"count((1 to 3)) = 3",
		". = ((1 to 3))",
		". = ()",
		". = (1 to 3, $value)",
		". = (1.5 to 3)",
		". = (1, 'a')",
		". = 1 to 99999999999999999999",
		". = 1 to 5000",
		". = (1 to 4000, 1 to 100)",
		". = (1 to 3,)",
		"1 to 3 eq .",
		"1 to 3",
	} {
		if _, ok := CompileAssertionTest(asRecord(expr), seededTypes, simple, asUses(t, nil), asNoElems); ok {
			t.Errorf("CompileAssertionTest(%q): compiled, want declined", expr)
		}
	}
	if _, ok := CompileAssertionTest(asRecord(". = 1 to 4096"), seededTypes, simple, asUses(t, nil), asNoElems); !ok {
		t.Error("CompileAssertionTest(. = 1 to 4096): declined, want compiled at ctaMaxSequenceLength")
	}
	for _, expr := range []string{"@a = (1 to 3)", "@a = 1 to 3", "(1 to 3) = @a"} {
		if _, ok := CompileCTATest(ctaExprRecord(expr, ""), seededTypes); ok {
			t.Errorf("CompileCTATest(%q): compiled, want declined", expr)
		}
	}
	intType := asBuiltin(t, "int")
	for _, tc := range []struct {
		lexical string
		want    value.AssertionOutcome
	}{
		{"20", value.AssertionHolds},
		{"15", value.AssertionFails},
	} {
		if got := FacetAssertions().Evaluate(backend(), seededTypes, intType, ctaExprRecord("$value = (1 to 10, 20)", ""), fcValue(t, intType, tc.lexical)); got != tc.want {
			t.Errorf("FacetAssertions().Evaluate($value = (1 to 10, 20)) over %s = %d, want %d", tc.lexical, got, tc.want)
		}
	}
}
