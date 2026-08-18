package xpath

import (
	"testing"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// ctaExpr builds an XPath Expression property record over expr, binding the
// prefix/namespace pairs as its {namespace bindings}. defaultNS, when
// non-empty, is its {default namespace} — which no test in this file expects
// to change an answer, because the CTA grammar never consults it.
func ctaExprRecord(expr, defaultNS string, bindings ...string) xsd.XPathExpression {
	var nb []xsd.NamespaceBinding
	for i := 0; i+1 < len(bindings); i += 2 {
		nb = append(nb, xsd.NewNamespaceBinding(bindings[i], bindings[i+1]))
	}
	var def *string
	if defaultNS != "" {
		def = &defaultNS
	}
	return xsd.NewXPathExpression(expr, nb, def, nil)
}

// uq is an ·expanded name· in no namespace, which is what every unprefixed
// attribute NameTest resolves to.
func uq(local string) xsd.QName { return xsd.QName{Local: local} }

// ctaAttrs is the [AttributeValue] over a fixed attribute set.
func ctaAttrs(m map[xsd.QName]string) AttributeValue {
	return func(name xsd.QName) (string, bool) {
		v, present := m[name]
		return v, present
	}
}

// seededTypes and seededBackend are the type knowledge and the value spaces
// every test here compiles and evaluates against: the builtin datatypes
// builtin.Seed produces, indexed by ·expanded name· exactly as a parsed
// schema's {type definitions} index them. They are built ONCE and together —
// Seed compiles each type's facets against the backend it is handed, so a
// compile and the evaluation that follows it must see the same instance.
//
// A failed Seed is a broken fixture rather than a failed assertion, and it
// would break every test in the file identically, so it panics here instead of
// reporting through a *testing.T none of these helpers holds.
var seededTypes, seededBackend = seedBuiltins()

// ctaTestTypes is the minimal [xsd.TypeResolver] the seeded builtins fill.
type ctaTestTypes map[xsd.QName]xsd.TypeDefinition

func (t ctaTestTypes) Type(name xsd.QName) (xsd.TypeDefinition, bool) {
	td, declared := t[name]
	return td, declared
}

func seedBuiltins() (ctaTestTypes, value.Backend) {
	b := strict.New()
	types, err := builtin.Seed(b)
	if err != nil {
		panic("builtin.Seed: " + err.Error())
	}
	resolver := make(ctaTestTypes, len(types))
	for _, st := range types {
		resolver[st.Name()] = st
	}
	return resolver, b
}

// backend is the value spaces the comparisons and the casts run in.
func backend() value.Backend { return seededBackend }

// compile compiles expr or fails the test, so an evaluation case never
// silently degrades into a decline case.
func compile(t *testing.T, expr string, bindings ...string) CTATest {
	t.Helper()
	c, ok := CompileCTATest(ctaExprRecord(expr, "", bindings...), seededTypes)
	if !ok {
		t.Fatalf("CompileCTATest(%q): declined, want compiled", expr)
	}
	return c
}

// TestCompileAdmitsRequiredSubset pins that every production [8]-[18] this
// engine evaluates parses: the two AttrName spellings ta-props-correct clause
// 2 admits, both Literal kinds, all six Comparators, the three BooleanExpr
// arms, and the or/and connectives with their nesting.
func TestCompileAdmitsRequiredSubset(t *testing.T) {
	for _, expr := range []string{
		"@kind",
		"@kind = 'book'",
		"@kind='book'",
		"attribute::kind = 'book'",
		"@kind != 'book'",
		"@n < 5",
		"@n <= 5",
		"@n > 5",
		"@n >= 5",
		"@n = 1.5",
		"@n = 1.5e2",
		"@n = .5",
		"'a' = 'a'",
		"1 = 1",
		"not(@kind = 'book')",
		"fn:not(@kind = 'book')",
		"(@kind = 'book')",
		"@a = 'x' or @b = 'y'",
		"@a = 'x' and @b = 'y'",
		"@a = 'x' and (@b = 'y' or @c = 'z')",
		"@a  =  'x'   or   @b = 'y'",
		"(: a comment :) @kind = 'book'",
		"@kind = 'it''s'",
		"@kind = \"it's\"",
		"@a.b = 'x'",
	} {
		if _, ok := CompileCTATest(ctaExprRecord(expr, "", "fn", ctaFunctionNS), seededTypes); !ok {
			t.Errorf("CompileCTATest(%q): declined, want compiled", expr)
		}
	}
}

// TestCompileResolvesUnprefixedFunctionInDefaultFunctionNamespace pins
// xpath-valid clause 2.2.4: a bare not(...) IS fn:not, with no binding needed,
// while a prefix bound elsewhere is a different function and declines.
func TestCompileResolvesUnprefixedFunctionInDefaultFunctionNamespace(t *testing.T) {
	if _, ok := CompileCTATest(ctaExprRecord("not(@k = 'x')", ""), seededTypes); !ok {
		t.Error("an unprefixed not(...) declined; it is fn:not by clause 2.2.4")
	}
	elsewhere := ctaExprRecord("p:not(@k = 'x')", "", "p", "http://example.com/p")
	if _, ok := CompileCTATest(elsewhere, seededTypes); ok {
		t.Error("p:not bound outside the F&O namespace compiled as fn:not")
	}
}

// TestCompileDeclines pins the withhold direction over its whole membership:
// text that is no Test production at all (which is what legal full XPath 2.0
// looks like here), the three §3.12.6 constructs this engine recognizes but
// does not evaluate, an unbound prefix, and a wildcard NameTest.
func TestCompileDeclines(t *testing.T) {
	for _, tc := range []struct{ expr, why string }{
		{"", "an empty {expression} is no Test production"},
		{"1 idiv string-length(@type) gt 0", "full XPath 2.0, outside the subset"},
		{"self::message", "an axis step the grammar has no production for"},
		{"@kind cast as xs:string", "ta-CastExpr's cast tail (#858)"},
		{"@kind cast as xs:string = 'x'", "ta-CastExpr's cast tail inside a comparison"},
		{"@n cast as xs:integer ?", "ta-CastExpr's optional occurrence indicator"},
		{"xs:int(@length) > xs:int(@width)", "ta-ConstructorFunction (#858)"},
		{"a:b('123')", "a QName '(' … ')' whose name is not fn:not is a constructor call"},
		{"@p:kind = 'x'", "a prefix with no binding is err:XPST0081"},
		{"p:not(@kind = 'x')", "an unbound prefix on the function name too"},
		{"@* = 'x'", "a wildcard NameTest"},
		{"@p:* = 'x'", "a prefixed wildcard NameTest"},
		{"@*:kind = 'x'", "a local-name wildcard NameTest"},
		{"@kind = 'x' extra", "trailing tokens are not part of a Test"},
		{"@kind = ", "a Comparator with no right operand"},
		{"(@kind = 'x'", "an unclosed parenthesis"},
		{"@kind = 'x", "an unclosed string literal"},
		{"(: unclosed", "an unclosed comment"},
		{"@n = -1", "no unary minus production reaches a Literal"},
		{"@a = 'x' or", "an 'or' with no right operand"},
	} {
		bindings := []string{"a", "http://example.com/a", "xs", xsd.XMLSchemaNS}
		if _, ok := CompileCTATest(ctaExprRecord(tc.expr, "", bindings...), seededTypes); ok {
			t.Errorf("CompileCTATest(%q): compiled, want declined (%s)", tc.expr, tc.why)
		}
	}
}

// TestEvaluateStringComparators pins all six operators over the untypedAtomic
// attribute against a string literal, which §3.5.2 rule 2.4 compares as
// xs:string and B.2 decides through the default (codepoint) collation.
func TestEvaluateStringComparators(t *testing.T) {
	attrs := ctaAttrs(map[xsd.QName]string{uq("k"): "book"})
	for _, tc := range []struct {
		expr string
		want bool
	}{
		{"@k = 'book'", true},
		{"@k = 'cd'", false},
		{"@k != 'cd'", true},
		{"@k != 'book'", false},
		{"@k < 'cd'", true},
		{"@k < 'book'", false},
		{"@k <= 'book'", true},
		{"@k > 'a'", true},
		{"@k >= 'book'", true},
		{"@k >= 'cd'", false},
	} {
		if got := compile(t, tc.expr).Evaluate(backend(), seededTypes, attrs); got != tc.want {
			t.Errorf("Evaluate(%q) = %v, want %v", tc.expr, got, tc.want)
		}
	}
}

// TestEvaluateNumericComparators pins §3.5.2 clause 2.1: an untypedAtomic
// operand meeting a numeric one is cast to xs:double, whatever the literal's
// own numeric type, and compared in that space — so "3" and "3.0" and "03" are
// one answer and not three lexical ones.
func TestEvaluateNumericComparators(t *testing.T) {
	attrs := ctaAttrs(map[xsd.QName]string{
		uq("n"):    "3",
		uq("wide"): "03.0",
		uq("bad"):  "three",
		uq("pad"):  "  3  ",
		uq("big"):  "10",
	})
	for _, tc := range []struct {
		expr string
		want bool
	}{
		{"@n = 3", true},
		{"@n = 3.0", true},
		{"@n = 3e0", true},
		{"@n != 4", true},
		{"@n < 5", true},
		{"@n < 3", false},
		{"@n <= 3", true},
		{"@n > 0", true},
		{"@n >= 4", false},
		// "10" sorts BEFORE "9" under the codepoint collation and after it
		// in xs:double, so these two fail for an evaluator that compared the
		// lexicals instead of casting.
		{"@big > 9", true},
		{"@big < 9", false},
		{"@wide = 3", true},
		{"@pad = 3", true},
		{"@bad = 3", false},
		{"@bad != 3", false},
		{"@bad < 3", false},
	} {
		if got := compile(t, tc.expr).Evaluate(backend(), seededTypes, attrs); got != tc.want {
			t.Errorf("Evaluate(%q) = %v, want %v", tc.expr, got, tc.want)
		}
	}
}

// TestEvaluateBadCastIsFalseNotDecline pins §3.12.4 clause 2 over its whole
// operator set: a cast that raises err:FORG0001 makes the {test} false rather
// than an error, and `!=` is false too — no pair of atomic values was formed,
// so nothing is unequal either.
func TestEvaluateBadCastIsFalseNotDecline(t *testing.T) {
	attrs := ctaAttrs(map[xsd.QName]string{uq("n"): "not a number"})
	for _, expr := range []string{"@n = 1", "@n != 1", "@n < 1", "@n >= 1"} {
		if compile(t, expr).Evaluate(backend(), seededTypes, attrs) {
			t.Errorf("Evaluate(%q) = true, want false", expr)
		}
	}
}

// TestEvaluateAbsentAttributeIsFalse pins the empty-sequence rule of §3.5.2:
// no pair can be formed against an absent attribute, so EVERY comparator is
// false — including `!=`, which a naive reading would make true.
func TestEvaluateAbsentAttributeIsFalse(t *testing.T) {
	attrs := ctaAttrs(map[xsd.QName]string{uq("other"): "x"})
	for _, expr := range []string{
		"@k = 'book'", "@k != 'book'", "@k < 'book'", "@k <= 'book'",
		"@k > 'book'", "@k >= 'book'", "@k = 1", "@k != 1",
	} {
		if compile(t, expr).Evaluate(backend(), seededTypes, attrs) {
			t.Errorf("Evaluate(%q) with no such attribute = true, want false", expr)
		}
	}
}

// TestEvaluateUnprefixedNameTestIsInNoNamespace pins xpath20.md:1521 against
// the trap it exists for: the record carries a {default namespace}, and the
// unprefixed NameTest still resolves to no namespace, because the attribute
// axis's principal node kind is never element (PRINCIPLES 15).
func TestEvaluateUnprefixedNameTestIsInNoNamespace(t *testing.T) {
	const ns = "http://example.com/d"
	c, ok := CompileCTATest(ctaExprRecord("@k = 'book'", ns), seededTypes)
	if !ok {
		t.Fatal("CompileCTATest: declined")
	}
	if !c.Evaluate(backend(), seededTypes, ctaAttrs(map[xsd.QName]string{uq("k"): "book"})) {
		t.Error("an unprefixed NameTest did not match the no-namespace attribute")
	}
	inDefault := ctaAttrs(map[xsd.QName]string{{Space: ns, Local: "k"}: "book"})
	if c.Evaluate(backend(), seededTypes, inDefault) {
		t.Error("an unprefixed NameTest matched an attribute in the {default namespace}")
	}
}

// TestEvaluatePrefixedNameTest pins that a bound prefix resolves against the
// record's {namespace bindings}.
func TestEvaluatePrefixedNameTest(t *testing.T) {
	const ns = "http://example.com/c2"
	c := compile(t, "@c2:min = 1", "c2", ns)
	if !c.Evaluate(backend(), seededTypes, ctaAttrs(map[xsd.QName]string{{Space: ns, Local: "min"}: "1"})) {
		t.Error("a prefixed NameTest did not match its expanded name")
	}
	if c.Evaluate(backend(), seededTypes, ctaAttrs(map[xsd.QName]string{uq("min"): "1"})) {
		t.Error("a prefixed NameTest matched a no-namespace attribute")
	}
}

// TestEvaluateAttributeAxisForm pins ta-props-correct clause 2.2: the
// unabbreviated attribute-axis spelling is the same AttrName as clause 2.1's
// abbreviation, and decides the same way.
func TestEvaluateAttributeAxisForm(t *testing.T) {
	attrs := ctaAttrs(map[xsd.QName]string{uq("k"): "book"})
	for _, expr := range []string{"attribute::k = 'book'", "attribute :: k = 'book'"} {
		if !compile(t, expr).Evaluate(backend(), seededTypes, attrs) {
			t.Errorf("Evaluate(%q) = false, want true", expr)
		}
	}
	if compile(t, "attribute::k = 'cd'").Evaluate(backend(), seededTypes, attrs) {
		t.Error("attribute::k = 'cd' = true, want false")
	}
}

// TestEvaluateConnectives pins [9], [10], [11]'s parenthesised arm and the
// fn:not the [12] production is fixed to, including that grouping changes the
// answer (so the test would notice an evaluator that ignored the parens).
func TestEvaluateConnectives(t *testing.T) {
	attrs := ctaAttrs(map[xsd.QName]string{uq("a"): "x", uq("b"): "y", uq("c"): "z"})
	for _, tc := range []struct {
		expr string
		want bool
	}{
		{"@a = 'x' or @b = 'no'", true},
		{"@a = 'no' or @b = 'no'", false},
		{"@a = 'x' and @b = 'y'", true},
		{"@a = 'x' and @b = 'no'", false},
		{"not(@a = 'x')", false},
		{"not(@a = 'no')", true},
		{"fn:not(@a = 'no')", true},
		{"@a = 'no' and @b = 'no' or @c = 'z'", true},
		{"@a = 'no' and (@b = 'no' or @c = 'z')", false},
		{"not(@a = 'no' or @b = 'no')", true},
	} {
		if got := compile(t, tc.expr, "fn", ctaFunctionNS).Evaluate(backend(), seededTypes, attrs); got != tc.want {
			t.Errorf("Evaluate(%q) = %v, want %v", tc.expr, got, tc.want)
		}
	}
}

// TestEvaluateRaisedErrorFalsifiesTheWholeTest pins key-cta-ta-select clause 2
// (§3.12.4) at the granularity its subject fixes: "the {test} is treated as if
// it had evaluated (without error) to false" — the {test}, not the node that
// raised. fn:not is a function, and xpath20.md §3.6 makes it raise its
// operand's error rather than absorb it, so no number of fn:not calls turns a
// raised error into a true {test}.
//
// The two error sources are separate: @n's failed cast to xs:double is
// err:FORG0001 (a dynamic error), and a string literal against a numeric one
// has no xpath20.md B.2 row at all, which is err:XPTY0004 (a type error).
// Clause 2 names both.
func TestEvaluateRaisedErrorFalsifiesTheWholeTest(t *testing.T) {
	attrs := ctaAttrs(map[xsd.QName]string{uq("n"): "abc", uq("k"): "x"})
	for _, tc := range []struct {
		expr string
		want bool
		why  string
	}{
		{"@n = 5", false, "err:FORG0001: a PRESENT @n that does not cast to xs:double"},
		{"not(@n = 5)", false, "fn:not raises the same error; it does not invert it"},
		{"not(not(@n = 5))", false, "two fn:not calls propagate it twice"},
		{"not(@n != 5)", false, "every comparator reaches the same failed cast"},
		{"not(@n < 5)", false, "including the ordering ones"},
		{"'a' = 5", false, "err:XPTY0004: no B.2 row for a string against a numeric"},
		{"not('a' = 5)", false, "the type error propagates through fn:not too"},
		{"not(not('a' = 5))", false, "and stays raised however deep the nesting"},
		// The discriminating rows: fn:not still inverts an answer that was
		// DECIDED, so the fix cannot be "not(...) is always false".
		{"not(@missing = 5)", true, "an absent attribute forms no pair — false, not an error"},
		{"not(@k = 'no')", true, "a decided false inverts"},
		{"not(@k = 'x')", false, "a decided true inverts"},
	} {
		if got := compile(t, tc.expr).Evaluate(backend(), seededTypes, attrs); got != tc.want {
			t.Errorf("Evaluate(%q) = %v, want %v (%s)", tc.expr, got, tc.want, tc.why)
		}
	}
}

// TestEvaluateRaisedErrorUnderConnectives pins xpath20.md §3.6's truth tables
// over an operand that RAISES, read with XPath 1.0 compatibility mode false
// (xpath-valid clause 2.2.1). Every bare row is the answer this evaluator
// already gave before the error became a state of its own, so a landing that
// moves one has changed something it should not have; the fn:not rows are
// where the tables become observable, because only there does the difference
// between "false" and "raised" survive to the {test}.
func TestEvaluateRaisedErrorUnderConnectives(t *testing.T) {
	attrs := ctaAttrs(map[xsd.QName]string{uq("n"): "abc", uq("k"): "x"})
	for _, tc := range []struct {
		expr string
		want bool
		cell string
	}{
		{"@n = 5 or @k = 'x'", true, "error or true: the cell permits either, and true is the standing answer"},
		{"@k = 'x' or @n = 5", true, "true or error: same cell, other order"},
		{"@n = 5 or @k = 'no'", false, "error or false: the error, which clause 2 makes false"},
		{"@n = 5 and @k = 'x'", false, "error and true: the error"},
		{"@k = 'no' and @n = 5", false, "false and error: the false decides, and the raising operand is never evaluated"},
		{"@n = 5 and @k = 'no'", false, "error and false: the cell permits either, and this evaluator takes the false"},
		{"@k = 'x' and @n = 5", false, "true and error: the error"},
		{"not(@n = 5 or @k = 'no')", false, "the or raised, so fn:not raises: NOT true"},
		{"not(@k = 'x' and @n = 5)", false, "the and raised, so fn:not raises: NOT true"},
		{"not(@n = 5 or @k = 'x')", false, "the or is a decided true, which inverts"},
		{"not(@k = 'no' and @n = 5)", true, "the and is a decided false, which inverts"},
		{"not(@n = 5 and @k = 'no')", true, "error and false: this evaluator takes the cell's false, so it inverts"},
	} {
		if got := compile(t, tc.expr).Evaluate(backend(), seededTypes, attrs); got != tc.want {
			t.Errorf("Evaluate(%q) = %v, want %v (%s)", tc.expr, got, tc.want, tc.cell)
		}
	}
}

// TestEvaluateEffectiveBooleanValue pins [11]'s third arm with its Comparator
// ABSENT: a bare ValueExpr in a boolean position takes its ·effective boolean
// value· (xpath20.md §2.4.3), which for an AttrName is presence and for a
// Literal is fn:boolean's rules 4 and 5.
func TestEvaluateEffectiveBooleanValue(t *testing.T) {
	attrs := ctaAttrs(map[xsd.QName]string{uq("a"): "x", uq("empty"): ""})
	for _, tc := range []struct {
		expr string
		want bool
	}{
		{"@a", true},
		{"@empty", true}, // rule 2: a node, whatever its string value
		{"@missing", false},
		{"@a and @empty", true},
		{"@a and @missing", false},
		{"not(@missing)", true},
		{"'x'", true},
		{"''", false},
		{"1", true},
		{"0", false},
		{"0.0", false},
		{"0e0", false},
		{"1.5", true},
	} {
		if got := compile(t, tc.expr).Evaluate(backend(), seededTypes, attrs); got != tc.want {
			t.Errorf("Evaluate(%q) = %v, want %v", tc.expr, got, tc.want)
		}
	}
}

// TestEvaluateLiteralOnlyComparisons pins the pairs with no untypedAtomic
// operand: two literals of one space compare in it, a decimal literal meeting
// a double one promotes, and a string literal against a numeric one is
// err:XPTY0004 — which §3.12.4 clause 2 decides as false rather than
// declining.
func TestEvaluateLiteralOnlyComparisons(t *testing.T) {
	attrs := ctaAttrs(nil)
	for _, tc := range []struct {
		expr string
		want bool
	}{
		{"'a' = 'a'", true},
		{"'a' = 'b'", false},
		{"'a' < 'b'", true},
		{"1 = 1", true},
		{"1 = 1.0", true},
		{"1 = 1e0", true},
		{"1 < 2", true},
		{"1.5 > 1", true},
		{"'1' = 1", false},
		{"'1' != 1", false},
		{"1 = 'a'", false},
	} {
		if got := compile(t, tc.expr).Evaluate(backend(), seededTypes, attrs); got != tc.want {
			t.Errorf("Evaluate(%q) = %v, want %v", tc.expr, got, tc.want)
		}
	}
}

// TestEvaluateStringLiteralEscapes pins xpath20.md [74]: the quote character
// appears inside a literal only doubled, and denotes one such character.
func TestEvaluateStringLiteralEscapes(t *testing.T) {
	attrs := ctaAttrs(map[xsd.QName]string{uq("k"): "it's"})
	for _, expr := range []string{"@k = 'it''s'", "@k = \"it's\""} {
		if !compile(t, expr).Evaluate(backend(), seededTypes, attrs) {
			t.Errorf("Evaluate(%q) = false, want true", expr)
		}
	}
	if compile(t, "@k = 'it''''s'").Evaluate(backend(), seededTypes, attrs) {
		t.Error("a doubled pair was not folded to one quote")
	}
}

// TestEvaluateZeroTestDecides pins that the zero CTATest — which no successful
// CompileCTATest produces — still decides rather than panicking.
func TestEvaluateZeroTestDecides(t *testing.T) {
	var zero CTATest
	if zero.Evaluate(backend(), seededTypes, ctaAttrs(nil)) {
		t.Error("the zero CTATest evaluated true")
	}
}

// TestNumericLiteralType pins which builtin datatype each NumericLiteral kind
// carries, because it is what decides whether two literals are compared as
// xs:decimal or promoted to xs:double.
func TestNumericLiteralType(t *testing.T) {
	known := compileTypes(t)
	for _, tc := range []struct {
		text string
		want string
	}{
		{"1", "decimal"},
		{"1.5", "decimal"},
		{".5", "decimal"},
		{"1e0", "double"},
		{"1.5E-2", "double"},
	} {
		if got := known.literal(tc.text).Name(); got != ctaBuiltin(tc.want) {
			t.Errorf("literal(%q) = %v, want xs:%s", tc.text, got, tc.want)
		}
	}
}

// compileTypes is the compile-time type knowledge over the seeded builtins.
func compileTypes(t *testing.T) ctaTypes {
	t.Helper()
	known, ok := ctaResolveTypes(seededTypes)
	if !ok {
		t.Fatal("ctaResolveTypes over the seeded builtins: declined")
	}
	return known
}

// TestComparisonType pins §3.5.2's casting rules and B.1's promotions as a
// table, so the type a pair is compared in is checked directly and not only
// through an evaluation that could agree by accident. It re-derives the table
// the deleted ctaComparisonSpace pinned, over datatypes rather than over the
// three-valued space that function could name.
func TestComparisonType(t *testing.T) {
	known := compileTypes(t)
	attr := ctaAttr{name: uq("a")}
	str := ctaLiteral{text: "x", st: known.str}
	dec := ctaLiteral{text: "1", st: known.decimal}
	dbl := ctaLiteral{text: "1e0", st: known.double}
	for _, tc := range []struct {
		name        string
		left, right ctaValue
		want        string
		admitted    bool
	}{
		{"two attributes are clause 1's xs:string", attr, attr, "string", true},
		{"attribute vs string literal is clause 2.4", attr, str, "string", true},
		{"attribute vs decimal literal is clause 2.1", attr, dec, "double", true},
		{"decimal literal vs attribute is clause 2.1", dec, attr, "double", true},
		{"attribute vs double literal is clause 2.1", attr, dbl, "double", true},
		{"two decimal literals stay xs:decimal", dec, dec, "decimal", true},
		{"decimal vs double literal promotes", dec, dbl, "double", true},
		{"two string literals stay xs:string", str, str, "string", true},
		{"string vs numeric literal has no B.2 row", str, dec, "", false},
	} {
		got, admitted := known.comparison(tc.left, tc.right)
		if admitted != tc.admitted {
			t.Errorf("%s: admitted = %v, want %v", tc.name, admitted, tc.admitted)
			continue
		}
		if admitted && got.Name() != ctaBuiltin(tc.want) {
			t.Errorf("%s: comparison type = %v, want xs:%s", tc.name, got.Name(), tc.want)
		}
	}
}
