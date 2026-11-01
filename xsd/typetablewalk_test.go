package xsd

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// twRestriction builds an anonymous simple type restricting the OWNED base, as
// a producer maps <simpleType><restriction> with an inline base; NewSimpleType
// defers st-props-correct clause 1 to Finalize.
func twRestriction(t *testing.T, base *SimpleType) *SimpleType {
	t.Helper()
	st, err := NewSimpleType(xsderr.Loc{}, QName{}, RestrictionDerivation{}, OwnedSimpleType{Definition: base}, nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType: %v", err)
	}
	return st
}

// twTable builds a {type table} holding one alternative testing @k over
// alternative, and a {default type definition} over dflt.
func twTable(t *testing.T, alternative, dflt TypeDefinitionOrRef) *TypeTable {
	t.Helper()
	test := NewXPathExpression("@k", nil, nil, nil)
	tt, err := NewTypeTable(xsderr.Loc{}, []TypeAlternative{iTypeAlternative(t, &test, alternative)}, iTypeAlternative(t, nil, dflt))
	if err != nil {
		t.Fatalf("NewTypeTable: %v", err)
	}
	return &tt
}

// twElement is a global element declaration e of type str carrying
// twTable(alternative, dflt).
func twElement(t *testing.T, alternative, dflt TypeDefinitionOrRef) ElementDeclaration {
	t.Helper()
	return ttElement(t, uq("e"), uq("str"), twTable(t, alternative, dflt))
}

// twOwningElement builds a global element declaration e of type xs:anyType
// whose first {type table} alternative owns an anonymous complex type, a
// restriction of xs:anyType holding one local element l of type local, and
// whose {default type definition} is xs:anyType.
func twOwningElement(t *testing.T, local *SimpleType) ElementDeclaration {
	t.Helper()
	l, err := NewElementDeclaration(xsderr.Loc{}, uq("l"), InlineTypeDefinition{Definition: local}, nil, uLocalScope(t),
		nil, false, nil, nil, nil, false, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration(l): %v", err)
	}
	ct := dType(t, QName{}, anyTypeName, dElementContent(t, false, uGroup(t, CompositorSequence, uOne(t, ResolvedTerm{Term: l}))), nil, nil)
	context, _ := ct.Context()
	tt := twTable(t, InlineTypeDefinition{Definition: ct}, TypeDefinitionRef{Name: anyTypeName})
	e, err := NewElementDeclarationOwningTypes(xsderr.Loc{}, context.ID(), uq("e"), TypeDefinitionRef{Name: anyTypeName}, tt,
		NewGlobalScope(), nil, false, nil, nil, nil, false, nil)
	if err != nil {
		t.Fatalf("NewElementDeclarationOwningTypes: %v", err)
	}
	return e
}

// TestFinalizeChargesTypeTableOwnedSimpleTypes pins that Finalize charges
// st-props-correct clause 1 against a simple type a {type table} owns, or one
// nested under such a type, built through SchemaBuilder: the builder twins of
// TestParseChargesTypeTableOwnedComponents's P5, P6, P7, P10 and P11 rows. Each
// row finalized before componentWalk's walkElementDeclaration entered the
// table.
func TestFinalizeChargesTypeTableOwnedSimpleTypes(t *testing.T) {
	str := TypeDefinitionRef{Name: uq("str")}
	tests := []struct {
		name  string
		build func(t *testing.T) ElementDeclaration
	}{
		{"alternative restricts anySimpleType", func(t *testing.T) ElementDeclaration {
			return twElement(t, InlineTypeDefinition{Definition: twRestriction(t, AnySimpleType())}, str)
		}},
		{"alternative restricts anyAtomicType", func(t *testing.T) ElementDeclaration {
			return twElement(t, InlineTypeDefinition{Definition: twRestriction(t, AnyAtomicType())}, str)
		}},
		{"default restricts anyAtomicType", func(t *testing.T) ElementDeclaration {
			return twElement(t, str, InlineTypeDefinition{Definition: twRestriction(t, AnyAtomicType())})
		}},
		{"two-step chain under an alternative", func(t *testing.T) ElementDeclaration {
			return twElement(t, InlineTypeDefinition{Definition: twRestriction(t, twRestriction(t, AnyAtomicType()))}, str)
		}},
		{"local element in an alternative's complex type", func(t *testing.T) ElementDeclaration {
			return twOwningElement(t, twRestriction(t, AnySimpleType()))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectRule(t, cwFinalize(t, func(b *SchemaBuilder) { b.AddElement(tt.build(t)) }), ruleSTPropsCorrect)
		})
	}
}

// TestFinalizeAcceptsWellFormedTypeTableOwnedSimpleTypes is the guard over
// TestFinalizeChargesTypeTableOwnedSimpleTypes: the same shapes over a
// well-formed restriction of str still finalize.
func TestFinalizeAcceptsWellFormedTypeTableOwnedSimpleTypes(t *testing.T) {
	str := TypeDefinitionRef{Name: uq("str")}
	tests := []struct {
		name  string
		build func(t *testing.T, base *SimpleType) ElementDeclaration
	}{
		{"alternative", func(t *testing.T, base *SimpleType) ElementDeclaration {
			return twElement(t, InlineTypeDefinition{Definition: twRestriction(t, base)}, str)
		}},
		{"default", func(t *testing.T, base *SimpleType) ElementDeclaration {
			return twElement(t, str, InlineTypeDefinition{Definition: twRestriction(t, base)})
		}},
		{"local element in an alternative's complex type", func(t *testing.T, base *SimpleType) ElementDeclaration {
			return twOwningElement(t, twRestriction(t, base))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewSchemaBuilder()
			b.AddType(dAnyType(t))
			base := dPrimitive(t, uq("str"))
			b.AddType(base)
			b.AddElement(tt.build(t, base))
			if _, err := b.Finalize(); err != nil {
				t.Fatalf("Finalize = %v, want success", err)
			}
		})
	}
}
