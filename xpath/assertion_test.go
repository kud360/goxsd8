package xpath

import (
	"testing"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The fixtures below drive the assertion façade: the §3.12.6 grammar over an
// element whose attributes are TYPED (cvc-assertion clause 1, §3.13.4.1).

// asTyped is one typed attribute of the element an assertion is evaluated
// against: its ·expanded name·, the builtin type its use declares, and the
// lexical its ·actual value· is mapped from.
type asTyped struct {
	name    xsd.QName
	typ     string
	lexical string
}

// asBuiltin is the seeded builtin simple type named local, or a fatal error.
func asBuiltin(t *testing.T, local string) *xsd.SimpleType {
	t.Helper()
	td, declared := seededTypes.Type(ctaBuiltin(local))
	if !declared {
		t.Fatalf("the seeded builtins hold no xs:%s", local)
	}
	st, simple := td.(*xsd.SimpleType)
	if !simple {
		t.Fatalf("xs:%s is not a simple type", local)
	}
	return st
}

// asUses is the [AttributeTypes] of an element whose governing type declares
// one use per entry of uses, name to builtin type local name.
func asUses(t *testing.T, uses map[string]string) AttributeTypes {
	t.Helper()
	typed := make(map[xsd.QName]*xsd.SimpleType, len(uses))
	for local, typ := range uses {
		typed[uq(local)] = asBuiltin(t, typ)
	}
	return func(name xsd.QName) (*xsd.SimpleType, bool) {
		st, ok := typed[name]
		return st, ok
	}
}

// asValues is the [TypedAttributes] over attrs in the order WRITTEN, each
// value mapped by the backend from its lexical against its builtin type.
func asValues(t *testing.T, attrs ...asTyped) TypedAttributes {
	t.Helper()
	vs := make([]value.Value, 0, len(attrs))
	for _, a := range attrs {
		v, err := value.ValidateLexical(backend(), seededTypes, asBuiltin(t, a.typ), a.lexical, nil)
		if err != nil {
			t.Fatalf("mapping %q as xs:%s: %v", a.lexical, a.typ, err)
		}
		vs = append(vs, v)
	}
	return func(yield func(xsd.QName, value.Value) bool) {
		for i, a := range attrs {
			if !yield(a.name, vs[i]) {
				return
			}
		}
	}
}

// asCompile compiles expr against uses or fails the test.
func asCompile(t *testing.T, expr string, uses AttributeTypes) AssertionTest {
	t.Helper()
	c, ok := CompileAssertionTest(ctaExprRecord(expr, "", "xs", xsd.XMLSchemaNS, "a", "http://example.com/a"), seededTypes, uses)
	if !ok {
		t.Fatalf("CompileAssertionTest(%q): declined, want compiled", expr)
	}
	return c
}

// An assertion reads its attributes TYPED, so a comparison runs in the
// attributes' own types and not in xs:string. `@min <= @max` over min="10" and
// max="9" is the discriminating case: as xs:int values 10 <= 9 is false, while
// the untyped reading a Type Alternative takes compares the strings "10" and
// "9" and answers true (TestEvaluateUntypedReadingOfTypedNames pins that half).
func TestAssertionEvaluatesTypedAttributes(t *testing.T) {
	uses := asUses(t, map[string]string{"x": "integer", "min": "int", "max": "int", "s": "string", "d": "double"})
	for _, tc := range []struct {
		expr  string
		attrs []asTyped
		want  bool
	}{
		{"@x > 300", []asTyped{{uq("x"), "integer", "500"}}, true},
		{"@x > 300", []asTyped{{uq("x"), "integer", "200"}}, false},
		{"@x > 300", nil, false},
		{"@x = 300", []asTyped{{uq("x"), "integer", "+0300"}}, true},
		{"@min <= @max", []asTyped{{uq("min"), "int", "10"}, {uq("max"), "int", "9"}}, false},
		{"@min <= @max", []asTyped{{uq("min"), "int", "9"}, {uq("max"), "int", "10"}}, true},
		{"@d > 1.5", []asTyped{{uq("d"), "double", "2"}}, true},
		{"@s = 'abc'", []asTyped{{uq("s"), "string", "abc"}}, true},
		{"@x", []asTyped{{uq("x"), "integer", "0"}}, true},
		{"@x", nil, false},
		{"not(@x) or @x > 0", nil, true},
		{"@s cast as xs:integer > 3", []asTyped{{uq("s"), "string", " 5 "}}, true},
		{"xs:integer(@s) = 5", []asTyped{{uq("s"), "string", "5"}}, true},
	} {
		got := asCompile(t, tc.expr, uses).Evaluate(backend(), seededTypes, asValues(t, tc.attrs...))
		if got != tc.want {
			t.Errorf("Evaluate(%q) over %v = %v, want %v", tc.expr, tc.attrs, got, tc.want)
		}
	}
}

// A TYPE ERROR is reachable over typed operands that an untyped reading never
// raises: `@b = 'true'` with @b typed xs:boolean compares an xs:boolean with
// an xs:string, which is err:XPTY0004 (xpath20.md §3.5.2, B.2). cvc-assertion
// makes a {test} that raises false — "evaluates to true ... without raising
// any dynamic error or type error" — so Evaluate answers false and the caller
// charges cvc-assertion. Spec-correct under the typed reading; do not "fix" it
// into a cast.
func TestAssertionTypeErrorIsFalse(t *testing.T) {
	uses := asUses(t, map[string]string{"b": "boolean"})
	attrs := asValues(t, asTyped{uq("b"), "boolean", "true"})

	if asCompile(t, "@b = 'true'", uses).Evaluate(backend(), seededTypes, attrs) {
		t.Error("Evaluate(@b = 'true') over a typed xs:boolean = true, want false: the comparison raises err:XPTY0004")
	}
	if asCompile(t, "not(@b = 'true')", uses).Evaluate(backend(), seededTypes, attrs) {
		t.Error("Evaluate(not(@b = 'true')) = true, want false: fn:not propagates the raised error, and the {test} raised")
	}
	if !asCompile(t, "@b = @b", uses).Evaluate(backend(), seededTypes, attrs) {
		t.Error("Evaluate(@b = @b) = false, want true: two xs:boolean operands are B.2-comparable")
	}
}

// The same names read UNTYPED through the Type Alternative façade answer the
// other way round, which is what shows the two façades build different trees
// over one grammar: "10" <= "9" holds as xs:string, and @b = 'true' is a
// string comparison that holds.
func TestEvaluateUntypedReadingOfTypedNames(t *testing.T) {
	if !compile(t, "@min <= @max").Evaluate(backend(), seededTypes, ctaAttrs(at("min", "10"), at("max", "9"))) {
		t.Error(`CTA Evaluate(@min <= @max) over "10", "9" = false, want true: untyped operands compare as xs:string`)
	}
	if !compile(t, "@b = 'true'").Evaluate(backend(), seededTypes, ctaAttrs(at("b", "true"))) {
		t.Error("CTA Evaluate(@b = 'true') = false, want true: an untyped operand is cast to xs:string")
	}
}

// asUnion is a union of xs:int and xs:string, whose value takes the type of
// its ·validating· member and so has no type fixed at compile time.
func asUnion(t *testing.T) *xsd.SimpleType {
	t.Helper()
	st, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: ctaUserNS, Local: "IntOrString"},
		xsd.UnionDerivation{Members: []xsd.SimpleTypeOrRef{
			xsd.SimpleTypeRef{Name: ctaBuiltin("int")}, xsd.SimpleTypeRef{Name: ctaBuiltin("string")},
		}}, xsd.SimpleTypeRef{Name: ctaBuiltin("anySimpleType")}, nil, nil)
	if err != nil {
		t.Fatalf("building the union: %v", err)
	}
	return st
}

// CompileAssertionTest DECLINES — ok false, never a tree that answers false —
// every {test} whose operand types it cannot fix, every construct outside the
// §3.12.6 grammar ($value and the value comparators among them), and every
// decline CompileCTATest itself makes.
func TestCompileAssertionTestDeclines(t *testing.T) {
	union := asUnion(t)
	types := asUses(t, map[string]string{
		"x": "integer", "l": "NMTOKENS", "any": "anySimpleType", "atom": "anyAtomicType",
		"q": "QName", "n": "NOTATION", "f": "float",
	})
	uses := func(name xsd.QName) (*xsd.SimpleType, bool) {
		if name == uq("u") {
			return union, true
		}
		return types(name)
	}
	for _, tc := range []struct{ expr, why string }{
		{"@y > 1", "a name with no attribute use, which AttributeTypes does not type"},
		{"@a:x > 1", "the same name in another namespace"},
		{"@* = 1", "a wildcard NameTest can match a wildcard-attributed attribute"},
		{"@a:* = 1", "a prefixed wildcard likewise"},
		{"@*:x = 1", "a local-name wildcard likewise"},
		{"@p:x = 1", "an unbound prefix"},
		{"@l = 'a'", "a list type atomizes to a sequence"},
		{"@u = 1", "a union's value takes its validating member's type"},
		{"@any = 'a'", "xs:anySimpleType has no primitive"},
		{"@atom = 'a'", "xs:anyAtomicType has no primitive"},
		{"@q = 'a'", "an xs:QName value has no canonical representation to convert through"},
		{"@n = 'a'", "an xs:NOTATION value likewise"},
		{"@x cast as xs:string = '5'", "a cast from a typed non-string attribute"},
		{"xs:integer(@x) = 5", "the constructor spelling of the same cast"},
		{"@f = 1e0", "B.1 rule 1.1's xs:float to xs:double promotion, CompileCTATest's own decline"},
		{"$value > 0", "$value is outside the grammar"},
		{"@x le 5", "a value comparison is outside the grammar"},
		{"@x eq 5", "likewise eq"},
		{"count(@x) = 1", "a function call outside fn:not and the constructors"},
	} {
		if _, ok := CompileAssertionTest(ctaExprRecord(tc.expr, "", "xs", xsd.XMLSchemaNS, "a", "http://example.com/a"), seededTypes, uses); ok {
			t.Errorf("CompileAssertionTest(%q): compiled, want declined (%s)", tc.expr, tc.why)
		}
	}
}

// A Type Alternative's {test} still declines $value and the value comparators:
// the assertion façade widened nothing in the grammar CompileCTATest admits.
func TestCompileCTATestStillDeclinesAssertionOnlyForms(t *testing.T) {
	for _, expr := range []string{"$value > 0", "@min le @max", "@x eq 5"} {
		if _, ok := CompileCTATest(ctaExprRecord(expr, ""), seededTypes); ok {
			t.Errorf("CompileCTATest(%q): compiled, want declined", expr)
		}
	}
}

// A general comparison whose comparison type is in the date/time family
// DECLINES at CompileAssertionTest and still compiles under CompileCTATest:
// without an implicit timezone (F&O §10.4, cvc-xpath clause 7) the engine
// decides `@d < @e or @d >= @e` over @d=2000-01-01 and @e=2000-01-01Z false,
// a tautology an assertion would be charged on. Each type below reaches the
// decline through its {primitive type definition}, xs:dateTimeStamp through
// xs:dateTime's.
func TestCompileAssertionTestDeclinesDateTimeComparisons(t *testing.T) {
	uses := asUses(t, map[string]string{
		"d": "date", "e": "date", "dt": "dateTime", "dts": "dateTimeStamp", "tm": "time",
		"gym": "gYearMonth", "gy": "gYear", "gmd": "gMonthDay", "gd": "gDay", "gm": "gMonth", "s": "string",
	})
	for _, expr := range []string{
		"@d < @e or @d >= @e",
		"@d = @e",
		"@dt < xs:dateTime('2000-01-01T00:00:00')",
		"@dts != xs:dateTime('2000-01-01T00:00:00')",
		"@tm <= xs:time('00:00:00')",
		"@gym = xs:gYearMonth('2000-01')",
		"@gy = xs:gYear('2000')",
		"@gmd = xs:gMonthDay('--01-01')",
		"@gd = xs:gDay('---01')",
		"@gm = xs:gMonth('--01')",
		"@s cast as xs:date < xs:date('2000-01-01')",
	} {
		record := ctaExprRecord(expr, "", "xs", xsd.XMLSchemaNS)
		if _, ok := CompileAssertionTest(record, seededTypes, uses); ok {
			t.Errorf("CompileAssertionTest(%q): compiled, want declined (a date/time comparison type)", expr)
		}
		if _, ok := CompileCTATest(record, seededTypes); !ok {
			t.Errorf("CompileCTATest(%q): declined, want compiled (the Type Alternative façade admits every comparison type)", expr)
		}
	}
}
