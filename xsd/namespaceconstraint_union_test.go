package xsd

import (
	"slices"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// uUnion folds a and b through UnionNamespaceConstraint, failing the test on the
// error slot the operator documents as unreachable for valid operands.
func uUnion(t *testing.T, a, b NamespaceConstraint) NamespaceConstraint {
	t.Helper()
	got, err := UnionNamespaceConstraint(xsderr.Loc{}, a, b)
	if err != nil {
		t.Fatalf("UnionNamespaceConstraint errored (unreachable for valid operands): %v", err)
	}
	return got
}

// TestUnionNamespaceConstraintVarieties pins the §3.10.6.3 five-case
// variety/{namespaces} table, including the two sub-cases (4.1 and 5.1) where an
// empty computed set turns the result into any rather than a not constraint
// w-props-correct clause 2 would reject. Every pair is checked in both operand
// orders, since cases 4 and 5 are the ones a one-sided arm silently gets wrong.
func TestUnionNamespaceConstraintVarieties(t *testing.T) {
	a, b, c := NamespaceName("urn:a"), NamespaceName("urn:b"), NamespaceName("urn:c")
	anyC := cNC(t, NamespaceConstraintAny, nil, nil, nil)
	enumA := cNC(t, NamespaceConstraintEnumeration, []Namespace{a}, nil, nil)
	enumAB := cNC(t, NamespaceConstraintEnumeration, []Namespace{a, b}, nil, nil)
	enumB := cNC(t, NamespaceConstraintEnumeration, []Namespace{b}, nil, nil)
	notA := cNC(t, NamespaceConstraintNot, []Namespace{a}, nil, nil)
	notB := cNC(t, NamespaceConstraintNot, []Namespace{b}, nil, nil)
	notAB := cNC(t, NamespaceConstraintNot, []Namespace{a, b}, nil, nil)
	notBC := cNC(t, NamespaceConstraintNot, []Namespace{b, c}, nil, nil)

	for _, tc := range []struct {
		name    string
		x, y    NamespaceConstraint
		wantVar NamespaceConstraintVariety
		wantNS  []Namespace // nil means "empty"
	}{
		{name: "case 1 identical enumeration", x: enumAB, y: enumAB,
			wantVar: NamespaceConstraintEnumeration, wantNS: []Namespace{a, b}},
		{name: "case 1 identical not", x: notA, y: notA,
			wantVar: NamespaceConstraintNot, wantNS: []Namespace{a}},
		{name: "case 1 identical any", x: anyC, y: anyC, wantVar: NamespaceConstraintAny},
		{name: "case 2 any over enumeration", x: anyC, y: enumA, wantVar: NamespaceConstraintAny},
		{name: "case 2 any over not", x: anyC, y: notA, wantVar: NamespaceConstraintAny},
		{name: "case 3 enumeration union", x: enumA, y: enumB,
			wantVar: NamespaceConstraintEnumeration, wantNS: []Namespace{a, b}},
		{name: "case 4.1 disjoint nots collapse to any", x: notA, y: notB, wantVar: NamespaceConstraintAny},
		{name: "case 4.2 nots intersect", x: notAB, y: notBC,
			wantVar: NamespaceConstraintNot, wantNS: []Namespace{b}},
		{name: "case 5.1 empty difference collapses to any", x: notA, y: enumA, wantVar: NamespaceConstraintAny},
		{name: "case 5.2 non-empty difference", x: notAB, y: enumA,
			wantVar: NamespaceConstraintNot, wantNS: []Namespace{b}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := uUnion(t, tc.x, tc.y)
			if got.Variety() != tc.wantVar {
				t.Errorf("variety = %s, want %s", got.Variety(), tc.wantVar)
			}
			if !slices.Equal(got.Namespaces(), tc.wantNS) {
				t.Errorf("namespaces = %v, want %v", got.Namespaces(), tc.wantNS)
			}
			// The swapped order must reach the same {variety} — cases 4 and 5
			// each have a one-sided arm that only fires for one orientation. Its
			// {namespaces} is compared as a SET, not by position: case 3 lays the
			// members out in operand document order (x's, then y's new ones), so
			// the reversed union of enum{a} and enum{b} is legitimately [b a].
			swapped := uUnion(t, tc.y, tc.x)
			if swapped.Variety() != tc.wantVar {
				t.Errorf("swapped variety = %s, want %s", swapped.Variety(), tc.wantVar)
			}
			if len(swapped.Namespaces()) != len(tc.wantNS) {
				t.Errorf("swapped namespaces = %v, want the members of %v", swapped.Namespaces(), tc.wantNS)
			}
			for _, n := range tc.wantNS {
				if !slices.Contains(swapped.Namespaces(), n) {
					t.Errorf("swapped namespaces = %v, missing %v", swapped.Namespaces(), n)
				}
			}
		})
	}
}

// TestUnionNamespaceConstraintSemantics is the core cos-aw-union property on the
// name axes the operator is exact about: the union admits a namespace name (and
// an expanded name) iff EITHER operand admits it. It probes a fixed set of
// namespace names and expanded names against every operand pair, so a wrong
// variety/set arm or a wrongly kept/dropped {disallowed names} member shows up as
// a disjunction mismatch.
//
// The keyword half of {disallowed names} is deliberately outside this property —
// §3.10.6.3 bullet 3's Note licenses the union to drop defined and thereby "allow
// QNames that are not allowed by either wildcard" — and is pinned separately by
// TestUnionDisallowedNameKeywords. AllowsName decides cvc-wildcard-name only, not
// the keyword clauses, so the disjunction is exact for it.
func TestUnionNamespaceConstraintSemantics(t *testing.T) {
	a, b, c := NamespaceName("urn:a"), NamespaceName("urn:b"), NamespaceName("urn:c")
	absent := Namespace{}
	qn := func(space, local string) QName { return QName{Space: space, Local: local} }

	anyC := cNC(t, NamespaceConstraintAny, nil, nil, nil)
	enumAB := cNC(t, NamespaceConstraintEnumeration, []Namespace{a, b}, nil, nil)
	enumBC := cNC(t, NamespaceConstraintEnumeration, []Namespace{b, c}, nil, nil)
	notA := cNC(t, NamespaceConstraintNot, []Namespace{a}, nil, nil)
	notB := cNC(t, NamespaceConstraintNot, []Namespace{b}, nil, nil)
	notAB := cNC(t, NamespaceConstraintNot, []Namespace{a, b}, nil, nil)
	// enum{a,b} disallowing {a}:x — the other operand decides whether the union
	// keeps it (bullets 1-2 are an expanded-name test, not a namespace test).
	enumABdisA := cNC(t, NamespaceConstraintEnumeration, []Namespace{a, b}, []QName{qn("urn:a", "x")}, nil)
	notAdisB := cNC(t, NamespaceConstraintNot, []Namespace{a}, []QName{qn("urn:b", "y")}, nil)
	anyDisA := cNC(t, NamespaceConstraintAny, nil, []QName{qn("urn:a", "x")}, nil)

	pairs := []struct {
		name string
		x, y NamespaceConstraint
	}{
		{"any∪enum", anyC, enumAB},
		{"any∪not", anyC, notA},
		{"enum∪enum", enumAB, enumBC},
		{"not∪not-disjoint", notA, notB},
		{"not∪not-overlapping", notAB, notA},
		{"not∪enum-covering", notA, enumAB},
		{"not∪enum-partial", notAB, enumAB},
		{"identical-enum", enumAB, enumAB},
		{"identical-not", notA, notA},
		{"identical-any", anyC, anyC},
		{"disallowed dropped by the other operand", enumABdisA, enumAB},
		{"disallowed kept against a disjoint enumeration", enumABdisA, enumBC},
		{"disallowed kept against a narrow not", enumABdisA, notAB},
		{"disallowed on a not operand", notAdisB, notA},
		{"disallowed on both operands", anyDisA, enumABdisA},
	}

	probeNS := []Namespace{a, b, c, absent}
	probeNames := []QName{
		qn("urn:a", "x"), qn("urn:b", "y"), qn("urn:a", "z"),
		qn("urn:b", "x"), qn("urn:c", "w"), qn("", "n"),
	}

	for _, p := range pairs {
		t.Run(p.name, func(t *testing.T) {
			// Commutativity: both orders must agree with the disjunction.
			for _, xy := range []struct{ x, y NamespaceConstraint }{{p.x, p.y}, {p.y, p.x}} {
				got := uUnion(t, xy.x, xy.y)
				for _, n := range probeNS {
					want := xy.x.AllowsNamespace(n) || xy.y.AllowsNamespace(n)
					if got.AllowsNamespace(n) != want {
						t.Errorf("AllowsNamespace(%v) = %v, want %v (disjunction)", n, got.AllowsNamespace(n), want)
					}
				}
				for _, name := range probeNames {
					want := xy.x.AllowsName(name) || xy.y.AllowsName(name)
					if got.AllowsName(name) != want {
						t.Errorf("AllowsName(%v) = %v, want %v (disjunction)", name, got.AllowsName(name), want)
					}
				}
			}
		})
	}
}

// TestUnionDisallowedNames pins §3.10.6.3 bullets 1-2 structurally: a QName
// member survives the union exactly when the OTHER operand does not allow it, as
// defined by cvc-wildcard-name (§3.10.4.2). The third case is the one that
// separates this predicate from cos-aw-intersect's, which asks only about the
// member's NAMESPACE name (cvc-wildcard-namespace): there the other operand
// allows the namespace urn:a but not the expanded name {urn:a}x, so a
// namespace-level test would wrongly drop the member.
func TestUnionDisallowedNames(t *testing.T) {
	a, b := NamespaceName("urn:a"), NamespaceName("urn:b")
	x := QName{Space: "urn:a", Local: "x"}
	for _, tc := range []struct {
		name string
		p, q NamespaceConstraint
		want []QName
	}{
		{name: "dropped: the other operand allows the name",
			p:    cNC(t, NamespaceConstraintEnumeration, []Namespace{a}, []QName{x}, nil),
			q:    cNC(t, NamespaceConstraintEnumeration, []Namespace{a}, nil, nil),
			want: nil},
		{name: "kept: the other operand rejects the namespace",
			p:    cNC(t, NamespaceConstraintEnumeration, []Namespace{a}, []QName{x}, nil),
			q:    cNC(t, NamespaceConstraintEnumeration, []Namespace{b}, nil, nil),
			want: []QName{x}},
		{name: "kept: the other operand allows the namespace but disallows the name",
			p:    cNC(t, NamespaceConstraintEnumeration, []Namespace{a}, []QName{x}, nil),
			q:    cNC(t, NamespaceConstraintAny, nil, []QName{x}, nil),
			want: []QName{x}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Both orders: the result is one set, built a-then-b, deduplicated.
			for _, pq := range []struct{ p, q NamespaceConstraint }{{tc.p, tc.q}, {tc.q, tc.p}} {
				got := uUnion(t, pq.p, pq.q)
				if !slices.Equal(got.disallowedNames, tc.want) {
					t.Errorf("{disallowed names} QName half = %v, want %v", got.disallowedNames, tc.want)
				}
			}
		})
	}
}

// TestUnionDisallowedNameKeywords pins §3.10.6.3 bullet 3, the clause whose
// polarity is the exact opposite of cos-aw-intersect's bullet 3 and which the LOG
// records having been transposed once already: defined survives the union only
// when BOTH operands carry it. The intersection of the same two operands is
// checked alongside, because the bug this guards against is invisible unless the
// two operators disagree.
func TestUnionDisallowedNameKeywords(t *testing.T) {
	defined := []DisallowedNameKeyword{DisallowedNameDefined}
	withDefined := cNC(t, NamespaceConstraintAny, nil, nil, defined)
	plain := cNC(t, NamespaceConstraintAny, nil, nil, nil)
	for _, tc := range []struct {
		name           string
		p, q           NamespaceConstraint
		wantUnion      bool
		wantIntersects bool
	}{
		{name: "both carry defined", p: withDefined, q: withDefined, wantUnion: true, wantIntersects: true},
		{name: "only one carries defined", p: withDefined, q: plain, wantUnion: false, wantIntersects: true},
		{name: "neither carries defined", p: plain, q: plain, wantUnion: false, wantIntersects: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, pq := range []struct{ p, q NamespaceConstraint }{{tc.p, tc.q}, {tc.q, tc.p}} {
				joined := uUnion(t, pq.p, pq.q).hasDisallowedNameKeyword(DisallowedNameDefined)
				if joined != tc.wantUnion {
					t.Errorf("union carries defined = %t, want %t", joined, tc.wantUnion)
				}
				meet, err := IntersectNamespaceConstraint(xsderr.Loc{}, pq.p, pq.q)
				if err != nil {
					t.Fatalf("IntersectNamespaceConstraint: %v", err)
				}
				if met := meet.hasDisallowedNameKeyword(DisallowedNameDefined); met != tc.wantIntersects {
					t.Errorf("intersection carries defined = %t, want %t", met, tc.wantIntersects)
				}
			}
		})
	}
}
