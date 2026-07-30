package xsd

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// facetBase builds a base type carrying ownFacets, rooted on a fresh primitive
// so each case is independent.
func facetBase(t *testing.T, ownFacets ...Facet) *SimpleType {
	t.Helper()
	prim := mustPrim(t, "string")
	base, err := NewSimpleType(xsderr.Loc{}, QName{Space: "urn:test", Local: "base"},
		NewAtomic(prim), prim, ownFacets, nil)
	if err != nil {
		t.Fatalf("build base: %v", err)
	}
	return base
}

// deriveFacets restricts base with ownFacets, returning the construction error.
func deriveFacets(base *SimpleType, ownFacets ...Facet) error {
	_, err := NewSimpleType(xsderr.Loc{}, QName{Space: "urn:test", Local: "derived"},
		base.variety, base, ownFacets, nil)
	return err
}

// wantRule asserts err is an *xsderr.Error charging rule.
func wantRule(t *testing.T, err error, rule xsderr.Rule) {
	t.Helper()
	if err == nil {
		t.Fatalf("want rejection charging %s, got nil", rule)
	}
	got, ok := xsderr.RuleOf(err)
	if !ok || got != rule {
		t.Fatalf("rejection charges %q (ok=%v), want %q; err=%v", got, ok, rule, err)
	}
}

// TestCountFacetValidRestriction covers all five count-valued valid-restriction
// SCCs in both polarities: the narrowing direction each rule permits must
// construct, and the widening direction must be rejected under that rule's own
// ID. length is the odd one out — §4.3.1.4 makes it an equality, so BOTH
// directions are violations.
func TestCountFacetValidRestriction(t *testing.T) {
	cases := []struct {
		name     string
		kind     FacetKind
		baseVal  string
		ownVal   string
		wantRule xsderr.Rule // empty means the restriction must be accepted
	}{
		{"length equal accepted", FacetLength, "5", "5", ""},
		{"length narrowed rejected", FacetLength, "5", "4", ruleLengthValidRestriction},
		{"length widened rejected", FacetLength, "5", "6", ruleLengthValidRestriction},
		{"minLength raised accepted", FacetMinLength, "2", "4", ""},
		{"minLength equal accepted", FacetMinLength, "2", "2", ""},
		{"minLength lowered rejected", FacetMinLength, "2", "1", ruleMinLengthValidRestriction},
		{"maxLength lowered accepted", FacetMaxLength, "9", "4", ""},
		{"maxLength raised rejected", FacetMaxLength, "9", "10", ruleMaxLengthValidRestriction},
		{"totalDigits lowered accepted", FacetTotalDigits, "6", "3", ""},
		{"totalDigits raised rejected", FacetTotalDigits, "6", "7", ruleTotalDigitsValidRestriction},
		{"fractionDigits lowered accepted", FacetFractionDigits, "4", "0", ""},
		{"fractionDigits raised rejected", FacetFractionDigits, "4", "5", ruleFractionDigitsValidRestriction},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			base := facetBase(t, NewFacet(c.kind, []string{c.baseVal}, false))
			err := deriveFacets(base, NewFacet(c.kind, []string{c.ownVal}, false))
			if c.wantRule == "" {
				if err != nil {
					t.Fatalf("restriction %s %s -> %s rejected: %v", c.kind, c.baseVal, c.ownVal, err)
				}
				return
			}
			wantRule(t, err, c.wantRule)
		})
	}
}

// TestCountFacetRestrictionVacuousWithoutBaseFacet proves the rules are keyed
// off the BASE's {facets}: a restriction introducing a count facet the base does
// not carry is unconstrained by them, in either direction.
func TestCountFacetRestrictionVacuousWithoutBaseFacet(t *testing.T) {
	base := facetBase(t)
	if err := deriveFacets(base, NewFacet(FacetMaxLength, []string{"100"}, false)); err != nil {
		t.Fatalf("introducing maxLength on a base without one: %v", err)
	}
	if err := deriveFacets(base, NewFacet(FacetLength, []string{"3"}, false)); err != nil {
		t.Fatalf("introducing length on a base without one: %v", err)
	}
}

// TestCountFacetRestrictionTransitive proves the comparison rides
// EffectiveFacets: a facet inherited unchanged through an intermediate level is
// still the operand two levels down, with no manual ancestor walk.
func TestCountFacetRestrictionTransitive(t *testing.T) {
	base := facetBase(t, NewFacet(FacetMaxLength, []string{"8"}, false))
	middle, err := NewSimpleType(xsderr.Loc{}, QName{Space: "urn:test", Local: "middle"},
		base.variety, base, nil, nil)
	if err != nil {
		t.Fatalf("build middle: %v", err)
	}
	_, err = NewSimpleType(xsderr.Loc{}, QName{Space: "urn:test", Local: "leaf"},
		middle.variety, middle, []Facet{NewFacet(FacetMaxLength, []string{"9"}, false)}, nil)
	wantRule(t, err, ruleMaxLengthValidRestriction)
}

// TestWhiteSpaceValidRestriction encodes §4.3.6.4's two numbered error
// conditions and only those: the restrictiveness order is preserve < replace <
// collapse, so moving up or staying put is legal and moving down is not. The
// replace-over-preserve row is the one the spec Note's own (contradictory)
// "preserve, collapse, replace" listing would wrongly reject.
func TestWhiteSpaceValidRestriction(t *testing.T) {
	cases := []struct {
		baseVal  string
		ownVal   string
		rejected bool
	}{
		{"preserve", "preserve", false},
		{"preserve", "replace", false},
		{"preserve", "collapse", false},
		{"replace", "replace", false},
		{"replace", "collapse", false},
		{"replace", "preserve", true}, // §4.3.6.4 condition 2
		{"collapse", "collapse", false},
		{"collapse", "replace", true},  // §4.3.6.4 condition 1
		{"collapse", "preserve", true}, // §4.3.6.4 condition 1
	}
	for _, c := range cases {
		t.Run(c.baseVal+"->"+c.ownVal, func(t *testing.T) {
			base := facetBase(t, NewFacet(FacetWhiteSpace, []string{c.baseVal}, false))
			err := deriveFacets(base, NewFacet(FacetWhiteSpace, []string{c.ownVal}, false))
			if !c.rejected {
				if err != nil {
					t.Fatalf("whiteSpace %s -> %s rejected: %v", c.baseVal, c.ownVal, err)
				}
				return
			}
			wantRule(t, err, ruleWhiteSpaceValidRestriction)
		})
	}
}

// TestTimezoneValidRestriction covers §4.3.14.4: a base {value} of optional may
// be narrowed to anything, while required or prohibited must be repeated
// verbatim.
func TestTimezoneValidRestriction(t *testing.T) {
	cases := []struct {
		baseVal  string
		ownVal   string
		rejected bool
	}{
		{"optional", "required", false},
		{"optional", "prohibited", false},
		{"optional", "optional", false},
		{"required", "required", false},
		{"required", "optional", true},
		{"required", "prohibited", true},
		{"prohibited", "required", true},
	}
	for _, c := range cases {
		t.Run(c.baseVal+"->"+c.ownVal, func(t *testing.T) {
			base := facetBase(t, NewFacet(FacetExplicitTimezone, []string{c.baseVal}, false))
			err := deriveFacets(base, NewFacet(FacetExplicitTimezone, []string{c.ownVal}, false))
			if !c.rejected {
				if err != nil {
					t.Fatalf("explicitTimezone %s -> %s rejected: %v", c.baseVal, c.ownVal, err)
				}
				return
			}
			wantRule(t, err, ruleTimezoneValidRestriction)
		})
	}
}

// TestCountFacetConsistency covers the same-type consistency SCCs, which are
// not restriction-specific: they constrain the type's OWN effective {facets},
// so they fire on a single type declaring both members of a pair.
func TestCountFacetConsistency(t *testing.T) {
	prim := mustPrim(t, "string")
	cases := []struct {
		name     string
		facets   []Facet
		wantRule xsderr.Rule
	}{
		{
			"minLength above maxLength",
			[]Facet{NewFacet(FacetMinLength, []string{"5"}, false), NewFacet(FacetMaxLength, []string{"3"}, false)},
			ruleMinLengthLEMaxLength,
		},
		{
			"minLength equal to maxLength accepted",
			[]Facet{NewFacet(FacetMinLength, []string{"3"}, false), NewFacet(FacetMaxLength, []string{"3"}, false)},
			"",
		},
		{
			"fractionDigits above totalDigits",
			[]Facet{NewFacet(FacetFractionDigits, []string{"4"}, false), NewFacet(FacetTotalDigits, []string{"2"}, false)},
			ruleFractionDigitsLETotalDigits,
		},
		{
			"minLength above length",
			[]Facet{NewFacet(FacetLength, []string{"4"}, false), NewFacet(FacetMinLength, []string{"5"}, false)},
			ruleLengthMinLengthMaxLength,
		},
		{
			"length above maxLength",
			[]Facet{NewFacet(FacetLength, []string{"6"}, false), NewFacet(FacetMaxLength, []string{"5"}, false)},
			ruleLengthMinLengthMaxLength,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := NewSimpleType(xsderr.Loc{}, QName{Space: "urn:test", Local: "t"},
				NewAtomic(prim), prim, c.facets, nil)
			if c.wantRule == "" {
				if err != nil {
					t.Fatalf("consistent facets rejected: %v", err)
				}
				return
			}
			wantRule(t, err, c.wantRule)
		})
	}
}

// TestCountFacetValueMalformed proves a count facet's {value} is parsed as an
// xs:nonNegativeInteger and charged under that facet's own rule when it is not
// one — reached only where a comparison actually needs the number.
func TestCountFacetValueMalformed(t *testing.T) {
	base := facetBase(t, NewFacet(FacetMaxLength, []string{"8"}, false))
	err := deriveFacets(base, NewFacet(FacetMaxLength, []string{"-1"}, false))
	wantRule(t, err, ruleMaxLengthValidRestriction)
	if !strings.Contains(err.Error(), "nonNegativeInteger") {
		t.Errorf("message %q does not name the expected {value} space", err.Error())
	}
}

// TestListApplicableFacets covers cos-st-restricts clause 2.2.2.4: the list
// applicable set is the §4.1.5 literal, so length/minLength/maxLength/pattern/
// enumeration/whiteSpace/assertions pass and anything else — here a bound
// facet — is rejected.
func TestListApplicableFacets(t *testing.T) {
	item := mustPrim(t, "string")
	_, err := NewSimpleType(xsderr.Loc{}, QName{Space: "urn:test", Local: "goodlist"},
		NewList(item), anySimpleType,
		[]Facet{NewFacet(FacetMinLength, []string{"1"}, false)}, nil)
	if err != nil {
		t.Fatalf("list with an applicable facet rejected: %v", err)
	}

	_, err = NewSimpleType(xsderr.Loc{}, QName{Space: "urn:test", Local: "badlist"},
		NewList(item), anySimpleType,
		[]Facet{NewFacet(FacetMaxInclusive, []string{"5"}, false)}, nil)
	wantRule(t, err, ruleCosSTRestricts)
	if !strings.Contains(err.Error(), "2.2.2.4") {
		t.Errorf("message %q does not name clause 2.2.2.4", err.Error())
	}
}

// TestUnionApplicableFacets covers cos-st-restricts clause 3.2.2.4: only
// pattern, enumeration and assertions are applicable to a union, so a facet
// applicable to a LIST but not a union (whiteSpace) is still rejected.
func TestUnionApplicableFacets(t *testing.T) {
	member := mustPrim(t, "string")
	_, err := NewSimpleType(xsderr.Loc{}, QName{Space: "urn:test", Local: "goodunion"},
		NewUnion(member), anySimpleType,
		[]Facet{NewFacet(FacetPattern, []string{"a+"}, false)}, nil)
	if err != nil {
		t.Fatalf("union with an applicable facet rejected: %v", err)
	}

	_, err = NewSimpleType(xsderr.Loc{}, QName{Space: "urn:test", Local: "badunion"},
		NewUnion(member), anySimpleType,
		[]Facet{NewFacet(FacetWhiteSpace, []string{"collapse"}, true)}, nil)
	wantRule(t, err, ruleCosSTRestricts)
	if !strings.Contains(err.Error(), "3.2.2.4") {
		t.Errorf("message %q does not name clause 3.2.2.4", err.Error())
	}
}

// TestAtomicVarietyApplicableFacetsNotCharged pins the placement decision: the
// ATOMIC applicable-facet clause (1.3.1) is NOT charged in this package — it
// needs the generated per-primitive table — so an inapplicable-looking facet on
// an atomic type still constructs here and is rejected one layer up, by
// builtin.CheckSimpleTypeRestriction.
func TestAtomicVarietyApplicableFacetsNotCharged(t *testing.T) {
	prim := mustPrim(t, "string")
	if _, err := NewSimpleType(xsderr.Loc{}, QName{Space: "urn:test", Local: "atomic"},
		NewAtomic(prim), prim,
		[]Facet{NewFacet(FacetMaxInclusive, []string{"5"}, false)}, nil); err != nil {
		t.Fatalf("atomic applicability must not be charged in package xsd, got: %v", err)
	}
}
