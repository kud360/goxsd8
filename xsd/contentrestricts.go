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
// sets cos-nonambig is decided over. One thing is chosen differently — the
// unfolding of a numeric occurrence range, which this file supplies as
// unfoldExactly through automaton.unfold.
//
// WHY THE UNFOLDING IS THE ONE THING NOT SHARED. maxMandatoryCopies /
// maxOptionalCopies bound a range to two copies of each kind, which is
// verdict-preserving for cos-nonambig — whose subject is which particle-
// IDENTIFIER sets are live in one state, and two copies realize every such set —
// and is NOT verdict-preserving here, because as a statement about LANGUAGE the
// bound rewrites the range itself: e{3,6} would read as e{2,4} and e{0,100} as
// e{0,2}. That rewrite is monotone in neither direction, and both directions are
// reachable on a bare single element particle: e{3,6} under e{0,100} is a VALID
// restriction whose fourth R-copy would find no live B-position and be rejected,
// and e{5,5} under e{3,3} is an INVALID one both of whose sides would collapse to
// two mandatory copies and become indistinguishable. Nothing licenses either
// outcome — clause 1 is pure set containment naming no algorithm, Appendix J's
// unfolding guidance is scoped by its own text to cos-nonambig, and §3.4.6.3's
// implementation-defined licence excuses provisional ACCEPTANCE of an undecided
// case, never a rejection. So the constants are not raised, which would only move
// the same two thresholds, but replaced for this consumer: unfoldExactly emits
// {max occurs} copies of a bounded range and {min occurs} copies plus a loop-back
// for an unbounded one, so the automaton accepts exactly L and this walk DECIDES
// containment over the declared {min occurs}/{max occurs} instead of over a
// truncated unfolding (#501).
//
// The exact unfolding does not step outside what cos-nonambig validated.
// maxMandatoryCopies' own argument is that copies past the second realize no
// identifier set the first two do not, so every state of the exactly-unfolded
// automaton offers an identifier set already present in the bounded one that
// Phase C accepted: the "no two competing particles in one state" premise the
// determinism note below rests on carries over unchanged. It is not load-bearing
// for soundness in any case — R's live positions are ALL explored and B is
// subset-constructed, which is the standard containment check over two NFAs, and
// neither half needs a deterministic automaton.
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
// fail-open and a spurious one would false-reject a valid schema, which
// §3.4.6.3's own implementation-defined licence (a processor may detect 2.4.2
// violations always statically, only from instances, or sometimes) makes
// unnecessary as well as harmful — though see contentModelRestricts' giveup site
// for how narrowly that licence is actually conditioned.

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
// {particle}, unfolding every numeric occurrence range exactly (unfoldExactly),
// so the automaton accepts exactly the sequences ·locally valid· with respect to
// that content model and the walk below decides containment rather than
// approximating it. Nothing here bounds the construction: the caller must have
// cleared the content model through unfoldedPositions first.
//
// It is finalize-scoped and memoized nowhere (STYLE D3), exactly as
// checkContentModelsUnambiguous builds and discards one per content model.
func (s *Schema) contentAutomatonOf(c ElementContent) (contentAutomaton, error) {
	b := &automaton{s: s, unfold: unfoldExactly}
	first, last, emptiable, err := b.addParticle(c.Particle)
	if err != nil {
		return contentAutomaton{}, err
	}
	return contentAutomaton{automaton: b, first: first, last: last, emptiable: emptiable}, nil
}

// unfoldExactly is the automaton.unfold policy cos-content-act-restrict is
// decided over: the unfolding of a numeric occurrence range that preserves the
// LANGUAGE of the particle, which is the only fact a containment walk reads.
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

// modelGroupPositions sums its members' unfolded positions, saturating as soon as
// the running total passes the ceiling so a wide group under a wide range is not
// summed to completion. Particles are walked in document order (STYLE D2).
//
// <sequence> and <choice> contribute each member's fragment exactly once —
// addSequence and addChoice differ in the edges they draw, never in the positions
// they emit. addAll emits each member's fragment TWICE, once for the star and
// once for the primed alternation that carries the group's ·last· set, so an
// <all> counts double. Keeping that factor in step with addAll is what makes this
// count the automaton's size rather than an estimate of it.
func (s *Schema) modelGroupPositions(g ModelGroup) int {
	copies := 1
	if g.Compositor() == CompositorAll {
		copies = 2
	}
	total := 0
	for _, p := range g.particles {
		total += s.unfoldedPositions(p)
		if total > maxContentPositions/copies {
			return maxContentPositions + 1
		}
	}
	return copies * total
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

// productState is one state of the product walk: a single state of R paired with
// the SET of B states B's runs may be in, the set named by its subsetTable
// identifier. Naming it rather than carrying it is what makes the state
// comparable, so the visited set keys on the state itself and no string is built
// per state (see subsetTable).
type productState struct {
	r int
	b int
}

// subsetTable interns the B-state sets the walk reaches: equal sets share an
// identifier, so a product state is a pair of ints and the visited set is a
// plain map over that pair.
//
// It exists because the identity of a product state has to be tested once per
// live R-position per state, while the SETS those states pair with are far
// fewer — every R-position of one source particle transitions into the SAME set
// (contentModelRestricts computes it once), so interning turns a per-transition
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
// The constant is MEASURED headroom rather than an unexamined guess (#282).
// Instrumenting both the giveup branch and every insertion into the visited set,
// then running the full W3C suite — 41858 cases over 6 lanes, which reach this
// product walk 688 times — recorded ZERO walks that hit the ceiling and a
// high-water mark of 15 product states, two orders of magnitude below 4096. The
// bound is therefore inert on every content model the suite contains; it is kept
// for the worst case the powerset admits, which nothing measures, not because
// any known schema approaches it.
const maxProductStates = 4096

// contentTypeRestricts is derivation-ok-restriction clause 2.4.2's delegate:
// whether T's {content type} ·restricts· B's as defined in Content type
// restricts (Complex Content) (§3.4.6.4, cos-content-act-restrict).
//
// It answers a bool rather than an error because clause 2.4.2 sits inside
// derivation-ok-restriction clause 2, a DISJUNCTION: the failure that reaches a
// user is charged once, by checkRestrictionContentType, after every branch has
// declined. The two conditions of cos-content-act-restrict are therefore not
// reported separately; contentModelRestricts records which one failed only in
// the comments at its two rejection sites.
//
// Four shapes are provisionally accepted rather than decided, each licensed and
// each fail-open:
//
//   - a non-element {content type} on either side. 2.4.1 (restrictionVarietyPairOK)
//     has already established both are element-only or mixed before this is
//     reached, so this is an assertion of that precondition, not a case.
//   - a present {open content} on either side. B's open content ADMITS elements
//     the automaton below does not model, so ignoring it would shrink B and
//     manufacture clause-1 rejections; R's only widens R, which is harmless, but
//     the pair is skipped together so the reason stays one reason.
//   - an ·all· group anywhere in R's content model. §3.4.6.3 states this
//     leniency in its own text — "If (1) the type definition being checked has
//     T.{content type}.{particle}.{term}.{compositor} = all and (2) an
//     implementation is unable to determine by examination of the schema in
//     isolation whether or not clause 2.4.2 is satisfied, then the implementation
//     may provisionally accept the derivation" — and precondition (2) genuinely
//     holds for the machinery here: addAll models all(P1…Pn) as a star over its
//     members followed by a primed replay of one of them, whose language is a
//     SUPERSET of the star's and so of the interleave — a single-member ·all·
//     already reads as that member repeated — so an R modelled that way admits sequences
//     R does not, and charging their absence from B would false-reject. B needs
//     no such leniency
//     for the mirror-image reason: the same over-approximation makes B look
//     larger, which can only accept. This is the NARROW §3.4.6.3 all-group
//     allowance, not the broader implementation-defined (a)/(b)/(c) clause that
//     the pre-#263 stub relied on; being spec-licensed rather than a deliberate
//     incompleteness, it carries no GAP marker.
//   - a content model whose exact unfolding would exceed maxContentPositions on
//     either side. This one alone is a RESOURCE ceiling rather than a modelling
//     gap — it bounds the automaton's size, see maxContentPositions for what that
//     does and does not bound — and it is the only path on which a declared
//     occurrence range does not reach the walk; the licence it leans on is the
//     one contentModelRestricts' giveup site states, and the marker is at the
//     branch itself.
func (s *Schema) contentTypeRestricts(tct, bct ContentType) bool {
	rc, ok := tct.(ElementContent)
	if !ok {
		return true
	}
	bc, ok := bct.(ElementContent)
	if !ok {
		return true
	}
	if rc.OpenContent != nil || bc.OpenContent != nil {
		// GAP(xsd): Open Content is not folded into either automaton, so a
		// content type carrying one is provisionally accepted. §3.4.4's
		// ·locally valid· sequences for an interleave or suffix Open Content
		// interleave the {wildcard} with the particle's own positions, which
		// this construction does not model. Since #230 the producer really does
		// emit {open content} (§3.4.2.3.3 clauses 5-6), so this arm is live for
		// ordinary schemas rather than programmatic construction alone: a
		// restriction whose Open Content is wider than its base's is accepted
		// where derivation-ok-restriction clause 2.4 would reject it. The
		// direction is fail-open — a missing rejection, never a false one — so it
		// costs verdicts and cannot fabricate them; the fold lands with #265.
		return true
	}
	if s.usesAllCompositor(rc.Particle.Term()) {
		return true // §3.4.6.3's all-group leniency; see the doc above
	}
	if s.unfoldedPositions(rc.Particle) > maxContentPositions || s.unfoldedPositions(bc.Particle) > maxContentPositions {
		// GAP(xsd): a content model whose exact unfolding would exceed
		// maxContentPositions is not unfolded at all, and the derivation is
		// provisionally accepted undecided. The alternative is not a smaller
		// automaton but a WRONG one: truncating the copies of an occurrence range
		// rewrites the range, which is monotone in neither direction and
		// false-rejects conforming schemas (see this file's header, #501). The
		// ceiling therefore declines the whole question, which is fail-open — a
		// missed rejection, never a fabricated one — and rests on exactly the
		// §3.4.6.3 reading contentModelRestricts' giveup site sets out, including
		// how narrowly that reading is licensed. Unlike this file's other
		// ceilings the branch is REACHED — six of the W3C suite's 1538 candidate
		// content models land here, each carrying a maxOccurs in the thousands to
		// millions (the measurement is recorded on maxContentPositions) — so the
		// incompleteness is live rather than latent, and it is retired only by a
		// construction that decides containment without materializing an
		// automaton per occurrence, never by raising the constant.
		return true
	}
	r, err := s.contentAutomatonOf(rc)
	if err != nil {
		// addParticle's error slot is not reachable on a *Schema that exists:
		// every arm of addTerm/addResolvedTerm either returns nil or panics on a
		// broken sealed sum, and a dangling <element ref>/<group ref> was already
		// charged src-resolve by Phase A. Should a future term kind make it
		// reachable, provisionally accepting is the fail-open direction
		// contentModelRestricts' giveup site states the actual, narrower licence
		// for; the argument is not restated here. The error is not silently
		// discarded — it decides this verdict (STYLE S3).
		return true
	}
	b, err := s.contentAutomatonOf(bc)
	if err != nil {
		return true // same unreachable error slot, same direction
	}
	return s.contentModelRestricts(r, b)
}

// contentModelRestricts walks the product of the two automata, deciding both
// conditions of cos-content-act-restrict in one pass.
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
// the copies of one particle live in a state all transition into one B-set. The
// copies are still enqueued SEPARATELY, each as its own R-state: they carry
// different ·follow· sets, so nothing about the state space is collapsed, only
// the recomputation of one answer per copy (#501). Iteration stays in ascending
// R-position order, so the walk order is unchanged by the memo.
//
// Two walk-scoped memos, both keyed on a fact rather than caching a computation
// whose inputs might drift (STYLE D3), and both bounded by maxProductStates
// because that is what bounds the entries they can acquire:
//
//   - liveOf, from a B-subset to what B may consume next from it. The live set is
//     a FUNCTION of the subset, and many R-states pair with one subset, so
//     without it the same union is rebuilt once per product state instead of once
//     per distinct subset.
//   - target, per state, from an R particle identifier to the subset its
//     transition lands in — the per-source-particle collapse above.
//
// Both die with the walk; nothing survives into the Schema, and no automaton is
// memoized anywhere (contentAutomatonOf builds one per call).
func (s *Schema) contentModelRestricts(r, b contentAutomaton) bool {
	subsets := newSubsetTable()
	liveOf := map[int]liveSet{}
	start := productState{r: startState, b: subsets.intern([]int{startState})}
	visited := map[productState]bool{start: true}
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
		target := map[int]int{}
		for _, p := range r.live(cur.r) {
			id, decided := target[r.positions[p].particleID]
			if !decided {
				matched := s.matchPositions(r.positions[p], b, live)
				if len(matched) == 0 {
					return false // clause 1: R can continue where B cannot
				}
				if !s.someBindingSubsumes(b, matched, r.positions[p]) {
					return false // clause 2, ctr-child-type-subsumption
				}
				id = subsets.intern(matched)
				target[r.positions[p].particleID] = id
			}
			next := productState{r: p, b: id}
			if visited[next] {
				continue
			}
			if len(visited) >= maxProductStates {
				// GAP(xsd): the walk is abandoned and the derivation
				// provisionally accepted once the product reaches maxProductStates.
				// The branch is unreached by the whole W3C suite (the measurement is
				// recorded on maxProductStates), so the incompleteness is latent —
				// but latent is not licensed, and the licence is narrower than it
				// looks.
				//
				// §3.4.6.3's leniency for an undecidable clause 2.4.2 is textually
				// anchored to a condition this branch does not test: "If (1) the type
				// definition being checked has T.{content
				// type}.{particle}.{term}.{compositor} = all and (2) an implementation
				// is unable to determine by examination of the schema in isolation
				// whether or not clause 2.4.2 is satisfied, then the implementation
				// may provisionally accept the derivation". The sentence that follows
				// — "It is ·implementation-defined· whether a processor (a) always
				// detects violations of clause 2.4.2 by examination of the schema in
				// isolation, (b) detects them only when some element information item
				// in the input document is valid against T but not against T.{base
				// type definition}, or (c) sometimes detects such violations by
				// examination of the schema in isolation and sometimes not" — states
				// no condition of its own, and the all-compositor condition appears
				// nowhere else in the document. A genuine {compositor} = all never
				// reaches here in any case: contentTypeRestricts takes the narrow,
				// correctly scoped allowance through usesAllCompositor before an
				// automaton is built. This ceiling applies uniformly to sequence and
				// choice models too, and for those it rests on reading (c) as a
				// RESIDUAL CATCH-ALL detached from condition (1) — defensible, since
				// nothing in the local specs forecloses it, but not textually
				// guaranteed. Naming that stretch is half of why this marker exists.
				//
				// The other half is that "provisionally accept" is not a
				// spec-guaranteed-safe resting state. §3.4.6.3 continues: "If any
				// instance encountered in the ·assessment· episode is valid against T
				// but not against T.{base type definition}, then the derivation of T
				// does not satisfy this constraint, the schema does not conform to
				// this specification, and no ·assessment· can be performed using that
				// schema." (b) and (c) as worded describe processors that perform that
				// runtime cross-check; this ceiling gives up permanently with no
				// runtime fallback, so a schema accepted here can be non-conforming
				// with nothing left to say so. What the ceiling does guarantee is
				// direction: it abandons the WHOLE walk rather than truncating one
				// into a verdict, so it is fail-open — a missed rejection, never a
				// false one.
				//
				// It is retired by a construction that decides containment without
				// materializing the product, never by raising the constant; #499
				// owns that retirement.
				return true
			}
			visited[next] = true
			queue = append(queue, next)
		}
	}
	return true
}

// matchPositions returns, in ascending order, every live B-position that admits
// each item the R-position p admits. An empty result is a clause-1 failure: R
// can consume something no run of B can.
//
// The result is a SET rather than one position because several B-positions may be
// live and admit p at once (see this file's determinism note): copies of one
// particle, and an ·element particle· beside a ·wildcard particle·, which 1.1
// permits to compete. Members may therefore carry different {term}s and different
// ·default bindings·, and neither caller may single one out. The walk continues
// into the union of their ·follow· sets, and someBindingSubsumes examines EVERY
// member, succeeding when any one of them ·subsumes· — never reading a binding
// off a representative.
//
// The admits test runs once per SOURCE PARTICLE live in the state, not once per
// unfolded copy: copies of one particle share a {term}, so they share the answer
// (see liveSet, which computes the partition once per product state). This is
// the walk's hot loop — it is entered once per live R-position per product state
// — and the collapse is what keeps its cost proportional to the number of
// distinct particles rather than to the declared {max occurs} (#501).
func (s *Schema) matchPositions(p position, b contentAutomaton, live liveSet) []int {
	admits := make([]bool, len(live.reps))
	for g, q := range live.reps {
		admits[g] = s.positionAdmits(b.positions[q], p)
	}
	var matched []int
	for i, q := range live.positions {
		if admits[live.group[i]] {
			matched = append(matched, q)
		}
	}
	if len(matched) > 0 {
		return matched
	}
	w, isWildcard := p.term.(Wildcard)
	if !isWildcard {
		return nil
	}
	return coveringWildcardUnion(w.NamespaceConstraint(), b, live)
}

// coveringWildcardUnion is the one place positionAdmits is too weak to be used
// alone: cos-ns-subset relates ONE Namespace Constraint to ONE other, but the
// base may cover a restriction's wildcard with SEVERAL of its own. W3C suite
// saxonData/Wild wild049 is exactly that — a base ·all· group carrying
// namespace="##local" beside notNamespace="##local", whose union is every name,
// against a single ##any wildcard in the restriction — and the two base
// wildcards are non-overlapping, so cos-nonambig leaves them both live.
//
// The union is COMPUTED, not assumed: the live wildcards' {namespace
// constraint}s are folded left through Attribute Wildcard Union (§3.10.6.3,
// cos-aw-union) — the same left fold the constraint's own final paragraph
// prescribes for more than two operands — and sub is then tested against the
// result by the same cos-ns-subset relation positionAdmits uses for one base
// wildcard. Both §3.10.6 relations are therefore exercised here: a covering union
// returns every live wildcard as the matched set (each is a run B may take, and
// none may be singled out — see matchPositions), while a union that does not cover
// sub is a clause-1 failure, reported as the empty result matchPositions and
// contentModelRestricts already read that way.
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
// wildcard is live — a single one is decided by cos-ns-subset alone in
// positionAdmits, where sibling IS compared. The local specs define no
// multi-operand element-wildcard union that preserves the keyword, and inventing
// one is not this seam's to do: #265 ruled the limitation PERMANENT rather than
// open, on the oracle grounding recorded on that issue.
//
// A single live wildcard is left to cos-ns-subset alone in positionAdmits, where
// the relation is already exact; folding it here would only restate that verdict.
//
// The fold is written out rather than delegated to an N-ary helper: it has one
// caller, and cos-aw-union's binary primitive plus the caller's own loop is how
// parser/produce_complex.go folds the intersection too (STYLE T4/T5).
//
// The positions and their constraints are gathered in ONE pass into two locals —
// the {term} assertion is made once per live position, never repeated to recover
// a constraint the fold needs — and both die with the call; nothing derivable is
// stored (STYLE D3).
//
// The FOLD runs over the distinct source particles (liveSet's representatives),
// while the RESULT names every live position they cover. Folding a repeated copy
// would add nothing DIFFERENT: copies of one particle carry one identical
// {namespace constraint} value (addParticle's allocator reset makes particleID
// -> {term} a function), so re-including a copy would hand
// UnionNamespaceConstraint an operand already in constraints, never a new one —
// this is a claim about which OPERANDS the fold sees, not that §3.10.6.3's union
// is idempotent as a relation: it is not (the GAP above is exactly a case where
// folding drops a sibling keyword, so X ∪ X can differ from X for a constraint
// that carries one). The guard is therefore on the number of DISTINCT wildcard
// particles: one particle's copies are decided by cos-ns-subset alone in
// positionAdmits, which has already answered for all of them, exactly as a
// single wildcard is.
func coveringWildcardUnion(sub NamespaceConstraint, b contentAutomaton, live liveSet) []int {
	var wildcards []int
	var constraints []NamespaceConstraint
	for i, q := range live.positions {
		w, ok := b.positions[q].term.(Wildcard)
		if !ok {
			continue
		}
		wildcards = append(wildcards, q)
		if q == live.reps[live.group[i]] {
			constraints = append(constraints, w.NamespaceConstraint())
		}
	}
	if len(constraints) < 2 {
		return nil
	}
	union := constraints[0]
	for _, next := range constraints[1:] {
		folded, err := UnionNamespaceConstraint(xsderr.Loc{}, union, next)
		if err != nil {
			// Unreachable: every operand is the {namespace constraint} of an
			// already-built Wildcard, and the union of two such records always
			// satisfies w-props-correct (see UnionNamespaceConstraint). Should a
			// future divergence reach it, the union is undecided, so the arm
			// resolves the way every approximation in this file resolves —
			// towards accepting, i.e. the union is assumed to cover — and the
			// error DECIDES that verdict rather than being dropped (STYLE S3),
			// exactly as contentTypeRestricts treats contentAutomatonOf's own
			// unreachable error slot. The loc is the zero xsderr.Loc{} because
			// this is a finalize-time decision with no source position of its
			// own; nothing user-visible is charged to it.
			return wildcards
		}
		union = folded
	}
	if !wildcardSubset(sub, union) {
		return nil
	}
	return wildcards
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
//     group·, so no base element particle covers a restriction wildcard.
//
// Both approximations here resolve towards admitting. Substitution-group
// membership is not one of them: inSubstitutionGroupOf decides
// cos-equiv-derived-ok-rec exactly (substitutiongroup.go), so this clause reads
// the true ·substitution group· whichever way membership pushes the verdict. And
// the base's wildcard is asked through Wildcard.AllowsName (cvc-wildcard-name)
// rather than through allowsElementWildcardName's defined/sibling keyword
// exclusions, for the same reason: the narrower test would shrink B and could
// only add rejections.
func (s *Schema) positionAdmits(general, specific position) bool {
	switch g := general.term.(type) {
	case ElementDeclaration:
		d, ok := specific.term.(ElementDeclaration)
		return ok && s.elementParticleAdmits(g, d)
	case Wildcard:
		switch sp := specific.term.(type) {
		case ElementDeclaration:
			return g.AllowsName(sp.Name())
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
// specific is ·substitutable· for general through a ·substitution group·, which
// already folds in plain expanded-name equality (cos-equiv-derived-ok-rec clause
// 1, so no separate name test is needed here).
//
// There is NO approximation left here. Until #281 this function carried a second
// arm admitting any two TOP-LEVEL declarations unconditionally, because no
// producer mapped substitutionGroup= into {substitution group affiliations} and
// charging the resulting non-membership false-rejected valid schemas (W3C
// MS-Element elemZ027_a/_b/_e/_f, MS-Particles particlesZ008/Z028 — each a base
// <element ref="head"/> restricted to a member of head's group). parser now maps
// the attribute, so inSubstitutionGroupOf sees the affiliation edges it needs and
// decides those pairings exactly; the escape hatch is gone, and a global pairing
// with no affiliation chain between them is now correctly REJECTED.
func (s *Schema) elementParticleAdmits(general, specific ElementDeclaration) bool {
	return s.inSubstitutionGroupOf(specific.Name(), general.Name())
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
// declaration· is an assessment-episode fact (key-governing-ed), so this static
// rendering always reports the keyword for a wildcard position. The direction is
// fail-open in both places it is read: keywordSubsumes answers true for a strict
// or skip general binding and for every non-skip specific one under lax, so
// reporting the keyword where case 1 would apply can only miss a rejection. It
// is the element-side twin of attributeDefaultBinding's case-3 marker and is
// retired only by an assessment-time consumer, never by a static check.
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

// usesAllCompositor reports whether an ·all· group is reachable through a
// particle's {term}, following <group ref> edges exactly as the automaton's own
// addTerm does. It is what selects §3.4.6.3's all-group leniency (see
// contentTypeRestricts).
//
// A top-level ·all· is the shape the spec's own wording names, and cos-all-limited
// (§3.8.6.2) confines an ·all· to that position or to another ·all· reached
// through a <group ref>, so the tree walk and the spec's
// "T.{content type}.{particle}.{term}.{compositor} = all" agree on every content
// model the grammar admits. The walk carries no visited set, licensed by Phase
// B's checkModelGroupsAcyclic (PRINCIPLES 9).
func (s *Schema) usesAllCompositor(t TermOrRef) bool {
	switch t := t.(type) {
	case ResolvedTerm:
		g, ok := t.Term.(ModelGroup)
		return ok && s.modelGroupUsesAllCompositor(g)
	case ModelGroupRef:
		mgd, ok := s.modelGroupIndex[t.Name]
		return ok && s.modelGroupUsesAllCompositor(mgd.ModelGroup())
	case ElementDeclarationRef:
		return false // an element declaration holds no model group
	default:
		panic("xsd: usesAllCompositor: non-exhaustive TermOrRef switch")
	}
}

// modelGroupUsesAllCompositor reports whether g or any group nested in it has
// {compositor} = all. Particles are walked in document order (STYLE D2).
func (s *Schema) modelGroupUsesAllCompositor(g ModelGroup) bool {
	if g.Compositor() == CompositorAll {
		return true
	}
	for _, p := range g.particles {
		if s.usesAllCompositor(p.Term()) {
			return true
		}
	}
	return false
}
