package xpath

import (
	"testing"
	"time"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// The fixtures below drive xpath20.md §3.4's arithmetic operators, which the
// assertion and facet façades admit and a Type Alternative's declines
// (ctaFacade.computes).

// aaUses is the attribute types every arithmetic row reads: i an xs:integer, m
// an xs:decimal, d an xs:double, f an xs:float, and u an xs:anySimpleType,
// whose typed value is xs:untypedAtomic.
func aaUses(t *testing.T) AttributeTypes {
	t.Helper()
	return asUses(t, map[string]string{"i": "integer", "m": "decimal", "d": "double", "f": "float", "g": "float", "u": "anySimpleType"})
}

// aaEval compiles expr for an element with empty content over aaUses and
// evaluates it over attrs, or fails the test.
func aaEval(t *testing.T, expr string, attrs ...asTyped) bool {
	t.Helper()
	return asCompile(t, expr, aaUses(t)).Evaluate(backend(), seededTypes, asValues(t, attrs...), asNoChildren, nil, ValueBinding{}, time.Time{})
}

// `$value mod 2 = 0` holds on an even value and is false — charged — on an odd
// one, through FacetAssertions (cvc-assertions-valid) and through
// CompileAssertionTest over simple content (cvc-assertion) alike. Every row
// fails with ctaAssertionFacade.computes and ctaFacetFacade.computes answering
// false: the facet rows decline, and the assertion rows decline at compile.
func TestArithmeticModOverValue(t *testing.T) {
	intType := asBuiltin(t, "int")
	for _, tc := range []struct {
		lexical string
		facet   value.AssertionOutcome
		holds   bool
	}{
		{"4", value.AssertionHolds, true},
		{"5", value.AssertionFails, false},
		{"-6", value.AssertionHolds, true},
	} {
		if got := FacetAssertions(time.Time{}).Evaluate(backend(), seededTypes, intType, ctaExprRecord("$value mod 2 = 0", ""), fcValue(t, intType, tc.lexical)); got != tc.facet {
			t.Errorf("FacetAssertions over %s = %d, want %d", tc.lexical, got, tc.facet)
		}
		test := asCompileFor(t, "$value mod 2 = 0", xsd.SimpleContent{SimpleType: intType}, asUses(t, nil))
		if got := test.Evaluate(backend(), seededTypes, asValues(t), asNoChildren, nil, asBind(t, intType, tc.lexical), time.Time{}); got != tc.holds {
			t.Errorf("CompileAssertionTest over %s: Evaluate = %v, want %v", tc.lexical, got, tc.holds)
		}
	}
}

// A zero divisor of an xs:integer or xs:decimal `div`, `idiv` or `mod` raises
// err:FOAR0001 (xpath-functions.md §6.2.4-6.2.6), and the raise is told apart
// from a false or an INF by fn:not, which propagates it: `not(@i div 0 = 1)` is
// false too. The xs:double contrast raises nothing — `@d div 0` is INF, so its
// fn:not row holds and `@d div 0 > 1e300` holds — and neither does `mod` by a
// double zero, which is NaN; `idiv` by any zero raises, an xs:double one
// included. An xs:untypedAtomic dividend is cast to xs:double (§3.4), so `@u div
// 0` is INF; `@i div 0.0` is xs:decimal division and raises, and `@i div 0e0`
// is xs:double division and does not. With any one of decimal's three zero
// checks removed, that operator's xs:decimal rows panic in big.Rat.Quo; with
// ctaArithItem answering an operator that raises with the empty sequence instead
// of ctaRaised, each fn:not row of a raising operator holds and fails.
func TestArithmeticDivisionByZero(t *testing.T) {
	i := asTyped{uq("i"), "integer", "4"}
	d := asTyped{uq("d"), "double", "4"}
	u := asTyped{uq("u"), "anySimpleType", "4"}
	for _, tc := range []struct {
		expr string
		attr asTyped
		want bool
	}{
		{"@i div 0 = 1", i, false},
		{"not(@i div 0 = 1)", i, false},
		{"not(@i idiv 0 = 1)", i, false},
		{"not(@i mod 0 = 1)", i, false},
		{"not(@i div 0.0 = 1)", i, false},
		{"not(@i div 0e0 = 1)", i, true},
		{"@i div 0e0 > 1e300", i, true},
		{"not(@d div 0 = 1)", d, true},
		{"@d div 0 > 1e300", d, true},
		{"not(@d mod 0 = 1)", d, true},
		{"not(@d idiv 0 = 1)", d, false},
		{"not(@u div 0 = 1)", u, true},
		{"@u div 0 > 1e300", u, true},
	} {
		if got := aaEval(t, tc.expr, tc.attr); got != tc.want {
			t.Errorf("Evaluate(%q) over %s=%q = %v, want %v", tc.expr, tc.attr.name.Local, tc.attr.lexical, got, tc.want)
		}
	}
}

// F&O §6.2's operator functions over xs:integer and xs:decimal: `*` binds
// tighter than `+` and both associate to the left (xpath20.md [13], [14]);
// `idiv` truncates toward zero and `mod` takes the dividend's sign (§6.2.5,
// §6.2.6, whose examples these are), an xs:double `mod` included, which is
// truncating and not IEEE 754's remainder: `5e0 mod 3` is 2, not -1; `div` of
// two integers is the exact xs:decimal where it terminates and
// ctaDecimalDivisionDigits digits where it does not. An xs:untypedAtomic
// operand is cast to xs:double (§3.4), and one that does not cast raises
// err:FORG0001, false under fn:not too. An absent attribute is the empty
// sequence, so the arithmetic over it is empty and no error: its comparison
// is false and the fn:not of it true.
func TestArithmeticOperators(t *testing.T) {
	for _, tc := range []struct {
		expr  string
		attrs []asTyped
		want  bool
	}{
		{"1 + 2 * 3 = 7", nil, true},
		{"10 - 4 - 3 = 3", nil, true},
		{"2 * 3 mod 4 = 2", nil, true},
		{"10 idiv 3 = 3", nil, true},
		{"@i idiv 3 = 1", []asTyped{{uq("i"), "integer", "-3"}}, false},
		{"@i idiv 3 + 1 = 0", []asTyped{{uq("i"), "integer", "-3"}}, true},
		{"@m idiv 3 + 1 = 0", []asTyped{{uq("m"), "decimal", "-3.5"}}, true},
		{"@i mod 3 + 1 = 0", []asTyped{{uq("i"), "integer", "-10"}}, true},
		{"@i mod 3 = 1", []asTyped{{uq("i"), "integer", "10"}}, true},
		{"@d mod 3 = 2", []asTyped{{uq("d"), "double", "5"}}, true},
		{"5 div 2 = 2.5", nil, true},
		{"1 div 3 = 0.333333333333333333", nil, true},
		{"2 div 3 = 0.666666666666666667", nil, true},
		{"@m * 2 = 3", []asTyped{{uq("m"), "decimal", "1.5"}}, true},
		{"@u + 1 = 3", []asTyped{{uq("u"), "anySimpleType", "2"}}, true},
		{"@u + 1 = 3", []asTyped{{uq("u"), "anySimpleType", "x"}}, false},
		{"not(@u + 1 = 3)", []asTyped{{uq("u"), "anySimpleType", "x"}}, false},
		{"@i + 1 = 1", nil, false},
		{"not(@i + 1 = 1)", nil, true},
		{"@i - 1", []asTyped{{uq("i"), "integer", "1"}}, false},
		{"@i - 1", []asTyped{{uq("i"), "integer", "2"}}, true},
		{"@i - 1 eq 1", []asTyped{{uq("i"), "integer", "2"}}, true},
	} {
		if got := aaEval(t, tc.expr, tc.attrs...); got != tc.want {
			t.Errorf("Evaluate(%q) over %v = %v, want %v", tc.expr, tc.attrs, got, tc.want)
		}
	}
}

// An xs:float operation is computed over the operands' xs:float values and
// rounded to xs:float once (ctaNumericKind.bits, ctaRounded). Parsing each
// xs:float lexical as an xs:double instead lands on a different product for
// these two, 2.0534678E0; skipping the xs:float rounding of an `idiv` quotient
// leaves 3E38 idiv 0.1 finite, where in xs:float it overflows to INF and
// raises err:FOAR0002 (F&O §6.2.5), which fn:not propagates.
func TestArithmeticFloatPrecision(t *testing.T) {
	f := asTyped{uq("f"), "float", "1.5149502"}
	g := asTyped{uq("g"), "float", "1.3554688"}
	if !aaEval(t, "@f * @g = xs:float('2.0534675')", f, g) {
		t.Error("Evaluate(@f * @g = xs:float('2.0534675')) over f=1.5149502, g=1.3554688 = false, want true")
	}
	big, tenth := asTyped{uq("f"), "float", "3E38"}, asTyped{uq("g"), "float", "0.1"}
	if aaEval(t, "not(@f idiv @g = 1)", big, tenth) {
		t.Error("Evaluate(not(@f idiv @g = 1)) over f=3E38, g=0.1 = true, want false (err:FOAR0002)")
	}
}

// B.1 rule 1.2 promotes an xs:decimal operand to xs:double "by casting", so
// `@m + @d` runs in xs:double and holds. Rule 1.1's xs:float to xs:double
// promotion is #889's and stays declined (ctaWider): an xs:float against an
// xs:double, and an xs:untypedAtomic — cast to xs:double by §3.4 — against an
// xs:float, both decline; xs:float against itself or an xs:decimal does not.
func TestArithmeticPromotion(t *testing.T) {
	if !aaEval(t, "@m + @d = 1.5e0", asTyped{uq("m"), "decimal", "0.5"}, asTyped{uq("d"), "double", "1"}) {
		t.Error("Evaluate(@m + @d = 1.5e0) over m=0.5, d=1 = false, want true")
	}
	if !aaEval(t, "@f + @m = xs:float('1.5')", asTyped{uq("f"), "float", "1"}, asTyped{uq("m"), "decimal", "0.5"}) {
		t.Error("Evaluate(@f + @m = xs:float('1.5')) over f=1, m=0.5 = false, want true")
	}
	for _, expr := range []string{"@f + @d = 1", "@d * @f = 1", "@u + @f = 1", "@f - @u = 1"} {
		if _, ok := CompileAssertionTest(asRecord(expr), seededTypes, xsd.EmptyContent{}, aaUses(t), asNoElems); ok {
			t.Errorf("CompileAssertionTest(%q): compiled, want declined (#889)", expr)
		}
	}
}

// The arithmetic façades decline what they do not compute: an operand outside
// the numeric primitives (B.2's duration and date/time rows, and the
// err:XPTY0004 of every other type), a unary sign, and a cast from an
// arithmetic result, which [16] ta-SimpleValue, the operand of both cast
// spellings, does not admit. A Type Alternative's {test} declines every
// arithmetic operator, word and symbol spellings alike: with
// ctaTypeAlternativeFacade.computes answering true, both CompileCTATest rows
// compile.
func TestArithmeticDeclines(t *testing.T) {
	uses := asUses(t, map[string]string{"s": "string", "dur": "dayTimeDuration", "i": "integer"})
	for _, expr := range []string{"@s + 1 = 2", "@dur + @dur = @dur", "-@i = 1", "+@i = 1", "xs:integer(@i div 2) = 1"} {
		if _, ok := CompileAssertionTest(asRecord(expr), seededTypes, xsd.EmptyContent{}, uses, asNoElems); ok {
			t.Errorf("CompileAssertionTest(%q): compiled, want declined", expr)
		}
	}
	for _, expr := range []string{"@a mod 2 = 0", "@a + 1 = 2"} {
		if _, ok := CompileCTATest(ctaExprRecord(expr, ""), seededTypes); ok {
			t.Errorf("CompileCTATest(%q): compiled, want declined", expr)
		}
	}
}

// An arithmetic node reads what its operands read: a typed child it holds is
// reported by ReadsChild, and an fn:count call under it is counted by the
// Tally — `count(inner) mod 2 = 0` holds over two inner children and is false
// over three. With ctaArith.readsChild or ctaArith.counted answering nothing
// for its operands, the matching row fails.
func TestArithmeticReadsItsOperands(t *testing.T) {
	test, ok := CompileAssertionTest(asRecord("n + 1 = 3"), seededTypes, xsd.EmptyContent{}, asUses(t, nil), asChildTypes(t))
	if !ok {
		t.Fatal("CompileAssertionTest(n + 1 = 3): declined, want compiled")
	}
	if !test.ReadsChild(uq("n")) {
		t.Error("(n + 1 = 3).ReadsChild(n) = false, want true")
	}
	if !test.Evaluate(backend(), seededTypes, asValues(t), asChildren(t, asChild{name: uq("n"), typ: "int", lexical: "2"}), nil, ValueBinding{}, time.Time{}) {
		t.Error("Evaluate(n + 1 = 3) over n=2 = false, want true")
	}

	counted := acCompile(t, asRecord("count(inner) mod 2 = 0"))
	for _, tc := range []struct {
		nodes []acNode
		want  bool
	}{
		{[]acNode{acEl(1, "inner"), acEl(1, "inner")}, true},
		{[]acNode{acEl(1, "inner"), acEl(1, "inner"), acEl(1, "inner")}, false},
	} {
		got := counted.Evaluate(backend(), seededTypes, asValues(t), asNoChildren, acTally(counted, tc.nodes...), ValueBinding{}, time.Time{})
		if got != tc.want {
			t.Errorf("Evaluate(count(inner) mod 2 = 0) over %d inner = %v, want %v", len(tc.nodes), got, tc.want)
		}
	}
}

// A facet {test} has no context item (cvc-assertions-valid clause 1.2), so
// arithmetic over `.` raises err:XPDY0002 and FAILS the facet rather than
// declining it (its Note). A `$value` of two items is err:XPTY0004 under
// arithmetic (§3.4), false under fn:not too; one of one item computes.
func TestFacetArithmeticRaises(t *testing.T) {
	intType := asBuiltin(t, "int")
	list := asList(t, "IntList", ctaBuiltin("int"))
	types := asTypesWith(list)
	listValue := func(lexical string) value.Value {
		t.Helper()
		v, err := value.ValidateLexical(backend(), types, list, lexical, nil, FacetAssertions(time.Time{}))
		if err != nil {
			t.Fatalf("mapping %q against the list: %v", lexical, err)
		}
		return v
	}
	for _, tc := range []struct {
		test string
		st   *xsd.SimpleType
		v    value.Value
		want value.AssertionOutcome
	}{
		{". + 1 = 2", intType, fcValue(t, intType, "1"), value.AssertionFails},
		{"not(. + 1 = 2)", intType, fcValue(t, intType, "1"), value.AssertionFails},
		{"$value + 1 = 2", list, listValue("1"), value.AssertionHolds},
		{"$value + 1 = 2", list, listValue("1 2"), value.AssertionFails},
		{"not($value + 1 = 2)", list, listValue("1 2"), value.AssertionFails},
	} {
		if got := FacetAssertions(time.Time{}).Evaluate(backend(), types, tc.st, ctaExprRecord(tc.test, ""), tc.v); got != tc.want {
			t.Errorf("Evaluate(%q over %s) = %d, want %d", tc.test, tc.st.Name(), got, tc.want)
		}
	}
}
