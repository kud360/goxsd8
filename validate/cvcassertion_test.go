package validate

import (
	"strconv"
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
	return xsd.NewAssertion(xsd.NewXPathExpression(expr, nil, nil, nil))
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
		xsd.DerivationRestriction, false, attrContent(uses), nil, nil, content, nil, assertions)
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
// assertion in T.{assertions}. One xpath DECLINES — `@a = @b` names attributes
// T has no use for, and `count(*) = 0` is outside the grammar — is recorded at
// the ELEMENT's location, in {assertions} order, and none is charged: an
// element whose only defect could be an undecided assertion is not rejected.
func TestDeclinedComplexTypeAssertionsAreRecordedNeverCharged(t *testing.T) {
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

// aRoot is <root> carrying the attributes name/lexical pairs give, in that
// order, at columns 10, 20, ….
func aRoot(pairs ...string) *testElement {
	e := &testElement{name: local("root"), loc: loc(1, 1)}
	for i := 0; i+1 < len(pairs); i += 2 {
		e.attrs = append(e.attrs, &testAttribute{name: local(pairs[i]), value: pairs[i+1], loc: loc(1, 10*(i/2+1))})
	}
	return e
}

// aTyped builds RootType with one optional use per name/type pair, over a
// builtin type each, and the assertions given.
func aTyped(t *testing.T, pairs []string, exprs ...string) *xsd.Schema {
	t.Helper()
	var uses []xsd.AttributeUse
	for i := 0; i+1 < len(pairs); i += 2 {
		uses = append(uses, typedUse(t, pairs[i], icBuiltin(pairs[i+1]), false, nil, nil))
	}
	return aSchema(t, aComplexType(t, uses, xsd.EmptyContent{}, aAssertions(exprs...)))
}

// wantAssertionCharge fails unless res charged exactly one violation, under
// cvc-assertion at <root>, whose message OPENS with the element and the
// assertion's position — the subject a swapped argument would move — and
// recorded nothing.
func wantAssertionCharge(t *testing.T, res *Result, opening string) {
	t.Helper()
	if got := res.Unevaluated(); len(got) != 0 {
		t.Fatalf("Unevaluated() = %v, want none: an evaluated assertion is not a decline", messages(got))
	}
	got := res.Violations()
	if len(got) != 1 {
		t.Fatalf("Violations() = %v, want one cvc-assertion charge", got)
	}
	if got[0].Rule != "cvc-assertion" || got[0].Loc != loc(1, 1) {
		t.Errorf("charge = %s at %s, want cvc-assertion at %s", got[0].Rule, got[0].Loc, loc(1, 1))
	}
	if !strings.HasPrefix(got[0].Msg, opening) {
		t.Errorf("Msg = %q, want it to open %q", got[0].Msg, opening)
	}
}

// An assertion inside the grammar xpath compiles is EVALUATED over the
// element's TYPED attributes (cvc-assertion clause 1): `@x > 300` with x an
// xs:integer holds for 500, is charged for 200, and is charged for an absent
// x — the empty sequence forms no pair, so the general comparison is false.
func TestAdmittedAssertionIsEvaluated(t *testing.T) {
	schema := aTyped(t, []string{"x", "integer"}, "@x > 300")

	if res := aAssess(t, schema, aRoot("x", "500")); len(res.Violations()) != 0 || len(res.Unevaluated()) != 0 {
		t.Errorf("x=500: Violations() = %v, Unevaluated() = %v, want both empty: the assertion holds",
			res.Violations(), messages(res.Unevaluated()))
	}
	wantAssertionCharge(t, aAssess(t, schema, aRoot("x", "200")),
		`the element root is not ·valid· with respect to assertion 1 of 1 in the {assertions} of the ·governing type definition· RootType, whose {test} is "@x > 300",`)
	wantAssertionCharge(t, aAssess(t, schema, aRoot()), "the element root is not ·valid· with respect to assertion 1 of 1")
}

// Attributes are compared TYPED: as xs:int values 10 <= 9 is false and
// charged, where the xs:string comparison an untyped reading makes holds.
func TestAssertionComparesTypedValues(t *testing.T) {
	schema := aTyped(t, []string{"min", "int", "max", "int"}, "@min <= @max")

	wantAssertionCharge(t, aAssess(t, schema, aRoot("min", "10", "max", "9")), "the element root is not ·valid·")
	if res := aAssess(t, schema, aRoot("min", "9", "max", "10")); len(res.Violations()) != 0 || len(res.Unevaluated()) != 0 {
		t.Errorf("9 <= 10: Violations() = %v, Unevaluated() = %v, want both empty", res.Violations(), messages(res.Unevaluated()))
	}
}

// A VALUE comparison is evaluated as §3.13.2's own example writes it:
// `@min le @max` over two xs:int attributes is charged for min="6" max="5" —
// the d4_3_15ii01 shape — and holds for min="5" max="6".
func TestValueComparisonAssertionIsEvaluated(t *testing.T) {
	schema := aTyped(t, []string{"min", "int", "max", "int"}, "@min le @max")

	wantAssertionCharge(t, aAssess(t, schema, aRoot("min", "6", "max", "5")),
		`the element root is not ·valid· with respect to assertion 1 of 1 in the {assertions} of the ·governing type definition· RootType, whose {test} is "@min le @max",`)
	if res := aAssess(t, schema, aRoot("min", "5", "max", "6")); len(res.Violations()) != 0 || len(res.Unevaluated()) != 0 {
		t.Errorf("5 le 6: Violations() = %v, Unevaluated() = %v, want both empty", res.Violations(), messages(res.Unevaluated()))
	}
}

// aSimple builds RootType over a simple {content type} of the builtin typ, with
// the attribute uses and assertions given, and declares <root> over it,
// {nillable} as given.
func aSimple(t *testing.T, typ string, nillable bool, uses []xsd.AttributeUse, exprs ...string) *xsd.Schema {
	t.Helper()
	td, declared := builtinType(t, typ)
	if !declared {
		t.Fatalf("no builtin xs:%s", typ)
	}
	ct := aComplexType(t, uses, xsd.SimpleContent{SimpleType: td}, aAssertions(exprs...))
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "root"},
		xsd.TypeDefinitionRef{Name: ct.Name()}, nil, xsd.NewGlobalScope(), nil, nillable, nil, nil, nil, false, nil)
	if err != nil {
		t.Fatalf("building the root element declaration: %v", err)
	}
	b := xsd.NewSchemaBuilder()
	aTypes(t, b)
	b.AddType(ct)
	b.AddElement(e)
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the schema: %v", err)
	}
	return schema
}

// builtinType is the seeded builtin simple type named local.
func builtinType(t *testing.T, local string) (*xsd.SimpleType, bool) {
	t.Helper()
	seeded, err := builtin.Seed(testBackend())
	if err != nil {
		t.Fatalf("seeding the builtin types: %v", err)
	}
	for _, st := range seeded {
		if st.Name() == icBuiltin(local) {
			return st, true
		}
	}
	return nil, false
}

// wantSatisfied fails unless res charged and recorded nothing.
func wantSatisfied(t *testing.T, res *Result, why string) {
	t.Helper()
	if len(res.Violations()) != 0 || len(res.Unevaluated()) != 0 {
		t.Errorf("%s: Violations() = %v, Unevaluated() = %v, want both empty", why, res.Violations(), messages(res.Unevaluated()))
	}
}

// `$value` over a SIMPLE {content type} is the element's ·actual value·
// (cvc-assertion clause 2.3.1), typed by the {simple type definition}: `$value
// eq 5` holds for "+05", is charged for "6", and the value the ·initial value·
// maps to is what is compared, white space collapsed.
func TestValueAssertionReadsSimpleContent(t *testing.T) {
	schema := aSimple(t, "int", false, nil, "$value eq 5")

	wantSatisfied(t, aAssess(t, schema, cRoot("# +05 ")), "$value eq 5 over +05")
	wantAssertionCharge(t, aAssess(t, schema, cRoot("#6")),
		`the element root is not ·valid· with respect to assertion 1 of 1 in the {assertions} of the ·governing type definition· RootType, whose {test} is "$value eq 5",`)
}

// `$value` over a {content type} that is not simple is the EMPTY SEQUENCE
// (cvc-assertion clause 2.3.2), so `$value eq 1` is the empty sequence, false
// under fn:boolean and charged, and its negation is satisfied.
func TestValueAssertionOverElementOnlyContentIsEmpty(t *testing.T) {
	content := cSequence(t, false, cParticle(t, "a", 0, 1))
	charged := aSchema(t, aComplexType(t, nil, content, aAssertions("$value eq 1")))
	wantAssertionCharge(t, aAssess(t, charged, cRoot()), "the element root is not ·valid· with respect to assertion 1 of 1")

	negated := aSchema(t, aComplexType(t, nil, content, aAssertions("not($value eq 1)")))
	wantSatisfied(t, aAssess(t, negated, cRoot()), "not($value eq 1) over element-only content")
}

// An element ALREADY KNOWN TO BE INVALID when its assertions are evaluated —
// here for a missing required attribute, cvc-complex-type clause 3 — binds
// `$value` to the empty sequence (cvc-assertion clauses 2.3.1.1 and 2.3.2),
// not to its ·actual value·: `not($value eq 5)` over "5" is satisfied, where
// the value would make it false.
func TestValueAssertionOverAnInvalidElementIsEmpty(t *testing.T) {
	uses := []xsd.AttributeUse{typedUse(t, "r", icBuiltin("string"), true, nil, nil)}
	schema := aSimple(t, "int", false, uses, "not($value eq 5)")

	res := aAssess(t, schema, cRoot("#5"))
	if got := res.Violations(); len(got) != 1 || got[0].Rule != "cvc-complex-type" {
		t.Fatalf("Violations() = %v, want the cvc-complex-type clause 3 charge alone: the assertion holds over the empty sequence", got)
	}
	if got := res.Unevaluated(); len(got) != 0 {
		t.Errorf("Unevaluated() = %v, want none", messages(got))
	}
	wantSatisfied(t, aAssess(t, aSimple(t, "int", false, nil, "$value eq 5"), cRoot("#5")), "$value eq 5 over a valid 5")
}

// A ·nilled· element binds `$value` to the empty sequence (cvc-assertion clause
// 2.3.1.2), whatever its type's {content type}. Over xs:string the case is
// discriminating: the empty ·initial value· a ·nilled· element carries is a
// valid xs:string, so reading it would make `$value` equal the empty string.
func TestValueAssertionOverANilledElementIsEmpty(t *testing.T) {
	root := cRoot()
	root.attrs = []Attribute{&testAttribute{
		name: xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "nil"}, value: "true", loc: loc(1, 10)}}

	wantSatisfied(t, aAssess(t, aSimple(t, "string", true, nil, "not($value eq '')"), root), "not($value eq '') over a ·nilled· element")
	wantAssertionCharge(t, aAssess(t, aSimple(t, "string", true, nil, "$value eq ''"), root), "the element root is not ·valid· with respect to assertion 1 of 1")
}

// Where String Valid over the ·initial value· is WITHHELD, `$value` is
// undecided, and every assertion of the element is declined — never charged,
// never satisfied — beside clause 1.2's own record.
func TestValueAssertionOverAnUndecidedValueIsDeclined(t *testing.T) {
	td, _ := builtinType(t, "decimal")
	schema := aSchema(t, aComplexType(t, nil, xsd.SimpleContent{SimpleType: td}, aAssertions("$value gt 1")))

	got, undecided := assessRecordedWith(t, gapBackend(icBuiltin("decimal")), schema, cRoot("#1.5"))
	if len(got) != 0 {
		t.Fatalf("Violations() = %v, want none: an undecided $value charges nothing", got)
	}
	if len(undecided) != 2 || undecided[1].Rule() != "cvc-assertion" ||
		!strings.Contains(undecided[1].Msg(), "the element root has simple content whose [schema actual value], which cvc-assertion clause 2.3.1 binds to $value, is undecided") {
		t.Fatalf("Unevaluated() = %v, want clause 1.2's record then the assertion declined for its undecided $value", messages(undecided))
	}
}

// An admitted {test} that RAISES is charged, never declined: `@b = 'true'`
// over an xs:boolean @b compares xs:boolean with xs:string, err:XPTY0004, and
// cvc-assertion is satisfied only by a {test} that is true "without raising
// any dynamic error or type error".
func TestAssertionRaisingATypeErrorIsCharged(t *testing.T) {
	schema := aTyped(t, []string{"b", "boolean"}, "@b = 'true'")

	wantAssertionCharge(t, aAssess(t, schema, aRoot("b", "true")), "the element root is not ·valid·")
}

// A {test} outside what xpath compiles is DECLINED through walk.decline —
// recorded under cvc-assertion at the element, never charged, never satisfied
// — whatever the attribute values would have made of it.
func TestOutOfFamilyAssertionIsDeclined(t *testing.T) {
	for _, expr := range []string{"$other > 0", "@x eq 5 eq 5", "count(@*) = 1", "@* = 5", "@y = 5"} {
		schema := aTyped(t, []string{"x", "integer"}, expr)
		res := aAssess(t, schema, aRoot("x", "500"))
		wantRecords(t, res, "cvc-assertion", loc(1, 1), "whose {test} is "+strconv.Quote(expr)+", was not evaluated: this engine's XPath evaluator declined it")
	}
}

// An element one of whose use-matched attributes has no ·actual value· has
// every assertion DECLINED: reading `@x` as the empty sequence would make
// `not(@x)` true and `@x > 300` false on a value the instance does carry. The
// cvc-attribute clause 3 charge stands beside the record.
func TestAssertionOverAnAttributeWithoutActualValueIsDeclined(t *testing.T) {
	schema := aTyped(t, []string{"x", "integer"}, "@x > 300")

	res := aAssess(t, schema, aRoot("x", "many"))

	if got := res.Violations(); len(got) != 1 || got[0].Rule != "cvc-attribute" {
		t.Fatalf("Violations() = %v, want the cvc-attribute clause 3 charge alone", got)
	}
	got := res.Unevaluated()
	if len(got) != 1 || got[0].Rule() != "cvc-assertion" ||
		!strings.Contains(got[0].Msg(), "the attribute x of the element root has no ·actual value·") {
		t.Fatalf("Unevaluated() = %v, want the assertion declined for x's missing ·actual value·", messages(got))
	}
}

// A {test} naming a ·defaulted attribute· the element does not carry is
// declined, whether the partial PSVI holds it being unruled; carried, the same
// attribute is read like any other.
func TestAssertionNamingAnUncarriedDefaultedAttributeIsDeclined(t *testing.T) {
	dflt := xsd.NewValueConstraint(xsd.ValueDefault, "500", nil, nil)
	uses := []xsd.AttributeUse{typedUse(t, "x", integerType(), false, nil, &dflt)}
	schema := aSchema(t, aComplexType(t, uses, xsd.EmptyContent{}, aAssertions("@x > 300")))

	wantRecords(t, aAssess(t, schema, aRoot()), "cvc-assertion", loc(1, 1), "XPath evaluator declined it")
	wantAssertionCharge(t, aAssess(t, schema, aRoot("x", "200")), "the element root is not ·valid·")
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

// An ·initial value· String Valid REJECTS still has its assertion sites recorded:
// the recording precedes the verdict, so a charge does not cost the site the way
// returning after the verdict would.
func TestRejectedInitialValueStillRecordsItsSites(t *testing.T) {
	types := aVarietyTypes(t)
	schema := aSchema(t, aComplexType(t, nil, xsd.SimpleContent{SimpleType: types[0]}, nil), types...)

	res := aAssess(t, schema, cRoot("#not an integer"))

	if len(res.Violations()) != 1 {
		t.Fatalf("Violations() = %v, want the clause 1.2 charge", res.Violations())
	}
	got := res.Unevaluated()
	if len(got) != 1 || !strings.Contains(got[0].Msg(), "AssertedInt") {
		t.Fatalf("Unevaluated() = %v, want the AssertedInt site recorded despite the charge", messages(got))
	}
}

// The two rule IDs are never conflated: one element carrying both an
// {assertions} on its complex type and an assertions facet on its simple
// {content type} records one site per rule, at the same Loc, discriminated by
// Rule alone. The facet's comes first: cvc-complex-type clause 1.2 reaches it
// over the ·initial value· once the [[children]] are exhausted, and clause 6
// is evaluated after them, over the partial ·PSVI· clause 1.2 is part of
// (cvc-assertion clause 1.1).
func TestBothRulesRecordAtOneElement(t *testing.T) {
	types := aVarietyTypes(t)
	schema := aSchema(t, aComplexType(t, nil, xsd.SimpleContent{SimpleType: types[0]},
		aAssertions("@a = @b")), types...)

	res := aAssess(t, schema, cRoot("#42"))

	got := res.Unevaluated()
	if len(got) != 2 {
		t.Fatalf("Unevaluated() = %v, want one site per rule", messages(got))
	}
	if got[0].Rule() != "cvc-assertions-valid" || got[1].Rule() != "cvc-assertion" {
		t.Errorf("rules = %q, %q, want cvc-assertions-valid then cvc-assertion",
			got[0].Rule(), got[1].Rule())
	}
}

// An attribute matching no {attribute use} is ·attributed to· the {attribute
// wildcard} instead, and under a strict or a lax wildcard it is assessed
// against the top-level declaration its name ·resolves· to
// ([walk.wildcardAttribute]), whose cvc-attribute clause 3 records the type's
// sites exactly ONCE — wantRecords counts them, so a second recording path
// beside [walk.declaredAttribute] fails it (#1891).
//
// Under ***skip*** nothing is recorded: the item is ·skipped·, no facet of any
// type is reached over its lexical (#1043), and there is no unevaluated
// assertion to report. The recording ran under skip as well while the
// ·attribution· could only be inferred from the wildcard's presence; #717
// decides it, so the over-report is gone.
func TestWildcardAttributeAssertionsAreRecorded(t *testing.T) {
	d, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, local("n"),
		xsd.TypeDefinitionRef{Name: local("AssertedInt")}, xsd.NewAttributeGlobalScope(), nil, false)
	if err != nil {
		t.Fatalf("building the top-level n declaration: %v", err)
	}
	assess := func(pc xsd.ProcessContents) *Result {
		t.Helper()
		ct, err := xsd.NewComplexType(xsderr.Loc{}, local("RootType"), xsd.QName{}, nil,
			xsd.DerivationRestriction, false, nil, nil, anyWildcard(t, pc), xsd.EmptyContent{}, nil, nil)
		if err != nil {
			t.Fatalf("building RootType: %v", err)
		}
		schema := cSchemaFrom(t, ct, func(b *xsd.SchemaBuilder) {
			aTypes(t, b, aVarietyTypes(t)...)
			b.AddAttribute(d)
		})
		return aAssess(t, schema, valuedRoot("n", "42"))
	}

	wantRecords(t, assess(xsd.ProcessStrict), "cvc-assertions-valid", loc(1, 10), "AssertedInt")
	wantRecords(t, assess(xsd.ProcessLax), "cvc-assertions-valid", loc(1, 10), "AssertedInt")
	if got := assess(xsd.ProcessSkip).Unevaluated(); len(got) != 0 {
		t.Errorf("Unevaluated() = %v, want none: a ·skipped· attribute reaches no assertion site", messages(got))
	}
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
