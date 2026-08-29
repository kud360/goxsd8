package xsd

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests pin the REACH both finalize folds gained in #414: the §3.4.2.4
// clause 3 and §3.4.2.5 clause 2 values of a complex type a DECLARATION owns,
// which is in no {type definitions} slice and so was reached by neither fold's
// position walk. What each rule computes is pinned by attributeusefold_test.go
// and attributewildcardfold_test.go; what is asserted here is only that a type
// at each owning slot gets the same answer a named one would.
//
// The component builders come from complexderivation_test.go,
// complexextension_test.go, elementconsistent_test.go and
// particleattribution_test.go — one set of helpers, not five (STYLE T4).

// oInline builds an ANONYMOUS complex type extending base, for the slots a
// declaration owns. dType's anonymous arm hard-codes DerivationRestriction and
// every fixture here needs an extension: clause 3.1 and clause 2.2 are the two
// branches that read the base at all, so a restriction would fold to its own
// value whether the widening works or not.
func oInline(t *testing.T, base QName, uses []AttributeUse, wildcard *Wildcard, content ContentType) ComplexType {
	t.Helper()
	ct, err := NewAnonymousComplexType(xsderr.Loc{}, ElementDeclarationContext{Component: NewComponentID()},
		base, nil, DerivationExtension, false, uses, nil, wildcard, content, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewAnonymousComplexType: %v", err)
	}
	return ct
}

// oBase is the named type every fixture extends: one attribute use and one
// <anyAttribute>, so a single base serves both folds.
func oBase(t *testing.T, name QName) ComplexType {
	t.Helper()
	w := uWildcard(t, NamespaceConstraintAny, nil, ProcessLax)
	return dType(t, name, anyTypeName, EmptyContent{}, []AttributeUse{dAttr(t, uq("b"), uq("str"))}, &w)
}

// oOwnedType reads back the complex type an element declaration's own {type
// definition} slot owns, failing the test if the slot does not hold one.
func oOwnedType(t *testing.T, e ElementDeclaration) ComplexType {
	t.Helper()
	c, owns := ownedComplexType(e.TypeDefinition())
	if !owns {
		t.Fatalf("element declaration %s owns no anonymous complex type", e.Name())
	}
	return c
}

// oAlternativeType reads back the complex type a Type Alternative's {type
// definition} slot owns, failing the test if the slot does not hold one.
func oAlternativeType(t *testing.T, a TypeAlternative) ComplexType {
	t.Helper()
	c, owns := ownedComplexType(a.TypeDefinition())
	if !owns {
		t.Fatal("the alternative owns no anonymous complex type")
	}
	return c
}

// oGlobal reads a top-level element declaration back off the finalized schema.
func oGlobal(t *testing.T, s *Schema, name QName) ElementDeclaration {
	t.Helper()
	e, ok := s.Element(name)
	if !ok {
		t.Fatalf("element declaration %s is not in the finalized schema", name)
	}
	return e
}

// oLocal reads back the single local element declaration in a named type's
// content model, following {content type} -> {particle} -> {term}. It is how a
// fixture observes a type nested in a PARTICLE TREE, the shape neither fold's
// outer walk could reach.
func oLocal(t *testing.T, s *Schema, typeName QName) ElementDeclaration {
	t.Helper()
	def, ok := s.Type(typeName)
	if !ok {
		t.Fatalf("type %s is not in the finalized schema", typeName)
	}
	content, ok := def.(ComplexType).ContentType().(ElementContent)
	if !ok {
		t.Fatalf("type %s does not carry element content", typeName)
	}
	g, ok := content.Particle.Term().(ResolvedTerm).Term.(ModelGroup)
	if !ok {
		t.Fatalf("type %s does not carry a model group", typeName)
	}
	e, ok := g.Particles()[0].Term().(ResolvedTerm).Term.(ElementDeclaration)
	if !ok {
		t.Fatalf("type %s does not open with an element declaration", typeName)
	}
	return e
}

// oUseNames is the expanded local names of a {attribute uses} value, in the
// order the property holds them. Order is asserted for the reason fUses asserts
// it: the fold is required to be document-ordered (STYLE D2).
func oUseNames(c ComplexType) []string {
	var names []string
	for _, u := range c.AttributeUses() {
		names = append(names, u.DeclarationName().Local)
	}
	return names
}

// oSchema finalizes a schema carrying xs:anyType, the simple type the attribute
// declarations name, and the base every fixture extends.
func oSchema(t *testing.T, build func(*SchemaBuilder)) *Schema {
	t.Helper()
	b := NewSchemaBuilder()
	b.AddType(dAnyType(t))
	b.AddType(dPrimitive(t, uq("str")))
	b.AddType(oBase(t, uq("Base")))
	build(b)
	s, err := b.Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	return s
}

// TestOwnedFoldGlobalElementOwnType pins both folds over the inline
// <complexType> child of a TOP-LEVEL <element>: §3.4.2.4 clause 3.1 inherits the
// base's attribute use, and §3.4.2.5 clause 2.2.2.2 hands back the ·base
// wildcard· the extension does not declare itself.
func TestOwnedFoldGlobalElementOwnType(t *testing.T) {
	s := oSchema(t, func(b *SchemaBuilder) {
		inline := oInline(t, uq("Base"), []AttributeUse{dAttr(t, uq("own"), uq("str"))}, nil, EmptyContent{})
		b.AddElement(dOwnInline(t, uq("g"), inline, NewGlobalScope()))
	})
	c := oOwnedType(t, oGlobal(t, s, uq("g")))
	if got, want := oUseNames(c), []string{"own", "b"}; !fEqual(got, want) {
		t.Fatalf("{attribute uses} of g's inline type = %v, want %v", got, want)
	}
	if _, ok := c.AttributeWildcard(); !ok {
		t.Fatal("{attribute wildcard} of g's inline type is absent; clause 2.2.2.2 inherits the base's")
	}
}

// TestOwnedFoldParticleTreeElement pins both folds over an inline
// <complexType> nested in a PARTICLE TREE — a local <element> inside a named
// type's content model, the shape the two GAP markers named.
func TestOwnedFoldParticleTreeElement(t *testing.T) {
	s := oSchema(t, func(b *SchemaBuilder) {
		inline := oInline(t, uq("Base"), []AttributeUse{dAttr(t, uq("own"), uq("str"))}, nil, EmptyContent{})
		local := dOwnInline(t, uq("child"), inline, uLocalScope(t))
		b.AddType(uCT(t, uq("Holder"), uOne(t, ResolvedTerm{Term: uGroup(t, CompositorSequence, uOne(t, ResolvedTerm{Term: local}))})))
	})
	c := oOwnedType(t, oLocal(t, s, uq("Holder")))
	if got, want := oUseNames(c), []string{"own", "b"}; !fEqual(got, want) {
		t.Fatalf("{attribute uses} of the nested inline type = %v, want %v", got, want)
	}
	if _, ok := c.AttributeWildcard(); !ok {
		t.Fatal("{attribute wildcard} of the nested inline type is absent; clause 2.2.2.2 inherits the base's")
	}
}

// TestOwnedFoldAlternativeInlineType pins both folds over an <alternative>'s own
// inline <complexType> (§3.12.2 declare-ta's second arm). That slot is the one a
// real parser reaches with no named-type fallback, and validate's
// cvc-complex-type clause 3 reads the selected type's {attribute uses}, so an
// unfolded extension there rejects an attribute its base declares (#851).
func TestOwnedFoldAlternativeInlineType(t *testing.T) {
	s := oSchema(t, func(b *SchemaBuilder) {
		inline := oInline(t, uq("Base"), []AttributeUse{dAttr(t, uq("own"), uq("str"))}, nil, EmptyContent{})
		test := NewXPathExpression("true()", nil, nil, nil)
		tt, err := NewTypeTable(xsderr.Loc{},
			[]TypeAlternative{iTypeAlternative(t, &test, InlineTypeDefinition{Definition: inline})},
			iTypeAlternative(t, nil, TypeDefinitionRef{Name: uq("Base")}))
		if err != nil {
			t.Fatalf("NewTypeTable: %v", err)
		}
		context, _ := inline.Context()
		e, err := NewElementDeclarationOwningTypes(xsderr.Loc{}, context.ID(), uq("g"),
			TypeDefinitionRef{Name: uq("Base")}, &tt, NewGlobalScope(),
			nil, false, nil, nil, nil, false, nil, nil)
		if err != nil {
			t.Fatalf("NewElementDeclarationOwningTypes(g): %v", err)
		}
		b.AddElement(e)
	})
	table, ok := oGlobal(t, s, uq("g")).TypeTable()
	if !ok {
		t.Fatal("element declaration g carries no {type table}")
	}
	c := oAlternativeType(t, table.Alternatives()[0])
	if got, want := oUseNames(c), []string{"own", "b"}; !fEqual(got, want) {
		t.Fatalf("{attribute uses} of the alternative's inline type = %v, want %v", got, want)
	}
	if _, ok := c.AttributeWildcard(); !ok {
		t.Fatal("{attribute wildcard} of the alternative's inline type is absent; clause 2.2.2.2 inherits the base's")
	}
}

// TestOwnedFoldNestedInsideOwnedType pins that the descent does not stop at the
// first owned type: an inline <complexType> declared INSIDE another inline one
// is folded too, so the widening is a full descent and not a single extra level.
func TestOwnedFoldNestedInsideOwnedType(t *testing.T) {
	s := oSchema(t, func(b *SchemaBuilder) {
		inner := oInline(t, uq("Base"), []AttributeUse{dAttr(t, uq("inner"), uq("str"))}, nil, EmptyContent{})
		local := dOwnInline(t, uq("child"), inner, uLocalScope(t))
		outer := oInline(t, uq("Base"), nil, nil,
			ElementContent{Particle: uOne(t, ResolvedTerm{Term: uGroup(t, CompositorSequence, uOne(t, ResolvedTerm{Term: local}))})})
		b.AddElement(dOwnInline(t, uq("g"), outer, NewGlobalScope()))
	})
	content, ok := oOwnedType(t, oGlobal(t, s, uq("g"))).ContentType().(ElementContent)
	if !ok {
		t.Fatal("g's inline type does not carry element content")
	}
	g := content.Particle.Term().(ResolvedTerm).Term.(ModelGroup)
	nested := oOwnedType(t, g.Particles()[0].Term().(ResolvedTerm).Term.(ElementDeclaration))
	if got, want := oUseNames(nested), []string{"inner", "b"}; !fEqual(got, want) {
		t.Fatalf("{attribute uses} of the twice-nested inline type = %v, want %v", got, want)
	}
}

// TestOwnedFoldModelGroupDefinition pins the third root: a top-level <group>'s
// particles reach the fold too, so a declaration inside a named model group is
// not folded only where some complex type happens to reference the group.
func TestOwnedFoldModelGroupDefinition(t *testing.T) {
	s := oSchema(t, func(b *SchemaBuilder) {
		inline := oInline(t, uq("Base"), []AttributeUse{dAttr(t, uq("own"), uq("str"))}, nil, EmptyContent{})
		local := dOwnInline(t, uq("child"), inline, uLocalScope(t))
		d, err := NewModelGroupDefinition(xsderr.Loc{}, uq("G"),
			uGroup(t, CompositorSequence, uOne(t, ResolvedTerm{Term: local})), nil)
		if err != nil {
			t.Fatalf("NewModelGroupDefinition: %v", err)
		}
		b.AddModelGroup(d)
	})
	d, ok := s.ModelGroup(uq("G"))
	if !ok {
		t.Fatal("model group definition G is not in the finalized schema")
	}
	c := oOwnedType(t, d.ModelGroup().Particles()[0].Term().(ResolvedTerm).Term.(ElementDeclaration))
	if got, want := oUseNames(c), []string{"own", "b"}; !fEqual(got, want) {
		t.Fatalf("{attribute uses} of the group's inline type = %v, want %v", got, want)
	}
}

// TestOwnedFoldLeavesTheCallersSlicesAlone pins the two copies the descent
// makes — modelGroup's {particles} and typeTable's {alternatives}. Both
// constructors copy their input slice, so the array the finalized Schema walks
// is the one the ModelGroup or TypeTable VALUE the caller still holds points
// at; rewriting a member in place would fold that retained component too, and
// AttributeUses documents an unfinalized component as "only what that caller
// passed in".
//
// Each case reads the same component twice: through the finalized Schema, where
// the fold must have run, and through the value the caller kept, where it must
// not have.
func TestOwnedFoldLeavesTheCallersSlicesAlone(t *testing.T) {
	t.Run("model group particles", func(t *testing.T) {
		inline := oInline(t, uq("Base"), []AttributeUse{dAttr(t, uq("own"), uq("str"))}, nil, EmptyContent{})
		retained := uGroup(t, CompositorSequence, uOne(t, ResolvedTerm{Term: dOwnInline(t, uq("child"), inline, uLocalScope(t))}))
		s := oSchema(t, func(b *SchemaBuilder) {
			b.AddType(uCT(t, uq("Holder"), uOne(t, ResolvedTerm{Term: retained})))
		})
		if got, want := oUseNames(oOwnedType(t, oLocal(t, s, uq("Holder")))), []string{"own", "b"}; !fEqual(got, want) {
			t.Fatalf("{attribute uses} of the inline type under Holder = %v, want %v", got, want)
		}
		kept := oOwnedType(t, retained.Particles()[0].Term().(ResolvedTerm).Term.(ElementDeclaration))
		if got, want := oUseNames(kept), []string{"own"}; !fEqual(got, want) {
			t.Fatalf("Finalize rewrote the {particles} array the caller still holds: {attribute uses} = %v, want %v", got, want)
		}
	})
	t.Run("type table alternatives", func(t *testing.T) {
		inline := oInline(t, uq("Base"), []AttributeUse{dAttr(t, uq("own"), uq("str"))}, nil, EmptyContent{})
		test := NewXPathExpression("true()", nil, nil, nil)
		retained, err := NewTypeTable(xsderr.Loc{},
			[]TypeAlternative{iTypeAlternative(t, &test, InlineTypeDefinition{Definition: inline})},
			iTypeAlternative(t, nil, TypeDefinitionRef{Name: uq("Base")}))
		if err != nil {
			t.Fatalf("NewTypeTable: %v", err)
		}
		s := oSchema(t, func(b *SchemaBuilder) {
			context, _ := inline.Context()
			e, err := NewElementDeclarationOwningTypes(xsderr.Loc{}, context.ID(), uq("g"),
				TypeDefinitionRef{Name: uq("Base")}, &retained, NewGlobalScope(),
				nil, false, nil, nil, nil, false, nil, nil)
			if err != nil {
				t.Fatalf("NewElementDeclarationOwningTypes(g): %v", err)
			}
			b.AddElement(e)
		})
		table, ok := oGlobal(t, s, uq("g")).TypeTable()
		if !ok {
			t.Fatal("element declaration g carries no {type table}")
		}
		if got, want := oUseNames(oAlternativeType(t, table.Alternatives()[0])), []string{"own", "b"}; !fEqual(got, want) {
			t.Fatalf("{attribute uses} of the alternative's inline type = %v, want %v", got, want)
		}
		kept := oAlternativeType(t, retained.Alternatives()[0])
		if got, want := oUseNames(kept), []string{"own"}; !fEqual(got, want) {
			t.Fatalf("Finalize rewrote the {alternatives} array the caller still holds: {attribute uses} = %v, want %v", got, want)
		}
	})
}
