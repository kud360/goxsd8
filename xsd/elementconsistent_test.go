package xsd

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests are package-internal: checkElementDeclarationsConsistent runs
// inside SchemaBuilder.Finalize and is unexported (STYLE T5), so the assertions
// are made on the error Finalize returns. The builders come from
// particleattribution_test.go — one set of component helpers, not two (STYLE T4).

// eTypeTable builds a {type table} with one alternative naming altType and a
// default naming defaultType.
func eTypeTable(t *testing.T, expression string, altType, defaultType QName) *TypeTable {
	t.Helper()
	test := NewXPathExpression(expression, nil, nil, nil)
	tt, err := NewTypeTable(xsderr.Loc{},
		[]TypeAlternative{iTypeAlternative(t, &test, TypeDefinitionRef{Name: altType})},
		iTypeAlternative(t, nil, TypeDefinitionRef{Name: defaultType}))
	if err != nil {
		t.Fatalf("NewTypeTable: %v", err)
	}
	return &tt
}

// iTypeAlternative builds a TypeAlternative over the given {type definition}
// slot, failing the test on any rejection. It is the package-internal twin of
// typealternative_test.go's mustTypeAlternative, which the external package
// cannot see.
func iTypeAlternative(t *testing.T, test *XPathExpression, typeDefinition TypeDefinitionOrRef) TypeAlternative {
	t.Helper()
	ta, err := NewTypeAlternative(xsderr.Loc{}, test, typeDefinition, nil)
	if err != nil {
		t.Fatalf("NewTypeAlternative(%+v): %v", typeDefinition, err)
	}
	return ta
}

// eLocalWithTable builds a LOCAL element declaration carrying a {type table}.
func eLocalWithTable(t *testing.T, name, typeName QName, tt *TypeTable) ElementDeclaration {
	t.Helper()
	e, err := NewElementDeclaration(xsderr.Loc{}, name, TypeDefinitionRef{Name: typeName}, tt, uLocalScope(t), nil, false, nil,
		nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	return e
}

// eGlobalWithTable builds a TOP-LEVEL element declaration carrying a {type table}.
func eGlobalWithTable(t *testing.T, name, typeName QName, tt *TypeTable) ElementDeclaration {
	t.Helper()
	e, err := NewElementDeclaration(xsderr.Loc{}, name, TypeDefinitionRef{Name: typeName}, tt, NewGlobalScope(), nil, false, nil,
		nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	return e
}

// eAnonymous builds a declaration whose declared {type definition} is the
// ANONYMOUS type of an inline <simpleType> — an InlineTypeDefinition, which is
// the only encoding of anonymity since the {type definition} slot became a
// sealed sum. It is the state cos-element-consistent clause 1 forbids for two or
// more same-named declarations. The other clause-1 trigger, an ABSENT slot, is
// eAbsentType below.
func eAnonymous(t *testing.T, name QName, scope Scope) ElementDeclaration {
	t.Helper()
	st, err := newCheckedSimpleType(xsderr.Loc{}, QName{}, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType: %v", err)
	}
	e, err := NewElementDeclaration(xsderr.Loc{}, name, InlineTypeDefinition{Definition: st}, nil, scope, nil, false, nil,
		nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	return e
}

// eAbsentType builds a declaration with an ABSENT {type definition} (a nil
// slot), the second state cos-element-consistent clause 1 forbids for two or
// more same-named declarations.
func eAbsentType(t *testing.T, name QName, scope Scope) ElementDeclaration {
	t.Helper()
	e, err := NewElementDeclaration(xsderr.Loc{}, name, nil, nil, scope, nil, false, nil,
		nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	return e
}

// TestEDCDifferentTypes pins clauses 2-3: two same-named element declarations in
// one content model must name the same top-level type definition. The two are in
// a sequence, so cos-nonambig is satisfied and this rejection can only be
// cos-element-consistent.
func TestEDCDifferentTypes(t *testing.T) {
	g := uGroup(t, CompositorSequence,
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("U"))}),
	)
	expectRule(t, uSchemaWithModel(t, g, nil), ruleCosElementConsistent)
}

// TestEDCSameTypePasses is the control: same name, same declared type.
func TestEDCSameTypePasses(t *testing.T) {
	g := uGroup(t, CompositorSequence,
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
	)
	if err := uSchemaWithModel(t, g, nil); err != nil {
		t.Fatalf("two same-named declarations of one type were rejected: %v", err)
	}
}

// TestEDCAnonymousTypes pins clause 1: two DISTINCT same-named declarations whose
// declared {type definition}s are anonymous cannot be shown to be the same
// top-level definition.
func TestEDCAnonymousTypes(t *testing.T) {
	g := uGroup(t, CompositorSequence,
		uOne(t, ResolvedTerm{Term: eAnonymous(t, uq("a"), uLocalScope(t))}),
		uOne(t, ResolvedTerm{Term: eAnonymous(t, uq("a"), uLocalScope(t))}),
	)
	expectRule(t, uSchemaWithModel(t, g, nil), ruleCosElementConsistent)
}

// TestEDCAbsentTypeDefinitions is clause 1's other trigger: two DISTINCT
// same-named declarations with an ABSENT {type definition} have no {name} to
// compare either. The sealed sum keeps this case distinct from the anonymous one
// above, which the single zero-QName encoding used to conflate with it.
func TestEDCAbsentTypeDefinitions(t *testing.T) {
	g := uGroup(t, CompositorSequence,
		uOne(t, ResolvedTerm{Term: eAbsentType(t, uq("a"), uLocalScope(t))}),
		uOne(t, ResolvedTerm{Term: eAbsentType(t, uq("a"), uLocalScope(t))}),
	)
	expectRule(t, uSchemaWithModel(t, g, nil), ruleCosElementConsistent)
}

// TestEDCSameDeclarationTwiceViaRefPasses pins component identity: <element
// ref="a"/> twice contains ONE element declaration, not two, so clause 1's
// non-absent-{name} requirement must not fire on it even when a's type is
// anonymous.
func TestEDCSameDeclarationTwiceViaRefPasses(t *testing.T) {
	g := uGroup(t, CompositorSequence,
		uOne(t, ElementDeclarationRef{Name: uq("a")}),
		uOne(t, ElementDeclarationRef{Name: uq("a")}),
	)
	err := uSchemaWithModel(t, g, func(b *SchemaBuilder) {
		b.AddElement(eAnonymous(t, uq("a"), NewGlobalScope()))
	})
	if err != nil {
		t.Fatalf("one element declaration reached twice was treated as two: %v", err)
	}
}

// TestEDCSameInlineDeclarationViaTwoGroupRefsPasses is the same point one level
// up: one inline declaration reached through two <group ref>s to the same
// definition is still one component.
func TestEDCSameInlineDeclarationViaTwoGroupRefsPasses(t *testing.T) {
	inner := uGroup(t, CompositorSequence,
		uOne(t, ResolvedTerm{Term: eAnonymous(t, uq("x"), uLocalScope(t))}),
	)
	mgd, err := NewModelGroupDefinition(xsderr.Loc{}, uq("g"), inner, nil)
	if err != nil {
		t.Fatalf("NewModelGroupDefinition: %v", err)
	}
	g := uGroup(t, CompositorSequence,
		uOne(t, ModelGroupRef{Name: uq("g")}),
		uOne(t, ModelGroupRef{Name: uq("g")}),
	)
	if err := uSchemaWithModel(t, g, func(b *SchemaBuilder) { b.AddModelGroup(mgd) }); err != nil {
		t.Fatalf("one inline declaration reached through two <group ref>s was treated as two: %v", err)
	}
}

// TestEDCTypeTableDisagreement pins clause 4: same name, same type, but one
// declaration carries a {type table} and the other does not.
func TestEDCTypeTableDisagreement(t *testing.T) {
	g := uGroup(t, CompositorSequence,
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: eLocalWithTable(t, uq("a"), uq("T"), eTypeTable(t, "@kind eq 'x'", uq("T"), uq("TSub")))}),
	)
	expectRule(t, uSchemaWithModel(t, g, nil), ruleCosElementConsistent)
}

// TestEDCEquivalentTypeTablesPass pins key-equiv-ta's five-clause minimum: two
// separately-built but field-identical type tables naming top-level types are
// ·equivalent·, so the declarations agree.
func TestEDCEquivalentTypeTablesPass(t *testing.T) {
	g := uGroup(t, CompositorSequence,
		uOne(t, ResolvedTerm{Term: eLocalWithTable(t, uq("a"), uq("T"), eTypeTable(t, "@kind eq 'x'", uq("T"), uq("TSub")))}),
		uOne(t, ResolvedTerm{Term: eLocalWithTable(t, uq("a"), uq("T"), eTypeTable(t, "@kind eq 'x'", uq("T"), uq("TSub")))}),
	)
	if err := uSchemaWithModel(t, g, nil); err != nil {
		t.Fatalf("two equivalent {type table}s were rejected: %v", err)
	}
}

// TestEDCDifferentTypeTableExpressions pins key-equiv-ta clause 4: the {test}
// {expression}s must have the same value.
func TestEDCDifferentTypeTableExpressions(t *testing.T) {
	g := uGroup(t, CompositorSequence,
		uOne(t, ResolvedTerm{Term: eLocalWithTable(t, uq("a"), uq("T"), eTypeTable(t, "@kind eq 'x'", uq("T"), uq("TSub")))}),
		uOne(t, ResolvedTerm{Term: eLocalWithTable(t, uq("a"), uq("T"), eTypeTable(t, "@kind eq 'y'", uq("T"), uq("TSub")))}),
	)
	expectRule(t, uSchemaWithModel(t, g, nil), ruleCosElementConsistent)
}

// TestEDCWildcardAndTopLevelDeclaration pins the second constraint: a strict
// ·wildcard particle· in the group ·matches· the contained declaration's name and
// a top-level declaration of that name exists, so their {type table}s must agree.
func TestEDCWildcardAndTopLevelDeclaration(t *testing.T) {
	g := uGroup(t, CompositorSequence,
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("q"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uWildcard(t, NamespaceConstraintAny, nil, ProcessStrict)}),
	)
	err := uSchemaWithModel(t, g, func(b *SchemaBuilder) {
		b.AddElement(eGlobalWithTable(t, uq("q"), uq("T"), eTypeTable(t, "@kind eq 'x'", uq("T"), uq("TSub"))))
	})
	expectRule(t, err, ruleCosElementConsistent)
}

// TestEDCSkipWildcardExcluded pins the skip exclusion: clause 2.1 names strict and
// lax wildcards only, so the identical model with a skip wildcard is accepted.
func TestEDCSkipWildcardExcluded(t *testing.T) {
	g := uGroup(t, CompositorSequence,
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("q"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uWildcard(t, NamespaceConstraintAny, nil, ProcessSkip)}),
	)
	err := uSchemaWithModel(t, g, func(b *SchemaBuilder) {
		b.AddElement(eGlobalWithTable(t, uq("q"), uq("T"), eTypeTable(t, "@kind eq 'x'", uq("T"), uq("TSub"))))
	})
	if err != nil {
		t.Fatalf("a skip wildcard triggered cos-element-consistent clause 2.1: %v", err)
	}
}

// TestEDCOpenContentWildcard pins clause 2.2: the wildcard is the containing
// complex type's {open content}, not a particle in the group.
func TestEDCOpenContentWildcard(t *testing.T) {
	oc, err := NewOpenContent(xsderr.Loc{}, OpenContentInterleave, uWildcard(t, NamespaceConstraintAny, nil, ProcessLax))
	if err != nil {
		t.Fatalf("NewOpenContent: %v", err)
	}
	g := uGroup(t, CompositorSequence,
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("q"), uq("T"))}),
	)
	ct, err := NewComplexType(xsderr.Loc{}, uq("ct"), QName{}, nil, DerivationRestriction, false,
		nil, nil, nil, ElementContent{Particle: uOne(t, ResolvedTerm{Term: g}), OpenContent: &oc}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType: %v", err)
	}
	b := NewSchemaBuilder()
	b.AddType(uNamedType(t, uq("T")))
	b.AddType(uNamedType(t, uq("U")))
	b.AddType(uSubOfT(t))
	b.AddElement(eGlobalWithTable(t, uq("q"), uq("T"), eTypeTable(t, "@kind eq 'x'", uq("T"), uq("TSub"))))
	b.AddType(ct)
	_, err = b.Finalize()
	expectRule(t, err, ruleCosElementConsistent)
}

// TestEDCImplicitContainment pins ·implicitly contains· (key-impl-cont): the
// content model holds <element ref="head"/> and a local declaration named member;
// the top-level member is in head's ·substitution group·, so it is implicitly
// contained and its declared type must agree with the local one's. The two are in
// a sequence, so cos-nonambig does not fire first.
func TestEDCImplicitContainment(t *testing.T) {
	g := uGroup(t, CompositorSequence,
		uOne(t, ElementDeclarationRef{Name: uq("head")}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("member"), uq("U"))}),
	)
	err := uSchemaWithModel(t, g, func(b *SchemaBuilder) {
		b.AddElement(uGlobal(t, uq("head"), uq("T")))
		b.AddElement(uGlobal(t, uq("member"), uq("T"), uq("head")))
	})
	expectRule(t, err, ruleCosElementConsistent)
}

// TestEDCUnreferencedModelGroupDefinition pins that a Model Group Definition's
// {model group} is checked in its own right, whether or not a <group ref> points
// at it (§3.8.6's unqualified chapeau over §3.7.1's Required {model group}).
func TestEDCUnreferencedModelGroupDefinition(t *testing.T) {
	inner := uGroup(t, CompositorSequence,
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("U"))}),
	)
	mgd, err := NewModelGroupDefinition(xsderr.Loc{}, uq("orphan"), inner, nil)
	if err != nil {
		t.Fatalf("NewModelGroupDefinition: %v", err)
	}
	b := NewSchemaBuilder()
	b.AddType(uNamedType(t, uq("T")))
	b.AddType(uNamedType(t, uq("U")))
	b.AddModelGroup(mgd)
	_, err = b.Finalize()
	expectRule(t, err, ruleCosElementConsistent)
}

// TestEDCNestedGroupScope pins that the constraint binds EVERY model group at
// every depth: the two same-named declarations sit in a nested sequence, not in
// the content model's root group.
func TestEDCNestedGroupScope(t *testing.T) {
	g := uGroup(t, CompositorSequence,
		uOne(t, ResolvedTerm{Term: uGroup(t, CompositorSequence,
			uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
			uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("U"))}),
		)}),
	)
	expectRule(t, uSchemaWithModel(t, g, nil), ruleCosElementConsistent)
}

// TestTypeDefinitionsEquivalentArms pins key-equiv-ta clause 5 per ARM of the
// {type definition} slot. The load-bearing rows are the INLINE ones: an owned
// anonymous type is never the same type definition as anything, itself included,
// because component identity for one is what §3.4.6.5's no-identity Note leaves
// undetermined — and a positive answer there would let two declarations the
// schema document keeps distinct read as agreeing under key-equiv-tt.
func TestTypeDefinitionsEquivalentArms(t *testing.T) {
	anon, err := NewAnonymousComplexType(xsderr.Loc{}, ElementDeclarationContext{Component: NewComponentID()},
		QName{}, nil, DerivationRestriction, false, nil, nil, nil, EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewAnonymousComplexType: %v", err)
	}
	inline := InlineTypeDefinition{Definition: anon}
	for _, tc := range []struct {
		what string
		a, b TypeDefinitionOrRef
		want bool
	}{
		{"same name", TypeDefinitionRef{Name: uq("T")}, TypeDefinitionRef{Name: uq("T")}, true},
		{"different name", TypeDefinitionRef{Name: uq("T")}, TypeDefinitionRef{Name: uq("U")}, false},
		{"same head", SubstitutionGroupHeadTypeRef{Head: uq("h")}, SubstitutionGroupHeadTypeRef{Head: uq("h")}, true},
		{"different head", SubstitutionGroupHeadTypeRef{Head: uq("h")}, SubstitutionGroupHeadTypeRef{Head: uq("g")}, false},
		{"mismatched arms", TypeDefinitionRef{Name: uq("T")}, SubstitutionGroupHeadTypeRef{Head: uq("T")}, false},
		{"inline against itself", inline, inline, false},
		{"inline against a name", inline, TypeDefinitionRef{Name: uq("T")}, false},
		{"a name against inline", TypeDefinitionRef{Name: uq("T")}, inline, false},
		{"absent against a name", nil, TypeDefinitionRef{Name: uq("T")}, false},
		{"a name against absent", TypeDefinitionRef{Name: uq("T")}, nil, false},
	} {
		if got := typeDefinitionsEquivalent(tc.a, tc.b); got != tc.want {
			t.Errorf("%s: typeDefinitionsEquivalent = %v, want %v", tc.what, got, tc.want)
		}
	}
}
