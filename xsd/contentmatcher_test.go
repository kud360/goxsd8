package xsd

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests drive Matcher through the exported entry points alone —
// Schema.ContentMatcher, Matcher.Next, Matcher.Accepting — but live in the
// package because the fixtures they share (uq, uOccurs, uGroup, uLocal, …) are
// particleattribution_test.go's. Every schema below is FINALIZED, which is what
// makes the models cos-nonambig-clean: a fixture that stopped being
// unambiguous would fail at cmSchema rather than silently exercise a model the
// walk is not licensed over.

// cmSchema finalizes a schema whose complex type {urn:upa}ct has p as its
// {content type} particle, and returns it with that type.
func cmSchema(t *testing.T, p Particle, extra func(*SchemaBuilder)) (*Schema, ComplexType) {
	t.Helper()
	ct := uCT(t, uq("ct"), p)
	b := NewSchemaBuilder()
	b.AddType(uNamedType(t, uq("T")))
	b.AddType(uNamedType(t, uq("U")))
	if extra != nil {
		extra(b)
	}
	b.AddType(ct)
	s, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the fixture schema: %v", err)
	}
	return s, ct
}

// cmMatcher is cmSchema plus the matcher over the type, failing when
// ContentMatcher declines — a decline is a distinct outcome every test that
// wants one asserts on its own.
func cmMatcher(t *testing.T, p Particle, extra func(*SchemaBuilder)) *Matcher {
	t.Helper()
	s, ct := cmSchema(t, p, extra)
	m, ok := s.ContentMatcher(ct)
	if !ok {
		t.Fatalf("ContentMatcher declined a model the test needs decided")
	}
	return m
}

// cmFeed puts each local name to m in turn and returns the index of the first
// one it rejected, or len(names) when it took them all.
func cmFeed(t *testing.T, m *Matcher, names ...string) int {
	t.Helper()
	for i, n := range names {
		if _, ok := m.Next(uq(n)); !ok {
			return i
		}
	}
	return len(names)
}

// cmAccept feeds names and asserts the whole sequence is accepted.
func cmAccept(t *testing.T, m *Matcher, names ...string) {
	t.Helper()
	if i := cmFeed(t, m, names...); i != len(names) {
		t.Fatalf("Next rejected %s at position %d of %v, want the whole sequence taken", names[i], i, names)
	}
	if !m.Accepting() {
		t.Errorf("Accepting() = false after %v, want true", names)
	}
}

// cmLeaf is a particle over a LOCAL element declaration named local, all of
// whose fixtures share the named type T so that two same-named declarations in
// one model do not trip cos-element-consistent.
func cmLeaf(t *testing.T, local string, o Occurs) Particle {
	t.Helper()
	return uParticle(t, o, ResolvedTerm{Term: uLocal(t, uq(local), uq("T"))})
}

// cmGroup wraps a model group in a particle with the given occurrence range.
func cmGroup(t *testing.T, o Occurs, compositor Compositor, particles ...Particle) Particle {
	t.Helper()
	return uParticle(t, o, ResolvedTerm{Term: uGroup(t, compositor, particles...)})
}

// A sequence takes its members in order, and each item is ·attributed to· the
// particle that consumed it (§3.8.4.1.1, cvc-accept clause 2.3.1).
func TestMatcherTakesASequenceInOrder(t *testing.T) {
	m := cmMatcher(t, cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
		cmLeaf(t, "a", uOccurs(t, 1, 1)),
		cmLeaf(t, "b", uOccurs(t, 1, 1))), nil)

	a, ok := m.Next(uq("a"))
	if !ok {
		t.Fatal("Next(a) declined the first member of the sequence")
	}
	d, isDecl := a.(ElementDeclaration)
	if !isDecl {
		t.Fatalf("Next(a) attributed the item to %T, want an ElementDeclaration", a)
	}
	if d.Name() != uq("a") {
		t.Errorf("attributed to %s, want %s", d.Name(), uq("a"))
	}
	if m.Accepting() {
		t.Error("Accepting() = true with the second member of the sequence still owed")
	}
	cmAccept(t, m, "b")
}

// A child no live particle admits is rejected, and rejecting it changes
// nothing: the same name put again after a name that DOES fit still fails, and
// the sequence is judged as if the rejected name had never been offered.
func TestMatcherRejectsAnUnadmittedChildWithoutMoving(t *testing.T) {
	m := cmMatcher(t, cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
		cmLeaf(t, "a", uOccurs(t, 1, 1)),
		cmLeaf(t, "b", uOccurs(t, 1, 1))), nil)

	if _, ok := m.Next(uq("a")); !ok {
		t.Fatal("Next(a) declined the first member")
	}
	if _, ok := m.Next(uq("zzz")); ok {
		t.Fatal("Next(zzz) took a name no particle admits")
	}
	cmAccept(t, m, "b")
}

// A sequence that stops short of a particle it must satisfy is not accepted,
// though every item in it was taken.
func TestMatcherRejectsASequenceThatEndsShort(t *testing.T) {
	m := cmMatcher(t, cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
		cmLeaf(t, "a", uOccurs(t, 1, 1)),
		cmLeaf(t, "b", uOccurs(t, 2, 3))), nil)

	if i := cmFeed(t, m, "a", "b"); i != 2 {
		t.Fatalf("Next rejected position %d, want both items taken", i)
	}
	if m.Accepting() {
		t.Error("Accepting() = true with only one of b's two required occurrences taken")
	}
	cmAccept(t, m, "b")
}

// An occurrence range is a counter, not an unfolding: a repeated particle takes
// items up to its {max occurs} and no further, and the particle after it takes
// the next one.
func TestMatcherCountsOccurrences(t *testing.T) {
	m := cmMatcher(t, cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
		cmLeaf(t, "a", uOccurs(t, 1, 2)),
		cmLeaf(t, "b", uOccurs(t, 1, 1))), nil)

	cmAccept(t, m, "a", "a", "b")

	over := cmMatcher(t, cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
		cmLeaf(t, "a", uOccurs(t, 1, 2)),
		cmLeaf(t, "b", uOccurs(t, 1, 1))), nil)
	if i := cmFeed(t, over, "a", "a", "a"); i != 2 {
		t.Errorf("the third a was rejected at position %d, want position 2 ({max occurs} = 2)", i)
	}
}

// A repeated choice starts a new iteration once the member it took is closed
// (§3.8.4.1.2), and stops at the group's own {max occurs}.
func TestMatcherRepeatsAChoice(t *testing.T) {
	model := func() Particle {
		return cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
			cmGroup(t, uOccurs(t, 1, 3), CompositorChoice,
				cmLeaf(t, "a", uOccurs(t, 1, 1)),
				cmLeaf(t, "b", uOccurs(t, 1, 1))))
	}
	cmAccept(t, cmMatcher(t, model(), nil), "a", "b", "a")

	over := cmMatcher(t, model(), nil)
	if i := cmFeed(t, over, "a", "b", "a", "b"); i != 3 {
		t.Errorf("the fourth item was rejected at position %d, want position 3 ({max occurs} = 3)", i)
	}
}

// An all group's members interleave (§3.8.4.1.3): each keeps its own counter,
// so a member may be taken again after another one has been.
func TestMatcherInterleavesAnAllGroup(t *testing.T) {
	model := func() Particle {
		return cmGroup(t, uOccurs(t, 1, 1), CompositorAll,
			cmLeaf(t, "a", uOccurs(t, 1, 2)),
			cmLeaf(t, "b", uOccurs(t, 1, 1)))
	}
	cmAccept(t, cmMatcher(t, model(), nil), "a", "b", "a")

	short := cmMatcher(t, model(), nil)
	if i := cmFeed(t, short, "a"); i != 1 {
		t.Fatalf("Next rejected a at position %d", i)
	}
	if short.Accepting() {
		t.Error("Accepting() = true with the all group's b member never taken")
	}
}

// An all group's member is SUSPENDED, not finished, when the next item belongs
// to a sibling: §3.8.4.1.3's interleave lets a member owing more occurrences be
// left and resumed, and only the group collects what every member owes. A walk
// that read a member's {min occurs} on the way out would reject this sequence
// at its second item, c having taken one of the two it needs.
func TestMatcherResumesAnAllGroupMemberThatOwesOccurrences(t *testing.T) {
	model := func() Particle {
		return cmGroup(t, uOccurs(t, 1, 1), CompositorAll,
			cmLeaf(t, "a", uOccurs(t, 0, 5)),
			cmLeaf(t, "b", uOccurs(t, 1, 5)),
			uParticle(t, uUnbounded(t, 2), ResolvedTerm{Term: uLocal(t, uq("c"), uq("T"))}),
			cmLeaf(t, "d", uOccurs(t, 1, 1)))
	}
	cmAccept(t, cmMatcher(t, model(), nil), "a", "b", "d", "c", "a", "c", "c", "a", "a", "b")

	short := cmMatcher(t, model(), nil)
	if i := cmFeed(t, short, "c", "b", "d"); i != 3 {
		t.Fatalf("Next rejected position %d, want all three taken", i)
	}
	if short.Accepting() {
		t.Error("Accepting() = true with only one of c's two required occurrences taken")
	}
}

// Where an ·element particle· and a ·wildcard particle· both admit one name —
// the competition 1.1 permits — the item goes to the Element Declaration, which
// is cvc-accept's closing Note and PRINCIPLES 14. A name only the wildcard
// admits still goes to the wildcard.
func TestMatcherPrefersAnElementParticleToAWildcard(t *testing.T) {
	model := func() Particle {
		return cmGroup(t, uOccurs(t, 1, 1), CompositorChoice,
			cmLeaf(t, "a", uOccurs(t, 1, 1)),
			uParticle(t, uOccurs(t, 1, 1), ResolvedTerm{
				Term: uWildcard(t, NamespaceConstraintAny, nil, ProcessLax)}))
	}

	got, ok := cmMatcher(t, model(), nil).Next(uq("a"))
	if !ok {
		t.Fatal("Next(a) declined a name both particles admit")
	}
	if _, isDecl := got.(ElementDeclaration); !isDecl {
		t.Errorf("Next(a) attributed the item to %T, want the ElementDeclaration", got)
	}

	got, ok = cmMatcher(t, model(), nil).Next(uq("other"))
	if !ok {
		t.Fatal("Next(other) declined a name the wildcard admits")
	}
	if _, isWild := got.(Wildcard); !isWild {
		t.Errorf("Next(other) attributed the item to %T, want the Wildcard", got)
	}
}

// cvc-accept clause 2.3.2 admits an item whose name is in the ·substitution
// group· of the particle's own declaration D, and the attribution is D — the
// particle's declaration, not the ·substituting declaration·.
func TestMatcherAdmitsASubstitutionGroupMember(t *testing.T) {
	extra := func(b *SchemaBuilder) {
		b.AddElement(uGlobal(t, uq("head"), uq("T")))
		b.AddElement(uGlobal(t, uq("member"), uq("T"), uq("head")))
	}
	m := cmMatcher(t, cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
		uParticle(t, uOccurs(t, 1, 1), ElementDeclarationRef{Name: uq("head")})), extra)

	got, ok := m.Next(uq("member"))
	if !ok {
		t.Fatal("Next(member) declined a member of the head's ·substitution group·")
	}
	d, isDecl := got.(ElementDeclaration)
	if !isDecl {
		t.Fatalf("attributed the item to %T, want an ElementDeclaration", got)
	}
	if d.Name() != uq("head") {
		t.Errorf("attributed to %s, want the particle's own declaration %s", d.Name(), uq("head"))
	}
	if !m.Accepting() {
		t.Error("Accepting() = false after the one member the model requires")
	}
}

// A repeated group is greedy: it fills its open iteration before starting the
// next one. That is only a verdict where the iterations still owed can be
// empty, and cvc-accept counts an ·emptiable· body's unstarted iterations as
// satisfying {min occurs} — so (a?, b?){2,2} takes "a b" as one iteration and
// still accepts, while (a?, b){2,2}, whose body is not ·emptiable·, does not.
func TestMatcherFillsAnIterationBeforeStartingTheNext(t *testing.T) {
	emptiable := cmMatcher(t, cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
		cmGroup(t, uOccurs(t, 2, 2), CompositorSequence,
			cmLeaf(t, "a", uOccurs(t, 0, 1)),
			cmLeaf(t, "b", uOccurs(t, 0, 1)))), nil)
	cmAccept(t, emptiable, "a", "b")

	required := cmMatcher(t, cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
		cmGroup(t, uOccurs(t, 2, 2), CompositorSequence,
			cmLeaf(t, "a", uOccurs(t, 0, 1)),
			cmLeaf(t, "b", uOccurs(t, 1, 1)))), nil)
	if i := cmFeed(t, required, "a", "b"); i != 2 {
		t.Fatalf("Next rejected position %d of a b, want both taken", i)
	}
	if required.Accepting() {
		t.Error("Accepting() = true after one iteration of a group that requires two")
	}
	cmAccept(t, required, "b")
}

// An empty child sequence is accepted exactly when the content model's particle
// is ·emptiable·.
func TestMatcherAcceptsTheEmptySequenceOnlyWhenEmptiable(t *testing.T) {
	optional := cmMatcher(t, cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
		cmLeaf(t, "a", uOccurs(t, 0, 1))), nil)
	if !optional.Accepting() {
		t.Error("Accepting() = false for an empty sequence against an ·emptiable· model")
	}

	required := cmMatcher(t, cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
		cmLeaf(t, "a", uOccurs(t, 1, 1))), nil)
	if required.Accepting() {
		t.Error("Accepting() = true for an empty sequence against a model requiring one item")
	}
}

// The walk COUNTS occurrences instead of unfolding them, so an occurrence range
// no unfolding could materialize costs one counter. A model built by unfolding
// would allocate 200000 positions before reading a single child.
func TestMatcherCountsAHugeOccurrenceRangeRatherThanUnfoldingIt(t *testing.T) {
	const min, max = 100000, 200000
	m := cmMatcher(t, cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
		cmLeaf(t, "a", uOccurs(t, min, max)),
		cmLeaf(t, "b", uOccurs(t, 1, 1))), nil)

	for i := 0; i < min; i++ {
		if _, ok := m.Next(uq("a")); !ok {
			t.Fatalf("Next(a) rejected occurrence %d of %d", i+1, min)
		}
	}
	if m.Accepting() {
		t.Error("Accepting() = true with b still owed")
	}
	cmAccept(t, m, "b")
}

// The walk descends each optional level once and never revisits it. This model
// nests 40 ·emptiable· groups, so an implementation that backtracked over
// "taken or skipped" at every level would explore 2^40 combinations to place
// one item (PRINCIPLES 14); this one places it in 40 steps.
func TestMatcherDoesNotBacktrackOverNestedOptionalGroups(t *testing.T) {
	const depth = 40
	inner := cmGroup(t, uOccurs(t, 1, 1), CompositorSequence, cmLeaf(t, "z", uOccurs(t, 1, 1)))
	for i := depth; i > 0; i-- {
		inner = cmGroup(t, uOccurs(t, 0, 1), CompositorSequence,
			cmLeaf(t, "x"+itoa(i), uOccurs(t, 0, 1)),
			inner)
	}
	m := cmMatcher(t, cmGroup(t, uOccurs(t, 1, 1), CompositorSequence, inner), nil)

	cmAccept(t, m, "z")
}

// itoa spells a small non-negative int without pulling strconv into the test's
// import set for one call.
func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

// A {content type} holding no particle is not this driver's question:
// cvc-complex-type clauses 1.1 and 1.2 govern the empty and simple varieties
// directly.
func TestContentMatcherDeclinesAParticlelessContentType(t *testing.T) {
	b := NewSchemaBuilder()
	empty := uNamedType(t, uq("empty"))
	b.AddType(empty)
	s, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing: %v", err)
	}
	if _, ok := s.ContentMatcher(empty); ok {
		t.Error("ContentMatcher decided an empty {content type}")
	}
}

// {open content} splits the sequence in two (cvc-complex-content clauses 2 and
// 3), which this walk does not do: it declines rather than matching the
// {particle} half alone and rejecting every item the open wildcard would take.
func TestContentMatcherDeclinesOpenContent(t *testing.T) {
	oc, err := NewOpenContent(xsderr.Loc{}, OpenContentInterleave,
		uWildcard(t, NamespaceConstraintAny, nil, ProcessLax))
	if err != nil {
		t.Fatalf("NewOpenContent: %v", err)
	}
	p := cmGroup(t, uOccurs(t, 1, 1), CompositorSequence, cmLeaf(t, "a", uOccurs(t, 1, 1)))
	ct, err := NewComplexType(xsderr.Loc{}, uq("ct"), QName{}, nil, DerivationRestriction, false,
		nil, nil, nil, ElementContent{Particle: p, OpenContent: &oc}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType: %v", err)
	}
	b := NewSchemaBuilder()
	b.AddType(uNamedType(t, uq("T")))
	b.AddType(ct)
	s, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing: %v", err)
	}
	if _, ok := s.ContentMatcher(ct); ok {
		t.Error("ContentMatcher decided a {content type} with {open content}")
	}
}

// Two nested particles with a {max occurs} above 1 are cvc-accept's own named
// non-determinism, where a greedy walk can reject a sequence another partition
// accepts — (a{1,2}, b?){2,2} accepts "a a b" as (a)(a b). The construction
// declines rather than answering it wrongly.
func TestContentMatcherDeclinesNestedRepetition(t *testing.T) {
	p := cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
		cmGroup(t, uOccurs(t, 2, 2), CompositorSequence,
			cmLeaf(t, "a", uOccurs(t, 1, 2)),
			cmLeaf(t, "b", uOccurs(t, 0, 1))))
	s, ct := cmSchema(t, p, nil)
	if _, ok := s.ContentMatcher(ct); ok {
		t.Error("ContentMatcher decided a model with nested repeated particles")
	}
}

// An all group nested in an all group interleaves two member sets at once,
// which the per-member counters cannot express; cos-all-limited clause 1.3 is
// the only shape that reaches it, and it declines.
func TestContentMatcherDeclinesAnAllGroupInsideAnAllGroup(t *testing.T) {
	p := uParticle(t, uOccurs(t, 1, 1), ResolvedTerm{Term: uGroup(t, CompositorAll,
		cmLeaf(t, "a", uOccurs(t, 1, 1)),
		cmGroup(t, uOccurs(t, 1, 1), CompositorAll, cmLeaf(t, "b", uOccurs(t, 1, 1))))})
	s, ct := cmSchema(t, p, nil)
	if _, ok := s.ContentMatcher(ct); ok {
		t.Error("ContentMatcher decided an all group holding an all group")
	}
}
