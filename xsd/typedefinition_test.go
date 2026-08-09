package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// anonSimpleType builds an anonymous (zero-{name}) simple type, the only thing
// xsd.InlineTypeDefinition legally wraps.
func anonSimpleType(t *testing.T) *xsd.SimpleType {
	t.Helper()
	st, err := newCheckedSimpleType(xsderr.Loc{}, xsd.QName{}, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType: %v", err)
	}
	return st
}

// namedSimpleType builds a NAMED simple type, which InlineTypeDefinition must
// refuse to wrap: a named type is reachable by name and so is always the
// TypeDefinitionRef arm.
func namedSimpleType(t *testing.T) *xsd.SimpleType {
	t.Helper()
	st, err := newCheckedSimpleType(xsderr.Loc{}, xsd.QName{Local: "T"}, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType: %v", err)
	}
	return st
}

// TestTypeDefinitionOrRefInvariants pins the three encodings of a {type
// definition} slot that NewElementDeclaration and NewAttributeDeclaration reject
// (see TypeDefinitionOrRef), and the three they accept. Both constructors share
// one checker, so both are driven from one table.
func TestTypeDefinitionOrRefInvariants(t *testing.T) {
	cases := []struct {
		name    string
		ref     func(t *testing.T) xsd.TypeDefinitionOrRef
		wantErr bool
	}{
		{"absent slot", func(*testing.T) xsd.TypeDefinitionOrRef { return nil }, false},
		{"named reference", func(*testing.T) xsd.TypeDefinitionOrRef {
			return xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}
		}, false},
		{"inline anonymous type", func(t *testing.T) xsd.TypeDefinitionOrRef {
			return xsd.InlineTypeDefinition{Definition: anonSimpleType(t)}
		}, false},
		{"zero-named reference", func(*testing.T) xsd.TypeDefinitionOrRef {
			return xsd.TypeDefinitionRef{}
		}, true},
		{"empty inline definition", func(*testing.T) xsd.TypeDefinitionOrRef {
			return xsd.InlineTypeDefinition{}
		}, true},
		{"inline definition wrapping a NAMED type", func(t *testing.T) xsd.TypeDefinitionOrRef {
			return xsd.InlineTypeDefinition{Definition: namedSimpleType(t)}
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, elemErr := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"}, tc.ref(t), nil,
				xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil, nil)
			_, attrErr := xsd.NewAttributeDeclaration(xsderr.Loc{}, xsd.QName{Local: "a"}, tc.ref(t),
				xsd.NewAttributeGlobalScope(), nil, false, nil)
			// A slice, not a map: which constructor's failure is reported must not
			// depend on iteration order (STYLE D1/D2).
			for _, got := range []struct {
				who string
				err error
			}{{"NewElementDeclaration", elemErr}, {"NewAttributeDeclaration", attrErr}} {
				who, err := got.who, got.err
				if !tc.wantErr {
					if err != nil {
						t.Fatalf("%s unexpected error: %v", who, err)
					}
					continue
				}
				if err == nil {
					t.Fatalf("%s succeeded, want a component-invariant rejection", who)
				}
				assertRule(t, err, "component-invariant")
			}
		})
	}
}

// TestTypeDefinitionRoundTrip pins that both declarations hand back the very arm
// they were built with, so a consumer can switch it without a lookup.
func TestTypeDefinitionRoundTrip(t *testing.T) {
	st := anonSimpleType(t)
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"},
		xsd.InlineTypeDefinition{Definition: st}, nil, xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	inline, ok := e.TypeDefinition().(xsd.InlineTypeDefinition)
	if !ok {
		t.Fatalf("ElementDeclaration.TypeDefinition() = %#v, want an InlineTypeDefinition", e.TypeDefinition())
	}
	if inline.Definition != xsd.TypeDefinition(st) {
		t.Error("the inline {type definition} is not the very component the constructor was given")
	}

	ref := xsd.TypeDefinitionRef{Name: xsd.QName{Space: "urn:t", Local: "T"}}
	a, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, xsd.QName{Local: "a"}, ref, adLocalScope(t), nil, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeDeclaration: %v", err)
	}
	if got := a.TypeDefinition(); got != xsd.TypeDefinitionOrRef(ref) {
		t.Errorf("AttributeDeclaration.TypeDefinition() = %#v, want %#v", got, ref)
	}
}

// TestInlineTypeDefinitionNeedsNoSymbolTableEntry pins the decision recorded in
// InlineTypeDefinition's doc: the anonymous type is reachable ONLY through the
// owning declaration, never registered in {type definitions}, and finalize
// resolves the declaration without it being there.
func TestInlineTypeDefinitionNeedsNoSymbolTableEntry(t *testing.T) {
	st := anonSimpleType(t)
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"},
		xsd.InlineTypeDefinition{Definition: st}, nil, xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	b := xsd.NewSchemaBuilder()
	b.AddElement(e)
	s, err := b.Finalize()
	if err != nil {
		t.Fatalf("Finalize(inline anonymous {type definition}): %v", err)
	}
	if _, ok := s.Type(xsd.QName{}); ok {
		t.Error("the anonymous inline type is reachable by the zero QName in {type definitions}")
	}
}
