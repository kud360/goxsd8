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
// # One particle per item, and more than one partition
//
// The walk never re-reads an item and never undoes one it has taken
// (PRINCIPLES 14). Two facts fix WHICH PARTICLE each item goes to, and neither
// is an assumption a comment makes:
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
// Neither settles WHICH ITERATION of a repeated ancestor an item falls in, the
// third non-determinism the same Note names: "nested particles each of which
// has {max occurs} greater than 1, where the input sequence can be partitioned
// in multiple ways ... there is no fixed rule for eliminating the
// non-determinism". cvc-accept clause 3.1 asks that question EXISTENTIALLY —
// "there is a partition of the sequence into n sub-sequences" — so a sequence
// is ·accepted· where SOME partition validates, and picking one partition and
// walking it answers a different question than the rule asks. A greedy pick —
// stay in the innermost open iteration for as long as it can — gets "a a b"
// wrong against (a{1,2}, b?){2,2}, which L(P) admits as the partition (a)(a b)
// and the greedy walk consumes as one iteration and then rejects.
//
// So the walk carries a SET of cursors, one per partition of the items so far
// that is still live, and a name is taken where any of them takes it. That is
// not a search: no cursor re-reads an item, no cursor is revisited, and the set
// is widened only where the greedy order is not already exact —
//
//   - a differing partition needs one particle P reachable both later in the
//     open iteration of some repeatable ancestor R and at the start of R's next
//     iteration (two DIFFERENT particles would ·compete·, which cos-nonambig has
//     already rejected);
//   - P starting R's body means every particle before it on its path is
//     skippable, so the shorter iteration the alternative closes before P holds
//     no mandatory particle of the body — and is a word of the body's language
//     only where the body has none at all, which is to say where it is
//     ·emptiable·, where the greedy pick is exact: its iteration count is never
//     ABOVE the alternative's, so no {max occurs} it satisfies is one the greedy
//     pick exceeds, and canExit lets the iterations still owed be empty, so
//     {min occurs} is met as well. The one way that iteration can hold a
//     mandatory particle is P ITSELF, taken again, which needs a repeating
//     particle inside a repeating one;
//   - so a cursor splits at a node R that repeats and holds such a P
//     (contentNode.ambiguous) and nowhere else. P has to stand at BOTH ends of
//     R's iteration boundary at once — reachable as the first item of a fresh
//     iteration, which needs every particle before it skippable, and reachable
//     with the open iteration already a whole word of the body, which needs
//     every particle after it skippable — so a body holding a mandatory
//     particle on either side of each of its repeating particles never splits,
//     however wide R's own occurrence range. A model with no such node carries
//     one cursor for the whole sequence and costs exactly what the greedy walk
//     cost.
//
// The set is bounded by the SCHEMA, never by the instance. A counter stops at
// its node's {max occurs}, or at its {min occurs} where {max occurs} is
// unbounded (cursor), because canRepeat and canExit are its only readers and
// neither can tell a larger value from the clamp — so two partitions that
// differ in nothing else are one cursor, and the set cannot outgrow the product
// of the clamped ranges over the widened subtrees. ContentMatcher computes that
// product and declines the models whose product is too large
// (maxPartitionStates), which is what makes one item's cost a constant of the
// schema rather than a function of the items already taken.
//
// # The {open content} split, and why it needs no search either
//
// Where the {content type}'s {open content} is PRESENT, cvc-complex-content
// (§3.4.4.3) clauses 2 and 3 decide the sequence S as two subsequences: an S1
// ·valid· with respect to {particle} (cvc-particle §3.9.4.2, clauses 2.2 and
// 3.2) and an S2 every member of which is ·valid· with respect to
// {open content}.{wildcard} (cvc-wildcard §3.10.4.1, clauses 2.4 and 3.4).
// Neither clause leaves the split open, so neither needs a search:
//
//   - clause 2.3 ({mode} suffix) admits an S2 only where S1 + E has no ·path·
//     in {particle} for S2's first element E (§3.8.4.1, key-path). Having a
//     path is prefix-closed, so exactly one S1 satisfies that: the longest
//     prefix of S that has one. A shorter prefix leaves 2.3 false for its own
//     first S2 item, and a longer one has no path.
//   - clause 3.3 ({mode} interleave) says the same of EVERY member E of S2,
//     against S3, the part of S1 preceding E. Each item's side is therefore
//     fixed by the items before it: {particle} takes the item wherever it can
//     extend, and the wildcard takes it only where {particle} provably cannot.
//
// That is PRINCIPLES 14's "explicit content beats an open-content wildcard at
// the current state" arriving as the clauses' own condition rather than as a
// tie-break this file invents. Next asks the two particle searches first and
// the open wildcard last, and an item the open wildcard takes leaves the walk
// state untouched, S2 being matched against that wildcard and nothing else.
// Clause 2.1's S = S1 + S2 is a concatenation, so suffix mode never offers
// {particle} another item once it has left it; clause 3.1's S1 × S2 is the
// interleave operator (§3.8.4.1.3), so interleave mode offers every item.

// Attribution is what one element information item of a matched child sequence
// is ·attributed to· (§3.4.4.4, key-att-to): the {term} of the particle
// [Matcher.Next] advanced over, or a present {open content} itself. It is a
// sealed sum (STYLE T2's closed-sum exception) with exactly three variants.
// [ElementDeclaration] (cvc-accept clause 2) and [Wildcard] (cvc-accept clause
// 1) are the two kinds of ·basic particle· an item can be attributed to — a
// Model Group is not one, and returning [Term] would make an attribution to one
// representable (STYLE T1). [*OpenContent] is the third, for an item
// cvc-complex-content clause 2.4 or 3.4 admitted, which §3.4.4.4 ·attributes
// to· the {open content} and not to any particle: the spec keeps the two apart
// throughout — §3.4.5.2 gives this case the [match information] keyword open
// where a ·wildcard particle· takes strict/lax/skip, and §3.4.6.4's
// key-dft-binding clauses 4-6 name them as alternative cases — so a consumer
// deciding a rule that quantifies over particles alone can decide it here.
//
// An [ElementDeclaration] result is the particle's own declaration D, which is
// the answer to "which particle consumed this item". For an item admitted
// through cvc-accept clause 2.3.2 the ·context-determined declaration· is the
// ·substituting declaration· S and not D; resolving S is the recursive
// assessment's job (§3.3.4.6), not this one's.
//
// A [Wildcard] result is the {term} of a ·wildcard particle· (§3.9.1, key-wp)
// and never the {wildcard} of an {open content}. That is the distinction
// e-validity clause 1.1.3 (§3.3.5.1) turns on: it quantifies over an item
// ·attributed to· a ***strict*** ·wildcard particle· and names no other
// component, so this arm decides it and the [*OpenContent] one does not.
//
// An [*OpenContent] result is the {open content} record of the {content type}
// being matched — never nil, and the same pointer [ElementContent] carries, so
// a consumer may compare it by identity as well as read {mode} and {wildcard}
// off it. A switch arm may therefore dereference it without a nil check:
// [Matcher.Next] reports no match at all where the {open content} is ·absent·.
type Attribution interface{ attribution() }

// attribution marks ElementDeclaration as an Attribution (cvc-accept clause
// 2.3); see the Attribution doc comment.
func (ElementDeclaration) attribution() {}

// attribution marks Wildcard as an Attribution (cvc-accept clause 1); see the
// Attribution doc comment.
func (Wildcard) attribution() {}

// attribution marks *OpenContent as an Attribution (§3.4.4.4: an item
// cvc-complex-content clause 2.4 or 3.4 admitted is ·attributed to· the {open
// content}, which is no particle); see the Attribution doc comment. The
// receiver is a POINTER so the arm has exactly one spelling — a value receiver
// would put the marker in both OpenContent's and *OpenContent's method sets,
// making two representations of one variant (STYLE T1) — and so that the arm
// carries the record's identity, which a copy would lose.
func (*OpenContent) attribution() {}

// contentNode is one particle of a flattened content model. children holds the
// node indices of a model group's {particles} in document order, and is nil for
// an ·element particle· or ·wildcard particle·, whose {term} is the leaf the
// walk matches names against.
//
// term is the RESOLVED {term}: the flattening follows an <element ref> through
// the element index and a <group ref> through the model group index, so no walk
// step re-resolves anything. Nodes are appended in preorder, so a node's index
// is less than every index in its subtree.
//
// ambiguous marks the one shape the greedy iteration boundary is not exact on
// (see the file comment): this node's {max occurs} is greater than 1, and its
// body holds a particle whose own {max occurs} is too and which the walk can
// reach both as the first item of a fresh iteration and with the open iteration
// already complete, so an item that particle takes can fall either side of the
// boundary. A cursor splits at such a node and at no other, which is why
// markAmbiguous computes it once here rather than the walk re-deriving it per
// item.
type contentNode struct {
	occurs    Occurs
	term      Term
	children  []int
	ambiguous bool
}

// cursor is one live partition of the items taken so far (cvc-accept clause
// 3.1): the occurrence counter of every node of the flattened model, and the
// path of nodes the last item was ·attributed to·, outermost first. A Matcher
// holds every cursor those items can have reached, and [Matcher.Accepting] asks
// its question of the set rather than of a chosen member.
//
// Counters are CLAMPED at counterCap, so a cursor records how a partition
// stands and not how it got there. Two partitions that differ only in counts
// neither canRepeat nor canExit can tell apart are the same cursor, and equal
// is what collapses them.
type cursor struct {
	counts []int
	path   []int
}

// clone copies the mutable walk state, so a search pass that fails leaves
// nothing behind and two cursors split from one share no array.
func (c *cursor) clone() cursor {
	return cursor{
		counts: append([]int(nil), c.counts...),
		path:   append([]int(nil), c.path...),
	}
}

// equal reports whether two cursors of one Matcher stand in the same place.
// Their counts slices are one per node of the same flattened model and so are
// the same length.
func (c *cursor) equal(o cursor) bool {
	if len(c.path) != len(o.path) {
		return false
	}
	for d, i := range c.path {
		if o.path[d] != i {
			return false
		}
	}
	for i, n := range c.counts {
		if o.counts[i] != n {
			return false
		}
	}
	return true
}

// addCursor appends t to cs unless cs already holds an equal cursor. Collapsing
// equal partitions is what holds the live set inside the product ContentMatcher
// bounded: without it two partitions that have converged would each go on
// splitting, and the set would grow with the instance.
func addCursor(cs []cursor, t cursor) []cursor {
	for _, c := range cs {
		if c.equal(t) {
			return cs
		}
	}
	return append(cs, t)
}

// Matcher advances one complex type's {content type} particle over an
// element-information-item sequence, deciding cvc-particle (§3.9.4.2) for the
// sequence one item at a time. Obtain one from [Schema.ContentMatcher]; the
// zero value is not usable.
//
// A Matcher is single-use and stateful: it holds every position in the content
// model the items so far can have reached (cursor) and, under a {mode} suffix
// {open content}, whether the sequence has left clause 2's S1 for its S2 — so
// the caller feeds it one element's [[children]] in document order and drops
// it. It is not safe for concurrent use, and nothing in it is shared with the
// schema beyond the immutable components the flattening read.
type Matcher struct {
	s     *Schema
	ct    ComplexType
	open  *OpenContent
	nodes []contentNode
	live  []cursor
	inS2  bool
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
// A present {open content} is DECIDED rather than declined: the Matcher holds
// it and [Matcher.Next] offers the open wildcard whatever {particle} cannot
// take, per cvc-complex-content clauses 2 and 3 (see the file comment).
//
// The two declines, none of them a violation:
//
//   - a {content type} whose {variety} is empty or simple, which holds no
//     particle at all. cvc-complex-type clauses 1.1 and 1.2 govern those
//     directly and need no matcher.
//   - GAP(xsd): an ·ambiguous· node whose widened subtrees' clamped occurrence
//     ranges admit more than maxPartitionStates partitions at once. The SHAPE
//     is decided — (a{1,2}, b?){2,2} takes "a a b" — and what stays declined is
//     the width: (a{1,500}){1,500} really does reach a quarter of a million
//     live partitions, within a percent of the product that bounds it, so no
//     tighter ESTIMATE reaches this shape. Only a state encoding carrying the
//     partitions as counter INTERVALS rather than one cursor each does, and
//     even that wants a ceiling above 500. Declining withholds the whole
//     element-sequence verdict, whose consumers are validate's
//     Result.violations and its one reader Result.Violations, both of which
//     carry violations PRESENT — so the decline costs a rejection and
//     manufactures none. #1557 owns its retirement.
func (s *Schema) ContentMatcher(t ComplexType) (*Matcher, bool) {
	ec, ok := t.ContentType().(ElementContent)
	if !ok {
		return nil, false
	}
	m := &Matcher{s: s, ct: t, open: ec.OpenContent}
	if _, ok := m.flatten(ec.Particle); !ok {
		return nil, false
	}
	m.markAmbiguous(0)
	if !m.partitionsBounded() {
		return nil, false
	}
	m.live = []cursor{{counts: make([]int, len(m.nodes))}}
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

// markAmbiguous sets contentNode.ambiguous over the subtree at i.
func (m *Matcher) markAmbiguous(i int) {
	for _, c := range m.nodes[i].children {
		m.markAmbiguous(c)
	}
	m.nodes[i].ambiguous = repeatable(m.nodes[i].occurs) && m.splitsInBody(i)
}

// splitsInBody reports whether one iteration of the group at i holds the
// particle P the file comment's first bullet asks for: one that repeats and
// stands at BOTH ends of i's iteration boundary, so that an item it takes can
// be the next one of the open iteration or the first of a fresh one. Those are
// exactly repeat's two conditions — enterBody reaching P, and iterationComplete
// holding where P stands — so a node whose body holds no such particle never
// widens the live set, however large its occurrence range.
func (m *Matcher) splitsInBody(i int) bool {
	g, isGroup := m.nodes[i].term.(ModelGroup)
	if !isGroup {
		return false
	}
	return m.splitsAmong(i, g, true, true)
}

// splitsAmong searches the {particles} of the group g at i for that particle.
// first and last say whether the walk can still reach this group's own members
// as the first item of the enclosing iteration and with that iteration already
// complete; a sequence narrows both by what it puts before and after each
// member (§3.8.4.1.1), while a choice takes one member whole (§3.8.4.1.2) and
// an all group interleaves them (§3.8.4.1.3), so neither narrows either.
//
// An all group's iterationComplete asks more than the members after the open
// one — it collects every member's own {min occurs} — so leaving last alone
// there over-counts the nodes that widen and never undercounts them.
func (m *Matcher) splitsAmong(i int, g ModelGroup, first, last bool) bool {
	children := m.nodes[i].children
	for k, ch := range children {
		f, l := first, last
		if g.Compositor() == CompositorSequence {
			f = f && m.allEmptiable(children[:k])
			l = l && m.allEmptiable(children[k+1:])
		}
		if f && l && repeatable(m.nodes[ch].occurs) {
			return true
		}
		if cg, isGroup := m.nodes[ch].term.(ModelGroup); isGroup && m.splitsAmong(ch, cg, f, l) {
			return true
		}
	}
	return false
}

// allEmptiable reports whether every particle of ps ·accepts· the empty
// sequence, which is what makes the ones before a member skippable on the way
// in and the ones after it skippable on the way out.
func (m *Matcher) allEmptiable(ps []int) bool {
	for _, p := range ps {
		if !m.emptiable(p) {
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

// maxPartitionStates is the ceiling on the cursors one Matcher will carry, and
// so on what one item of the instance costs: a name is put to every live cursor
// in turn. ContentMatcher declines a model that could exceed it rather than a
// Matcher declining a name mid-sequence.
const maxPartitionStates = 256

// partitionsBounded reports whether the live set provably stays inside
// maxPartitionStates. Every live cursor stands at the same ·basic particle· —
// two that did not would ·compete· for the name that put them there, which
// cos-nonambig has already rejected — so they differ only in the counters an
// ·ambiguous· node's iteration boundary moves, which are its own and its
// subtree's. A clamped counter takes counterCap+1 values, so the product of
// those over the widened subtrees bounds the cursors that can be live at once.
//
// The product is of {max occurs} values and would overflow for a model that
// combines large ones, so it is never formed: each factor is checked against
// the room the running product leaves, and a factor with no room ends the count
// at once.
func (m *Matcher) partitionsBounded() bool {
	widened := make([]bool, len(m.nodes))
	m.markWidened(0, false, widened)
	states := 1
	for i, w := range widened {
		if !w {
			continue
		}
		values := m.counterCap(i) + 1
		if values > maxPartitionStates || states > maxPartitionStates/values {
			return false
		}
		states *= values
	}
	return true
}

// markWidened marks every node of the subtree at i that lies inside an
// ·ambiguous· node's subtree, i's own ambiguity included. inside says whether an
// ancestor of i is one.
func (m *Matcher) markWidened(i int, inside bool, widened []bool) {
	inside = inside || m.nodes[i].ambiguous
	widened[i] = inside
	for _, c := range m.nodes[i].children {
		m.markWidened(c, inside, widened)
	}
}

// counterCap is the occurrence count beyond which canRepeat and canExit stop
// changing their answers: {max occurs} where it is a number, and {min occurs}
// where it is unbounded and only the {min occurs} half of cvc-accept clauses
// 1.1, 2.1 and 3.1 is left to decide.
func (m *Matcher) counterCap(i int) int {
	if max, bounded := m.nodes[i].occurs.Max(); bounded {
		return max
	}
	return m.nodes[i].occurs.Min()
}

// count records one more occurrence of the node at i in c, clamped at
// counterCap so that a partition's cursor records where it stands rather than
// how far it has gone (cursor).
func (m *Matcher) count(c *cursor, i int) {
	if c.counts[i] < m.counterCap(i) {
		c.counts[i]++
	}
}

// Next advances the content model over one element information item whose
// ·expanded name· is name, reporting what the item is ·attributed to·
// (§3.4.4.4). It reports (nil, false) when no particle live under any partition
// of the items already taken admits the name, which is the cvc-accept
// (§3.9.4.3) rejection its caller charges against that item's own location; the
// Matcher is unchanged by a rejected name, so a caller may stop at the first one
// or keep feeding.
//
// The first two searches are cvc-accept's element/wildcard precedence: every
// live particle of every live partition is offered the name as an ·element
// particle· first (clause 2.3.1's expanded-name match, then clause 2.3.2's
// ·substitution group· membership), and only a name no element particle
// admits is offered to the wildcard particles (clause 1, cvc-wildcard
// §3.10.4.1 in full).
//
// A third search follows them where the {content type}'s {open content} is
// present, and is reachable only once both have failed — which is exactly
// cvc-complex-content clause 2.3's and 3.3's "has no ·path· in {particle}"
// (openNext, and the file comment for why the resulting split is the only one
// those clauses admit).
func (m *Matcher) Next(name QName) (Attribution, bool) {
	if m.inS2 {
		return m.openNext(name)
	}
	if a, ok := m.step(name, admitElements); ok {
		return a, true
	}
	if a, ok := m.step(name, admitWildcards); ok {
		return a, true
	}
	return m.openNext(name)
}

// openNext offers name to {open content}.{wildcard}, the third tier
// cvc-complex-content clauses 2.4 and 3.4 add over cvc-accept's two, and
// reports the admitted item as ·attributed to· the {open content} (§3.4.4.4) —
// the record itself, on [Attribution]'s terms, and not its {wildcard}.
//
// The walk is not advanced: an S2 member is matched against the wildcard alone,
// so {particle} stands where S1 left it and [Matcher.Accepting] still asks
// clause 2.2's and 3.2's question of S1 and of nothing else. In {mode} suffix
// an admitted item also closes S1 for good (clause 2.1's S = S1 + S2 being a
// concatenation), which is the one thing an admitted name changes beyond the
// answer. A REJECTED name changes nothing at all, on [Matcher.Next]'s terms.
func (m *Matcher) openNext(name QName) (Attribution, bool) {
	if m.open == nil {
		return nil, false
	}
	if !m.s.allowsElementWildcardName(m.open.Wildcard(), m.ct, name) {
		return nil, false
	}
	if m.open.Mode() == OpenContentSuffix {
		m.inS2 = true
	}
	return m.open, true
}

// Accepting reports whether the sequence fed so far is ·accepted· by the
// content model — whether every open particle can be closed where the sequence
// stopped. A false result is the "the sequence ends short of a particle it must
// satisfy" half of cvc-accept, which the caller charges against the containing
// element rather than any child, there being no child at the offending
// position.
//
// Under a present {open content} the sequence asked about is S1 and not every
// item fed: cvc-complex-content clauses 2.2 and 3.2 put S1 alone to
// cvc-particle, and the items the open wildcard took are S2, which clauses 2.4
// and 3.4 have already decided one at a time.
//
// SOME partition has to close, not every one: cvc-accept clause 3.1 asks for
// the existence of one, so a sequence the greedy partition leaves owing a
// particle is still ·accepted· where another live partition owes nothing.
func (m *Matcher) Accepting() bool {
	for i := range m.live {
		if m.accepts(&m.live[i]) {
			return true
		}
	}
	return false
}

// accepts reports whether one partition closes where the sequence stopped:
// every node on its path can be left (canExit), or it took no item at all and
// the model is ·emptiable·.
func (m *Matcher) accepts(c *cursor) bool {
	if len(c.path) == 0 {
		return m.emptiable(0)
	}
	for d := len(c.path) - 1; d >= 0; d-- {
		if !m.canExit(c, d) {
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

// step runs one search pass over every live cursor, replacing the live set with
// the cursors that took the name and leaving it untouched where none did.
//
// Every cursor that takes the name ·attributes· it to the same ·basic
// particle·: two different ones live for one name would ·compete·, which
// cos-nonambig has already rejected, so the first attribution is the
// attribution and the partitions differ in nothing the caller can see.
func (m *Matcher) step(name QName, kind admitKind) (Attribution, bool) {
	var taken Attribution
	var next []cursor
	for i := range m.live {
		a, cs := m.advance(&m.live[i], name, kind)
		if len(cs) == 0 {
			continue
		}
		if taken == nil {
			taken = a
		}
		for _, c := range cs {
			next = addCursor(next, c)
		}
	}
	if taken == nil {
		return nil, false
	}
	m.live = next
	return taken, true
}

// advance collects every cursor reachable from c by consuming name. It works
// outward exactly as one greedy walk does — the particle the last item was
// attributed to, then the rest of the iteration containing it, then a further
// iteration of that particle's group, then the same three questions one level
// out — and moving out is legal only while the level being left can be closed
// (canExit), which is where an unsatisfied {min occurs} stops the search rather
// than being noticed later.
//
// What the set adds to that walk is confined to an ·ambiguous· node, where BOTH
// answers are kept: the cursor that stays in the open iteration and the cursor
// that closes it and starts the next. Everywhere else the first answer is the
// only one (see the file comment), and the search ends as soon as no ·ambiguous·
// node is left to widen at.
func (m *Matcher) advance(c *cursor, name QName, kind admitKind) (Attribution, []cursor) {
	if len(c.path) == 0 {
		t := c.clone()
		a, ok := m.enter(&t, 0, name, kind)
		if !ok {
			return nil, nil
		}
		return a, []cursor{t}
	}
	var taken Attribution
	var out []cursor
	for d := len(c.path) - 1; d >= 0; d-- {
		t := c.clone()
		if a, ok := m.continueIn(&t, d, name, kind); ok {
			taken, out = a, append(out, t)
		}
		if len(out) == 0 || m.nodes[c.path[d]].ambiguous {
			r := c.clone()
			if a, ok := m.repeat(&r, d, name, kind); ok {
				taken, out = a, append(out, r)
			}
		}
		if len(out) > 0 && !m.widensAbove(c, d) {
			return taken, out
		}
		if !m.canExit(c, d) {
			return taken, out
		}
	}
	return taken, out
}

// widensAbove reports whether c's path holds an ·ambiguous· node shallower than
// depth d. Where it does not, a cursor already found is the only one the rest of
// the path can reach, so a model with no nested repetition costs one cursor and
// one pass out through its path, which is what the walk cost when this shape was
// declined.
func (m *Matcher) widensAbove(c *cursor, d int) bool {
	for _, i := range c.path[:d] {
		if m.nodes[i].ambiguous {
			return true
		}
	}
	return false
}

// continueIn consumes name inside the OPEN occurrence of the node at depth d of
// c's path, whose own deeper position has already failed to consume it and been
// closed: one more occurrence of a leaf, or a later member of a group's open
// iteration.
func (m *Matcher) continueIn(c *cursor, d int, name QName, kind admitKind) (Attribution, bool) {
	i := c.path[d]
	g, isGroup := m.nodes[i].term.(ModelGroup)
	if !isGroup {
		a, ok := m.admits(c, i, name, kind)
		if !ok {
			return nil, false
		}
		m.count(c, i)
		return a, true
	}
	return m.continueIteration(c, d, g, m.slotOf(i, c.path[d+1]), name, kind)
}

// repeat consumes name as the first item of a FRESH iteration of the group at
// depth d of c's path, which needs that group's open iteration to be a whole
// word of its language already (iterationComplete) and its {max occurs} to admit
// another (cvc-accept clause 3.2). A leaf has no iteration to close — its
// occurrences are continueIn's counter — so it never repeats this way.
func (m *Matcher) repeat(c *cursor, d int, name QName, kind admitKind) (Attribution, bool) {
	i := c.path[d]
	g, isGroup := m.nodes[i].term.(ModelGroup)
	if !isGroup {
		return nil, false
	}
	if !m.iterationComplete(c, i, g, m.slotOf(i, c.path[d+1])) {
		return nil, false
	}
	if !m.canRepeat(c, i) {
		return nil, false
	}
	m.clearSubtree(c, i)
	c.path = c.path[:d+1]
	a, ok := m.enterBody(c, i, g, name, kind)
	if !ok {
		return nil, false
	}
	m.count(c, i)
	return a, true
}

// continueIteration consumes name later in the OPEN iteration of the group at
// depth d, whose member at slot has just been closed. Each compositor offers
// what §3.8.4.1 says follows: a sequence its remaining members while each
// skipped one is ·emptiable· (§3.8.4.1.1), a choice nothing at all, since an
// iteration of a choice is one member (§3.8.4.1.2), and an all group any member
// that can still take the item — one short of its {max occurs}, or a nested all
// group with an occurrence open to resume (offerAllMembers) — since
// S1 × … × Sn interleaves them (§3.8.4.1.3).
func (m *Matcher) continueIteration(c *cursor, d int, g ModelGroup, slot int, name QName, kind admitKind) (Attribution, bool) {
	children := m.nodes[c.path[d]].children
	switch g.Compositor() {
	case CompositorChoice:
		return nil, false
	case CompositorAll:
		i := c.path[d]
		c.path = c.path[:d+1]
		return m.offerAllMembers(c, i, name, kind)
	case CompositorSequence:
		c.path = c.path[:d+1]
		for _, ch := range children[slot+1:] {
			if a, ok := m.enter(c, ch, name, kind); ok {
				return a, true
			}
			if !m.emptiable(ch) {
				return nil, false
			}
		}
		return nil, false
	default:
		panic("xsd: Matcher.continueIteration: non-exhaustive Compositor switch")
	}
}

// offerAllMembers offers name to each member of the all group at i in turn,
// which §3.8.4.1.3 licenses whatever member the last item went to: L(M) is
// S1 × … × Sn, the INTERLEAVE of the members' own sequences, so nothing
// requires one member's items to be contiguous.
//
// A member that is itself a model group — cos-all-limited (§3.8.6.2) clause 2
// admits an all group there and nothing else — is RESUMED where it already has
// an occurrence open and entered where it does not, and its occurrence counter
// is the whole of that discriminator: clause 1.3 pins the member to
// {min occurs} = {max occurs} = 1, so a non-zero count is the one open
// occurrence it will ever have.
func (m *Matcher) offerAllMembers(c *cursor, i int, name QName, kind admitKind) (Attribution, bool) {
	for _, ch := range m.nodes[i].children {
		if _, isGroup := m.nodes[ch].term.(ModelGroup); isGroup && c.counts[ch] > 0 {
			if a, ok := m.resumeAll(c, ch, name, kind); ok {
				return a, true
			}
			continue
		}
		if a, ok := m.enter(c, ch, name, kind); ok {
			return a, true
		}
	}
	return nil, false
}

// resumeAll consumes name inside the occurrence of the nested all group at i
// that an earlier item already opened, putting i back on the path with its
// counters as they stand and offering the name to its own members — to whatever
// depth clause 1.3's exactly-once nesting reaches. Neither the occurrence
// counter nor the subtree beneath it moves: this is the same occurrence
// suspended by an item that went to a sibling, not a new one. It leaves the
// path untouched when the name is not admitted, as enter does.
func (m *Matcher) resumeAll(c *cursor, i int, name QName, kind admitKind) (Attribution, bool) {
	c.path = append(c.path, i)
	a, ok := m.offerAllMembers(c, i, name, kind)
	if !ok {
		c.path = c.path[:len(c.path)-1]
		return nil, false
	}
	return a, true
}

// enter consumes name as the FIRST item of a fresh occurrence of the node at i,
// appending the nodes it descended through to the path. It leaves the walk
// state untouched when the name is not admitted, so a caller may try members in
// turn.
func (m *Matcher) enter(c *cursor, i int, name QName, kind admitKind) (Attribution, bool) {
	g, isGroup := m.nodes[i].term.(ModelGroup)
	if !isGroup {
		a, ok := m.admits(c, i, name, kind)
		if !ok {
			return nil, false
		}
		m.count(c, i)
		c.path = append(c.path, i)
		return a, true
	}
	if !m.canRepeat(c, i) {
		return nil, false
	}
	c.path = append(c.path, i)
	a, ok := m.enterBody(c, i, g, name, kind)
	if !ok {
		c.path = c.path[:len(c.path)-1]
		return nil, false
	}
	m.count(c, i)
	return a, true
}

// enterBody consumes name as the first item of one iteration of the group at i:
// the first member of a sequence that admits it, while every member skipped
// before it is ·emptiable· (§3.8.4.1.1), or any member of a choice (§3.8.4.1.2)
// or an all group (§3.8.4.1.3). Members are tried in document order, which
// decides nothing a second member could have decided differently — cos-nonambig
// has already rejected a group where two of them admit one name.
func (m *Matcher) enterBody(c *cursor, i int, g ModelGroup, name QName, kind admitKind) (Attribution, bool) {
	for _, ch := range m.nodes[i].children {
		if a, ok := m.enter(c, ch, name, kind); ok {
			return a, true
		}
		if g.Compositor() == CompositorSequence && !m.emptiable(ch) {
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
func (m *Matcher) admits(c *cursor, i int, name QName, kind admitKind) (Attribution, bool) {
	if !m.canRepeat(c, i) {
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
func (m *Matcher) canRepeat(c *cursor, i int) bool {
	max, bounded := m.nodes[i].occurs.Max()
	return !bounded || c.counts[i] < max
}

// canExit reports whether the node at depth d of the path can be left where the
// sequence stands: its open iteration closed, and its own occurrence count at
// {min occurs} — or short of it with an ·emptiable· body, since the iterations
// still owed can then each be empty (cvc-accept clauses 1.1, 2.1 and 3.1).
//
// A member of an ALL group is left without its {min occurs} or its own open
// iteration being consulted at all. §3.8.4.1.3 makes an all group's language
// S1 × … × Sn, the INTERLEAVE of its members' own sequences, so a member is
// suspended and resumed rather than finished when the next item belongs to a
// sibling; what the member owes is owed to the group, and iterationComplete is
// where the group collects it from every member at once.
func (m *Matcher) canExit(c *cursor, d int) bool {
	i := c.path[d]
	if m.inAllGroup(c, d) {
		return true
	}
	if g, isGroup := m.nodes[i].term.(ModelGroup); isGroup {
		if !m.iterationComplete(c, i, g, m.slotOf(i, c.path[d+1])) {
			return false
		}
	}
	if c.counts[i] >= m.nodes[i].occurs.Min() {
		return true
	}
	return m.bodyEmptiable(i)
}

// inAllGroup reports whether the node at depth d of c's path is a member of an
// all group.
func (m *Matcher) inAllGroup(c *cursor, d int) bool {
	if d == 0 {
		return false
	}
	g, isGroup := m.nodes[c.path[d-1]].term.(ModelGroup)
	return isGroup && g.Compositor() == CompositorAll
}

// iterationComplete reports whether the open iteration of the group at i, whose
// member at slot has just been closed, has contributed a whole word of the
// group's language: for a sequence every later member is skippable, for a
// choice the one member taken is the whole of it, and for an all group every
// member has reached its own {min occurs}.
func (m *Matcher) iterationComplete(c *cursor, i int, g ModelGroup, slot int) bool {
	children := m.nodes[i].children
	switch g.Compositor() {
	case CompositorSequence:
		for _, ch := range children[slot+1:] {
			if !m.emptiable(ch) {
				return false
			}
		}
		return true
	case CompositorChoice:
		return true
	case CompositorAll:
		for _, ch := range children {
			if !m.memberSatisfied(c, ch) {
				return false
			}
		}
		return true
	default:
		panic("xsd: Matcher.iterationComplete: non-exhaustive Compositor switch")
	}
}

// memberSatisfied reports whether the member at i of an all group owes that
// group nothing where the sequence stands: its occurrence count is at
// {min occurs}, or short of it with an ·emptiable· body so the occurrences
// still owed can each be empty (cvc-accept clauses 1.1, 2.1 and 3.1).
//
// A member that is itself an all group — cos-all-limited (§3.8.6.2) clause 2
// admits no other kind there — owes what its OWN members owe as well, since
// §3.8.4.1.3 folds its language into the same interleave rather than closing it
// where the last of its items fell. That is the debt canExit declines to
// collect from a member of an all group, recursively over whatever depth clause
// 1.3 permits. A leaf has no members, so the recursion ends there.
func (m *Matcher) memberSatisfied(c *cursor, i int) bool {
	if c.counts[i] < m.nodes[i].occurs.Min() && !m.bodyEmptiable(i) {
		return false
	}
	for _, ch := range m.nodes[i].children {
		if !m.memberSatisfied(c, ch) {
			return false
		}
	}
	return true
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

// clearSubtree resets the occurrence counters beneath i in c, which a new
// iteration of i starts over. i's own counter is the iteration count and is not
// touched.
func (m *Matcher) clearSubtree(c *cursor, i int) {
	for _, ch := range m.nodes[i].children {
		c.counts[ch] = 0
		m.clearSubtree(c, ch)
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
