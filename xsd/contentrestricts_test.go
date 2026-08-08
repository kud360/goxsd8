package xsd

import (
	"testing"
	"time"

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

// cUnbounded is a particle over a local element declaration of the given name,
// typed T, whose {max occurs} is unbounded.
func cUnbounded(t *testing.T, local string, minOccurs int) Particle {
	t.Helper()
	return uParticle(t, uUnbounded(t, minOccurs), ResolvedTerm{Term: uLocal(t, uq(local), uq("T"))})
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
// though neither base wildcard alone is a cos-ns-subset superset of it. This is
// W3C saxonData/Wild wild049's shape, and cos-aw-union (§3.10.6.3) case 5.1
// decides it: the difference of the not set and the enumeration set is empty, so
// the union is ##any, of which the restriction's ##any is a ·wildcard subset·.
// The single-wildcard control alongside is what keeps the exact relation exact.
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

// TestContentRestrictsWildcardUnionShortfall is the other side of
// coveringWildcardUnion, and the half a union that is merely ASSUMED to cover
// cannot decide: two live base wildcards whose cos-aw-union is a bounded
// enumeration (case 3, ##local beside one other namespace) do not admit a ##any
// restriction, so the walk reports the empty matched set and clause 1 rejects.
// The paired acceptance narrows the restriction to one of the two enumerated
// namespaces, which the same union does cover — without it, a coveringWildcardUnion
// that simply returned nil whenever two wildcards were live would pass this test.
func TestContentRestrictsWildcardUnionShortfall(t *testing.T) {
	local := []Namespace{NamespaceName("")}
	other := []Namespace{NamespaceName(uns)}
	optional := func(w Wildcard) Particle {
		return uParticle(t, uOccurs(t, 0, 1), ResolvedTerm{Term: w})
	}
	split := uGroup(t, CompositorSequence,
		optional(uWildcard(t, NamespaceConstraintEnumeration, local, ProcessSkip)),
		optional(uWildcard(t, NamespaceConstraintEnumeration, other, ProcessSkip)))
	anyOne := uGroup(t, CompositorSequence, cAny(t, NamespaceConstraintAny, nil, ProcessSkip))
	expectRule(t, cRestricts(t, split, anyOne), ruleDerivationOKRestriction)
	within := uGroup(t, CompositorSequence, cAny(t, NamespaceConstraintEnumeration, other, ProcessSkip))
	if err := cRestricts(t, split, within); err != nil {
		t.Fatalf("a wildcard inside the union of two base wildcards was rejected: %v", err)
	}
}

// cNillableElem is a once-occurring particle over a local element declaration
// with the given {nillable}.
func cNillableElem(t *testing.T, local string, nillable bool) Particle {
	t.Helper()
	e, err := NewElementDeclaration(xsderr.Loc{}, uq(local), TypeDefinitionRef{Name: uq("T")}, nil, uLocalScope(t), nil, nillable,
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

// TestContentRestrictsSubsetConstruction pins the SUBSET construction itself:
// contentModelRestricts must carry EVERY matched B-position forward, not one of
// them. The base offers <e><a> or <any><b>; cos-nonambig leaves the ·element
// particle· <e> and the ·wildcard particle· live together at the start, since 1.1
// no longer forbids that pairing, so an <e> in the restriction matches BOTH. Only
// the wildcard's branch continues into <b>, and it is the higher-indexed member —
// a walk that committed to the first match would follow <e> into <a> alone and
// false-reject a restriction the base plainly admits ("e b" is the second branch
// with the wildcard taking e).
func TestContentRestrictsSubsetConstruction(t *testing.T) {
	base := uGroup(t, CompositorChoice,
		uOne(t, ResolvedTerm{Term: uGroup(t, CompositorSequence, cElem(t, "e", 1, 1), cElem(t, "a", 1, 1))}),
		uOne(t, ResolvedTerm{Term: uGroup(t, CompositorSequence, cAny(t, NamespaceConstraintAny, nil, ProcessSkip), cElem(t, "b", 1, 1))}))
	if err := cRestricts(t, base, uGroup(t, CompositorSequence, cElem(t, "e", 1, 1), cElem(t, "b", 1, 1))); err != nil {
		t.Fatalf("a continuation reachable only through a later matched B-position was rejected: %v", err)
	}
	// The control: <c> is admitted by the wildcard too, but neither branch
	// continues into <a> after it, so the same walk still decides a rejection.
	expectRule(t, cRestricts(t, base,
		uGroup(t, CompositorSequence, cElem(t, "c", 1, 1), cElem(t, "a", 1, 1))), ruleDerivationOKRestriction)
}

// TestContentRestrictsMatchedSetMixedTerms pins clause 2 as an EXISTENTIAL over
// the matched set. The base's start state holds an ·element particle· <e> that is
// not {nillable} beside a skip ·wildcard particle·, and both admit <e>; the
// restriction names <e> as {nillable}, which loc-testSubP clause 4.1 refuses
// against the element particle's ·default binding· while the skip keyword subsumes
// anything (clause 1). The element particle is the FIRST member of the matched
// set, so reading any one member — rather than asking whether SOME member
// subsumes — decides the opposite verdict.
//
// The acceptance is the fail-open reading someBindingSubsumes documents, not the
// spec's own: ·attribution· would give the item the Element Declaration.
func TestContentRestrictsMatchedSetMixedTerms(t *testing.T) {
	base := uGroup(t, CompositorSequence,
		cElem(t, "e", 0, 1),
		uParticle(t, uOccurs(t, 0, 1), ResolvedTerm{Term: uWildcard(t, NamespaceConstraintAny, nil, ProcessSkip)}))
	derived := uGroup(t, CompositorSequence, cNillableElem(t, "e", true))
	if err := cRestricts(t, base, derived); err != nil {
		t.Fatalf("clause 2 must be decided existentially over the matched set: %v", err)
	}
	// The control: with the wildcard gone the matched set holds the element
	// particle alone, and clause 4.1 is charged exactly as before.
	expectRule(t, cRestricts(t, uGroup(t, CompositorSequence, cElem(t, "e", 0, 1)), derived), ruleDerivationOKRestriction)
}

// cGlobalRefModel is a one-particle content model over an <element ref> to a
// top-level declaration.
func cGlobalRefModel(t *testing.T, name string) ModelGroup {
	t.Helper()
	return uGroup(t, CompositorSequence, uOne(t, ElementDeclarationRef{Name: uq(name)}))
}

// TestContentRestrictsGlobalSubstitution pins elementParticleAdmits now that it
// carries no approximation: a base <element ref="head"/> restricted to
// <element ref="member"/> is admitted exactly when member is in head's
// ·substitution group·, and rejected when it is not.
//
// It was TestContentRestrictsGlobalSubstitutionGap, which pinned the opposite —
// the fail-open arm admitting ANY two top-level declarations, needed while no
// producer mapped substitutionGroup= into {substitution group affiliations} (a
// universally-empty set makes every real member answer "not in the group", which
// false-rejects). The producer now maps it (#281), so the case that used to be
// the fail-open's whole justification is decided on real data instead, and the
// unaffiliated pairing that used to slip through is charged.
//
// The LOCAL half is retained: a local declaration is in no substitution group at
// all (e-props-correct clause 3 confines affiliations to a global {scope}), so it
// is admitted only under cos-equiv-derived-ok-rec clause 1, expanded-name
// equality — which is what makes the removal of the global arm a real narrowing
// rather than a relabelling. Its particle is named for a declaration the schema
// does not declare globally, so the membership walk's own name lookup misses and
// the LOCAL reading is what is under test; a local particle sharing a global
// declaration's expanded name would be answered from that global one, since
// inSubstitutionGroupOf takes names rather than components (substitutiongroup.go).
func TestContentRestrictsGlobalSubstitution(t *testing.T) {
	globalRestriction := func(memberAffiliations []QName, derived ModelGroup) error {
		return dFinalize(t, func(b *SchemaBuilder) {
			b.AddElement(uGlobal(t, uq("head"), uq("T")))
			b.AddElement(uGlobal(t, uq("member"), uq("T"), memberAffiliations...))
			b.AddType(dType(t, uq("base"), anyTypeName, dElementContent(t, false, cGlobalRefModel(t, "head")), nil, nil))
			b.AddType(dType(t, uq("derived"), uq("base"), dElementContent(t, false, derived), nil, nil))
		})
	}
	affiliated := []QName{uq("head")}
	if err := globalRestriction(affiliated, cGlobalRefModel(t, "member")); err != nil {
		t.Fatalf("member is in head's substitution group, so the element particle pairing must be admitted: %v", err)
	}
	// Same pairing, affiliation removed: no chain reaches head, and the arm that
	// used to admit it unconditionally is gone.
	expectRule(t, globalRestriction(nil, cGlobalRefModel(t, "member")), ruleDerivationOKRestriction)
	// A LOCAL declaration joins no substitution group, so a differently-named
	// local particle is not admitted against the base's <element ref="head"/>.
	expectRule(t, globalRestriction(affiliated, uGroup(t, CompositorSequence, cElem(t, "local", 1, 1))), ruleDerivationOKRestriction)
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

// TestContentRestrictsDeclaredOccurrenceBounds pins cos-content-act-restrict
// clause 1 over occurrence ranges no fixed unfolding bound can decide. The class
// this test guards is not the five rows but the defect they witness: an
// unfolding that truncates a range REWRITES it, and the rewrite is monotone in
// neither direction — {3,6} truncated to two copies of each kind reads as {2,4},
// which adds 2 and drops 5 and 6 — so a truncating walk both rejects valid
// restrictions and accepts invalid ones, at thresholds that move but never
// vanish when the constant is raised. The walk therefore decides over the
// declared {min occurs}/{max occurs} (unfoldExactly, #501).
//
// Rows 1-2 are the false rejects: R's range is a subset of B's, sharing B's
// maximum in row 2, and both were charged derivation-ok-restriction. Rows 3-5
// are the false accepts: R demands strictly more occurrences than B permits, and
// both sides collapsed to the same two mandatory copies. Every row is a bare
// single element particle with no group nesting, which is what makes the class
// reachable by an ordinary schema.
//
// Rows 6-8 carry the class past the shapes the constants happen to sit at: a
// range wider than any small bound on both sides, an unbounded base (which must
// contain every bounded restriction of it), and an unbounded restriction under a
// bounded base (which no base range can contain).
func TestContentRestrictsDeclaredOccurrenceBounds(t *testing.T) {
	for _, tc := range []struct {
		name           string
		base, derived  Particle
		wantRestricted bool
	}{
		{name: "subset range under a much wider base", base: cElem(t, "e", 0, 100), derived: cElem(t, "e", 3, 6), wantRestricted: true},
		{name: "subset range sharing the base maximum", base: cElem(t, "e", 0, 6), derived: cElem(t, "e", 3, 6), wantRestricted: true},
		{name: "fixed count well above a fixed base", base: cElem(t, "e", 3, 3), derived: cElem(t, "e", 5, 5), wantRestricted: false},
		{name: "fixed count one above a fixed base", base: cElem(t, "e", 3, 3), derived: cElem(t, "e", 4, 4), wantRestricted: false},
		{name: "fixed count above a smaller fixed base", base: cElem(t, "e", 2, 2), derived: cElem(t, "e", 3, 3), wantRestricted: false},
		{name: "wide range one past a wide base", base: cElem(t, "e", 10, 40), derived: cElem(t, "e", 10, 41), wantRestricted: false},
		{name: "bounded range under an unbounded base", base: cUnbounded(t, "e", 0), derived: cElem(t, "e", 7, 9), wantRestricted: true},
		{name: "unbounded range under a bounded base", base: cElem(t, "e", 0, 9), derived: cUnbounded(t, "e", 0), wantRestricted: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := cRestricts(t,
				uGroup(t, CompositorSequence, tc.base),
				uGroup(t, CompositorSequence, tc.derived))
			if tc.wantRestricted && err != nil {
				t.Fatalf("a valid content-model restriction was rejected: %v", err)
			}
			if !tc.wantRestricted {
				expectRule(t, err, ruleDerivationOKRestriction)
			}
		})
	}
}

// TestContentRestrictsOccurrenceBoundsNested is the same class one level down: a
// numeric occurrence range on a <sequence> particle, not on an element particle,
// so the copies the unfolding emits are whole group fragments. The verdicts are
// the containment verdicts — (a, b){3,6} is a subset of (a, b){0,100} and
// (a, b){5,5} is not a subset of (a, b){3,3} — and nothing about them depends on
// the members being leaves.
func TestContentRestrictsOccurrenceBoundsNested(t *testing.T) {
	pair := func(minOccurs, maxOccurs int) ModelGroup {
		inner := uGroup(t, CompositorSequence, cElem(t, "a", 1, 1), cElem(t, "b", 1, 1))
		return uGroup(t, CompositorSequence, uParticle(t, uOccurs(t, minOccurs, maxOccurs), ResolvedTerm{Term: inner}))
	}
	if err := cRestricts(t, pair(0, 100), pair(3, 6)); err != nil {
		t.Fatalf("a valid restriction of a repeated group was rejected: %v", err)
	}
	expectRule(t, cRestricts(t, pair(3, 3), pair(5, 5)), ruleDerivationOKRestriction)
}

// TestContentRestrictsBeyondPositionCeiling pins the direction of the
// maxContentPositions giveup: a content model whose exact unfolding does not fit
// is left undecided and provisionally ACCEPTED, never rejected. Both pairs here
// are past the ceiling, and the invalid one is accepted for that reason — the
// fail-open the GAP marker at the giveup site records. What the test guards is
// that the ceiling never manufactures a rejection, which is the failure mode the
// truncating unfolding it replaced actually had.
func TestContentRestrictsBeyondPositionCeiling(t *testing.T) {
	huge := maxContentPositions + 1
	if err := cRestricts(t,
		uGroup(t, CompositorSequence, cElem(t, "e", 0, huge)),
		uGroup(t, CompositorSequence, cElem(t, "e", 3, 6))); err != nil {
		t.Fatalf("a valid restriction of an unmaterializable base was rejected: %v", err)
	}
	if err := cRestricts(t,
		uGroup(t, CompositorSequence, cElem(t, "e", huge, huge)),
		uGroup(t, CompositorSequence, cElem(t, "e", huge+1, huge+1))); err != nil {
		t.Fatalf("an undecidable derivation was rejected rather than provisionally accepted: %v", err)
	}
}

// TestContentRestrictsWideRangeStaysCheap pins the COST of the containment walk,
// which the exact unfolding made a load-bearing property rather than an
// incidental one: unfoldExactly emits {max occurs} positions, so a range an
// ordinary schema may carry — maxOccurs="300" narrowed to maxOccurs="150" — puts
// hundreds of positions into both automata, and every per-copy quantity the walk
// touches per transition multiplies out from there. The first exact unfolding
// paid that per COPY and took 15.2 s on the identical-range row below (8.3 s at
// width 200, 2 m 25 s at 1024); the walk now decides one transition per SOURCE
// PARTICLE live in a state, and per distinct B-subset rather than per product
// state, which is what these rows guard (#501).
//
// The assertion is a wall clock with two orders of magnitude of slack, not a
// benchmark: the rows measured ~100 ms and ~40 ms on the machine whose numbers
// maxContentPositions records, against a 5 s budget, so a machine fifty times
// slower still passes while the defect they pin — which was 150x and 85x over
// its row — cannot. The verdicts are asserted too, since a walk that got fast by
// deciding something else is not the thing being kept.
func TestContentRestrictsWideRangeStaysCheap(t *testing.T) {
	const budget = 5 * time.Second
	for _, tc := range []struct {
		name       string
		bMax, rMax int
	}{
		{name: "identical wide ranges", bMax: 300, rMax: 300},
		{name: "wide range narrowed", bMax: 200, rMax: 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			err := cRestricts(t,
				uGroup(t, CompositorSequence, cElem(t, "e", 0, tc.bMax)),
				uGroup(t, CompositorSequence, cElem(t, "e", 0, tc.rMax)))
			elapsed := time.Since(start)
			if err != nil {
				t.Fatalf("e{0,%d} under e{0,%d} is a valid restriction and was rejected: %v", tc.rMax, tc.bMax, err)
			}
			if elapsed > budget {
				t.Fatalf("deciding e{0,%d} under e{0,%d} took %v, over the %v budget: the containment walk is doing work per unfolded copy again",
					tc.rMax, tc.bMax, elapsed, budget)
			}
		})
	}
}
