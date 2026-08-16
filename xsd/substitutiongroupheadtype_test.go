package xsd

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests are package-internal because the owner-of-owner chain they pin is
// a state Finalize REJECTS, so reading ResolvedType's answer on one requires
// assembling a *Schema by hand rather than through the builder — which is also
// the shape the depth-1 rule exists for, a read-time accessor reached on a
// graph no resolution pass has vetted.
//
// They share substitutiongroup_test.go's builders (sq, sgElement, sgRef, sgType)
// and add only the one shape those cannot express: a declaration that OWNS an
// anonymous complex type (STYLE T4).

// sghOwner builds a top-level element declaration owning the anonymous complex
// type of an inline <complexType> child (§3.3.2.1 dcl.elt.common clause 1),
// returning the declaration and the identity its {context} carries.
func sghOwner(t *testing.T, name QName, affiliations ...QName) (ElementDeclaration, ComponentID) {
	t.Helper()
	id := NewComponentID()
	ct, err := NewAnonymousComplexType(xsderr.Loc{}, ElementDeclarationContext{Component: id},
		QName{}, nil, DerivationRestriction, false, nil, nil, nil, EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewAnonymousComplexType(%s): %v", name, err)
	}
	e, err := NewElementDeclarationOwningTypes(xsderr.Loc{}, id, name,
		InlineTypeDefinition{Definition: ct}, nil, NewGlobalScope(), nil,
		false, nil, affiliations, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclarationOwningTypes(%s): %v", name, err)
	}
	return e, id
}

// sghSchema assembles a *Schema from element declarations WITHOUT running the
// resolution pass, so a state Phase A rejects can still be read. Only the
// element slice and its index are populated; these tests read nothing else.
func sghSchema(decls ...ElementDeclaration) *Schema {
	index := make(map[QName]ElementDeclaration, len(decls))
	for _, d := range decls {
		index[d.Name()] = d
	}
	return &Schema{elements: decls, elementIndex: index, valueSpace: undecidedValueSpace{}}
}

// TestTypeOfFollowsSubstitutionGroupHeadOneHop pins the DEPTH-1 read: the arm
// names the head that OWNS the anonymous type, so following Head once lands on
// the component itself. Identity is asserted on the {context} ComponentID with
// == — never reflect.DeepEqual, which is identity-blind for a ComponentID and
// would pass on a WRONG context (see ComponentID).
func TestTypeOfFollowsSubstitutionGroupHeadOneHop(t *testing.T) {
	owner, id := sghOwner(t, sq("head"))
	member := sgElement(t, sq("member"), SubstitutionGroupHeadTypeRef{Head: sq("head")}, nil, sq("head"))
	s := sghSchema(owner, member)

	got, ok := s.ResolvedType(member.TypeDefinition())
	if !ok {
		t.Fatalf("ResolvedType declined the head-inherited arm, want the head's own anonymous type")
	}
	ct, isComplex := got.(ComplexType)
	if !isComplex {
		t.Fatalf("ResolvedType = %T, want the head's anonymous ComplexType", got)
	}
	context, present := ct.Context()
	if !present {
		t.Fatalf("the inherited type has an absent {context}, want the owning head's identity")
	}
	if context.ID() != id {
		t.Fatalf("the inherited type's {context} names a different component than the head that owns it")
	}
}

// TestTypeOfDeclinesOwnerOfOwnerChain pins that the depth-1 claim is ENFORCED
// rather than assumed: a head whose own {type definition} is itself inherited
// answers not-ok instead of recursing. The producer's terminal-head walk never
// mints such a chain and Phase A rejects one outright, but ResolvedType is reachable
// on a programmatically built schema that has seen neither.
func TestTypeOfDeclinesOwnerOfOwnerChain(t *testing.T) {
	owner, _ := sghOwner(t, sq("owner"))
	middle := sgElement(t, sq("middle"), SubstitutionGroupHeadTypeRef{Head: sq("owner")}, nil, sq("owner"))
	member := sgElement(t, sq("member"), SubstitutionGroupHeadTypeRef{Head: sq("middle")}, nil, sq("middle"))
	s := sghSchema(owner, middle, member)

	if _, ok := s.ResolvedType(member.TypeDefinition()); ok {
		t.Fatalf("ResolvedType followed an owner-of-owner chain, want not-ok")
	}
	// The control: the same schema's middle link, one legal hop, still resolves.
	if _, ok := s.ResolvedType(middle.TypeDefinition()); !ok {
		t.Fatalf("ResolvedType declined a single legal hop, so the case above proves nothing")
	}
}

// TestTypeOfDeclinesAbsentSubstitutionGroupHead pins the §5.3 (Missing
// Sub-components) shape: a head naming no declaration is an ·absent· member, a
// VALID schema, so there is simply no type to read — not-ok, never a panic and
// never a fabricated component.
func TestTypeOfDeclinesAbsentSubstitutionGroupHead(t *testing.T) {
	member := sgElement(t, sq("member"), SubstitutionGroupHeadTypeRef{Head: sq("nosuchhead")}, nil, sq("nosuchhead"))
	if _, ok := sghSchema(member).ResolvedType(member.TypeDefinition()); ok {
		t.Fatalf("ResolvedType resolved an ·absent· head, want not-ok")
	}
}

// TestResolveRejectsOwnerOfOwnerHeadTypeChain is the Phase A invariant that
// makes ResolvedType's not-ok branch unreachable for any schema that survived
// finalize (STYLE P3): the arm must name the OWNER, and a head that inherits its
// own type is not one. It is a representation invariant, so it is charged to
// xsderr.RuleComponentInvariant and not to a spec rule.
func TestResolveRejectsOwnerOfOwnerHeadTypeChain(t *testing.T) {
	owner, _ := sghOwner(t, sq("owner"))
	middle := sgElement(t, sq("middle"), SubstitutionGroupHeadTypeRef{Head: sq("owner")}, nil, sq("owner"))
	member := sgElement(t, sq("member"), SubstitutionGroupHeadTypeRef{Head: sq("middle")}, nil, sq("middle"))
	err := cvsFinalize(t, func(b *SchemaBuilder) {
		b.AddElement(owner)
		b.AddElement(middle)
		b.AddElement(member)
	})
	expectRule(t, err, xsderr.RuleComponentInvariant)
}

// TestResolveAllowsAbsentSubstitutionGroupHeadType pins that Phase A does NOT
// charge src-resolve clause 1.3 when the arm names nothing. That head name
// reached the component through {substitution group affiliations}, the one
// reference slot §5.3 exempts — W3C saxonData/Missing missing002 is a VALID
// schema whose substitutionGroup names nothing — so rejecting the {type
// definition} it induced would reject the very schema Phase A allows.
func TestResolveAllowsAbsentSubstitutionGroupHeadType(t *testing.T) {
	member := sgElement(t, sq("member"), SubstitutionGroupHeadTypeRef{Head: sq("nosuchhead")}, nil, sq("nosuchhead"))
	if err := cvsFinalize(t, func(b *SchemaBuilder) { b.AddElement(member) }); err != nil {
		t.Fatalf("an ·absent· substitution group head was rejected (§5.3): %v", err)
	}
}

// TestEPropsCorrectClause4AcceptsSharedAnonymousHeadType is the companion fix
// #342 could not land without: two declarations legitimately share ONE anonymous
// complex type, and sameTypeDefinition reports two anonymous types as different
// (§3.4.6.5's no-identity licence, correct in general), so without the
// construction-identity shortcut clause 4 walks the base chain and FALSE-REJECTS
// a schema that violates nothing.
//
// Both edges are covered: the DIRECT one (member affiliated to the owning head)
// and the CHAIN one (member affiliated to an intermediate head that inherits the
// same type), since the shortcut compares {context} identities and so is blind
// to which arm each side arrived through.
func TestEPropsCorrectClause4AcceptsSharedAnonymousHeadType(t *testing.T) {
	t.Run("direct", func(t *testing.T) {
		owner, _ := sghOwner(t, sq("head"))
		member := sgElement(t, sq("member"), SubstitutionGroupHeadTypeRef{Head: sq("head")}, nil, sq("head"))
		if err := cvsFinalize(t, func(b *SchemaBuilder) {
			b.AddElement(owner)
			b.AddElement(member)
		}); err != nil {
			t.Fatalf("a member sharing its head's anonymous type was rejected: %v", err)
		}
	})
	t.Run("through an intermediate head", func(t *testing.T) {
		owner, _ := sghOwner(t, sq("owner"))
		middle := sgElement(t, sq("middle"), SubstitutionGroupHeadTypeRef{Head: sq("owner")}, nil, sq("owner"))
		member := sgElement(t, sq("member"), SubstitutionGroupHeadTypeRef{Head: sq("owner")}, nil, sq("middle"))
		if err := cvsFinalize(t, func(b *SchemaBuilder) {
			b.AddElement(owner)
			b.AddElement(middle)
			b.AddElement(member)
		}); err != nil {
			t.Fatalf("a member sharing the terminal head's anonymous type was rejected: %v", err)
		}
	})
}

// TestEPropsCorrectClause4RejectsNonFirstHeadAnonymousType is the residual #342
// deliberately left in place, pinned so a later widening is a choice rather than
// an accident. With substitutionGroup="h1 h2" clause 3 reads h1 ALONE, so the
// member carries h1's named type while h2 owns an anonymous one — two genuinely
// different components. The construction-identity shortcut must not admit that
// edge; the ordinary anonymous rule rejects it, and that is correct.
func TestEPropsCorrectClause4RejectsNonFirstHeadAnonymousType(t *testing.T) {
	owner, _ := sghOwner(t, sq("h2"))
	err := cvsFinalize(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("H"), QName{}, DerivationRestriction))
		b.AddElement(cvsElement(t, sq("h1"), sgRef(sq("H")), nil))
		b.AddElement(owner)
		b.AddElement(cvsElement(t, sq("member"), sgRef(sq("H")), nil, sq("h1"), sq("h2")))
	})
	expectRule(t, err, ruleEPropsCorrect)
}

// TestSubstitutionGroupHeadTypeRefSlotLegality pins the arm × slot table
// checkTypeDefinitionOrRef owns. §3.3.2.1 dcl.elt.common clause 3 has no analog
// in §3.2.2.2 (three tiers, no substitution groups on attributes) or in §3.4.2's
// base=, so the arm is legal in the ELEMENT slot alone.
//
// It calls the checker directly rather than going through the constructors
// because the base slot has no exported entry point that takes a
// TypeDefinitionOrRef — which is exactly why the legality table must be total
// here rather than spelled at whichever constructors happen to exist today.
func TestSubstitutionGroupHeadTypeRefSlotLegality(t *testing.T) {
	present := SubstitutionGroupHeadTypeRef{Head: sq("head")}
	for _, tc := range []struct {
		name    string
		slot    typeDefinitionSlot
		ref     TypeDefinitionOrRef
		wantErr bool
	}{
		{"element slot admits it", elementTypeSlot, present, false},
		{"attribute slot rejects it", attributeTypeSlot, present, true},
		{"base type slot rejects it", baseTypeSlot, present, true},
		{"element slot rejects an absent head", elementTypeSlot, SubstitutionGroupHeadTypeRef{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkTypeDefinitionOrRef(xsderr.Loc{}, tc.ref, tc.slot, "component c")
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("unexpected rejection: %v", err)
				}
				return
			}
			expectRule(t, err, xsderr.RuleComponentInvariant)
		})
	}
}

// TestAttributeDeclarationRejectsSubstitutionGroupHeadTypeRef is the slot table
// reached the way a caller reaches it, through the one constructor outside the
// element path that takes the sum: the rejection is not merely available, it is
// wired up.
func TestAttributeDeclarationRejectsSubstitutionGroupHeadTypeRef(t *testing.T) {
	_, err := NewAttributeDeclaration(xsderr.Loc{}, sq("a"), SubstitutionGroupHeadTypeRef{Head: sq("head")},
		NewAttributeGlobalScope(), nil, false, nil)
	expectRule(t, err, xsderr.RuleComponentInvariant)
}
