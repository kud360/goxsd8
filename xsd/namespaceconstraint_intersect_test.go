package xsd

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// TestIntersectNamespaceConstraintSemantics is the core cos-aw-intersect
// (§3.10.6.4) property: the intersection admits an expanded name (and a namespace
// name) iff BOTH operands admit it. It probes a fixed set of namespace names and
// expanded names against every operand pair, so a wrong variety/set arm or a
// dropped/kept {disallowed names} member is caught as a conjunction mismatch.
func TestIntersectNamespaceConstraintSemantics(t *testing.T) {
	nsA, nsB, nsC := NamespaceName("urn:a"), NamespaceName("urn:b"), NamespaceName("urn:c")
	absent := Namespace{}
	qn := func(space, local string) QName { return QName{Space: space, Local: local} }

	// Operands spanning all five §3.10.6.4 cases plus {disallowed names}.
	any := cNC(t, NamespaceConstraintAny, nil, nil, nil)
	enumAB := cNC(t, NamespaceConstraintEnumeration, []Namespace{nsA, nsB}, nil, nil)
	enumBC := cNC(t, NamespaceConstraintEnumeration, []Namespace{nsB, nsC}, nil, nil)
	// Disjoint from enum{A,B}: their intersection is the EMPTY enumeration.
	enumC := cNC(t, NamespaceConstraintEnumeration, []Namespace{nsC}, nil, nil)
	notA := cNC(t, NamespaceConstraintNot, []Namespace{nsA}, nil, nil)
	notB := cNC(t, NamespaceConstraintNot, []Namespace{nsB}, nil, nil)
	// enum with a disallowed name in nsA (allowed by enum{A,B}): intersecting with
	// enum{B,C} (which rejects nsA) must DROP it — feeding it unfiltered to
	// NewNamespaceConstraint would trip w-props-correct clause 4.
	enumABdisA := cNC(t, NamespaceConstraintEnumeration,
		[]Namespace{nsA, nsB}, []QName{qn("urn:a", "x")}, nil)
	// not{A} with a disallowed name in nsB (allowed by not{A}): survives ∩ not{A}.
	notAdisB := cNC(t, NamespaceConstraintNot,
		[]Namespace{nsA}, []QName{qn("urn:b", "y")}, nil)

	pairs := []struct {
		name string
		a, b NamespaceConstraint
	}{
		{"any∩enum", any, enumAB},
		{"any∩not", any, notA},
		{"enum∩enum", enumAB, enumBC},
		{"not∩not", notA, notB},
		{"not∩enum", notA, enumAB},
		{"enum∩enum-empty", enumAB, enumC}, // {A,B} ∩ {C} = ∅
		{"identical-enum", enumAB, enumAB},
		{"identical-not", notA, notA},
		{"identical-any", any, any},
		{"enum-disallowed-dropped", enumABdisA, enumBC},
		{"not-disallowed-kept", notAdisB, notA},
	}

	probeNS := []Namespace{nsA, nsB, nsC, absent}
	probeNames := []QName{
		qn("urn:a", "x"), qn("urn:b", "y"), qn("urn:a", "z"),
		qn("urn:b", "x"), qn("urn:c", "w"), qn("", "n"),
	}

	for _, p := range pairs {
		// Commutativity: both orders must agree with the conjunction.
		for _, ab := range []struct{ a, b NamespaceConstraint }{{p.a, p.b}, {p.b, p.a}} {
			got, err := intersectNamespaceConstraint(xsderr.Loc{}, ab.a, ab.b)
			if err != nil {
				t.Fatalf("%s: intersectNamespaceConstraint errored (should be unreachable for valid operands): %v", p.name, err)
			}
			for _, n := range probeNS {
				want := ab.a.AllowsNamespace(n) && ab.b.AllowsNamespace(n)
				if got.AllowsNamespace(n) != want {
					t.Errorf("%s: AllowsNamespace(%v) = %v, want %v (conjunction)", p.name, n, got.AllowsNamespace(n), want)
				}
			}
			for _, name := range probeNames {
				want := ab.a.AllowsName(name) && ab.b.AllowsName(name)
				if got.AllowsName(name) != want {
					t.Errorf("%s: AllowsName(%v) = %v, want %v (conjunction)", p.name, name, got.AllowsName(name), want)
				}
			}
		}
	}
}

// TestIntersectNamespaceConstraintVarieties pins the §3.10.6.4 variety/set table
// on representative cases so a regression in an arm is caught structurally, not
// only via the semantic conjunction.
func TestIntersectNamespaceConstraintVarieties(t *testing.T) {
	nsA, nsB, nsC := NamespaceName("urn:a"), NamespaceName("urn:b"), NamespaceName("urn:c")
	any := cNC(t, NamespaceConstraintAny, nil, nil, nil)
	enumAB := cNC(t, NamespaceConstraintEnumeration, []Namespace{nsA, nsB}, nil, nil)
	enumBC := cNC(t, NamespaceConstraintEnumeration, []Namespace{nsB, nsC}, nil, nil)
	notA := cNC(t, NamespaceConstraintNot, []Namespace{nsA}, nil, nil)
	notB := cNC(t, NamespaceConstraintNot, []Namespace{nsB}, nil, nil)

	cases := []struct {
		name    string
		a, b    NamespaceConstraint
		wantVar NamespaceConstraintVariety
		wantNS  []Namespace // nil means "empty"
	}{
		{"any∩enum→enum", any, enumAB, NamespaceConstraintEnumeration, []Namespace{nsA, nsB}},
		{"any∩not→not", any, notA, NamespaceConstraintNot, []Namespace{nsA}},
		{"enum∩enum→intersection", enumAB, enumBC, NamespaceConstraintEnumeration, []Namespace{nsB}},
		{"not∩not→union", notA, notB, NamespaceConstraintNot, []Namespace{nsA, nsB}},
		{"not∩enum→enum-minus", notA, enumAB, NamespaceConstraintEnumeration, []Namespace{nsB}},
	}
	for _, c := range cases {
		got, err := intersectNamespaceConstraint(xsderr.Loc{}, c.a, c.b)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", c.name, err)
		}
		if got.Variety() != c.wantVar {
			t.Errorf("%s: variety = %s, want %s", c.name, got.Variety(), c.wantVar)
		}
		if !sameNamespaces(got.Namespaces(), c.wantNS) {
			t.Errorf("%s: namespaces = %v, want %v", c.name, got.Namespaces(), c.wantNS)
		}
	}
}

// sameNamespaces compares two namespace slices for equality in order.
func sameNamespaces(got, want []Namespace) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
