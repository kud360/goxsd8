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

// A repeated group whose body holds no repeating particle fills its open
// iteration before starting the next, and carries one cursor doing it: no other
// partition of the items accepts what that one rejects. cvc-accept counts an
// ·emptiable· body's unstarted iterations as satisfying {min occurs} — so
// (a?, b?){2,2} takes "a b" as one iteration and still accepts, while
// (a?, b){2,2}, whose body is not ·emptiable·, does not.
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

// cmOpenMatcher is cmMatcher over a {content type} whose {open content} is
// PRESENT, carrying mode and the wildcard w. The type is built here rather than
// through uCT because that fixture takes no {open content}, and the schema is
// FINALIZED on cmSchema's terms.
func cmOpenMatcher(t *testing.T, mode OpenContentMode, w Wildcard, p Particle) *Matcher {
	t.Helper()
	oc, err := NewOpenContent(xsderr.Loc{}, mode, w)
	if err != nil {
		t.Fatalf("NewOpenContent: %v", err)
	}
	ct, err := NewComplexType(xsderr.Loc{}, uq("ct"), QName{}, nil, DerivationRestriction, false,
		nil, nil, nil, ElementContent{Particle: p, OpenContent: &oc}, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType: %v", err)
	}
	b := NewSchemaBuilder()
	b.AddType(uNamedType(t, uq("T")))
	b.AddType(ct)
	s, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the fixture schema: %v", err)
	}
	m, ok := s.ContentMatcher(ct)
	if !ok {
		t.Fatalf("ContentMatcher declined a {content type} whose {open content} is present")
	}
	return m
}

// cmNext takes one name and asserts it was taken, returning what it was
// ·attributed to·.
func cmNext(t *testing.T, m *Matcher, local string) Attribution {
	t.Helper()
	a, ok := m.Next(uq(local))
	if !ok {
		t.Fatalf("Next(%s) rejected a name the model admits", local)
	}
	return a
}

// cmWantDeclaration asserts the item was ·attributed to· the element
// declaration named local — the {particle} half, S1.
func cmWantDeclaration(t *testing.T, a Attribution, local string) {
	t.Helper()
	d, isDecl := a.(ElementDeclaration)
	if !isDecl {
		t.Fatalf("attributed to %T, want the ElementDeclaration %s", a, local)
	}
	if d.Name() != uq(local) {
		t.Errorf("attributed to %s, want %s", d.Name(), uq(local))
	}
}

// cmWantWildcard asserts the item was ·attributed to· a Wildcard whose
// {process contents} is pc, which is how these fixtures tell the {open
// content}'s wildcard from a ·wildcard particle·'s: the two are separate
// ·attributions· (§3.4.4.4) that [Attribution] spells with one variant.
func cmWantWildcard(t *testing.T, a Attribution, pc ProcessContents) {
	t.Helper()
	w, isWild := a.(Wildcard)
	if !isWild {
		t.Fatalf("attributed to %T, want a Wildcard", a)
	}
	if w.ProcessContents() != pc {
		t.Errorf("attributed to a %s wildcard, want the %s one", w.ProcessContents(), pc)
	}
}

// cmOpenWildcard is the {open content} wildcard these fixtures share: it admits
// every name, so what a test's items are ·attributed to· turns on the
// {particle} alone. Its {process contents} is skip, which no fixture below
// gives a ·wildcard particle·.
func cmOpenWildcard(t *testing.T) Wildcard {
	t.Helper()
	return uWildcard(t, NamespaceConstraintAny, nil, ProcessSkip)
}

// Under {mode} suffix, S1 is the longest prefix the {particle} takes and S2 is
// everything after it, ·attributed to· the {open content} (cvc-complex-content
// clause 2, §3.4.4.4). S1 still has to satisfy the {particle}, which
// Accepting reports.
func TestMatcherTakesASuffixOfOpenContent(t *testing.T) {
	m := cmOpenMatcher(t, OpenContentSuffix, cmOpenWildcard(t),
		cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
			cmLeaf(t, "a", uOccurs(t, 1, 1)),
			cmLeaf(t, "b", uOccurs(t, 1, 1))))

	cmWantDeclaration(t, cmNext(t, m, "a"), "a")
	cmWantDeclaration(t, cmNext(t, m, "b"), "b")
	cmWantWildcard(t, cmNext(t, m, "x"), ProcessSkip)
	cmWantWildcard(t, cmNext(t, m, "y"), ProcessSkip)
	if !m.Accepting() {
		t.Error("Accepting() = false for an S1 the {particle} took whole")
	}
}

// Clause 2.1's S = S1 + S2 is a CONCATENATION, so suffix mode never offers the
// {particle} another item once an open-content item has started S2: the b here
// goes to the {open content} and the {particle} is left owing it, which
// Accepting charges. Interleave mode takes the same sequence (the test below),
// which is the whole difference between the two {mode}s.
func TestMatcherSuffixOpenContentNeverReturnsToTheParticle(t *testing.T) {
	m := cmOpenMatcher(t, OpenContentSuffix, cmOpenWildcard(t),
		cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
			cmLeaf(t, "a", uOccurs(t, 1, 1)),
			cmLeaf(t, "b", uOccurs(t, 1, 1))))

	cmWantDeclaration(t, cmNext(t, m, "a"), "a")
	cmWantWildcard(t, cmNext(t, m, "x"), ProcessSkip)
	cmWantWildcard(t, cmNext(t, m, "b"), ProcessSkip)
	if m.Accepting() {
		t.Error("Accepting() = true with the b particle left unsatisfied by S1")
	}
}

// Clause 3.1's S1 × S2 is the interleave operator, so an open-content item is
// not a suffix of anything: the {particle} takes the item after it exactly as
// if the item had not arrived (clause 3.3 evaluated against S3, the part of S1
// before it). This sequence is the minimum discriminator between the two
// {mode}s — clause 2 rejects it, clause 3 accepts it.
func TestMatcherInterleavesOpenContentWithTheParticle(t *testing.T) {
	m := cmOpenMatcher(t, OpenContentInterleave, cmOpenWildcard(t),
		cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
			cmLeaf(t, "a", uOccurs(t, 1, 1)),
			cmLeaf(t, "b", uOccurs(t, 1, 1))))

	cmWantDeclaration(t, cmNext(t, m, "a"), "a")
	cmWantWildcard(t, cmNext(t, m, "x"), ProcessSkip)
	cmWantDeclaration(t, cmNext(t, m, "b"), "b")
	if !m.Accepting() {
		t.Error("Accepting() = false after the {particle} took the whole of S1")
	}
}

// The {open content} is the LAST tier, under both kinds of ·basic particle·:
// clause 3.3 licenses it only where the {particle} has no ·path· for the item,
// so a name an ·element particle· admits goes to the declaration and a name
// only a ·wildcard particle· admits goes to that particle's wildcard
// (PRINCIPLES 14).
func TestMatcherPrefersTheParticleToTheOpenContentWildcard(t *testing.T) {
	m := cmOpenMatcher(t, OpenContentInterleave, cmOpenWildcard(t),
		cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
			cmLeaf(t, "a", uOccurs(t, 1, 1)),
			uParticle(t, uOccurs(t, 1, 1), ResolvedTerm{
				Term: uWildcard(t, NamespaceConstraintNot, []Namespace{NamespaceName(uns)}, ProcessLax)})))

	cmWantDeclaration(t, cmNext(t, m, "a"), "a")
	got, ok := m.Next(QName{Space: "urn:other", Local: "wp"})
	if !ok {
		t.Fatal("Next rejected a name the ·wildcard particle· admits")
	}
	cmWantWildcard(t, got, ProcessLax)
	cmWantWildcard(t, cmNext(t, m, "x"), ProcessSkip)
}

// An item the {particle} cannot take and {open content}.{wildcard} does not
// admit (cvc-wildcard §3.10.4.1) satisfies neither clause 2 nor clause 3 and is
// rejected — and rejecting it leaves the walk where it stood, so the item the
// {particle} was waiting for is still taken.
func TestMatcherRejectsANameNeitherTheParticleNorTheOpenWildcardAdmits(t *testing.T) {
	m := cmOpenMatcher(t, OpenContentInterleave,
		uWildcard(t, NamespaceConstraintNot, []Namespace{NamespaceName(uns)}, ProcessSkip),
		cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
			cmLeaf(t, "a", uOccurs(t, 1, 1)),
			cmLeaf(t, "b", uOccurs(t, 1, 1))))

	cmWantDeclaration(t, cmNext(t, m, "a"), "a")
	if _, ok := m.Next(uq("zzz")); ok {
		t.Fatal("Next took a name neither the {particle} nor the {open content} wildcard admits")
	}
	cmWantDeclaration(t, cmNext(t, m, "b"), "b")
	if !m.Accepting() {
		t.Error("Accepting() = false after a rejected name that should have changed nothing")
	}
}

// cmNested is (a{1,2}, b?){2,2}, cvc-accept's own named non-determinism and the
// file comment's worked counter-example: L(a{1,2}, b?) is {(a), (a b), (a a),
// (a a b)} and the outer particle's fixed n = 2 makes L(P) the concatenation of
// two of them.
func cmNested(t *testing.T) Particle {
	t.Helper()
	return cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
		cmGroup(t, uOccurs(t, 2, 2), CompositorSequence,
			cmLeaf(t, "a", uOccurs(t, 1, 2)),
			cmLeaf(t, "b", uOccurs(t, 0, 1))))
}

// cvc-accept clause 3.1 asks whether SOME partition of the sequence into
// iterations is valid, and the greedy one is not always it: (a{1,2}, b?){2,2}
// admits "a a b" as (a)(a b), while filling the open iteration first consumes
// the whole of it as one iteration and leaves the second owing a mandatory a.
// "a a a b" is (a a)(a b), where the greedy partition is the one that works, so
// the two run side by side.
func TestMatcherAcceptsAPartitionTheGreedyWalkMisses(t *testing.T) {
	cmAccept(t, cmMatcher(t, cmNested(t), nil), "a", "a", "b")
	cmAccept(t, cmMatcher(t, cmNested(t), nil), "a", "a", "a", "b")
}

// No partition is not every partition. "a a b b" needs an iteration ending in
// two b's and "a a a a a" a fifth a with both iterations full, and neither is a
// word of the body's language, so widening the walk over partitions accepts
// nothing L(P) does not.
func TestMatcherRejectsASequenceNoPartitionAdmits(t *testing.T) {
	over := cmMatcher(t, cmNested(t), nil)
	if i := cmFeed(t, over, "a", "a", "b", "b"); i != 3 {
		t.Errorf("the second b was rejected at position %d, want position 3", i)
	}

	full := cmMatcher(t, cmNested(t), nil)
	if i := cmFeed(t, full, "a", "a", "a", "a", "a"); i != 4 {
		t.Errorf("the fifth a was rejected at position %d, want position 4", i)
	}

	short := cmMatcher(t, cmNested(t), nil)
	if i := cmFeed(t, short, "a"); i != 1 {
		t.Fatalf("Next rejected a at position %d", i)
	}
	if short.Accepting() {
		t.Error("Accepting() = true with the second iteration's mandatory a still owed")
	}
}

// An iteration a partition closes has to be a whole word of the group's
// language: (a{1,2}, b){2,2} cannot draw a boundary after the second a, b being
// mandatory, so "a a b" is outside L(P) even though both the greedy partition
// and the one that starts a second iteration at that a can consume every item
// of it. Widening the walk over partitions must not manufacture that boundary.
func TestMatcherClosesAnIterationOnlyOnAWholeWordOfTheBody(t *testing.T) {
	model := func() Particle {
		return cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
			cmGroup(t, uOccurs(t, 2, 2), CompositorSequence,
				cmLeaf(t, "a", uOccurs(t, 1, 2)),
				cmLeaf(t, "b", uOccurs(t, 1, 1))))
	}
	cmAccept(t, cmMatcher(t, model(), nil), "a", "b", "a", "b")
	cmAccept(t, cmMatcher(t, model(), nil), "a", "a", "b", "a", "b")

	short := cmMatcher(t, model(), nil)
	if i := cmFeed(t, short, "a", "a", "b"); i != 3 {
		t.Fatalf("Next rejected position %d of a a b, want all three taken", i)
	}
	if short.Accepting() {
		t.Error("Accepting() = true for a a b, which no partition into two iterations of (a{1,2}, b) covers")
	}
}

// The FIRST viable partition is not the greedy one here, and no later item
// rescues it: (a{2,3}){2,2} takes "a a a a" only as (a a)(a a), while a walk
// that fills the open iteration draws (a a a)(a) and leaves the second
// iteration one occurrence short of a's {min occurs}. One a fewer and one a
// more are outside L(P) either way.
func TestMatcherSplitsIterationsAgainstTheGreedyBoundary(t *testing.T) {
	model := func() Particle {
		return cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
			cmGroup(t, uOccurs(t, 2, 2), CompositorSequence,
				cmLeaf(t, "a", uOccurs(t, 2, 3))))
	}
	cmAccept(t, cmMatcher(t, model(), nil), "a", "a", "a", "a")
	cmAccept(t, cmMatcher(t, model(), nil), "a", "a", "a", "a", "a")
	cmAccept(t, cmMatcher(t, model(), nil), "a", "a", "a", "a", "a", "a")

	short := cmMatcher(t, model(), nil)
	if i := cmFeed(t, short, "a", "a", "a"); i != 3 {
		t.Fatalf("Next rejected position %d of three a's, want all three taken", i)
	}
	if short.Accepting() {
		t.Error("Accepting() = true for three a's, which no partition into two iterations of a{2,3} covers")
	}

	over := cmMatcher(t, model(), nil)
	if i := cmFeed(t, over, "a", "a", "a", "a", "a", "a", "a"); i != 6 {
		t.Errorf("the seventh a was rejected at position %d, want position 6 (2 × 3)", i)
	}
}

// Widening the walk over partitions is not a search over the items already
// taken: the cursor set is CLAMPED and collapsed, so it stays inside
// maxPartitionStates however long the instance is. This model's partitions of n
// a's into iterations of one to three number tribonacci(n) — about 1.8^n, which
// is 10^5000 here — and a walk holding one live state per partition boundary it
// had drawn would not return.
func TestMatcherBoundsTheCursorSetOverANestedRepetition(t *testing.T) {
	const items = 20000
	m := cmMatcher(t, cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
		cmGroup(t, uUnbounded(t, 1), CompositorSequence,
			cmLeaf(t, "a", uOccurs(t, 1, 3)))), nil)

	for i := 0; i < items; i++ {
		if _, ok := m.Next(uq("a")); !ok {
			t.Fatalf("Next(a) rejected occurrence %d of %d", i+1, items)
		}
		if len(m.live) > maxPartitionStates {
			t.Fatalf("after %d items the walk carries %d cursors, want at most %d", i+1, len(m.live), maxPartitionStates)
		}
	}
	if !m.Accepting() {
		t.Error("Accepting() = false after a whole number of iterations")
	}
}

// The cursor set is bounded at CONSTRUCTION, not pruned mid-sequence: a nested
// repetition whose occurrence ranges could put more partitions in flight than
// maxPartitionStates is declined outright, so a Matcher that exists still
// decides every name put to it.
func TestContentMatcherDeclinesANestedRepetitionTooWideToCarry(t *testing.T) {
	p := cmGroup(t, uOccurs(t, 1, 1), CompositorSequence,
		cmGroup(t, uOccurs(t, 1, 500), CompositorSequence,
			cmLeaf(t, "a", uOccurs(t, 1, 500))))
	s, ct := cmSchema(t, p, nil)
	if _, ok := s.ContentMatcher(ct); ok {
		t.Error("ContentMatcher decided a model whose partitions outnumber maxPartitionStates")
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
