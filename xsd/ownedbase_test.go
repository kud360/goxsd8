package xsd

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests are package-internal because their subject is what the finalize
// walks do with the InlineTypeDefinition arm of {base type definition} — the arm
// §4.2.4 src-expredef clause 1.1 introduces, where the base is an ANONYMOUS
// component in no by-name index. Two of the walks (derivedOKComplex,
// derivationAdmitsSubstitution) are unexported predicates with no exported
// caller that isolates them, and the folds are observed through the finalized
// property. The component builders come from complexderivation_test.go — one set
// of helpers, not three (STYLE T4).

// oPair builds src-expredef's clause-1.1/1.2 pairing directly: a NAMED complex
// type deriving from an ANONYMOUS one it owns, with ONE minted identity threaded
// into both halves so the original's {context} points back at the redefinition.
// It is what parser/redefine.go assembles from a <redefine> child, built here
// without a producer so the xsd package's own walks are the subject.
func oPair(t *testing.T, name, originalBase QName, method DerivationMethod, ownUses, originalUses []AttributeUse, ownWildcard, originalWildcard *Wildcard) ComplexType {
	t.Helper()
	id := NewComponentID()
	original, err := NewAnonymousComplexType(xsderr.Loc{}, ComplexTypeDefinitionContext{Component: id},
		originalBase, nil, DerivationRestriction, false, originalUses, nil, originalWildcard, EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewAnonymousComplexType (the clause-1.1 original of %s): %v", name, err)
	}
	ct, err := NewComplexTypeOwningBase(xsderr.Loc{}, id, name, original, nil, method, false,
		ownUses, nil, ownWildcard, EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexTypeOwningBase(%s): %v", name, err)
	}
	return ct
}

// TestOwnedBaseContextMustNameTheOwner pins the state
// NewComplexTypeOwningBase exists to make unrepresentable: an owned original
// whose {context} names some OTHER component. It is the redefine-side form of
// PRINCIPLES 16's context-tracking hazard, and no shape check downstream could
// reach it.
func TestOwnedBaseContextMustNameTheOwner(t *testing.T) {
	stranger, err := NewAnonymousComplexType(xsderr.Loc{}, ComplexTypeDefinitionContext{Component: NewComponentID()},
		anyTypeName, nil, DerivationRestriction, false, nil, nil, nil, EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewAnonymousComplexType: %v", err)
	}
	for _, tc := range []struct {
		what string
		id   ComponentID
		base ComplexType
	}{
		{"a {context} naming a different component", NewComponentID(), stranger},
		{"an unminted owner identity", ComponentID{}, stranger},
	} {
		t.Run(tc.what, func(t *testing.T) {
			if _, err := NewComplexTypeOwningBase(xsderr.Loc{}, tc.id, uq("T"), tc.base, nil,
				DerivationRestriction, false, nil, nil, nil, EmptyContent{}, nil, nil, nil); err == nil {
				t.Fatalf("NewComplexTypeOwningBase accepted %s", tc.what)
			}
		})
	}
}

// TestOwnedBaseRejectsElementDeclarationContext pins the other half of the
// {context} check: src-expredef clause 1.1 makes the original's {context} "the
// redefining component", which for a <complexType> redefinition is a Complex Type
// Definition. An ElementDeclarationContext there is §3.4.2.1's case, not this one.
func TestOwnedBaseRejectsElementDeclarationContext(t *testing.T) {
	id := NewComponentID()
	wrongArm, err := NewAnonymousComplexType(xsderr.Loc{}, ElementDeclarationContext{Component: id},
		anyTypeName, nil, DerivationRestriction, false, nil, nil, nil, EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewAnonymousComplexType: %v", err)
	}
	if _, err := NewComplexTypeOwningBase(xsderr.Loc{}, id, uq("T"), wrongArm, nil,
		DerivationRestriction, false, nil, nil, nil, EmptyContent{}, nil, nil, nil); err == nil {
		t.Fatal("NewComplexTypeOwningBase accepted an owned base contexted in an ELEMENT DECLARATION")
	}
}

// TestOwnedBaseRejectsNamedBase pins that the owned arm admits only an anonymous
// component: a NAMED type is reachable by name and so is always the
// TypeDefinitionRef arm (typedefinition.go).
func TestOwnedBaseRejectsNamedBase(t *testing.T) {
	named := dType(t, uq("named"), anyTypeName, EmptyContent{}, nil, nil)
	if _, err := NewComplexTypeOwningBase(xsderr.Loc{}, NewComponentID(), uq("T"), named, nil,
		DerivationRestriction, false, nil, nil, nil, EmptyContent{}, nil, nil, nil); err == nil {
		t.Fatal("NewComplexTypeOwningBase accepted a NAMED owned base")
	}
}

// TestOwnedBaseCycleThroughAnonymousHopIsRejected is the termination case the
// pairing creates: U's base is T, T's base is the anonymous original, and the
// original's base is U — so the chain CLOSES through a hop no by-name walk can
// see. Phase B must descend the InlineTypeDefinition arm; a walk that stopped on
// it would report no cycle here, and the two attribute folds — which do follow
// the hop — would then recurse without bound (#505).
//
// The test would HANG rather than fail if the fix regressed, which is itself the
// signal: the folds run only after Phase B accepts.
func TestOwnedBaseCycleThroughAnonymousHopIsRejected(t *testing.T) {
	err := dFinalize(t, func(b *SchemaBuilder) {
		b.AddType(oPair(t, uq("ct"), uq("u"), DerivationExtension, nil, nil, nil, nil))
		b.AddType(dType(t, uq("u"), uq("ct"), EmptyContent{}, nil, nil))
	})
	expectRule(t, err, ruleCTPropsCorrect)
}

// TestOwnedBaseAcyclicChainAccepted is the control the test above needs: the same
// shape with the cycle removed must FINALIZE, not merely fail to hang. Without it
// a Phase B change that rejected every anonymous hop outright would still pass.
func TestOwnedBaseAcyclicChainAccepted(t *testing.T) {
	if err := dFinalize(t, func(b *SchemaBuilder) {
		b.AddType(oPair(t, uq("ct"), uq("u"), DerivationExtension, nil, nil, nil, nil))
		b.AddType(dType(t, uq("u"), anyTypeName, EmptyContent{}, nil, nil))
	}); err != nil {
		t.Fatalf("Finalize rejected an acyclic chain running through an owned anonymous base: %v", err)
	}
}

// TestOwnedBaseAttributeFolds pins §3.4.2.4 clause 3 and §3.4.2.5 clause 2.2
// across the anonymous hop. Both clauses name the {base type definition}'s own
// property, and this base is in neither fold's position map, so a fold that only
// followed names would read the miss as clause 3.3 / clause 2.2.1.2 and leave the
// redefinition without what its original carries. A missing {attribute use}
// REJECTS an instance that supplies that attribute, so the direction is pass→fail
// (#505), not an under-report.
func TestOwnedBaseAttributeFolds(t *testing.T) {
	wildcard := dWildcard(t)
	s := xSchema(t, func(b *SchemaBuilder) {
		b.AddType(dSimple(t, uq("str"), AnyAtomicType()))
		b.AddType(dType(t, uq("u"), anyTypeName, EmptyContent{}, []AttributeUse{dAttr(t, uq("grand"), uq("str"))}, nil))
		b.AddType(oPair(t, uq("ct"), uq("u"), DerivationExtension,
			[]AttributeUse{dAttr(t, uq("own"), uq("str"))},
			[]AttributeUse{dAttr(t, uq("orig"), uq("str"))},
			nil, &wildcard))
	})
	// own first (clause 1), then the original's own, then what the original in
	// turn inherits from the named type above it — the whole chain, in order.
	if got, want := fUses(t, s, uq("ct")), []string{"own", "orig", "grand"}; !fEqual(got, want) {
		t.Fatalf("{attribute uses} of ct = %v, want %v", got, want)
	}
	def, _ := s.Type(uq("ct"))
	if _, present := def.(ComplexType).AttributeWildcard(); !present {
		t.Fatal("{attribute wildcard} of ct is absent, but §3.4.2.5 clause 2.2 unions in the ·base wildcard· its owned original declares")
	}
}

// dWildcard builds a lax ##any attribute wildcard for the fold test above.
func dWildcard(t *testing.T) Wildcard {
	t.Helper()
	nc, err := NewNamespaceConstraint(xsderr.Loc{}, NamespaceConstraintAny, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewNamespaceConstraint: %v", err)
	}
	w, err := NewWildcard(xsderr.Loc{}, nc, ProcessLax, nil)
	if err != nil {
		t.Fatalf("NewWildcard: %v", err)
	}
	return w
}

// TestDerivedOKComplexThroughOwnedBase pins cos-ct-derived-ok (§3.4.6.5) over the
// anonymous hop, in BOTH directions — a predicate that merely stopped walking
// would answer false to the first case and pass the second by accident.
//
// A false here is what makes an instance validation fail, so terminating on the
// hop is an over-REJECT for every element typed by a redefined complex type, not
// a conservative answer (#505).
func TestDerivedOKComplexThroughOwnedBase(t *testing.T) {
	s := xSchema(t, func(b *SchemaBuilder) {
		b.AddType(oPair(t, uq("ct"), uq("u"), DerivationExtension, nil, nil, nil, nil))
		b.AddType(dType(t, uq("u"), anyTypeName, EmptyContent{}, nil, nil))
		b.AddType(dType(t, uq("unrelated"), anyTypeName, EmptyContent{}, nil, nil))
	})
	ct := mustComplex(t, s, uq("ct"))
	u, _ := s.Type(uq("u"))
	unrelated, _ := s.Type(uq("unrelated"))
	if !s.derivedOKComplex(ct, u, nil) {
		t.Error("ct is NOT reported as derived from u, but its base chain reaches u through the owned anonymous original")
	}
	if s.derivedOKComplex(ct, unrelated, nil) {
		t.Error("ct is reported as derived from an unrelated type, so the walk is answering true without reaching it")
	}
}

// TestDerivationAdmitsSubstitutionThroughOwnedBase is the substitution-group twin
// (§3.3.6.3 cos-equiv-derived-ok-rec's blocking half). The walk terminating on the
// anonymous hop answers TRUE — a fail-OPEN accept, not a conservative refusal.
//
// The discriminating case is a keyword the chain uses ONLY past the hop:
// ct EXTENDS its owned original, which RESTRICTS u, so a head that blocks
// restriction must refuse the substitution. A walk that stopped at the hop would
// have collected extension alone and answered true.
func TestDerivationAdmitsSubstitutionThroughOwnedBase(t *testing.T) {
	for _, tc := range []struct {
		what    string
		blocked []DerivationMethod
		want    bool
	}{
		{"head blocks the restriction step past the anonymous hop", []DerivationMethod{DerivationRestriction}, false},
		{"head blocks nothing", nil, true},
	} {
		t.Run(tc.what, func(t *testing.T) {
			s := xSchema(t, func(b *SchemaBuilder) {
				// ct extends its owned original, which restricts u: the
				// extension step is only visible past the anonymous hop.
				b.AddType(oPair(t, uq("ct"), uq("u"), DerivationExtension, nil, nil, nil, nil))
				b.AddType(dType(t, uq("u"), anyTypeName, EmptyContent{}, nil, nil))
				b.AddElement(oHead(t, uq("head"), uq("u"), tc.blocked))
				b.AddElement(oMember(t, uq("member"), uq("ct"), uq("head")))
			})
			head, _ := s.Element(uq("head"))
			member, _ := s.Element(uq("member"))
			if got := s.derivationAdmitsSubstitution(member, head); got != tc.want {
				t.Fatalf("derivationAdmitsSubstitution = %v, want %v", got, tc.want)
			}
		})
	}
}

// mustComplex reads a named complex type back off a finalized schema.
func mustComplex(t *testing.T, s *Schema, name QName) ComplexType {
	t.Helper()
	def, ok := s.Type(name)
	if !ok {
		t.Fatalf("type %s is not in the finalized schema", name)
	}
	c, ok := def.(ComplexType)
	if !ok {
		t.Fatalf("type %s is a %T, want a ComplexType", name, def)
	}
	return c
}

// oHead builds a global element declaration whose {disallowed substitutions}
// carry blocked — the head of the substitution group the test above walks.
func oHead(t *testing.T, name, typeName QName, blocked []DerivationMethod) ElementDeclaration {
	t.Helper()
	e, err := NewElementDeclaration(xsderr.Loc{}, name, TypeDefinitionRef{Name: typeName}, nil, NewGlobalScope(),
		nil, false, nil, nil, nil, false, blocked, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration(%s): %v", name, err)
	}
	return e
}

// oMember builds a global element declaration affiliated to head.
func oMember(t *testing.T, name, typeName, head QName) ElementDeclaration {
	t.Helper()
	e, err := NewElementDeclaration(xsderr.Loc{}, name, TypeDefinitionRef{Name: typeName}, nil, NewGlobalScope(),
		nil, false, nil, []QName{head}, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration(%s): %v", name, err)
	}
	return e
}
