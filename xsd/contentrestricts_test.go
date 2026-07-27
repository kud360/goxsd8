package xsd

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// cRestricts finalizes a schema whose "derived" complex type restricts its
// "base", both with the given element-only content models, and reports whether
// derivation-ok-restriction accepted it. Every other clause of the constraint is
// discharged by construction here — the base is not {final}, neither type
// carries an {attribute use}, and both content types are element-only — so the
// only clause that can decline is 2.4.2.
func cRestricts(t *testing.T, base, derived ModelGroup) error {
	t.Helper()
	return dFinalize(t, func(b *SchemaBuilder) {
		b.AddType(dType(t, uq("base"), anyTypeName, dElementContent(t, false, base), nil, nil))
		b.AddType(dType(t, uq("derived"), uq("base"), dElementContent(t, false, derived), nil, nil))
	})
}

// cElem is a particle over a local element declaration of the given name, typed
// T, with the given occurrence bounds.
func cElem(t *testing.T, local string, minOccurs, maxOccurs int) Particle {
	t.Helper()
	return uParticle(t, uOccurs(t, minOccurs, maxOccurs), ResolvedTerm{Term: uLocal(t, uq(local), uq("T"))})
}

// cAny is a particle over a wildcard with the given {namespace constraint} and
// {process contents}, occurring exactly once.
func cAny(t *testing.T, variety NamespaceConstraintVariety, namespaces []Namespace, pc ProcessContents) Particle {
	t.Helper()
	return uOne(t, ResolvedTerm{Term: uWildcard(t, variety, namespaces, pc)})
}

// TestContentRestrictsOccurrenceRange pins cos-content-act-restrict clause 1
// over the occurrence ranges the position automaton unfolds: narrowing a range
// is a restriction, widening one is not, because a widened range admits
// sequences the base's automaton has no state to accept.
func TestContentRestrictsOccurrenceRange(t *testing.T) {
	for _, tc := range []struct {
		name           string
		bMin, bMax     int
		rMin, rMax     int
		wantRestricted bool
	}{
		{name: "identical", bMin: 1, bMax: 1, rMin: 1, rMax: 1, wantRestricted: true},
		{name: "narrowed maximum", bMin: 1, bMax: 3, rMin: 1, rMax: 1, wantRestricted: true},
		{name: "raised minimum", bMin: 0, bMax: 2, rMin: 2, rMax: 2, wantRestricted: true},
		{name: "widened maximum", bMin: 1, bMax: 1, rMin: 1, rMax: 3, wantRestricted: false},
		{name: "lowered minimum", bMin: 2, bMax: 2, rMin: 0, rMax: 2, wantRestricted: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := cRestricts(t,
				uGroup(t, CompositorSequence, cElem(t, "e", tc.bMin, tc.bMax)),
				uGroup(t, CompositorSequence, cElem(t, "e", tc.rMin, tc.rMax)))
			if tc.wantRestricted && err != nil {
				t.Fatalf("a valid content-model restriction was rejected: %v", err)
			}
			if !tc.wantRestricted {
				expectRule(t, err, ruleDerivationOKRestriction)
			}
		})
	}
}

// TestContentRestrictsNewElementRejected pins cos-content-act-restrict clause 1
// for the shape no occurrence range can excuse: the restriction adds a member
// the base's content model never admits, so a sequence valid against the
// restriction is not valid against the base.
func TestContentRestrictsNewElementRejected(t *testing.T) {
	err := cRestricts(t,
		uGroup(t, CompositorSequence, cElem(t, "a", 1, 1)),
		uGroup(t, CompositorSequence, cElem(t, "a", 1, 1), cElem(t, "b", 1, 1)))
	expectRule(t, err, ruleDerivationOKRestriction)
}

// TestContentRestrictsDroppedElementAccepted is the control: dropping an
// optional member narrows the language, which is exactly what a restriction is.
func TestContentRestrictsDroppedElementAccepted(t *testing.T) {
	err := cRestricts(t,
		uGroup(t, CompositorSequence, cElem(t, "a", 1, 1), cElem(t, "b", 0, 1)),
		uGroup(t, CompositorSequence, cElem(t, "a", 1, 1)))
	if err != nil {
		t.Fatalf("dropping an optional member was rejected: %v", err)
	}
}

// TestContentRestrictsChoiceNarrowed pins clause 1 across a compositor change:
// one branch of a base <choice> is a restriction of the whole choice, and a
// branch the base does not offer is not.
func TestContentRestrictsChoiceNarrowed(t *testing.T) {
	base := uGroup(t, CompositorChoice, cElem(t, "a", 1, 1), cElem(t, "b", 1, 1))
	if err := cRestricts(t, base, uGroup(t, CompositorSequence, cElem(t, "a", 1, 1))); err != nil {
		t.Fatalf("narrowing a choice to one of its branches was rejected: %v", err)
	}
	err := cRestricts(t, base, uGroup(t, CompositorSequence, cElem(t, "c", 1, 1)))
	expectRule(t, err, ruleDerivationOKRestriction)
}

// TestContentRestrictsElementUnderWildcard pins the element-versus-wildcard
// transition: naming an element the base admits only through a ·wildcard
// particle· is the canonical restriction shape, and it survives clause 2 because
// loc-testSubP's keyword lattice is read as keywordSubsumes documents.
func TestContentRestrictsElementUnderWildcard(t *testing.T) {
	for _, pc := range []ProcessContents{ProcessSkip, ProcessLax, ProcessStrict} {
		t.Run(pc.String(), func(t *testing.T) {
			err := cRestricts(t,
				uGroup(t, CompositorSequence, cAny(t, NamespaceConstraintAny, nil, pc)),
				uGroup(t, CompositorSequence, cElem(t, "e", 1, 1)))
			if err != nil {
				t.Fatalf("naming an element the base's wildcard admits was rejected: %v", err)
			}
		})
	}
}

// TestContentRestrictsElementOutsideWildcard pins clause 1 for the same shape
// when the base's wildcard does NOT admit the name: a ##other wildcard over the
// test namespace excludes exactly the names declared in it.
func TestContentRestrictsElementOutsideWildcard(t *testing.T) {
	err := cRestricts(t,
		uGroup(t, CompositorSequence, cAny(t, NamespaceConstraintNot, []Namespace{NamespaceName(uns)}, ProcessLax)),
		uGroup(t, CompositorSequence, cElem(t, "e", 1, 1)))
	expectRule(t, err, ruleDerivationOKRestriction)
}

// TestContentRestrictsWildcardUnderElementRejected pins the reverse pairing: a
// ·wildcard particle· in the restriction admits an open set of names no single
// Element Declaration in the base covers.
func TestContentRestrictsWildcardUnderElementRejected(t *testing.T) {
	err := cRestricts(t,
		uGroup(t, CompositorSequence, cElem(t, "e", 1, 1)),
		uGroup(t, CompositorSequence, cAny(t, NamespaceConstraintAny, nil, ProcessLax)))
	expectRule(t, err, ruleDerivationOKRestriction)
}

// TestContentRestrictsWildcardNarrowed pins the wildcard-versus-wildcard
// transition, which is cos-ns-subset: an enumeration wildcard restricts an any
// wildcard, and an any wildcard does not restrict an enumeration one.
func TestContentRestrictsWildcardNarrowed(t *testing.T) {
	anyW := uGroup(t, CompositorSequence, cAny(t, NamespaceConstraintAny, nil, ProcessLax))
	oneNS := uGroup(t, CompositorSequence, cAny(t, NamespaceConstraintEnumeration, []Namespace{NamespaceName(uns)}, ProcessLax))
	if err := cRestricts(t, anyW, oneNS); err != nil {
		t.Fatalf("narrowing ##any to one namespace was rejected: %v", err)
	}
	expectRule(t, cRestricts(t, oneNS, anyW), ruleDerivationOKRestriction)
}

// TestContentRestrictsProcessContentsSubsumption pins loc-testSubP clauses 1-2
// through cos-content-act-restrict clause 2 (ctr-child-type-subsumption): a lax
// base wildcard does not subsume a skip restriction wildcard, while a skip base
// subsumes anything. The transition itself is compatible in both directions —
// the two wildcards have identical {namespace constraint}s — so only clause 2
// can be deciding the verdict.
func TestContentRestrictsProcessContentsSubsumption(t *testing.T) {
	for _, tc := range []struct {
		name           string
		general        ProcessContents
		specific       ProcessContents
		wantRestricted bool
	}{
		{name: "skip over lax", general: ProcessSkip, specific: ProcessLax, wantRestricted: true},
		{name: "lax over strict", general: ProcessLax, specific: ProcessStrict, wantRestricted: true},
		{name: "lax over skip", general: ProcessLax, specific: ProcessSkip, wantRestricted: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := cRestricts(t,
				uGroup(t, CompositorSequence, cAny(t, NamespaceConstraintAny, nil, tc.general)),
				uGroup(t, CompositorSequence, cAny(t, NamespaceConstraintAny, nil, tc.specific)))
			if tc.wantRestricted && err != nil {
				t.Fatalf("a subsuming ·default binding· was rejected: %v", err)
			}
			if !tc.wantRestricted {
				expectRule(t, err, ruleDerivationOKRestriction)
			}
		})
	}
}

// TestContentRestrictsWildcardUnion pins coveringWildcardUnion: a base that
// splits every expanded name across two non-overlapping ·wildcard particles·
// (##local beside not-##local, which cos-nonambig leaves both live because they
// do not ·overlap·) admits a restriction carrying one ##any wildcard, even
// though neither base wildcard alone is a cos-ns-subset superset of it. The
// single-wildcard control alongside is what keeps the exact relation exact.
func TestContentRestrictsWildcardUnion(t *testing.T) {
	local := []Namespace{NamespaceName("")}
	optional := func(w Wildcard) Particle {
		return uParticle(t, uOccurs(t, 0, 1), ResolvedTerm{Term: w})
	}
	split := uGroup(t, CompositorSequence,
		optional(uWildcard(t, NamespaceConstraintEnumeration, local, ProcessSkip)),
		optional(uWildcard(t, NamespaceConstraintNot, local, ProcessSkip)))
	anyOne := uGroup(t, CompositorSequence, cAny(t, NamespaceConstraintAny, nil, ProcessSkip))
	if err := cRestricts(t, split, anyOne); err != nil {
		t.Fatalf("a wildcard covered by the union of two base wildcards was rejected: %v", err)
	}
	halfOnly := uGroup(t, CompositorSequence,
		optional(uWildcard(t, NamespaceConstraintEnumeration, local, ProcessSkip)))
	expectRule(t, cRestricts(t, halfOnly, anyOne), ruleDerivationOKRestriction)
}

// cNillableElem is a once-occurring particle over a local element declaration
// with the given {nillable}.
func cNillableElem(t *testing.T, local string, nillable bool) Particle {
	t.Helper()
	e, err := NewElementDeclaration(xsderr.Loc{}, uq(local), uq("T"), nil, ScopeLocal, nil, nillable,
		nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	return uOne(t, ResolvedTerm{Term: e})
}

// TestContentRestrictsNillableSubsumption pins loc-testSubP clause 4.1, the
// sub-clause that only the ELEMENT half of ·subsumes· can reach: a restriction
// may keep or drop {nillable}, but may not introduce it where the base's
// declaration is not nillable.
func TestContentRestrictsNillableSubsumption(t *testing.T) {
	for _, tc := range []struct {
		name              string
		general, specific bool
		wantRestricted    bool
	}{
		{name: "both plain", wantRestricted: true},
		{name: "base nillable, restriction not", general: true, wantRestricted: true},
		{name: "both nillable", general: true, specific: true, wantRestricted: true},
		{name: "restriction adds nillable", specific: true, wantRestricted: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := cRestricts(t,
				uGroup(t, CompositorSequence, cNillableElem(t, "e", tc.general)),
				uGroup(t, CompositorSequence, cNillableElem(t, "e", tc.specific)))
			if tc.wantRestricted && err != nil {
				t.Fatalf("a valid restriction was rejected by loc-testSubP clause 4.1: %v", err)
			}
			if !tc.wantRestricted {
				expectRule(t, err, ruleDerivationOKRestriction)
			}
		})
	}
}

// TestContentRestrictsAllGroupLeniency pins §3.4.6.3's all-group allowance: a
// content model the position automaton models as a star cannot be decided
// exactly, so an ·all· group in the RESTRICTION is provisionally accepted even
// where the same shape under <sequence> is rejected. The two halves must move
// together — the sequence half is what proves the all half is a leniency and not
// a check that happens to pass.
func TestContentRestrictsAllGroupLeniency(t *testing.T) {
	base := uGroup(t, CompositorSequence, cElem(t, "a", 1, 1))
	widened := []Particle{cElem(t, "a", 1, 1), cElem(t, "b", 1, 1)}
	if err := cRestricts(t, base, uGroup(t, CompositorAll, widened...)); err != nil {
		t.Fatalf("an ·all· restriction was declined, but §3.4.6.3 licenses provisional acceptance: %v", err)
	}
	expectRule(t, cRestricts(t, base, uGroup(t, CompositorSequence, widened...)), ruleDerivationOKRestriction)
}

// TestContentRestrictsAllGroupBaseDecided is the other side of that leniency: an
// ·all· group in the BASE needs none, because the same star model makes the base
// look LARGER, which can only accept. The restriction is therefore still decided
// — and here declined, since <b> is not among the base's members at all.
func TestContentRestrictsAllGroupBaseDecided(t *testing.T) {
	base := uGroup(t, CompositorAll, cElem(t, "a", 1, 1))
	if err := cRestricts(t, base, uGroup(t, CompositorSequence, cElem(t, "a", 1, 1))); err != nil {
		t.Fatalf("restricting an ·all· base to one of its members was rejected: %v", err)
	}
	expectRule(t, cRestricts(t, base, uGroup(t, CompositorSequence, cElem(t, "b", 1, 1))), ruleDerivationOKRestriction)
}

// cNC builds a Namespace Constraint for the wildcardSubset table.
func cNC(t *testing.T, variety NamespaceConstraintVariety, namespaces []Namespace, disallowed []QName, keywords []DisallowedNameKeyword) NamespaceConstraint {
	t.Helper()
	nc, err := NewNamespaceConstraint(xsderr.Loc{}, variety, namespaces, disallowed, keywords)
	if err != nil {
		t.Fatalf("NewNamespaceConstraint: %v", err)
	}
	return nc
}

// TestWildcardSubset pins cos-ns-subset (§3.10.6.2) directly: its four
// variety/namespaces cases AND the conjunctive {disallowed names} tail, which is
// the half easiest to drop as though it were optional.
func TestWildcardSubset(t *testing.T) {
	a := NamespaceName("urn:a")
	b := NamespaceName("urn:b")
	for _, tc := range []struct {
		name       string
		sub, super NamespaceConstraint
		want       bool
	}{
		{name: "clause 1 super any", sub: cNC(t, NamespaceConstraintNot, []Namespace{a}, nil, nil),
			super: cNC(t, NamespaceConstraintAny, nil, nil, nil), want: true},
		{name: "clause 2 enumeration subset", sub: cNC(t, NamespaceConstraintEnumeration, []Namespace{a}, nil, nil),
			super: cNC(t, NamespaceConstraintEnumeration, []Namespace{a, b}, nil, nil), want: true},
		{name: "clause 2 enumeration not a subset", sub: cNC(t, NamespaceConstraintEnumeration, []Namespace{a, b}, nil, nil),
			super: cNC(t, NamespaceConstraintEnumeration, []Namespace{a}, nil, nil), want: false},
		{name: "clause 3 disjoint", sub: cNC(t, NamespaceConstraintEnumeration, []Namespace{a}, nil, nil),
			super: cNC(t, NamespaceConstraintNot, []Namespace{b}, nil, nil), want: true},
		{name: "clause 3 overlapping", sub: cNC(t, NamespaceConstraintEnumeration, []Namespace{a}, nil, nil),
			super: cNC(t, NamespaceConstraintNot, []Namespace{a}, nil, nil), want: false},
		{name: "clause 4 wider not", sub: cNC(t, NamespaceConstraintNot, []Namespace{a, b}, nil, nil),
			super: cNC(t, NamespaceConstraintNot, []Namespace{a}, nil, nil), want: true},
		{name: "clause 4 narrower not", sub: cNC(t, NamespaceConstraintNot, []Namespace{a}, nil, nil),
			super: cNC(t, NamespaceConstraintNot, []Namespace{a, b}, nil, nil), want: false},
		{name: "any under enumeration", sub: cNC(t, NamespaceConstraintAny, nil, nil, nil),
			super: cNC(t, NamespaceConstraintEnumeration, []Namespace{a}, nil, nil), want: false},
		{name: "tail 1 super disallows a name sub allows",
			sub:   cNC(t, NamespaceConstraintAny, nil, nil, nil),
			super: cNC(t, NamespaceConstraintAny, nil, []QName{{Space: "urn:a", Local: "x"}}, nil), want: false},
		{name: "tail 1 sub disallows it too",
			sub:   cNC(t, NamespaceConstraintAny, nil, []QName{{Space: "urn:a", Local: "x"}}, nil),
			super: cNC(t, NamespaceConstraintAny, nil, []QName{{Space: "urn:a", Local: "x"}}, nil), want: true},
		{name: "tail 2 defined not carried down",
			sub:   cNC(t, NamespaceConstraintAny, nil, nil, nil),
			super: cNC(t, NamespaceConstraintAny, nil, nil, []DisallowedNameKeyword{DisallowedNameDefined}), want: false},
		{name: "tail 3 sibling not carried down",
			sub:   cNC(t, NamespaceConstraintAny, nil, nil, []DisallowedNameKeyword{DisallowedNameDefined}),
			super: cNC(t, NamespaceConstraintAny, nil, nil, []DisallowedNameKeyword{DisallowedNameDefined, DisallowedNameSibling}), want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := wildcardSubset(tc.sub, tc.super); got != tc.want {
				t.Fatalf("wildcardSubset = %t, want %t", got, tc.want)
			}
		})
	}
}
