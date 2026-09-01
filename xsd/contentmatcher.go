package xsd

// This file is the M5 pull driver xsd/doc.go names: Matcher, the
// instance-guided advance of one complex type's {content type} particle, one
// child at a time. It decides Element Sequence Locally Valid (Particle)
// (§3.9.4.2, cvc-particle) — a one-clause wrapper over Element Sequence
// Accepted (Particle) (§3.9.4.3, cvc-accept) — and, through cvc-accept clause
// 3, Element Sequence Valid (§3.8.4.3, cvc-model-group) over §3.8.4.1's
// per-compositor recognition rules. The caller charges the enclosing
// cvc-complex-content (§3.4.4.3) or cvc-complex-type (§3.4.4.2) clause; nothing
// here builds an error, because "no particle admits this name" is a local
// non-match at every position but the last, and only the caller knows which
// position it is looking at.
//
// The walk COUNTS: state is one occurrence counter per particle of the
// flattened content model plus the path of particles the last item was
// ·attributed to·. No occurrence range is unfolded into copies, so
// maxOccurs="100000" costs one counter and not a hundred thousand positions —
// the opposite trade from particleattribution.go's automaton, which unfolds
// because ·compete· is a question about the MODEL and this one is a question
// about an INSTANCE. The two constructions are deliberately not shared:
// automaton.addAll accepts a documented SUPERSET of the interleave language
// (particleattribution.go), which is the right direction for a schema
// constraint and a false accept for an instance one, and its unfolding bound
// changes the accepted language outright (e{3,6} reads as e{2,4}).
//
// # Why one path, and never a second
//
// The walk commits to one particle per item and never reconsiders (PRINCIPLES
// 14). Two facts license that, and neither is an assumption a comment makes:
//
//   - cos-nonambig (§3.8.6.4) has run. A Matcher is reachable only through a
//     *Schema, which only SchemaBuilder.Finalize produces, and Phase C rejects
//     a content model with two ·competing· particles. So at most one ELEMENT
//     particle live in one state admits a given ·expanded name·, and the
//     particle an item is ·attributed to· is not a choice this file makes.
//   - Where an element particle and a wildcard particle both admit one name —
//     the one competition 1.1 permits (Appendix G.1.3) — cvc-accept's closing
//     Note fixes the answer: "the validation process defined in this
//     specification matches the element information item against the Element
//     Declaration, both in identifying the Element Declaration as the item's
//     ·context-determined declaration·, and in choosing alternative paths
//     through a content model". Next therefore searches the live particles for
//     an element admission first and for a wildcard admission only if that
//     search fails, which is PRINCIPLES 14's "explicit content beats a
//     wildcard" stated as a search order.
//
// The one non-determinism those two do not settle is the third the same Note
// names: "nested particles each of which has {max occurs} greater than 1, where
// the input sequence can be partitioned in multiple ways ... there is no fixed
// rule for eliminating the non-determinism". A greedy walk resolves that
// partition by staying in the innermost open iteration for as long as it can,
// and for THAT shape the choice is observable — (a{1,2}, b?){2,2} accepts "aab"
// as (a)(ab) and greedily consumes it as one iteration and then rejects. So the
// shape is DECLINED at construction (supported below) rather than answered
// wrongly, and the greedy walk is exact on everything that is not declined:
//
//   - a differing partition needs one particle P reachable both later in the
//     open iteration of some repeatable ancestor R and at the start of R's next
//     iteration (two DIFFERENT particles would ·compete·, which cos-nonambig has
//     already rejected);
//   - P starting R's body means every particle before it on its path is
//     skippable, so the shorter iteration the alternative closes before P holds
//     no mandatory particle of the body — and is a word of the body's language
//     only where the body has none at all, which is to say where it is
//     ·emptiable·. The one way that iteration can hold a mandatory particle is P
//     ITSELF, taken again, which needs a repeating particle inside a repeating
//     one: the declined shape, and (a{1,2}, b?){2,2} is exactly it;
//   - the greedy walk's iteration count is therefore never ABOVE the
//     alternative's, so no {max occurs} it satisfies is one the greedy walk
//     exceeds; and wherever the two counts differ the body is ·emptiable·, so
//     canExit lets the iterations still owed be empty and {min occurs} is met
//     as well.

// Attribution is what one element information item of a matched child sequence
// is ·attributed to· (§3.4.4.4, key-att-to): the {term} of the particle
// [Matcher.Next] advanced over. It is a sealed sum (STYLE T2's closed-sum
// exception) with exactly two variants, [ElementDeclaration] (cvc-accept clause
// 2) and [Wildcard] (cvc-accept clause 1), because those are the two kinds of
// ·basic particle· an item can be attributed to — a Model Group is not one, and
// returning [Term] would make an attribution to one representable (STYLE T1).
//
// An [ElementDeclaration] result is the particle's own declaration D, which is
// the answer to "which particle consumed this item". For an item admitted
// through cvc-accept clause 2.3.2 the ·context-determined declaration· is the
// ·substituting declaration· S and not D; resolving S is the recursive
// assessment's job (§3.3.4.6), not this one's.
type Attribution interface{ attribution() }

// attribution marks ElementDeclaration as an Attribution (cvc-accept clause
// 2.3); see the Attribution doc comment.
func (ElementDeclaration) attribution() {}

// attribution marks Wildcard as an Attribution (cvc-accept clause 1); see the
// Attribution doc comment.
func (Wildcard) attribution() {}

// contentNode is one particle of a flattened content model. children holds the
// node indices of a model group's {particles} in document order, and is nil for
// an ·element particle· or ·wildcard particle·, whose {term} is the leaf the
// walk matches names against.
//
// term is the RESOLVED {term}: the flattening follows an <element ref> through
// the element index and a <group ref> through the model group index, so no walk
// step re-resolves anything. Nodes are appended in preorder, so a node's index
// is less than every index in its subtree.
type contentNode struct {
	occurs   Occurs
	term     Term
	children []int
}

// Matcher advances one complex type's {content type} particle over an
// element-information-item sequence, deciding cvc-particle (§3.9.4.2) for the
// sequence one item at a time. Obtain one from [Schema.ContentMatcher]; the
// zero value is not usable.
//
// A Matcher is single-use and stateful: it holds the position the items so far
// reached in the content model, so the caller feeds it one element's
// [[children]] in document order and drops it. It is not safe for concurrent
// use, and nothing in it is shared with the schema beyond the immutable
// components the flattening read.
type Matcher struct {
	s      *Schema
	ct     ComplexType
	nodes  []contentNode
	counts []int
	path   []int
}

// ContentMatcher returns a [Matcher] over t's {content type} particle, or (nil,
// false) where the element sequence of an element ·governed by· t is not
// decidable here. The decision is made ONCE, at construction: a Matcher that
// exists decides every name put to it, and never declines mid-sequence.
//
// It is a method on *Schema rather than a free constructor because the walk
// rests on constraints Finalize has already decided — cos-nonambig (§3.8.6.4)
// for the determinism it never re-derives, mg-props-correct clause 2 for the
// <group ref> acyclicity that lets the flattening carry no visited set (STYLE
// D4) — and a *Schema is the one thing that cannot exist before they ran.
//
// The four declines, none of them a violation:
//
//   - a {content type} whose {variety} is empty or simple, which holds no
//     particle at all. cvc-complex-type clauses 1.1 and 1.2 govern those
//     directly and need no matcher.
//   - GAP(xsd): a present {open content}. cvc-complex-content clauses 2 and 3
//     split the sequence into a part matched against {particle} and a part
//     matched against the {open content} wildcard, which this walk does not do
//     (#717). The withheld value is the whole element-sequence verdict, whose
//     consumer set is validate's Result.violations and its one reader
//     Result.Violations: both carry violations PRESENT, so withholding the
//     verdict costs a rejection and manufactures none.
//   - GAP(xsd): a particle with {max occurs} greater than 1 holding another
//     such particle. That is cvc-accept's own named non-determinism, where the
//     greedy walk can reject a sequence some other partition accepts (see the
//     file comment); declining is what keeps this file free of false rejects.
//     #782 owns its retirement: deciding it needs a walk over a SET of
//     live partitions, which is a different engine, not a wider case here.
//   - GAP(xsd): an <all> group with a model group among its {particles}, which
//     cos-all-limited clause 2 admits only as a nested all group. Interleaving
//     two all groups' members needs per-member positions this walk does not
//     keep, and it keeps only counters because every other all group's members
//     are leaves. #783 owns its retirement.
func (s *Schema) ContentMatcher(t ComplexType) (*Matcher, bool) {
	ec, ok := t.ContentType().(ElementContent)
	if !ok {
		return nil, false
	}
	if ec.OpenContent != nil {
		return nil, false
	}
	m := &Matcher{s: s, ct: t}
	if _, ok := m.flatten(ec.Particle); !ok {
		return nil, false
	}
	if !m.supported(0, false) {
		return nil, false
	}
	m.counts = make([]int, len(m.nodes))
	return m, true
}

// flatten appends the node for p and, for a model group {term}, its whole
// subtree, returning p's node index. It reports false for a reference that
// resolves to nothing — unreachable on a *Schema, whose Phase A rejected a
// dangling <element ref>/<group ref> (src-resolve clauses 1.3 and 1.5), and a
// decline rather than a skipped particle so an unresolved name can never widen
// the accepted language.
func (m *Matcher) flatten(p Particle) (int, bool) {
	t, ok := m.resolveTerm(p.Term())
	if !ok {
		return 0, false
	}
	i := len(m.nodes)
	m.nodes = append(m.nodes, contentNode{occurs: p.Occurs(), term: t})
	g, isGroup := t.(ModelGroup)
	if !isGroup {
		return i, true
	}
	for _, c := range g.Particles() {
		j, ok := m.flatten(c)
		if !ok {
			return 0, false
		}
		m.nodes[i].children = append(m.nodes[i].children, j)
	}
	return i, true
}

// resolveTerm reads a particle's {term} slot, following an <element ref>
// through elementIndex and a <group ref> through modelGroupIndex to the
// referenced definition's {model group} (§3.7.2), exactly as
// particleattribution.go's addTerm and wildcardadmit.go's termContainsName read
// them, so no Schema.ModelGroup accessor is minted for an in-package reader
// (STYLE T5).
func (m *Matcher) resolveTerm(t TermOrRef) (Term, bool) {
	switch t := t.(type) {
	case ResolvedTerm:
		return t.Term, t.Term != nil
	case ElementDeclarationRef:
		d, ok := m.s.Element(t.Name)
		return d, ok
	case ModelGroupRef:
		mgd, ok := m.s.modelGroupIndex[t.Name]
		if !ok {
			return nil, false
		}
		return mgd.ModelGroup(), true
	default:
		panic("xsd: Matcher.resolveTerm: non-exhaustive TermOrRef switch")
	}
}

// supported reports whether the greedy walk is exact on the subtree at i.
// repeating says whether some ancestor of i has a {max occurs} greater than 1,
// which makes a second such particle beneath it the partition-ambiguous shape
// the file comment declines.
func (m *Matcher) supported(i int, repeating bool) bool {
	n := m.nodes[i]
	repeats := repeatable(n.occurs)
	if repeats && repeating {
		return false
	}
	g, isGroup := n.term.(ModelGroup)
	if !isGroup {
		return true
	}
	for _, c := range n.children {
		if g.Compositor() == CompositorAll {
			if _, nested := m.nodes[c].term.(ModelGroup); nested {
				return false
			}
		}
		if !m.supported(c, repeating || repeats) {
			return false
		}
	}
	return true
}

// repeatable reports whether an occurrence range admits more than one
// occurrence.
func repeatable(o Occurs) bool {
	max, bounded := o.Max()
	return !bounded || max > 1
}

// Next advances the content model over one element information item whose
// ·expanded name· is name, reporting what the item is ·attributed to·
// (§3.4.4.4). It reports (nil, false) when no particle live at the current
// position admits the name, which is the cvc-accept (§3.9.4.3) rejection its
// caller charges against that item's own location; the Matcher is unchanged by
// a rejected name, so a caller may stop at the first one or keep feeding.
//
// The two searches are cvc-accept's element/wildcard precedence: every live
// particle is offered the name as an ·element particle· first (clause 2.3.1's
// expanded-name match, then clause 2.3.2's ·substitution group· membership),
// and only a name no element particle admits is offered to the wildcard
// particles (clause 1, cvc-wildcard §3.10.4.1 in full).
func (m *Matcher) Next(name QName) (Attribution, bool) {
	if a, ok := m.step(name, admitElements); ok {
		return a, true
	}
	return m.step(name, admitWildcards)
}

// Accepting reports whether the sequence fed so far is ·accepted· by the
// content model — whether every open particle can be closed where the sequence
// stopped. A false result is the "the sequence ends short of a particle it must
// satisfy" half of cvc-accept, which the caller charges against the containing
// element rather than any child, there being no child at the offending
// position.
func (m *Matcher) Accepting() bool {
	if len(m.path) == 0 {
		return m.emptiable(0)
	}
	for d := len(m.path) - 1; d >= 0; d-- {
		if !m.canExit(d) {
			return false
		}
	}
	return true
}

// admitKind is which kind of ·basic particle· one search pass will match a name
// against, so that cvc-accept's Note — an item both a Wildcard and an Element
// Declaration accept goes to the Element Declaration — is one search order and
// not a tie-break buried at the match site.
type admitKind int

const (
	admitElements admitKind = iota
	admitWildcards
)

// step runs one search pass, committing the walk state only if it matched.
func (m *Matcher) step(name QName, kind admitKind) (Attribution, bool) {
	t := m.clone()
	a, ok := t.advance(name, kind)
	if !ok {
		return nil, false
	}
	m.counts, m.path = t.counts, t.path
	return a, true
}

// clone copies the walk state and shares the immutable rest, so a search pass
// that fails leaves nothing behind.
func (m *Matcher) clone() *Matcher {
	return &Matcher{
		s:      m.s,
		ct:     m.ct,
		nodes:  m.nodes,
		counts: append([]int(nil), m.counts...),
		path:   append([]int(nil), m.path...),
	}
}

// advance consumes name at the innermost position that admits it, working
// outward: the particle the last item was attributed to, then the rest of the
// iteration containing it, then a further iteration of that particle's group,
// then the same three questions one level out. Moving out is legal only while
// the level being left can be closed (canExit), which is where an unsatisfied
// {min occurs} stops the search rather than being noticed later.
func (m *Matcher) advance(name QName, kind admitKind) (Attribution, bool) {
	if len(m.path) == 0 {
		return m.enter(0, name, kind)
	}
	for d := len(m.path) - 1; d >= 0; d-- {
		t := m.clone()
		if a, ok := t.continueAt(d, name, kind); ok {
			m.counts, m.path = t.counts, t.path
			return a, true
		}
		if !m.canExit(d) {
			return nil, false
		}
	}
	return nil, false
}

// continueAt consumes name inside the node at depth d of the path, whose own
// deeper position has already failed to consume it and been closed.
func (m *Matcher) continueAt(d int, name QName, kind admitKind) (Attribution, bool) {
	i := m.path[d]
	g, isGroup := m.nodes[i].term.(ModelGroup)
	if !isGroup {
		a, ok := m.admits(i, name, kind)
		if !ok {
			return nil, false
		}
		m.counts[i]++
		return a, true
	}
	slot := m.slotOf(i, m.path[d+1])
	complete := m.iterationComplete(i, g, slot)
	if a, ok := m.continueIteration(d, g, slot, name, kind); ok {
		return a, true
	}
	if !complete || !m.canRepeat(i) {
		return nil, false
	}
	m.clearSubtree(i)
	m.path = m.path[:d+1]
	a, ok := m.enterBody(i, g, name, kind)
	if !ok {
		return nil, false
	}
	m.counts[i]++
	return a, true
}

// continueIteration consumes name later in the OPEN iteration of the group at
// depth d, whose member at slot has just been closed. Each compositor offers
// what §3.8.4.1 says follows: a sequence its remaining members while each
// skipped one is ·emptiable· (§3.8.4.1.1), a choice nothing at all, since an
// iteration of a choice is one member (§3.8.4.1.2), and an all group any member
// that has not reached its {max occurs}, since S1 × … × Sn interleaves them
// (§3.8.4.1.3).
func (m *Matcher) continueIteration(d int, g ModelGroup, slot int, name QName, kind admitKind) (Attribution, bool) {
	i := m.path[d]
	children := m.nodes[i].children
	switch g.Compositor() {
	case CompositorChoice:
		return nil, false
	case CompositorAll:
		m.path = m.path[:d+1]
		for _, c := range children {
			if a, ok := m.enter(c, name, kind); ok {
				return a, true
			}
		}
		return nil, false
	case CompositorSequence:
		m.path = m.path[:d+1]
		for _, c := range children[slot+1:] {
			if a, ok := m.enter(c, name, kind); ok {
				return a, true
			}
			if !m.emptiable(c) {
				return nil, false
			}
		}
		return nil, false
	default:
		panic("xsd: Matcher.continueIteration: non-exhaustive Compositor switch")
	}
}

// enter consumes name as the FIRST item of a fresh occurrence of the node at i,
// appending the nodes it descended through to the path. It leaves the walk
// state untouched when the name is not admitted, so a caller may try members in
// turn.
func (m *Matcher) enter(i int, name QName, kind admitKind) (Attribution, bool) {
	g, isGroup := m.nodes[i].term.(ModelGroup)
	if !isGroup {
		a, ok := m.admits(i, name, kind)
		if !ok {
			return nil, false
		}
		m.counts[i]++
		m.path = append(m.path, i)
		return a, true
	}
	if !m.canRepeat(i) {
		return nil, false
	}
	m.path = append(m.path, i)
	a, ok := m.enterBody(i, g, name, kind)
	if !ok {
		m.path = m.path[:len(m.path)-1]
		return nil, false
	}
	m.counts[i]++
	return a, true
}

// enterBody consumes name as the first item of one iteration of the group at i:
// the first member of a sequence that admits it, while every member skipped
// before it is ·emptiable· (§3.8.4.1.1), or any member of a choice (§3.8.4.1.2)
// or an all group (§3.8.4.1.3). Members are tried in document order, which
// decides nothing a second member could have decided differently — cos-nonambig
// has already rejected a group where two of them admit one name.
func (m *Matcher) enterBody(i int, g ModelGroup, name QName, kind admitKind) (Attribution, bool) {
	for _, c := range m.nodes[i].children {
		if a, ok := m.enter(c, name, kind); ok {
			return a, true
		}
		if g.Compositor() == CompositorSequence && !m.emptiable(c) {
			return nil, false
		}
	}
	return nil, false
}

// admits reports what the leaf particle at i attributes an item named name to,
// under one search pass's kind, or false when the particle is exhausted or does
// not admit the name.
//
// The element case takes cvc-accept clause 2.3.1 before clause 2.3.2, which is
// what makes the reported declaration the particle's own D and not the
// ·substituting declaration· S. Clause 2.3.2's other conjuncts — D top-level, D
// not blocking substitution, S ·substitutable· for D — are the whole of what
// inSubstitutionGroupOf decides (substitutiongroup.go), so no part of the
// clause is restated here.
//
// The wildcard case is cvc-wildcard (§3.10.4.1) in full, including the
// defined/sibling {disallowed names} keywords, which need the containing
// complex type — the reason ContentMatcher takes one.
func (m *Matcher) admits(i int, name QName, kind admitKind) (Attribution, bool) {
	if !m.canRepeat(i) {
		return nil, false
	}
	switch t := m.nodes[i].term.(type) {
	case ElementDeclaration:
		if kind != admitElements {
			return nil, false
		}
		if t.Name() == name {
			return t, true
		}
		if m.s.nameInSubstitutionGroupOf(name, t) {
			return t, true
		}
		return nil, false
	case Wildcard:
		if kind != admitWildcards {
			return nil, false
		}
		if !m.s.allowsElementWildcardName(t, m.ct, name) {
			return nil, false
		}
		return t, true
	case ModelGroup:
		panic("xsd: Matcher.admits: a model group is not a leaf of a flattened content model")
	default:
		panic("xsd: Matcher.admits: non-exhaustive Term switch")
	}
}

// canRepeat reports whether the node at i may take one more occurrence
// (cvc-accept clauses 1.2, 2.2 and 3.2, the {max occurs} half).
func (m *Matcher) canRepeat(i int) bool {
	max, bounded := m.nodes[i].occurs.Max()
	return !bounded || m.counts[i] < max
}

// canExit reports whether the node at depth d of the path can be left where the
// sequence stands: its open iteration closed, and its own occurrence count at
// {min occurs} — or short of it with an ·emptiable· body, since the iterations
// still owed can then each be empty (cvc-accept clauses 1.1, 2.1 and 3.1).
//
// A member of an ALL group is left without its {min occurs} being consulted at
// all. §3.8.4.1.3 makes an all group's language S1 × … × Sn, the INTERLEAVE of
// its members' own sequences, so a member is suspended and resumed rather than
// finished when the next item belongs to a sibling; what the member owes is
// owed to the group, and iterationComplete is where the group collects it from
// every member at once.
func (m *Matcher) canExit(d int) bool {
	i := m.path[d]
	if g, isGroup := m.nodes[i].term.(ModelGroup); isGroup {
		if !m.iterationComplete(i, g, m.slotOf(i, m.path[d+1])) {
			return false
		}
	}
	if m.inAllGroup(d) {
		return true
	}
	if m.counts[i] >= m.nodes[i].occurs.Min() {
		return true
	}
	return m.bodyEmptiable(i)
}

// inAllGroup reports whether the node at depth d of the path is a member of an
// all group.
func (m *Matcher) inAllGroup(d int) bool {
	if d == 0 {
		return false
	}
	g, isGroup := m.nodes[m.path[d-1]].term.(ModelGroup)
	return isGroup && g.Compositor() == CompositorAll
}

// iterationComplete reports whether the open iteration of the group at i, whose
// member at slot has just been closed, has contributed a whole word of the
// group's language: for a sequence every later member is skippable, for a
// choice the one member taken is the whole of it, and for an all group every
// member has reached its own {min occurs}.
func (m *Matcher) iterationComplete(i int, g ModelGroup, slot int) bool {
	children := m.nodes[i].children
	switch g.Compositor() {
	case CompositorSequence:
		for _, c := range children[slot+1:] {
			if !m.emptiable(c) {
				return false
			}
		}
		return true
	case CompositorChoice:
		return true
	case CompositorAll:
		for _, c := range children {
			if m.counts[c] < m.nodes[c].occurs.Min() {
				return false
			}
		}
		return true
	default:
		panic("xsd: Matcher.iterationComplete: non-exhaustive Compositor switch")
	}
}

// emptiable reports whether the particle at i ·accepts· the empty sequence
// (cos-group-emptiable §3.9.6.3 read over the particle rather than the group):
// either it need not occur at all, or one occurrence of it can be empty.
func (m *Matcher) emptiable(i int) bool {
	return m.nodes[i].occurs.Min() == 0 || m.bodyEmptiable(i)
}

// bodyEmptiable reports whether ONE occurrence of the particle at i can be
// empty. A leaf occurrence is one element information item and never is; a
// sequence or all group needs every member ·emptiable· and a choice needs one
// (§3.8.4.1.1-3), and an empty choice — which accepts nothing at all, not the
// empty sequence — needs none.
func (m *Matcher) bodyEmptiable(i int) bool {
	g, isGroup := m.nodes[i].term.(ModelGroup)
	if !isGroup {
		return false
	}
	children := m.nodes[i].children
	if g.Compositor() == CompositorChoice {
		for _, c := range children {
			if m.emptiable(c) {
				return true
			}
		}
		return false
	}
	for _, c := range children {
		if !m.emptiable(c) {
			return false
		}
	}
	return true
}

// clearSubtree resets the occurrence counters beneath i, which a new iteration
// of i starts over. i's own counter is the iteration count and is not touched.
func (m *Matcher) clearSubtree(i int) {
	for _, c := range m.nodes[i].children {
		m.counts[c] = 0
		m.clearSubtree(c)
	}
}

// slotOf reports the position of child among the {particles} of the group at i.
// It panics if child is not one of them: the path is built by descending
// through those {particles}, so a child that is not there is a walk bug and not
// an input the caller could have caused.
func (m *Matcher) slotOf(i, child int) int {
	for k, c := range m.nodes[i].children {
		if c == child {
			return k
		}
	}
	panic("xsd: Matcher.slotOf: path node is not a member of its parent group")
}
