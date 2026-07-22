package xsd

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// mustPrim builds a primitive datatype (base xs:anyAtomicType) or fails the
// test; it is the atomic building block for the derivation-graph tests.
func mustPrim(t *testing.T, local string) *SimpleType {
	t.Helper()
	st, err := NewPrimitiveType(xsderr.Loc{}, QName{Space: XMLSchemaNS, Local: local}, nil, nil)
	if err != nil {
		t.Fatalf("build primitive %s: %v", local, err)
	}
	return st
}

// mustST builds a simple type through NewSimpleType or fails the test — used for
// the prerequisite base/item/member types the negative cases restrict.
func mustST(t *testing.T, name string, v Variety, base *SimpleType, facets []Facet, final []DerivationMethod) *SimpleType {
	t.Helper()
	st, err := NewSimpleType(xsderr.Loc{}, QName{Local: name}, v, base, facets, final)
	if err != nil {
		t.Fatalf("build %s: %v", name, err)
	}
	return st
}

// TestSTGraphChecks exercises checkSTGraph and the per-variety cos-st-restricts
// case checks in derivation.go: both polarities for atomic (clause 1.1), list
// (clause 2, both the constructed B==anySimpleType and restricted branches),
// union (clause 3, both branches), the {final}-blocking clause
// 3/1.2/2.2.2.2/3.2.2.2 (shared site) and 2.2.1.1/3.2.1.1, and clause 5. Every
// prerequisite type is built through the real constructors — no XML fixtures.
func TestSTGraphChecks(t *testing.T) {
	dec := mustPrim(t, "decimal")
	str := mustPrim(t, "string")

	// int restricts decimal; int2 restricts int (a two-step atomic chain, for the
	// cos-st-derived-ok clause-2.2.2 base-chain walk).
	intT := mustST(t, "int", Atomic{Primitive: dec}, dec, nil, nil)
	int2 := mustST(t, "int2", Atomic{Primitive: dec}, intT, nil, nil)

	// Constructed lists/unions (base xs:anySimpleType).
	listOverDec := mustST(t, "decList", List{Item: dec}, anySimpleType, nil, nil)
	unionDecStr := mustST(t, "decStr", Union{Members: []*SimpleType{dec, str}}, anySimpleType, nil, nil)
	unionAllAtomic := mustST(t, "uAtomic", Union{Members: []*SimpleType{dec, str}}, anySimpleType, nil, nil)
	unionWithList := mustST(t, "uList", Union{Members: []*SimpleType{dec, listOverDec}}, anySimpleType, nil, nil)

	// Atomic items whose own {final} blocks a use, and a base whose {final}
	// blocks restriction (clause 3 / 1.2 / 2.2.2.2 / 3.2.2.2 shared site).
	itemFinalList := mustST(t, "finalList", Atomic{Primitive: dec}, dec, nil, []DerivationMethod{DerivationList})
	memberFinalUnion := mustST(t, "finalUnion", Atomic{Primitive: dec}, dec, nil, []DerivationMethod{DerivationUnion})
	baseFinalRestrict := mustST(t, "sealed", Atomic{Primitive: dec}, dec, nil, []DerivationMethod{DerivationRestriction})

	loc := xsderr.Loc{}
	qn := QName{Local: "D"}
	badFacet := NewFacet(FacetKind(200), []string{"x"}, false)

	tests := []struct {
		name     string
		build    func() error
		wantRule xsderr.Rule // "" => success expected
		wantSub  string
	}{
		// --- atomic (cos-st-restricts case 1) ---
		{"atomic ok (1.1)", func() error {
			_, e := NewSimpleType(loc, qn, Atomic{Primitive: dec}, dec, nil, nil)
			return e
		}, "", ""},
		{"atomic non-atomic base (1.1)", func() error {
			_, e := NewSimpleType(loc, qn, Atomic{Primitive: nil}, listOverDec, nil, nil)
			return e
		}, ruleCosSTRestricts, "clause 1.1"},

		// --- shared {final}-blocking site (clause 3 / 1.2 / 2.2.2.2 / 3.2.2.2) ---
		{"base {final} contains restriction (clause 3)", func() error {
			_, e := NewSimpleType(loc, qn, Atomic{Primitive: dec}, baseFinalRestrict, nil, nil)
			return e
		}, ruleSTPropsCorrect, "clause 3"},

		// --- clause 5 (facet support) ---
		{"unsupported facet (clause 5)", func() error {
			_, e := NewSimpleType(loc, qn, Atomic{Primitive: dec}, dec, []Facet{badFacet}, nil)
			return e
		}, ruleSTPropsCorrect, "clause 5"},

		// --- list (cos-st-restricts case 2) ---
		{"list ok constructed atomic item", func() error {
			_, e := NewSimpleType(loc, qn, List{Item: dec}, anySimpleType, nil, nil)
			return e
		}, "", ""},
		{"list ok constructed atomic-union item", func() error {
			_, e := NewSimpleType(loc, qn, List{Item: unionAllAtomic}, anySimpleType, nil, nil)
			return e
		}, "", ""},
		{"list special item (2.1)", func() error {
			_, e := NewSimpleType(loc, qn, List{Item: anyAtomicType}, anySimpleType, nil, nil)
			return e
		}, ruleCosSTRestricts, "clause 2.1"},
		{"list nested-list item (2.1)", func() error {
			_, e := NewSimpleType(loc, qn, List{Item: listOverDec}, anySimpleType, nil, nil)
			return e
		}, ruleCosSTRestricts, "clause 2.1"},
		{"list union-with-list item (2.1)", func() error {
			_, e := NewSimpleType(loc, qn, List{Item: unionWithList}, anySimpleType, nil, nil)
			return e
		}, ruleCosSTRestricts, "clause 2.1"},
		{"list constructed item {final} has list (2.2.1.1)", func() error {
			_, e := NewSimpleType(loc, qn, List{Item: itemFinalList}, anySimpleType, nil, nil)
			return e
		}, ruleCosSTRestricts, "clause 2.2.1.1"},
		{"list ok restricted same item", func() error {
			_, e := NewSimpleType(loc, qn, List{Item: dec}, listOverDec, nil, nil)
			return e
		}, "", ""},
		{"list ok restricted derived item (2.2.2.3 chain)", func() error {
			_, e := NewSimpleType(loc, qn, List{Item: int2}, listOverDec, nil, nil)
			return e
		}, "", ""},
		{"list restricted non-list base (2.2.2.1)", func() error {
			_, e := NewSimpleType(loc, qn, List{Item: dec}, dec, nil, nil)
			return e
		}, ruleCosSTRestricts, "clause 2.2.2.1"},
		{"list restricted item not derived (2.2.2.3)", func() error {
			_, e := NewSimpleType(loc, qn, List{Item: str}, listOverDec, nil, nil)
			return e
		}, ruleCosSTRestricts, "clause 2.2.2.3"},

		// --- union (cos-st-restricts case 3) ---
		{"union ok constructed", func() error {
			_, e := NewSimpleType(loc, qn, Union{Members: []*SimpleType{dec, str}}, anySimpleType, nil, nil)
			return e
		}, "", ""},
		{"union special member (3.1)", func() error {
			_, e := NewSimpleType(loc, qn, Union{Members: []*SimpleType{dec, anyAtomicType}}, anySimpleType, nil, nil)
			return e
		}, ruleCosSTRestricts, "clause 3.1"},
		{"union constructed member {final} has union (3.2.1.1)", func() error {
			_, e := NewSimpleType(loc, qn, Union{Members: []*SimpleType{memberFinalUnion}}, anySimpleType, nil, nil)
			return e
		}, ruleCosSTRestricts, "clause 3.2.1.1"},
		{"union ok restricted corresponding members", func() error {
			_, e := NewSimpleType(loc, qn, Union{Members: []*SimpleType{dec, str}}, unionDecStr, nil, nil)
			return e
		}, "", ""},
		{"union restricted non-union base (3.2.2.1)", func() error {
			_, e := NewSimpleType(loc, qn, Union{Members: []*SimpleType{dec, str}}, dec, nil, nil)
			return e
		}, ruleCosSTRestricts, "clause 3.2.2.1"},
		{"union restricted member count mismatch (3.2.2.3)", func() error {
			_, e := NewSimpleType(loc, qn, Union{Members: []*SimpleType{dec}}, unionDecStr, nil, nil)
			return e
		}, ruleCosSTRestricts, "clause 3.2.2.3"},
		{"union restricted member not derived (3.2.2.3)", func() error {
			_, e := NewSimpleType(loc, qn, Union{Members: []*SimpleType{str, dec}}, unionDecStr, nil, nil)
			return e
		}, ruleCosSTRestricts, "clause 3.2.2.3"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.build()
			if tc.wantRule == "" {
				if err != nil {
					t.Fatalf("build() = %v, want success", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("build() = nil, want rejection %s", tc.wantRule)
			}
			gotRule, ok := xsderr.RuleOf(err)
			if !ok || gotRule != tc.wantRule {
				t.Fatalf("build() rule = %q (ok=%v), want %q; err=%v", gotRule, ok, tc.wantRule, err)
			}
			if tc.wantSub != "" && !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("build() message %q does not cite %q", err.Error(), tc.wantSub)
			}
		})
	}
}

// scaleFacet is a test shorthand for a maxScale/minScale Facet with a lexical
// integer {value} and a {fixed} flag.
func scaleFacet(kind FacetKind, value string, fixed bool) Facet {
	return NewFacet(kind, []string{value}, fixed)
}

// TestScaleFacetSCCs exercises checkScaleFacets: the value-restriction SCCs
// (maxScale-valid-restriction §4.2.4, minScale-valid-restriction §4.3.4), the
// minScale ≤ maxScale consistency SCC (spec anchor minScale-totalDigits), and
// the {fixed}-inheritance SCCs (f-ms-fixed §4.2.1, f-mns-fixed §4.3.1). Every
// prerequisite type is built through the real constructors.
func TestScaleFacetSCCs(t *testing.T) {
	pdec := mustPrim(t, "precisionDecimal")

	// Bases carrying effective scale facets for the restriction SCCs.
	baseMax5 := mustST(t, "baseMax5", Atomic{Primitive: pdec}, pdec,
		[]Facet{scaleFacet(FacetMaxScale, "5", false)}, nil)
	baseMin2 := mustST(t, "baseMin2", Atomic{Primitive: pdec}, pdec,
		[]Facet{scaleFacet(FacetMinScale, "2", false)}, nil)
	baseMaxFixed5 := mustST(t, "baseMaxFixed5", Atomic{Primitive: pdec}, pdec,
		[]Facet{scaleFacet(FacetMaxScale, "5", true)}, nil)
	baseMinFixed2 := mustST(t, "baseMinFixed2", Atomic{Primitive: pdec}, pdec,
		[]Facet{scaleFacet(FacetMinScale, "2", true)}, nil)

	// Multi-level chain A(fixed maxScale=5) <- B(no own scale) for the transitive
	// {fixed} check through EffectiveFacets.
	chainB := mustST(t, "chainB", Atomic{Primitive: pdec}, baseMaxFixed5, nil, nil)

	loc := xsderr.Loc{}
	qn := QName{Local: "D"}

	tests := []struct {
		name     string
		build    func() error
		wantRule xsderr.Rule // "" => success expected
	}{
		// --- maxScale-valid-restriction (§4.2.4): may only move down ---
		{"maxScale narrows below base ok", func() error {
			_, e := NewSimpleType(loc, qn, Atomic{Primitive: pdec}, baseMax5,
				[]Facet{scaleFacet(FacetMaxScale, "3", false)}, nil)
			return e
		}, ""},
		{"maxScale equals base ok", func() error {
			_, e := NewSimpleType(loc, qn, Atomic{Primitive: pdec}, baseMax5,
				[]Facet{scaleFacet(FacetMaxScale, "5", false)}, nil)
			return e
		}, ""},
		{"maxScale above base rejected", func() error {
			_, e := NewSimpleType(loc, qn, Atomic{Primitive: pdec}, baseMax5,
				[]Facet{scaleFacet(FacetMaxScale, "7", false)}, nil)
			return e
		}, ruleMaxScaleValidRestriction},
		{"maxScale vacuous no base maxScale ok", func() error {
			_, e := NewSimpleType(loc, qn, Atomic{Primitive: pdec}, pdec,
				[]Facet{scaleFacet(FacetMaxScale, "9", false)}, nil)
			return e
		}, ""},
		{"maxScale non-integer literal rejected not panic", func() error {
			// A malformed scale {value} ("abc") reaches scaleValue through the
			// public NewFacet/NewSimpleType API; it must charge a real *xsderr.Error,
			// not panic (regression guard for #157).
			_, e := NewSimpleType(loc, qn, Atomic{Primitive: pdec}, baseMax5,
				[]Facet{scaleFacet(FacetMaxScale, "abc", false)}, nil)
			return e
		}, ruleMaxScaleValidRestriction},

		// --- minScale-valid-restriction (§4.3.4): may only move up ---
		{"minScale above base ok", func() error {
			_, e := NewSimpleType(loc, qn, Atomic{Primitive: pdec}, baseMin2,
				[]Facet{scaleFacet(FacetMinScale, "4", false)}, nil)
			return e
		}, ""},
		{"minScale equals base ok", func() error {
			_, e := NewSimpleType(loc, qn, Atomic{Primitive: pdec}, baseMin2,
				[]Facet{scaleFacet(FacetMinScale, "2", false)}, nil)
			return e
		}, ""},
		{"minScale below base rejected", func() error {
			_, e := NewSimpleType(loc, qn, Atomic{Primitive: pdec}, baseMin2,
				[]Facet{scaleFacet(FacetMinScale, "1", false)}, nil)
			return e
		}, ruleMinScaleValidRestriction},

		// --- minScale ≤ maxScale consistency (anchor minScale-totalDigits) ---
		{"minScale gt maxScale rejected", func() error {
			_, e := NewSimpleType(loc, qn, Atomic{Primitive: pdec}, pdec,
				[]Facet{
					scaleFacet(FacetMinScale, "5", false),
					scaleFacet(FacetMaxScale, "2", false),
				}, nil)
			return e
		}, ruleMinScaleLEMaxScale},
		{"minScale eq maxScale ok", func() error {
			_, e := NewSimpleType(loc, qn, Atomic{Primitive: pdec}, pdec,
				[]Facet{
					scaleFacet(FacetMinScale, "3", false),
					scaleFacet(FacetMaxScale, "3", false),
				}, nil)
			return e
		}, ""},

		// --- f-ms-fixed (§4.2.1): fixed base maxScale may not be overridden ---
		{"fixed maxScale repeated identical ok", func() error {
			_, e := NewSimpleType(loc, qn, Atomic{Primitive: pdec}, baseMaxFixed5,
				[]Facet{scaleFacet(FacetMaxScale, "5", true)}, nil)
			return e
		}, ""},
		{"fixed maxScale further-narrowing rejected", func() error {
			// value 3 satisfies maxScale-valid-restriction (3 < 5) yet still
			// overrides the {fixed} base facet — proves f-ms-fixed is independent.
			_, e := NewSimpleType(loc, qn, Atomic{Primitive: pdec}, baseMaxFixed5,
				[]Facet{scaleFacet(FacetMaxScale, "3", true)}, nil)
			return e
		}, ruleMaxScaleFixed},

		// --- f-mns-fixed (§4.3.1): fixed base minScale may not be overridden ---
		{"fixed minScale repeated identical ok", func() error {
			_, e := NewSimpleType(loc, qn, Atomic{Primitive: pdec}, baseMinFixed2,
				[]Facet{scaleFacet(FacetMinScale, "2", true)}, nil)
			return e
		}, ""},
		{"fixed minScale further-widening rejected", func() error {
			// value 4 satisfies minScale-valid-restriction (4 > 2) yet still
			// overrides the {fixed} base facet — proves f-mns-fixed is independent.
			_, e := NewSimpleType(loc, qn, Atomic{Primitive: pdec}, baseMinFixed2,
				[]Facet{scaleFacet(FacetMinScale, "4", true)}, nil)
			return e
		}, ruleMinScaleFixed},

		// --- multi-level transitive {fixed} through EffectiveFacets ---
		{"chain C overrides inherited fixed maxScale rejected", func() error {
			// C restricts B which restricts A; A fixes maxScale=5, B declares no
			// own scale facet, so C inherits the fixed facet transitively.
			_, e := NewSimpleType(loc, qn, Atomic{Primitive: pdec}, chainB,
				[]Facet{scaleFacet(FacetMaxScale, "4", true)}, nil)
			return e
		}, ruleMaxScaleFixed},
		{"chain C inherits fixed maxScale unchanged ok", func() error {
			_, e := NewSimpleType(loc, qn, Atomic{Primitive: pdec}, chainB, nil, nil)
			return e
		}, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.build()
			if tc.wantRule == "" {
				if err != nil {
					t.Fatalf("build() = %v, want success", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("build() = nil, want rejection %s", tc.wantRule)
			}
			gotRule, ok := xsderr.RuleOf(err)
			if !ok || gotRule != tc.wantRule {
				t.Fatalf("build() rule = %q (ok=%v), want %q; err=%v", gotRule, ok, tc.wantRule, err)
			}
		})
	}
}

// TestDerivedOKSimple pins the cos-st-derived-ok (§3.16.6.3) relation directly:
// identity (clause 1), the base-chain walk (clause 2.2.1/2.2.2), the
// list/union-of-anySimpleType shortcut (clause 2.2.3), and the union-member
// alternative (clause 2.2.4), plus its negative.
func TestDerivedOKSimple(t *testing.T) {
	dec := mustPrim(t, "decimal")
	str := mustPrim(t, "string")
	intT := mustST(t, "int", Atomic{Primitive: dec}, dec, nil, nil)
	int2 := mustST(t, "int2", Atomic{Primitive: dec}, intT, nil, nil)
	listOverDec := mustST(t, "decList", List{Item: dec}, anySimpleType, nil, nil)
	unionDecStr := mustST(t, "decStr", Union{Members: []*SimpleType{dec, str}}, anySimpleType, nil, nil)

	tests := []struct {
		name string
		d, b *SimpleType
		want bool
	}{
		{"identity (1)", dec, dec, true},
		{"direct base (2.2.1)", intT, dec, true},
		{"base chain (2.2.2)", int2, dec, true},
		{"list from anySimpleType (2.2.3)", listOverDec, anySimpleType, true},
		{"union from anySimpleType (2.2.3)", unionDecStr, anySimpleType, true},
		{"union member (2.2.4)", dec, unionDecStr, true},
		{"unrelated (none)", str, dec, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := derivedOKSimple(tc.d, tc.b); got != tc.want {
				t.Fatalf("derivedOKSimple = %v, want %v", got, tc.want)
			}
		})
	}
}
