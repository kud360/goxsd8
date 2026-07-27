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
// automata. The automata are particleattribution.go's, unchanged and unforked:
// the same addParticle, the same unfoldCopies bound, the same first/follow/last
// sets cos-nonambig was decided over (STYLE T4). A second, differently-bounded
// unfolding would explore a state space cos-nonambig never validated.
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
// The walk terminates on its visited set: positions(R) is finite because
// unfoldCopies bounds each particle's copies, and the B-sets are drawn from a
// finite powerset, with maxProductStates as a hard ceiling on top. That set is a
// walk-scoped graph-reachability guard over two automata already built, not a
// component-resolution cycle check, so PRINCIPLES 5 / STYLE D4 are untouched.
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
// unnecessary as well as harmful.

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
// {particle}. It is finalize-scoped and memoized nowhere (STYLE D3), exactly as
// checkContentModelsUnambiguous builds and discards one per content model.
func (s *Schema) contentAutomatonOf(c ElementContent) (contentAutomaton, error) {
	b := &automaton{s: s}
	first, last, emptiable, err := b.addParticle(c.Particle)
	if err != nil {
		return contentAutomaton{}, err
	}
	return contentAutomaton{automaton: b, first: first, last: last, emptiable: emptiable}, nil
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
func (a contentAutomaton) liveIn(states []int) []int {
	var live []int
	for _, st := range states {
		live = mergePositions(live, a.live(st))
	}
	return live
}

// productState is one state of the product walk: a single state of R paired with
// the SET of B states B's runs may be in. Both position lists are ascending, so
// the key productKey builds is canonical.
type productState struct {
	r int
	b []int
}

// productKey is the canonical string identity of a product state, for the
// visited set. It is a lookup key only — the set is never ranged (STYLE D2).
func productKey(st productState) string {
	buf := strconv.AppendInt(nil, int64(st.r), 10)
	for _, q := range st.b {
		buf = append(buf, ' ')
		buf = strconv.AppendInt(buf, int64(q), 10)
	}
	return string(buf)
}

// maxProductStates bounds how many product states the walk visits before giving
// up and provisionally accepting. The subset construction's state space is a
// powerset in the worst case; every content model the W3C suite and real schemas
// contain settles in a handful of states, but a bound keeps a pathological one
// from turning schema assembly into an exponential walk. It is a ceiling on
// WORK, never on the verdict of a walk that finishes.
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
// Three shapes are provisionally accepted rather than decided, each licensed and
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
//     holds for the machinery here: addAll models all(P1…Pn) as (P1|…|Pn)*, whose
//     language is a SUPERSET of the interleave — a single-member ·all· already
//     reads as that member repeated — so an R modelled that way admits sequences
//     R does not, and charging their absence from B would false-reject. B needs
//     no such leniency
//     for the mirror-image reason: the same over-approximation makes B look
//     larger, which can only accept. This is the NARROW §3.4.6.3 all-group
//     allowance, not the broader implementation-defined (a)/(b)/(c) clause that
//     the pre-#263 stub relied on; being spec-licensed rather than a deliberate
//     incompleteness, it carries no GAP marker.
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
		// this construction does not model. No producer emits {open content}
		// yet (parser/produce_complex.go declines <openContent>), so the arm
		// guards programmatic construction only; it lands with the producer
		// (#265).
		return true
	}
	if s.usesAllCompositor(rc.Particle.Term()) {
		return true // §3.4.6.3's all-group leniency; see the doc above
	}
	r, err := s.contentAutomatonOf(rc)
	if err != nil {
		// addParticle's error slot is not reachable on a *Schema that exists:
		// every arm of addTerm/addResolvedTerm either returns nil or panics on a
		// broken sealed sum, and a dangling <element ref>/<group ref> was already
		// charged src-resolve by Phase A. Should a future term kind make it
		// reachable, provisionally accepting is the fail-open direction §3.4.6.3's
		// implementation-defined licence covers, and the error is not silently
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
func (s *Schema) contentModelRestricts(r, b contentAutomaton) bool {
	start := productState{r: startState, b: []int{startState}}
	visited := map[string]bool{productKey(start): true}
	queue := []productState{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if r.accepting(cur.r) && !b.acceptingAny(cur.b) {
			// Clause 1: a sequence that ends here is ·locally valid· with respect
			// to R and leaves every run of B in a non-final state.
			return false
		}
		live := b.liveIn(cur.b)
		for _, p := range r.live(cur.r) {
			matched := s.matchPositions(r.positions[p], b, live)
			if len(matched) == 0 {
				return false // clause 1: R can continue where B cannot
			}
			if !s.someBindingSubsumes(b, matched, r.positions[p]) {
				return false // clause 2, ctr-child-type-subsumption
			}
			next := productState{r: p, b: matched}
			key := productKey(next)
			if visited[key] {
				continue
			}
			if len(visited) >= maxProductStates {
				// GAP(xsd): the walk is abandoned and the derivation provisionally
				// accepted once the product exceeds maxProductStates. §3.4.6.3
				// licenses it — "It is ·implementation-defined· whether a processor
				// (a) always detects violations of clause 2.4.2 by examination of the
				// schema in isolation, (b) detects them only when some element
				// information item in the input document is valid against T but not
				// against T.{base type definition}, or (c) sometimes detects such
				// violations" — so this is choice (c) for the models that reach the
				// ceiling, and it is fail-open, never a false reject. It is marked
				// nonetheless because it is an incompleteness of THIS algorithm, not
				// of the spec's demands. It is retired by a construction that decides
				// containment without materializing the product, never by raising the
				// constant.
				return true
			}
			visited[key] = true
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
func (s *Schema) matchPositions(p position, b contentAutomaton, live []int) []int {
	var matched []int
	for _, q := range live {
		if s.positionAdmits(b.positions[q], p) {
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
// unionNamespaceConstraint). A base set that collectively disallows a
// sibling-excluded name can therefore be read as COVERING a restriction that
// should be rejected on that basis — fail-open, never a false reject, in the
// direction this file's header fixes for every approximation.
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
func coveringWildcardUnion(sub NamespaceConstraint, b contentAutomaton, live []int) []int {
	var wildcards []int
	var constraints []NamespaceConstraint
	for _, q := range live {
		w, ok := b.positions[q].term.(Wildcard)
		if !ok {
			continue
		}
		wildcards = append(wildcards, q)
		constraints = append(constraints, w.NamespaceConstraint())
	}
	if len(wildcards) < 2 {
		return nil
	}
	union := constraints[0]
	for _, next := range constraints[1:] {
		folded, err := unionNamespaceConstraint(xsderr.Loc{}, union, next)
		if err != nil {
			// Unreachable: every operand is the {namespace constraint} of an
			// already-built Wildcard, and the union of two such records always
			// satisfies w-props-correct (see unionNamespaceConstraint). Should a
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
// membership uses the OVER-approximating predicate — membership makes the
// transition compatible, i.e. makes cos-content-act-restrict easier to satisfy,
// so an over-broad group can only miss a rejection, the opposite polarity from
// cos-nonambig's ·overlap·, which FIRES a constraint on membership and so takes
// the under-approximating predicate (substitutiongroup.go). And the base's
// wildcard is asked through Wildcard.AllowsName (cvc-wildcard-name) rather than
// through allowsElementWildcardName's defined/sibling keyword exclusions, for
// the same reason: the narrower test would shrink B and could only add
// rejections.
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
// the same expanded name, or specific ·substitutable· for general through a
// ·substitution group·.
//
// GAP(xsd): two TOP-LEVEL declarations with different expanded names are
// admitted unconditionally, without the membership walk deciding it. No producer
// in this repo maps substitutionGroup= yet — parser/produce.go passes a nil
// {substitution group affiliations} to every NewElementDeclaration — so
// mayBeInSubstitutionGroupOf answers false for a member the spec puts squarely
// in the group, and charging that absence FALSE-REJECTS a valid schema: W3C
// suite MS-Element elemZ027_a/_b/_e/_f and MS-Particles particlesZ008/Z028 are
// each a base <element ref="head"/> restricted to a member of head's group. The
// escape is scoped to the pairing where an affiliation is even possible —
// e-props-correct clause 3 confines {substitution group affiliations} to a
// global declaration, so a LOCAL declaration on either side still needs the
// expanded names to agree, which is the shape most content models have.
// Accepting is fail-open, never a false reject; it is retired by the change that
// first maps substitutionGroup= into {substitution group affiliations}, after
// which mayBeInSubstitutionGroupOf answers exactly and this arm can go.
func (s *Schema) elementParticleAdmits(general, specific ElementDeclaration) bool {
	if general.Name() == specific.Name() {
		return true
	}
	if s.mayBeInSubstitutionGroupOf(specific.Name(), general.Name()) {
		return true
	}
	return general.ScopeVariety() == ScopeGlobal && specific.ScopeVariety() == ScopeGlobal
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
// B's checkModelGroupsAcyclic (PRINCIPLES 5).
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
