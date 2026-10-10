package xsd

import (
	"slices"
	"strconv"

	"github.com/kud360/goxsd8/xsderr"
)

// This file decides Content type restricts (Complex Content) (Structures
// §3.4.6.4, cos-content-act-restrict), the delegate of
// derivation-ok-restriction clause 2.4.2 (complexderivation.go). Both of the
// constraint's conditions are decided here:
//
//	1 Every sequence of element information items ·locally valid· with respect
//	  to R is also ·locally valid· with respect to B.
//	2 [ctr-child-type-subsumption] For all such sequences ES, for all elements E
//	  in ES, B's ·default binding· for E ·subsumes· that defined by R.
//
// Clause 1 is stated extensionally — language containment L(R) ⊆ L(B), with no
// syntactic recipe — and clause 2 quantifies over the same sequences, so both
// reduce to one walk of the PRODUCT of the two content models' position
// automata. The automata are particleattribution.go's construction, reused whole
// rather than forked (STYLE T4): the same addParticle, the same first/follow/last
// sets cos-nonambig is decided over. Two things are chosen differently, and this
// file supplies both as languagePolicy: the unfolding of a numeric occurrence
// range (unfoldExactly) and the rendering of an <all> group (addInterleave).
//
// WHY THE UNFOLDING IS NOT SHARED. maxMandatoryCopies / maxOptionalCopies bound a
// range to two copies of each kind, which is verdict-preserving for cos-nonambig
// — whose subject is which particle- IDENTIFIER sets are live in one state, and
// two copies realize every such set — and is NOT verdict-preserving here, because
// as a statement about LANGUAGE the bound rewrites the range itself: e{3,6} would
// read as e{2,4} and e{0,100} as e{0,2}. That rewrite is monotone in neither
// direction, and both directions are reachable on a bare single element particle:
// e{3,6} under e{0,100} is a VALID restriction whose fourth R-copy would find no
// live B-position and be rejected, and e{5,5} under e{3,3} is an INVALID one both
// of whose sides would collapse to two mandatory copies and become
// indistinguishable. Nothing licenses either outcome — clause 1 is pure set
// containment naming no algorithm, Appendix J's unfolding guidance is scoped by
// its own text to cos-nonambig, and §3.4.6.3's "may provisionally accept the
// derivation" (xmlschema11-1.md:2041) licenses ACCEPTANCE of an undecided case,
// never a rejection. So the constants are not raised, which would only move the
// same two thresholds, but replaced for this consumer: unfoldExactly emits {max
// occurs} copies of a bounded range and {min occurs} copies plus a loop-back for
// an unbounded one, so the automaton accepts exactly L and this walk DECIDES
// containment over the declared {min occurs}/{max occurs} instead of over a
// truncated unfolding (#501).
//
// WHY THE <all> RENDERING IS NOT SHARED. addAll renders all(P1…Pn) as a star over
// its members followed by a primed replay of one of them, which is exact for
// ·compete· and a strict superset of the interleave §3.8.4.1.3 defines, so it
// would make an ·all· in R admit sequences R does not (a false reject) and one in
// B admit sequences B does not (a false accept). addInterleave builds the
// interleave itself, so an <all> on either side is decided exactly, as
// derivation-ok-restriction clause 2.4.2 asks of cos-content-act-restrict
// (#1930).
//
// The exact unfolding does not step outside what cos-nonambig validated.
// maxMandatoryCopies' own argument is that copies past the second realize no
// identifier set the first two do not, so every state of the exactly-unfolded
// automaton offers an identifier set already present in the bounded one that
// Phase C accepted: the "no two competing particles in one state" premise the
// determinism note below rests on carries over unchanged. That argument is the
// unfolding's alone: addInterleave's states are not claimed to be among the ones
// Phase C checked over addAll's star, and nothing needs them to be. The premise
// is not load-bearing for soundness in any case — R's live positions are ALL
// explored and B is subset-constructed, which is the standard containment check
// over two NFAs, and neither half needs a deterministic automaton.
//
// WHY THE WALK NEEDS NO BACKTRACKING, AND WHY B IS STILL DETERMINIZED.
// cos-nonambig (Phase C, which runs before this Phase D check) has already
// rejected any content model with two competing ·element particles· or two
// competing ·wildcard particles· in one state, so no state offers a choice
// between two DISTINCT particles for one expanded name and nothing backtracks
// (PRINCIPLES 14). It does NOT make the position automaton deterministic:
// ·compete· relates two particles, so the unfolded COPIES of one particle are
// exempt by particle identity (particleattribution.go's competes) and several of
// them are routinely live at once — every unbounded {max occurs} loops the last
// copy back onto itself beside its successors. R is walked position by position,
// which is sound because any copy R can take is a sequence R admits; B is walked
// as a SET of positions, the standard subset construction, because picking one
// member would commit to a continuation another member allows.
//
// A matched set is HETEROGENEOUS in general, and nothing here may read one member
// as a representative of the rest. Two shapes put different {term}s into one set:
// copies of one particle, exempt from ·compete· by particle identity; and — since
// 1.1 stopped forbidding that pairing (Appendix G.1.3) — an ·element particle·
// beside a ·wildcard particle· both admitting one name, which cos-nonambig
// deliberately leaves live together. Members of the second shape carry genuinely
// different, and possibly disagreeing, ·default bindings·. Both conditions are
// therefore decided over the WHOLE set: clause 1 continues into the union of
// every member's ·follow· set, and clause 2 is EXISTENTIAL (someBindingSubsumes)
// — it passes when SOME member's binding ·subsumes· R's, which is this file's
// fail-open direction, not a claim that the members agree.
//
// The walk terminates on its visited set: positions(R) is finite because a
// content model past maxContentPositions is never unfolded at all, and the B-sets
// are drawn from a finite powerset, with maxProductStates as a hard ceiling on
// top. That set is a walk-scoped graph-reachability guard over two automata
// already built, not a component-resolution cycle check, so PRINCIPLES 9 /
// STYLE D4 are untouched.
//
// WHAT cos-ns-subset IS DOING HERE. §3.10.6.2's Wildcard Subset relation is NOT
// cited by cos-content-act-restrict — clause 1 names no algorithm at all. It is
// this reduction's own per-transition compatibility test for the
// wildcard-versus-wildcard case, where "every name R's wildcard admits, B's
// wildcard admits too" is exactly what a wildcard subset says. Do not read its
// presence as a spec cross-reference.
//
// DIRECTION OF EVERY APPROXIMATION. Each place this walk cannot decide a fact
// exactly, it resolves so that R looks SMALLER or B looks LARGER, i.e. towards
// accepting the derivation: clause 2.4.2 is one conjunct of
// derivation-ok-restriction clause 2's disjunction, so a missed rejection is
// fail-open and a spurious one would false-reject a valid schema. The direction
// rests on that asymmetry alone and claims no spec licence for the file as a
// whole: each branch that provisionally accepts states its own standing where it
// does so.

// contentAutomaton is one content model's position automaton together with the
// three fragment facts addParticle returns for its root particle. The automaton
// is embedded rather than copied: positions and follow are read, never extended,
// once construction is done.
type contentAutomaton struct {
	*automaton
	first     []int
	last      []int
	emptiable bool
}

// contentAutomatonOf builds the position automaton of one Content Type's
// {particle} under languagePolicy — every numeric occurrence range unfolded
// exactly and every <all> group built as its interleave — so the automaton
// accepts exactly the sequences ·locally valid· with respect to
// that content model and the walk below decides containment rather than
// approximating it. Nothing here bounds the construction: the caller must have
// cleared the content model through unfoldedPositions first.
//
// It is finalize-scoped and memoized nowhere (STYLE D3), exactly as
// checkContentModelsUnambiguous builds and discards one per content model.
func (s *Schema) contentAutomatonOf(c ElementContent) (contentAutomaton, error) {
	b := &automaton{s: s, policy: languagePolicy}
	first, last, emptiable, err := b.addParticle(c.Particle)
	if err != nil {
		return contentAutomaton{}, err
	}
	return contentAutomaton{automaton: b, first: first, last: last, emptiable: emptiable}, nil
}

// languagePolicy is cos-content-act-restrict's constructionPolicy: both of its
// renderings preserve the LANGUAGE of the content model, which is the only fact a
// containment walk reads.
var languagePolicy = constructionPolicy{unfold: unfoldExactly, all: (*automaton).addInterleave}

// unfoldExactly is languagePolicy's unfolding of a numeric occurrence range: the
// one that preserves the LANGUAGE of the particle.
//
// It is unfoldCopies (particleattribution.go) with the two copy caps removed, and
// removing them is precisely what makes it language-exact. A bounded {m,n} emits
// n copies of which the first m are mandatory, which addParticle concatenates
// into L^m·(L∪ε)^(n-m) — the union of L^j for j from m to n, i.e. the range
// itself. An unbounded {m,unbounded} emits max(m,1) copies, all mandatory when
// m ≥ 1, with a loop-back edge on the last, which is L^m·L*. A vacuous {0,0}
// emits nothing, exactly as before. No copy count is truncated, so no declared
// occurrence range is silently rewritten into a different one.
//
// This is an implementation decision and is cited as one: §3.4.6.4 clause 1
// states containment extensionally and names no decision procedure, no other
// clause supplies one, and the only unfolding guidance the local specs carry
// (Appendix J) is scoped by its own text to cos-nonambig. What keeps the
// construction finite is therefore not a bound on the copy count — which would
// have to be a bound on the LANGUAGE, and there is no sound one — but
// maxContentPositions, a bound on the whole content model that abandons it
// wholesale instead of truncating it into a verdict.
func unfoldExactly(o Occurs) (copies, mandatory int, loop bool) {
	bound, bounded := o.Max()
	if bounded && bound == 0 {
		return 0, 0, false
	}
	if !bounded {
		return max(o.Min(), 1), o.Min(), true
	}
	return bound, o.Min(), false
}

// maxContentPositions bounds how many positions one content model's exact
// unfolding may emit before the derivation is left undecided and provisionally
// accepted. The exact unfolding is linear in {max occurs}, and §3.9.2 types
// maxOccurs as a nonNegativeInteger, so maxOccurs="4294967295" is a schema a
// processor may be handed and must not try to materialize.
//
// It bounds the SIZE of the automaton — its position count — and never a
// verdict, and that second half is the whole difference between it and the
// two-copy bound it replaced: a content model within it is unfolded exactly and
// DECIDED exactly, while one beyond it is abandoned whole, never truncated into
// an answer. Abandoning is fail-open in the direction this file's header fixes;
// truncating was monotone in neither.
//
// It is NOT a bound on time, and must not be read as one. Position count and
// cost are related through the SHAPE of the model, not through a constant: a
// 2970-position model out of the W3C suite costs nothing measurable, while a
// 200-position all-optional range on both sides cost 8.3 seconds before the
// per-source-particle collapse below went in (#501). What actually holds the
// walk's cost down is that collapse — per transition it does work proportional
// to the number of distinct SOURCE PARTICLES live in a state rather than to the
// unfolded copies of them (liveSet, subsetTable, contentModelRestricts) — with
// maxProductStates bounding the states on top of it.
//
// The constant is MEASURED, on the same footing as maxProductStates, and unlike
// that one it is NOT inert on the W3C suite. Instrumenting unfoldedPositions and
// running the full suite — this check is reached 1538 times, twice per candidate
// pair — recorded 1532 content models at 2970 positions or fewer, and six beyond
// the ceiling: one at 30001, one at 999999, three at 9999999, and one past
// 16777216. Those six carry a maxOccurs in the thousands to millions and are the
// shapes an exact unfolding cannot be asked to materialize. So the ceiling sits
// 1.38× above the widest model the suite DECIDES (4096 over 2970) and a factor
// of 7 below the narrowest it DECLINES (30001 over 4096). The gap is narrow on
// the lower side, and that is exactly why the constant is not LOWERED to control
// cost: every step down starts declining models the suite decides today, and
// each decline is a verdict lost.
//
// What it costs at its own value is stated rather than asserted. The worst shape
// it admits is a wide all-optional range on both sides; measured through
// cRestricts (a whole Finalize, one machine, one run, e{0,n} under e{0,n}):
//
//	n =  200    50 ms      48 MB allocated    11 MB peak heap
//	n =  500   288 ms     396 MB allocated    15 MB peak heap
//	n = 1024   1.4 s      3.0 GB allocated    27 MB peak heap
//	n = 4096    78 s      180 GB allocated   287 MB peak heap
//
// Peak heap is modest throughout; the wall clock and the allocation CHURN are
// what grow. At n = 1024 a CPU profile puts 60% of that in addFollow, inside the
// SHARED automaton construction (mergePositions recopies a follow set per edge,
// and an all-optional run makes every position follow every later one), against
// 8% in this file's walk. The residual at the ceiling is therefore a
// construction cost, not a containment-walk cost, and it is retired by a cheaper
// follow-set representation there or by a procedure that decides containment
// without materializing an automaton per occurrence — not by moving this
// constant, which the suite pins from below.
const maxContentPositions = 4096

// unfoldedPositions reports how many positions the exact unfolding of one
// particle contributes to a content automaton, saturating one past
// maxContentPositions so a maxOccurs of 4294967295 is answered by arithmetic
// rather than by construction. It counts exactly what addParticle emits under
// unfoldExactly — one fragment per copy, each fragment holding the {term}'s own
// positions — so the count is the automaton's size, not an estimate of it.
//
// The walk follows <group ref> and stops at <element ref>, exactly as addTerm
// does, and carries no visited set for the same reason addTerm carries none:
// Phase B's checkModelGroupsAcyclic (mg-props-correct clause 2) has already
// rejected a circular group, so PRINCIPLES 9 / STYLE D4 are untouched.
//
// It counts rather than builds because the count has to be known BEFORE the
// automaton exists — an automaton already built past the ceiling has already cost
// what the ceiling exists to refuse — so this is a second traversal of the same
// tree addParticle/addTerm/addModelGroup traverse, and it must stay in step with
// them: a term kind that starts emitting a different number of positions has to
// be reflected here in the same commit, or the ceiling stops bounding what it
// claims to bound. The three arms below mirror those three functions one for one
// so the correspondence is checkable by reading them side by side.
func (s *Schema) unfoldedPositions(p Particle) int {
	copies, _, _ := unfoldExactly(p.Occurs())
	if copies == 0 {
		return 0 // a vacuous {0,0} particle emits no fragment at all
	}
	inner := s.termPositions(p.Term())
	if inner == 0 {
		return 0
	}
	if copies > (maxContentPositions+1)/inner {
		return maxContentPositions + 1
	}
	return copies * inner
}

// termPositions is unfoldedPositions for a particle's {term}: the positions ONE
// copy of the term's fragment emits. An unresolvable <group ref> contributes
// nothing, as addTerm's own unreachable arm does; an <element ref> contributes
// the one position addLeaf emits for it, counted whether or not it resolves,
// since over-counting can only send this content model to the fail-open ceiling.
func (s *Schema) termPositions(t TermOrRef) int {
	switch t := t.(type) {
	case ResolvedTerm:
		return s.resolvedTermPositions(t.Term)
	case ElementDeclarationRef:
		return 1
	case ModelGroupRef:
		mgd, ok := s.modelGroupIndex[t.Name]
		if !ok {
			return 0
		}
		return s.modelGroupPositions(mgd.ModelGroup())
	default:
		panic("xsd: termPositions: non-exhaustive TermOrRef switch")
	}
}

// resolvedTermPositions is termPositions over the sealed Term sum: one position
// for either kind of leaf, and the sum of its members for a model group.
func (s *Schema) resolvedTermPositions(t Term) int {
	switch t := t.(type) {
	case ElementDeclaration:
		return 1
	case ModelGroup:
		return s.modelGroupPositions(t)
	case Wildcard:
		return 1
	default:
		panic("xsd: resolvedTermPositions: non-exhaustive Term switch")
	}
}

// modelGroupPositions is termPositions for a model group, saturating as soon as
// the running total passes the ceiling so a wide group under a wide range is not
// summed to completion. Particles are walked in document order (STYLE D2).
//
// <sequence> and <choice> contribute each member's fragment exactly once —
// addSequence and addChoice differ in the edges they draw, never in the positions
// they emit — so they count the sum of their members. An <all> is
// interleavePositions'.
func (s *Schema) modelGroupPositions(g ModelGroup) int {
	if g.Compositor() == CompositorAll {
		return s.interleavePositions(g)
	}
	total := 0
	for _, p := range g.particles {
		total += s.unfoldedPositions(p)
		if total > maxContentPositions {
			return maxContentPositions + 1
		}
	}
	return total
}

// interleavePositions is modelGroupPositions for an <all> group: an upper bound
// on the positions addInterleave emits, saturating one past maxContentPositions.
//
// addInterleave emits one position per pair (v, i) of a reachable member-state
// vector v and a member i whose own state v[i] is a position of member i's
// fragment. With Pi the positions of member i's fragment, member i has 1 + Pi
// states (not yet started, or just consumed one of its positions), so there are
// at most Pi × Π(j≠i)(1 + Pj) such pairs for each i, and the count is their sum.
// It is folded member by member — a prefix with S states and N positions extended
// by a member of P positions has S(1 + P) states and N(1 + P) + P·S positions —
// and every intermediate is a product of two factors at most one past the
// ceiling, since each non-start vector is entered by at least one position and
// so S ≤ N + 1.
//
// It is exact when every position of every member's fragment is reachable from
// that fragment's start, which every shape but an unreachable tail behind an empty
// <choice> satisfies, and an over-count there only sends the model to the
// fail-open ceiling sooner.
func (s *Schema) interleavePositions(g ModelGroup) int {
	states, total := 1, 0
	for _, p := range g.particles {
		n := s.unfoldedPositions(p)
		total = total*(1+n) + n*states
		states *= 1 + n
		if total > maxContentPositions {
			return maxContentPositions + 1
		}
	}
	return total
}

// addInterleave is languagePolicy's rendering of an <all> group: the exact
// interleave §3.8.4.1.3 defines, L(all(P1…Pn)) = S1 × … × Sn over Si ∈ L(Pi),
// where × interleaves (partitions a sequence into subsequences, not contiguous
// blocks).
//
// Each member is built into its own fragment (a contentAutomaton over a scratch
// automaton), and the interleave is the product of those fragments: a state is
// a vector v holding one state per member — startState, or the member position
// just consumed — and an item moves exactly one member i one step, from v[i] to a
// position live there. The group accepts where every member does, so a member
// contributes the empty sequence only when it is emptiable. Nesting composes: a
// member that is itself an <all> (cos-all-limited clause 2) is built by this
// same function inside its own fragment.
//
// The emitted automaton is a position automaton like every other fragment, so
// the enclosing construction composes with it unchanged: position (v, i) is
// entered only by consuming member i's position v[i], so it carries that
// position's particle identifier and {term}, and its ·follow· set is every (w, j)
// one step from v. Member fragments allocate their particle identifiers from this
// automaton's allocator, in member order, so the identifier -> {term} function
// the walk's per-source-particle collapse relies on holds across the whole
// automaton, and a copy made by an enclosing addParticle replays them exactly.
//
// Vectors are enumerated breadth-first from the all-startState vector, members
// in document order and a member's live positions ascending, so the position
// numbering depends only on the content model (STYLE D1). The two maps are
// lookups only and are never ranged (STYLE D2). The enumeration is finite
// because the member fragments are, and its size is what interleavePositions
// bounds: contentTypeRestricts refuses a content model past maxContentPositions
// before this is ever reached.
func (b *automaton) addInterleave(g ModelGroup) ([]int, []int, bool, error) {
	members := make([]contentAutomaton, 0, len(g.particles))
	for _, p := range g.particles {
		m := &automaton{s: b.s, policy: b.policy, nextParticleID: b.nextParticleID}
		first, last, emptiable, err := m.addParticle(p)
		if err != nil {
			return nil, nil, false, err
		}
		b.nextParticleID = m.nextParticleID
		members = append(members, contentAutomaton{automaton: m, first: first, last: last, emptiable: emptiable})
	}
	start := make([]int, len(members))
	for i := range start {
		start[i] = startState
	}
	vectors := [][]int{start}
	vectorIDs := map[string]int{positionsKey(start): 0}
	entry := map[[2]int]int{} // (vector id, member) -> position in b
	var entered []int         // position - base -> the vector id it enters
	base := len(b.positions)
	var successors [][]int // vector id -> ascending positions one step from it
	for u := 0; u < len(vectors); u++ {
		var next []int
		for i, m := range members {
			for _, q := range m.live(vectors[u][i]) {
				w := slices.Clone(vectors[u])
				w[i] = q
				key := positionsKey(w)
				wid, seen := vectorIDs[key]
				if !seen {
					wid = len(vectors)
					vectorIDs[key] = wid
					vectors = append(vectors, w)
				}
				pos, emitted := entry[[2]int{wid, i}]
				if !emitted {
					pos = len(b.positions)
					entry[[2]int{wid, i}] = pos
					b.positions = append(b.positions, m.positions[q])
					b.follow = append(b.follow, nil)
					entered = append(entered, wid)
				}
				next = append(next, pos)
			}
		}
		slices.Sort(next)
		successors = append(successors, next)
	}
	var last []int
	for k, wid := range entered {
		b.addFollow(base+k, successors[wid])
		if interleaveAccepts(members, vectors[wid]) {
			last = append(last, base+k)
		}
	}
	return slices.Clone(successors[0]), last, interleaveAccepts(members, start), nil
}

// interleaveAccepts reports whether a member-state vector ends a sequence the
// interleave accepts: every member's own state accepts.
func interleaveAccepts(members []contentAutomaton, v []int) bool {
	for i, m := range members {
		if !m.accepting(v[i]) {
			return false
		}
	}
	return true
}

// startState is the automaton state before any element has been consumed. Every
// other state is "just consumed position i" and is named by that i, so a
// negative sentinel is the one value that cannot collide with a position index.
const startState = -1

// live returns the positions that may be consumed next from a state, in
// ascending index order: the ·first· set at the start, the ·follow· set of the
// position just consumed otherwise.
func (a contentAutomaton) live(state int) []int {
	if state == startState {
		return a.first
	}
	return a.follow[state]
}

// accepting reports whether a state ends a ·locally valid· sequence: the start
// state does when the whole particle accepts the empty sequence, any other when
// its position is in the ·last· set.
func (a contentAutomaton) accepting(state int) bool {
	if state == startState {
		return a.emptiable
	}
	return slices.Contains(a.last, state)
}

// acceptingAny reports whether any member of a B-state set accepts, which is
// what the subset construction makes of "B's run may end here".
func (a contentAutomaton) acceptingAny(states []int) bool {
	for _, st := range states {
		if a.accepting(st) {
			return true
		}
	}
	return false
}

// liveIn returns the ascending union of the positions live in every member of a
// B-state set: the alphabet B may consume next, taken over all runs that reach
// this set.
//
// The union is MARKED rather than merged pairwise. mergePositions returns a
// fresh slice per operand, so folding n ·follow· sets through it recopies the
// accumulator n times — quadratic in the set's width on top of the members it
// actually reads, and this is the walk's per-state entry point. Marking a
// scratch []bool costs one write per member position and one scan of the
// automaton's positions, and the scan is what makes the result ascending without
// sorting (STYLE D2). The scratch is call-scoped: nothing derivable outlives the
// call (STYLE D3).
func (a contentAutomaton) liveIn(states []int) []int {
	marked := make([]bool, len(a.positions))
	total := 0
	for _, st := range states {
		for _, q := range a.live(st) {
			if marked[q] {
				continue
			}
			marked[q] = true
			total++
		}
	}
	live := make([]int, 0, total)
	for q, ok := range marked {
		if ok {
			live = append(live, q)
		}
	}
	return live
}

// liveSet is what B may consume next from one B-state set — liveIn's ascending
// union — together with the partition of those positions by SOURCE PARTICLE.
//
// The partition exists for cost, and it is exact rather than an approximation.
// positionAdmits reads nothing off a position but its {term} (and
// coveringWildcardUnion nothing but its {namespace constraint}), and every
// unfolded copy of one particle replays that particle's identifier over the same
// source subtree, so within one automaton a particleID determines the {term}
// (addParticle's allocator reset — the same property competes relies on). Every
// member of a group therefore gives positionAdmits the same answer, and asking
// one member answers for all of them.
//
// The STATE SET is not collapsed, and must not be: copies of one particle carry
// DIFFERENT ·follow· sets (copy k of e{0,n} is followed by copies k+1…n), so
// dropping copies from the matched set would drop continuations B allows. What
// collapses is only the number of positionAdmits CALLS per transition; matched
// still names every live position the group covers, in ascending order.
//
// positions is ascending; group[i] is the group index of positions[i]; reps
// holds one representative position per group, in the order the groups were
// first seen scanning positions. The map inside liveGroups is a lookup only —
// every output order here comes from the ascending scan (STYLE D2).
type liveSet struct {
	positions []int
	group     []int
	reps      []int
}

// liveGroups is liveIn with that partition computed: one pass over the union,
// which is the only pass that consults a particle identifier.
func (a contentAutomaton) liveGroups(states []int) liveSet {
	positions := a.liveIn(states)
	set := liveSet{positions: positions, group: make([]int, len(positions))}
	index := make(map[int]int, len(positions))
	for i, q := range positions {
		id := a.positions[q].particleID
		g, seen := index[id]
		if !seen {
			g = len(set.reps)
			index[id] = g
			set.reps = append(set.reps, q)
		}
		set.group[i] = g
	}
	return set
}

// positionsOf returns, ascending, every live position whose group in[g] marks:
// a whole source particle's copies, never some of them.
func (l liveSet) positionsOf(in []bool) []int {
	var out []int
	for i, q := range l.positions {
		if in[l.group[i]] {
			out = append(out, q)
		}
	}
	return out
}

// productState is one state of the product walk: a single state of R paired with
// the SET of B states B's runs may be in, the set named by its subsetTable
// identifier. Naming it rather than carrying it is what makes the state
// comparable, so the visited set keys on a pair of ints and no string is built
// per state (see subsetTable). The visited set's keys carry R's futureClasses
// class in r where the queue carries R's state.
type productState struct {
	r int
	b int
}

// futureClasses numbers R's states by their future, indexed by state + 1 so
// startState is entry 0: two states share a number exactly when they have the
// same live positions and the same acceptance, which is everything the walk
// reads off an R-state after entering it. Numbers are assigned in ascending
// state order and the map is a lookup only (STYLE D2).
func (a contentAutomaton) futureClasses() []int {
	classes := make([]int, len(a.positions)+1)
	ids := map[string]int{}
	for st := startState; st < len(a.positions); st++ {
		key := positionsKey(a.live(st))
		if a.accepting(st) {
			key = "accept " + key
		}
		id, ok := ids[key]
		if !ok {
			id = len(ids)
			ids[key] = id
		}
		classes[st+1] = id
	}
	return classes
}

// subsetTable interns the B-state sets the walk reaches: equal sets share an
// identifier, so a product state is a pair of ints and the visited set is a
// plain map over that pair.
//
// It exists because the identity of a product state has to be tested once per
// live R-position per state, while the SETS those states pair with are far
// fewer — every R-position of one source particle transitions into the SAME sets
// (contentModelRestricts computes them once), so interning turns a per-transition
// canonical-string build, linear in the set's width, into one build per distinct
// set reached.
//
// The canonical form is the ascending members, exactly as before: identity is
// still set equality and nothing about the walk's verdict depends on the
// numbering. Both the id map and the visited map keyed on these ids are lookups
// only, never ranged (STYLE D2).
//
// The two fields are the two directions of ONE bijection, not a fact stored
// twice (STYLE D3): the walk asks "have I seen this set" on the way in and "what
// were that identifier's members" on the way out, and recovering either from the
// other would mean rebuilding a key or scanning the table.
type subsetTable struct {
	ids  map[string]int
	sets [][]int
}

// newSubsetTable returns an empty table.
func newSubsetTable() *subsetTable {
	return &subsetTable{ids: map[string]int{}}
}

// intern returns the identifier of an ascending B-state set, assigning the next
// one the first time that set is reached.
func (t *subsetTable) intern(set []int) int {
	key := positionsKey(set)
	if id, ok := t.ids[key]; ok {
		return id
	}
	id := len(t.sets)
	t.ids[key] = id
	t.sets = append(t.sets, set)
	return id
}

// set returns the members an identifier names, ascending.
func (t *subsetTable) set(id int) []int {
	return t.sets[id]
}

// positionsKey is the canonical string identity of an ascending position set.
// It is a lookup key only — the map it indexes is never ranged (STYLE D2).
func positionsKey(states []int) string {
	var buf []byte
	for i, q := range states {
		if i > 0 {
			buf = append(buf, ' ')
		}
		buf = strconv.AppendInt(buf, int64(q), 10)
	}
	return string(buf)
}

// maxProductStates bounds how many product states the walk visits before giving
// up and provisionally accepting. The subset construction's state space is a
// powerset in the worst case, so a bound keeps a pathological content model from
// turning schema assembly into an exponential walk. It is a ceiling on WORK,
// never on the verdict of a walk that finishes.
//
// The constant is MEASURED headroom rather than an unexamined guess, and the
// headroom has narrowed. Three counters — entries into contentModelRestricts,
// the giveup branch below, and every insertion into the visited set so the
// high-water mark comes from walks that finish — run over the full W3C suite at
// the same submodule record this series, one point per measurement:
//
//   - 2026-08-04 (#282): walkEntries=688 ceilingHits=0 maxVisited=15
//   - 2026-09-19 (#499): walkEntries=1987 ceilingHits=0 maxVisited=1002
//   - 2026-09-29 (#1609, main afa784d): walkEntries=1985 ceilingHits=0 maxVisited=1002
//   - 2026-09-29 (#1609, main 4ef04a9): walkEntries=1740 ceilingHits=0 maxVisited=1002
//   - 2026-09-30 (#1609 with #1939, main cf8a9f2 plus #1939's per-part split):
//     walkEntries=1687 ceilingHits=0 maxVisited=1188, in a different unit: since
//     #1930 the visited set keys on R's future class, so maxVisited counts (R
//     future class, B-set) states, not (R position, B-set) ones, and is not
//     like-for-like with 1002. #1930's build without that key reached 2273 on one
//     ·all·:·all· walk (saxonData All all221-224); it never landed, so it fired
//     nothing, but it is the headroom the quotient buys.
//   - 2026-09-30 (#1609 with #1954, wip/issue-1954 at 6a68d1a plus #1954's
//     element-name split): walkEntries=1674 ceilingHits=0 maxVisited=1188, in
//     the future-class unit of the point above.
//   - 2026-10-01 (#1609 with #2000, wip/issue-2000 at 378418a, main f1353a0
//     plus #2000's local-particle guard): walkEntries=1673 ceilingHits=0
//     maxVisited=1188, in the future-class unit.
//
// Read both halves of that. No walk has ever reached the ceiling, so the bound
// is inert on every content model the suite contains and the incompleteness it
// guards is latent. But the deepest walk visits 1188 of the 4096 states it is
// allowed at the latest point (2026-10-01) — a factor of 3.4 below the ceiling
// where it was a factor of 273 on 2026-08-04 — and between the first two points
// maxVisited grew 66.8× while the walk entries grew only 2.9×, so the walks that
// reached this code went DEEPER rather than merely happening more often. From
// 2026-09-19 to 4ef04a9 maxVisited did not move and walkEntries only fell, to
// 1740, and it has fallen again since, to 1687, 1674 and 1673. A single future
// content model, not a wider population, is enough to cross. What drove either
// movement is not established here: each window holds lane-widening landings,
// and no causal claim is made from a correlation nobody checked.
//
// A margin that has moved that far between two measurements is not evidence for
// an unexamined constant, which is why the ruling at contentModelRestricts'
// giveup site names a re-measurement threshold instead of waiting for a breach.
const maxProductStates = 4096

// contentRestrictionScope names WHICH of cos-content-act-restrict's two
// conditions a caller is charging, because the two callers charge different
// rules over the same walk and the difference is a spec reading, not a tuning
// knob.
//
// It is a closed set of two, and every call site names its member literally, so
// the narrower reading is a cited decision at the site that takes it rather than
// an inherited default (STYLE T1).
type contentRestrictionScope int

const (
	// restrictsFully is clauses 1 and 2 together: derivation-ok-restriction
	// (§3.4.6.3) clause 2.4.2, which invokes cos-content-act-restrict whole.
	restrictsFully contentRestrictionScope = iota
	// restrictsLanguage is clause 1 alone — language containment, V(R) ⊆ V(B).
	// src-redefine (§4.2.4) clause 6.2.2 states only that "the {model group} …
	// accepts a subset of the element sequences accepted by that model group
	// definition in S2", cites §3.7.2 for the mapping and cos-content-act-restrict
	// for nothing; clause 2's ·default binding· subsumption is an extra condition
	// specific to complex-type restriction and charging it here would reject a
	// redefining group that retypes an element while accepting the same sequences
	// (redefinition.go).
	restrictsLanguage
)

// contentTypeRestricts is derivation-ok-restriction clause 2.4.2's delegate:
// whether T's {content type} ·restricts· B's as defined in Content type
// restricts (Complex Content) (§3.4.6.4, cos-content-act-restrict). scope selects
// how much of that constraint is charged (contentRestrictionScope): the clause
// 2.4.2 caller takes restrictsFully, src-redefine clause 6.2.2 restrictsLanguage.
//
// It answers a bool rather than an error because clause 2.4.2 sits inside
// derivation-ok-restriction clause 2, a DISJUNCTION: the failure that reaches a
// user is charged once, by checkRestrictionContentType, after every branch has
// declined. The two conditions of cos-content-act-restrict are therefore not
// reported separately; contentModelRestricts records which one failed only in
// the comments at its two rejection sites.
//
// An ·all· group on either side is DECIDED, like any other content model:
// languagePolicy builds it as its exact interleave (addInterleave). That is
// option (a) of §3.4.6.3's implementation-defined choice, "always detects
// violations of clause 2.4.2 by examination of the schema in isolation", which
// the provisional-acceptance sentence before it (xmlschema11-1.md:2041) permits
// and does not require (#1930).
//
// Three shapes are provisionally accepted rather than decided, each fail-open.
// The first is an assertion, the second leans on §3.4.6.3's licence only where
// R's top compositor is ·all· (its marker states that the rest carries none),
// and the third is a ruled resource approximation with the same split:
//
//   - a non-element {content type} on either side. 2.4.1 (restrictionVarietyPairOK)
//     has already established both are element-only or mixed before this is
//     reached, so this is an assertion of that precondition, not a case.
//   - a present {open content} on either side. B's open content ADMITS elements
//     the automaton below does not model, so ignoring it would shrink B and
//     manufacture clause-1 rejections; R's only widens R, which is harmless, but
//     the pair is skipped together so the reason stays one reason.
//   - a content model whose exact construction would exceed maxContentPositions
//     on either side. This one alone is a RESOURCE ceiling rather than a
//     modelling gap — it bounds the automaton's size, see maxContentPositions for
//     what that does and does not bound — and it is the only path on which a
//     declared occurrence range or an ·all· group does not reach the walk. The
//     marker at the branch itself states the ruling, the licence it has and
//     lacks, and what reopens it.
func (s *Schema) contentTypeRestricts(tct, bct ContentType, scope contentRestrictionScope) bool {
	rc, ok := tct.(ElementContent)
	if !ok {
		return true
	}
	bc, ok := bct.(ElementContent)
	if !ok {
		return true
	}
	if rc.OpenContent != nil || bc.OpenContent != nil {
		// GAP(xsd): a content type carrying an {open content} on either side is
		// provisionally accepted rather than decided, and that is a RULED
		// deferral rather than a fold in progress. The ruling below landed with
		// #413, now closed; #1374 owns what retires it.
		//
		// The sentence this arm once rested on is §3.4.6.3's, quoted whole: "It is
		// ·implementation-defined· whether a processor (a) always detects
		// violations of clause 2.4.2 by examination of the schema in isolation,
		// (b) detects them only when some element information item in the input
		// document is valid against T but not against T.{base type definition}, or
		// (c) sometimes detects such violations by examination of the schema in
		// isolation and sometimes not", followed by "In the latter case, the
		// circumstances in which the processor does one or the other are
		// ·implementation-dependent·" (xmlschema11-1.md:2043). It names clause
		// 2.4.2 and states no condition of its own, unlike the ·all·-scoped
		// sentence at :2041, and this arm's cases split on that sentence's
		// antecedent. Where T.{content type}.{particle}.{term}.{compositor} IS
		// all, the narrow :2041 sentence covers the case outright: its condition
		// (1) holds and its condition (2) is this very inability.
		// Everywhere else the arm has no licence, exactly as
		// contentModelRestricts' giveup site below has none: the (a)/(b)/(c)
		// sentence says WHEN a processor already inside :2041's antecedent detects
		// clause-2.4.2 violations and grants nothing outside it, and reading (c)
		// as a residual catch-all detached from its condition (1) is ruled out,
		// not merely unproved (#1378). It carries that site's other half too: (b)
		// and (c) describe processors that defer detection to instance time and
		// cross-check each instance against T.{base type definition}, and this arm
		// performs no such cross-check, so a schema accepted here can be
		// non-conforming with nothing left to say so.
		//
		// What retires the deferral is a construction, not a correction. §3.4.4.3
		// (cvc-complex-content) states ·locally valid· under a present {open
		// content} extensionally too, and per {mode}, with designated exclusions —
		// ONE in total for suffix, one per S2 element for interleave — rather than
		// a condition on every split point. Clause 2, suffix: S = S1 + S2, S1
		// ·valid· against {particle}, every element of S2 ·valid· against
		// {wildcard}, and — S2 non-empty — S1 + E without a ·path· in {particle}
		// for E the FIRST element of S2, S1 taken whole. Clause 3, interleave: S a
		// member of S1 × S2 under §3.8.4.1.3's interleave operator, S1 and S2 as
		// before, and for every E in S2, S3 + E without a ·path· where S3 is the
		// LONGEST prefix of S1 whose members precede E in S. Quantifying over
		// every prefix instead states a strictly stronger condition: (a?, b) under
		// a wildcard admitting a accepts b a, since S3 there is b and b a has no
		// ·path·, where the empty prefix plus a would reject it. An existential
		// decomposition with that per-E designated exclusion is a new automaton
		// carrying its own soundness argument — as addInterleave is for ·all· —
		// and no algorithm for it is given: §3.4.6.3/.4 state only the
		// containment, and Appendix J's construction guidance is scoped by its own
		// text to cos-nonambig.
		//
		// The arm is live rather than latent: since #230 the producer emits {open
		// content} from <openContent>/<defaultOpenContent> (§3.4.2.3.3 clauses
		// 5-6, parser/produce_complex.go), so an ordinary schema whose Open
		// Content is wider than its base's is accepted here.
		//
		// Fail-open for both readers of this true. checkRestrictionContentType
		// (complexderivation.go) charges derivation-ok-restriction and is the
		// clause-2.4.2 caller §3.4.6.3 names; checkExtensionTwoStepDerivable
		// (complexextension.go) charges cos-ct-extends clause 1.5, which
		// §3.4.6.3 does not reach, and carries its own marker for that. Each loses
		// a rejection it could have made and neither gains one.
		// checkModelGroupRedefinitions (redefinition.go) does not reach this arm
		// at all — modelGroupContent leaves {open content} ·absent·.
		return true
	}
	if s.unfoldedPositions(rc.Particle) > maxContentPositions || s.unfoldedPositions(bc.Particle) > maxContentPositions {
		// GAP(xsd): a content model whose exact unfolding would exceed
		// maxContentPositions is not unfolded at all, and the derivation is
		// provisionally accepted undecided. RULED permanent by #1378 (STYLE P3b)
		// rather than a fold in progress: that thread carries the ruling in full,
		// and it rests on the bounded-resource argument below rather than on a
		// spec licence. The landing this file's header records (#501, now closed)
		// introduced the ceiling; it is provenance and licensed nothing.
		//
		// The alternative is not a smaller automaton but a WRONG one: truncating
		// the copies of an occurrence range rewrites the range, which is monotone
		// in neither direction and false-rejects conforming schemas (see this
		// file's header). The ceiling therefore declines the whole question
		// instead of answering a different one.
		//
		// A spec licence covers one sub-population of this branch, and this marker
		// claims none beyond it. §3.4.6.3's provisional-acceptance sentence is
		// gated by a two-conjunct antecedent — "If (1) the type definition being
		// checked has T.{content type}.{particle}.{term}.{compositor} = all and (2)
		// an implementation is unable to determine by examination of the schema in
		// isolation whether or not clause 2.4.2 is satisfied". An ·all· group
		// reaches this line like any other content model (#1930), so an R whose
		// top compositor is all and whose interleave exceeds the ceiling meets
		// both conditions — (2) is this very inability — for its clause-2.4.2
		// caller. Everything else here is outside that antecedent: a sequence or
		// choice R, whatever B holds, and every restrictsLanguage or cos-ct-extends
		// reader, since the licence names clause 2.4.2. Neither §3.4.6.3 nor
		// §3.4.6.4 grants fail-open for that population anywhere. What the branch
		// does guarantee is the direction enumerated below: it abandons the WHOLE
		// walk rather than truncating one into a verdict, carrying the caveat
		// contentModelRestricts' giveup site states, that a schema accepted
		// provisionally with no runtime cross-check can be non-conforming with
		// nothing left to say so.
		//
		// Fail-open for all three readers of this true, each of which charges only
		// on false and reads nothing else out of it (STYLE P3a).
		// checkRestrictionContentType (complexderivation.go) calls in with
		// restrictsFully and charges derivation-ok-restriction clause 2.4.2.
		// checkExtensionTwoStepDerivable (complexextension.go) reaches the same
		// call through checkDerivationOKRestriction and re-charges the failure as
		// cos-ct-extends clause 1.5 at T's own position, a rule §3.4.6.3's licence
		// does not reach either. checkModelGroupRedefinitions (redefinition.go)
		// calls in with restrictsLanguage and charges src-redefine clause 6.2.2;
		// it reaches this branch, unlike the {open content} arm above, because
		// modelGroupContent's wrapper only leaves {open content} ·absent· and
		// constrains no compositor. Each loses a rejection it could have made and
		// none gains one.
		//
		// What makes the approximation permanent is measured cost against a
		// constant pinned from both sides. maxContentPositions' own doc records
		// what its present value already costs — 78 s and 180 GB allocated for one
		// n = 4096 pair — and records the suite pinning it from below at 2970
		// positions, so every step DOWN starts declining derivations decided
		// today. A step UP buys nothing: the narrowest model that lands here is
		// 30001 positions, seven times the ceiling, and the widest past 16777216.
		// Unlike this file's other ceilings the branch is REACHED — six of the
		// W3C suite's 1538 candidate content models, each carrying a maxOccurs in
		// the thousands to millions — so the incompleteness is live rather than
		// latent, and it is retired by a construction that decides containment
		// without materializing an automaton per occurrence, never by moving the
		// constant. Since ·all· groups reach this line (#1930), so does one more
		// shape: an interleave's size is the PRODUCT of its members' state counts,
		// and saxonData All all225 and all226 restrict to an ·all· of three
		// members ranging to 15, 15 and 22 occurrences, 16 × 16 × 23 member-state
		// vectors, past the ceiling. Both are suite-valid, so neither is the
		// second reopening finding below.
		//
		// Two findings reopen the ruling, and a case merely arriving here is
		// neither, since six already do. One is a containment procedure that
		// decides §3.4.6.4 clause 1 in bounded or sub-exponential space being
		// identified, which retires the ceiling outright. The other is a suite
		// case past the ceiling shown to be provisionally accepted where the exact
		// unfolding would REJECT, which prices the missed rejections instead of
		// leaving them merely counted.
		return true
	}
	r, err := s.contentAutomatonOf(rc)
	if err != nil {
		// addParticle's error slot is not reachable on a *Schema that exists:
		// every arm of addTerm/addResolvedTerm either returns nil or panics on a
		// broken sealed sum, and a dangling <element ref>/<group ref> was already
		// charged src-resolve by Phase A. Should a future term kind make it
		// reachable, provisionally accepting is the fail-open direction this
		// file's header argues, and like contentModelRestricts' giveup site it
		// claims no spec licence; the argument is not restated here. The error is
		// not silently discarded — it decides this verdict (STYLE S3).
		return true
	}
	b, err := s.contentAutomatonOf(bc)
	if err != nil {
		return true // same unreachable error slot, same direction
	}
	return s.contentModelRestricts(r, b, scope)
}

// contentModelRestricts walks the product of the two automata, deciding both
// conditions of cos-content-act-restrict in one pass — or clause 1 alone, when
// scope is restrictsLanguage. The walk is otherwise identical: clause 2 is a
// per-transition test over the same matched sets, never a separate traversal.
//
// States are drained FIFO from a slice seeded with the start pair, and each
// state's R-positions are visited in ascending index order, so the walk order
// depends only on the two content models. The visited set is a map used purely
// as a membership test — it is never ranged, and no iteration order reaches the
// verdict (STYLE D2).
//
// One transition is decided per SOURCE PARTICLE live in R, not per unfolded copy
// of it. Everything a transition depends on — which B-positions match
// (matchPositions), whether some matched binding subsumes (someBindingSubsumes)
// — is read off the R-position's {term}, and copies of one particle share it, so
// the copies of one particle live in a state all transition into the same
// B-sets: one, or one per name or part of a wildcard coveringWildcardUnion
// splits. The copies are still enqueued SEPARATELY, each as its own R-state:
// they carry different ·follow· sets, so the per-particle memo collapses only
// the recomputation of one answer per copy (#501), and the future quotient below
// does not merge them either. Iteration stays in ascending R-position order, so
// the walk order is unchanged by the memo. Every transition of a state is
// decided before any of its successors is enqueued, so a clause-1 or clause-2
// rejection anywhere in the state is reached before the maxProductStates giveup
// can provisionally accept it.
//
// An R wildcard position whose {namespace constraint} admits no namespace — an
// empty enumeration, namespace="" — takes no transition. No expanded name
// satisfies cvc-wildcard clause 1 (cvc-wildcard-namespace clause 3) against
// it, so no sequence locally valid with respect to R contains an item
// ·attributed· to it, and neither clause of cos-content-act-restrict
// quantifies over one. keywordSubsumes' exact refusals rely on this (#2546).
//
// The visited set keys on R's FUTURE rather than on its position
// (futureClasses): two R-states with the same ·follow· set and the same
// acceptance continue identically against any B-set, so once one of them has
// been paired with a B-set the other adds nothing. That quotient is exact, and
// it is what keeps an ·all· affordable: addInterleave enters one member-state
// vector by one position per member that can move into it, and every one of
// them has that vector's future (#1930).
//
// Two walk-scoped memos, both keyed on a fact rather than caching a computation
// whose inputs might drift (STYLE D3), and both bounded by maxProductStates
// because that is what bounds the entries they can acquire:
//
//   - liveOf, from a B-subset to what B may consume next from it. The live set is
//     a FUNCTION of the subset, and many R-states pair with one subset, so
//     without it the same union is rebuilt once per product state instead of once
//     per distinct subset.
//   - target, per state, from an R particle identifier to the subsets its
//     transition lands in — the per-source-particle collapse above.
//
// Both die with the walk; nothing survives into the Schema, and no automaton is
// memoized anywhere (contentAutomatonOf builds one per call).
func (s *Schema) contentModelRestricts(r, b contentAutomaton, scope contentRestrictionScope) bool {
	subsets := newSubsetTable()
	liveOf := map[int]liveSet{}
	future := r.futureClasses()
	start := productState{r: startState, b: subsets.intern([]int{startState})}
	visited := map[productState]bool{{r: future[startState+1], b: start.b}: true}
	queue := []productState{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if r.accepting(cur.r) && !b.acceptingAny(subsets.set(cur.b)) {
			// Clause 1: a sequence that ends here is ·locally valid· with respect
			// to R and leaves every run of B in a non-final state.
			return false
		}
		live, computed := liveOf[cur.b]
		if !computed {
			live = b.liveGroups(subsets.set(cur.b))
			liveOf[cur.b] = live
		}
		target := map[int][]int{}
		var edges []productState
		for _, p := range r.live(cur.r) {
			if w, ok := r.positions[p].term.(Wildcard); ok && !admitsSomeNamespace(w.NamespaceConstraint()) {
				continue // no item is ·attributed· to a wildcard admitting no name
			}
			ids, decided := target[r.positions[p].particleID]
			if !decided {
				successors := s.matchPositions(r.positions[p], b, live)
				if len(successors) == 0 {
					return false // clause 1: R can continue where B cannot
				}
				for _, next := range successors {
					if scope == restrictsFully && !next.governable && !s.someBindingSubsumes(b, next.positions, r.positions[p]) {
						return false // clause 2, ctr-child-type-subsumption
					}
					ids = append(ids, subsets.intern(next.positions))
				}
				target[r.positions[p].particleID] = ids
			}
			for _, id := range ids {
				edges = append(edges, productState{r: p, b: id})
			}
		}
		for _, next := range edges {
			seen := productState{r: future[next.r+1], b: next.b}
			if visited[seen] {
				continue
			}
			if len(visited) >= maxProductStates {
				// GAP(xsd): the walk is abandoned and the derivation provisionally
				// accepted once the product reaches maxProductStates. RULED permanent
				// by #499 (STYLE P3b) rather than a fold in progress: that thread
				// carries the ruling in full, and it rests on the bounded-resource
				// argument below rather than on a spec licence. The branch is
				// unreached by the whole W3C suite — maxProductStates' doc records
				// the measurement, and the margin it has left — so the incompleteness
				// is latent, and latent is not licensed.
				//
				// A spec licence covers one sub-population of this branch, and this
				// marker claims none beyond it. §3.4.6.3's leniency for an undecidable
				// clause 2.4.2 is gated by a two-conjunct antecedent — "If (1) the type
				// definition being checked has T.{content
				// type}.{particle}.{term}.{compositor} = all and (2) an implementation
				// is unable to determine by examination of the schema in isolation
				// whether or not clause 2.4.2 is satisfied, then the implementation may
				// provisionally accept the derivation" — and the
				// ·implementation-defined· sentence after it, "whether a processor (a)
				// always detects violations of clause 2.4.2 by examination of the
				// schema in isolation, (b) detects them only when some element
				// information item in the input document is valid against T but not
				// against T.{base type definition}, or (c) sometimes detects such
				// violations by examination of the schema in isolation and sometimes
				// not", says WHEN a processor already inside that antecedent detects
				// them. It states no condition of its own and grants nothing outside
				// it, and the all-compositor condition appears nowhere else in the
				// document. An ·all· group reaches this walk like any other content
				// model (#1930), so an R whose top compositor is all meets condition
				// (1) here and condition (2) is this very inability: that R, charged
				// by its clause-2.4.2 caller, is covered outright. Everything else
				// that reaches this line — a sequence or choice R, whatever B holds,
				// and every reader charging a rule other than clause 2.4.2 — is
				// outside the antecedent. Reading (c) as a RESIDUAL CATCH-ALL detached from
				// condition (1) was this marker's own earlier position; it is ruled
				// out, not merely unproved (#1378).
				//
				// "Provisionally accept" is not a spec-guaranteed-safe resting state
				// either. §3.4.6.3 continues: "If any instance encountered in the
				// ·assessment· episode is valid against T but not against T.{base type
				// definition}, then the derivation of T does not satisfy this
				// constraint, the schema does not conform to this specification, and
				// no ·assessment· can be performed using that schema." (b) and (c) as
				// worded describe processors that perform that runtime cross-check;
				// this ceiling gives up permanently with no runtime fallback, so a
				// schema accepted here can be non-conforming with nothing left to say
				// so. What the ceiling does guarantee is the direction enumerated
				// below: it abandons the WHOLE walk rather than truncating one into a
				// verdict.
				//
				// Fail-open for every reader of this true, and that reader set is
				// contentTypeRestricts' own (STYLE P3a): the call below its
				// maxContentPositions ceiling is this function's only caller
				// tree-wide, so the three readers that ceiling's marker enumerates by
				// identifier — with their rules, their scopes, and their charge on
				// false alone — are exactly these, and are not restated here. Each
				// loses a rejection it could have made and none gains one.
				//
				// What makes the approximation permanent is that the cost it refuses
				// is the powerset and nothing smaller. Clause 1 is stated
				// extensionally over two automata, the subset construction is the
				// decision procedure this file has for it, and that construction's
				// state space is exponential in the B-positions in the worst case — so
				// SOME bound on the walk is not optional, and every value of one
				// declines somewhere. Raising the constant moves where it declines,
				// buying walks whose cost grows with the states they are newly allowed
				// and no verdict anything has measured — ceilingHits is 0. Lowering it
				// is pinned from below by the series' 2026-10-01 point: the deepest walk the
				// suite finishes visits maxVisited=1188 states, so any value below 1188
				// starts declining walks that decide today. It is retired by a
				// construction that decides containment without materializing the
				// product, never by raising the constant.
				//
				// The review trigger is a RE-MEASUREMENT rather than a breach, because
				// a breach is the one warning that arrives too late: the high-water
				// mark moved 66.8× in six and a half weeks (2026-08-04 to 2026-09-19),
				// and at the series' latest point (2026-10-01) it stands at under a
				// third of the ceiling. Re-run the three counters maxProductStates' doc
				// names and reopen this ruling on EITHER ceilingHits > 0 or maxVisited
				// at 2048, half the ceiling. Half is what #499's two measurements
				// picked out: from their 1002 it was barely a doubling away against the 66.8×
				// observed between them, and a walk that stops there still finishes
				// and still decides. The two conditions are NOT independent: the
				// ceiling check above precedes the insertion into visited, so
				// maxVisited cannot exceed maxProductStates, and 2048 is a point on
				// the same climb to the same ceiling. A high-water mark moving at
				// anything like the 2026-08-04 to 2026-09-19 rate crosses that
				// factor-of-2 band between two measurements and reports ceilingHits > 0
				// without ever reading 2048, so the band fires with room left to act in
				// only at measurement resolution, not in calendar time; when the
				// re-measurement runs is #1609's. #499's grounding comment carries the
				// recipe in full.
				return true
			}
			visited[seen] = true
			queue = append(queue, next)
		}
	}
	return true
}

// successorSet is one B-set an R transition continues into, ascending, as
// matchPositions returns it.
//
// governable marks a set coveringWildcardUnion split off for ONE expanded name
// that a live ·element particle· admits, when R's wildcard is lax or strict and
// that name ·resolves· to a top-level element declaration: key-governing-ed
// clause 3 may then give the item a ·governing element declaration·, so R's
// ·default binding· for it may be that declaration rather than the keyword
// elementPositionBinding renders, and clause 2 is not charged on the set. The
// GAP(xsd) at that split carries the residual.
type successorSet struct {
	positions  []int
	governable bool
}

// matchPositions returns the B-sets the R-position p's transition continues
// into. When some live B-position admits every item p admits, it is one set:
// every live B-position that does. Otherwise, when p is a wildcard, it is
// coveringWildcardUnion's split: one set per expanded name a live ·element
// particle· admits, and one per part of the rest of p's {namespace constraint}
// that a different base wildcard admits. An empty result is a clause-1 failure:
// R can consume something no run of B can.
//
// A set rather than one position, because several B-positions may be live and
// admit p at once (see this file's determinism note): copies of one particle, and
// an ·element particle· beside a ·wildcard particle·, which 1.1 permits to
// compete. Members may therefore carry different {term}s and different ·default
// bindings·, and neither caller may single one out. The walk continues into the
// union of their ·follow· sets, and someBindingSubsumes examines EVERY member,
// succeeding when any one of them ·subsumes· — never reading a binding off a
// representative.
//
// The admits test runs once per SOURCE PARTICLE live in the state, not once per
// unfolded copy: copies of one particle share a {term}, so they share the answer
// (see liveSet, which computes the partition once per product state). This is
// the walk's hot loop — it is entered once per live R-position per product state
// — and the collapse is what keeps its cost proportional to the number of
// distinct particles rather than to the declared {max occurs} (#501).
func (s *Schema) matchPositions(p position, b contentAutomaton, live liveSet) []successorSet {
	admits := make([]bool, len(live.reps))
	for g, q := range live.reps {
		admits[g] = s.positionAdmits(b.positions[q], p)
	}
	if matched := live.positionsOf(admits); len(matched) > 0 {
		return []successorSet{{positions: matched}}
	}
	w, isWildcard := p.term.(Wildcard)
	if !isWildcard {
		return nil
	}
	return s.coveringWildcardUnion(w, b, live)
}

// coveringWildcardUnion is the one place positionAdmits is too weak to be used
// alone: cos-ns-subset relates ONE Namespace Constraint to ONE other, but the
// base may cover a restriction's wildcard w with SEVERAL of its own particles.
// W3C suite saxonData/Wild wild049 is the all-wildcard shape — a base ·all·
// group carrying namespace="##local" beside notNamespace="##local", whose union
// is every name, against a single ##any wildcard in the restriction — and the
// two base wildcards are non-overlapping, so cos-nonambig leaves them both live.
// Live ·element particles· cover names too: clause 1 is language containment,
// so a base wildcard carrying notQName="a" beside a live element particle a
// covers the whole namespace between them (#1954).
//
// The names element particles admit are SPLIT OFF first. Each live element
// particle admits finitely many expanded names — its declaration's, and, for a
// top-level declaration, the names of the declarations in its ·substitution
// group· — so the names sub (w's {namespace constraint}) admits among them form
// a finite list, in live-group order and then {element declarations} order
// (STYLE D2), and each is decided exactly: its set is S(n), every live position
// admitting n, element and wildcard alike, by the same relations positionAdmits
// uses (elementParticleAdmits, and cvc-wildcard-name through allowsName). The
// REST of sub is sub with those names added to its {disallowed names}, and only
// the rest is left to the wildcards. Admission here is cvc-wildcard-name on both
// sides: the keyword half of {disallowed names} is not resolved into names,
// which the GAP(xsd) on the keyword residual below records.
//
// Coverage of the rest is COMPUTED, not assumed: the live wildcards' {namespace
// constraint}s are folded left through Attribute Wildcard Union (§3.10.6.3,
// cos-aw-union) — the same left fold the constraint's own final paragraph
// prescribes for more than two operands — and the rest is then tested against
// the result by the same cos-ns-subset relation positionAdmits uses for one base
// wildcard. A rest the union does not cover, or a sub with no live wildcard at
// all, holds a name no live position admits — the rest admits a namespace,
// hence infinitely many names, and no element particle is left to admit one — so
// the empty result matchPositions and contentModelRestricts read as a clause-1
// failure is an exact rejection there, modulo the two keyword markers below.
//
// A covering union is then SPLIT, because it shows only that every name the
// rest admits is admitted by SOME base wildcard, never which base wildcard's
// {min occurs}/{max occurs} an item counts towards (derivation-ok-restriction
// clause 2.4.2, cos-content-act-restrict clause 1). Each distinct live wildcard
// particle whose constraint C meets the rest yields one part, rest ∩ C
// (Attribute Wildcard Intersection, §3.10.6.4, cos-aw-intersect), and one result
// set: every live position of every wildcard particle whose constraint meets
// that part. The walk continues from EVERY set — universal over R's items, since
// each item sub admits is a split-off name or lies in some part, and existential
// over B's runs within a set. Where the live wildcards are pairwise disjoint
// within the rest, a part's set is its own particle's positions alone, so an
// item counts towards the one base wildcard that admits it. saxonData
// All/all244 is that shape: B = all(any{one,two}{5,∞}, any{three}{0,2}) and R =
// all(any{one}{3,∞}, any{two,three}{2,2}), and R's (one, one, one, three, three)
// sends its three items to B's second wildcard alone, leaving B's first at 3 of
// its 5.
//
// Why the result never makes the walk answer false where the exact check
// answers true. The exact check (cos-content-act-restrict clause 1, with clause
// 2 at each item) moves B, on an item n, to S(n): the live positions admitting
// n. A split-off name's set IS S(n), or wider where a base wildcard's keyword
// exclusions are read as admitting (allowsName). Every part's set CONTAINS S(n)
// for some n in its part: a part is non-empty only when it admits a namespace,
// hence infinitely many names, while no name of the rest is admitted by an
// element particle, so some n in the part is admitted by wildcards alone, and
// the set holds every wildcard particle that can admit a name of the part. A
// walk branch entering such a set therefore shadows the exact branch taking n,
// through the same R-states, with a B-set at least as large; and each of the
// walk's rejections on that branch — no member accepting where R may end, no
// live position admitting R's next item, no member's binding subsuming
// (someBindingSubsumes) — quantifies over every member, so it holds of the
// smaller exact set too and is an exact rejection. The argument is per branch
// and assumes no monotonicity of the walk in its B-set: from a larger set
// matchPositions may return a direct match where a smaller one would return a
// split, and the argument covers that step the same way, since a direct match
// holds S(n) for every name of sub that no element particle and no other live
// wildcard admits, and such a name exists unless live wildcards overlap within
// sub — the first shape the GAP(xsd) below records.
//
// Construction cost stays within maxProductStates (#499): one R transition
// yields at most k + m sets, k the distinct live wildcard particles and m the
// split-off names, where it yielded one; no R position is added, each set is
// interned like any other B-set, and every product state still counts against
// the one ceiling.
//
// GAP(xsd): two shapes are not decided and return a set wider than an item's
// runs. Live base wildcards that overlap within the rest put every overlapping
// particle's positions into a part's set, so an item one of them admits also
// counts towards the others' runs; cos-nonambig forbids two distinct wildcard
// particles live after one prefix to ·overlap·, but addInterleave's states are
// not among those Phase C checked, so the shape is not shown unreachable. And
// the four unreachable error slots below return every live wildcard position as
// one set, as the whole union did before the split. Both sets contain S(n) for
// some n, so by the argument above the walk answers true more often and never
// less. #1953 owns deciding them (derivation-ok-restriction clause 2.4.2,
// cos-aw-intersect). Fail-open for all three readers of that true, each of which
// charges only on false (STYLE P3a): checkRestrictionContentType
// (complexderivation.go) charges derivation-ok-restriction clause 2.4.2;
// checkExtensionTwoStepDerivable (complexextension.go) re-charges the same call
// through checkDerivationOKRestriction as cos-ct-extends clause 1.5; and
// checkModelGroupRedefinitions (redefinition.go) charges src-redefine clause
// 6.2.2. Each loses a rejection it could have made and none gains one. The same
// overlap met by matchPositions' direct match instead can leave a set holding
// no S(n), and the direction there is unestablished.
//
// GAP(xsd): the verdict is exact for {namespaces} and for the defined/QName half
// of {disallowed names}, but §3.10.6.3 has no sibling bullet, so the fold silently
// drops a sibling keyword a live base wildcard carried (see
// UnionNamespaceConstraint). A base set that collectively disallows a
// sibling-excluded name can therefore be read as COVERING a restriction that
// should be rejected on that basis — fail-open, never a false reject, in the
// direction this file's header fixes for every approximation.
//
// That missing bullet is NOT an omission in cos-aw-union and is not a future
// issue's to fix: §3.10.6.3 is titled Attribute Wildcard Union, sibling is defined
// only for ELEMENT wildcards (cvc-wildcard §3.10.4.1 clause 3 gates the sibling
// test on "W is an element wildcard"), and ##definedSibling is not even
// grammatically available on <anyAttribute> — §3.10.2's notQName there admits
// ##defined alone, which w-props-correct clause 5 restates as a component
// invariant (rejectSiblingOnAttributeWildcard). So an attribute wildcard cannot
// carry sibling, and the constraint correctly has no bullet for it.
//
// The loss is therefore local to THIS call site, the one place the attribute-only
// union algebra is applied to element wildcards, and only when more than one base
// wildcard is live — a single one is never folded, and cos-ns-subset compares its
// sibling, whether positionAdmits asks it of sub or this function of the rest.
// The local specs define no multi-operand element-wildcard union that preserves
// the keyword, and inventing one is not this seam's to do: #265 ruled the
// limitation PERMANENT rather than open, on the oracle grounding recorded on that
// issue.
//
// GAP(xsd): the keyword half of {disallowed names} is not resolved into names on
// either side of the split, and the direction is a REJECTION where the language
// is contained. On B's side cos-ns-subset's tail refuses a base wildcard
// carrying defined or sibling for a rest that does not carry it too, though the
// names those keywords exclude are finitely many (cvc-wildcard clauses 2.1 and
// 3) and a live element particle may admit every one of them: with a the only
// top-level declaration, B = choice(<element ref="a"/>, any notQName="##defined")
// under R = any is contained, and both src-redefine clause 6.2.2 and
// derivation-ok-restriction charge it. On R's side a name a keyword of w's
// excludes is still split off and walked, which can only add branches. Resolving
// them needs the declaration graph, and for sibling the containing type, which
// restrictsLanguage's model-group reader does not have. The readers are this
// function's three (checkRestrictionContentType, checkExtensionTwoStepDerivable
// and checkModelGroupRedefinitions, named in the marker above), each of which
// charges on the false. Tracked by #1998.
//
// A single live wildcard with no split-off name reaches the same false
// positionAdmits already answered: its rest is sub, its union itself, and
// cos-ns-subset refused that pair when matchPositions asked it directly.
//
// The fold is written out rather than delegated to an N-ary helper: it has one
// caller, and cos-aw-union's binary primitive plus the caller's own loop is how
// parser/produce_complex.go folds the intersection too (STYLE T4/T5).
//
// The fold and the split run over the distinct source particles (liveSet's
// representatives), while every result set names each live position they cover.
// Folding a repeated copy would add nothing DIFFERENT: copies of one particle
// carry one identical {namespace constraint} value (addParticle's allocator
// reset makes particleID -> {term} a function), so re-including a copy would
// hand UnionNamespaceConstraint an operand already in constraints, never a new
// one — this is a claim about which OPERANDS the fold sees, not that §3.10.6.3's
// union is idempotent as a relation: it is not (the sibling-keyword GAP above is
// exactly a case where folding drops a sibling keyword, so X ∪ X can differ from
// X for a constraint that carries one). The fold and the split therefore see
// one operand per DISTINCT wildcard particle, and a single particle's copies are
// never folded with one another. The locals die with the call; nothing derivable
// is stored (STYLE D3).
func (s *Schema) coveringWildcardUnion(w Wildcard, b contentAutomaton, live liveSet) []successorSet {
	sub := w.NamespaceConstraint()
	var groups []int
	var constraints []NamespaceConstraint
	var elements []int
	for g, q := range live.reps {
		switch t := b.positions[q].term.(type) {
		case Wildcard:
			groups = append(groups, g)
			constraints = append(constraints, t.NamespaceConstraint())
		case ElementDeclaration:
			elements = append(elements, g)
		default:
			panic("xsd: coveringWildcardUnion: position {term} is neither an element declaration nor a wildcard")
		}
	}
	if len(constraints) == 0 {
		return nil
	}
	names := s.elementCoveredNames(sub, b, live, elements)
	// Every error below is unreachable: each operand is the {namespace
	// constraint} of an already-built Wildcard, sub with names it admits added to
	// its {disallowed names}, or the rest's intersection with one, and each such
	// record satisfies w-props-correct (clause 4 holds of every added name, since
	// sub admits it; see UnionNamespaceConstraint and intersectNamespaceConstraint
	// for the other two). Should a future divergence reach one, the arm returns
	// every live wildcard position as one set, the GAP(xsd) shape above, and the
	// error DECIDES that verdict rather than being dropped (STYLE S3), exactly as
	// contentTypeRestricts treats contentAutomatonOf's own unreachable error slot.
	// The loc is the zero xsderr.Loc{} because this is a finalize-time decision
	// with no source position of its own; nothing user-visible is charged to it.
	undecided := []successorSet{{positions: live.positionsOf(groupsIn(len(live.reps), groups))}}
	rest, err := NewNamespaceConstraint(xsderr.Loc{}, sub.variety, sub.namespaces,
		append(slices.Clone(sub.disallowedNames), names...), sub.disallowedNameKeywords)
	if err != nil {
		return undecided
	}
	union := constraints[0]
	for _, next := range constraints[1:] {
		folded, err := UnionNamespaceConstraint(xsderr.Loc{}, union, next)
		if err != nil {
			return undecided
		}
		union = folded
	}
	if !wildcardSubset(rest, union) {
		return nil
	}
	var sets []successorSet
	for _, n := range names {
		sets = append(sets, s.elementCoveredSet(n, w, b, live, elements, groups, constraints))
	}
	for i, c := range constraints {
		part, err := intersectNamespaceConstraint(xsderr.Loc{}, rest, c)
		if err != nil {
			return undecided
		}
		if !admitsSomeNamespace(part) {
			continue
		}
		members := []int{groups[i]}
		for j, other := range constraints {
			if j == i {
				continue
			}
			shared, err := intersectNamespaceConstraint(xsderr.Loc{}, part, other)
			if err != nil {
				return undecided
			}
			if admitsSomeNamespace(shared) {
				members = append(members, groups[j])
			}
		}
		sets = append(sets, successorSet{positions: live.positionsOf(groupsIn(len(live.reps), members))})
	}
	return sets
}

// elementCoveredNames lists, without repeats, the expanded names sub admits
// (cvc-wildcard-name) that some live ·element particle· in elements admits: each
// particle's declaration's own name, then, when that declaration is top-level,
// every top-level declaration it admits through its ·substitution group·, in
// {element declarations} order (STYLE D2). A local declaration heads no
// ·substitution group· (cvc-accept clause 2.3.2), so it contributes its own name
// alone, even when a same-named top-level declaration has members (#2000). A
// particle admits a name exactly when elementParticleAdmits would admit that
// name's declaration, so the list is the finite set coveringWildcardUnion splits
// off.
func (s *Schema) elementCoveredNames(sub NamespaceConstraint, b contentAutomaton, live liveSet, elements []int) []QName {
	var names []QName
	add := func(n QName) {
		if !sub.AllowsName(n) || slices.Contains(names, n) {
			return
		}
		names = append(names, n)
	}
	for _, g := range elements {
		d := b.positions[live.reps[g]].term.(ElementDeclaration)
		add(d.Name())
		if d.ScopeVariety() != ScopeGlobal {
			continue // a local declaration heads no substitution group
		}
		for _, e := range s.elements {
			if s.inSubstitutionGroupOf(e.Name(), d.Name()) {
				add(e.Name())
			}
		}
	}
	return names
}

// elementCoveredSet is S(n) for one name coveringWildcardUnion splits off: every
// live position of every element particle that admits n and of every wildcard
// particle whose {namespace constraint} admits it (cvc-wildcard-name, as
// positionAdmits asks it).
//
// GAP(xsd): clause 2 (ctr-child-type-subsumption) is not charged on the set when
// w is lax or strict and n ·resolves· to a top-level element declaration. There
// key-governing-ed clause 3 may make that declaration the item's ·governing
// element declaration·, so R's ·default binding· for it is case 1's Element
// Declaration rather than the keyword elementPositionBinding renders, and an
// Element Declaration in B never subsumes a keyword (loc-testSubP clauses 1-4).
// This is #345's gap met on the SPECIFIC side, against an element particle
// rather than another keyword, and it is RULED permanent by #345: whether case 1
// applies to a wildcard-attributed item is the assessment episode's fact. Where
// it cannot apply — w is skip (§3.10.4.1's closing Note), or n resolves to no
// top-level declaration, so key-governing-ed clauses 3 and 4 find none — the
// keyword is R's exact binding and clause 2 is charged as elsewhere. The
// direction is fail-open: contentModelRestricts reads governable only to skip a
// clause-2 charge, and only under restrictsFully, whose readers are
// checkRestrictionContentType (derivation-ok-restriction clause 2.4.2) and
// checkExtensionTwoStepDerivable (cos-ct-extends clause 1.5), each charging on
// false; checkModelGroupRedefinitions charges clause 1 alone and never reaches
// the read. Each loses a rejection it could have made and neither gains one.
func (s *Schema) elementCoveredSet(n QName, w Wildcard, b contentAutomaton, live liveSet, elements, groups []int, constraints []NamespaceConstraint) successorSet {
	var members []int
	for _, g := range elements {
		if s.inlineDeclarationMatchesName(b.positions[live.reps[g]].term.(ElementDeclaration), n) {
			members = append(members, g)
		}
	}
	for i, c := range constraints {
		if c.AllowsName(n) {
			members = append(members, groups[i])
		}
	}
	_, global := s.Element(n)
	return successorSet{
		positions:  live.positionsOf(groupsIn(len(live.reps), members)),
		governable: w.ProcessContents() != ProcessSkip && global,
	}
}

// groupsIn marks the given liveSet group indexes in a slice of n flags, the form
// liveSet.positionsOf reads.
func groupsIn(n int, groups []int) []bool {
	in := make([]bool, n)
	for _, g := range groups {
		in[g] = true
	}
	return in
}

// positionAdmits reports whether the base particle at position general admits
// every element information item the restriction's particle at position specific
// admits — the per-transition compatibility test clause 1's reduction needs.
//
//   - element over element: the same expanded name, or the restriction's
//     declaration ·substitutable· for the base's through a ·substitution group·.
//   - wildcard over element: the base's {namespace constraint} admits the
//     restriction's expanded name (cvc-wildcard-name).
//   - wildcard over wildcard: the restriction's {namespace constraint} is a
//     ·wildcard subset· of the base's (§3.10.6.2, cos-ns-subset).
//   - element over wildcard: never. A wildcard admits an open set of expanded
//     names, and one Element Declaration admits one name plus its ·substitution
//     group·, so no base element particle ALONE covers a restriction wildcard.
//     It can cover some of its names beside live base wildcards that cover the
//     rest, which coveringWildcardUnion decides for the whole live set (#1954).
//
// Both approximations here resolve towards admitting. Substitution-group
// membership is not one of them: elementParticleAdmits credits a top-level
// declaration with its ·substitution group·, as inSubstitutionGroupOf decides
// cos-equiv-derived-ok-rec (substitutiongroup.go), and a local one with none
// (cvc-accept clause 2.3.2), so this clause reads the true ·substitution group·
// whichever way membership pushes the verdict. And the base's wildcard is asked
// through Wildcard.allowsName (cvc-wildcard-name) rather than through
// allowsElementWildcardName's defined/sibling keyword exclusions, for the same
// reason: the narrower test would shrink B and could only add rejections.
func (s *Schema) positionAdmits(general, specific position) bool {
	switch g := general.term.(type) {
	case ElementDeclaration:
		d, ok := specific.term.(ElementDeclaration)
		return ok && s.elementParticleAdmits(g, d)
	case Wildcard:
		switch sp := specific.term.(type) {
		case ElementDeclaration:
			return g.allowsName(sp.Name())
		case Wildcard:
			return wildcardSubset(sp.NamespaceConstraint(), g.NamespaceConstraint())
		default:
			panic("xsd: positionAdmits: position {term} is neither an element declaration nor a wildcard")
		}
	default:
		panic("xsd: positionAdmits: position {term} is neither an element declaration nor a wildcard")
	}
}

// elementParticleAdmits reports whether an ·element particle· whose {term} is
// general admits every item whose ·governing element declaration· is specific:
// specific has general's expanded name (cvc-accept clause 2.3.1), or general is
// top-level and specific is ·substitutable· for it through its ·substitution
// group· (cvc-accept clause 2.3.2, cos-equiv-derived-ok-rec).
//
// A LOCAL general is credited with no substitution group (#2000):
// inSubstitutionGroupOf resolves its head BY NAME through {element
// declarations}, so asked directly it would answer a local declaration from a
// same-named top-level one and credit it with that one's members.
// inlineDeclarationMatchesName applies the scope guard before that lookup.
//
// Until #281 this function carried a second arm admitting any two TOP-LEVEL
// declarations unconditionally, because no producer mapped substitutionGroup=
// into {substitution group affiliations} and charging the resulting
// non-membership false-rejected valid schemas (W3C MS-Element
// elemZ027_a/_b/_e/_f, MS-Particles particlesZ008/Z028 — each a base <element
// ref="head"/> restricted to a member of head's group). parser now maps the
// attribute, so inSubstitutionGroupOf sees the affiliation edges it needs and
// decides those pairings exactly; a global pairing with no affiliation chain
// between them is REJECTED.
func (s *Schema) elementParticleAdmits(general, specific ElementDeclaration) bool {
	return s.inlineDeclarationMatchesName(general, specific.Name())
}

// someBindingSubsumes is cos-content-act-restrict clause 2
// (ctr-child-type-subsumption) at one matched transition: B's ·default binding·
// for the item must ·subsume· R's.
//
// It is satisfied when ANY matched B-position's binding subsumes. Copies of one
// particle all carry the same {term} and so the same binding, which is the
// normal case and makes the quantifier immaterial. It becomes visible only where
// an ·element particle· and a ·wildcard particle· of B both admit the name —
// legal in 1.1, with ·attribution· going to the Element Declaration — and there
// the existential reading is the FAIL-OPEN one: it accepts on the wildcard's
// keyword binding where attribution would have used the declaration's. Charging
// the declaration's instead would reject on a run B might never take.
//
// GAP(xsd): the existential is this file's, not the spec's. Clause 2 quantifies
// in the SINGULAR — "B's ·default binding· for E ·subsumes· that defined by R" —
// and ·default binding· (key-dft-binding) is a "(partial) functional mapping"
// from an item to ONE binding, so a literal reading has no set to choose among
// and passing on any member over-approximates it. The one consumer is
// contentModelRestricts, which does nothing with a false answer except charge
// clause 2 and stop: every acceptance the quantifier adds therefore costs a
// rejection, and none can fabricate one. Deciding it exactly means committing to
// the single position ·attribution· selects, which the paragraph above records
// as the direction deliberately not taken.
func (s *Schema) someBindingSubsumes(b contentAutomaton, matched []int, p position) bool {
	specific := elementPositionBinding(p)
	for _, q := range matched {
		if s.bindingSubsumes(elementPositionBinding(b.positions[q]), specific) {
			return true
		}
	}
	return false
}

// elementPositionBinding is ·default binding· (§3.4.6.4, key-dft-binding) for an
// element information item ·attributed· to the particle at one position of a
// content model: case 1 (a ·governing element declaration·) for an ·element
// particle·, cases 4/5/6 (the strict/lax/skip keyword) for a ·wildcard
// particle·.
//
// GAP(xsd): cases 4 and 5 carry the qualifier "and it does not have a ·governing
// element declaration·"; when the item DOES have one, case 1 applies even though
// it was ·attributed· to a wildcard. Whether an item has a ·governing element
// declaration· is an assessment-episode fact (key-governing-ed clauses 3 and 4
// resolve it by expanded name against the schema of a real episode, and clause 1
// lets the processor stipulate one outright), so this static rendering reports
// the keyword for every wildcard position and never case 1. RULED permanent by
// #345 (STYLE P3b): no schema shape can conclude that case 1 APPLIES to a
// wildcard-attributed item, and the real binding is the instance validator's to
// render. It is the element-side twin of attributeDefaultBinding's case-3
// marker.
//
// The gap is NARROWER than the whole wildcard branch. Two wildcard shapes fall
// statically outside key-governing-ed clauses 2-4, and for them the keyword
// returned below is the EXACT key-dft-binding — modulo clause 1's stipulation,
// which this schema-authoring-time constraint sets aside as
// attributeDefaultBinding does:
//
//   - {process contents} skip. §3.10.4.1's closing Note states that "if the
//     wildcard has a {process contents} property of skip, then the item has no
//     ·governing· declaration": clause 2 is unavailable to any
//     wildcard-attributed item, clause 3 fires only for a strict or lax
//     ·wildcard particle·, and key-skipped makes the item ·skipped·, which
//     clause 4.1 excludes. The keyword is case 6, which carries no qualifier.
//   - {disallowed names} containing defined (notQName="##defined").
//     cvc-wildcard clause 2.1 makes "the expanded name of I does not ·resolve·
//     to an element declaration" a precondition of valid attribution to that
//     wildcard (key-att-to), so the item has no declaration for clause 3 or
//     clause 4 to resolve to; clauses 4.2 and 4.3 qualify a declaration clause
//     2.1 has already excluded. sibling is NOT a further exclusion: cvc-wildcard
//     clause 3 tests ·match· against the declarations ·contained· in the
//     containing type's content model, not the global ·resolve· that
//     key-governing-ed clauses 3 and 4 use.
//
// The remainder — a strict or lax wildcard whose {disallowed names} does not
// contain defined — is fail-open by ruling. Both reads of this function are in
// someBindingSubsumes, as bindingSubsumes' general and specific arguments, and
// its one consumer, contentModelRestricts, charges clause 2 on a false answer
// and on nothing else. As the general argument, keywordSubsumes answers a
// remainder keyword at least as permissively as bindingSubsumes answers the
// case-1 Element Declaration in its place, so reporting the keyword can only
// miss a charge. As the specific argument against a general keyword it does
// the same in every pairing but one: a strict G refuses a lax S where it would
// accept an Element Declaration. That refusal is exact rather than
// manufactured. The pairing is met either at a set matching R's wildcard over a
// namespace — matchPositions' direct match or a coveringWildcardUnion part — so
// some item there has no ·governing element declaration· in any episode
// (keywordSubsumes' doc, #2546), or at a set split off for one name that
// ·resolves· to no top-level declaration, where the keyword is exact. The
// specific keyword also meets an Element Declaration, in a set
// coveringWildcardUnion splits off for one name a live ·element particle·
// admits, and there a keyword is never subsumed, so the remainder keyword would
// manufacture the charge: that set is marked governable and not charged
// instead, the GAP(xsd) at elementCoveredSet (#1954).
//
// Every subset returns the same wildcardKeywordBinding, so nothing here branches
// on which one applies (STYLE D3); what differs is only whether that value is
// exact or fail-open.
func elementPositionBinding(p position) defaultBinding {
	switch t := p.term.(type) {
	case ElementDeclaration:
		return elementDeclarationBinding{decl: t} // case 1
	case Wildcard:
		return wildcardKeywordBinding{keyword: t.ProcessContents()} // cases 4/5/6
	default:
		panic("xsd: elementPositionBinding: position {term} is neither an element declaration nor a wildcard")
	}
}
