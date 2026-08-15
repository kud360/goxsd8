package validate

import (
	"slices"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The fixtures below drive the charges that need a value space: cvc-attribute
// (§3.2.4.1) clauses 3 and 4, cvc-au (§3.5.4), and cvc-complex-type (§3.4.4.2)
// clause 4. Unlike cvccomplextype_test.go's, every schema here SEEDS the
// builtin simple types, because a declaration whose {type definition} names no
// component in the schema resolves to nothing and declines before any of them
// is reached.
//
// xs:integer is the type throughout: strict maps it, its lexical space rejects
// "abc" outright, and "01" and "1" are one value under two lexical forms —
// which is the whole difference between the value-space comparison these rules
// demand and the lexical one they forbid.

func integerType() xsd.QName { return xsd.QName{Space: xsd.XMLSchemaNS, Local: "integer"} }

// typedUse builds an attribute use over a sibling local declaration of type
// typ, carrying the DECLARATION's {value constraint} (declVC, read by
// cvc-attribute clause 4) and the USE's own (useVC, read by cvc-au). The two
// are separate parameters because the two rules read separate properties, and
// a fixture that could not set them independently could not tell the charges
// apart.
func typedUse(t *testing.T, local string, typ xsd.QName, required bool, declVC, useVC *xsd.ValueConstraint) xsd.AttributeUse {
	t.Helper()
	decl, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, xsd.QName{Local: local},
		xsd.TypeDefinitionRef{Name: typ}, xsd.NewAttributeGlobalScope(), declVC, false, nil)
	if err != nil {
		t.Fatalf("building the %s attribute declaration: %v", local, err)
	}
	u, err := xsd.NewAttributeUse(xsderr.Loc{}, required,
		xsd.LocalAttributeDeclaration{Declaration: decl}, useVC, false, nil)
	if err != nil {
		t.Fatalf("building the %s attribute use: %v", local, err)
	}
	return u
}

// typedSchema is governedSchema with the builtin simple types seeded, so a
// declaration naming xs:integer resolves to the real component and its facets
// are the spec's.
//
// It finalizes through Finalize and not FinalizeWith: no value space is
// installed, so cos-valid-simple-default (§3.2.6.2) is waved through at
// assembly and a fixture may carry a {value constraint} whose {lexical form} is
// invalid — which is exactly the state cvc-complex-type clause 4 exists to
// catch at assessment time.
func typedSchema(t *testing.T, uses []xsd.AttributeUse) *xsd.Schema {
	t.Helper()
	ct, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "RootType"}, xsd.QName{}, nil,
		xsd.DerivationRestriction, false, uses, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("building RootType: %v", err)
	}
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "root"},
		xsd.TypeDefinitionRef{Name: xsd.QName{Local: "RootType"}}, nil, xsd.NewGlobalScope(),
		nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("building the root element declaration: %v", err)
	}
	seeded, err := builtin.Seed(testBackend())
	if err != nil {
		t.Fatalf("seeding the builtin types: %v", err)
	}
	b := xsd.NewSchemaBuilder()
	for _, st := range seeded {
		b.AddType(st)
	}
	b.AddType(ct)
	b.AddElement(e)
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the typed schema: %v", err)
	}
	return schema
}

// valuedRoot is a childless <root> carrying one attribute of the given name and
// lexical value, at the Loc every assertion below cites.
func valuedRoot(name string, lexical string) *testElement {
	return &testElement{
		name:  xsd.QName{Local: "root"},
		attrs: []Attribute{&testAttribute{name: local(name), value: lexical, loc: loc(1, 10)}},
		loc:   loc(1, 1),
	}
}

// assessTyped assesses root against a schema declaring "root" over uses, with
// the builtin types seeded.
func assessTyped(t *testing.T, root Element, uses []xsd.AttributeUse) []*xsderr.Error {
	t.Helper()
	v, err := New(typedSchema(t, uses), testBackend())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	res := v.Assess(root)
	if res.Err() != nil {
		t.Fatalf("Err() = %v, want nil", res.Err())
	}
	return res.Violations()
}

// onlyCharge fails unless exactly one violation was charged under rule, and
// returns it.
func onlyCharge(t *testing.T, got []*xsderr.Error, rule xsderr.Rule) *xsderr.Error {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("Violations() = %v, want exactly one %s charge", got, rule)
	}
	if got[0].Rule != rule {
		t.Fatalf("Rule = %q, want %q", got[0].Rule, rule)
	}
	return got[0]
}

// cvc-attribute clause 3: a lexical outside the declaration's {type
// definition}'s lexical space is charged, and one inside it is not. This is the
// charge the whole seam exists for.
func TestAttributeLexicalIsCheckedAgainstItsType(t *testing.T) {
	uses := []xsd.AttributeUse{typedUse(t, "n", integerType(), false, nil, nil)}

	got := onlyCharge(t, assessTyped(t, valuedRoot("n", "abc"), uses), "cvc-attribute")
	if got.Loc != loc(1, 10) {
		t.Errorf("Loc = %s, want the attribute's %s", got.Loc, loc(1, 10))
	}
	if !strings.Contains(got.Msg, "clause 3") || !strings.Contains(got.Msg, "String Valid") {
		t.Errorf("Msg = %q, want clause 3 and its String Valid delegation named", got.Msg)
	}

	wantSilence(t, assessTyped(t, valuedRoot("n", "42"), uses), "42 is an xs:integer")
	// String Valid clause 1 normalizes before Datatype Valid runs, so a
	// lexical the whiteSpace facet collapses to a valid one is valid.
	wantSilence(t, assessTyped(t, valuedRoot("n", "  42  "), uses),
		"whiteSpace normalization precedes datatype validation (String Valid clause 1)")
}

// The regression the classification predicate exists for: an attribute
// declaration with NO @type is xs:anySimpleType (§3.2.2.2's third tier), which
// no backend maps, and value.ValidateLexical reports that under
// cvc-datatype-valid exactly as it reports a real rejection. Charging it would
// reject every typeless attribute in existence.
func TestTypelessAttributeIsNeverCharged(t *testing.T) {
	anySimple := xsd.QName{Space: xsd.XMLSchemaNS, Local: "anySimpleType"}
	for _, typ := range []xsd.QName{anySimple, {Space: xsd.XMLSchemaNS, Local: "anyAtomicType"}} {
		uses := []xsd.AttributeUse{typedUse(t, "n", typ, false, nil, nil)}
		wantSilence(t, assessTyped(t, valuedRoot("n", "anything at all"), uses),
			"an ungoverned type is a backend gap, not a verdict about the lexical")
	}
}

// cvc-attribute clause 4 compares ·actual values·: "01" and "1" are one
// xs:integer, so a fixed constraint written one way is satisfied by the other
// spelling, while a genuinely different value is charged.
func TestDeclarationFixedIsComparedInTheValueSpace(t *testing.T) {
	fixed := xsd.NewValueConstraint(xsd.ValueFixed, "1", nil, nil)
	uses := []xsd.AttributeUse{typedUse(t, "n", integerType(), false, &fixed, nil)}

	wantSilence(t, assessTyped(t, valuedRoot("n", "01"), uses),
		`"01" and "1" are one xs:integer value; a lexical comparison would reject this`)

	got := onlyCharge(t, assessTyped(t, valuedRoot("n", "2"), uses), "cvc-attribute")
	if !strings.Contains(got.Msg, "clause 4") || !strings.Contains(got.Msg, "attribute declaration") {
		t.Errorf("Msg = %q, want clause 4 charged against the DECLARATION's {value constraint}", got.Msg)
	}

	// A DEFAULT {value constraint} constrains a present attribute not at all:
	// clause 4 tests the fixed {variety} alone.
	dflt := xsd.NewValueConstraint(xsd.ValueDefault, "1", nil, nil)
	wantSilence(t, assessTyped(t, valuedRoot("n", "2"),
		[]xsd.AttributeUse{typedUse(t, "n", integerType(), false, &dflt, nil)}),
		"a default {value constraint} constrains no attribute that is present")
}

// cvc-au reads the USE's own {value constraint} and cvc-attribute clause 4 the
// DECLARATION's. They are independent rules over two different properties, so a
// use-only fixed value charges cvc-au alone — and an attribute disagreeing with
// both is charged twice, declaration first.
func TestUseFixedChargesCvcAuIndependently(t *testing.T) {
	useFixed := xsd.NewValueConstraint(xsd.ValueFixed, "1", nil, nil)

	got := onlyCharge(t, assessTyped(t, valuedRoot("n", "2"),
		[]xsd.AttributeUse{typedUse(t, "n", integerType(), false, nil, &useFixed)}), "cvc-au")
	if got.Loc != loc(1, 10) {
		t.Errorf("Loc = %s, want the attribute's %s", got.Loc, loc(1, 10))
	}
	if !strings.Contains(got.Msg, "attribute use") {
		t.Errorf("Msg = %q, want the USE's {value constraint} named", got.Msg)
	}

	declFixed := xsd.NewValueConstraint(xsd.ValueFixed, "1", nil, nil)
	both := assessTyped(t, valuedRoot("n", "2"),
		[]xsd.AttributeUse{typedUse(t, "n", integerType(), false, &declFixed, &useFixed)})
	rules := make([]xsderr.Rule, 0, len(both))
	for _, e := range both {
		rules = append(rules, e.Rule)
	}
	if !slices.Equal(rules, []xsderr.Rule{"cvc-attribute", "cvc-au"}) {
		t.Errorf("charged %v, want both rules, the declaration's clause 4 first", rules)
	}
}

// An attribute that is not datatype-valid has no ·actual value·, so the two
// fixed-agreement rules are not also charged for the one defect: clause 3 is
// the whole verdict.
func TestAnInvalidLexicalChargesClauseThreeAlone(t *testing.T) {
	fixed := xsd.NewValueConstraint(xsd.ValueFixed, "1", nil, nil)
	got := assessTyped(t, valuedRoot("n", "abc"),
		[]xsd.AttributeUse{typedUse(t, "n", integerType(), false, &fixed, &fixed)})
	charge := onlyCharge(t, got, "cvc-attribute")
	if !strings.Contains(charge.Msg, "clause 3") {
		t.Errorf("Msg = %q, want clause 3 alone", charge.Msg)
	}
}

// cvc-complex-type clause 4: an OPTIONAL use the instance did not match, whose
// ·effective value constraint· is not ·absent·, has that constraint's own
// {lexical form} validated against the declaration's {type definition}. The
// charge sits on the ELEMENT, since no attribute information item exists to
// carry it.
func TestDefaultedAttributeDefaultIsValidated(t *testing.T) {
	bad := xsd.NewValueConstraint(xsd.ValueDefault, "abc", nil, nil)
	root := &testElement{name: xsd.QName{Local: "root"}, loc: loc(1, 1)}

	got := onlyCharge(t, assessTyped(t, root,
		[]xsd.AttributeUse{typedUse(t, "n", integerType(), false, &bad, nil)}), "cvc-complex-type")
	if got.Loc != loc(1, 1) {
		t.Errorf("Loc = %s, want the element's %s", got.Loc, loc(1, 1))
	}
	if !strings.Contains(got.Msg, "clause 4") {
		t.Errorf("Msg = %q, want clause 4", got.Msg)
	}

	// The use's own {value constraint} wins over the declaration's, which is
	// what makes this the ·effective value constraint· (§3.5.4 key-evc) and not
	// either property read alone.
	good := xsd.NewValueConstraint(xsd.ValueDefault, "1", nil, nil)
	wantSilence(t, assessTyped(t, root,
		[]xsd.AttributeUse{typedUse(t, "n", integerType(), false, &bad, &good)}),
		"the use's own {value constraint} is the effective one")
}

// The four conjuncts of ·defaulted attribute· that exclude a use from clause 4,
// each on its own. None of these charges, though every one of them carries a
// {lexical form} that is not datatype-valid.
func TestNonDefaultedAttributesEscapeClauseFour(t *testing.T) {
	bad := xsd.NewValueConstraint(xsd.ValueDefault, "abc", nil, nil)
	badFixed := xsd.NewValueConstraint(xsd.ValueFixed, "abc", nil, nil)
	bare := &testElement{name: xsd.QName{Local: "root"}, loc: loc(1, 1)}

	// clause 2: {required} = true. Its own absence is clause 3's charge, not
	// clause 4's, so the assertion is that no CLAUSE 4 charge joins it.
	got := assessTyped(t, bare, []xsd.AttributeUse{typedUse(t, "n", integerType(), true, &bad, nil)})
	charge := onlyCharge(t, got, "cvc-complex-type")
	if !strings.Contains(charge.Msg, "clause 3") {
		t.Errorf("Msg = %q, want clause 3 alone: a {required} use is never a ·defaulted attribute·", charge.Msg)
	}

	// clause 3: an ·absent· ·effective value constraint· — neither property set.
	wantSilence(t, assessTyped(t, bare, []xsd.AttributeUse{typedUse(t, "n", integerType(), false, nil, nil)}),
		"a use with no value constraint at all is not a ·defaulted attribute·")

	// clause 5: the instance DID carry a matching attribute, so the default was
	// never supplied. The present attribute is valid, so nothing else fires.
	wantSilence(t, assessTyped(t, valuedRoot("n", "42"),
		[]xsd.AttributeUse{typedUse(t, "n", integerType(), false, &badFixed, nil)}),
		"a matched declaration supplies no default")
}

// The debug log names cvc-attribute and cvc-au and the clause each settled, so
// a DECLINE is distinguishable from a PASS, which charge the same nothing.
func TestValueChargeOutcomesAreLogged(t *testing.T) {
	fixed := xsd.NewValueConstraint(xsd.ValueFixed, "1", nil, nil)
	log, visits := recordingLogger()
	v, err := New(typedSchema(t, []xsd.AttributeUse{typedUse(t, "n", integerType(), false, &fixed, &fixed)}),
		testBackend(), WithLogger(log))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	v.Assess(valuedRoot("n", "01"))

	want := []string{
		"assessing element validate.name=root validate.loc=instance.xml:1:1",
		"assessing attribute validate.name=n validate.loc=instance.xml:1:10 validate.rule=cvc-attribute validate.clause=3 validate.outcome=satisfied",
		"assessing attribute validate.name=n validate.loc=instance.xml:1:10 validate.rule=cvc-attribute validate.clause=4 validate.outcome=satisfied",
		// cvc-au names no clause: the rule is one undivided sentence.
		"assessing attribute validate.name=n validate.loc=instance.xml:1:10 validate.rule=cvc-au validate.outcome=satisfied",
	}
	if !slices.Equal(*visits, want) {
		t.Errorf("walk logged\n\t%s\nwant\n\t%s",
			strings.Join(*visits, "\n\t"), strings.Join(want, "\n\t"))
	}
}
