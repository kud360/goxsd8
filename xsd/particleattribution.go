package xsd

import (
	"fmt"

	"github.com/kud360/goxsd8/xsderr"
)

// ruleCosNonambig is Unique Particle Attribution (Structures §3.8.6.4,
// id="cos-nonambig"): "A content model must not contain two ·element particles·
// which ·compete· with each other, nor two ·wildcard particles· which ·compete·
// with each other." An ·element particle· and a ·wildcard particle· competing is
// explicitly NOT prohibited — that is the sole substantive 1.0→1.1 change to this
// constraint (Appendix G.1.3), with the Element Declaration winning ·attribution·.
//
// Appendix J (non-normative) is the only algorithmic account the spec offers and
// is cited in comments below as guidance; the rule ID charged to every rejection
// is cos-nonambig, never "Appendix J", which is not a rule.
const ruleCosNonambig xsderr.Rule = "cos-nonambig"

// This file decides cos-nonambig by building the position automaton (the
// Glushkov automaton) of a content model and checking that no state, and no
// subset of states one ·path· prefix reaches, offers two competing transitions
// (decideUnambiguous). Appendix J sketches this construction —
// "transcribe the content model into an automaton in the usual way using epsilon
// transitions for optionality and unbounded maxOccurs, unfolding other numeric
// occurrence ranges … using not element QNames as transition labels, but rather
// pairs of element QNames and positions in the model" — and the theorem that
// makes it a decision procedure rather than a heuristic is Brüggemann-Klein and
// Wood's characterization of one-unambiguous regular languages ("One-Unambiguous
// Regular Languages", Information and Computation 140(2), 1998), which Appendix
// J's own bibliography points at: a regular expression is 1-unambiguous exactly
// when its position automaton is deterministic. That theorem is about an
// expression whose positions are its particles. An unfolding is not one: its
// copies of one particle are several positions sharing one particle identifier,
// and a ·path· (§3.8.4.1, key-path) does not say which copy it passed through.
// ·compete· (§3.8.4.2, key-compete) — two particles reachable by two ·paths· of
// one sequence differing only in their last item — is therefore two positions
// live in one SUBSET of the automaton, the positions one identifier sequence
// reaches (checkPaths); two live in one state is the special case check finds.
//
// The two-bullet shortcut Appendix J states first ("both in the {particles} of a
// choice or all group", or "may ·validate· adjacent items and the first has
// {min occurs} < {max occurs}") is NOT used: it misses (a, b) | (a, c), whose
// choice {particles} are two SEQUENCE particles, not the two a's. The automaton
// finds it, because both a's are live in the start state.
//
// Nothing built here is memoized on any component. The position table, first
// sets, follow sets and emptiability are finalize-scoped intermediates, built per
// content model and discarded on return (STYLE D3). One ModelGroup value is
// reachable from two different content models — the same aliasing hazard
// wildcardadmit.go's file comment records — so a set cached on it would answer
// the second content model's question with the first's answer.
//
// The flatten follows exactly two reference edges, both licensed by Finalize's
// Phase B having already run, which is why it carries no visited set (PRINCIPLES
// 5): ModelGroupRef edges (checkModelGroupsAcyclic, mg-props-correct clause 2)
// and — inside the ·overlap· relation, not the flatten — {substitution group
// affiliations} edges (checkSubstitutionGroupsAcyclic, e-props-correct clause 5).
// The FLATTEN itself does not follow {base type definition}: a derived type's
// {content type} already folds in whatever the derivation contributes
// (§3.4.2.3.3), so the base chain has nothing to add to a content model. The
// ·overlap· relation does follow it, for cos-equiv-derived-ok-rec clause 2.3,
// and terminates on checkComplexBaseAcyclic (ct-props-correct clause 3) plus the
// explicit test for the one self-based type §3.4.7 permits (substitutiongroup.go).

// maxMandatoryCopies and maxOptionalCopies bound how many copies of a particle's
// {term} fragment the unfolding emits, so that maxOccurs="100000" does not build
// a hundred thousand of them.
//
// The bound is verdict-preserving for check, the per-state half of
// cos-nonambig (decideUnambiguous), and for that reader alone — the property
// is quantified, not flat (STYLE P3a). What it preserves is which
// particle-IDENTIFIER sets are live together in one state. It does NOT
// preserve the language the automaton accepts, which it demonstrably changes
// (e{3,6} reads as e{2,4}), nor the subsets one ·path· prefix reaches, which
// it changes in both directions (checkPaths gives a shape each way). The two
// readers of those — checkPaths, the reached-subset half of cos-nonambig, and
// contentrestricts.go's cos-content-act-restrict walk, which decides language
// containment — therefore build under languagePolicy's exact unfolding, and
// these constants are on neither path (#501).
//
// The argument for the per-state claim: every unfolded copy of one source
// particle replays the SAME particle identifiers (see automaton.addParticle),
// so all copies of a fragment have identical ·first·, ·last· and internal follow
// sets when read as sets of particle identifiers. The competition sets a chain of
// n copies produces are therefore, as identifier sets, only these: the fragment's
// own internal ones; FIRST (from the whole particle's start, and from the last of
// each mandatory copy); and FIRST ∪ SUCC (from the last of a copy that some later
// optional copy may follow, where SUCC is what follows the whole particle). Two
// mandatory copies already realize both "last of a mandatory copy" cases, and two
// optional copies already realize both "one optional copy left" and "more than
// one optional copy left"; further copies repeat an identifier set already
// present.
const (
	maxMandatoryCopies = 2
	maxOptionalCopies  = 2
)

// maxPathPositions and maxPathSubsets bound checkPaths' work: the positions of
// the exactly unfolded automaton it walks (unfoldedPositions' count, checked
// before the automaton is built) and the subsets it visits. Each is a ceiling
// on cost, never on the verdict of a walk that runs to its end, and passing
// either leaves the reached-subset pairs uncharged under the GAP(xsd) markers
// at decideUnambiguous and checkPaths.
//
// Both are measured, on this commit's construction (one machine, one run).
// The exact unfolding of a wide optional range is cubic to walk, because a
// copy is followed by every later one: a whole Finalize of e{0,n} took 142 ms
// at n = 512 and 889 ms at n = 1024, and 44 µs at n = 2048, past the ceiling,
// where check alone runs. Over the full W3C suite, run with the
// position ceiling at 4096, the walk was reached 65096 times and charged no
// content model; 15 of those walks were over more than 1024 positions, and so
// are declined at this value, and every walk within 1024 positions visited at
// most 1025 subsets.
const (
	maxPathPositions = 1024
	maxPathSubsets   = 2048
)

// position is one occurrence, in a flattened content model, of an ·element
// particle· or a ·wildcard particle· (§3.8.6.4, key-ep/key-wp).
//
// particleID identifies the SOURCE particle, and is what ·compete· is about:
// §3.8.6.4 says "particles at different points in the content model are always
// distinct from one another, even if they originated from the same named model
// group", so two <group ref>s to one definition yield distinct identifiers, while
// the unfolded copies of ONE particle's numeric occurrence range share theirs.
// Comparing identifiers — never names, never pointers — is what makes the
// unfolding safe: without it,
//
//	<sequence><element name="a" minOccurs="2" maxOccurs="2"/><element name="a"/></sequence>
//
// would be rejected, since the two copies of the first particle would look like
// two competing particles.
//
// term is an ElementDeclaration or a Wildcard, never a ModelGroup: a model group
// is not a leaf of the flattening.
type position struct {
	particleID int
	term       Term
}

// automaton is the position automaton of ONE content model under construction.
// follow is indexed by position, and every position set is an ascending []int, so
// no map is consulted to decide which violation is reported (STYLE D1/D2).
//
// policy is the construction policy, not automaton state: it is chosen once by
// the caller and read only by addParticle and addModelGroup (see
// constructionPolicy).
type automaton struct {
	s              *Schema
	positions      []position
	follow         [][]int
	nextParticleID int
	policy         constructionPolicy
}

// constructionPolicy is how one consumer's automaton renders the two constructs
// whose faithful rendering depends on the fact being decided: a numeric
// occurrence range (unfold, read by addParticle) and an <all> group (all, read
// by addModelGroup). The readers of this construction need different renderings
// and none may be given another's. cos-nonambig's per-state check takes
// attributionPolicy: unfoldCopies, whose two-copy bound is verdict-preserving
// for identifier-set competition in one state, and addAll, whose star is exact
// for the pairs one state holds but a strict superset of the interleave's
// language. cos-nonambig's reached-subset checkPaths and contentrestricts.go's
// cos-content-act-restrict walk take languagePolicy: unfoldExactly and
// addInterleave, because both of attributionPolicy's renderings rewrite the
// accepted language and the subsets a ·path· prefix reaches (see
// maxMandatoryCopies, addAll and checkPaths). The two halves are one
// value so that no automaton pairs one consumer's unfolding with the other's
// <all>, and every construction site names its policy, so none is inherited by
// default.
type constructionPolicy struct {
	unfold func(Occurs) (copies, mandatory int, loop bool)
	all    func(b *automaton, g ModelGroup) ([]int, []int, bool, error)
}

// attributionPolicy is the constructionPolicy of cos-nonambig's per-state check.
var attributionPolicy = constructionPolicy{unfold: unfoldCopies, all: (*automaton).addAll}

// checkContentModelsUnambiguous is the cos-nonambig half of Phase C. It walks the
// compiled set in document order (STYLE D2) and returns the first violation.
//
// Every Model Group is checked, as §3.8.6's unqualified chapeau requires ("All
// model groups … must satisfy the following constraints"): checking a complex
// type's content model root discharges every group nested inside it, since the
// root's automaton spans them all, and a Model Group Definition's {model group} is
// checked in its own right whether or not any <group ref> points at it — §3.7.1
// makes it a Model Group component the moment <group name="…"> is processed, and
// §3.8.4.1.4 calls a standalone group shape violating UPA non-conforming without
// mentioning reference status.
//
// The complex types are complextypewalk.go's, not s.types: an ANONYMOUS type a
// declaration owns has a content model the same chapeau binds, and ranging
// s.types reached none of them (#438). A declaration nested in one type's
// content model is entered for its own owned types, so the walk is by descent
// and not by a second root loop.
func (s *Schema) checkContentModelsUnambiguous() error {
	w := complexTypeWalk{
		complexType:          s.checkContentModelUnambiguous,
		modelGroupDefinition: s.checkModelGroupDefinitionUnambiguous,
	}
	return w.schema(s)
}

// checkContentModelUnambiguous charges cos-nonambig against one complex type's
// content model. An empty or simple {content type} holds no particle and so no
// content model to be ambiguous.
//
// The context phrase is complexTypeOwner's, which renders an anonymous type by
// what it is: its {name} is the zero QName, so naming it would leave a hole in
// the message.
func (s *Schema) checkContentModelUnambiguous(c ComplexType) error {
	ec, ok := c.ContentType().(ElementContent)
	if !ok {
		return nil
	}
	build := func(b *automaton) ([]int, []int, bool, error) { return b.addParticle(ec.Particle) }
	return s.decideUnambiguous(build, s.unfoldedPositions(ec.Particle), complexTypeOwner(c)+" content model")
}

// checkModelGroupDefinitionUnambiguous charges cos-nonambig against one Model
// Group Definition's {model group}.
func (s *Schema) checkModelGroupDefinitionUnambiguous(mgd ModelGroupDefinition) error {
	g := mgd.ModelGroup()
	build := func(b *automaton) ([]int, []int, bool, error) { return b.addModelGroup(g) }
	return s.decideUnambiguous(build, s.modelGroupPositions(g), "model group definition "+mgd.Name().Display())
}

// decideUnambiguous charges cos-nonambig against one content model, which build
// emits into an automaton and whose exact unfolding emits positions positions
// (unfoldedPositions' count). It runs two passes and reports the first
// competing pair either finds:
//
//  1. check, over the attributionPolicy automaton: the pairs live together in
//     one state, which maxMandatoryCopies' bound preserves;
//  2. checkPaths, over the languagePolicy automaton: the pairs live together
//     in one SUBSET a ·path· prefix reaches, which no copy bound preserves, so
//     that automaton unfolds every occurrence range exactly.
//
// check runs first and runs on every content model; checkPaths runs only
// within maxPathPositions, and only once check has charged nothing.
func (s *Schema) decideUnambiguous(build func(*automaton) ([]int, []int, bool, error), positions int, ctx string) error {
	b := &automaton{s: s, policy: attributionPolicy}
	first, _, _, err := build(b)
	if err != nil {
		return err
	}
	if err := b.check(first, ctx); err != nil {
		return err
	}
	if positions > maxPathPositions {
		// GAP(xsd): cos-nonambig's reached-subset pass (checkPaths) is not run on
		// a content model whose exact unfolding would exceed maxPathPositions, so
		// two particles that ·compete· only across unfolded copies of a counted
		// particle (§3.8.4.2, key-compete) go uncharged there and only check's
		// per-state pairs are charged. No owner issue is filed yet. Direction, by
		// consumer: SchemaBuilder.Finalize, and parser.Parse through it, loses the
		// rejection and gains none. Schema.ContentMatcher's Matcher, which
		// validate's cvc-complex-content walk and codec's Decode read, assumes no
		// such pair exists; on sequence(sequence{2}(c?, a+), c) it was measured
		// rejecting a a c, which the model accepts. Whether it can also accept a
		// sequence the model rejects is unestablished.
		return nil
	}
	x := &automaton{s: s, policy: languagePolicy}
	first, _, _, err = build(x)
	if err != nil {
		return err
	}
	return x.checkPaths(first, ctx)
}

// check reports the first competing pair live in one state of the automaton,
// the per-state half of decideUnambiguous. States are visited in
// a fixed order — the start state first, then the state after position 0, after
// position 1, … — and within a state the pair with the least (i, j) by position
// index is reported, so the reported violation is the lexicographically least
// (state, i, j) and identical inputs always reject identically (STYLE D1).
func (b *automaton) check(first []int, ctx string) error {
	if err := b.checkState(first, ctx, "at the start of"); err != nil {
		return err
	}
	for i := range b.follow {
		if err := b.checkState(b.follow[i], ctx, "after "+b.describe(i)+" in"); err != nil {
			return err
		}
	}
	return nil
}

// checkPaths reports the first competing pair live in one reached SUBSET of the
// automaton: the positions one ·path· prefix may stand at. A ·path· (§3.8.4.1,
// key-path) is "the sequence of ·basic particles· which the items of S match",
// and the unfolded copies of one particle are one particle, so a prefix that
// two copies can each have consumed reaches both, and ·compete· (§3.8.4.2,
// key-compete) asks about the union of what may follow them.
// sequence(sequence{2}(c?, a+), c) is the shape no single state shows: after
// a a, copy 1's a is followed by copy 2's c and copy 2's a by the outer c, and
// only the subset {copy 1's a, copy 2's a} holds both c's.
//
// It is the subset construction over particle identifiers: from the start
// subset, the positions live next are partitioned by particleID, and each part
// is the subset the prefix extended by that particle reaches. The walk is
// contentrestricts.go's B-side subset walk reused (STYLE T4):
// contentAutomaton.liveGroups for the partition and subsetTable for the visited
// set, whose identifier order is the breadth-first queue. Every part is interned
// in ascending order of its first position, so the reported pair is fixed by the
// content model alone (STYLE D1/D2). Only one representative per group is put
// to checkState: a particleID determines the {term} (liveSet), so competes
// answers every member of a group as it answers the representative, and two
// members of one group never compete.
//
// b must be built under languagePolicy, so that the reached subsets are the
// language's own. Over attributionPolicy's two-copy bound they are not, in
// either direction: sequence(choice{3,3}(b{2,3}, a), a) has the competing a's
// after b b b b b b (three b{2,3} blocks or two), which choice{2,2} cannot
// show, and sequence(choice{2,2}(b{3,5}, a), a), which has no such pair, would
// read b{3,5} as b{2,4} and find one after b b b b. And addAll's star would put
// an <all> member's star and primed positions in one subset whose union offers
// the group's successor beside the members, a competition the interleave does
// not have.
func (b *automaton) checkPaths(first []int, ctx string) error {
	a := contentAutomaton{automaton: b, first: first}
	reached := newSubsetTable()
	reached.intern([]int{startState})
	for id := 0; id < len(reached.sets); id++ {
		if id == maxPathSubsets {
			// GAP(xsd): cos-nonambig's reached-subset pass stops after
			// maxPathSubsets subsets, and a pair live only in a later one goes
			// uncharged. No owner issue is filed yet; the direction, by
			// consumer, is decideUnambiguous' maxPathPositions marker's.
			return nil
		}
		subset := reached.set(id)
		live := a.liveGroups(subset)
		if err := b.checkState(live.reps, ctx, b.reachedAt(subset)); err != nil {
			return err
		}
		part := make([]bool, len(live.reps))
		for g := range live.reps {
			part[g] = true
			reached.intern(live.positionsOf(part))
			part[g] = false
		}
	}
	return nil
}

// reachedAt is the "where" phrase checkState's message gives for one subset
// checkPaths reached. Every member of a subset past the start is a copy of one
// particle, so its first member names them all.
func (b *automaton) reachedAt(subset []int) string {
	if subset[0] == startState {
		return "at the start of"
	}
	if len(subset) == 1 {
		return "after " + b.describe(subset[0]) + " in"
	}
	return fmt.Sprintf("after %s, where one ·path· may stand at any of %d copies of it, in", b.describe(subset[0]), len(subset))
}

// checkState rejects the first competing pair live in one state of the automaton.
func (b *automaton) checkState(state []int, ctx, where string) error {
	for x := 0; x < len(state); x++ {
		for y := x + 1; y < len(state); y++ {
			competes, err := b.competes(state[x], state[y])
			if err != nil {
				return fmt.Errorf("checking unique particle attribution %s %s: %w", where, ctx, err)
			}
			if !competes {
				continue
			}
			return xsderr.New(ruleCosNonambig, xsderr.Loc{},
				"%s %s, %s and %s ·compete· with each other, but cos-nonambig forbids two ·element particles· (or two ·wildcard particles·) that compete",
				where, ctx, b.describe(state[x]), b.describe(state[y]))
		}
	}
	return nil
}

// competes reports whether the particles at two positions live in one state
// ·compete· in the sense cos-nonambig forbids.
//
// Two positions of the SAME source particle never compete: ·compete· relates two
// particles, and §3.8.6.4's "identical except that one path has P1 as its last
// item and the other has P2" cannot hold for P1 = P2. This is the test that makes
// unfolding a numeric occurrence range sound.
//
// An ·element particle· against a ·wildcard particle· never competes in the
// forbidden sense: 1.1 relaxed exactly that pairing (Appendix G.1.3, "competition
// between an element particle and a wildcard particle is no longer forbidden"),
// the Element Declaration winning ·attribution·. The two kinds are therefore kept
// apart here rather than folded into one kind-blind overlap test.
func (b *automaton) competes(i, j int) (bool, error) {
	pi, pj := b.positions[i], b.positions[j]
	if pi.particleID == pj.particleID {
		return false, nil
	}
	switch ti := pi.term.(type) {
	case ElementDeclaration:
		dj, ok := pj.term.(ElementDeclaration)
		if !ok {
			return false, nil // element particle vs wildcard particle: legal in 1.1
		}
		return b.s.elementsOverlap(ti, dj), nil
	case Wildcard:
		wj, ok := pj.term.(Wildcard)
		if !ok {
			return false, nil // wildcard particle vs element particle: legal in 1.1
		}
		return wildcardsOverlap(ti, wj)
	default:
		panic("xsd: automaton.competes: position {term} is neither an element declaration nor a wildcard")
	}
}

// describe names a position for an error message.
func (b *automaton) describe(i int) string {
	switch t := b.positions[i].term.(type) {
	case ElementDeclaration:
		return "element declaration " + t.Name().Display()
	case Wildcard:
		return "wildcard " + t.NamespaceConstraint().Variety().String()
	default:
		return "particle"
	}
}

// elementsOverlap is Appendix J's ·overlap· relation for two ·element particles·,
// bullets 1-3, cheapest first:
//
//  1. the two declarations have the same expanded name;
//  2. one's expanded name is the name of a declaration in the other's
//     ·substitution group·;
//  3. both are GLOBAL declarations whose ·substitution groups· contain the same
//     element declaration.
//
// {scope} does not enter the expanded-name test: §3.8.6.4's closing Note says
// "the scope of declarations is not relevant to enforcing either the Unique
// Particle Attribution constraint or the Element Declarations Consistent
// constraint". It enters bullets 2-3 only because a LOCAL declaration heads no
// ·substitution group· at all (§3.3.6.4 defines one per declaration "in the
// {element declarations} of a schema", which §3.17.1 restricts to top-level).
//
// Membership is inSubstitutionGroupOf, which decides cos-equiv-derived-ok-rec
// exactly (substitutiongroup.go), so the ·overlap· relation this constraint fires
// on is the spec's relation and no over-broad group can reject a valid schema.
func (s *Schema) elementsOverlap(a, b ElementDeclaration) bool {
	if a.Name() == b.Name() {
		return true // bullet 1
	}
	if s.nameInSubstitutionGroupOf(a.Name(), b) {
		return true // bullet 2
	}
	if s.nameInSubstitutionGroupOf(b.Name(), a) {
		return true // bullet 2, the other direction
	}
	return s.substitutionGroupsIntersect(a, b) // bullet 3
}

// nameInSubstitutionGroupOf reports whether name is the expanded name of an
// element declaration in head's ·substitution group·. The test is on the NAME,
// not on the declaration: Appendix J bullet 2 says "one of them has the same
// expanded name as an element declaration in the other's substitution group", so
// a local declaration named x overlaps a head whose group contains the top-level
// x.
func (s *Schema) nameInSubstitutionGroupOf(name QName, head ElementDeclaration) bool {
	if head.ScopeVariety() != ScopeGlobal {
		return false // a local declaration heads no substitution group
	}
	return s.inSubstitutionGroupOf(name, head.Name())
}

// substitutionGroupsIntersect is Appendix J bullet 3: two GLOBAL element
// declarations overlap when their ·substitution groups· share a member. 1.1
// allows a declaration several {substitution group affiliations}, so this is not
// subsumed by bullet 2 — one member can sit in two otherwise unrelated groups.
//
// The candidate members are enumerated from the document-order {element
// declarations} slice, never from elementIndex, so no map iteration order reaches
// the verdict (STYLE D2).
func (s *Schema) substitutionGroupsIntersect(a, b ElementDeclaration) bool {
	if a.ScopeVariety() != ScopeGlobal || b.ScopeVariety() != ScopeGlobal {
		return false
	}
	for _, e := range s.elements {
		if !s.inSubstitutionGroupOf(e.Name(), a.Name()) {
			continue
		}
		if s.inSubstitutionGroupOf(e.Name(), b.Name()) {
			return true
		}
	}
	return false
}

// wildcardsOverlap is Appendix J's ·overlap· relation for two ·wildcard
// particles·: their {namespace constraint}s overlap when the Attribute Wildcard
// Intersection (§3.10.6.4, cos-aw-intersect) of the two has {variety} any, or
// {variety} not, or {variety} enumeration with a non-empty {namespaces}.
//
// The intersection itself is not re-derived: intersectNamespaceConstraint is the
// one canonical §3.10.6.4 implementation (STYLE T4). Its error is documented
// unreachable for two validly-constructed operands, but it is propagated rather
// than dropped (STYLE S3) so any future divergence surfaces as a failure instead
// of a silently wrong verdict.
func wildcardsOverlap(a, b Wildcard) (bool, error) {
	c, err := intersectNamespaceConstraint(xsderr.Loc{}, a.NamespaceConstraint(), b.NamespaceConstraint())
	if err != nil {
		return false, fmt.Errorf("intersecting the {namespace constraint}s of two wildcard particles (cos-aw-intersect): %w", err)
	}
	return admitsSomeNamespace(c), nil
}

// admitsSomeNamespace reports whether c admits at least one namespace name:
// {variety} any, or {variety} not, or {variety} enumeration with a non-empty
// {namespaces}. It is the emptiness half of wildcardsOverlap's test, shared with
// contentrestricts.go: coveringWildcardUnion asks it of the intersections it
// splits a restriction wildcard into, and contentModelRestricts of each
// restriction wildcard before taking its transition. {disallowed names} does
// not enter: its QNames, and the names its defined and sibling keywords resolve
// to in a finite schema, are finitely many, while an admitted namespace holds
// infinitely many local names.
func admitsSomeNamespace(c NamespaceConstraint) bool {
	switch c.Variety() {
	case NamespaceConstraintAny, NamespaceConstraintNot:
		return true
	default:
		return len(c.namespaces) > 0
	}
}

// addParticle emits the fragment for one particle of a content model, returning
// its ·first· positions, its ·last· positions, and whether it accepts the empty
// sequence. Both position sets are ascending.
//
// A numeric occurrence range is UNFOLDED into copies, as Appendix J directs, not
// encoded as a loop: a loop conflates "which repetition am I in" and so
// false-rejects a{2,2} followed by a. Only optionality and an unbounded {max
// occurs} become epsilon-like structure (a skippable copy, and a loop-back edge
// on the last copy).
//
// Every copy REPLAYS the particle identifiers the first copy allocated, by
// resetting the allocator before each copy. The traversal is deterministic and
// depends only on the source subtree, so copy k assigns exactly the identifiers
// copy 1 did — which is precisely the statement that an unfolded copy is the same
// source particle, at every depth beneath it as well as at its own leaf.
func (b *automaton) addParticle(p Particle) ([]int, []int, bool, error) {
	copies, mandatory, loop := b.policy.unfold(p.Occurs())
	if copies == 0 {
		return nil, nil, true, nil // a vacuous {0,0} particle accepts only the empty sequence
	}
	startID := b.nextParticleID
	seq := sequenceFragment{emptiable: true}
	var tailFirst, tailLast []int
	for i := 0; i < copies; i++ {
		b.nextParticleID = startID
		f, l, e, err := b.addTerm(p.Term())
		if err != nil {
			return nil, nil, false, err
		}
		if i >= mandatory {
			e = true // a copy past {min occurs} may be skipped
		}
		b.appendToSequence(&seq, f, l, e)
		tailFirst, tailLast = f, l
	}
	if loop {
		// An unbounded {max occurs}: after the last copy the same copy may start
		// again. This is the one place a cycle enters the automaton, and it is
		// the epsilon-transition Appendix J licenses for unbounded ranges.
		for _, q := range tailLast {
			b.addFollow(q, tailFirst)
		}
	}
	return seq.first, seq.last, seq.emptiable, nil
}

// unfoldCopies is the unfolding cos-nonambig's per-state check is decided over
// (attributionPolicy): how many copies of a particle's {term} fragment to emit
// for an occurrence range, how many of them are mandatory, and whether the last
// one carries a loop-back edge. See maxMandatoryCopies for why bounding the copy
// count changes no verdict of check — and for why that argument is quantified
// over that reader alone, so a consumer deciding some other fact, checkPaths
// among them, must supply its own unfolding rather than inherit this one.
func unfoldCopies(o Occurs) (copies, mandatory int, loop bool) {
	bound, bounded := o.Max()
	if bounded && bound == 0 {
		return 0, 0, false
	}
	mandatory = min(o.Min(), maxMandatoryCopies)
	if !bounded {
		return max(mandatory, 1), mandatory, true
	}
	return mandatory + min(bound-o.Min(), maxOptionalCopies), mandatory, false
}

// addTerm emits the fragment for a particle's {term}. A <group ref> is followed
// through modelGroupIndex, and an <element ref> through elementIndex, exactly as
// wildcardadmit.go and resolve.go read them, so no Schema.ModelGroup accessor is
// minted for an in-package reader (STYLE T5). A ref that resolves to nothing
// contributes nothing: Phase A already rejected a dangling reference
// (src-resolve), so that arm is unreachable on a *Schema that exists.
func (b *automaton) addTerm(t TermOrRef) ([]int, []int, bool, error) {
	switch t := t.(type) {
	case ResolvedTerm:
		return b.addResolvedTerm(t.Term)
	case ElementDeclarationRef:
		d, ok := b.s.Element(t.Name)
		if !ok {
			return nil, nil, true, nil
		}
		return b.addLeaf(d)
	case ModelGroupRef:
		mgd, ok := b.s.modelGroupIndex[t.Name]
		if !ok {
			return nil, nil, true, nil
		}
		return b.addModelGroup(mgd.ModelGroup())
	default:
		panic("xsd: automaton.addTerm: non-exhaustive TermOrRef switch")
	}
}

// addResolvedTerm emits the fragment for an inline {term}, exhaustively over the
// sealed Term sum.
func (b *automaton) addResolvedTerm(t Term) ([]int, []int, bool, error) {
	switch t := t.(type) {
	case ElementDeclaration:
		return b.addLeaf(t)
	case ModelGroup:
		return b.addModelGroup(t)
	case Wildcard:
		return b.addLeaf(t)
	default:
		panic("xsd: automaton.addResolvedTerm: non-exhaustive Term switch")
	}
}

// addLeaf emits one position for an ·element particle· or ·wildcard particle·,
// allocating the source-particle identifier the unfolding replays.
func (b *automaton) addLeaf(t Term) ([]int, []int, bool, error) {
	id := b.nextParticleID
	b.nextParticleID++
	i := len(b.positions)
	b.positions = append(b.positions, position{particleID: id, term: t})
	b.follow = append(b.follow, nil)
	return []int{i}, []int{i}, false, nil
}

// addModelGroup emits the fragment for a model group, dispatching on its
// {compositor} (§3.8.4.1's per-compositor ·path· construction rules).
func (b *automaton) addModelGroup(g ModelGroup) ([]int, []int, bool, error) {
	switch g.Compositor() {
	case CompositorSequence:
		return b.addSequence(g)
	case CompositorChoice:
		return b.addChoice(g)
	case CompositorAll:
		return b.policy.all(b, g)
	default:
		panic("xsd: automaton.addModelGroup: non-exhaustive Compositor switch")
	}
}

// addSequence emits a concatenation (§3.8.4.1.1).
func (b *automaton) addSequence(g ModelGroup) ([]int, []int, bool, error) {
	seq := sequenceFragment{emptiable: true}
	for _, p := range g.particles {
		f, l, e, err := b.addParticle(p)
		if err != nil {
			return nil, nil, false, err
		}
		b.appendToSequence(&seq, f, l, e)
	}
	return seq.first, seq.last, seq.emptiable, nil
}

// addChoice emits a union (§3.8.4.1.2): every member is live at the start, which
// is where Appendix J's "both in the {particles} of a choice group" case is
// decided, and no follow edge is added between members.
//
// An empty <choice> accepts nothing at all, so it is not emptiable — the empty
// sequence reaches it only through a {min occurs} of 0 on the particle above,
// which addParticle already accounts for.
func (b *automaton) addChoice(g ModelGroup) ([]int, []int, bool, error) {
	var first, last []int
	emptiable := false
	for _, p := range g.particles {
		f, l, e, err := b.addParticle(p)
		if err != nil {
			return nil, nil, false, err
		}
		first = mergePositions(first, f)
		last = mergePositions(last, l)
		emptiable = emptiable || e
	}
	return first, last, emptiable, nil
}

// allFragment is one pass over an <all> group's members: the union of their
// ·first· positions, the union of their ·last· positions, whether every one of
// them accepts the empty sequence, and the group's RESIDUAL — the positions still
// live once the group has been completed.
//
// residual is what §3.8.4.1.3 leaves live after S1 × … × Sn is accounted for: a
// member whose Si may be empty (·emptiable·, cos-group-emptiable §3.9.6.3) has
// not been taken yet and may still start, and a member taken as far as one of its
// own ·last· positions may still continue past it when its {max occurs} exceeds
// the occurrences already consumed. It is snapshotted while the members are
// emitted, before addAll draws any group-level edge, so it holds each member's
// own continuations and nothing the group adds around them.
type allFragment struct {
	first     []int
	last      []int
	residual  []int
	emptiable bool
}

// addAllMembers emits one fragment per member of an <all> group and collects the
// four facts addAll composes them from. Members are walked in document order
// (STYLE D2).
func (b *automaton) addAllMembers(g ModelGroup) (allFragment, error) {
	frag := allFragment{emptiable: true}
	for _, p := range g.particles {
		f, l, e, err := b.addParticle(p)
		if err != nil {
			return allFragment{}, err
		}
		frag.first = mergePositions(frag.first, f)
		frag.last = mergePositions(frag.last, l)
		frag.emptiable = frag.emptiable && e
		if e {
			frag.residual = mergePositions(frag.residual, f)
		}
		for _, q := range l {
			frag.residual = mergePositions(frag.residual, b.follow[q])
		}
	}
	return frag, nil
}

// addAll emits an <all> group (§3.8.4.1.3, interleave). The literal interleave of
// n members has n! orderings and must not be built. What is built instead is
//
//	all(P1 … Pn) ≈ (P1 | P2 | … | Pn)* · (P1′ | P2′ | … | Pn′)
//
// where Pi′ is a SECOND fragment for member Pi replaying Pi's particle
// identifiers, exactly as the unfolding of a numeric occurrence range replays
// them (addParticle). The star carries the group's ·first· set and the "some
// other member may still be taken here" edges; the primed alternation carries the
// group's ·last· set, so the state an enclosing sequence appends a successor onto
// is the state reached by taking the LAST member, not the state reached by taking
// any member.
//
// Splitting the two is what §3.8.4.1.3 requires and a bare star conflates. L(M)
// for an all group is S1 × … × Sn with each Si in L(Pi), so the group is complete
// only once every member has contributed, a member contributing the empty
// sequence only when it is ·emptiable·. A successor is therefore live after a
// member only in the state where every OTHER non-emptiable member has already
// been taken — a fact about the SUBSET taken so far, which no per-member flag can
// carry, since with k non-emptiable members there are k orders and each ends on a
// different one. The primed alternation is that state: reaching it is taking one
// more member, and beside the successor only a residual is live there.
//
// The residual drawn onto the primed ·last· positions is the STAR pass's, not the
// primed pass's, because the members it speaks for are the ones taken before the
// last: an emptiable member never taken is offered its star ·first· set, and a
// member taken to one of its star ·last· positions its own continuation from
// there. A member's primed copy needs no entry here — its own continuations are
// already in its own follow sets, and its ·first· set is the same source particle
// as its star copy, which competes exempts by identity.
//
// WITHIN the star the transcription is exactly equivalent to the interleave for
// deciding ·compete·, and that much is unchanged: the two agree on which
// positions are live at the start (all members', in both) and on which may follow
// a given one (in the interleave, any member other than the one just taken; in
// the star, any member including it), so the star's extra pairs are exactly the
// self-pairs of one member, which competes rejects by particle identity before
// ever consulting the terms. Nesting composes, since an <all> reached through a
// <group ref> inside another <all> (which 1.1 permits at {min occurs} = {max
// occurs} = 1) contributes its own first/last sets to the outer union like any
// other member, and its own residual to the outer one through the follow sets of
// its ·last· positions.
//
// The sequences the fragment ACCEPTS are a STRICT superset of the interleave:
// all(a, b) admits a a, and a alone. That is harmless for the pairs one state
// holds, the one fact attributionPolicy renders it for (check), and wrong for
// any consumer that reads the language or the subsets a ·path· prefix reaches,
// which is why contentrestricts.go's containment walk and checkPaths render an
// <all> through addInterleave instead (languagePolicy) and never read this
// fragment (#1930).
//
// Emptiability is NOT taken from the star form, which would always be emptiable:
// an <all> accepts the empty sequence exactly when every member does, and that is
// what an enclosing sequence needs in order not to invent competition sets.
func (b *automaton) addAll(g ModelGroup) ([]int, []int, bool, error) {
	startID := b.nextParticleID
	taken, err := b.addAllMembers(g)
	if err != nil {
		return nil, nil, false, err
	}
	b.nextParticleID = startID
	final, err := b.addAllMembers(g)
	if err != nil {
		return nil, nil, false, err
	}
	first := mergePositions(taken.first, final.first)
	for _, q := range taken.last {
		b.addFollow(q, first)
	}
	for _, q := range final.last {
		b.addFollow(q, taken.residual)
	}
	return first, final.last, taken.emptiable, nil
}

// sequenceFragment accumulates the ·first·/·last·/emptiable sets of a
// concatenation under construction. It is the one place the concatenation rule
// lives (STYLE T4): both <sequence> and the unfolding of a numeric occurrence
// range are concatenations.
type sequenceFragment struct {
	first     []int
	last      []int
	emptiable bool
}

// appendToSequence concatenates one member, whose fragment is (f, l, e), onto the
// prefix accumulated so far: every position that could end the prefix may now be
// followed by every position that could start the member, the member's first
// positions join the whole sequence's first set only while the prefix can be
// empty, and the member's last positions replace the prefix's unless the member
// itself can be skipped.
func (b *automaton) appendToSequence(seq *sequenceFragment, f, l []int, e bool) {
	if seq.emptiable {
		seq.first = mergePositions(seq.first, f)
	}
	for _, q := range seq.last {
		b.addFollow(q, f)
	}
	carried := seq.last
	seq.last = l
	if e {
		seq.last = mergePositions(l, carried)
	}
	seq.emptiable = seq.emptiable && e
}

// addFollow records that every position in targets may follow position i.
func (b *automaton) addFollow(i int, targets []int) {
	b.follow[i] = mergePositions(b.follow[i], targets)
}

// mergePositions returns the ascending union of two ascending position sets. The
// result is a fresh slice, so neither input is aliased into the automaton — a
// ·first· set handed to addFollow must not become the follow set itself.
func mergePositions(a, b []int) []int {
	if len(a) == 0 {
		return append([]int(nil), b...)
	}
	if len(b) == 0 {
		return append([]int(nil), a...)
	}
	out := make([]int, 0, len(a)+len(b))
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i] < b[j] {
			out = append(out, a[i])
			i++
			continue
		}
		if b[j] < a[i] {
			out = append(out, b[j])
			j++
			continue
		}
		out = append(out, a[i])
		i++
		j++
	}
	out = append(out, a[i:]...)
	return append(out, b[j:]...)
}
