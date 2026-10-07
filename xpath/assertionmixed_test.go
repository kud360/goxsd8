package xpath

import (
	"testing"
	"time"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The fixtures below drive a child step read for its value where the child's
// ·locally declared type· is a complex type whose {content type}.{variety} is
// mixed (ctaUntypedChild): xpath-datamodel §6.2.4 makes its typed value its
// string-value as xs:untypedAtomic, which [ChildElements] yields as [Untyped].
// body is typed by a mixed type named xs:anyType, um by a user-defined mixed
// type, eo by an element-only one and n by xs:int.

// amElems is the [ElementTypes] the rows below compile against.
func amElems(t *testing.T) ElementTypes {
	t.Helper()
	anyType, err := xsd.NewComplexType(xsderr.Loc{}, ctaBuiltin("anyType"), ctaBuiltin("anyType"), nil,
		xsd.DerivationRestriction, false, nil, nil, nil, asElementContent(t, true), nil, nil)
	if err != nil {
		t.Fatalf("building xs:anyType: %v", err)
	}
	return asElems(map[xsd.QName]xsd.TypeDefinition{
		uq("body"): anyType,
		uq("um"):   asComplex(t, "UserMixed", asElementContent(t, true)),
		uq("eo"):   asComplex(t, "ElementOnly", asElementContent(t, false)),
		uq("n"):    asBuiltin(t, "int"),
	})
}

// amCompile compiles expr over amElems, or fails the test.
func amCompile(t *testing.T, expr string) AssertionTest {
	t.Helper()
	test, ok := CompileAssertionTest(asRecord(expr), seededTypes, asElementContent(t, true), asUses(t, nil), amElems(t))
	if !ok {
		t.Fatalf("CompileAssertionTest(%q): declined, want compiled", expr)
	}
	return test
}

// amChildren is the [ChildElements] over children in the order written.
func amChildren(children ...ChildElement) ChildElements {
	return func(yield func(ChildElement) bool) {
		for _, c := range children {
			if !yield(c) {
				return
			}
		}
	}
}

// A child of mixed content is read as its string-value, xs:untypedAtomic
// (xpath-datamodel §6.2.4), everywhere an untyped attribute is:
// fn:string-length counts its characters, none of them is the empty sequence,
// the zero-length string (xpath-functions.md §7.4.4), and two are err:XPTY0004
// against the xs:string? parameter (xpath20.md §3.1.5), false under fn:not
// too. A general comparison casts it to the other operand's type (§3.5.2), so
// `body = 3` compares numbers, and is existential over two; a value comparison
// casts it to xs:string (§3.5.1); a cast validates its lexical (F&O §17.1.1),
// raising err:FORG0001 for one that does not; fn:string, fn:contains,
// arithmetic and fn:distinct-values read it as xs:untypedAtomic. um is a
// user-defined mixed type, not xs:anyType: the compile admits on the {variety},
// never on the name, so its rows decline with ctaMixedType testing the name
// xs:anyType instead. A ·nilled· child is Child(name, nil), whose typed value is
// the empty sequence (xpath20.md §2.5.2 item 4.1): `string-length(body) eq 0`
// holds over it and `body = ”` does not. With the ctaUntypedChild arm of
// ctaItemOf removed, its default reading every child as the empty sequence,
// every row whose answer that changes fails.
func TestAssertionReadsMixedChildUntyped(t *testing.T) {
	body := func(lexical string) ChildElement { return Child(uq("body"), Untyped(lexical)) }
	nilled := Child(uq("body"), nil)
	for _, tc := range []struct {
		expr     string
		children []ChildElement
		want     bool
	}{
		{"string-length(body) eq 5", []ChildElement{body("hello")}, true},
		{"string-length(body) gt 0", []ChildElement{body("hello")}, true},
		{"string-length(body) gt 0", []ChildElement{body("")}, false},
		{"string-length(body) eq 0", nil, true},
		{"string-length(body) ge 0", []ChildElement{body("a"), body("b")}, false},
		{"not(string-length(body) ge 0)", []ChildElement{body("a"), body("b")}, false},
		{"string-length(um) eq 3", []ChildElement{Child(uq("um"), Untyped("abc"))}, true},
		{"um = 'abc'", []ChildElement{Child(uq("um"), Untyped("abc"))}, true},
		{"body = 'hello'", []ChildElement{body("hello")}, true},
		{"body = 'hello'", []ChildElement{body("other")}, false},
		{"body = 'hello'", []ChildElement{body("other"), body("hello")}, true},
		{"body = 3", []ChildElement{body("3.0")}, true},
		{"body eq 'hello'", []ChildElement{body("hello")}, true},
		{"body eq 'hello'", []ChildElement{body("hello"), body("hello")}, false},
		{"xs:integer(body) eq 3", []ChildElement{body("3")}, true},
		{"body cast as xs:integer eq 3", []ChildElement{body("3")}, true},
		{"xs:integer(body) eq 3", []ChildElement{body("x")}, false},
		{"string(body) eq 'hello'", []ChildElement{body("hello")}, true},
		{"string(body) eq ''", nil, true},
		{"contains(body, 'ell')", []ChildElement{body("hello")}, true},
		{"body + 1 eq 4", []ChildElement{body("3")}, true},
		{"count(distinct-values(body)) eq 1", []ChildElement{body("a"), body("a")}, true},
		{"string-length(body) eq 0", []ChildElement{nilled}, true},
		{"body = ''", []ChildElement{nilled}, false},
		{"body = ''", []ChildElement{body("")}, true},
	} {
		t.Run(tc.expr, func(t *testing.T) {
			test := amCompile(t, tc.expr)
			if got := test.Evaluate(backend(), seededTypes, asValues(t), amChildren(tc.children...), nil, ValueBinding{}, time.Time{}); got != tc.want {
				t.Errorf("Evaluate(%q) over %v = %v, want %v", tc.expr, tc.children, got, tc.want)
			}
		})
	}
}

// A [Typed] value under a mixed child's name breaks the obligation
// [ChildElements] states, and every node reading it raises, on
// ctaMatchedAttributes' terms for a typed input (ctaEachUntypedChild): each
// row is false, and its fn:not false too. With ctaEachUntypedChild accepting
// the Typed arm as its canonical lexical, `string-length(body) ge 0` holds
// instead.
func TestAssertionRaisesOnATypedMixedChild(t *testing.T) {
	v, err := value.ValidateLexical(backend(), seededTypes, asBuiltin(t, "string"), "x", nil, FacetAssertions(time.Time{}))
	if err != nil {
		t.Fatalf("mapping x: %v", err)
	}
	children := amChildren(Child(uq("body"), Typed(v)))
	for _, expr := range []string{"string-length(body) ge 0", "not(string-length(body) ge 0)", "body = 'x'", "not(body = 'x')"} {
		if amCompile(t, expr).Evaluate(backend(), seededTypes, asValues(t), children, nil, ValueBinding{}, time.Time{}) {
			t.Errorf("Evaluate(%q) over a Typed mixed child = true, want false: the read raises", expr)
		}
	}
}

// A step whose existence alone is asked reads no value, whatever the child's
// type, so a mixed child's `body`, `exists(body)` and `.//body` are counted by
// the [Tally] and ReadsChild(body) is false for them; a value read of it,
// `string-length(body) gt 0`, reads it off [ChildElements] with no Tally.
// With ctaUntypedChild.readsChild false, the value row fails.
func TestAssertionMixedChildReadsAndCounts(t *testing.T) {
	read := amCompile(t, "string-length(body) gt 0")
	if !read.ReadsChild(uq("body")) || read.ReadsChild(uq("um")) || read.Tally() != nil {
		t.Errorf("(string-length(body) gt 0): ReadsChild(body) = %v, ReadsChild(um) = %v, Tally() = %v, want true, false and nil",
			read.ReadsChild(uq("body")), read.ReadsChild(uq("um")), read.Tally())
	}
	for _, expr := range []string{"body", "exists(body)", ".//body"} {
		test := amCompile(t, expr)
		if test.ReadsChild(uq("body")) || test.Tally() == nil {
			t.Errorf("(%s): ReadsChild(body) = %v, Tally() = %v, want false and a Tally", expr, test.ReadsChild(uq("body")), test.Tally())
		}
	}
}

// A value predicate over a mixed child DECLINES (ctaParser.valuePredicate): a
// candidate is typed, and an untyped one has no arm. The two predicate rows
// compile, and fail, with valuePredicate reading `.` as xs:string wherever the
// step is no ctaTypedChild. An element-only child read for its value still
// declines, a guard: its atomization is a type error (xpath-datamodel §6.2.4).
func TestAssertionMixedChildDeclines(t *testing.T) {
	for _, expr := range []string{"count(body[. = 'x']) eq 1", "count(um[. = 'x']) ge 0", "eo = 1", "string-length(eo) eq 0"} {
		if _, ok := CompileAssertionTest(asRecord(expr), seededTypes, asElementContent(t, true), asUses(t, nil), amElems(t)); ok {
			t.Errorf("CompileAssertionTest(%q): compiled, want declined", expr)
		}
	}
}

// ctaUntypedChild is a step: its nodes are the children it selects, a ·nilled·
// one counting, which fn:empty and fn:exists read through ctaSequenceLength
// (ctaStep). A Typed value under its name raises. With ctaUntypedChild.nodes
// removed, ctaSequenceLength reads its static type, xs:untypedAtomic, as the
// empty sequence and answers 0.
func TestUntypedChildNodes(t *testing.T) {
	step := ctaUntypedChild{name: uq("body")}
	in := ctaTypedInput{attrs: asValues(t), children: amChildren(Child(uq("body"), nil), Child(uq("um"), Untyped("x")), Child(uq("body"), Untyped("y")))}
	if n, ok := ctaSequenceLength(step, ctaEnv{backend: backend(), types: seededTypes, input: in}); n != 2 || !ok {
		t.Errorf("ctaSequenceLength(body) = %d, %v, want 2, true", n, ok)
	}
	v, err := value.ValidateLexical(backend(), seededTypes, asBuiltin(t, "string"), "x", nil, FacetAssertions(time.Time{}))
	if err != nil {
		t.Fatalf("mapping x: %v", err)
	}
	in.children = amChildren(Child(uq("body"), Typed(v)))
	if _, ok := ctaSequenceLength(step, ctaEnv{backend: backend(), types: seededTypes, input: in}); ok {
		t.Error("ctaSequenceLength(body) over a Typed child = ok, want raised")
	}
}
