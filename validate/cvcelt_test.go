package validate

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The fixtures below drive cvc-elt (§3.3.4.3) clauses 3, 4 and 5.2.2 — the two
// instance attributes that change what the rest of the assessment reads.
//
// eSchema declares "root" over Base, an EMPTY {content type}, with two more
// top-level types beside it:
//
//	Base       empty content
//	Derived    extension of Base, element-only sequence( a )
//	Unrelated  element-only sequence( a ), derived from nothing
//
// So an xsi:type of Derived ·overrides· Base and admits an <a> that Base
// rejects, and one of Unrelated ·resolves· but does not ·override·.

// eType builds a NAMED complex type over content, derived from base by method.
func eType(t *testing.T, name, base string, method xsd.DerivationMethod, content xsd.ContentType) xsd.ComplexType {
	t.Helper()
	var baseName xsd.QName
	if base != "" {
		baseName = xsd.QName{Local: base}
	}
	ct, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: name}, baseName, nil,
		method, false, nil, nil, nil, content, nil, nil, nil)
	if err != nil {
		t.Fatalf("building the %s complex type: %v", name, err)
	}
	return ct
}

// eSchema declares "root" over Base, with the {nillable}, {value constraint}
// and {disallowed substitutions} the fixture asks for.
func eSchema(t *testing.T, nillable bool, vc *xsd.ValueConstraint, blocked ...xsd.DerivationMethod) *xsd.Schema {
	t.Helper()
	b := xsd.NewSchemaBuilder()
	seeded, err := builtin.Seed(testBackend())
	if err != nil {
		t.Fatalf("seeding the builtin types: %v", err)
	}
	for _, st := range seeded {
		b.AddType(st)
	}
	b.AddType(eType(t, "Base", "", xsd.DerivationRestriction, xsd.EmptyContent{}))
	b.AddType(eType(t, "Derived", "Base", xsd.DerivationExtension,
		cSequence(t, false, cParticle(t, "a", 1, 1))))
	b.AddType(eType(t, "Unrelated", "", xsd.DerivationRestriction,
		cSequence(t, false, cParticle(t, "a", 1, 1))))
	d, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "root"},
		xsd.TypeDefinitionRef{Name: xsd.QName{Local: "Base"}}, nil, xsd.NewGlobalScope(),
		vc, nillable, nil, nil, nil, false, blocked, nil)
	if err != nil {
		t.Fatalf("building the root element declaration: %v", err)
	}
	b.AddElement(d)
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the cvc-elt schema: %v", err)
	}
	return schema
}

// eRoot is a <root> carrying the given xsi: attributes (by local name) and the
// given [[children]], on cRoot's spelling.
func eRoot(xsiAttrs map[string]string, kids ...string) *testElement {
	e := cRoot(kids...)
	for _, local := range []string{"type", "nil"} {
		v, present := xsiAttrs[local]
		if !present {
			continue
		}
		e.attrs = append(e.attrs, &testAttribute{
			name:  xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: local},
			value: v, loc: loc(1, 10)})
	}
	return e
}

// eWantClause fails unless the violations are exactly one cvc-elt charge naming
// clause inline.
func eWantClause(t *testing.T, got []*xsderr.Error, clause string) {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("Violations() = %v, want exactly one", got)
	}
	if got[0].Rule != "cvc-elt" {
		t.Errorf("Rule = %q, want cvc-elt", got[0].Rule)
	}
	if !strings.Contains(got[0].Msg, "cvc-elt clause "+clause) {
		t.Errorf("Msg = %q, want it to name cvc-elt clause %s inline (STYLE E4)", got[0].Msg, clause)
	}
}

// Clause 3.1 quantifies over PRESENCE: a declaration whose {nillable} is false
// admits no xsi:nil attribute at all, whatever its ·actual value·, so both the
// true and the false lexical are charged — and an element carrying none is not.
func TestNonNillableDeclarationChargesAnyXSINil(t *testing.T) {
	schema := eSchema(t, false, nil)

	for _, lexical := range []string{"true", "false", "1", "0", "TRUE"} {
		eWantClause(t, cAssess(t, schema, eRoot(map[string]string{"nil": lexical})), "3.1")
	}
	wantSilence(t, cAssess(t, schema, eRoot(nil)), "an element with no xsi:nil satisfies clause 3.1")
}

// Clause 3.2's three arms are no attribute, the value false and the value true.
// A lexical outside xs:boolean's lexical space satisfies none of them, and the
// element is charged on clause 3.2 itself: there is no arm a readable value
// would have failed.
func TestNillableDeclarationChargesAnUnreadableXSINil(t *testing.T) {
	schema := eSchema(t, true, nil)

	eWantClause(t, cAssess(t, schema, eRoot(map[string]string{"nil": "TRUE"})), "3.2")
	eWantClause(t, cAssess(t, schema, eRoot(map[string]string{"nil": ""})), "3.2")
	wantSilence(t, cAssess(t, schema, eRoot(map[string]string{"nil": "false"})),
		"xsi:nil = false satisfies clause 3.2.2")
}

// xsi:nil is typed by the built-in xs:boolean, whose whiteSpace is a fixed
// collapse, so a padded lexical is the same ·actual value· as a bare one.
func TestXSINilLexicalIsCollapsed(t *testing.T) {
	schema := eSchema(t, true, nil)

	wantSilence(t, cAssess(t, schema, eRoot(map[string]string{"nil": " \t true \n "})),
		"a collapsed \"true\" is xs:boolean's true, so the element is ·nilled· and empty")
}

// Clause 3.2.3.1 admits no character or element information item [[children]] on
// a ·nilled· element, and the charge cites the offending child's own position.
func TestNilledElementChargesAnyContent(t *testing.T) {
	schema := eSchema(t, true, nil)

	got := cAssess(t, schema, eRoot(map[string]string{"nil": "true"}, "a"))
	eWantClause(t, got, "3.2.3.1")
	if got[0].Loc != loc(2, 1) {
		t.Errorf("Loc = %s, want the offending child's position %s", got[0].Loc, loc(2, 1))
	}

	// Character content counts, white space included: clause 3.2.3.1 states no
	// exception of any kind, exactly as cvc-complex-type clause 1.1 does not.
	eWantClause(t, cAssess(t, schema, eRoot(map[string]string{"nil": "true"}, "#  ")), "3.2.3.1")
}

// Clause 3.2.3.2 forbids a ·nilled· element's declaration to carry a {value
// constraint} with {variety} = fixed. A DEFAULT one is admitted: the clause
// names fixed alone.
func TestNilledElementChargesAFixedValueConstraint(t *testing.T) {
	fixed := xsd.NewValueConstraint(xsd.ValueFixed, "1", nil, nil)
	dflt := xsd.NewValueConstraint(xsd.ValueDefault, "1", nil, nil)
	nilledRoot := eRoot(map[string]string{"nil": "true"})

	eWantClause(t, cAssess(t, simpleTypedSchema(t, icBuiltin("integer"), &fixed, true), nilledRoot), "3.2.3.2")
	wantSilence(t, cAssess(t, simpleTypedSchema(t, icBuiltin("integer"), &dflt, true), nilledRoot),
		"clause 3.2.3.2 names a FIXED {value constraint} and no other")
}

// An xsi:type that ·overrides· the ·selected type definition· becomes the
// ·governing type definition· for the rest of the assessment: content the
// declared type rejects is admitted by the substituted one, and rejected again
// without the attribute.
func TestOverridingXSITypeChangesTheGoverningType(t *testing.T) {
	schema := eSchema(t, false, nil)

	wantSilence(t, cAssess(t, schema, eRoot(map[string]string{"type": "Derived"}, "a")),
		"Derived is validly substitutable for Base and its {content type} admits <a>")
	if got := cAssess(t, schema, eRoot(nil, "a")); len(got) != 1 {
		t.Fatalf("Violations() = %v, want exactly one: Base's empty {content type} admits no <a>", got)
	}
}

// A ·resolved· xsi:type that is not ·validly substitutable· for the ·selected
// type definition· fails clause 4, and the element is then assessed against the
// SELECTED type — the fallback the Note under cvc-elt states, and
// key-governing-type-elem clause 4.
func TestNonOverridingXSITypeIsChargedAndFallsBack(t *testing.T) {
	schema := eSchema(t, false, nil)

	eWantClause(t, cAssess(t, schema, eRoot(map[string]string{"type": "Unrelated"})), "4")

	// Assessed against Base: <a> is charged by Base's empty {content type},
	// which Unrelated's own would have admitted.
	got := cAssess(t, schema, eRoot(map[string]string{"type": "Unrelated"}, "a"))
	if len(got) != 2 {
		t.Fatalf("Violations() = %v, want two: the clause 4 charge and Base's clause 1.1", got)
	}
	if got[1].Rule != "cvc-complex-type" {
		t.Errorf("Rule = %q, want the second charge to be cvc-complex-type against the SELECTED type", got[1].Rule)
	}
}

// {disallowed substitutions} on the ·governing element declaration· is the
// blocking set ·overrides· works under (key-overrides clause 2), so a
// declaration blocking extension rejects the very type it admitted without it.
func TestDisallowedSubstitutionsBlockTheOverride(t *testing.T) {
	blocking := eSchema(t, false, nil, xsd.DerivationExtension)

	eWantClause(t, cAssess(t, blocking, eRoot(map[string]string{"type": "Derived"})), "4")
}

// An xsi:type that does not ·resolve· leaves E with no ·instance-specified type
// definition· at all (§3.3.4.1 key-itd clause 3), so clause 4's antecedent is
// false and the clause is VACUOUSLY satisfied — the Note under cvc-elt
// resolving W3C issue 11764. The element falls back to the selected type
// silently.
func TestUnresolvedXSITypeIsNotChargedAtAll(t *testing.T) {
	schema := eSchema(t, false, nil)

	for _, lexical := range []string{
		"Missing",     // a name nothing declares
		"p:Derived",   // an unbound prefix
		"a:b:Derived", // not a QName at all
		":Derived",    // an empty prefix
		"Derived:",    // an empty local part
		"",            // empty
	} {
		wantSilence(t, cAssess(t, schema, eRoot(map[string]string{"type": lexical})),
			"an xsi:type of "+lexical+" ·resolves· to nothing, which charges no clause")
	}
}

// Clause 5.2.2.1 admits no element information item [[children]] on an element
// whose declaration carries a FIXED {value constraint}. The clause is reached
// only through clause 5.2, which an element WITH [[children]] takes; an empty
// one takes clause 5.1's arm and is not charged here.
//
// The fixture is MIXED content, because that is the shape 5.2.2.1 can be
// observed on: cos-valid-default clause 2.1 admits a fixed {value constraint}
// for mixed content and for a simple type alone, and under a SIMPLE one the
// element child is charged where it SITS by cvc-type clause 3.1.2 first, one
// element carrying at most one content charge (cvccomplexcontent.go). Both say
// the same thing of the same item.
func TestFixedValueConstraintChargesElementChildren(t *testing.T) {
	fixed := xsd.NewValueConstraint(xsd.ValueFixed, "1", nil, nil)
	mixed := nillableFixedMixedSchema(t, cSequence(t, true, cParticle(t, "a", 0, 1)), &fixed)

	eWantClause(t, cAssess(t, mixed, cRoot("a")), "5.2.2.1")

	simple := simpleTypedSchema(t, icBuiltin("integer"), &fixed, false)
	wantContentCharge(t, cAssess(t, simple, cRoot("a")), "cvc-type", "3.1.2", loc(2, 1))
	wantSilence(t, cAssess(t, simple, cRoot()), "an empty element takes clause 5.1's arm")
}

// Clause 5.2.2.2.2 compares ·actual values· for a simple ·governing type
// definition·, so an equal value written differently agrees and an unequal one
// does not. It is skipped for a ·nilled· element, whose fixed-constraint
// interaction clause 3.2.3.2 already decided.
func TestFixedValueConstraintComparesActualValues(t *testing.T) {
	fixed := xsd.NewValueConstraint(xsd.ValueFixed, "01", nil, nil)
	schema := simpleTypedSchema(t, icBuiltin("integer"), &fixed, false)

	wantSilence(t, cAssess(t, schema, cRoot("#1")), "1 and 01 are one xs:integer ·actual value·")
	eWantClause(t, cAssess(t, schema, cRoot("#2")), "5.2.2.2.2")
}

// simpleTypedSchema declares "root" over a builtin SIMPLE type, with the given
// {value constraint} and {nillable}.
func simpleTypedSchema(t *testing.T, typ xsd.QName, vc *xsd.ValueConstraint, nillable bool) *xsd.Schema {
	t.Helper()
	b := xsd.NewSchemaBuilder()
	seeded, err := builtin.Seed(testBackend())
	if err != nil {
		t.Fatalf("seeding the builtin types: %v", err)
	}
	for _, st := range seeded {
		b.AddType(st)
	}
	d, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "root"},
		xsd.TypeDefinitionRef{Name: typ}, nil, xsd.NewGlobalScope(),
		vc, nillable, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("building the root element declaration: %v", err)
	}
	b.AddElement(d)
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the simple-typed schema: %v", err)
	}
	return schema
}

// Clause 5.2.2.2.1 compares LEXICALS for a MIXED {content type}: a mixed type
// has no {simple type definition} to map either side through, so the ·initial
// value· has to match the {lexical form} as written.
func TestFixedValueConstraintMatchesAMixedLexical(t *testing.T) {
	fixed := xsd.NewValueConstraint(xsd.ValueFixed, "01", nil, nil)
	mixed := cSequence(t, true, cParticle(t, "a", 0, 0))
	schema := nillableFixedMixedSchema(t, mixed, &fixed)

	wantSilence(t, cAssess(t, schema, cRoot("#01")), "the ·initial value· matches the {lexical form}")
	eWantClause(t, cAssess(t, schema, cRoot("#1")), "5.2.2.2.1")
}

// nillableFixedMixedSchema declares "root" over a MIXED complex type carrying
// the given {value constraint}, which cos-valid-default clause 2.1 admits for
// mixed content alone.
func nillableFixedMixedSchema(t *testing.T, content xsd.ContentType, vc *xsd.ValueConstraint) *xsd.Schema {
	t.Helper()
	b := xsd.NewSchemaBuilder()
	b.AddType(eType(t, "MixedType", "", xsd.DerivationRestriction, content))
	d, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "root"},
		xsd.TypeDefinitionRef{Name: xsd.QName{Local: "MixedType"}}, nil, xsd.NewGlobalScope(),
		vc, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("building the root element declaration: %v", err)
	}
	b.AddElement(d)
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the mixed schema: %v", err)
	}
	return schema
}

// A ·nilled· element skips clause 5.2.2 entirely — the clause is gated on "E is
// not ·nilled·" — so a nillable declaration with a fixed constraint charges
// clause 3.2.3.2 and nothing from clause 5.
func TestNilledElementSkipsTheFixedValueCheck(t *testing.T) {
	fixed := xsd.NewValueConstraint(xsd.ValueFixed, "01", nil, nil)
	schema := simpleTypedSchema(t, icBuiltin("integer"), &fixed, true)
	root := cRoot()
	root.attrs = []Attribute{&testAttribute{
		name:  xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "nil"},
		value: "true", loc: loc(1, 10)}}

	eWantClause(t, cAssess(t, schema, root), "3.2.3.2")
}
