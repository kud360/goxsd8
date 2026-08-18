package validate

import (
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
		xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
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
		xsd.DerivationExtension, false, uses, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
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
		test := xsd.NewXPathExpression(a.test, nil, nil, nil)
		tas = append(tas, namedTypeAlternative(t, &test, local(a.typ)))
	}
	table, err := xsd.NewTypeTable(xsderr.Loc{}, tas,
		namedTypeAlternative(t, nil, local("Fallback")))
	if err != nil {
		t.Fatalf("building the type table: %v", err)
	}
	d, err := xsd.NewElementDeclaration(xsderr.Loc{}, local("root"),
		xsd.TypeDefinitionRef{Name: local("Fallback")}, &table, xsd.NewGlobalScope(),
		nil, false, nil, nil, nil, false, nil, nil)
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
	v, err := New(ctaSchema(t, alts...), testBackend())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	res := v.Assess(root)
	if res.Err() != nil {
		t.Fatalf("Err() = %v, want nil", res.Err())
	}
	return res.Violations()
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

// A {test} reads the element's OWN [[attributes]] (§3.12.4 clause 1.1.2), so
// an absent one makes the comparison false rather than an error — the
// alternative simply does not select.
func TestAbsentAttributeSelectsNothing(t *testing.T) {
	// <root other="v"/> carries no kind at all. Fallback declares neither
	// attribute, so the ONE charge names other; First would have charged
	// needFirst as well.
	ctaWantGoverned(t, ctaAssess(t, ctaRoot("", local("other")),
		ctaAlt{"@kind = 'book'", "First"}), "other")
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
	ta, err := xsd.NewTypeAlternative(xsderr.Loc{}, test, xsd.TypeDefinitionRef{Name: typeName}, nil)
	if err != nil {
		t.Fatalf("NewTypeAlternative(%v): %v", typeName, err)
	}
	return ta
}
