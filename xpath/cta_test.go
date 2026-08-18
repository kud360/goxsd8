package xpath

import (
	"testing"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
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
// looks like here), an unbound prefix, and a wildcard NameTest. The cast
// TARGETS that decline have their own test below, because they decline for a
// reason the grammar cannot see.
func TestCompileDeclines(t *testing.T) {
	for _, tc := range []struct{ expr, why string }{
		{"", "an empty {expression} is no Test production"},
		{"1 idiv string-length(@type) gt 0", "full XPath 2.0, outside the subset"},
		{"self::message", "an axis step the grammar has no production for"},
		{"a:b('123')", "a constructor call naming a type the {type definitions} do not declare"},
		{"(@type cast as xs:float)='float'", "[11]'s parenthesised arm is a BooleanExpr, which no Comparator may follow"},
		{"double('3' cast as float > 2)", "a constructor argument is a SimpleValue, never a comparison"},
		{"3 cast as 3", "a cast target is a QName, not a Literal"},
		{"cast as decimal 3", "'cast' where a SimpleValue must open [15]"},
		{"3 cast 'as' decimal", "the 'as' keyword is a name token, not a string literal"},
		{"6 > cast as decimal", "a Comparator's right operand is a ValueExpr"},
		{"3 cast as @a:kind > 1", "a cast target is a QName, not an AttrName"},
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

// TestCompileAdmitsCasts pins the three cast-shaped constructs of the required
// subset, which are the whole of what this issue's landing added to the
// admitted set: [15] ta-CastExpr's `cast as QName` tail, its `?` occurrence
// indicator, and [18] ta-ConstructorFunction — the last being §3.12.6 clause
// 3's reading of any QName '(' SimpleValue ')' whose name is not fn:not.
func TestCompileAdmitsCasts(t *testing.T) {
	for _, expr := range []string{
		"@kind cast as xs:string",
		"@kind cast as xs:string = 'x'",
		"@n cast as xs:integer ?",
		"@n cast as xs:integer? = 3",
		"xs:int(@length) > xs:int(@width)",
		"@length cast as xs:int > @width cast as xs:int",
		"xs:int(@length) = @width cast as xs:int",
		"xs:date('2026-01-01') = @d cast as xs:date",
		"not(@n cast as xs:integer > 3)",
	} {
		if _, ok := CompileCTATest(ctaExprRecord(expr, "", "xs", xsd.XMLSchemaNS), seededTypes); !ok {
			t.Errorf("CompileCTATest(%q): declined, want compiled", expr)
		}
	}
}

// ctaUserNS is the namespace of the two NON-builtin type definitions the cast
// target classification has to tell apart from a builtin.
const ctaUserNS = "http://example.com/a"

// ctaFixtureTypes is the seeded builtins plus a user-defined ATOMIC simple type
// and a COMPLEX type, both in ctaUserNS. Neither is a legal cast target, and
// the two are illegal for different reasons the classification keeps apart:
// a:Kind is valid XPath outside §3.12.6's required subset, a:Box is
// err:XPST0051.
func ctaFixtureTypes(t *testing.T) ctaTestTypes {
	t.Helper()
	types := make(ctaTestTypes, len(seededTypes)+2)
	for name, td := range seededTypes {
		types[name] = td
	}
	str, declared := seededTypes.Type(ctaBuiltin("string"))
	if !declared {
		t.Fatal("the seeded builtins hold no xs:string")
	}
	kind, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: ctaUserNS, Local: "Kind"},
		xsd.RestrictionDerivation{}, xsd.OwnedSimpleType{Definition: str.(*xsd.SimpleType)}, nil, nil)
	if err != nil {
		t.Fatalf("building the user-defined atomic type: %v", err)
	}
	types[kind.Name()] = kind
	box, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Space: ctaUserNS, Local: "Box"},
		xsd.QName{}, nil, xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("building the complex type: %v", err)
	}
	types[box.Name()] = box
	return types
}

// TestCompileClassifiesCastTargets pins the THREE-way split §3.12.6 clause 4
// and xpath20.md §3.10.2 make of a cast target, over both spellings:
//
//   - (a) a builtin atomic datatype is the required subset's own case and
//     compiles;
//   - (b) another in-scope atomic type is valid XPath outside the required
//     subset, which §3.12.6's Note licenses declining;
//   - (c) an unresolvable, complex, non-atomic or excluded target is
//     err:XPST0051/err:XPST0080, a STATIC error, which shares (b)'s decline
//     because it shares its consequence at the caller.
//
// The two declines are one encoding on purpose; what this test pins is that
// every member of both lands in it and that (a) does not.
func TestCompileClassifiesCastTargets(t *testing.T) {
	types := ctaFixtureTypes(t)
	bindings := []string{"a", ctaUserNS, "xs", xsd.XMLSchemaNS}
	for _, expr := range []string{
		"@k cast as xs:string",
		"@k cast as xs:integer",
		"xs:int(@k)",
		"@k cast as xs:date",
	} {
		if _, ok := CompileCTATest(ctaExprRecord(expr, "", bindings...), types); !ok {
			t.Errorf("CompileCTATest(%q): declined, want compiled — (a) is a builtin atomic datatype", expr)
		}
	}
	for _, tc := range []struct{ expr, why string }{
		{"@k cast as a:Kind", "(b) a user-defined atomic type is outside the required subset"},
		{"a:Kind(@k)", "(b) in the constructor spelling too"},
		{"@k cast as a:Box", "(c) a complex type is no atomic type: err:XPST0051"},
		{"@k cast as xs:nosuchtype", "(c) an unresolvable target: err:XPST0051"},
		{"@k cast as xs:anySimpleType", "(c) an ·absent· {variety}: err:XPST0051"},
		{"@k cast as xs:IDREFS", "(c) a list builtin is no atomic type: err:XPST0051"},
		{"@k cast as xs:anyAtomicType", "(c) excluded by name: err:XPST0080"},
		{"@k cast as xs:NOTATION", "(c) excluded by name: err:XPST0080"},
		{"@k cast as xs:QName", "context-dependent, declined under its own marker"},
		{"@k cast as Kind", "an unprefixed target is in the {default namespace}, which declares none"},
	} {
		if _, ok := CompileCTATest(ctaExprRecord(tc.expr, "", bindings...), types); ok {
			t.Errorf("CompileCTATest(%q): compiled, want declined (%s)", tc.expr, tc.why)
		}
	}
}

// TestCompileResolvesCastTargetInDefaultNamespace pins xpath20.md §3.10.2's
// one use of the {default namespace} in this grammar: "if the target type has
// no namespace prefix, it is considered to be in the default element/type
// namespace". The same unprefixed name declines against a record carrying no
// {default namespace}, so the test would notice a compiler that ignored the
// property.
func TestCompileResolvesCastTargetInDefaultNamespace(t *testing.T) {
	if _, ok := CompileCTATest(ctaExprRecord("@k cast as string", xsd.XMLSchemaNS), seededTypes); !ok {
		t.Error("an unprefixed cast target did not resolve in the {default namespace}")
	}
	if _, ok := CompileCTATest(ctaExprRecord("@k cast as string", ""), seededTypes); ok {
		t.Error("an unprefixed cast target resolved with no {default namespace} to take")
	}
}

// TestEvaluateCastToBuiltin pins that a cast DECIDES: the operand is validated
// against the target datatype and the comparison then runs in that type's
// value space rather than over lexicals.
func TestEvaluateCastToBuiltin(t *testing.T) {
	attrs := ctaAttrs(map[xsd.QName]string{
		uq("n"):      "4",
		uq("frac"):   "3.5",
		uq("length"): "10",
		uq("width"):  "9",
		uq("d"):      "2026-03-01",
		uq("k"):      "book",
	})
	for _, tc := range []struct {
		expr string
		want bool
		why  string
	}{
		{"@n cast as xs:integer > 3", true, "the Goal's own example"},
		{"@n cast as xs:integer > 4", false, "a decided false"},
		{"@n cast as xs:integer = 4", true, "and equality in the numeric space"},
		{"xs:int(@length) > xs:int(@width)", true, "10 > 9 numerically, though \"10\" sorts before \"9\""},
		{"@length cast as xs:int > @width cast as xs:int", true, "the same pair in the cast spelling"},
		{"xs:int(@length) = @width cast as xs:int", false, "the two spellings mix in one comparison"},
		{"@d cast as xs:date = xs:date('2026-03-01')", true, "a constructor over a StringLiteral"},
		{"@d cast as xs:date > xs:date('2026-01-01')", true, "B.2 gives xs:date all six operators"},
		{"@d cast as xs:date < xs:date('2026-01-01')", false, "and orders them the right way round"},
		{"@k cast as xs:string = 'book'", true, "a cast to xs:string is the identity"},
	} {
		got := compile(t, tc.expr, "xs", xsd.XMLSchemaNS).Evaluate(backend(), seededTypes, attrs)
		if got != tc.want {
			t.Errorf("Evaluate(%q) = %v, want %v (%s)", tc.expr, got, tc.want, tc.why)
		}
	}
}

// TestEvaluateFailedCastIsFalseNotDecline pins §3.12.4 clause 2 for the cast
// productions: a cast whose operand is not in the target type's ·lexical
// space· raises err:FORG0001, which makes the {test} false — not a decline,
// and not a node-local false that fn:not could invert.
func TestEvaluateFailedCastIsFalseNotDecline(t *testing.T) {
	attrs := ctaAttrs(map[xsd.QName]string{uq("frac"): "3.5", uq("k"): "book", uq("n"): "4"})
	for _, tc := range []struct {
		expr string
		want bool
		why  string
	}{
		{"@frac cast as xs:integer > 3", false, "3.5 is not in xs:integer's lexical space"},
		{"not(@frac cast as xs:integer > 3)", false, "raised, so fn:not raises the same error"},
		{"@k cast as xs:integer = 1", false, "nor is a word"},
		{"xs:int(@k) = 1", false, "the constructor spelling raises identically"},
		{"not(xs:int(@k) = 1)", false, "and propagates identically"},
		{"@frac cast as xs:decimal > 3", true, "the discriminating row: a cast that SUCCEEDS still decides"},
		{"not(@n cast as xs:integer > 3)", false, "and a decided true still inverts"},
	} {
		got := compile(t, tc.expr, "xs", xsd.XMLSchemaNS).Evaluate(backend(), seededTypes, attrs)
		if got != tc.want {
			t.Errorf("Evaluate(%q) = %v, want %v (%s)", tc.expr, got, tc.want, tc.why)
		}
	}
}

// TestEvaluateCastOfAbsentAttribute pins xpath20.md §3.10.2 rule 3, which the
// `?` occurrence indicator splits in two: with `?` the cast of an absent
// attribute IS the empty sequence, which forms no pair and is a DECIDED false;
// without `?` it is err:XPTY0004, which is a RAISED false. fn:not is what
// tells the two apart, and §3.10.4's T($arg) ≡ (($arg) cast as T?) is why the
// constructor spelling always takes the first.
func TestEvaluateCastOfAbsentAttribute(t *testing.T) {
	attrs := ctaAttrs(map[xsd.QName]string{uq("other"): "x"})
	for _, tc := range []struct {
		expr string
		want bool
		why  string
	}{
		{"@n cast as xs:integer = 3", false, "err:XPTY0004: an empty operand with no `?`"},
		{"not(@n cast as xs:integer = 3)", false, "which is RAISED, so fn:not raises it too"},
		{"@n cast as xs:integer? = 3", false, "with `?` the cast is the empty sequence"},
		{"not(@n cast as xs:integer? = 3)", true, "which forms no pair: a decided false, and it inverts"},
		{"xs:integer(@n) = 3", false, "a constructor call over an absent attribute"},
		{"not(xs:integer(@n) = 3)", true, "carries the `?` of §3.10.4's equivalence"},
	} {
		got := compile(t, tc.expr, "xs", xsd.XMLSchemaNS).Evaluate(backend(), seededTypes, attrs)
		if got != tc.want {
			t.Errorf("Evaluate(%q) = %v, want %v (%s)", tc.expr, got, tc.want, tc.why)
		}
	}
}

// TestEvaluateCastAgainstStringLiteral pins the grounding's worked case: a
// cast operand compared against a plain StringLiteral has no xpath20.md B.2
// row, because a literal is xs:string and shares no primitive with xs:float,
// and §3.5.2's untypedAtomic casting rules never fire for it. That is
// err:XPTY0004 and so a RAISED false, which is not the same answer as a
// comparison that ran.
func TestEvaluateCastAgainstStringLiteral(t *testing.T) {
	attrs := ctaAttrs(map[xsd.QName]string{uq("f"): "1.5", uq("type"): "float"})
	for _, tc := range []struct {
		expr string
		want bool
		why  string
	}{
		{"@f cast as xs:float = 'float'", false, "err:XPTY0004, though the cast itself succeeds"},
		{"not(@f cast as xs:float = 'float')", false, "raised, so fn:not raises it"},
		{"@type cast as xs:string = 'float'", true, "the discriminating row: xs:string against xs:string compares"},
	} {
		got := compile(t, tc.expr, "xs", xsd.XMLSchemaNS).Evaluate(backend(), seededTypes, attrs)
		if got != tc.want {
			t.Errorf("Evaluate(%q) = %v, want %v (%s)", tc.expr, got, tc.want, tc.why)
		}
	}
}

// TestEvaluateCastNormalizesWhiteSpace pins xpath-functions.md §17.1.1's
// "whitespace normalization is applied as indicated by the whiteSpace facet
// for the datatype": a cast is the whole datatype validation, so casting to
// xs:token collapses the lexical before the comparison sees it, and the same
// operand uncast does not.
func TestEvaluateCastNormalizesWhiteSpace(t *testing.T) {
	attrs := ctaAttrs(map[xsd.QName]string{uq("k"): "a  b", uq("pad"): "  padded  "})
	for _, tc := range []struct {
		expr string
		want bool
	}{
		{"@k cast as xs:token = 'a b'", true},
		{"@k = 'a b'", false},
		{"@pad cast as xs:token = 'padded'", true},
		{"@pad = 'padded'", false},
	} {
		got := compile(t, tc.expr, "xs", xsd.XMLSchemaNS).Evaluate(backend(), seededTypes, attrs)
		if got != tc.want {
			t.Errorf("Evaluate(%q) = %v, want %v", tc.expr, got, tc.want)
		}
	}
}

// TestEvaluateEffectiveBooleanValueOfACast pins fn:boolean's rules over a bare
// cast in a boolean position (xpath20.md §2.4.3): rule 1 for the empty
// sequence, rule 3 for xs:boolean, rule 4 for the string family, rule 5 for
// the numeric one — NaN included, which is false and not an error — and rule
// 6's err:FORG0006 for every other type.
func TestEvaluateEffectiveBooleanValueOfACast(t *testing.T) {
	attrs := ctaAttrs(map[xsd.QName]string{
		uq("zero"):  "0",
		uq("five"):  "5",
		uq("nan"):   "NaN",
		uq("empty"): "",
		uq("word"):  "book",
		uq("yes"):   "true",
		uq("no"):    "0",
		uq("d"):     "2026-03-01",
	})
	for _, tc := range []struct {
		expr string
		want bool
		why  string
	}{
		{"@zero cast as xs:integer", false, "rule 5: numerically zero"},
		{"@five cast as xs:integer", true, "rule 5: anything else"},
		{"@nan cast as xs:double", false, "rule 5 names NaN outright"},
		{"not(@nan cast as xs:double)", true, "and it is a DECIDED false, so it inverts"},
		{"@empty cast as xs:string", false, "rule 4: zero length"},
		{"@word cast as xs:string", true, "rule 4: anything else"},
		{"@yes cast as xs:boolean", true, "rule 3: the value unchanged"},
		{"@no cast as xs:boolean", false, "including the lexical 0"},
		{"@d cast as xs:date", false, "rule 6: err:FORG0006 on every other type"},
		{"not(@d cast as xs:date)", false, "which is RAISED, so fn:not raises it"},
		{"xs:integer(@missing)", false, "rule 1: the empty sequence"},
		{"not(xs:integer(@missing))", true, "a decided false, and it inverts"},
	} {
		got := compile(t, tc.expr, "xs", xsd.XMLSchemaNS).Evaluate(backend(), seededTypes, attrs)
		if got != tc.want {
			t.Errorf("Evaluate(%q) = %v, want %v (%s)", tc.expr, got, tc.want, tc.why)
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

// castNode is a compiled `@a cast as xs:<local>` node, which is the only way a
// TYPED operand other than a Literal reaches a comparison.
func castNode(t *testing.T, known ctaTypes, local string) ctaValue {
	t.Helper()
	target, admitted := known.castTarget(ctaBuiltin(local))
	if !admitted {
		t.Fatalf("castTarget(xs:%s): declined, want admitted", local)
	}
	return ctaCast{operand: ctaAttr{name: uq("a")}, target: target}
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
		{"a cast vs a literal of its own primitive", castNode(t, known, "int"), dec, "decimal", true},
		{"an attribute vs a numeric cast is clause 2.1", attr, castNode(t, known, "int"), "double", true},
		{"an attribute vs a date cast is clause 2.4", attr, castNode(t, known, "date"), "date", true},
		{"an attribute vs a dayTimeDuration cast is clause 2.2", attr, castNode(t, known, "dayTimeDuration"), "dayTimeDuration", true},
		{"an attribute vs a yearMonthDuration cast is clause 2.3", attr, castNode(t, known, "yearMonthDuration"), "yearMonthDuration", true},
		{"two date casts share their primitive", castNode(t, known, "date"), castNode(t, known, "date"), "date", true},
		{"a date cast vs a string literal shares nothing", castNode(t, known, "date"), str, "", false},
		{"xs:boolean pairs are admitted for every comparator", castNode(t, known, "boolean"), castNode(t, known, "boolean"), "boolean", true},
		{"a token cast vs a string literal is xs:string", castNode(t, known, "token"), str, "string", true},
		{"an anyURI cast vs a string literal is B.1's URI promotion", castNode(t, known, "anyURI"), str, "string", true},
		{"a float cast vs a decimal literal promotes to the wider", castNode(t, known, "float"), dec, "float", true},
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
