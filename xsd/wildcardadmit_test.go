package xsd

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests are package-internal: allowsElementWildcardName and
// allowsAttributeWildcardName stay unexported until M5 supplies the caller
// (STYLE T5), so only an in-package test can reach them.

const wns = "urn:wildcardadmit"

func wq(local string) QName { return QName{Space: wns, Local: local} }

func wOccurs(t *testing.T) Occurs {
	t.Helper()
	o, err := NewOccurs(xsderr.Loc{}, 1, 1)
	if err != nil {
		t.Fatalf("NewOccurs: %v", err)
	}
	return o
}

func wParticle(t *testing.T, term TermOrRef) Particle {
	t.Helper()
	p, err := NewParticle(xsderr.Loc{}, wOccurs(t), term, nil)
	if err != nil {
		t.Fatalf("NewParticle: %v", err)
	}
	return p
}

func wGroup(t *testing.T, compositor Compositor, particles ...Particle) ModelGroup {
	t.Helper()
	g, err := NewModelGroup(xsderr.Loc{}, compositor, particles, nil)
	if err != nil {
		t.Fatalf("NewModelGroup: %v", err)
	}
	return g
}

// wCT builds an element-only complex type whose {content type} particle is term.
func wCT(t *testing.T, name QName, term TermOrRef) ComplexType {
	t.Helper()
	ct, err := NewComplexType(xsderr.Loc{}, name, QName{}, nil, DerivationRestriction, false,
		nil, nil, nil, ElementContent{Particle: wParticle(t, term)}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType: %v", err)
	}
	return ct
}

// wElement builds an element declaration; affiliations forces global scope
// (e-props-correct clause 3).
func wElement(t *testing.T, name QName, scope Scope, affiliations []QName, disallowedSubstitutions []DerivationMethod) ElementDeclaration {
	t.Helper()
	e, err := NewElementDeclaration(xsderr.Loc{}, name, nil, nil, scope, nil, false, nil,
		affiliations, nil, false, disallowedSubstitutions, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	return e
}

// wWildcard builds an ##any wildcard carrying the given {disallowed names}
// keywords.
func wWildcard(t *testing.T, keywords ...DisallowedNameKeyword) Wildcard {
	t.Helper()
	nc, err := NewNamespaceConstraint(xsderr.Loc{}, NamespaceConstraintAny, nil, nil, keywords)
	if err != nil {
		t.Fatalf("NewNamespaceConstraint: %v", err)
	}
	w, err := NewWildcard(xsderr.Loc{}, nc, ProcessStrict, nil)
	if err != nil {
		t.Fatalf("NewWildcard: %v", err)
	}
	return w
}

func wFinalize(t *testing.T, b *SchemaBuilder) *Schema {
	t.Helper()
	s, err := b.Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	return s
}

// TestAllowsElementWildcardNameDefined pins cvc-wildcard (§3.10.4.1) clause 2.1:
// the name must not ·resolve· to a TOP-LEVEL element declaration.
func TestAllowsElementWildcardNameDefined(t *testing.T) {
	w := wWildcard(t, DisallowedNameDefined)
	plain := wWildcard(t)
	local := wElement(t, wq("localOnly"), uLocalScope(t), nil, nil)
	ct := wCT(t, wq("ct"), ResolvedTerm{Term: local})

	b := NewSchemaBuilder()
	b.AddElement(wElement(t, wq("top"), NewGlobalScope(), nil, nil))
	b.AddAttribute(mustAttributeDecl(t, wq("attr")))
	b.AddType(ct)
	s := wFinalize(t, b)

	if s.allowsElementWildcardName(w, ct, wq("top")) {
		t.Error("defined admitted a name resolving to a top-level element declaration (cvc-wildcard clause 2.1)")
	}
	if !s.allowsElementWildcardName(w, ct, wq("undeclared")) {
		t.Error("defined rejected a name resolving to no element declaration")
	}
	// Scope: {element declarations} is top-level only (§3.17.1). A name declared
	// ONLY as a local declaration inside the content model is still admitted —
	// this fails if the resolution ever indexes local declarations.
	if !s.allowsElementWildcardName(w, ct, wq("localOnly")) {
		t.Error("defined rejected a name declared only locally (clause 2.1 resolves against top-level declarations)")
	}
	// The keyword, not the wildcard shape, drives the rejection.
	if !s.allowsElementWildcardName(plain, ct, wq("top")) {
		t.Error("a keyword-free wildcard rejected a top-level name")
	}
	// Clause 2.1 is the ELEMENT table: an attribute's name is not consulted.
	if !s.allowsElementWildcardName(w, ct, wq("attr")) {
		t.Error("defined on an element wildcard consulted {attribute declarations} (clauses 2.1 and 2.2 conflated)")
	}
}

// TestAllowsAttributeWildcardNameDefined pins cvc-wildcard clause 2.2.
func TestAllowsAttributeWildcardNameDefined(t *testing.T) {
	w := wWildcard(t, DisallowedNameDefined)
	plain := wWildcard(t)

	b := NewSchemaBuilder()
	b.AddElement(wElement(t, wq("top"), NewGlobalScope(), nil, nil))
	b.AddAttribute(mustAttributeDecl(t, wq("attr")))
	s := wFinalize(t, b)

	if s.allowsAttributeWildcardName(w, wq("attr")) {
		t.Error("defined admitted a name resolving to a top-level attribute declaration (cvc-wildcard clause 2.2)")
	}
	if !s.allowsAttributeWildcardName(w, wq("undeclared")) {
		t.Error("defined rejected a name resolving to no attribute declaration")
	}
	// Clause 2.2 is the ATTRIBUTE table: an element's name is not consulted.
	if !s.allowsAttributeWildcardName(w, wq("top")) {
		t.Error("defined on an attribute wildcard consulted {element declarations} (clauses 2.1 and 2.2 conflated)")
	}
	if !s.allowsAttributeWildcardName(plain, wq("attr")) {
		t.Error("a keyword-free attribute wildcard rejected a top-level attribute name")
	}
}

// TestAllowsElementWildcardNameClause1 pins that clause 1 still governs: a name
// the {namespace constraint} rejects is rejected whatever the keywords say.
func TestAllowsElementWildcardNameClause1(t *testing.T) {
	nc, err := NewNamespaceConstraint(xsderr.Loc{}, NamespaceConstraintNot,
		[]Namespace{NamespaceName(wns)}, nil, []DisallowedNameKeyword{DisallowedNameDefined, DisallowedNameSibling})
	if err != nil {
		t.Fatalf("NewNamespaceConstraint: %v", err)
	}
	w, err := NewWildcard(xsderr.Loc{}, nc, ProcessStrict, nil)
	if err != nil {
		t.Fatalf("NewWildcard: %v", err)
	}
	ct := wCT(t, wq("ct"), ResolvedTerm{Term: wElement(t, wq("a"), uLocalScope(t), nil, nil)})
	s := wFinalize(t, NewSchemaBuilder())
	if s.allowsElementWildcardName(w, ct, wq("a")) {
		t.Error("clause 1 (cvc-wildcard-name) no longer rejects a name outside the {namespace constraint}")
	}
}

// TestAllowsElementWildcardNameSibling pins cvc-wildcard clause 3 across every
// containment arm: ·directly·, ·indirectly· (nested groups and <group ref>), and
// ·implicitly· (substitution groups). Each case names the arm it would lose.
func TestAllowsElementWildcardNameSibling(t *testing.T) {
	w := wWildcard(t, DisallowedNameSibling)

	// A model group definition referenced by <group ref>, holding element "g".
	groupDef, err := NewModelGroupDefinition(xsderr.Loc{}, wq("grp"),
		wGroup(t, CompositorSequence, wParticle(t, ResolvedTerm{Term: wElement(t, wq("g"), uLocalScope(t), nil, nil)})), nil)
	if err != nil {
		t.Fatalf("NewModelGroupDefinition: %v", err)
	}

	// depth-3 nesting: sequence > choice > all > element "deep".
	deep := wGroup(t, CompositorSequence, wParticle(t, ResolvedTerm{Term: wGroup(t, CompositorChoice,
		wParticle(t, ResolvedTerm{Term: wGroup(t, CompositorAll,
			wParticle(t, ResolvedTerm{Term: wElement(t, wq("deep"), uLocalScope(t), nil, nil)}))}))}))

	// The content model: direct local "direct", the deep nest, a <group ref>, an
	// <element ref> to top-level "head", and the wildcard itself.
	model := wGroup(t, CompositorSequence,
		wParticle(t, ResolvedTerm{Term: wElement(t, wq("direct"), uLocalScope(t), nil, nil)}),
		wParticle(t, ResolvedTerm{Term: deep}),
		wParticle(t, ModelGroupRef{Name: wq("grp")}),
		wParticle(t, ElementDeclarationRef{Name: wq("head")}),
		wParticle(t, ResolvedTerm{Term: w}),
	)
	ct := wCT(t, wq("ct"), ResolvedTerm{Term: model})

	b := NewSchemaBuilder()
	b.AddModelGroup(groupDef)
	b.AddType(ct)
	b.AddElement(wElement(t, wq("head"), NewGlobalScope(), nil, nil))
	b.AddElement(wElement(t, wq("member"), NewGlobalScope(), []QName{wq("head")}, nil))
	b.AddElement(wElement(t, wq("grandchild"), NewGlobalScope(), []QName{wq("member")}, nil))
	b.AddElement(wElement(t, wq("unrelated"), NewGlobalScope(), nil, nil))
	s := wFinalize(t, b)

	cases := []struct {
		name    string
		q       QName
		admit   bool
		arm     string
		wrongNS bool
	}{
		{name: "directly-contained", q: wq("direct"), admit: false, arm: "·directly contains· (the particle's own {term})"},
		{name: "nested-groups", q: wq("deep"), admit: false, arm: "·indirectly contains· through nested sequence/choice/all"},
		{name: "group-ref", q: wq("g"), admit: false, arm: "·indirectly contains· through <group ref> (§3.7.2)"},
		{name: "element-ref", q: wq("head"), admit: false, arm: "·indirectly contains· through <element ref>"},
		{name: "substitution-member", q: wq("member"), admit: false, arm: "·implicitly contains· (substitution group, §3.3.6.4)"},
		{name: "substitution-transitive", q: wq("grandchild"), admit: false, arm: "·implicitly contains· through a transitive affiliation chain (cos-equiv-derived-ok-rec clause 2.2)"},
		{name: "absent-from-model", q: wq("unrelated"), admit: true, arm: "a name in no arm must stay admitted"},
		{name: "other-namespace", q: QName{Space: "urn:other", Local: "direct"}, admit: true, arm: "·match· is expanded-name equality, not local-name equality"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := s.allowsElementWildcardName(w, ct, c.q); got != c.admit {
				t.Errorf("allowsElementWildcardName(%s) = %v, want %v — arm: %s", c.q, got, c.admit, c.arm)
			}
		})
	}
}

// TestSiblingSubstitutionGroupBlocked is the negative twin of the
// substitution-member case: a head whose {disallowed substitutions} contains
// substitution heads no substitution group (cos-equiv-derived-ok-rec clause 2.1),
// so its affiliates are NOT implicitly contained and stay admitted.
func TestSiblingSubstitutionGroupBlocked(t *testing.T) {
	w := wWildcard(t, DisallowedNameSibling)
	ct := wCT(t, wq("ct"), ElementDeclarationRef{Name: wq("head")})

	b := NewSchemaBuilder()
	b.AddType(ct)
	b.AddElement(wElement(t, wq("head"), NewGlobalScope(), nil, []DerivationMethod{DerivationSubstitution}))
	b.AddElement(wElement(t, wq("member"), NewGlobalScope(), []QName{wq("head")}, nil))
	s := wFinalize(t, b)

	if s.allowsElementWildcardName(w, ct, wq("head")) {
		t.Error("blocking substitution changed the ·match· on the head itself")
	}
	if !s.allowsElementWildcardName(w, ct, wq("member")) {
		t.Error("a member of a head that blocks substitution was treated as implicitly contained (cos-equiv-derived-ok-rec clause 2.1)")
	}
}

// TestSiblingAbstractHeadStillContains pins that {abstract} does NOT filter
// substitution-group membership: it appears nowhere in cos-equiv-derived-ok-rec.
func TestSiblingAbstractHeadStillContains(t *testing.T) {
	w := wWildcard(t, DisallowedNameSibling)
	ct := wCT(t, wq("ct"), ElementDeclarationRef{Name: wq("head")})

	abstractHead, err := NewElementDeclaration(xsderr.Loc{}, wq("head"), nil, nil, NewGlobalScope(), nil, false, nil,
		nil, nil, true /* abstract */, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	b := NewSchemaBuilder()
	b.AddType(ct)
	b.AddElement(abstractHead)
	b.AddElement(wElement(t, wq("member"), NewGlobalScope(), []QName{wq("head")}, nil))
	s := wFinalize(t, b)

	if s.allowsElementWildcardName(w, ct, wq("member")) {
		t.Error("an abstract head's substitution group was filtered out; {abstract} plays no part in cos-equiv-derived-ok-rec")
	}
}

// TestSiblingLocalDeclarationHeadsNoGroup pins that a LOCAL declaration in the
// content model heads no substitution group (§3.3.6.4 defines one only for
// members of {element declarations}, which is top-level), even when a same-named
// top-level declaration has members.
func TestSiblingLocalDeclarationHeadsNoGroup(t *testing.T) {
	w := wWildcard(t, DisallowedNameSibling)
	ct := wCT(t, wq("ct"), ResolvedTerm{Term: wElement(t, wq("head"), uLocalScope(t), nil, nil)})

	b := NewSchemaBuilder()
	b.AddType(ct)
	b.AddElement(wElement(t, wq("head"), NewGlobalScope(), nil, nil))
	b.AddElement(wElement(t, wq("member"), NewGlobalScope(), []QName{wq("head")}, nil))
	s := wFinalize(t, b)

	if s.allowsElementWildcardName(w, ct, wq("head")) {
		t.Error("the local declaration's own expanded name is no longer ·matched·")
	}
	if !s.allowsElementWildcardName(w, ct, wq("member")) {
		t.Error("a local declaration was treated as heading the same-named top-level declaration's substitution group")
	}
}

// TestSiblingIsNotMemoizedPerWildcard is the mutation guard for the one
// optimization the grounding forbids: the sibling name set is
// per-(complex type, occurrence), so the SAME Wildcard value placed in two
// complex types must answer differently. Any cache keyed on the Wildcard fails
// this.
func TestSiblingIsNotMemoizedPerWildcard(t *testing.T) {
	w := wWildcard(t, DisallowedNameSibling)

	withFoo := wCT(t, wq("withFoo"), ResolvedTerm{Term: wGroup(t, CompositorSequence,
		wParticle(t, ResolvedTerm{Term: wElement(t, wq("foo"), uLocalScope(t), nil, nil)}),
		wParticle(t, ResolvedTerm{Term: w}))})
	withoutFoo := wCT(t, wq("withoutFoo"), ResolvedTerm{Term: wGroup(t, CompositorSequence,
		wParticle(t, ResolvedTerm{Term: wElement(t, wq("bar"), uLocalScope(t), nil, nil)}),
		wParticle(t, ResolvedTerm{Term: w}))})

	b := NewSchemaBuilder()
	b.AddType(withFoo)
	b.AddType(withoutFoo)
	s := wFinalize(t, b)

	if s.allowsElementWildcardName(w, withFoo, wq("foo")) {
		t.Error("sibling admitted foo in the complex type that contains foo")
	}
	if !s.allowsElementWildcardName(w, withoutFoo, wq("foo")) {
		t.Error("sibling rejected foo in a complex type that does NOT contain it — the resolution is memoized on the Wildcard")
	}
	// And the reverse order, so the failure is not order-dependent.
	if !s.allowsElementWildcardName(w, withoutFoo, wq("foo")) || s.allowsElementWildcardName(w, withFoo, wq("foo")) {
		t.Error("repeat queries disagree with the first pair")
	}
}

// TestSiblingIgnoresNonElementContent pins that an empty or simple {content
// type} contains no element declaration, and that a nested wildcard is not one.
func TestSiblingIgnoresNonElementContent(t *testing.T) {
	w := wWildcard(t, DisallowedNameSibling)
	empty, err := NewComplexType(xsderr.Loc{}, wq("empty"), QName{}, nil, DerivationRestriction, false,
		nil, nil, nil, EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType: %v", err)
	}
	onlyWildcard := wCT(t, wq("onlyWildcard"), ResolvedTerm{Term: w})

	b := NewSchemaBuilder()
	b.AddType(empty)
	b.AddType(onlyWildcard)
	s := wFinalize(t, b)

	if !s.allowsElementWildcardName(w, empty, wq("anything")) {
		t.Error("sibling found an element declaration in an empty {content type}")
	}
	if !s.allowsElementWildcardName(w, onlyWildcard, wq("anything")) {
		t.Error("sibling treated a contained Wildcard as an element declaration")
	}
}

// mustAttributeDecl builds a top-level attribute declaration for the defined
// tests.
func mustAttributeDecl(t *testing.T, name QName) AttributeDeclaration {
	t.Helper()
	a, err := NewAttributeDeclaration(xsderr.Loc{}, name, nil, NewAttributeGlobalScope(), nil, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeDeclaration: %v", err)
	}
	return a
}

// TestDefinedSiblingSeesExtensionFoldedBaseElement is the consuming half of
// #51's ask, restated for #228. The sibling walk answers cvc-wildcard clause 3
// from the containing type's OWN {content type} and deliberately never follows
// {base type definition} (xs:anyType is its own base, and STYLE D3 makes the
// derived {content type} the single encoding of what the type contains). That is
// correct only because §3.4.2.3.3 clause 4.2.3.3 folds the ·base particle· INTO
// the derived {content type} — a 1..1 particle over a sequence of [base particle,
// effective content], which is exactly the shape the parser now produces and
// exactly the shape built here.
//
// So an element declared in the base must read as a sibling, with no change to
// wildcardadmit.go. If this ever fails, the defect is in the clause-4.2 merge,
// not in the walk.
func TestDefinedSiblingSeesExtensionFoldedBaseElement(t *testing.T) {
	w := wWildcard(t, DisallowedNameSibling)
	baseParticle := wParticle(t, ResolvedTerm{Term: wGroup(t, CompositorSequence,
		wParticle(t, ResolvedTerm{Term: wElement(t, wq("fromBase"), uLocalScope(t), nil, nil)}))})
	effective := wParticle(t, ResolvedTerm{Term: wGroup(t, CompositorSequence,
		wParticle(t, ResolvedTerm{Term: wElement(t, wq("fromDerived"), uLocalScope(t), nil, nil)}),
		wParticle(t, ResolvedTerm{Term: w}))})
	derived := wCT(t, wq("derived"), ResolvedTerm{Term: wGroup(t, CompositorSequence, baseParticle, effective)})

	b := NewSchemaBuilder()
	b.AddType(derived)
	s := wFinalize(t, b)

	if s.allowsElementWildcardName(w, derived, wq("fromBase")) {
		t.Error("sibling admitted a name the extension folded in from its BASE particle (cvc-wildcard clause 3)")
	}
	if s.allowsElementWildcardName(w, derived, wq("fromDerived")) {
		t.Error("sibling admitted a name declared in the extension's own effective content (cvc-wildcard clause 3)")
	}
	if !s.allowsElementWildcardName(w, derived, wq("elsewhere")) {
		t.Error("sibling rejected a name declared nowhere in the content model")
	}
}
