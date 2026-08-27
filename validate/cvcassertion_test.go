package validate

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The fixtures below drive the two assertion rules, which charge nothing and
// record instead: cvc-assertion (§3.13.4.1) for a complex type's {assertions},
// and cvc-assertions-valid (§4.3.13.3) for a simple type's assertions facet at
// every variety level (PRINCIPLES 12). Every schema seeds the builtin types,
// because an assertion-bearing restriction is a restriction OF one.

// aAssertion is one Assertion whose {test} is expr. The expression is never
// parsed or evaluated by anything this file drives; it is there so a fixture's
// assertions are distinguishable when a message names one.
func aAssertion(expr string) xsd.Assertion {
	return xsd.NewAssertion(xsd.NewXPathExpression(expr, nil, nil, nil), nil)
}

// aAssertions turns test expressions into an assertions facet's {value}.
func aAssertions(exprs ...string) []xsd.Assertion {
	out := make([]xsd.Assertion, 0, len(exprs))
	for _, e := range exprs {
		out = append(out, aAssertion(e))
	}
	return out
}

// aRestriction builds a NAMED restriction of base carrying one assertions
// facet. It serves every variety: an atomic base, a list base and a union base
// all take their own assertions this way, which is the only way §4.3.13.2
// xr-assertions puts one on a list or a union at all — a freshly constructed
// <list> or <union> mints no assertions facet of its own.
func aRestriction(t *testing.T, name string, base xsd.QName, exprs ...string) *xsd.SimpleType {
	t.Helper()
	st, err := xsd.NewSimpleType(xsderr.Loc{}, local(name), xsd.RestrictionDerivation{},
		xsd.SimpleTypeRef{Name: base},
		[]xsd.Facet{xsd.NewAssertionsFacet(aAssertions(exprs...))}, nil)
	if err != nil {
		t.Fatalf("building the %s restriction: %v", name, err)
	}
	return st
}

// aList builds a NAMED list simple type over a by-name {item type definition},
// with the fixed collapse whiteSpace §3.16.2.1 manufactures for every <list>.
func aList(t *testing.T, name string, item xsd.QName) *xsd.SimpleType {
	t.Helper()
	st, err := xsd.NewSimpleType(xsderr.Loc{}, local(name),
		xsd.ListDerivation{Item: xsd.SimpleTypeRef{Name: item}},
		xsd.SimpleTypeRef{Name: icBuiltin("anySimpleType")},
		[]xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, true)}, nil)
	if err != nil {
		t.Fatalf("building the %s list: %v", name, err)
	}
	return st
}

// aUnion builds a NAMED union simple type over by-name members in declared
// order — the order [walk.assertionSites] must visit them in.
func aUnion(t *testing.T, name string, members ...xsd.QName) *xsd.SimpleType {
	t.Helper()
	slots := make([]xsd.SimpleTypeOrRef, 0, len(members))
	for _, m := range members {
		slots = append(slots, xsd.SimpleTypeRef{Name: m})
	}
	st, err := xsd.NewSimpleType(xsderr.Loc{}, local(name), xsd.UnionDerivation{Members: slots},
		xsd.SimpleTypeRef{Name: icBuiltin("anySimpleType")}, nil, nil)
	if err != nil {
		t.Fatalf("building the %s union: %v", name, err)
	}
	return st
}

// aVarietyTypes is the fixture set every variety level is read off: an atomic
// restriction carrying an assertion, a second one over a different primitive,
// a list of the first, that list restricted with an assertion of its own, a
// union of the two atomics, and that union restricted the same way.
func aVarietyTypes(t *testing.T) []*xsd.SimpleType {
	t.Helper()
	return []*xsd.SimpleType{
		aRestriction(t, "AssertedInt", integerType(), "$value > 0"),
		aRestriction(t, "AssertedStr", icBuiltin("string"), "string-length($value) > 0"),
		aList(t, "PlainList", local("AssertedInt")),
		aRestriction(t, "AssertedList", local("PlainList"), "count($value) > 1"),
		aUnion(t, "PlainUnion", local("AssertedInt"), local("AssertedStr")),
		aRestriction(t, "AssertedUnion", local("PlainUnion"), "$value != 'x'"),
	}
}

// aComplexType builds the governing type <root> is declared over.
func aComplexType(t *testing.T, uses []xsd.AttributeUse, content xsd.ContentType, assertions []xsd.Assertion) xsd.ComplexType {
	t.Helper()
	ct, err := xsd.NewComplexType(xsderr.Loc{}, local("RootType"), xsd.QName{}, nil,
		xsd.DerivationRestriction, false, uses, nil, nil, content, nil, assertions, nil)
	if err != nil {
		t.Fatalf("building RootType: %v", err)
	}
	return ct
}

// aTypes adds the builtin simple types and extra to b. The builtins are always
// seeded: an assertion-bearing type is a restriction OF one, and a base that
// resolves to nothing declines before any facet is read.
func aTypes(t *testing.T, b *xsd.SchemaBuilder, extra ...*xsd.SimpleType) {
	t.Helper()
	seeded, err := builtin.Seed(testBackend())
	if err != nil {
		t.Fatalf("seeding the builtin types: %v", err)
	}
	for _, st := range seeded {
		b.AddType(st)
	}
	for _, st := range extra {
		b.AddType(st)
	}
}

// aSchema finalizes a schema declaring <root> of type ct, with the builtin
// simple types seeded and extra added alongside them.
func aSchema(t *testing.T, ct xsd.ComplexType, extra ...*xsd.SimpleType) *xsd.Schema {
	t.Helper()
	return cSchemaFrom(t, ct, func(b *xsd.SchemaBuilder) { aTypes(t, b, extra...) })
}

// aAssess assesses root against schema and returns the whole Result: these
// tests read both accessors, since the claim is always about the two together.
func aAssess(t *testing.T, schema *xsd.Schema, root Element) *Result {
	t.Helper()
	v, err := New(schema, testBackend())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	res := v.Assess(root)
	if res.Err() != nil {
		t.Fatalf("Err() = %v, want nil", res.Err())
	}
	return res
}

// wantRecords fails unless res charged nothing and recorded exactly one
// Unevaluated per want entry, in order, each under rule at at and naming its
// want string in the message.
func wantRecords(t *testing.T, res *Result, rule xsderr.Rule, at xsderr.Loc, want ...string) {
	t.Helper()
	if got := res.Violations(); len(got) != 0 {
		t.Fatalf("Violations() = %v, want none: an unevaluated check is never a charge", got)
	}
	got := res.Unevaluated()
	if len(got) != len(want) {
		t.Fatalf("Unevaluated() recorded %d sites %v, want %d: %v", len(got), messages(got), len(want), want)
	}
	for i, u := range got {
		if u.Rule() != rule {
			t.Errorf("Unevaluated()[%d].Rule() = %q, want %q", i, u.Rule(), rule)
		}
		if u.Loc() != at {
			t.Errorf("Unevaluated()[%d].Loc() = %s, want %s", i, u.Loc(), at)
		}
		if !strings.Contains(u.Msg(), want[i]) {
			t.Errorf("Unevaluated()[%d].Msg() = %q, want it to name %s", i, u.Msg(), want[i])
		}
	}
}

func messages(us []Unevaluated) []string {
	out := make([]string, 0, len(us))
	for _, u := range us {
		out = append(out, u.Msg())
	}
	return out
}

// cvc-complex-type clause 6 sends E to cvc-assertion (§3.13.4.1) once per
// assertion in T.{assertions}. Each is recorded at the ELEMENT's location, in
// {assertions} order, and none is charged: an element whose only defect could
// be an assertion is not rejected.
func TestComplexTypeAssertionsAreRecordedNeverCharged(t *testing.T) {
	schema := aSchema(t, aComplexType(t, nil, xsd.EmptyContent{},
		aAssertions("@a = @b", "count(*) = 0")))

	res := aAssess(t, schema, &testElement{name: local("root"), loc: loc(1, 1)})

	// The {test} identifies WHICH assertion each record stands for, so the
	// {assertions} order is observable and not merely counted.
	wantRecords(t, res, "cvc-assertion", loc(1, 1), `"@a = @b"`, `"count(*) = 0"`)
	if !strings.Contains(res.Unevaluated()[0].Msg(), "clause 6") {
		t.Errorf("Msg = %q, want cvc-complex-type clause 6 named", res.Unevaluated()[0].Msg())
	}
	if !strings.Contains(res.Unevaluated()[0].Msg(), "assertion 1 of 2") {
		t.Errorf("Msg = %q, want the assertion's position in {assertions} named", res.Unevaluated()[0].Msg())
	}
}

// A complex type with no {assertions} records nothing: the visit is per
// assertion, not per element.
func TestElementWithoutAssertionsRecordsNothing(t *testing.T) {
	schema := aSchema(t, aComplexType(t, nil, xsd.EmptyContent{}, nil))

	res := aAssess(t, schema, &testElement{name: local("root"), loc: loc(1, 1)})

	if got := res.Unevaluated(); got != nil {
		t.Errorf("Unevaluated() = %v, want nil for a type carrying no assertions", messages(got))
	}
}

// cvc-attribute clause 3 reaches the assertions facet of the declaration's
// {type definition} through cvc-datatype-valid clause 3, so the site is
// recorded at the ATTRIBUTE's location and under the simple-type rule — never
// under cvc-assertion, which is the complex-type variety and a different rule.
func TestAttributeSimpleTypeAssertionsAreRecorded(t *testing.T) {
	uses := []xsd.AttributeUse{typedUse(t, "n", local("AssertedInt"), false, nil, nil)}
	schema := aSchema(t, aComplexType(t, uses, xsd.EmptyContent{}, nil), aVarietyTypes(t)...)

	res := aAssess(t, schema, valuedRoot("n", "42"))

	wantRecords(t, res, "cvc-assertions-valid", loc(1, 10), "AssertedInt")
}

// cvc-complex-type clause 1.2 reaches the same facet over an element's
// ·initial value·, and records it at the CONTAINING element's location — the
// value is assembled from every character run and belongs to none of them.
func TestSimpleContentAssertionsAreRecorded(t *testing.T) {
	types := aVarietyTypes(t)
	schema := aSchema(t, aComplexType(t, nil, xsd.SimpleContent{SimpleType: types[0]}, nil), types...)

	res := aAssess(t, schema, cRoot("#42"))

	wantRecords(t, res, "cvc-assertions-valid", loc(1, 1), "AssertedInt")
}

// PRINCIPLES 12: assertions live at every variety level, and the collection is
// STATIC — constituents first (cvc-datatype-valid clause 2), then the type's
// own facet (clause 3). A union therefore records one site per MEMBER TYPE
// visited, whether or not that member is the ·validating· one, and a list one
// site for its {item type definition} rather than one per item.
func TestAssertionSitesRecurseThroughListAndUnion(t *testing.T) {
	for _, c := range []struct {
		name    string
		typ     string
		lexical string
		want    []string
	}{
		{"atomic", "AssertedInt", "42", []string{"AssertedInt"}},
		{"list item alone", "PlainList", "1 2", []string{"AssertedInt"}},
		{"list item then the list's own", "AssertedList", "1 2",
			[]string{"AssertedInt", "AssertedList"}},
		{"each union member in declared order", "PlainUnion", "42",
			[]string{"AssertedInt", "AssertedStr"}},
		{"union members then the union's own", "AssertedUnion", "42",
			[]string{"AssertedInt", "AssertedStr", "AssertedUnion"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			uses := []xsd.AttributeUse{typedUse(t, "n", local(c.typ), false, nil, nil)}
			schema := aSchema(t, aComplexType(t, uses, xsd.EmptyContent{}, nil), aVarietyTypes(t)...)

			res := aAssess(t, schema, valuedRoot("n", c.lexical))

			wantRecords(t, res, "cvc-assertions-valid", loc(1, 10), c.want...)
		})
	}
}

// An element with no character content declines String Valid (cvccomplexcontent.go)
// and still has its ·initial value· read against the same type by cvcid.go and
// cvcidentityconstraint.go, so the site is recorded before that decline rather
// than after it.
func TestEmptyInitialValueStillRecordsItsSites(t *testing.T) {
	types := aVarietyTypes(t)
	schema := aSchema(t, aComplexType(t, nil, xsd.SimpleContent{SimpleType: types[0]}, nil), types...)

	res := aAssess(t, schema, cRoot())

	wantRecords(t, res, "cvc-assertions-valid", loc(1, 1), "AssertedInt")
}

// The two rule IDs are never conflated: one element carrying both an
// {assertions} on its complex type and an assertions facet on its simple
// {content type} records one site per rule, at the same Loc, discriminated by
// Rule alone.
func TestBothRulesRecordAtOneElement(t *testing.T) {
	types := aVarietyTypes(t)
	schema := aSchema(t, aComplexType(t, nil, xsd.SimpleContent{SimpleType: types[0]},
		aAssertions("@a = @b")), types...)

	res := aAssess(t, schema, cRoot("#42"))

	got := res.Unevaluated()
	if len(got) != 2 {
		t.Fatalf("Unevaluated() = %v, want one site per rule", messages(got))
	}
	if got[0].Rule() != "cvc-assertion" || got[1].Rule() != "cvc-assertions-valid" {
		t.Errorf("rules = %q, %q, want cvc-assertion then cvc-assertions-valid",
			got[0].Rule(), got[1].Rule())
	}
}

// An attribute matching no {attribute use} is left to an {attribute wildcard}
// this package does not evaluate, so [walk.matchedAttribute] never runs for it
// — but cvcid.go and cvcidentityconstraint.go decide its lexical against the
// type of the top-level declaration its name ·resolves· to, which is a site
// with no other recording path. The recording follows that pipeline for every
// {process contents}, skip included (#1043).
func TestWildcardAttributeAssertionsAreRecorded(t *testing.T) {
	ct, err := xsd.NewComplexType(xsderr.Loc{}, local("RootType"), xsd.QName{}, nil,
		xsd.DerivationRestriction, false, nil, nil, anyWildcard(t), xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("building RootType: %v", err)
	}
	d, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, local("n"),
		xsd.TypeDefinitionRef{Name: local("AssertedInt")}, xsd.NewAttributeGlobalScope(), nil, false, nil)
	if err != nil {
		t.Fatalf("building the top-level n declaration: %v", err)
	}
	schema := cSchemaFrom(t, ct, func(b *xsd.SchemaBuilder) {
		aTypes(t, b, aVarietyTypes(t)...)
		b.AddAttribute(d)
	})

	res := aAssess(t, schema, valuedRoot("n", "42"))

	wantRecords(t, res, "cvc-assertions-valid", loc(1, 10), "AssertedInt")
}

// cvc-complex-type clause 4 validates a ·defaulted attribute·'s {lexical form}
// against the declaration's {type definition}, reaching that type's assertions
// facet with no attribute information item present — so the site is recorded
// at the ELEMENT's location.
func TestDefaultedAttributeAssertionsAreRecorded(t *testing.T) {
	dflt := xsd.NewValueConstraint(xsd.ValueDefault, "42", nil, nil)
	uses := []xsd.AttributeUse{typedUse(t, "n", local("AssertedInt"), false, nil, &dflt)}
	schema := aSchema(t, aComplexType(t, uses, xsd.EmptyContent{}, nil), aVarietyTypes(t)...)

	res := aAssess(t, schema, &testElement{name: local("root"), loc: loc(1, 1)})

	wantRecords(t, res, "cvc-assertions-valid", loc(1, 1), "AssertedInt")
}

func TestResultUnevaluatedIsCopied(t *testing.T) {
	// Unevaluated is a window, not a handle, on Violations' terms.
	res := &Result{unevaluated: []Unevaluated{
		newUnevaluated("cvc-assertion", loc(1, 1), "sample")}}
	got := res.Unevaluated()
	got[0] = Unevaluated{}
	if res.Unevaluated()[0] == (Unevaluated{}) {
		t.Error("Unevaluated() shares its backing array with the Result")
	}
	if (&Result{}).Unevaluated() != nil {
		t.Error("Unevaluated() = non-nil for a Result carrying none")
	}
}

// A skipped check must never be mistakable for a verdict: an Unevaluated that
// satisfied error could be joined into a violation list, matched by errors.Is,
// or appended to a Result's violations, turning a check nothing was decided by
// into a false reject.
func TestUnevaluatedIsNotAnError(t *testing.T) {
	u := newUnevaluated("cvc-assertion", loc(1, 1), "sample")
	if _, isError := any(u).(error); isError {
		t.Fatal("Unevaluated satisfies error; it must carry no Error method")
	}
	// The pointer type is asserted separately because Result.Unevaluated()
	// hands back an ADDRESSABLE slice: a func (u *Unevaluated) Error() string
	// would leave the value assertion above clean while &got[0] mixed into a
	// violation list, which is the same false reject by another route.
	if _, isError := any(&u).(error); isError {
		t.Fatal("*Unevaluated satisfies error; it must carry no Error method on either receiver")
	}
}
