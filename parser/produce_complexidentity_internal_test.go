package parser

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// complexTypeIdentity is unexported and never escapes the producer, so these
// tests are package-internal. What they pin is the partition itself: which
// xsd.ElementScopeParent / xsd.AttributeScopeParent variant each arm emits, and
// that the ZERO identity — which neither entry point can produce — is an
// assertion failure rather than a silently empty arm. An empty arm would travel
// down the content tree and surface as an e-props-correct / a-props-correct
// verdict against an innocent schema.

func TestComplexTypeIdentityScopeParentArms(t *testing.T) {
	name := xsd.QName{Space: "urn:po", Local: "T"}
	if got := namedComplexType(name).scopeParent(); got != (xsd.ComplexTypeScopeParent{Name: name}) {
		t.Errorf("named identity scopeParent() = %#v, want a ComplexTypeScopeParent naming %s", got, name)
	}
	owner := xsd.NewComponentID()
	if got := anonymousComplexType(owner).scopeParent(); got != (xsd.AnonymousComplexTypeScopeParent{Owner: owner}) {
		t.Error("anonymous identity scopeParent() does not carry the owner identity it was built with")
	}
}

// TestComplexTypeIdentityAttributeScopeParentArms is the §3.2.1 sc_a twin of the
// test above. The two methods emit DIFFERENT sum types for the same identity —
// sc_a's alternation is CTD | AGD where §3.3.1 sc_e's is CTD | MGD — so a
// producer that returned the element-side variant here would not compile, and one
// that dropped the named/anonymous split would mis-scope every attribute of an
// inline anonymous <complexType>.
func TestComplexTypeIdentityAttributeScopeParentArms(t *testing.T) {
	name := xsd.QName{Space: "urn:po", Local: "T"}
	if got := namedComplexType(name).attributeScopeParent(); got != (xsd.AttributeComplexTypeScopeParent{Name: name}) {
		t.Errorf("named identity attributeScopeParent() = %#v, want an AttributeComplexTypeScopeParent naming %s", got, name)
	}
	owner := xsd.NewComponentID()
	if got := anonymousComplexType(owner).attributeScopeParent(); got != (xsd.AttributeAnonymousComplexTypeScopeParent{Owner: owner}) {
		t.Error("anonymous identity attributeScopeParent() does not carry the owner identity it was built with")
	}
}

func TestComplexTypeIdentityNewComplexTypeArms(t *testing.T) {
	name := xsd.QName{Space: "urn:po", Local: "T"}
	named, err := namedComplexType(name).newComplexType(xsderr.Loc{}, anyTypeName, nil,
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
	anon, err := anonymousComplexType(owner).newComplexType(xsderr.Loc{}, anyTypeName, nil,
		xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("anonymous newComplexType: %v", err)
	}
	if anon.Name() != (xsd.QName{}) {
		t.Errorf("anonymous arm built {name} = %s, want the absent QName", anon.Name())
	}
	context, ok := anon.Context()
	if !ok {
		t.Fatal("anonymous arm built no {context}, which §3.4.1 makes Required when {name} is absent")
	}
	if _, isED := context.(xsd.ElementDeclarationContext); !isED {
		t.Fatalf("{context} = %T, want an ElementDeclarationContext (§3.4.2.1 dcl.ctd.common)", context)
	}
	// == and never reflect.DeepEqual, which is identity-blind on a ComponentID.
	if context.ID() != owner {
		t.Error("anonymous arm's {context} is not the owner identity it was built with")
	}
}

func TestComplexTypeIdentityZeroValuePanics(t *testing.T) {
	t.Run("scopeParent", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("scopeParent() on the zero identity returned normally, want the assertion panic")
			}
		}()
		_ = complexTypeIdentity{}.scopeParent()
	})
	t.Run("attributeScopeParent", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("attributeScopeParent() on the zero identity returned normally, want the assertion panic")
			}
		}()
		_ = complexTypeIdentity{}.attributeScopeParent()
	})
	t.Run("newComplexType", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("newComplexType() on the zero identity returned normally, want the assertion panic")
			}
		}()
		_, _ = complexTypeIdentity{}.newComplexType(xsderr.Loc{}, anyTypeName, nil,
			xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	})
}
