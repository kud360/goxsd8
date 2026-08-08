package xsd_test

import (
	"reflect"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// mustSimpleType builds a minimal valid *SimpleType (xs:anySimpleType shape:
// nil variety, nil base) for use as a SimpleContent {simple type definition}.
func mustSimpleType(t *testing.T) *xsd.SimpleType {
	t.Helper()
	st, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Local: "st"}, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType unexpected error: %v", err)
	}
	return st
}

// mustParticleWithTerm builds a valid Particle carrying a present {term}.
func mustParticleWithTerm(t *testing.T) xsd.Particle {
	t.Helper()
	p, err := xsd.NewParticle(xsderr.Loc{}, xsd.Occurs{}, xsd.ElementDeclarationRef{Name: xsd.QName{Local: "e"}}, nil)
	if err != nil {
		t.Fatalf("NewParticle unexpected error: %v", err)
	}
	return p
}

// mustAttributeUse builds a valid AttributeUse referencing a top-level
// declaration by name.
func mustAttributeUse(t *testing.T) xsd.AttributeUse {
	t.Helper()
	u, err := xsd.NewAttributeUse(xsderr.Loc{}, false, xsd.AttributeDeclarationRef{Name: xsd.QName{Local: "a"}}, nil, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeUse unexpected error: %v", err)
	}
	return u
}

func TestNewComplexTypeEmptyContent(t *testing.T) {
	c, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "ct"}, xsd.QName{Local: "base"}, nil,
		xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType unexpected error: %v", err)
	}
	if c.ContentType().Variety() != xsd.ContentEmpty {
		t.Errorf("Variety() = %s, want empty", c.ContentType().Variety())
	}
	if got := c.Base(); got != (xsd.TypeDefinitionOrRef(xsd.TypeDefinitionRef{Name: xsd.QName{Local: "base"}})) {
		t.Errorf("Base() = %#v, want a TypeDefinitionRef naming {base}", got)
	}
	if c.DerivationMethod() != xsd.DerivationRestriction {
		t.Errorf("DerivationMethod() = %s, want restriction", c.DerivationMethod())
	}
}

func TestNewComplexTypeSimpleContent(t *testing.T) {
	st := mustSimpleType(t)
	c, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "ct"}, xsd.QName{Local: "base"}, nil,
		xsd.DerivationExtension, false, nil, nil, nil, xsd.SimpleContent{SimpleType: st}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType unexpected error: %v", err)
	}
	sc, ok := c.ContentType().(xsd.SimpleContent)
	if !ok {
		t.Fatalf("ContentType() type = %T, want SimpleContent", c.ContentType())
	}
	if sc.SimpleType != st {
		t.Errorf("SimpleContent.SimpleType = %p, want %p", sc.SimpleType, st)
	}
	if sc.Variety() != xsd.ContentSimple {
		t.Errorf("Variety() = %s, want simple", sc.Variety())
	}
}

func TestNewComplexTypeElementContentVarietyDerivation(t *testing.T) {
	cases := []struct {
		name  string
		mixed bool
		want  xsd.ContentTypeVariety
	}{
		{"element-only", false, xsd.ContentElementOnly},
		{"mixed", true, xsd.ContentMixed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ec := xsd.ElementContent{Mixed: tc.mixed, Particle: mustParticleWithTerm(t)}
			c, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "ct"}, xsd.QName{Local: "base"}, nil,
				xsd.DerivationRestriction, false, nil, nil, nil, ec, nil, nil, nil)
			if err != nil {
				t.Fatalf("NewComplexType unexpected error: %v", err)
			}
			if c.ContentType().Variety() != tc.want {
				t.Errorf("Variety() = %s, want %s", c.ContentType().Variety(), tc.want)
			}
		})
	}
}

func TestNewComplexTypeElementContentWithOpenContent(t *testing.T) {
	w := mustWildcard(t, mustConstraint(t, xsd.NamespaceConstraintAny, nil, nil), xsd.ProcessLax, nil)
	oc, err := xsd.NewOpenContent(xsderr.Loc{}, xsd.OpenContentInterleave, w)
	if err != nil {
		t.Fatalf("NewOpenContent unexpected error: %v", err)
	}
	ec := xsd.ElementContent{Mixed: true, Particle: mustParticleWithTerm(t), OpenContent: &oc}
	c, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "ct"}, xsd.QName{Local: "base"}, nil,
		xsd.DerivationExtension, false, nil, nil, nil, ec, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType unexpected error: %v", err)
	}
	got, ok := c.ContentType().(xsd.ElementContent)
	if !ok {
		t.Fatalf("ContentType() type = %T, want ElementContent", c.ContentType())
	}
	if got.OpenContent == nil || got.OpenContent.Mode() != xsd.OpenContentInterleave {
		t.Errorf("OpenContent.Mode() = %v, want interleave", got.OpenContent)
	}
}

func TestNewComplexTypeRejectsNilContentType(t *testing.T) {
	_, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "ct"}, xsd.QName{Local: "base"}, nil,
		xsd.DerivationRestriction, false, nil, nil, nil, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("NewComplexType accepted a nil {content type}, want ct-props-correct error")
	}
	assertRule(t, err, "ct-props-correct")
}

func TestNewComplexTypeRejectsNilSimpleType(t *testing.T) {
	_, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "ct"}, xsd.QName{Local: "base"}, nil,
		xsd.DerivationExtension, false, nil, nil, nil, xsd.SimpleContent{SimpleType: nil}, nil, nil, nil)
	if err == nil {
		t.Fatal("NewComplexType accepted a nil {simple type definition}, want ct-props-correct error")
	}
	assertRule(t, err, "ct-props-correct")
}

func TestNewComplexTypeRejectsElementContentAbsentTerm(t *testing.T) {
	// A zero Particle{} has an absent {term}; NewComplexType must reject an
	// ElementContent built around it (bypassing NewParticle's own check).
	ec := xsd.ElementContent{Mixed: false, Particle: xsd.Particle{}}
	_, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "ct"}, xsd.QName{Local: "base"}, nil,
		xsd.DerivationRestriction, false, nil, nil, nil, ec, nil, nil, nil)
	if err == nil {
		t.Fatal("NewComplexType accepted an ElementContent with an absent {term}, want ct-props-correct error")
	}
	assertRule(t, err, "ct-props-correct")
}

func TestNewComplexTypeRejectsInvalidDerivationMethod(t *testing.T) {
	for _, m := range []xsd.DerivationMethod{xsd.DerivationSubstitution, xsd.DerivationList, xsd.DerivationUnion, 0} {
		_, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "ct"}, xsd.QName{Local: "base"}, nil,
			m, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
		if err == nil {
			t.Fatalf("NewComplexType accepted {derivation method} = %s, want ct-props-correct error", m)
		}
		assertRule(t, err, "ct-props-correct")
	}
}

func TestNewComplexTypeRejectsInvalidFinal(t *testing.T) {
	_, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "ct"}, xsd.QName{Local: "base"},
		[]xsd.DerivationMethod{xsd.DerivationSubstitution},
		xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err == nil {
		t.Fatal("NewComplexType accepted an invalid {final} member, want ct-props-correct error")
	}
	assertRule(t, err, "ct-props-correct")
}

func TestNewComplexTypeRejectsInvalidProhibitedSubstitutions(t *testing.T) {
	_, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "ct"}, xsd.QName{Local: "base"}, nil,
		xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{},
		[]xsd.DerivationMethod{xsd.DerivationUnion}, nil, nil)
	if err == nil {
		t.Fatal("NewComplexType accepted an invalid {prohibited substitutions} member, want ct-props-correct error")
	}
	assertRule(t, err, "ct-props-correct")
}

func TestNewOpenContentRejectsInvalidMode(t *testing.T) {
	w := mustWildcard(t, mustConstraint(t, xsd.NamespaceConstraintAny, nil, nil), xsd.ProcessStrict, nil)
	_, err := xsd.NewOpenContent(xsderr.Loc{}, 0, w)
	if err == nil {
		t.Fatal("NewOpenContent accepted an invalid {mode}, want ct-props-correct error")
	}
	assertRule(t, err, "ct-props-correct")
}

func TestComplexTypeSlicesDoNotAlias(t *testing.T) {
	final := []xsd.DerivationMethod{xsd.DerivationExtension}
	prohibited := []xsd.DerivationMethod{xsd.DerivationRestriction}
	uses := []xsd.AttributeUse{mustAttributeUse(t)}
	c, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "ct"}, xsd.QName{Local: "base"}, final,
		xsd.DerivationRestriction, false, uses, nil, nil, xsd.EmptyContent{}, prohibited, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType unexpected error: %v", err)
	}
	final[0] = xsd.DerivationRestriction
	prohibited[0] = xsd.DerivationExtension
	uses[0] = xsd.AttributeUse{}
	if c.Final()[0] != xsd.DerivationExtension {
		t.Errorf("ComplexType aliased the {final} slice: got %s", c.Final()[0])
	}
	if c.ProhibitedSubstitutions()[0] != xsd.DerivationRestriction {
		t.Errorf("ComplexType aliased the {prohibited substitutions} slice: got %s", c.ProhibitedSubstitutions()[0])
	}
	if len(c.AttributeUses()) != 1 {
		t.Fatalf("AttributeUses() len = %d, want 1", len(c.AttributeUses()))
	}
}

func TestComplexTypeAttributeWildcardOptional(t *testing.T) {
	c, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "ct"}, xsd.QName{Local: "base"}, nil,
		xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType unexpected error: %v", err)
	}
	if _, ok := c.AttributeWildcard(); ok {
		t.Error("AttributeWildcard() present, want absent")
	}
	w := mustWildcard(t, mustConstraint(t, xsd.NamespaceConstraintAny, nil, nil), xsd.ProcessStrict, nil)
	c, err = xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "ct"}, xsd.QName{Local: "base"}, nil,
		xsd.DerivationRestriction, false, nil, nil, &w, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType unexpected error: %v", err)
	}
	if _, ok := c.AttributeWildcard(); !ok {
		t.Error("AttributeWildcard() absent, want present")
	}
}

// --- {context} (§3.4.1 ctd-context) ---------------------------------------

func TestNewComplexTypeRejectsAbsentName(t *testing.T) {
	// §3.4.1 makes {context} Required when {name} is absent, and this entry
	// point cannot supply one. The Space-only case is the reason the guard
	// tests name.Local rather than name == QName{}: QName{Space: "urn:x"}
	// would otherwise pass as a NAMED type.
	for _, name := range []xsd.QName{{}, {Space: "urn:x"}, {Space: "urn:x", Local: ""}} {
		_, err := xsd.NewComplexType(xsderr.Loc{}, name, xsd.QName{Local: "base"}, nil,
			xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
		if err == nil {
			t.Fatalf("NewComplexType accepted {name} = %v, want ct-props-correct error", name)
		}
		assertRule(t, err, "ct-props-correct")
	}
}

func TestNewComplexTypeNamedHasAbsentContext(t *testing.T) {
	c, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "ct"}, xsd.QName{Local: "base"}, nil,
		xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType unexpected error: %v", err)
	}
	got, ok := c.Context()
	if ok {
		t.Errorf("Context() = (%v, true) on a NAMED type, want (nil, false) per the §3.4.1 tableau", got)
	}
	if got != nil {
		t.Errorf("Context() first result = %v, want nil when absent", got)
	}
}

func TestNewAnonymousComplexTypeRejectsNilContext(t *testing.T) {
	_, err := xsd.NewAnonymousComplexType(xsderr.Loc{}, nil, xsd.QName{Local: "base"}, nil,
		xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err == nil {
		t.Fatal("NewAnonymousComplexType accepted a nil {context}, want ct-props-correct error")
	}
	assertRule(t, err, "ct-props-correct")
}

func TestNewAnonymousComplexTypeRejectsUnmintedIdentity(t *testing.T) {
	// Both arms, each carrying the zero ComponentID: a representation
	// invariant this package owns, not a spec clause.
	for _, ctx := range []xsd.ComplexTypeContext{
		xsd.ElementDeclarationContext{},
		xsd.ComplexTypeDefinitionContext{},
	} {
		_, err := xsd.NewAnonymousComplexType(xsderr.Loc{}, ctx, xsd.QName{Local: "base"}, nil,
			xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
		if err == nil {
			t.Fatalf("NewAnonymousComplexType accepted %T with an unminted identity, want component-invariant error", ctx)
		}
		assertRule(t, err, xsderr.RuleComponentInvariant)
	}
}

func TestNewAnonymousComplexTypeContext(t *testing.T) {
	for _, tc := range []struct {
		name string
		make func(xsd.ComponentID) xsd.ComplexTypeContext
	}{
		{"element declaration", func(id xsd.ComponentID) xsd.ComplexTypeContext {
			return xsd.ElementDeclarationContext{Component: id}
		}},
		{"complex type definition", func(id xsd.ComponentID) xsd.ComplexTypeContext {
			return xsd.ComplexTypeDefinitionContext{Component: id}
		}},
	} {
		id := xsd.NewComponentID()
		ctx := tc.make(id)
		c, err := xsd.NewAnonymousComplexType(xsderr.Loc{}, ctx, xsd.QName{Local: "base"}, nil,
			xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
		if err != nil {
			t.Fatalf("%s: NewAnonymousComplexType unexpected error: %v", tc.name, err)
		}
		if got := c.Name(); got != (xsd.QName{}) {
			t.Errorf("%s: Name() = %v, want the absent QName", tc.name, got)
		}
		got, ok := c.Context()
		if !ok {
			t.Fatalf("%s: Context() absent, want present", tc.name)
		}
		if reflect.TypeOf(got) != reflect.TypeOf(ctx) {
			t.Errorf("%s: Context() arm = %T, want %T", tc.name, got, ctx)
		}
		if got.ID() != id {
			t.Errorf("%s: Context().ID() is not the minted identity", tc.name)
		}
		if got.ID() == (xsd.ComponentID{}) {
			t.Errorf("%s: Context().ID() is the unminted identity", tc.name)
		}
	}
}

// TestComplexTypeContextSurvivesCopy is the end-to-end form of the claim
// ComponentID rests on: a ComplexType is a value, and copying one — by
// assignment, through a by-value parameter, and by boxing into TypeDefinition
// and back out — preserves the {context} identity under ==.
func TestComplexTypeContextSurvivesCopy(t *testing.T) {
	id := xsd.NewComponentID()
	c, err := xsd.NewAnonymousComplexType(xsderr.Loc{}, xsd.ElementDeclarationContext{Component: id},
		xsd.QName{Local: "base"}, nil, xsd.DerivationRestriction, false, nil, nil, nil,
		xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewAnonymousComplexType unexpected error: %v", err)
	}

	contextID := func(t *testing.T, what string, ct xsd.ComplexType) xsd.ComponentID {
		t.Helper()
		got, ok := ct.Context()
		if !ok {
			t.Fatalf("%s: Context() absent, want present", what)
		}
		return got.ID()
	}

	if got := contextID(t, "original", c); got != id {
		t.Fatal("original: Context().ID() is not the minted identity")
	}

	copied := c
	if got := contextID(t, "assignment copy", copied); got != id {
		t.Error("assignment copy: Context().ID() != the minted identity")
	}

	// Boxed into the sealed TypeDefinition sum and asserted back out.
	var td xsd.TypeDefinition = c
	back, ok := td.(xsd.ComplexType)
	if !ok {
		t.Fatalf("TypeDefinition round-trip yielded %T, want xsd.ComplexType", td)
	}
	if got := contextID(t, "TypeDefinition round-trip", back); got != id {
		t.Error("TypeDefinition round-trip: Context().ID() != the minted identity")
	}

	// A DIFFERENT identity must not compare equal, so the assertions above
	// are not satisfied by any ComponentID at all.
	other := xsd.NewComponentID()
	if got := contextID(t, "original", c); got == other {
		t.Error("Context().ID() equals an unrelated minted identity")
	}
}
