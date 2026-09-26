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
		xsd.TypeDefinitionRef{Name: typ}, xsd.NewAttributeGlobalScope(), declVC, false)
	if err != nil {
		t.Fatalf("building the %s attribute declaration: %v", local, err)
	}
	u, err := xsd.NewAttributeUse(xsderr.Loc{}, required,
		xsd.LocalAttributeDeclaration{Declaration: decl}, useVC, nil)
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
// catch at assessment time. extra adds the fixture's own named simple types
// beside the builtin ones.
func typedSchema(t *testing.T, uses []xsd.AttributeUse, extra ...*xsd.SimpleType) *xsd.Schema {
	t.Helper()
	ct, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "RootType"}, xsd.QName{}, nil,
		xsd.DerivationRestriction, false, attrContent(uses), nil, nil, xsd.EmptyContent{}, nil, nil)
	if err != nil {
		t.Fatalf("building RootType: %v", err)
	}
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "root"},
		xsd.TypeDefinitionRef{Name: xsd.QName{Local: "RootType"}}, nil, xsd.NewGlobalScope(),
		nil, false, nil, nil, nil, false, nil)
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
	for _, st := range extra {
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
// the builtin types and extra seeded.
func assessTyped(t *testing.T, root Element, uses []xsd.AttributeUse, extra ...*xsd.SimpleType) []*xsderr.Error {
	t.Helper()
	v, err := New(typedSchema(t, uses, extra...), testBackend())
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

// cvc-attribute (§3.2.4.1) clause 5 charges an xsi:type attribute whose
// ·actual value· ·resolves· to no type definition, at the ATTRIBUTE's own
// position. It is the built-in declaration for the type attribute (§3.2.7.1)
// charging on its own: no attribute use ever matches xsi:type, so
// [walk.matchedAttribute] never sees the item and none of the clauses above
// reaches it (#1494). The prefix is BOUND, so the resolved name and the lexical
// the message carries are different strings and neither can stand in for the
// other: two arguments swapped inside the fmt.Errorf leaves every asserted
// substring present, and only pinning the opening as a prefix catches it
// (#1048).
func TestUnresolvableXSITypeChargesClauseFive(t *testing.T) {
	schema := eSchema(t, false, nil)
	root := eRoot(map[string]string{"type": "p:Missing"})
	root.bindings = map[string]string{"p": "urn:x"}

	got := cAssess(t, schema, root)

	if len(got) != 1 {
		t.Fatalf("Violations() = %v, want exactly one", got)
	}
	if got[0].Rule != "cvc-attribute" {
		t.Errorf("Rule = %q, want cvc-attribute", got[0].Rule)
	}
	if got[0].Loc != loc(1, 10) {
		t.Errorf("Loc = %s, want the xsi:type attribute's own position %s", got[0].Loc, loc(1, 10))
	}
	const opening = `the xsi:type attribute of the element root has the lexical "p:Missing", ` +
		`and the schema declares no type definition named "{urn:x}Missing" for it to ·resolve· to`
	if !strings.HasPrefix(got[0].Msg, opening) {
		t.Errorf("Msg = %q, want it to open %q", got[0].Msg, opening)
	}
	if !strings.Contains(got[0].Msg, "cvc-attribute clause 5") {
		t.Errorf("Msg = %q, want it to name cvc-attribute clause 5 inline (STYLE E4)", got[0].Msg)
	}

	wantSilence(t, cAssess(t, schema, eRoot(map[string]string{"type": "Derived"}, "a")),
		"an xsi:type that ·resolves· satisfies clause 5")
}

// Clause 5 leaves the ELEMENT where the Note under cvc-elt puts it: assessed
// against its ·selected type definition·, whose own charges arrive beside the
// attribute's and are not displaced by it. Base's {content type} is EMPTY, so
// the <a> is charged by the fallback type exactly as it is without any xsi:type
// at all.
func TestClauseFiveLeavesTheFallbackTypeGoverning(t *testing.T) {
	schema := eSchema(t, false, nil)

	got := cAssess(t, schema, eRoot(map[string]string{"type": "Missing"}, "a"))

	if len(got) != 2 {
		t.Fatalf("Violations() = %v, want two: the clause 5 charge and Base's empty {content type}", got)
	}
	if got[0].Rule != "cvc-attribute" || got[1].Rule != "cvc-complex-type" {
		t.Errorf("Rules = %q, %q, want cvc-attribute then cvc-complex-type against the SELECTED type",
			got[0].Rule, got[1].Rule)
	}
}

// The charge goes through neither arm of cvc-type clause 3, so a SIMPLE
// ·governing type definition· reaches it too — the shape #1494 was filed on,
// where the declared type is xs:string and cvc-complex-type never runs. The
// character content still validates against that fallback type, so clause 5's
// charge is the whole of what the document is told.
func TestClauseFiveChargesUnderASimpleGoverningType(t *testing.T) {
	schema := simpleTypedSchema(t, icBuiltin("string"), nil, false)

	got := cAssess(t, schema, eRoot(map[string]string{"type": "Missing"}, "#hello"))

	if len(got) != 1 {
		t.Fatalf("Violations() = %v, want exactly one: clause 5, with xs:string admitting the content", got)
	}
	if got[0].Rule != "cvc-attribute" || !strings.Contains(got[0].Msg, "cvc-attribute clause 5") {
		t.Errorf("charge = %v, want cvc-attribute clause 5", got[0])
	}
	wantSilence(t, cAssess(t, schema, eRoot(nil, "#hello")),
		"an element carrying no xsi:type at all has no ·actual value· for clause 5 to quantify over")
}

// cvc-attribute (§3.2.4.1) clause 3 charges an xsi:type lexical that is not
// String Valid against xs:QName, the built-in declaration's {type definition}
// (§3.2.7.1), and clause 5 is declined for it: such a lexical has no ·actual
// value· to ·resolve· (#1641). The first three stop at resolveInstanceQName's
// split and were charged by nothing; the last two clear it and were charged
// under clause 5 as QNames naming no type. The charge wraps the Datatype Valid
// verdict as its cause, and its opening is pinned as a prefix so that the
// element name and the lexical cannot trade places unseen (#1048).
func TestNonQNameXSITypeChargesClauseThree(t *testing.T) {
	schema := eSchema(t, false, nil)

	for _, lexical := range []string{
		"p:Derived",   // a prefix with no binding in scope
		"",            // empty
		"a:b:Derived", // a colon structure no QName has
		"23.789",      // an NCName cannot open with a digit
		"not a name",  // nor carry a space
	} {
		got, visits := cAssessLogged(t, schema, eRoot(map[string]string{"type": lexical}))

		if len(got) != 1 {
			t.Errorf("xsi:type=%q: Violations() = %v, want exactly one", lexical, got)
			continue
		}
		if got[0].Rule != "cvc-attribute" || got[0].Loc != loc(1, 10) {
			t.Errorf("xsi:type=%q: charge = %v, want cvc-attribute at the attribute's own %s", lexical, got[0], loc(1, 10))
		}
		opening := `the xsi:type attribute of the element root has the ·initial value· "` + lexical +
			`", which is not ·valid· with respect to {http://www.w3.org/2001/XMLSchema}QName`
		if !strings.HasPrefix(got[0].Msg, opening) {
			t.Errorf("xsi:type=%q: Msg = %q, want it to open %q", lexical, got[0].Msg, opening)
		}
		if !strings.Contains(got[0].Msg, "cvc-attribute clause 3") || strings.Contains(got[0].Msg, "clause 5") {
			t.Errorf("xsi:type=%q: Msg = %q, want it to name cvc-attribute clause 3 and not clause 5", lexical, got[0].Msg)
		}
		if rule, _ := xsderr.RuleOf(got[0].Unwrap()); rule != "cvc-datatype-valid" {
			t.Errorf("xsi:type=%q: cause = %v, want the cvc-datatype-valid verdict wrapped", lexical, got[0].Unwrap())
		}
		want := []string{"3/charged", "5/declined"}
		if outcomes := attributeOutcomes(*visits); !slices.Equal(outcomes, want) {
			t.Errorf("xsi:type=%q: logged %v, want %v", lexical, outcomes, want)
		}
	}
}

// A lexical in xs:QName's lexical space satisfies clause 3 and reaches clause 5
// alone: one naming no type is charged there and nowhere else, and one naming a
// type the schema carries is charged nowhere. The prefix is bound, so clause 3
// maps it under the element's bindings exactly as clause 5 resolves it.
func TestQNameXSITypeSatisfiesClauseThree(t *testing.T) {
	schema := eSchema(t, false, nil)
	root := eRoot(map[string]string{"type": "xs:unknownType"})
	root.bindings = map[string]string{"xs": xsd.XMLSchemaNS}

	got, visits := cAssessLogged(t, schema, root)

	if len(got) != 1 || !strings.Contains(got[0].Msg, "cvc-attribute clause 5") {
		t.Fatalf("Violations() = %v, want exactly one, under cvc-attribute clause 5", got)
	}
	if want := []string{"3/satisfied", "5/charged"}; !slices.Equal(attributeOutcomes(*visits), want) {
		t.Errorf("logged %v, want %v", attributeOutcomes(*visits), want)
	}

	got, visits = cAssessLogged(t, schema, eRoot(map[string]string{"type": "Derived"}, "a"))
	wantSilence(t, got, "an xsi:type that is a QName and ·resolves· satisfies clauses 3 and 5")
	if want := []string{"3/satisfied", "5/satisfied"}; !slices.Equal(attributeOutcomes(*visits), want) {
		t.Errorf("logged %v, want %v", attributeOutcomes(*visits), want)
	}
}

// attributeOutcomes is the clause/outcome pair of every cvc-attribute line the
// walk logged, in order.
func attributeOutcomes(visits []string) []string {
	outcomes := []string{}
	for _, line := range visits {
		if field(line, "validate.rule=") != "cvc-attribute" {
			continue
		}
		outcomes = append(outcomes, field(line, "validate.clause=")+"/"+field(line, "validate.outcome="))
	}
	return outcomes
}
