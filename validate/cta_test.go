package validate

import (
	"slices"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The fixtures below drive key-cta-select (§3.3.4.1) end to end against
// <root kind="…"/>, and every candidate type charges that element a DIFFERENT
// violation, which is how a test here tells one selection from another at all:
//
//   - Fallback declares no attribute use and no wildcard, so the kind
//     attribute is cvc-complex-type clause 2's charge;
//   - First and Second extend it with an optional kind use and a REQUIRED one
//     named after themselves, so the missing needFirst/needSecond is clause
//     3's charge.
//
// Silence means the ·governing type definition· was withheld.

// ctaAlt pairs one <alternative>'s {test} with the name of the type it names.
type ctaAlt struct{ test, typ string }

// The message fragment each selection leaves in the one violation it charges.
const (
	ctaGovernedByFallback = "kind"
	ctaGovernedByFirst    = "needFirst"
	ctaGovernedBySecond   = "needSecond"
)

// ctaFallbackType is the {default type definition}'s type and the
// declaration's own {type definition}: empty content, no attribute use, no
// wildcard.
func ctaFallbackType(t *testing.T) xsd.ComplexType {
	t.Helper()
	ct, err := xsd.NewComplexType(xsderr.Loc{}, local("Fallback"), xsd.QName{}, nil,
		xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil)
	if err != nil {
		t.Fatalf("building the Fallback complex type: %v", err)
	}
	return ct
}

// ctaCandidateType EXTENDS Fallback with an optional kind use and a required
// one named after itself. The extension is what e-props-correct clause 7 needs:
// every {alternatives} member's type must be ·validly substitutable· for the
// declaration's own {type definition}.
func ctaCandidateType(t *testing.T, name string) xsd.ComplexType {
	t.Helper()
	uses := []xsd.AttributeUse{aUse(t, "kind", false, nil), aUse(t, "need"+name, true, nil)}
	ct, err := xsd.NewComplexType(xsderr.Loc{}, local(name), local("Fallback"), nil,
		xsd.DerivationExtension, false, attrContent(uses), nil, nil, xsd.EmptyContent{}, nil, nil)
	if err != nil {
		t.Fatalf("building the %s complex type: %v", name, err)
	}
	return ct
}

// ctaSchema declares "root" with a {type table} over alts and a {default type
// definition} naming Fallback, which is also the declaration's own {type
// definition} — so a test that reads the DECLARED type instead of running
// key-cta-select is indistinguishable from one that fell through to the
// default, and both are distinguishable from a real selection.
//
// The builtin datatypes are seeded because a {test} is evaluated against them:
// xpath.CompileCTATest resolves the types §3.5.2's casting rules name — and
// the type any explicit cast targets — through the schema's {type
// definitions}, which is where parser.Parse's own builtin.Seed leaves them for
// every parsed schema.
func ctaSchema(t *testing.T, alts ...ctaAlt) *xsd.Schema {
	t.Helper()
	b := xsd.NewSchemaBuilder()
	for _, st := range ctaBuiltins(t) {
		b.AddType(st)
	}
	b.AddType(ctaFallbackType(t))
	b.AddType(ctaCandidateType(t, "First"))
	b.AddType(ctaCandidateType(t, "Second"))
	var tas []xsd.TypeAlternative
	for _, a := range alts {
		// Every {test} carries the xs binding a real <alternative> inherits
		// from the schema document it is written in, so a fixture can write
		// the explicit cast §3.12.4 clause 1's Note calls for over untyped
		// nodes; a {test} not naming a prefix is unaffected by its presence.
		test := xsd.NewXPathExpression(a.test,
			[]xsd.NamespaceBinding{xsd.NewNamespaceBinding("xs", xsd.XMLSchemaNS)}, nil, nil)
		tas = append(tas, namedTypeAlternative(t, &test, local(a.typ)))
	}
	table, err := xsd.NewTypeTable(xsderr.Loc{}, tas,
		namedTypeAlternative(t, nil, local("Fallback")))
	if err != nil {
		t.Fatalf("building the type table: %v", err)
	}
	d, err := xsd.NewElementDeclaration(xsderr.Loc{}, local("root"),
		xsd.TypeDefinitionRef{Name: local("Fallback")}, &table, xsd.NewGlobalScope(),
		nil, false, nil, nil, nil, false, nil)
	if err != nil {
		t.Fatalf("building the root element declaration: %v", err)
	}
	b.AddElement(d)
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the tabled schema: %v", err)
	}
	return schema
}

// ctaBuiltins is the seeded builtin datatype cohort every fixture schema
// carries.
func ctaBuiltins(t *testing.T) []*xsd.SimpleType {
	t.Helper()
	types, err := builtin.Seed(testBackend())
	if err != nil {
		t.Fatalf("seeding the builtin datatypes: %v", err)
	}
	return types
}

// ctaRoot is <root kind="…"/>, or <root/> when kind is empty.
func ctaRoot(kind string, extra ...xsd.QName) *testElement {
	e := &testElement{name: local("root"), loc: loc(1, 1)}
	if kind != "" {
		e.attrs = append(e.attrs, &testAttribute{name: local("kind"), value: kind, loc: loc(1, 7)})
	}
	for i, n := range extra {
		e.attrs = append(e.attrs, &testAttribute{name: n, value: "v", loc: loc(1, 20+i)})
	}
	return e
}

// ctaAssess assesses root against a table built from alts.
func ctaAssess(t *testing.T, root Element, alts ...ctaAlt) []*xsderr.Error {
	t.Helper()
	return ctaResult(t, root, alts...).Violations()
}

// ctaResult is ctaAssess's whole [Result], for the tests that read the
// withhold channel as well as the charges.
func ctaResult(t *testing.T, root Element, alts ...ctaAlt) *Result {
	t.Helper()
	v, err := New(ctaSchema(t, alts...), testBackend())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	res := v.Assess(root)
	if res.Err() != nil {
		t.Fatalf("Err() = %v, want nil", res.Err())
	}
	return res
}

// ctaWantGoverned fails unless exactly one violation was charged and it
// carries want — the fragment identifying which type governed.
func ctaWantGoverned(t *testing.T, got []*xsderr.Error, want string) {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("Violations() = %v, want exactly one, naming %s", got, want)
	}
	if !strings.Contains(got[0].Msg, want) {
		t.Errorf("Msg = %q, want it to name %s — another type governed", got[0].Msg, want)
	}
}

// A {test} that evaluates to true makes its <alternative>'s type the
// ·selected type definition·, and the element is assessed against THAT type
// and not against the declaration's own {type definition} (§3.3.4.1
// key-selected-type clause 1 over key-cta-select clause 1).
func TestConditionallySelectedTypeGoverns(t *testing.T) {
	ctaWantGoverned(t, ctaAssess(t, ctaRoot("book"),
		ctaAlt{"@kind = 'book'", "First"}), ctaGovernedByFirst)
}

// No {test} true falls to T.{default type definition}.{type definition}
// (key-cta-select clause 2), which is a real selection and not a decline.
func TestNoAlternativeSelectsFallsToTheDefaultType(t *testing.T) {
	ctaWantGoverned(t, ctaAssess(t, ctaRoot("cd"),
		ctaAlt{"@kind = 'book'", "First"}), ctaGovernedByFallback)
}

// The alternatives are tried IN ORDER and the first success stops the scan:
// "if any Type Alternative ·successfully selects· a type definition, none of
// the following Type Alternatives are tried". Both tests below are true, so
// only the order decides.
func TestFirstSuccessfulAlternativeWins(t *testing.T) {
	ctaWantGoverned(t, ctaAssess(t, ctaRoot("book"),
		ctaAlt{"@kind = 'book'", "First"},
		ctaAlt{"@kind", "Second"}), ctaGovernedByFirst)
	ctaWantGoverned(t, ctaAssess(t, ctaRoot("book"),
		ctaAlt{"@kind", "Second"},
		ctaAlt{"@kind = 'book'", "First"}), ctaGovernedBySecond)
}

// A {test} the §3.12.6 evaluator cannot evaluate withholds the whole element's
// ·governing type definition· — nothing is charged at or below it. It does NOT
// fall through to the next alternative or to the default: the undecided test
// might have been true, and assessing the element against a type the rule may
// not have selected manufactures a false reject.
func TestUnevaluableTestWithholdsTheGoverningType(t *testing.T) {
	wantSilence(t, ctaAssess(t, ctaRoot("book"),
		ctaAlt{"count(@kind) > 0", "First"}),
		"a {test} outside the required subset leaves the selected type undetermined")
	wantSilence(t, ctaAssess(t, ctaRoot("book"),
		ctaAlt{"count(@kind) > 0", "First"},
		ctaAlt{"@kind = 'book'", "Second"}),
		"a decline BEFORE a matching alternative stops the scan")
}

// The silence above is the whole point of the record: an element whose
// ·governing type definition· CTA withheld charges nothing, so without an
// [Unevaluated] it is byte-identical at the [Result] API to one that passed
// every rule. The record names the alternative, its {test} and the
// key-cta-select consequence, and sits at the ELEMENT's location.
func TestUnevaluableTestIsRecordedAsUnevaluated(t *testing.T) {
	res := ctaResult(t, ctaRoot("book"),
		ctaAlt{"@kind = 'book'", "First"},
		ctaAlt{"count(@kind) > 0", "Second"},
		ctaAlt{"@kind = 'cd'", "First"})
	// The first alternative selects, so the second is never tried: a record
	// here would report a check the scan did not reach. First's kind use is
	// typeless (aUse), so the element's own attribute half records its
	// cvc-attribute decline, and that one record is all there is.
	if got := res.Unevaluated(); len(got) != 1 || got[0].Rule() != ruleCvcAttribute {
		t.Fatalf("Unevaluated() = %v, want only the kind attribute's cvc-attribute record — the scan stopped before the unevaluable alternative", messages(got))
	}
	res = ctaResult(t, ctaRoot("book"),
		ctaAlt{"@kind = 'cd'", "First"},
		ctaAlt{"count(@kind) > 0", "Second"},
		ctaAlt{"@kind = 'book'", "First"})
	wantSilence(t, res.Violations(), "the withheld type charges nothing")
	// The withheld type is one record, and the one other is what it costs the
	// ID/IDREF table: an element of undetermined type is an item cvc-id clause 1
	// cannot read, recorded as the element leaves.
	got := res.Unevaluated()
	if len(got) != 2 || got[1].Rule() != ruleCvcID {
		t.Fatalf("Unevaluated() = %v, want exactly one record for the withheld ·conditionally selected· type, then cvc-id's", messages(got))
	}
	if got[0].Rule() != "key-cta-ta-select" {
		t.Errorf("Rule() = %q, want key-cta-ta-select", got[0].Rule())
	}
	if got[0].Loc() != loc(1, 1) {
		t.Errorf("Loc() = %s, want the element's %s", got[0].Loc(), loc(1, 1))
	}
	for _, want := range []string{"alternative 2 of 3", "count(@kind) > 0", "key-cta-select"} {
		if !strings.Contains(got[0].Msg(), want) {
			t.Errorf("Msg() = %q, want it to name %s", got[0].Msg(), want)
		}
	}
}

// The withhold TestUnevaluableTestIsRecordedAsUnevaluated reads off the
// [Result] is logged as well as recorded: one "declined" line at root under
// key-cta-ta-select clause 2, the {test} evaluation the compiler declined.
func TestUnevaluableTestLogsTheDecline(t *testing.T) {
	log, visits := recordingLogger()
	v, err := New(ctaSchema(t,
		ctaAlt{"@kind = 'cd'", "First"},
		ctaAlt{"count(@kind) > 0", "Second"},
		ctaAlt{"@kind = 'book'", "First"}), testBackend(), WithLogger(log))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	res := v.Assess(ctaRoot("book"))
	if got := res.Unevaluated(); len(got) == 0 || got[0].Rule() != ruleKeyCTATASelect {
		t.Fatalf("Unevaluated() = %v, want the withheld type's key-cta-ta-select record first", messages(got))
	}
	const want = "assessing element validate.name=root validate.loc=instance.xml:1:1 validate.rule=key-cta-ta-select validate.clause=2 validate.outcome=declined"
	var lines []string
	for _, line := range *visits {
		if strings.Contains(line, "validate.rule=key-cta-ta-select") {
			lines = append(lines, line)
		}
	}
	if len(lines) != 1 || lines[0] != want {
		t.Errorf("walk logged key-cta-ta-select\n\t%s\nwant exactly one line\n\t%s", strings.Join(lines, "\n\t"), want)
	}
}

// A dynamic or type error inside a {test} this engine CAN evaluate is a
// decided false, not a withhold: key-cta-ta-select clause 2 makes it "treated
// as if it had evaluated (without error) to false", so the alternative was
// tried and did not select, the scan reaches the {default type definition},
// and NOTHING is recorded. `@kind cast as xs:integer` over kind="book" raises
// err:FORG0001 and over an absent kind raises err:XPTY0004 — both are that
// clause's false and neither is this package's decline.
func TestDynamicErrorInAnEvaluableTestRecordsNothing(t *testing.T) {
	for _, tc := range []struct {
		root *testElement
		// governed is the attribute Fallback does not declare, which is the
		// one charge a REAL selection of the {default type definition}
		// leaves — silence would mean the type was withheld instead.
		governed string
		why      string
	}{
		{ctaRoot("book"), ctaGovernedByFallback, "a cast whose operand is not a valid lexical (err:FORG0001)"},
		{ctaRoot("", local("other")), "other", "a cast over the empty sequence (err:XPTY0004)"},
	} {
		res := ctaResult(t, tc.root, ctaAlt{"@kind cast as xs:integer = 3", "First"})
		if got := res.Unevaluated(); got != nil {
			t.Errorf("Unevaluated() = %v, want none: %s is a decided false, not a withhold", messages(got), tc.why)
		}
		ctaWantGoverned(t, res.Violations(), tc.governed)
	}
}

// An unevaluable {test} BEHIND an alternative that already succeeded costs the
// element nothing: key-cta-select never tries it, so the lazy scan must not
// either.
func TestUnevaluableTestBehindASuccessIsNeverTried(t *testing.T) {
	ctaWantGoverned(t, ctaAssess(t, ctaRoot("book"),
		ctaAlt{"@kind = 'book'", "First"},
		ctaAlt{"count(@kind) > 0", "Second"}), ctaGovernedByFirst)
}

// An empty {alternatives} sequence is the whole table conditionally selecting
// its {default type definition}: there is nothing to try, so nothing declines.
func TestEmptyAlternativesSelectsTheDefaultType(t *testing.T) {
	ctaWantGoverned(t, ctaAssess(t, ctaRoot("book")), ctaGovernedByFallback)
}

// A {test} at the ·validation root·, which inherits nothing, reads the
// element's OWN [[attributes]] (§3.12.4 clause 1.1.2) alone, so an absent one
// makes the comparison false rather than an error — the alternative simply does
// not select.
func TestAbsentAttributeSelectsNothing(t *testing.T) {
	// <root other="v"/> carries no kind at all. Fallback declares neither
	// attribute, so the ONE charge names other; First would have charged
	// needFirst as well.
	ctaWantGoverned(t, ctaAssess(t, ctaRoot("", local("other")),
		ctaAlt{"@kind = 'book'", "First"}), "other")
}

// A WILDCARD NameTest reads the same [[attributes]] through the same walk
// ctaAttributes hands the engine (#859), so a `@*` selects over the whole
// sequence rather than over one ·expanded name·. The last case is the
// discriminating one: `@*` matches the attribute the element DOES carry, whose
// value satisfies nothing, so the alternative does not select — a wildcard is
// not "true whenever E has an attribute".
func TestWildcardTestSelectsOverTheElementsAttributes(t *testing.T) {
	ctaWantGoverned(t, ctaAssess(t, ctaRoot("book"),
		ctaAlt{"@* = 'book'", "First"}), ctaGovernedByFirst)
	ctaWantGoverned(t, ctaAssess(t, ctaRoot("book"),
		ctaAlt{"@*:kind = 'book'", "First"}), ctaGovernedByFirst)
	ctaWantGoverned(t, ctaAssess(t, ctaRoot("", local("other")),
		ctaAlt{"@* = 'book'", "First"}), "other")
}

// cvc-elt clause 4's ·override· is read against the CONDITIONALLY SELECTED
// type, not against the declaration's own {type definition}: an xsi:type
// naming Fallback — the declared type — does not ·override· the selected
// First, so clause 4 is charged. A processor that had kept using the declared
// type as "selected" would find the two identical and charge nothing.
func TestXSITypeOverridesTheConditionallySelectedType(t *testing.T) {
	root := ctaRoot("book")
	root.attrs = append(root.attrs, &testAttribute{
		name:  xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "type"},
		value: "Fallback",
		loc:   loc(1, 20),
	})
	got := ctaAssess(t, root, ctaAlt{"@kind = 'book'", "First"})
	if len(got) == 0 {
		t.Fatal("Violations() = none, want a cvc-elt clause 4 charge")
	}
	if got[0].Rule != "cvc-elt" {
		t.Errorf("Rule = %q, want cvc-elt", got[0].Rule)
	}
}

// namedTypeAlternative builds a Type Alternative whose {type definition} is the
// by-name reference §3.12.2 declare-ta's type= arm yields, failing the test on
// any rejection. Every fixture in package validate builds that arm; the inline
// arm is the parser's to construct and is exercised there.
func namedTypeAlternative(t *testing.T, test *xsd.XPathExpression, typeName xsd.QName) xsd.TypeAlternative {
	t.Helper()
	ta, err := xsd.NewTypeAlternative(xsderr.Loc{}, test, xsd.TypeDefinitionRef{Name: typeName})
	if err != nil {
		t.Fatalf("NewTypeAlternative(%v): %v", typeName, err)
	}
	return ta
}

// absentHead is a {type definition} naming a substitution group head that no
// declaration in the schema carries: a head reference Finalize accepts as an
// ·absent· component (§5.3 Missing Sub-components) and
// [xsd.Schema.ResolvedType] resolves to nothing. The parser never emits it — an
// unresolvable substitutionGroup head gives its member xs:anyType — so only a
// directly built declaration carries it.
var absentHead = xsd.SubstitutionGroupHeadTypeRef{Head: local("nosuchhead")}

// absentHeadSchema is selectionSchema with "root" a member of absentHead's
// substitution group and no declaration of that head.
func absentHeadSchema(t *testing.T, declare func() (xsd.TypeDefinitionOrRef, *xsd.TypeTable)) *xsd.Schema {
	t.Helper()
	return selectionSchema(t, absentHead.Head, nil, declare)
}

// selectionSchema is ctaSchema's cohort with "root" declared by declare, in
// head's substitution group, beside the declarations extra and a top-level
// "child" typed Fallback — so each test below can place one shape in the
// {type definition} slot key-selected-type reads for root, and tell from
// selectionRoot's child whether root's [[children]] were assessed at all.
func selectionSchema(t *testing.T, head xsd.QName, extra []xsd.ElementDeclaration, declare func() (xsd.TypeDefinitionOrRef, *xsd.TypeTable)) *xsd.Schema {
	t.Helper()
	b := xsd.NewSchemaBuilder()
	for _, st := range ctaBuiltins(t) {
		b.AddType(st)
	}
	b.AddType(ctaFallbackType(t))
	b.AddType(ctaCandidateType(t, "First"))
	typ, table := declare()
	d, err := xsd.NewElementDeclaration(xsderr.Loc{}, local("root"), typ, table, xsd.NewGlobalScope(),
		nil, false, nil, []xsd.QName{head}, nil, false, nil)
	if err != nil {
		t.Fatalf("building the root element declaration: %v", err)
	}
	b.AddElement(d)
	child, err := xsd.NewElementDeclaration(xsderr.Loc{}, local("child"), xsd.TypeDefinitionRef{Name: local("Fallback")},
		nil, xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil)
	if err != nil {
		t.Fatalf("building the child element declaration: %v", err)
	}
	b.AddElement(child)
	for _, x := range extra {
		b.AddElement(x)
	}
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing a schema whose root has an unresolvable {type definition}: %v", err)
	}
	return schema
}

// selectionRoot is <root kind="…"><child kind="x"/></root>. The child's kind
// attribute is cvc-complex-type clause 2's charge against Fallback, which
// selectionSchema's top-level "child" declares, so it is charged exactly where
// the child is assessed against that declaration — and root's ·lax
// assessment· against xs:anyType does that, through the lax wildcard its
// [[children]] are ·attributed to· (key-lva, cvc-assess-elt clause 3.1).
func selectionRoot(kind string) *testElement {
	child := &testElement{name: local("child"), loc: loc(2, 3),
		attrs: []Attribute{&testAttribute{name: local("kind"), value: "x", loc: loc(2, 10)}}}
	root := ctaRoot(kind)
	root.kids = []Child{ElementChild(child)}
	return root
}

// selectionResult assesses selectionRoot(kind) against schema.
func selectionResult(t *testing.T, schema *xsd.Schema, kind string) *Result {
	t.Helper()
	v, err := New(schema, testBackend())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	res := v.Assess(selectionRoot(kind))
	if res.Err() != nil {
		t.Fatalf("Err() = %v, want nil", res.Err())
	}
	return res
}

// absentHeadTable is a {type table} whose one alternative tests @kind = 'book'
// and names alt, and whose {default type definition} names dflt.
func absentHeadTable(t *testing.T, alt, dflt xsd.TypeDefinitionOrRef) *xsd.TypeTable {
	t.Helper()
	test := xsd.NewXPathExpression("@kind = 'book'", nil, nil, nil)
	a, err := xsd.NewTypeAlternative(xsderr.Loc{}, &test, alt)
	if err != nil {
		t.Fatalf("NewTypeAlternative: %v", err)
	}
	d, err := xsd.NewTypeAlternative(xsderr.Loc{}, nil, dflt)
	if err != nil {
		t.Fatalf("NewTypeAlternative (default): %v", err)
	}
	table, err := xsd.NewTypeTable(xsderr.Loc{}, []xsd.TypeAlternative{a}, d)
	if err != nil {
		t.Fatalf("NewTypeTable: %v", err)
	}
	return &table
}

// wantAbsentSelection fails unless assessing selectionRoot(kind) against schema
// charges exactly two violations and records no [Unevaluated]. The first is
// cvc-elt at root's own location, its message opening on root's ·absent·
// ·selected type definition· and slot, the slot key-selected-type read, and
// naming clause 1 and §5.3: the charge §5.3 makes "as if clause 1 of Element
// Locally Valid (Element) had failed". The second is cvc-complex-type at the
// child's kind attribute, which only the ·lax assessment· §5.3 falls back to
// reaches: root assessed against nothing attributes its child to nothing, and
// the child is then assessed against nothing too. No cvc-id record follows,
// because a ·laxly assessed· element is no withheld type ([walk.idElement]).
func wantAbsentSelection(t *testing.T, schema *xsd.Schema, kind, slot string) {
	t.Helper()
	res := selectionResult(t, schema, kind)
	if got := res.Unevaluated(); len(got) != 0 {
		t.Errorf("Unevaluated() = %v, want none: an ·absent· selected type is charged and laxly assessed, not withheld", messages(got))
	}
	got := res.Violations()
	if len(got) != 2 {
		t.Fatalf("Violations() = %v, want the cvc-elt clause 1 charge at root, then the child's cvc-complex-type charge", got)
	}
	if got[0].Rule != ruleCvcElt || got[0].Loc != loc(1, 1) {
		t.Errorf("Violations()[0] = %v, want cvc-elt at root's %s", got[0], loc(1, 1))
	}
	opening := "the ·selected type definition· of the element root is ·absent·: " + slot + " names a substitution group head"
	if !strings.HasPrefix(got[0].Msg, opening) {
		t.Errorf("Msg = %q, want it to open %q", got[0].Msg, opening)
	}
	for _, want := range []string{"cvc-elt clause 1", "§5.3", "·laxly assessed·"} {
		if !strings.Contains(got[0].Msg, want) {
			t.Errorf("Msg = %q, want it to name %s", got[0].Msg, want)
		}
	}
	if got[1].Rule != ruleCvcComplexType || got[1].Loc != loc(2, 10) {
		t.Errorf("Violations()[1] = %v, want cvc-complex-type at the child's kind attribute %s: root was not laxly assessed", got[1], loc(2, 10))
	}
}

// A declaration with no {type table} whose own {type definition} names an
// ·absent· head leaves key-selected-type clause 2 with an ·absent· type: charged
// under §5.3, and the element laxly assessed.
func TestAbsentDeclaredTypeIsChargedAndLaxlyAssessed(t *testing.T) {
	wantAbsentSelection(t, absentHeadSchema(t, func() (xsd.TypeDefinitionOrRef, *xsd.TypeTable) {
		return absentHead, nil
	}), "book", "the {type definition} of its ·governing element declaration·")
}

// An alternative that ·successfully selects· a {type definition} naming an
// ·absent· head leaves key-cta-select clause 1 with an ·absent· type: charged
// under cvc-elt, not key-cta-ta-select, because the selection itself succeeded.
func TestAbsentSelectedAlternativeTypeIsChargedAndLaxlyAssessed(t *testing.T) {
	wantAbsentSelection(t, absentHeadSchema(t, func() (xsd.TypeDefinitionOrRef, *xsd.TypeTable) {
		return xsd.TypeDefinitionRef{Name: local("Fallback")},
			absentHeadTable(t, absentHead, xsd.TypeDefinitionRef{Name: local("Fallback")})
	}), "book", "the {type definition} of alternative 1 of 1 in the {alternatives} of the {type table} of its ·governing element declaration·, which ·successfully selects· it,")
}

// A {default type definition} whose {type definition} names an ·absent· head
// leaves key-cta-select clause 2 with an ·absent· type: charged. The
// alternative's {test} is false over kind="cd" and its own type, First,
// resolves, so only the default can be what was charged.
func TestAbsentDefaultTypeIsChargedAndLaxlyAssessed(t *testing.T) {
	wantAbsentSelection(t, absentHeadSchema(t, func() (xsd.TypeDefinitionOrRef, *xsd.TypeTable) {
		return xsd.TypeDefinitionRef{Name: local("Fallback")},
			absentHeadTable(t, xsd.TypeDefinitionRef{Name: local("First")}, absentHead)
	}), "cd", "the {type definition} of the {default type definition} of the {type table} of its ·governing element declaration·, which the table ·conditionally selects· because no alternative ·successfully selects· a type definition,")
}

// The §5.3 charge logs its decision at root on the "assessing element" event
// (STYLE L1), as the nil-slot decline logs its own (rootTypeDeclined).
func TestAbsentSelectionLogsTheCharge(t *testing.T) {
	log, visits := recordingLogger()
	v, err := New(absentHeadSchema(t, func() (xsd.TypeDefinitionOrRef, *xsd.TypeTable) {
		return absentHead, nil
	}), testBackend(), WithLogger(log))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	v.Assess(selectionRoot("book"))
	const want = "assessing element validate.name=root validate.loc=instance.xml:1:1 validate.rule=cvc-elt validate.clause=1 validate.outcome=charged"
	if !slices.Contains(*visits, want) {
		t.Errorf("walk logged\n\t%s\nwant a line\n\t%s", strings.Join(*visits, "\n\t"), want)
	}
}

// wantNilSelectionDeclined fails unless assessing selectionRoot("book")
// against schema charges nothing and records exactly two [Unevaluated]: the
// withheld ·selected type definition· under cvc-elt at root, its message
// opening on root's undetermined type, and then the cvc-id record the untyped
// element costs the ID/IDREF table. The silence — no cvc-elt clause 1 charge,
// and the child's kind attribute never assessed — is the nil-slot decline
// [walk.resolvedSelection]'s GAP(validate) marker names, RULED permanent by
// #2166 (STYLE P3b).
func wantNilSelectionDeclined(t *testing.T, schema *xsd.Schema) {
	t.Helper()
	res := selectionResult(t, schema, "book")
	wantSilence(t, res.Violations(), "a nil {type definition} is declined, not charged, and its element's children are not assessed")
	got := res.Unevaluated()
	if len(got) != 2 || got[0].Rule() != ruleCvcElt || got[1].Rule() != ruleCvcID {
		t.Fatalf("Unevaluated() = %v, want a cvc-elt record for the withheld type, then cvc-id's", messages(got))
	}
	if got[0].Loc() != loc(1, 1) {
		t.Errorf("Loc() = %s, want root's %s", got[0].Loc(), loc(1, 1))
	}
	opening := "the ·selected type definition· of the element root was not determined"
	if !strings.HasPrefix(got[0].Msg(), opening) {
		t.Errorf("Msg() = %q, want it to open %q", got[0].Msg(), opening)
	}
	for _, want := range []string{"the {type definition} of its ·governing element declaration· resolves to no type definition", "·absent· (nil)", "cvc-elt clause 1"} {
		if !strings.Contains(got[0].Msg(), want) {
			t.Errorf("Msg() = %q, want it to name %s", got[0].Msg(), want)
		}
	}
}

// A nil {type definition} on a directly built declaration is the
// sch-props-correct clause 1 fault xsd defers, not §5.3's missing
// sub-component, so it keeps its decline (#774's terms).
func TestNilDeclaredTypeIsDeclined(t *testing.T) {
	wantNilSelectionDeclined(t, selectionSchema(t, absentHead.Head, nil, func() (xsd.TypeDefinitionOrRef, *xsd.TypeTable) {
		return nil, nil
	}))
}

// A substitution group head that IS declared, with a nil {type definition} of
// its own, is that same nil slot reached through the head reference — not an
// ·absent· head — so it is declined too, not charged.
func TestNilHeadTypeIsDeclined(t *testing.T) {
	head, err := xsd.NewElementDeclaration(xsderr.Loc{}, local("head"), nil, nil, xsd.NewGlobalScope(),
		nil, false, nil, nil, nil, false, nil)
	if err != nil {
		t.Fatalf("building the head element declaration: %v", err)
	}
	wantNilSelectionDeclined(t, selectionSchema(t, local("head"), []xsd.ElementDeclaration{head}, func() (xsd.TypeDefinitionOrRef, *xsd.TypeTable) {
		return xsd.SubstitutionGroupHeadTypeRef{Head: local("head")}, nil
	}))
}
