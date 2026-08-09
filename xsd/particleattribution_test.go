package xsd

import (
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests are package-internal: checkContentModelsUnambiguous runs inside
// SchemaBuilder.Finalize and is unexported (STYLE T5), so the assertions are made
// on the error Finalize returns.

const uns = "urn:upa"

func uq(local string) QName { return QName{Space: uns, Local: local} }

func uOccurs(t *testing.T, minOccurs, maxOccurs int) Occurs {
	t.Helper()
	o, err := NewOccurs(xsderr.Loc{}, minOccurs, maxOccurs)
	if err != nil {
		t.Fatalf("NewOccurs(%d,%d): %v", minOccurs, maxOccurs, err)
	}
	return o
}

func uUnbounded(t *testing.T, minOccurs int) Occurs {
	t.Helper()
	o, err := NewUnboundedOccurs(xsderr.Loc{}, minOccurs)
	if err != nil {
		t.Fatalf("NewUnboundedOccurs(%d): %v", minOccurs, err)
	}
	return o
}

func uParticle(t *testing.T, o Occurs, term TermOrRef) Particle {
	t.Helper()
	p, err := NewParticle(xsderr.Loc{}, o, term, nil)
	if err != nil {
		t.Fatalf("NewParticle: %v", err)
	}
	return p
}

// uOne wraps a term in a particle with {min occurs} = {max occurs} = 1.
func uOne(t *testing.T, term TermOrRef) Particle {
	t.Helper()
	return uParticle(t, uOccurs(t, 1, 1), term)
}

func uGroup(t *testing.T, compositor Compositor, particles ...Particle) ModelGroup {
	t.Helper()
	g, err := NewModelGroup(xsderr.Loc{}, compositor, particles, nil)
	if err != nil {
		t.Fatalf("NewModelGroup: %v", err)
	}
	return g
}

// uLocalScope is a local {scope} whose {parent} names an arbitrary containing
// complex type. It is shared by every package-internal test that needs a
// non-global element declaration; those tests read only {scope}.{variety}, never
// which container the declaration is scoped to.
func uLocalScope(t *testing.T) Scope {
	t.Helper()
	s, err := NewLocalScope(xsderr.Loc{}, ComplexTypeScopeParent{Name: uq("container")})
	if err != nil {
		t.Fatalf("NewLocalScope: %v", err)
	}
	return s
}

// uLocal builds a LOCAL element declaration with a named (non-anonymous) {type
// definition}, so that a model group holding two same-named ones exercises
// cos-nonambig without also tripping cos-element-consistent.
func uLocal(t *testing.T, name QName, typeName QName) ElementDeclaration {
	t.Helper()
	e, err := NewElementDeclaration(xsderr.Loc{}, name, TypeDefinitionRef{Name: typeName}, nil, uLocalScope(t), nil, false, nil,
		nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	return e
}

// uGlobal builds a TOP-LEVEL element declaration with a named {type definition}
// and the given {substitution group affiliations}.
func uGlobal(t *testing.T, name QName, typeName QName, affiliations ...QName) ElementDeclaration {
	t.Helper()
	e, err := NewElementDeclaration(xsderr.Loc{}, name, TypeDefinitionRef{Name: typeName}, nil, NewGlobalScope(), nil, false, nil,
		affiliations, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	return e
}

// uUntyped builds a global declaration with an ABSENT {type definition},
// affiliated to head: the intermediate link that lets a member reach a head
// whose type blocks it without tripping e-props-correct clause 4, which skips an
// affiliation edge with no type definition on one end (substitutiongroup_test.go's
// sgLink carries the full argument).
func uUntyped(t *testing.T, name QName, head QName) ElementDeclaration {
	t.Helper()
	e, err := NewElementDeclaration(xsderr.Loc{}, name, nil, nil, NewGlobalScope(), nil, false, nil,
		[]QName{head}, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	return e
}

func uWildcard(t *testing.T, variety NamespaceConstraintVariety, namespaces []Namespace, pc ProcessContents) Wildcard {
	t.Helper()
	nc, err := NewNamespaceConstraint(xsderr.Loc{}, variety, namespaces, nil, nil)
	if err != nil {
		t.Fatalf("NewNamespaceConstraint: %v", err)
	}
	w, err := NewWildcard(xsderr.Loc{}, nc, pc, nil)
	if err != nil {
		t.Fatalf("NewWildcard: %v", err)
	}
	return w
}

// uCT builds an element-only complex type whose {content type} is the particle p.
func uCT(t *testing.T, name QName, p Particle) ComplexType {
	t.Helper()
	ct, err := NewComplexType(xsderr.Loc{}, name, QName{}, nil, DerivationRestriction, false,
		nil, nil, nil, ElementContent{Particle: p}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType: %v", err)
	}
	return ct
}

// uNamedType builds a trivial empty-content complex type to serve as a named
// {type definition} target.
func uNamedType(t *testing.T, name QName) ComplexType {
	t.Helper()
	ct, err := NewComplexType(xsderr.Loc{}, name, QName{}, nil, DerivationRestriction, false,
		nil, nil, nil, EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType: %v", err)
	}
	return ct
}

// uSchemaWithModel finalizes a schema whose only content model is the given
// model group, plus the named types the declarations in it refer to.
func uSchemaWithModel(t *testing.T, g ModelGroup, extra func(*SchemaBuilder)) error {
	t.Helper()
	b := NewSchemaBuilder()
	b.AddType(uNamedType(t, uq("T")))
	b.AddType(uNamedType(t, uq("U")))
	if extra != nil {
		extra(b)
	}
	b.AddType(uCT(t, uq("ct"), uOne(t, ResolvedTerm{Term: g})))
	_, err := b.Finalize()
	return err
}

// expectRule asserts that err is an *xsderr.Error carrying rule.
func expectRule(t *testing.T, err error, rule xsderr.Rule) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected a %s rejection, got nil", rule)
	}
	var xe *xsderr.Error
	if !errors.As(err, &xe) {
		t.Fatalf("expected an *xsderr.Error carrying %s, got %T: %v", rule, err, err)
	}
	if xe.Rule != rule {
		t.Fatalf("expected rule %s, got %s: %v", rule, xe.Rule, err)
	}
}

// TestUPAChoiceCompetition pins the simplest cos-nonambig violation: two
// same-named element particles in one <choice>, both live in the start state.
func TestUPAChoiceCompetition(t *testing.T) {
	g := uGroup(t, CompositorChoice,
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
	)
	expectRule(t, uSchemaWithModel(t, g, nil), ruleCosNonambig)
}

// TestUPADisjointNamesPass pins the pass case: a choice over differently-named
// element particles is deterministic.
func TestUPADisjointNamesPass(t *testing.T) {
	g := uGroup(t, CompositorChoice,
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("b"), uq("T"))}),
	)
	if err := uSchemaWithModel(t, g, nil); err != nil {
		t.Fatalf("a choice over distinct names was rejected: %v", err)
	}
}

// TestUPANestedSequenceCompetition pins the case Appendix J's two-bullet
// shortcut misses: (a, b) | (a, c). The choice's {particles} are two SEQUENCE
// particles, not the two a's, so only the position automaton finds it.
func TestUPANestedSequenceCompetition(t *testing.T) {
	g := uGroup(t, CompositorChoice,
		uOne(t, ResolvedTerm{Term: uGroup(t, CompositorSequence,
			uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
			uOne(t, ResolvedTerm{Term: uLocal(t, uq("b"), uq("T"))}),
		)}),
		uOne(t, ResolvedTerm{Term: uGroup(t, CompositorSequence,
			uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
			uOne(t, ResolvedTerm{Term: uLocal(t, uq("c"), uq("T"))}),
		)}),
	)
	expectRule(t, uSchemaWithModel(t, g, nil), ruleCosNonambig)
}

// TestUPAOptionalThenSameName pins the adjacency case: a? then a puts both in the
// start state.
func TestUPAOptionalThenSameName(t *testing.T) {
	g := uGroup(t, CompositorSequence,
		uParticle(t, uOccurs(t, 0, 1), ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
	)
	expectRule(t, uSchemaWithModel(t, g, nil), ruleCosNonambig)
}

// TestUPAFiniteRangeUnfoldingPasses is warden's correction (ii): a FINITE numeric
// occurrence range must be unfolded, not loop-encoded.
//
//	<sequence><element name="a" minOccurs="2" maxOccurs="2"/><element name="a"/></sequence>
//
// has no two ·paths· differing only in their last item — the first three items are
// attributed to the first particle twice and then the second once — so it must
// PASS. A loop-encoded first particle would put both particles in one state and
// false-reject it.
func TestUPAFiniteRangeUnfoldingPasses(t *testing.T) {
	g := uGroup(t, CompositorSequence,
		uParticle(t, uOccurs(t, 2, 2), ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
	)
	if err := uSchemaWithModel(t, g, nil); err != nil {
		t.Fatalf("a{2,2} followed by a was rejected: %v", err)
	}
}

// TestUPAFiniteRangeWithSlackViolates is the boundary the previous test sits on:
// once the first particle's {max occurs} exceeds its {min occurs}, a sequence of
// three a's can be split two ways, so the two particles do compete.
func TestUPAFiniteRangeWithSlackViolates(t *testing.T) {
	g := uGroup(t, CompositorSequence,
		uParticle(t, uOccurs(t, 2, 3), ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
	)
	expectRule(t, uSchemaWithModel(t, g, nil), ruleCosNonambig)
}

// TestUPARepeatedSequenceUnfoldingPasses pins that unfolded copies share particle
// identity at every depth, not just at the leaf: (a, b){0,2} is deterministic,
// and the two copies' a positions must not look like two competing particles.
func TestUPARepeatedSequenceUnfoldingPasses(t *testing.T) {
	g := uGroup(t, CompositorSequence,
		uParticle(t, uOccurs(t, 0, 2), ResolvedTerm{Term: uGroup(t, CompositorSequence,
			uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
			uOne(t, ResolvedTerm{Term: uLocal(t, uq("b"), uq("T"))}),
		)}),
	)
	if err := uSchemaWithModel(t, g, nil); err != nil {
		t.Fatalf("(a, b){0,2} was rejected: %v", err)
	}
}

// TestUPAUnboundedThenSameName pins the loop-back edge: a* then a competes.
func TestUPAUnboundedThenSameName(t *testing.T) {
	g := uGroup(t, CompositorSequence,
		uParticle(t, uUnbounded(t, 0), ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
	)
	expectRule(t, uSchemaWithModel(t, g, nil), ruleCosNonambig)
}

// TestUPAElementAndWildcardPass is the sole 1.0→1.1 relaxation (Appendix G.1.3):
// an ·element particle· competing with a ·wildcard particle· is NOT prohibited —
// the Element Declaration wins ·attribution·. This is the case a kind-blind
// overlap test would false-reject.
func TestUPAElementAndWildcardPass(t *testing.T) {
	g := uGroup(t, CompositorChoice,
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uWildcard(t, NamespaceConstraintAny, nil, ProcessStrict)}),
	)
	if err := uSchemaWithModel(t, g, nil); err != nil {
		t.Fatalf("an element particle competing with a wildcard particle was rejected: %v", err)
	}
}

// TestUPAWildcardsCompete pins the wildcard/wildcard half, decided through
// cos-aw-intersect: two ##any wildcards intersect to ##any, which is non-empty.
func TestUPAWildcardsCompete(t *testing.T) {
	g := uGroup(t, CompositorChoice,
		uOne(t, ResolvedTerm{Term: uWildcard(t, NamespaceConstraintAny, nil, ProcessStrict)}),
		uOne(t, ResolvedTerm{Term: uWildcard(t, NamespaceConstraintAny, nil, ProcessLax)}),
	)
	expectRule(t, uSchemaWithModel(t, g, nil), ruleCosNonambig)
}

// TestUPADisjointWildcardsPass pins the other side of cos-aw-intersect: two
// enumeration wildcards over disjoint namespace sets intersect to an enumeration
// with an EMPTY {namespaces}, which Appendix J reads as no overlap.
func TestUPADisjointWildcardsPass(t *testing.T) {
	g := uGroup(t, CompositorChoice,
		uOne(t, ResolvedTerm{Term: uWildcard(t, NamespaceConstraintEnumeration, []Namespace{NamespaceName("urn:x")}, ProcessStrict)}),
		uOne(t, ResolvedTerm{Term: uWildcard(t, NamespaceConstraintEnumeration, []Namespace{NamespaceName("urn:y")}, ProcessStrict)}),
	)
	if err := uSchemaWithModel(t, g, nil); err != nil {
		t.Fatalf("two wildcards over disjoint namespaces were rejected: %v", err)
	}
}

// TestUPASubstitutionGroupOverlap pins Appendix J bullet 2: two differently-named
// element particles overlap when one's name is that of a declaration in the
// other's ·substitution group·.
func TestUPASubstitutionGroupOverlap(t *testing.T) {
	g := uGroup(t, CompositorChoice,
		uOne(t, ElementDeclarationRef{Name: uq("head")}),
		uOne(t, ElementDeclarationRef{Name: uq("member")}),
	)
	err := uSchemaWithModel(t, g, func(b *SchemaBuilder) {
		b.AddElement(uGlobal(t, uq("head"), uq("T")))
		b.AddElement(uGlobal(t, uq("member"), uq("T"), uq("head")))
	})
	expectRule(t, err, ruleCosNonambig)
}

// TestUPAUnrelatedGlobalsPass is the control for the previous test: without the
// affiliation the same two names are disjoint.
func TestUPAUnrelatedGlobalsPass(t *testing.T) {
	g := uGroup(t, CompositorChoice,
		uOne(t, ElementDeclarationRef{Name: uq("head")}),
		uOne(t, ElementDeclarationRef{Name: uq("member")}),
	)
	err := uSchemaWithModel(t, g, func(b *SchemaBuilder) {
		b.AddElement(uGlobal(t, uq("head"), uq("T")))
		b.AddElement(uGlobal(t, uq("member"), uq("T")))
	})
	if err != nil {
		t.Fatalf("two unaffiliated global declarations were rejected: %v", err)
	}
}

// TestUPAUnrelatedProhibitedSubstitutionsStillOverlap pins that
// cos-equiv-derived-ok-rec clause 2.3 is decided on the ·derivation· of the
// member's {type definition} from the head's, not on whether SOME complex type in
// the schema carries a {prohibited substitutions} member: an unrelated blocking
// type leaves head and member in one ·substitution group·, so the two particles
// still ·overlap· and cos-nonambig fires.
func TestUPAUnrelatedProhibitedSubstitutionsStillOverlap(t *testing.T) {
	blocking, err := NewComplexType(xsderr.Loc{}, uq("Blocking"), QName{}, nil, DerivationRestriction, false,
		nil, nil, nil, EmptyContent{}, []DerivationMethod{DerivationExtension}, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType: %v", err)
	}
	g := uGroup(t, CompositorChoice,
		uOne(t, ElementDeclarationRef{Name: uq("head")}),
		uOne(t, ElementDeclarationRef{Name: uq("member")}),
	)
	err = uSchemaWithModel(t, g, func(b *SchemaBuilder) {
		b.AddType(blocking)
		b.AddElement(uGlobal(t, uq("head"), uq("T")))
		b.AddElement(uGlobal(t, uq("member"), uq("T"), uq("head")))
	})
	expectRule(t, err, ruleCosNonambig)
}

// TestUPASubstitutionBlockedByClause23 is the companion: when clause 2.3 really
// does block — the head's {type definition} prohibits extension and the member's
// type reaches it by extension — the member is in no ·substitution group· of the
// head, the two element particles do not ·overlap·, and the content model stands.
//
// The member is affiliated to the head through a TYPELESS link declaration, for
// the reason substitutiongroup_test.go's sgLink records: e-props-correct clause
// 4 would reject the direct edge at Finalize, and this test is about what
// cos-nonambig does with a schema that stands.
func TestUPASubstitutionBlockedByClause23(t *testing.T) {
	head, err := NewComplexType(xsderr.Loc{}, uq("Head"), QName{}, nil, DerivationRestriction, false,
		nil, nil, nil, EmptyContent{}, []DerivationMethod{DerivationExtension}, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType: %v", err)
	}
	derived, err := NewComplexType(xsderr.Loc{}, uq("Derived"), uq("Head"), nil, DerivationExtension, false,
		nil, nil, nil, EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType: %v", err)
	}
	g := uGroup(t, CompositorChoice,
		uOne(t, ElementDeclarationRef{Name: uq("head")}),
		uOne(t, ElementDeclarationRef{Name: uq("member")}),
	)
	err = uSchemaWithModel(t, g, func(b *SchemaBuilder) {
		b.AddType(head)
		b.AddType(derived)
		b.AddElement(uGlobal(t, uq("head"), uq("Head")))
		b.AddElement(uUntyped(t, uq("link"), uq("head")))
		b.AddElement(uGlobal(t, uq("member"), uq("Derived"), uq("link")))
	})
	if err != nil {
		t.Fatalf("a clause 2.3 non-member manufactured an ·overlap·: %v", err)
	}
}

// TestUPAAllGroupCompetition pins the <all> treatment: every member is live at the
// start, so two same-named members compete.
func TestUPAAllGroupCompetition(t *testing.T) {
	g := uGroup(t, CompositorAll,
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
	)
	expectRule(t, uSchemaWithModel(t, g, nil), ruleCosNonambig)
}

// TestUPAAllGroupDistinctNamesPass pins that the (P1|…|Pn)* transcription does not
// invent competition between an <all> member and itself: a well-formed <all> over
// distinct names passes even though the loop-back edge lands on every member.
func TestUPAAllGroupDistinctNamesPass(t *testing.T) {
	g := uGroup(t, CompositorAll,
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("b"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("c"), uq("T"))}),
	)
	if err := uSchemaWithModel(t, g, nil); err != nil {
		t.Fatalf("an <all> group over distinct names was rejected: %v", err)
	}
}

// TestUPAAllGroupNonEmptiableInSequencePasses pins addAll's emptiability
// carve-out. An <all> with a mandatory member does not accept the empty
// sequence, so in an enclosing <sequence> the particle after it never joins a
// state the particle before it still occupies. Taking emptiability from the
// (P1|…|Pn)* transcription instead — where it is unconditionally true — carries
// the leading x? past the <all>, merges the trailing x into the start state
// beside it, and false-rejects this model.
func TestUPAAllGroupNonEmptiableInSequencePasses(t *testing.T) {
	g := uGroup(t, CompositorSequence,
		uParticle(t, uOccurs(t, 0, 1), ResolvedTerm{Term: uLocal(t, uq("x"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uGroup(t, CompositorAll,
			uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
			uOne(t, ResolvedTerm{Term: uLocal(t, uq("b"), uq("T"))}),
		)}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("x"), uq("T"))}),
	)
	if err := uSchemaWithModel(t, g, nil); err != nil {
		t.Fatalf("x? then a non-emptiable <all> then x was rejected: %v", err)
	}
}

// uAllThen builds sequence(all(members…), successor), the shape cos-all-limited
// clause 1 (§3.8.6.2) keeps out of a schema document and the exported xsd
// constructors reach directly.
func uAllThen(t *testing.T, successor Particle, members ...Particle) ModelGroup {
	t.Helper()
	return uGroup(t, CompositorSequence,
		uOne(t, ResolvedTerm{Term: uGroup(t, CompositorAll, members...)}),
		successor,
	)
}

// TestUPAAllGroupMandatoryMemberDoesNotCompeteWithSuccessor pins the false reject
// #468 removes: in sequence(all(a, b), a) the two a particles do not ·compete·.
// ·compete· (§3.8.4.2) asks for two ·paths· "identical except that one path has
// P1 as its last item and the other has P2", so the prefix PATH is shared: either
// it already attributed an a to the <all>'s member, in which case that member's
// {max occurs} is spent and it cannot be the last item of the other path, or it
// did not, in which case §3.8.4.1.3's S = S1 × S2 leaves the group incomplete and
// the successor is not reachable at all. Transcribing the <all> as a bare
// (a | b)* claims the group may end after EITHER member and false-rejects this.
func TestUPAAllGroupMandatoryMemberDoesNotCompeteWithSuccessor(t *testing.T) {
	g := uAllThen(t, uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("b"), uq("T"))}),
	)
	if err := uSchemaWithModel(t, g, nil); err != nil {
		t.Fatalf("sequence(all(a, b), a) was rejected: %v", err)
	}
}

// TestUPAAllGroupMandatoryMemberAfterOptionalPredecessor is the same model with a
// leading optional particle, which puts the <all>'s ·first· set into the start
// state beside z and so exercises the carve-out and the successor wiring at once.
func TestUPAAllGroupMandatoryMemberAfterOptionalPredecessor(t *testing.T) {
	g := uGroup(t, CompositorSequence,
		uParticle(t, uOccurs(t, 0, 1), ResolvedTerm{Term: uLocal(t, uq("z"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uGroup(t, CompositorAll,
			uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
			uOne(t, ResolvedTerm{Term: uLocal(t, uq("b"), uq("T"))}),
		)}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
	)
	if err := uSchemaWithModel(t, g, nil); err != nil {
		t.Fatalf("sequence(z?, all(a, b), a) was rejected: %v", err)
	}
}

// TestUPAAllGroupThreeMandatoryMembersGeneralize pins that the fix is about the
// SUBSET of members still owed and not about the two-member case: with three
// mandatory members there are three orders, each ending on a different member, and
// a rule that gated one member's ·last· positions on every other member being
// ·emptiable· would return an empty exit set for all three.
func TestUPAAllGroupThreeMandatoryMembersGeneralize(t *testing.T) {
	g := uAllThen(t, uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("b"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("c"), uq("T"))}),
	)
	if err := uSchemaWithModel(t, g, nil); err != nil {
		t.Fatalf("sequence(all(a, b, c), a) was rejected: %v", err)
	}
}

// TestUPAAllGroupEmptiableMemberCompetesWithSuccessor is the negative direction,
// and it is what keeps the fix from being indistinguishable from deleting the
// check. In sequence(all(a, b?), b) the member b IS ·emptiable·, so the path
// (all's a) leaves the group complete with b never taken: the sequence <a/><b/>
// has one path ending at the <all>'s b and another, identical up to its last
// item, ending at the successor b. The two ·compete· and cos-nonambig fires.
func TestUPAAllGroupEmptiableMemberCompetesWithSuccessor(t *testing.T) {
	g := uAllThen(t, uOne(t, ResolvedTerm{Term: uLocal(t, uq("b"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uParticle(t, uOccurs(t, 0, 1), ResolvedTerm{Term: uLocal(t, uq("b"), uq("T"))}),
	)
	expectRule(t, uSchemaWithModel(t, g, nil), ruleCosNonambig)
}

// TestUPAAllGroupSatisfiedRepeatableMemberCompetesWithSuccessor is the other
// residual: a member with {max occurs} past {min occurs} is satisfied after its
// first occurrence, so it may still be taken once the group is complete. The
// sequence <b/><a/><a/> has a path whose last item is the <all>'s a taken a
// second time and another whose last item is the successor a, so cos-nonambig
// fires here too — the exit state is not empty just because every member is
// non-·emptiable·.
func TestUPAAllGroupSatisfiedRepeatableMemberCompetesWithSuccessor(t *testing.T) {
	g := uAllThen(t, uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uParticle(t, uOccurs(t, 1, 2), ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("b"), uq("T"))}),
	)
	expectRule(t, uSchemaWithModel(t, g, nil), ruleCosNonambig)
}

// TestUPAAllGroupNonEmptiableMemberBesideEmptiableOnePasses is the control for
// the two negative pins above: in sequence(all(a, b?), a) the successor collides
// with the NON-·emptiable· member, so the shared prefix path argument of
// TestUPAAllGroupMandatoryMemberDoesNotCompeteWithSuccessor applies unchanged and
// b's emptiability is beside the point — b never ·overlaps· a. A successor is
// live in the exit state without thereby competing with anything.
func TestUPAAllGroupNonEmptiableMemberBesideEmptiableOnePasses(t *testing.T) {
	g := uAllThen(t, uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uParticle(t, uOccurs(t, 0, 1), ResolvedTerm{Term: uLocal(t, uq("b"), uq("T"))}),
	)
	if err := uSchemaWithModel(t, g, nil); err != nil {
		t.Fatalf("sequence(all(a, b?), a) was rejected: %v", err)
	}
}

// TestUPAGroupRefExpansion pins that a <group ref> is expanded, and that two
// references to one definition are two DISTINCT particles (§3.8.6.4: "particles
// at different points in the content model are always distinct from one another,
// even if they originated from the same named model group") — so a choice over
// two references to the same group competes.
func TestUPAGroupRefExpansion(t *testing.T) {
	inner := uGroup(t, CompositorSequence,
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
	)
	mgd, err := NewModelGroupDefinition(xsderr.Loc{}, uq("g"), inner, nil)
	if err != nil {
		t.Fatalf("NewModelGroupDefinition: %v", err)
	}
	g := uGroup(t, CompositorChoice,
		uOne(t, ModelGroupRef{Name: uq("g")}),
		uOne(t, ModelGroupRef{Name: uq("g")}),
	)
	err = uSchemaWithModel(t, g, func(b *SchemaBuilder) { b.AddModelGroup(mgd) })
	expectRule(t, err, ruleCosNonambig)
}

// TestUPAUnreferencedModelGroupDefinition pins the follow-up grounding's reading:
// §3.8.6's chapeau binds ALL model groups, and a Model Group Definition's {model
// group} is a Model Group component the moment <group name="…"> is processed, so
// an ambiguous group definition is rejected even when no <group ref> points at it.
func TestUPAUnreferencedModelGroupDefinition(t *testing.T) {
	inner := uGroup(t, CompositorChoice,
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
	)
	mgd, err := NewModelGroupDefinition(xsderr.Loc{}, uq("orphan"), inner, nil)
	if err != nil {
		t.Fatalf("NewModelGroupDefinition: %v", err)
	}
	b := NewSchemaBuilder()
	b.AddType(uNamedType(t, uq("T")))
	b.AddModelGroup(mgd)
	_, err = b.Finalize()
	expectRule(t, err, ruleCosNonambig)
}

// TestUPADeterministicFirstFailure pins STYLE D1: a model group with two
// violations always reports the same one — the lexicographically least
// (state, i, j), here the start state's first competing pair.
func TestUPADeterministicFirstFailure(t *testing.T) {
	g := uGroup(t, CompositorChoice,
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("b"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("b"), uq("T"))}),
	)
	first := uSchemaWithModel(t, g, nil)
	expectRule(t, first, ruleCosNonambig)
	for i := 0; i < 8; i++ {
		again := uSchemaWithModel(t, g, nil)
		if again.Error() != first.Error() {
			t.Fatalf("rejection is not deterministic:\n  %v\n  %v", first, again)
		}
	}
	if want := uq("a").String(); !strings.Contains(first.Error(), want) {
		t.Errorf("expected the first competing pair (%s) to be reported, got: %v", want, first)
	}
}
