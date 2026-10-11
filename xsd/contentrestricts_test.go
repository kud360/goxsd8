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

// cAny is a particle over a wildcard with the given {namespace constraint} —
// its {disallowed names} holding keywords and no QName — and {process contents},
// occurring exactly once.
func cAny(t *testing.T, variety NamespaceConstraintVariety, namespaces []Namespace, pc ProcessContents, keywords ...DisallowedNameKeyword) Particle {
	t.Helper()
	w, err := NewWildcard(xsderr.Loc{}, cNC(t, variety, namespaces, nil, keywords), pc)
	if err != nil {
		t.Fatalf("NewWildcard: %v", err)
	}
	return uOne(t, ResolvedTerm{Term: w})
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

// TestContentRestrictsProcessContentsSubsumption pins loc-testSubP clauses 1-3
// through cos-content-act-restrict clause 2 (ctr-child-type-subsumption): a lax
// base wildcard does not subsume a skip restriction wildcard, a strict one
// subsumes neither a skip one nor a lax one, with or without defined in its
// {disallowed names}, while a skip base subsumes anything and a lax one
// anything but skip. The transition itself is compatible in every row — the
// specific wildcard's {namespace constraint} is the general's, plus at most
// the defined keyword, which cos-ns-subset lets a subset add — so only
// cos-content-act-restrict clause 2 can be deciding the verdict. The
// strict-over-skip row is W3C suite wildZ008's shape, the strict-over-lax row
// MS-Errata10 errC008's, and the lax-over-strict row MS-Wildcards wildZ009b's.
func TestContentRestrictsProcessContentsSubsumption(t *testing.T) {
	defined := []DisallowedNameKeyword{DisallowedNameDefined}
	for _, tc := range []struct {
		name             string
		general          ProcessContents
		specific         ProcessContents
		specificKeywords []DisallowedNameKeyword
		wantRestricted   bool
	}{
		{name: "skip over lax", general: ProcessSkip, specific: ProcessLax, wantRestricted: true},
		{name: "skip over strict", general: ProcessSkip, specific: ProcessStrict, wantRestricted: true},
		{name: "lax over strict", general: ProcessLax, specific: ProcessStrict, wantRestricted: true},
		{name: "lax over lax", general: ProcessLax, specific: ProcessLax, wantRestricted: true},
		{name: "lax over skip", general: ProcessLax, specific: ProcessSkip, wantRestricted: false},
		{name: "strict over strict", general: ProcessStrict, specific: ProcessStrict, wantRestricted: true},
		{name: "strict over skip", general: ProcessStrict, specific: ProcessSkip, wantRestricted: false},
		{name: "strict over ##defined lax", general: ProcessStrict, specific: ProcessLax, specificKeywords: defined, wantRestricted: false},
		{name: "strict over lax", general: ProcessStrict, specific: ProcessLax, wantRestricted: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := cRestricts(t,
				uGroup(t, CompositorSequence, cAny(t, NamespaceConstraintAny, nil, tc.general)),
				uGroup(t, CompositorSequence, cAny(t, NamespaceConstraintAny, nil, tc.specific, tc.specificKeywords...)))
			if tc.wantRestricted && err != nil {
				t.Fatalf("a subsuming ·default binding· was rejected: %v", err)
			}
			if !tc.wantRestricted {
				expectRule(t, err, ruleDerivationOKRestriction)
			}
		})
	}
}

// TestContentRestrictsStrictOverLaxNamespaces pins the W3C suite MS-Particles
// particlesOb001-Ob009 shapes #2546 charges: a strict ##any base wildcard
// restricted by a lax one, in a choice, in each namespace form those schemas
// spell and with ranges clause 1 accepts. Every such R wildcard admits a name
// no declaration governs, bound strict under B and lax under R, and
// loc-testSubP has no clause for that pairing (ctr-child-type-subsumption).
// Two twins per row guard against charging the shape for anything but that
// pairing: R's wildcard made strict, and B's made lax, are both accepted.
func TestContentRestrictsStrictOverLaxNamespaces(t *testing.T) {
	absent := NamespaceName("")
	wild := func(o Occurs, variety NamespaceConstraintVariety, namespaces []Namespace, pc ProcessContents) ModelGroup {
		return uGroup(t, CompositorChoice, uParticle(t, o, ResolvedTerm{Term: uWildcard(t, variety, namespaces, pc)}))
	}
	for _, tc := range []struct {
		name             string
		bOccurs, rOccurs Occurs
		variety          NamespaceConstraintVariety
		namespaces       []Namespace
	}{
		{name: "Ob001 ##any", bOccurs: uOccurs(t, 1, 1), rOccurs: uOccurs(t, 1, 1), variety: NamespaceConstraintAny},
		{name: "Ob004 ##local, 1..1 under 0..2", bOccurs: uOccurs(t, 0, 2), rOccurs: uOccurs(t, 1, 1),
			variety: NamespaceConstraintEnumeration, namespaces: []Namespace{absent}},
		{name: "Ob008 target and local, 2..3 under 1..5", bOccurs: uOccurs(t, 1, 5), rOccurs: uOccurs(t, 2, 3),
			variety: NamespaceConstraintEnumeration, namespaces: []Namespace{NamespaceName(uns), absent}},
		{name: "Ob009 a URI list, unbounded", bOccurs: uUnbounded(t, 1), rOccurs: uUnbounded(t, 1),
			variety: NamespaceConstraintEnumeration, namespaces: []Namespace{NamespaceName("foo"), absent, NamespaceName("bar")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			strictBase := wild(tc.bOccurs, NamespaceConstraintAny, nil, ProcessStrict)
			expectRule(t, cRestricts(t, strictBase, wild(tc.rOccurs, tc.variety, tc.namespaces, ProcessLax)), ruleDerivationOKRestriction)
			if err := cRestricts(t, strictBase, wild(tc.rOccurs, tc.variety, tc.namespaces, ProcessStrict)); err != nil {
				t.Fatalf("strict over strict with the same ranges and namespaces was rejected: %v", err)
			}
			laxBase := wild(tc.bOccurs, NamespaceConstraintAny, nil, ProcessLax)
			if err := cRestricts(t, laxBase, wild(tc.rOccurs, tc.variety, tc.namespaces, ProcessLax)); err != nil {
				t.Fatalf("lax over lax with the same ranges and namespaces was rejected: %v", err)
			}
		})
	}
}

// TestContentRestrictsEmptyWildcard pins contentModelRestricts' skip of an R
// wildcard admitting no namespace (namespace=""): no item is ·attributed· to
// it, so it adds nothing to either clause of cos-content-act-restrict. Under a
// strict base wildcard an optional empty one is accepted at every non-strict
// {process contents}, where clause 2 would otherwise refuse the keyword, and
// under a base holding one optional element particle it is accepted too, where
// clause 1 would otherwise find no base particle admitting it. Each row's
// control gives the same wildcard the absent namespace and is rejected.
func TestContentRestrictsEmptyWildcard(t *testing.T) {
	optionalAny := func(namespaces []Namespace, pc ProcessContents) Particle {
		return uParticle(t, uOccurs(t, 0, 1), ResolvedTerm{Term: uWildcard(t, NamespaceConstraintEnumeration, namespaces, pc)})
	}
	strictBase := uGroup(t, CompositorSequence,
		uParticle(t, uOccurs(t, 0, 1), ResolvedTerm{Term: uWildcard(t, NamespaceConstraintAny, nil, ProcessStrict)}))
	elementBase := uGroup(t, CompositorSequence, cElem(t, "a", 0, 1))
	for _, tc := range []struct {
		name string
		base ModelGroup
		pc   ProcessContents
	}{
		{name: "lax under a strict wildcard", base: strictBase, pc: ProcessLax},
		{name: "skip under a strict wildcard", base: strictBase, pc: ProcessSkip},
		{name: "lax under an element particle", base: elementBase, pc: ProcessLax},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := cRestricts(t, tc.base, uGroup(t, CompositorSequence, optionalAny(nil, tc.pc))); err != nil {
				t.Fatalf("a wildcard admitting no name was charged: %v", err)
			}
			local := uGroup(t, CompositorSequence, optionalAny([]Namespace{NamespaceName("")}, tc.pc))
			expectRule(t, cRestricts(t, tc.base, local), ruleDerivationOKRestriction)
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

// cAnyExcept is a once-occurring wildcard over the given namespaces whose
// {disallowed names} holds the given QNames and no keyword.
func cAnyExcept(t *testing.T, namespaces []Namespace, pc ProcessContents, disallowed ...QName) Particle {
	t.Helper()
	w, err := NewWildcard(xsderr.Loc{}, cNC(t, NamespaceConstraintEnumeration, namespaces, disallowed, nil), pc)
	if err != nil {
		t.Fatalf("NewWildcard: %v", err)
	}
	return uOne(t, ResolvedTerm{Term: w})
}

// TestContentRestrictsElementCoveredWildcard pins cos-content-act-restrict
// clause 1 where R's wildcard is covered only by B's live ·element particles·
// and live ·wildcard particles· together: the names B's wildcards leave out
// (here by notQName) are exactly the names a live element particle admits.
// Clause 1 is language containment, so the pair satisfies it whichever particle
// admits each item, and src-redefine clause 6.2.2 (restrictsLanguage), which
// charges clause 1 alone, accepts it at every {process contents}. The two
// element-removed rows are the rejecting siblings: without the element
// particle the name a is admitted by nothing in B.
//
// The "a must be followed by b" rows check that the name split off for the
// element particle continues into that particle's ·follow· set: R's single
// item a is a sequence B does not accept when B's element a requires b after
// it, and is one B accepts when that b is optional. In "a also admitted by a
// wildcard" B's element a is followed by b and its urn:upa wildcard by c, and R
// follows every item with c, so R's item a is accepted only through the
// wildcard: the set split off for a must hold every live position admitting a,
// the wildcard's as well as the element's.
func TestContentRestrictsElementCoveredWildcard(t *testing.T) {
	a := uq("a")
	ns := []Namespace{NamespaceName(uns)}
	both := []Namespace{NamespaceName(uns), NamespaceName("")}
	local := []Namespace{NamespaceName("")}
	choice := func(ps ...Particle) Particle {
		return uOne(t, ResolvedTerm{Term: uGroup(t, CompositorChoice, ps...)})
	}
	seq := func(ps ...Particle) ModelGroup { return uGroup(t, CompositorSequence, ps...) }
	for _, pc := range []ProcessContents{ProcessSkip, ProcessLax, ProcessStrict} {
		t.Run(pc.String(), func(t *testing.T) {
			for _, tc := range []struct {
				name          string
				base, derived ModelGroup
				wantSubset    bool
			}{
				{name: "one wildcard beside the element",
					base:    seq(choice(cElem(t, "a", 1, 1), cAnyExcept(t, ns, pc, a))),
					derived: seq(cAnyExcept(t, ns, pc)), wantSubset: true},
				{name: "one wildcard, element removed",
					base:    seq(cAnyExcept(t, ns, pc, a)),
					derived: seq(cAnyExcept(t, ns, pc))},
				{name: "two wildcards beside the element",
					base:    seq(choice(cElem(t, "a", 1, 1), cAnyExcept(t, ns, pc, a), cAnyExcept(t, local, pc))),
					derived: seq(cAnyExcept(t, both, pc)), wantSubset: true},
				{name: "two wildcards, element removed",
					base:    seq(choice(cAnyExcept(t, ns, pc, a), cAnyExcept(t, local, pc))),
					derived: seq(cAnyExcept(t, both, pc))},
				{name: "a must be followed by b",
					base: seq(choice(
						uOne(t, ResolvedTerm{Term: seq(cElem(t, "a", 1, 1), cElem(t, "b", 1, 1))}),
						cAnyExcept(t, ns, pc, a))),
					derived: seq(cAnyExcept(t, ns, pc))},
				{name: "a may be followed by b",
					base: seq(choice(
						uOne(t, ResolvedTerm{Term: seq(cElem(t, "a", 1, 1), cElem(t, "b", 0, 1))}),
						cAnyExcept(t, ns, pc, a))),
					derived: seq(cAnyExcept(t, ns, pc)), wantSubset: true},
				{name: "a also admitted by a wildcard",
					base: seq(choice(
						uOne(t, ResolvedTerm{Term: seq(cElem(t, "a", 1, 1), cElem(t, "b", 1, 1))}),
						uOne(t, ResolvedTerm{Term: seq(cAnyExcept(t, ns, pc), cElem(t, "c", 1, 1))}),
						uOne(t, ResolvedTerm{Term: seq(cAnyExcept(t, local, pc), cElem(t, "c", 1, 1))}))),
					derived: seq(cAnyExcept(t, both, pc), cElem(t, "c", 1, 1)), wantSubset: true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					err := mgdRedefines(t, tc.base, tc.derived)
					if tc.wantSubset && err != nil {
						t.Fatalf("a redefinition whose wildcard B's element and wildcard particles jointly cover was rejected: %v", err)
					}
					if !tc.wantSubset {
						expectRule(t, err, ruleSrcRedefine)
					}
				})
			}
		})
	}
}

// TestContentRestrictsElementCoveredSubstitution pins the names an ·element
// particle· contributes to TestContentRestrictsElementCoveredWildcard's split:
// its declaration's own name and every member of its ·substitution group·. B's
// <element ref="head"/> admits member only when member is affiliated to head, so
// B's wildcard excluding both names is covered by the element particle in the
// first row and leaves member admitted by nothing in the second.
func TestContentRestrictsElementCoveredSubstitution(t *testing.T) {
	head, member := uq("head"), uq("member")
	ns := []Namespace{NamespaceName(uns)}
	redefines := func(affiliations ...QName) error {
		return dFinalize(t, func(b *SchemaBuilder) {
			b.AddElement(uGlobal(t, head, uq("T")))
			b.AddElement(uGlobal(t, member, uq("T"), affiliations...))
			original := uGroup(t, CompositorSequence, uOne(t, ResolvedTerm{Term: uGroup(t, CompositorChoice,
				uOne(t, ElementDeclarationRef{Name: head}), cAnyExcept(t, ns, ProcessLax, head, member))}))
			redefining := uGroup(t, CompositorSequence, cAnyExcept(t, ns, ProcessLax))
			b.AddRedefiningModelGroup(dMGD(t, uq("g"), redefining), dMGD(t, uq("g"), original))
		})
	}
	if err := redefines(head); err != nil {
		t.Fatalf("a wildcard covered by a substitution group head and a wildcard was rejected: %v", err)
	}
	expectRule(t, redefines(), ruleSrcRedefine)
}

// TestContentRestrictsElementCoveredWildcardBinding pins cos-content-act-restrict
// clause 2 on the item TestContentRestrictsElementCoveredWildcard splits off: an
// item named a that R's wildcard admits and only B's ·element particle· admits.
// Clause 1 holds for every row (the same pairs are accepted by src-redefine
// clause 6.2.2 above), so derivation-ok-restriction's verdict is clause 2's.
//
//   - B's particle is a LOCAL a and no top-level a exists: R's ·default binding·
//     for the item is its wildcard's keyword at every {process contents}
//     (key-governing-ed resolves no declaration), B's is the Element Declaration,
//     and loc-testSubP never lets an Element Declaration subsume a keyword. An
//     exact rejection.
//   - B's particle is <element ref="a"/> to a top-level a: under a skip wildcard
//     R's binding is still the keyword (§3.10.4.1's closing Note), so the pair is
//     an exact rejection too. Under a lax or strict one key-governing-ed clause 3
//     may govern the item by that top-level a, which elementPositionBinding does
//     not render (#345), and the pair is ACCEPTED fail-open, the GAP(xsd) at
//     coveringWildcardUnion's element split. The skip row is its rejecting
//     sibling.
func TestContentRestrictsElementCoveredWildcardBinding(t *testing.T) {
	a := uq("a")
	ns := []Namespace{NamespaceName(uns)}
	choice := func(ps ...Particle) Particle {
		return uOne(t, ResolvedTerm{Term: uGroup(t, CompositorChoice, ps...)})
	}
	seq := func(ps ...Particle) ModelGroup { return uGroup(t, CompositorSequence, ps...) }
	global := func(pc ProcessContents) error {
		return dFinalize(t, func(b *SchemaBuilder) {
			b.AddElement(uGlobal(t, a, uq("T")))
			base := seq(choice(uOne(t, ElementDeclarationRef{Name: a}), cAnyExcept(t, ns, pc, a)))
			b.AddType(dType(t, uq("base"), anyTypeName, dElementContent(t, false, base), nil, nil))
			b.AddType(dType(t, uq("derived"), uq("base"), dElementContent(t, false, seq(cAnyExcept(t, ns, pc))), nil, nil))
		})
	}
	for _, tc := range []struct {
		pc             ProcessContents
		wantRestricted bool
	}{
		{pc: ProcessSkip},
		{pc: ProcessLax, wantRestricted: true},
		{pc: ProcessStrict, wantRestricted: true},
	} {
		t.Run(tc.pc.String(), func(t *testing.T) {
			local := cRestricts(t,
				seq(choice(cElem(t, "a", 1, 1), cAnyExcept(t, ns, tc.pc, a))),
				seq(cAnyExcept(t, ns, tc.pc)))
			expectRule(t, local, ruleDerivationOKRestriction)
			err := global(tc.pc)
			if tc.wantRestricted && err != nil {
				t.Fatalf("an item a top-level declaration may govern was charged clause 2: %v", err)
			}
			if !tc.wantRestricted {
				expectRule(t, err, ruleDerivationOKRestriction)
			}
		})
	}
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

// TestContentRestrictsWildcardUnionCounts pins the per-part split in
// coveringWildcardUnion: an item of a union-covered restriction wildcard
// advances only the base wildcard that admits it, so it counts towards that
// wildcard's occurrence range and no other's.
//
// The rejecting row is W3C saxonData All/all244's shape. B = all(any{one,two}
// 5..unbounded, any{three} 0..2) and R = all(any{one} 3..unbounded,
// any{two,three} 2..2): R admits (one, one, one, three, three), whose three items
// only B's second wildcard admits, which leaves B's first at 3 of its 5. With
// the covering union's every live wildcard advanced on one R transition, the
// three items counted towards B's first wildcard too and the row was accepted.
// The accepting row keeps the union, lowers B's first minimum to 2, which every
// choice of R's covered items meets, and adds a required any{four} on both
// sides, a live base wildcard that meets no part of R's {two,three}. An item
// must advance no run outside its own part: a split that also advanced B's
// any{four} would spend it on a two or three item and leave R's own four item
// nothing to match.
func TestContentRestrictsWildcardUnionCounts(t *testing.T) {
	one, two := NamespaceName("http://one.uri/"), NamespaceName("http://two.uri/")
	three, four := NamespaceName("http://three.uri/"), NamespaceName("http://four.uri/")
	wild := func(o Occurs, ns ...Namespace) Particle {
		return uParticle(t, o, ResolvedTerm{Term: uWildcard(t, NamespaceConstraintEnumeration, ns, ProcessStrict)})
	}
	derived := uGroup(t, CompositorAll,
		wild(uUnbounded(t, 3), one),
		wild(uOccurs(t, 2, 2), two, three))
	all244 := uGroup(t, CompositorAll,
		wild(uUnbounded(t, 5), one, two),
		wild(uOccurs(t, 0, 2), three))
	expectRule(t, cRestricts(t, all244, derived), ruleDerivationOKRestriction)
	covered := uGroup(t, CompositorAll,
		wild(uUnbounded(t, 2), one, two),
		wild(uOccurs(t, 0, 2), three),
		wild(uOccurs(t, 1, 1), four))
	withFour := uGroup(t, CompositorAll,
		wild(uUnbounded(t, 3), one),
		wild(uOccurs(t, 2, 2), two, three),
		wild(uOccurs(t, 1, 1), four))
	if err := cRestricts(t, covered, withFour); err != nil {
		t.Fatalf("a restriction wildcard whose every item some base wildcard counts was rejected: %v", err)
	}
}

// cNillableElem is a once-occurring particle over a local element declaration
// with the given {nillable}.
func cNillableElem(t *testing.T, local string, nillable bool) Particle {
	t.Helper()
	e, err := NewElementDeclaration(xsderr.Loc{}, uq(local), TypeDefinitionRef{Name: uq("T")}, nil, uLocalScope(t), nil, nillable,
		nil, nil, nil, false, nil)
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

// TestContentRestrictsAllGroupDecided pins derivation-ok-restriction clause
// 2.4.2 (cos-content-act-restrict clause 1) where an ·all· group appears on
// either side: the interleave §3.8.4.1.3 defines is decided exactly, so a
// restriction admitting a sequence the base does not is rejected whichever side
// holds the ·all·, and one admitting only the base's sequences is accepted.
//
// The rejecting rows are the ones the pre-#1930 renderings got wrong. An ·all·
// in R was provisionally accepted undecided ("all restriction widening a
// sequence"), and an ·all· in B was modelled by addAll's star, which admits a
// alone under all(a, b) and so accepted the three "drops a required member"
// rows. The accepting rows guard the other direction: reordering the members
// ("sequence reorders the members") and a choice restricting an all of optional
// members are VALID in 1.1, where no pairwise Recurse rule survives.
func TestContentRestrictsAllGroupDecided(t *testing.T) {
	all := func(ps ...Particle) ModelGroup { return uGroup(t, CompositorAll, ps...) }
	seq := func(ps ...Particle) ModelGroup { return uGroup(t, CompositorSequence, ps...) }
	for _, tc := range []struct {
		name           string
		base, derived  ModelGroup
		wantRestricted bool
	}{
		{name: "all:all drops a required member",
			base: all(cElem(t, "a", 1, 1), cElem(t, "b", 1, 1)), derived: all(cElem(t, "a", 1, 1))},
		{name: "all:all drops an optional member",
			base: all(cElem(t, "a", 1, 1), cElem(t, "b", 0, 1)), derived: all(cElem(t, "a", 1, 1)), wantRestricted: true},
		{name: "all:all narrows a member's range",
			base: all(cElem(t, "a", 0, 3), cElem(t, "b", 1, 1)), derived: all(cElem(t, "b", 1, 1), cElem(t, "a", 1, 2)), wantRestricted: true},
		{name: "all:all widens a member's range",
			base: all(cElem(t, "a", 1, 2), cElem(t, "b", 1, 1)), derived: all(cElem(t, "a", 1, 3), cElem(t, "b", 1, 1))},
		{name: "sequence under all drops a required member",
			base: all(cElem(t, "a", 1, 1), cElem(t, "b", 1, 1)), derived: seq(cElem(t, "a", 1, 1))},
		{name: "sequence reorders the members",
			base: all(cElem(t, "a", 1, 1), cElem(t, "b", 1, 1)), derived: seq(cElem(t, "b", 1, 1), cElem(t, "a", 1, 1)), wantRestricted: true},
		{name: "sequence under all repeats a member",
			base: all(cElem(t, "a", 1, 1), cElem(t, "b", 1, 1)), derived: seq(cElem(t, "a", 1, 1), cElem(t, "a", 0, 1), cElem(t, "b", 1, 1))},
		{name: "choice under all of required members",
			base: all(cElem(t, "a", 1, 1), cElem(t, "b", 1, 1)), derived: uGroup(t, CompositorChoice, cElem(t, "a", 1, 1), cElem(t, "b", 1, 1))},
		{name: "choice under all of optional members",
			base: all(cElem(t, "a", 0, 1), cElem(t, "b", 0, 1)), derived: uGroup(t, CompositorChoice, cElem(t, "a", 1, 1), cElem(t, "b", 1, 1)), wantRestricted: true},
		{name: "all restriction widening a sequence",
			base: seq(cElem(t, "a", 1, 1)), derived: all(cElem(t, "a", 1, 1), cElem(t, "b", 1, 1))},
		{name: "all restriction of one optional member",
			base: seq(cElem(t, "a", 0, 1), cElem(t, "b", 0, 1)), derived: all(cElem(t, "a", 0, 1)), wantRestricted: true},
		{name: "all restriction admitting the reverse order",
			base: seq(cElem(t, "a", 0, 1), cElem(t, "b", 0, 1)), derived: all(cElem(t, "a", 0, 1), cElem(t, "b", 0, 1))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := cRestricts(t, tc.base, tc.derived)
			if tc.wantRestricted && err != nil {
				t.Fatalf("a valid restriction of an ·all· content model was rejected: %v", err)
			}
			if !tc.wantRestricted {
				expectRule(t, err, ruleDerivationOKRestriction)
			}
		})
	}
}

// TestContentRestrictsAllGroupBaseDecided restricts an ·all· base to one of its
// members, and to an element that is not among them at all.
func TestContentRestrictsAllGroupBaseDecided(t *testing.T) {
	base := uGroup(t, CompositorAll, cElem(t, "a", 1, 1))
	if err := cRestricts(t, base, uGroup(t, CompositorSequence, cElem(t, "a", 1, 1))); err != nil {
		t.Fatalf("restricting an ·all· base to one of its members was rejected: %v", err)
	}
	expectRule(t, cRestricts(t, base, uGroup(t, CompositorSequence, cElem(t, "b", 1, 1))), ruleDerivationOKRestriction)
}

// TestContentRestrictsOptionalAllGroupUnderSequence pins W3C suite
// MS-Particles particlesK006's shape as ACCEPTED, guarding against
// over-charging it: B is all(a0?, a1, a2?) with the ·all· particle itself
// minOccurs="0", and R is sequence(a1?). pt-actual-restriction clause 1 is
// language containment, and R's {(), (a1)} is within B's, () through the
// ·all·'s minOccurs 0; nothing compares R's a1 0..1 with B's 1..1 per
// particle. Clause 2 holds by loc-testSubP clause 4. #2546 records the suite's
// invalid expectation as a 1.0-era divergence. The control makes the ·all·
// particle required, which leaves () outside B's language.
func TestContentRestrictsOptionalAllGroupUnderSequence(t *testing.T) {
	restricts := func(allMin int) error {
		return dFinalize(t, func(b *SchemaBuilder) {
			all := uGroup(t, CompositorAll, cElem(t, "a0", 0, 1), cElem(t, "a1", 1, 1), cElem(t, "a2", 0, 1))
			base := ElementContent{Particle: uParticle(t, uOccurs(t, allMin, 1), ResolvedTerm{Term: all})}
			b.AddType(dType(t, uq("base"), anyTypeName, base, nil, nil))
			derived := dElementContent(t, false, uGroup(t, CompositorSequence, cElem(t, "a1", 0, 1)))
			b.AddType(dType(t, uq("derived"), uq("base"), derived, nil, nil))
		})
	}
	if err := restricts(0); err != nil {
		t.Fatalf("an optional member restricting an optional ·all· was rejected: %v", err)
	}
	expectRule(t, restricts(1), ruleDerivationOKRestriction)
}

// TestContentRestrictsNestedAllGroup pins addInterleave's composition: an ·all·
// member that is itself an ·all· (cos-all-limited clause 1.3, through a <group
// ref>) interleaves its own members with its siblings', so c may come first
// and b last, and every member at every depth is still required.
func TestContentRestrictsNestedAllGroup(t *testing.T) {
	restricts := func(derived ModelGroup) error {
		return dFinalize(t, func(b *SchemaBuilder) {
			inner := uGroup(t, CompositorAll, cElem(t, "b", 1, 1), cElem(t, "c", 1, 1))
			mgd, err := NewModelGroupDefinition(xsderr.Loc{}, uq("g"), inner)
			if err != nil {
				t.Fatalf("NewModelGroupDefinition: %v", err)
			}
			b.AddModelGroup(mgd)
			base := uGroup(t, CompositorAll, cElem(t, "a", 1, 1), uOne(t, ModelGroupRef{Name: uq("g")}))
			b.AddType(dType(t, uq("base"), anyTypeName, dElementContent(t, false, base), nil, nil))
			b.AddType(dType(t, uq("derived"), uq("base"), dElementContent(t, false, derived), nil, nil))
		})
	}
	seq := func(names ...string) ModelGroup {
		ps := make([]Particle, 0, len(names))
		for _, n := range names {
			ps = append(ps, cElem(t, n, 1, 1))
		}
		return uGroup(t, CompositorSequence, ps...)
	}
	if err := restricts(seq("c", "a", "b")); err != nil {
		t.Fatalf("an order the nested interleave admits was rejected: %v", err)
	}
	expectRule(t, restricts(seq("c", "a")), ruleDerivationOKRestriction)
}

// TestContentRestrictsFutureQuotient pins what the walk's visited set may
// merge (futureClasses): only R-states with the same live positions AND the same
// acceptance. In each row two R-states reach one B-set, since the base's
// wildcard takes both x and y, and only the second of them leads to the
// sequence the base lacks, so a quotient merging them skips the violation.
//
//   - "different live sets": x is followed by a, y by c, and the base admits no
//     c. Merging on acceptance alone accepts.
//   - "different acceptance": x and y are both followed by nothing, but only y
//     ends a sequence, since x is followed by an empty <choice>, which accepts
//     nothing. R is therefore {y}, a single item the two-item base does not
//     accept. Merging on the live set alone accepts.
func TestContentRestrictsFutureQuotient(t *testing.T) {
	anySkip := func() Particle { return cAny(t, NamespaceConstraintAny, nil, ProcessSkip) }
	one := func(g ModelGroup) Particle { return uOne(t, ResolvedTerm{Term: g}) }
	for _, tc := range []struct {
		name          string
		base, derived ModelGroup
	}{
		{name: "different live sets",
			base: uGroup(t, CompositorSequence, anySkip(), one(uGroup(t, CompositorChoice, cElem(t, "a", 1, 1), cElem(t, "b", 1, 1)))),
			derived: uGroup(t, CompositorChoice,
				one(uGroup(t, CompositorSequence, cElem(t, "x", 1, 1), cElem(t, "a", 1, 1))),
				one(uGroup(t, CompositorSequence, cElem(t, "y", 1, 1), cElem(t, "c", 1, 1))))},
		{name: "different acceptance",
			base: uGroup(t, CompositorSequence, anySkip(), anySkip()),
			derived: uGroup(t, CompositorChoice,
				one(uGroup(t, CompositorSequence, cElem(t, "x", 1, 1), one(uGroup(t, CompositorChoice)))),
				cElem(t, "y", 1, 1))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expectRule(t, cRestricts(t, tc.base, tc.derived), ruleDerivationOKRestriction)
		})
	}
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
// does not declare globally; a local particle in B sharing a top-level
// declaration's expanded name is TestContentRestrictsLocalParticleSubstitution's.
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

// TestContentRestrictsLocalParticleSubstitution pins that a LOCAL declaration in
// B heads no ·substitution group· even when a top-level declaration of the same
// expanded name heads one (cvc-accept clause 2.3.2 requires a top-level D;
// #2000). The schema declares top-level a and m, m affiliated to a, and k, which
// m is also affiliated to where a row says so.
//
//   - "element": B's particle is a local a or <element ref="a"/>, and R's is
//     <element ref="m"/>. Only the ref admits m (elementParticleAdmits).
//   - "wildcard": B is choice(a, lax urn:upa wildcard excluding m), and R one lax
//     urn:upa wildcard. Only the ref leaves m covered; the local row's every
//     other name is covered, and item a is governable by the top-level a, so its
//     clause-2 charge is not taken (the GAP(xsd) at elementCoveredSet) and the
//     verdict is clause 1's on m. The local row fails only with BOTH
//     elementCoveredNames' and elementCoveredSet's guards removed: a name split
//     off with no admitting position is charged as surely as one left in the
//     rest, so elementCoveredNames' guard keeps its list exact and alone moves
//     no row here.
//   - "second head": B is choice(seq(local a, b), <element ref="k"/>, lax
//     urn:upa wildcard excluding a, k and m), each branch but k's followed by b,
//     and R is seq(lax urn:upa wildcard excluding k, b). m is split off through
//     ref k, so elementCoveredSet decides which positions admit it: k's alone,
//     whose follow set holds no b, so R's item m followed by b is charged.
//     Crediting the local a with m would admit that b.
func TestContentRestrictsLocalParticleSubstitution(t *testing.T) {
	a, m, k := uq("a"), uq("m"), uq("k")
	ns := []Namespace{NamespaceName(uns)}
	ref := func(n QName) Particle { return uOne(t, ElementDeclarationRef{Name: n}) }
	choice := func(ps ...Particle) Particle {
		return uOne(t, ResolvedTerm{Term: uGroup(t, CompositorChoice, ps...)})
	}
	seq := func(ps ...Particle) ModelGroup { return uGroup(t, CompositorSequence, ps...) }
	restricts := func(mAffiliations []QName, base, derived ModelGroup) error {
		return dFinalize(t, func(b *SchemaBuilder) {
			b.AddElement(uGlobal(t, a, uq("T")))
			b.AddElement(uGlobal(t, k, uq("T")))
			b.AddElement(uGlobal(t, m, uq("T"), mAffiliations...))
			b.AddType(dType(t, uq("base"), anyTypeName, dElementContent(t, false, base), nil, nil))
			b.AddType(dType(t, uq("derived"), uq("base"), dElementContent(t, false, derived), nil, nil))
		})
	}
	toA := []QName{a}
	for _, tc := range []struct {
		name           string
		mAffiliations  []QName
		base, derived  ModelGroup
		wantRestricted bool
	}{
		{name: "element local", mAffiliations: toA,
			base: seq(cElem(t, "a", 1, 1)), derived: seq(ref(m))},
		{name: "element local unaffiliated",
			base: seq(cElem(t, "a", 1, 1)), derived: seq(ref(m))},
		{name: "element ref", mAffiliations: toA,
			base: seq(ref(a)), derived: seq(ref(m)), wantRestricted: true},
		{name: "wildcard local", mAffiliations: toA,
			base:    seq(choice(cElem(t, "a", 1, 1), cAnyExcept(t, ns, ProcessLax, m))),
			derived: seq(cAnyExcept(t, ns, ProcessLax))},
		{name: "wildcard ref", mAffiliations: toA,
			base:    seq(choice(ref(a), cAnyExcept(t, ns, ProcessLax, m))),
			derived: seq(cAnyExcept(t, ns, ProcessLax)), wantRestricted: true},
		{name: "second head", mAffiliations: []QName{a, k},
			base: seq(choice(
				uOne(t, ResolvedTerm{Term: seq(cElem(t, "a", 1, 1), cElem(t, "b", 1, 1))}),
				ref(k),
				uOne(t, ResolvedTerm{Term: seq(cAnyExcept(t, ns, ProcessLax, a, k, m), cElem(t, "b", 1, 1))}))),
			derived: seq(cAnyExcept(t, ns, ProcessLax, k), cElem(t, "b", 1, 1))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := restricts(tc.mAffiliations, tc.base, tc.derived)
			if tc.wantRestricted && err != nil {
				t.Fatalf("m is in the top-level a's substitution group, so B's ref to a admits it: %v", err)
			}
			if !tc.wantRestricted {
				expectRule(t, err, ruleDerivationOKRestriction)
			}
		})
	}
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

// TestInterleavePositionsCountsConstruction pins that interleavePositions counts
// what addInterleave emits, which is what makes maxContentPositions bound the
// automaton rather than estimate it (see unfoldedPositions): for each ·all·
// shape the count equals the built automaton's position table, and a wide ·all·
// saturates one past the ceiling rather than being counted out.
func TestInterleavePositionsCountsConstruction(t *testing.T) {
	s := rSchema(t, nil)
	all := func(ps ...Particle) ModelGroup { return uGroup(t, CompositorAll, ps...) }
	for _, tc := range []struct {
		name string
		g    ModelGroup
		want int
	}{
		{name: "one member", g: all(cElem(t, "a", 1, 1)), want: 1},
		{name: "two members", g: all(cElem(t, "a", 1, 1), cElem(t, "b", 0, 1)), want: 4},
		{name: "a repeated member", g: all(cElem(t, "a", 0, 3), cElem(t, "b", 1, 1)), want: 10},
		{name: "an unbounded member", g: all(cUnbounded(t, "a", 2), cElem(t, "b", 1, 1), cElem(t, "c", 0, 1)), want: 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := uOne(t, ResolvedTerm{Term: tc.g})
			if got := s.unfoldedPositions(p); got != tc.want {
				t.Fatalf("unfoldedPositions = %d, want %d", got, tc.want)
			}
			a, err := s.contentAutomatonOf(ElementContent{Particle: p})
			if err != nil {
				t.Fatalf("contentAutomatonOf: %v", err)
			}
			if got := len(a.positions); got != tc.want {
				t.Fatalf("addInterleave emitted %d positions, want the %d interleavePositions counts", got, tc.want)
			}
		})
	}
	wide := make([]Particle, 0, 13)
	for _, name := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m"} {
		wide = append(wide, cElem(t, name, 0, 1))
	}
	if got := s.unfoldedPositions(uOne(t, ResolvedTerm{Term: all(wide...)})); got != maxContentPositions+1 {
		t.Fatalf("a 13-member ·all· of optional members (13 × 2^12 positions) counts %d, want the saturated %d", got, maxContentPositions+1)
	}
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
