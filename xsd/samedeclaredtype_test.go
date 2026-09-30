package xsd

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// TestSameDeclaredType pins sameDeclaredType's two arms (cos-ct-derived-ok
// clause 2.1): named types by expanded name, anonymous complex types by the
// identity their {context} carries — and nothing else, so two anonymous types
// of two declarations stay distinct.
func TestSameDeclaredType(t *testing.T) {
	anon := func(id ComponentID) ComplexType {
		t.Helper()
		c, err := NewAnonymousComplexType(xsderr.Loc{}, ElementDeclarationContext{Component: id},
			anyTypeName, nil, DerivationRestriction, false, nil, nil, nil, EmptyContent{}, nil, nil)
		if err != nil {
			t.Fatalf("NewAnonymousComplexType: %v", err)
		}
		return c
	}
	decl, other := NewComponentID(), NewComponentID()
	named := xType(t, uq("N"), anyTypeName, EmptyContent{}, nil, nil)
	tests := []struct {
		name string
		a, b TypeDefinition
		want bool
	}{
		{"one declaration's anonymous type", anon(decl), anon(decl), true},
		{"two declarations' anonymous types", anon(decl), anon(other), false},
		{"one named type", named, named, true},
		{"named and anonymous", named, anon(decl), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sameDeclaredType(tt.a, tt.b); got != tt.want {
				t.Errorf("sameDeclaredType = %v, want %v", got, tt.want)
			}
		})
	}
}
