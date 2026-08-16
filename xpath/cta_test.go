package xpath

import (
	"testing"

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

// backend is the value spaces the numeric comparisons run in. Only xs:decimal
// and xs:double are ever asked for.
func backend() value.Backend { return strict.New() }

// compile compiles expr or fails the test, so an evaluation case never
// silently degrades into a decline case.
func compile(t *testing.T, expr string, bindings ...string) CTATest {
	t.Helper()
	c, ok := CompileCTATest(ctaExprRecord(expr, "", bindings...))
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
		if _, ok := CompileCTATest(ctaExprRecord(expr, "", "fn", ctaFunctionNS)); !ok {
			t.Errorf("CompileCTATest(%q): declined, want compiled", expr)
		}
	}
}

// TestCompileResolvesUnprefixedFunctionInDefaultFunctionNamespace pins
// xpath-valid clause 2.2.4: a bare not(...) IS fn:not, with no binding needed,
// while a prefix bound elsewhere is a different function and declines.
func TestCompileResolvesUnprefixedFunctionInDefaultFunctionNamespace(t *testing.T) {
	if _, ok := CompileCTATest(ctaExprRecord("not(@k = 'x')", "")); !ok {
		t.Error("an unprefixed not(...) declined; it is fn:not by clause 2.2.4")
	}
	elsewhere := ctaExprRecord("p:not(@k = 'x')", "", "p", "http://example.com/p")
	if _, ok := CompileCTATest(elsewhere); ok {
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
		{"@kind = 'x' extra", "trailing tokens are not part of a Test"},
		{"@kind = ", "a Comparator with no right operand"},
		{"(@kind = 'x'", "an unclosed parenthesis"},
		{"@kind = 'x", "an unclosed string literal"},
		{"(: unclosed", "an unclosed comment"},
		{"@n = -1", "no unary minus production reaches a Literal"},
		{"@a = 'x' or", "an 'or' with no right operand"},
	} {
		bindings := []string{"a", "http://example.com/a", "xs", xsd.XMLSchemaNS}
		if _, ok := CompileCTATest(ctaExprRecord(tc.expr, "", bindings...)); ok {
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
		if got := compile(t, tc.expr).Evaluate(backend(), attrs); got != tc.want {
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
		if got := compile(t, tc.expr).Evaluate(backend(), attrs); got != tc.want {
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
		if compile(t, expr).Evaluate(backend(), attrs) {
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
		if compile(t, expr).Evaluate(backend(), attrs) {
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
	c, ok := CompileCTATest(ctaExprRecord("@k = 'book'", ns))
	if !ok {
		t.Fatal("CompileCTATest: declined")
	}
	if !c.Evaluate(backend(), ctaAttrs(map[xsd.QName]string{uq("k"): "book"})) {
		t.Error("an unprefixed NameTest did not match the no-namespace attribute")
	}
	inDefault := ctaAttrs(map[xsd.QName]string{{Space: ns, Local: "k"}: "book"})
	if c.Evaluate(backend(), inDefault) {
		t.Error("an unprefixed NameTest matched an attribute in the {default namespace}")
	}
}

// TestEvaluatePrefixedNameTest pins that a bound prefix resolves against the
// record's {namespace bindings}.
func TestEvaluatePrefixedNameTest(t *testing.T) {
	const ns = "http://example.com/c2"
	c := compile(t, "@c2:min = 1", "c2", ns)
	if !c.Evaluate(backend(), ctaAttrs(map[xsd.QName]string{{Space: ns, Local: "min"}: "1"})) {
		t.Error("a prefixed NameTest did not match its expanded name")
	}
	if c.Evaluate(backend(), ctaAttrs(map[xsd.QName]string{uq("min"): "1"})) {
		t.Error("a prefixed NameTest matched a no-namespace attribute")
	}
}

// TestEvaluateAttributeAxisForm pins ta-props-correct clause 2.2: the
// unabbreviated attribute-axis spelling is the same AttrName as clause 2.1's
// abbreviation, and decides the same way.
func TestEvaluateAttributeAxisForm(t *testing.T) {
	attrs := ctaAttrs(map[xsd.QName]string{uq("k"): "book"})
	for _, expr := range []string{"attribute::k = 'book'", "attribute :: k = 'book'"} {
		if !compile(t, expr).Evaluate(backend(), attrs) {
			t.Errorf("Evaluate(%q) = false, want true", expr)
		}
	}
	if compile(t, "attribute::k = 'cd'").Evaluate(backend(), attrs) {
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
		if got := compile(t, tc.expr, "fn", ctaFunctionNS).Evaluate(backend(), attrs); got != tc.want {
			t.Errorf("Evaluate(%q) = %v, want %v", tc.expr, got, tc.want)
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
		if got := compile(t, tc.expr).Evaluate(backend(), attrs); got != tc.want {
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
		if got := compile(t, tc.expr).Evaluate(backend(), attrs); got != tc.want {
			t.Errorf("Evaluate(%q) = %v, want %v", tc.expr, got, tc.want)
		}
	}
}

// TestEvaluateStringLiteralEscapes pins xpath20.md [74]: the quote character
// appears inside a literal only doubled, and denotes one such character.
func TestEvaluateStringLiteralEscapes(t *testing.T) {
	attrs := ctaAttrs(map[xsd.QName]string{uq("k"): "it's"})
	for _, expr := range []string{"@k = 'it''s'", "@k = \"it's\""} {
		if !compile(t, expr).Evaluate(backend(), attrs) {
			t.Errorf("Evaluate(%q) = false, want true", expr)
		}
	}
	if compile(t, "@k = 'it''''s'").Evaluate(backend(), attrs) {
		t.Error("a doubled pair was not folded to one quote")
	}
}

// TestEvaluateZeroTestDecides pins that the zero CTATest — which no successful
// CompileCTATest produces — still decides rather than panicking.
func TestEvaluateZeroTestDecides(t *testing.T) {
	var zero CTATest
	if zero.Evaluate(backend(), ctaAttrs(nil)) {
		t.Error("the zero CTATest evaluated true")
	}
}

// TestNumericSpaceOfLiteral pins which value space each NumericLiteral kind
// puts a literal in, because it is what decides whether two literals are
// compared as xs:decimal or promoted to xs:double.
func TestNumericSpaceOfLiteral(t *testing.T) {
	for _, tc := range []struct {
		text string
		want ctaSpace
	}{
		{"1", ctaSpaceDecimal},
		{"1.5", ctaSpaceDecimal},
		{".5", ctaSpaceDecimal},
		{"1e0", ctaSpaceDouble},
		{"1.5E-2", ctaSpaceDouble},
	} {
		if got := ctaNumericSpace(tc.text); got != tc.want {
			t.Errorf("ctaNumericSpace(%q) = %v, want %v", tc.text, got, tc.want)
		}
	}
}

// TestComparisonSpace pins §3.5.2's casting rules as a table, so the space a
// pair is compared in is checked directly and not only through an evaluation
// that could agree by accident.
func TestComparisonSpace(t *testing.T) {
	attr := ctaAttr{name: uq("a")}
	str := ctaLiteral{text: "x", space: ctaSpaceString}
	dec := ctaLiteral{text: "1", space: ctaSpaceDecimal}
	dbl := ctaLiteral{text: "1e0", space: ctaSpaceDouble}
	for _, tc := range []struct {
		name        string
		left, right ctaValue
		want        ctaSpace
		admitted    bool
	}{
		{"two attributes are rule 1's xs:string", attr, attr, ctaSpaceString, true},
		{"attribute vs string literal is rule 2.4", attr, str, ctaSpaceString, true},
		{"attribute vs decimal literal is rule 2.1", attr, dec, ctaSpaceDouble, true},
		{"decimal literal vs attribute is rule 2.1", dec, attr, ctaSpaceDouble, true},
		{"attribute vs double literal is rule 2.1", attr, dbl, ctaSpaceDouble, true},
		{"two decimal literals stay xs:decimal", dec, dec, ctaSpaceDecimal, true},
		{"decimal vs double literal promotes", dec, dbl, ctaSpaceDouble, true},
		{"two string literals stay xs:string", str, str, ctaSpaceString, true},
		{"string vs numeric literal has no B.2 row", str, dec, ctaSpaceString, false},
	} {
		got, admitted := ctaComparisonSpace(tc.left, tc.right)
		if admitted != tc.admitted {
			t.Errorf("%s: admitted = %v, want %v", tc.name, admitted, tc.admitted)
			continue
		}
		if admitted && got != tc.want {
			t.Errorf("%s: space = %v, want %v", tc.name, got, tc.want)
		}
	}
}
