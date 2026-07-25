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
// Glushkov automaton) of a content model and checking that no state offers two
// competing transitions. Appendix J sketches exactly this construction —
// "transcribe the content model into an automaton in the usual way using epsilon
// transitions for optionality and unbounded maxOccurs, unfolding other numeric
// occurrence ranges … using not element QNames as transition labels, but rather
// pairs of element QNames and positions in the model" — and the theorem that
// makes it a decision procedure rather than a heuristic is Brüggemann-Klein and
// Wood's characterization of one-unambiguous regular languages ("One-Unambiguous
// Regular Languages", Information and Computation 140(2), 1998), which Appendix
// J's own bibliography points at: a regular expression is 1-unambiguous exactly
// when its position automaton is deterministic. ·compete· (§3.8.4.2) — two
// particles reachable by two ·paths· of one sequence differing only in their last
// item — is precisely two positions live in one state of that automaton.
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
// It does NOT follow {base type definition}: §3.4.7 lets xs:anyType be its own
// base, and a derived type's {content type} already folds in whatever the
// derivation contributes (§3.4.2.3.3), so the base chain has nothing to add and
// would reintroduce the self-loop this design relies on Phase B to have excluded.

// maxMandatoryCopies and maxOptionalCopies bound how many copies of a particle's
// {term} fragment the unfolding emits, so that maxOccurs="100000" does not build
// a hundred thousand of them.
//
// The bound is verdict-preserving, not a heuristic. Every unfolded copy of one
// source particle replays the SAME particle identifiers (see automaton.addParticle),
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
type automaton struct {
	s              *Schema
	positions      []position
	follow         [][]int
	nextParticleID int
}

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
func (s *Schema) checkContentModelsUnambiguous(facts substitutionFacts) error {
	for _, t := range s.types {
		ct, ok := t.(ComplexType)
		if !ok {
			continue // a *SimpleType has no content model
		}
		ec, ok := ct.ContentType().(ElementContent)
		if !ok {
			continue // empty and simple {content type}s hold no particle
		}
		b := &automaton{s: s}
		first, _, _, err := b.addParticle(ec.Particle)
		if err != nil {
			return err
		}
		if err := b.check(first, facts, "complex type "+ct.Name().String()+" content model"); err != nil {
			return err
		}
	}
	for _, mgd := range s.modelGroups {
		b := &automaton{s: s}
		first, _, _, err := b.addModelGroup(mgd.ModelGroup())
		if err != nil {
			return err
		}
		if err := b.check(first, facts, "model group definition "+mgd.Name().String()); err != nil {
			return err
		}
	}
	return nil
}

// check reports the first competing pair in the automaton. States are visited in
// a fixed order — the start state first, then the state after position 0, after
// position 1, … — and within a state the pair with the least (i, j) by position
// index is reported, so the reported violation is the lexicographically least
// (state, i, j) and identical inputs always reject identically (STYLE D1).
func (b *automaton) check(first []int, facts substitutionFacts, ctx string) error {
	if err := b.checkState(first, facts, ctx, "at the start of"); err != nil {
		return err
	}
	for i := range b.follow {
		if err := b.checkState(b.follow[i], facts, ctx, "after "+b.describe(i)+" in"); err != nil {
			return err
		}
	}
	return nil
}

// checkState rejects the first competing pair live in one state of the automaton.
func (b *automaton) checkState(state []int, facts substitutionFacts, ctx, where string) error {
	for x := 0; x < len(state); x++ {
		for y := x + 1; y < len(state); y++ {
			competes, err := b.competes(state[x], state[y], facts)
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
func (b *automaton) competes(i, j int, facts substitutionFacts) (bool, error) {
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
		return b.s.elementsOverlap(ti, dj, facts), nil
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
		return "element declaration " + t.Name().String()
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
// Membership uses the UNDER-approximating predicate: overlap here FIRES a schema
// component constraint, so an over-broad substitution group would reject a valid
// schema (see substitutiongroup.go).
func (s *Schema) elementsOverlap(a, b ElementDeclaration, facts substitutionFacts) bool {
	if a.Name() == b.Name() {
		return true // bullet 1
	}
	if !facts.anyAffiliation {
		return false // bullets 2-3 need an affiliation edge; the schema declares none
	}
	if s.nameInSubstitutionGroupOf(a.Name(), b, facts) {
		return true // bullet 2
	}
	if s.nameInSubstitutionGroupOf(b.Name(), a, facts) {
		return true // bullet 2, the other direction
	}
	return s.substitutionGroupsIntersect(a, b, facts) // bullet 3
}

// nameInSubstitutionGroupOf reports whether name is the expanded name of an
// element declaration in head's ·substitution group·. The test is on the NAME,
// not on the declaration: Appendix J bullet 2 says "one of them has the same
// expanded name as an element declaration in the other's substitution group", so
// a local declaration named x overlaps a head whose group contains the top-level
// x.
func (s *Schema) nameInSubstitutionGroupOf(name QName, head ElementDeclaration, facts substitutionFacts) bool {
	if head.ScopeVariety() != ScopeGlobal {
		return false // a local declaration heads no substitution group
	}
	return s.certainlyInSubstitutionGroupOf(name, head.Name(), facts)
}

// substitutionGroupsIntersect is Appendix J bullet 3: two GLOBAL element
// declarations overlap when their ·substitution groups· share a member. 1.1
// allows a declaration several {substitution group affiliations}, so this is not
// subsumed by bullet 2 — one member can sit in two otherwise unrelated groups.
//
// The candidate members are enumerated from the document-order {element
// declarations} slice, never from elementIndex, so no map iteration order reaches
// the verdict (STYLE D2).
func (s *Schema) substitutionGroupsIntersect(a, b ElementDeclaration, facts substitutionFacts) bool {
	if a.ScopeVariety() != ScopeGlobal || b.ScopeVariety() != ScopeGlobal {
		return false
	}
	for _, e := range s.elements {
		if !s.inSubstitutionGroup(e.Name(), a.Name(), facts) {
			continue
		}
		if s.inSubstitutionGroup(e.Name(), b.Name(), facts) {
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
// The intersection itself is not re-derived: IntersectNamespaceConstraint is the
// one canonical §3.10.6.4 implementation (STYLE T4). Its error is documented
// unreachable for two validly-constructed operands, but it is propagated rather
// than dropped (STYLE S3) so any future divergence surfaces as a failure instead
// of a silently wrong verdict.
func wildcardsOverlap(a, b Wildcard) (bool, error) {
	c, err := IntersectNamespaceConstraint(xsderr.Loc{}, a.NamespaceConstraint(), b.NamespaceConstraint())
	if err != nil {
		return false, fmt.Errorf("intersecting the {namespace constraint}s of two wildcard particles (cos-aw-intersect): %w", err)
	}
	switch c.Variety() {
	case NamespaceConstraintAny, NamespaceConstraintNot:
		return true, nil
	default:
		return len(c.namespaces) > 0, nil
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
	copies, mandatory, loop := unfoldCopies(p.Occurs())
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

// unfoldCopies computes how many copies of a particle's {term} fragment to emit
// for an occurrence range, how many of them are mandatory, and whether the last
// one carries a loop-back edge. See maxMandatoryCopies for why the copy count is
// bounded without changing any verdict.
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
		return b.addAll(g)
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

// addAll emits an <all> group (§3.8.4.1.3, interleave). The literal interleave of
// n members has n! orderings and must not be built. For deciding ·compete· it is
// exactly equivalent to
//
//	all(P1 … Pn) ≈ (P1 | P2 | … | Pn)*
//
// because the two agree on the only two facts the position automaton reads: which
// positions are live at the start (all members', in both), and which positions
// may follow a given one (in the interleave, any member other than the one just
// taken; in the star, any member including the one just taken). The star's extra
// pairs are exactly the self-pairs of one member, which competes rejects by
// particle identity before ever consulting the terms. Nesting composes, since an
// <all> reached through a <group ref> inside another <all> (which 1.1 permits at
// {min occurs} = {max occurs} = 1) contributes its own first/last sets to the
// outer union like any other member.
//
// Emptiability is NOT taken from the star form, which would always be emptiable:
// an <all> accepts the empty sequence exactly when every member does, and that is
// what an enclosing sequence needs in order not to invent competition sets.
func (b *automaton) addAll(g ModelGroup) ([]int, []int, bool, error) {
	var first, last []int
	emptiable := true
	for _, p := range g.particles {
		f, l, e, err := b.addParticle(p)
		if err != nil {
			return nil, nil, false, err
		}
		first = mergePositions(first, f)
		last = mergePositions(last, l)
		emptiable = emptiable && e
	}
	for _, q := range last {
		b.addFollow(q, first)
	}
	return first, last, emptiable, nil
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
