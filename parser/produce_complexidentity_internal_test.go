package parser

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// complexTypeIdentity is unexported and never escapes the producer, so these
// tests are package-internal. What they pin is the partition itself: which
// xsd.ElementScopeParent / xsd.AttributeScopeParent variant each of the four arms
// emits, and which xsd constructor each drives. The zero-value tests the struct
// shape needed are gone with the zero value: the sum has no arm that carries
// neither a name nor an owner, so the state the panics guarded is
// unrepresentable (#505).

func TestComplexTypeIdentityScopeParentArms(t *testing.T) {
	name := xsd.QName{Space: "urn:po", Local: "T"}
	owner := xsd.NewComponentID()
	for _, tc := range []struct {
		what string
		id   complexTypeIdentity
		want xsd.ElementScopeParent
	}{
		{"named", namedComplexType{name: name}, xsd.ComplexTypeScopeParent{Name: name}},
		{"element-owned", elementOwnedComplexType{owner: owner}, xsd.AnonymousComplexTypeScopeParent{Owner: owner}},
		// The redefining arm is NAMED, so its locals report a by-name parent.
		// Deriving anonymity from owner-presence would emit the anonymous arm
		// here and mis-scope every local element a redefinition declares.
		{"redefining", redefiningComplexType{name: name, owner: owner}, xsd.ComplexTypeScopeParent{Name: name}},
		{"redefine original", redefineOriginalComplexType{owner: owner}, xsd.AnonymousComplexTypeScopeParent{Owner: owner}},
	} {
		if got := scopeParentOf(tc.id); got != tc.want {
			t.Errorf("%s identity scopeParentOf() = %#v, want %#v", tc.what, got, tc.want)
		}
	}
}

// TestComplexTypeIdentityAttributeScopeParentArms is the §3.2.1 sc_a twin of the
// test above. The two functions emit DIFFERENT sum types for the same identity —
// sc_a's alternation is CTD | AGD where §3.3.1 sc_e's is CTD | MGD — so a
// producer that returned the element-side variant here would not compile, and one
// that dropped the named/anonymous split would mis-scope every attribute of an
// anonymous <complexType>.
func TestComplexTypeIdentityAttributeScopeParentArms(t *testing.T) {
	name := xsd.QName{Space: "urn:po", Local: "T"}
	owner := xsd.NewComponentID()
	for _, tc := range []struct {
		what string
		id   complexTypeIdentity
		want xsd.AttributeScopeParent
	}{
		{"named", namedComplexType{name: name}, xsd.AttributeComplexTypeScopeParent{Name: name}},
		{"element-owned", elementOwnedComplexType{owner: owner}, xsd.AttributeAnonymousComplexTypeScopeParent{Owner: owner}},
		{"redefining", redefiningComplexType{name: name, owner: owner}, xsd.AttributeComplexTypeScopeParent{Name: name}},
		{"redefine original", redefineOriginalComplexType{owner: owner}, xsd.AttributeAnonymousComplexTypeScopeParent{Owner: owner}},
	} {
		if got := attributeScopeParentOf(tc.id); got != tc.want {
			t.Errorf("%s identity attributeScopeParentOf() = %#v, want %#v", tc.what, got, tc.want)
		}
	}
}

func TestComplexTypeIdentityNewComplexTypeArms(t *testing.T) {
	p := &producer{}
	name := xsd.QName{Space: "urn:po", Local: "T"}
	anyTypeRef := xsd.TypeDefinitionRef{Name: anyTypeName}

	named, err := p.newComplexType(namedComplexType{name: name}, xsderr.Loc{}, anyTypeRef, nil,
		xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("named newComplexType: %v", err)
	}
	if named.Name() != name {
		t.Errorf("named arm built {name} = %s, want %s", named.Name(), name)
	}
	if _, ok := named.Context(); ok {
		t.Error("named arm built a {context}, which §3.4.1's XOR makes absent when {name} is present")
	}

	owner := xsd.NewComponentID()
	anon, err := p.newComplexType(elementOwnedComplexType{owner: owner}, xsderr.Loc{}, anyTypeRef, nil,
		xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("element-owned newComplexType: %v", err)
	}
	if anon.Name() != (xsd.QName{}) {
		t.Errorf("element-owned arm built {name} = %s, want the absent QName", anon.Name())
	}
	context, ok := anon.Context()
	if !ok {
		t.Fatal("element-owned arm built no {context}, which §3.4.1 makes Required when {name} is absent")
	}
	if _, isED := context.(xsd.ElementDeclarationContext); !isED {
		t.Fatalf("{context} = %T, want an ElementDeclarationContext (§3.4.2.1 dcl.ctd.common)", context)
	}
	// == and never reflect.DeepEqual, which is identity-blind on a ComponentID.
	if context.ID() != owner {
		t.Error("element-owned arm's {context} is not the owner identity it was built with")
	}
}

// TestComplexTypeIdentityRedefineArms pins the src-expredef pairing at the
// constructor seam: the ORIGINAL arm builds an anonymous type whose {context} is
// a ComplexTypeDefinitionContext (never the ElementDeclarationContext the inline
// arm builds), and the REDEFINING arm builds a named type that OWNS that
// original as its {base type definition}, with the two identities agreeing.
func TestComplexTypeIdentityRedefineArms(t *testing.T) {
	p := &producer{}
	name := xsd.QName{Space: "urn:po", Local: "T"}
	owner := xsd.NewComponentID()

	original, err := p.newComplexType(redefineOriginalComplexType{owner: owner}, xsderr.Loc{}, xsd.TypeDefinitionRef{Name: anyTypeName}, nil,
		xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("redefine-original newComplexType: %v", err)
	}
	if original.Name() != (xsd.QName{}) {
		t.Errorf("redefine original built {name} = %s, want the absent QName (src-expredef clause 1.1)", original.Name())
	}
	context, ok := original.Context()
	if !ok {
		t.Fatal("redefine original built no {context}, which src-expredef clause 1.1 makes the redefining component")
	}
	if _, isCTD := context.(xsd.ComplexTypeDefinitionContext); !isCTD {
		t.Fatalf("{context} = %T, want a ComplexTypeDefinitionContext (src-expredef clause 1.1)", context)
	}
	if context.ID() != owner {
		t.Error("redefine original's {context} is not the owner identity it was built with")
	}

	redefining, err := p.newComplexType(redefiningComplexType{name: name, owner: owner}, xsderr.Loc{}, xsd.InlineTypeDefinition{Definition: original}, nil,
		xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("redefining newComplexType: %v", err)
	}
	if redefining.Name() != name {
		t.Errorf("redefining arm built {name} = %s, want %s (src-expredef clause 1.2)", redefining.Name(), name)
	}
	inline, owns := redefining.Base().(xsd.InlineTypeDefinition)
	if !owns {
		t.Fatalf("redefining arm's {base type definition} = %T, want the InlineTypeDefinition holding the clause-1.1 original", redefining.Base())
	}
	if inline.Definition.Name() != (xsd.QName{}) {
		t.Error("redefining arm's owned base is named, but src-expredef clause 1.1 makes its {name} absent")
	}
}

// TestComplexTypeIdentityRedefiningArmWithoutOriginal pins the unreachable
// mismatch: a redefining identity handed a by-NAME base has no clause-1.1
// original to own, and is refused as a plain producer fault rather than built as
// a self-derivation. src-redefine clause 5 (checkRedefinedComplexType) is what
// keeps it unreachable.
func TestComplexTypeIdentityRedefiningArmWithoutOriginal(t *testing.T) {
	p := &producer{}
	name := xsd.QName{Space: "urn:po", Local: "T"}
	_, err := p.newComplexType(redefiningComplexType{name: name, owner: xsd.NewComponentID()}, xsderr.Loc{}, xsd.TypeDefinitionRef{Name: name}, nil,
		xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err == nil {
		t.Fatal("a redefining identity with a by-name base built a component, want the producer fault")
	}
}
