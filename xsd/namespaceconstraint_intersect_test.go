package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// TestIntersectNamespaceConstraintSemantics is the core cos-aw-intersect
// (§3.10.6.4) property: the intersection admits an expanded name (and a namespace
// name) iff BOTH operands admit it. It probes a fixed set of namespace names and
// expanded names against every operand pair, so a wrong variety/set arm or a
// dropped/kept {disallowed names} member is caught as a conjunction mismatch.
func TestIntersectNamespaceConstraintSemantics(t *testing.T) {
	nsA, nsB, nsC := xsd.NamespaceName("urn:a"), xsd.NamespaceName("urn:b"), xsd.NamespaceName("urn:c")
	absent := xsd.Namespace{}
	qn := func(space, local string) xsd.QName { return xsd.QName{Space: space, Local: local} }

	// Operands spanning all five §3.10.6.4 cases plus {disallowed names}.
	any := mustConstraint(t, xsd.NamespaceConstraintAny, nil, nil)
	enumAB := mustConstraint(t, xsd.NamespaceConstraintEnumeration, []xsd.Namespace{nsA, nsB}, nil)
	enumBC := mustConstraint(t, xsd.NamespaceConstraintEnumeration, []xsd.Namespace{nsB, nsC}, nil)
	notA := mustConstraint(t, xsd.NamespaceConstraintNot, []xsd.Namespace{nsA}, nil)
	notB := mustConstraint(t, xsd.NamespaceConstraintNot, []xsd.Namespace{nsB}, nil)
	// enum with a disallowed name in nsA (allowed by enum{A,B}): intersecting with
	// enum{B,C} (which rejects nsA) must DROP it — feeding it unfiltered to
	// NewNamespaceConstraint would trip w-props-correct clause 4.
	enumABdisA := mustConstraint(t, xsd.NamespaceConstraintEnumeration,
		[]xsd.Namespace{nsA, nsB}, []xsd.QName{qn("urn:a", "x")})
	// not{A} with a disallowed name in nsB (allowed by not{A}): survives ∩ not{A}.
	notAdisB := mustConstraint(t, xsd.NamespaceConstraintNot,
		[]xsd.Namespace{nsA}, []xsd.QName{qn("urn:b", "y")})

	pairs := []struct {
		name string
		a, b xsd.NamespaceConstraint
	}{
		{"any∩enum", any, enumAB},
		{"any∩not", any, notA},
		{"enum∩enum", enumAB, enumBC},
		{"not∩not", notA, notB},
		{"not∩enum", notA, enumAB},
		{"enum∩enum-empty", enumAB, enumBC}, // {B}
		{"identical-enum", enumAB, enumAB},
		{"identical-not", notA, notA},
		{"identical-any", any, any},
		{"enum-disallowed-dropped", enumABdisA, enumBC},
		{"not-disallowed-kept", notAdisB, notA},
	}

	probeNS := []xsd.Namespace{nsA, nsB, nsC, absent}
	probeNames := []xsd.QName{
		qn("urn:a", "x"), qn("urn:b", "y"), qn("urn:a", "z"),
		qn("urn:b", "x"), qn("urn:c", "w"), qn("", "n"),
	}

	for _, p := range pairs {
		// Commutativity: both orders must agree with the conjunction.
		for _, ab := range []struct{ a, b xsd.NamespaceConstraint }{{p.a, p.b}, {p.b, p.a}} {
			got, err := xsd.IntersectNamespaceConstraint(xsderr.Loc{}, ab.a, ab.b)
			if err != nil {
				t.Fatalf("%s: IntersectNamespaceConstraint errored (should be unreachable for valid operands): %v", p.name, err)
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
	nsA, nsB, nsC := xsd.NamespaceName("urn:a"), xsd.NamespaceName("urn:b"), xsd.NamespaceName("urn:c")
	any := mustConstraint(t, xsd.NamespaceConstraintAny, nil, nil)
	enumAB := mustConstraint(t, xsd.NamespaceConstraintEnumeration, []xsd.Namespace{nsA, nsB}, nil)
	enumBC := mustConstraint(t, xsd.NamespaceConstraintEnumeration, []xsd.Namespace{nsB, nsC}, nil)
	notA := mustConstraint(t, xsd.NamespaceConstraintNot, []xsd.Namespace{nsA}, nil)
	notB := mustConstraint(t, xsd.NamespaceConstraintNot, []xsd.Namespace{nsB}, nil)

	cases := []struct {
		name    string
		a, b    xsd.NamespaceConstraint
		wantVar xsd.NamespaceConstraintVariety
		wantNS  []xsd.Namespace // nil means "empty"
	}{
		{"any∩enum→enum", any, enumAB, xsd.NamespaceConstraintEnumeration, []xsd.Namespace{nsA, nsB}},
		{"any∩not→not", any, notA, xsd.NamespaceConstraintNot, []xsd.Namespace{nsA}},
		{"enum∩enum→intersection", enumAB, enumBC, xsd.NamespaceConstraintEnumeration, []xsd.Namespace{nsB}},
		{"not∩not→union", notA, notB, xsd.NamespaceConstraintNot, []xsd.Namespace{nsA, nsB}},
		{"not∩enum→enum-minus", notA, enumAB, xsd.NamespaceConstraintEnumeration, []xsd.Namespace{nsB}},
	}
	for _, c := range cases {
		got, err := xsd.IntersectNamespaceConstraint(xsderr.Loc{}, c.a, c.b)
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
func sameNamespaces(got, want []xsd.Namespace) bool {
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
