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
