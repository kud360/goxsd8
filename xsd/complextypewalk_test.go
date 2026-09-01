package xsd

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests pin the REACH the three read-only finalize passes gained in #438:
// cos-nonambig (particleattribution.go), cos-element-consistent
// (elementconsistent.go) and the §3.4.6 derivation constraints
// (complexderivation.go) over a complex type a DECLARATION owns, which is in no
// §3.17.2 {type definitions} slice and so was reached by none of the three
// s.types loops they replaced. What each rule DECIDES is pinned by those files'
// own tests; what is asserted here is only that a type at each owning slot gets
// the same verdict a named one already got.
//
// The component builders come from complexderivation_test.go,
// elementconsistent_test.go and particleattribution_test.go — one set of
// helpers, not four (STYLE T4).

// cwFinalize assembles a schema carrying xs:anyType, the two named complex types
// the fixtures' declarations refer to and the simple type their attributes name,
// plus whatever build adds, then finalizes it.
func cwFinalize(t *testing.T, build func(*SchemaBuilder)) error {
	t.Helper()
	b := NewSchemaBuilder()
	b.AddType(dAnyType(t))
	b.AddType(uNamedType(t, uq("T")))
	b.AddType(uNamedType(t, uq("U")))
	b.AddType(dPrimitive(t, uq("str")))
	build(b)
	_, err := b.Finalize()
	return err
}

// cwDuplicateAttributes builds an ANONYMOUS complex type violating
// ct-props-correct (§3.4.6.1) clause 4: two {attribute uses} sharing an
// {attribute declaration} expanded name. It is the cheapest §3.4.6 violation to
// seat in a slot — clause 4 is charged before either derivation constraint, so
// the fixture's base needs no derivation story of its own.
func cwDuplicateAttributes(t *testing.T) ComplexType {
	t.Helper()
	return dType(t, QName{}, QName{}, EmptyContent{},
		[]AttributeUse{dAttr(t, uq("a"), uq("str")), dAttr(t, uq("a"), uq("str"))}, nil)
}

// cwAmbiguous builds an ANONYMOUS complex type whose content model violates
// cos-nonambig: two same-named element particles of ONE named type in a
// <choice>, both live in the start state. Naming one type keeps
// cos-element-consistent — the pass that runs next — satisfied, so a rejection
// can only be the one the fixture is about.
func cwAmbiguous(t *testing.T) ComplexType {
	t.Helper()
	g := uGroup(t, CompositorChoice,
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
	)
	return dType(t, QName{}, QName{}, dElementContent(t, false, g), nil, nil)
}

// cwInconsistent builds an ANONYMOUS complex type whose content model violates
// cos-element-consistent clauses 2-3: two same-named element declarations naming
// DIFFERENT top-level types. They sit in a <sequence>, so cos-nonambig — the
// pass that runs first — is satisfied and the rejection can only be this one.
func cwInconsistent(t *testing.T) ComplexType {
	t.Helper()
	g := uGroup(t, CompositorSequence,
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
		uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("U"))}),
	)
	return dType(t, QName{}, QName{}, dElementContent(t, false, g), nil, nil)
}

// cwTableElement builds a TOP-LEVEL element declaration named name and typed
// uq("T"), whose {type table} seats owned in one slot: the sole {alternatives}
// member when inAlternatives, otherwise the {default type definition}. The other
// slot names uq("T") by name, which is the declaration's own {type definition},
// so e-props-correct clause 7 is satisfied by every entry and cannot pre-empt
// the verdict a fixture is about.
//
// The element declaration is minted with the owned type's OWN {context}
// identity, as §3.4.2.1 dcl.ctd.common requires of a type reached through an
// <alternative>: the context walks past the <alternative> to the enclosing
// Element Declaration.
func cwTableElement(t *testing.T, name QName, owned ComplexType, inAlternatives bool) ElementDeclaration {
	t.Helper()
	test := NewXPathExpression("true()", nil, nil, nil)
	alternatives := []TypeAlternative{iTypeAlternative(t, &test, TypeDefinitionRef{Name: uq("T")})}
	dflt := iTypeAlternative(t, nil, InlineTypeDefinition{Definition: owned})
	if inAlternatives {
		alternatives = []TypeAlternative{iTypeAlternative(t, &test, InlineTypeDefinition{Definition: owned})}
		dflt = iTypeAlternative(t, nil, TypeDefinitionRef{Name: uq("T")})
	}
	tt, err := NewTypeTable(xsderr.Loc{}, alternatives, dflt)
	if err != nil {
		t.Fatalf("NewTypeTable: %v", err)
	}
	context, ok := owned.Context()
	if !ok {
		t.Fatal("cwTableElement: the owned type is NAMED, so no declaration owns it")
	}
	e, err := NewElementDeclarationOwningTypes(xsderr.Loc{}, context.ID(), name,
		TypeDefinitionRef{Name: uq("T")}, &tt, NewGlobalScope(),
		nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclarationOwningTypes(%s): %v", name, err)
	}
	return e
}

// cwUnreferencedGroup builds a top-level Model Group Definition uq("G") whose
// particles hold ONE local element owning ct, and which no <group ref> names.
// It is s.modelGroups' claim to be an independent root: nothing reachable from
// s.types or s.elements reaches ct.
func cwUnreferencedGroup(t *testing.T, ct ComplexType) ModelGroupDefinition {
	t.Helper()
	local := dOwnInline(t, uq("child"), ct, uLocalScope(t))
	d, err := NewModelGroupDefinition(xsderr.Loc{}, uq("G"),
		uGroup(t, CompositorSequence, uOne(t, ResolvedTerm{Term: local})), nil)
	if err != nil {
		t.Fatalf("NewModelGroupDefinition(G): %v", err)
	}
	return d
}

// cwRedefinePair builds src-expredef's clause-1.1/1.2 pairing with the ORIGINAL
// carrying a content model: the NAMED complex type uq("R") restricting an
// ANONYMOUS one it owns through {base type definition}, with g as the original's
// sole particle's {term}.
//
// R's own {content type} is EMPTY, so whatever g holds is reachable through the
// base slot and through nothing else. The original's particle is optional so
// that emptying it is a legal restriction (derivation-ok-restriction clause 5.2,
// an ·emptiable· base particle) and R itself contributes no rejection.
func cwRedefinePair(t *testing.T, g ModelGroup) ComplexType {
	t.Helper()
	id := NewComponentID()
	original, err := NewAnonymousComplexType(xsderr.Loc{}, ComplexTypeDefinitionContext{Component: id},
		QName{}, nil, DerivationRestriction, false, nil, nil, nil,
		ElementContent{Particle: uParticle(t, uOccurs(t, 0, 1), ResolvedTerm{Term: g})}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewAnonymousComplexType (the clause-1.1 original of R): %v", err)
	}
	ct, err := NewComplexTypeOwningBase(xsderr.Loc{}, id, uq("R"), original, nil, DerivationRestriction, false,
		nil, nil, nil, EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexTypeOwningBase(R): %v", err)
	}
	return ct
}

// TestWalkChargesElementOwnedType pins the first owning slot, an <element>'s own
// inline <complexType> (§3.3.2.1 dcl.elt.common clause 1): all three passes
// charge the type it seats.
func TestWalkChargesElementOwnedType(t *testing.T) {
	t.Run("ct-props-correct clause 4", func(t *testing.T) {
		expectRule(t, cwFinalize(t, func(b *SchemaBuilder) {
			b.AddElement(dOwnInline(t, uq("g"), cwDuplicateAttributes(t), NewGlobalScope()))
		}), ruleCTPropsCorrect)
	})
	t.Run("cos-nonambig", func(t *testing.T) {
		expectRule(t, cwFinalize(t, func(b *SchemaBuilder) {
			b.AddElement(dOwnInline(t, uq("g"), cwAmbiguous(t), NewGlobalScope()))
		}), ruleCosNonambig)
	})
	t.Run("cos-element-consistent", func(t *testing.T) {
		expectRule(t, cwFinalize(t, func(b *SchemaBuilder) {
			b.AddElement(dOwnInline(t, uq("g"), cwInconsistent(t), NewGlobalScope()))
		}), ruleCosElementConsistent)
	})
}

// TestWalkChargesAlternativeOwnedType pins the second owning slot, the inline
// <complexType> of an <alternative> (§3.12.2 declare-ta's inline arm). It is a
// slot componentwalk.go's shared descent does not reach at all —
// walkElementDeclaration enters only e.TypeDefinition() — so routing these three
// passes through that walk would have left this shape uncharged.
func TestWalkChargesAlternativeOwnedType(t *testing.T) {
	t.Run("ct-props-correct clause 4", func(t *testing.T) {
		expectRule(t, cwFinalize(t, func(b *SchemaBuilder) {
			b.AddElement(cwTableElement(t, uq("g"), cwDuplicateAttributes(t), true))
		}), ruleCTPropsCorrect)
	})
	t.Run("cos-nonambig", func(t *testing.T) {
		expectRule(t, cwFinalize(t, func(b *SchemaBuilder) {
			b.AddElement(cwTableElement(t, uq("g"), cwAmbiguous(t), true))
		}), ruleCosNonambig)
	})
	t.Run("cos-element-consistent", func(t *testing.T) {
		expectRule(t, cwFinalize(t, func(b *SchemaBuilder) {
			b.AddElement(cwTableElement(t, uq("g"), cwInconsistent(t), true))
		}), ruleCosElementConsistent)
	})
}

// TestWalkChargesDefaultTypeDefinitionOwnedType pins the third owning slot, {type
// table}.{default type definition}.{type definition}. A TRAILING untested
// <alternative> feeds this slot and no other, so an owned type can appear here
// alone (ownedTypeSlots' doc states the same for its other readers).
func TestWalkChargesDefaultTypeDefinitionOwnedType(t *testing.T) {
	expectRule(t, cwFinalize(t, func(b *SchemaBuilder) {
		b.AddElement(cwTableElement(t, uq("g"), cwDuplicateAttributes(t), false))
	}), ruleCTPropsCorrect)
}

// TestWalkChargesUnreferencedModelGroupOwnedType pins s.modelGroups as an
// INDEPENDENT root. §3.7.2 gives <group name="…"> an (all|choice|sequence)
// child that may nest <element> children with inline <complexType>s, and where
// no <group ref> names that definition — as here — those types are reachable
// from neither s.types nor s.elements.
func TestWalkChargesUnreferencedModelGroupOwnedType(t *testing.T) {
	t.Run("cos-nonambig", func(t *testing.T) {
		expectRule(t, cwFinalize(t, func(b *SchemaBuilder) {
			b.AddModelGroup(cwUnreferencedGroup(t, cwAmbiguous(t)))
		}), ruleCosNonambig)
	})
	t.Run("ct-props-correct clause 4", func(t *testing.T) {
		expectRule(t, cwFinalize(t, func(b *SchemaBuilder) {
			b.AddModelGroup(cwUnreferencedGroup(t, cwDuplicateAttributes(t)))
		}), ruleCTPropsCorrect)
	})
	t.Run("cos-element-consistent", func(t *testing.T) {
		expectRule(t, cwFinalize(t, func(b *SchemaBuilder) {
			b.AddModelGroup(cwUnreferencedGroup(t, cwInconsistent(t)))
		}), ruleCosElementConsistent)
	})
}

// TestWalkDescendsIntoAnOwnedType pins that the descent does not stop at the
// first owned type: an inline <complexType> declared INSIDE another inline one,
// under a top-level <element>, is charged too. Without it the widening would be
// a single extra level rather than the transitive walk §3.4.6's chapeau needs.
func TestWalkDescendsIntoAnOwnedType(t *testing.T) {
	expectRule(t, cwFinalize(t, func(b *SchemaBuilder) {
		local := dOwnInline(t, uq("child"), cwDuplicateAttributes(t), uLocalScope(t))
		outer := dType(t, QName{}, QName{}, dElementContent(t, false,
			uGroup(t, CompositorSequence, uOne(t, ResolvedTerm{Term: local}))), nil, nil)
		b.AddElement(dOwnInline(t, uq("g"), outer, NewGlobalScope()))
	}), ruleCTPropsCorrect)
}

// TestWalkChargesAValidOwnedTypeNothing is the control the widening needs most:
// a well-formed anonymous type in every owning slot, at two depths, still
// finalizes. Without it a walk that rejected everything it newly reached would
// pass every test above.
func TestWalkChargesAValidOwnedTypeNothing(t *testing.T) {
	err := cwFinalize(t, func(b *SchemaBuilder) {
		inner := dType(t, QName{}, QName{}, EmptyContent{}, []AttributeUse{dAttr(t, uq("a"), uq("str"))}, nil)
		nested := dType(t, QName{}, QName{}, dElementContent(t, false,
			uGroup(t, CompositorSequence, uOne(t, ResolvedTerm{Term: dOwnInline(t, uq("child"), inner, uLocalScope(t))}))), nil, nil)
		b.AddElement(dOwnInline(t, uq("g"), nested, NewGlobalScope()))
		b.AddElement(cwTableElement(t, uq("h"), dType(t, QName{}, uq("T"), EmptyContent{}, nil, nil), true))
		b.AddModelGroup(cwUnreferencedGroup(t, dType(t, QName{}, QName{}, EmptyContent{}, nil, nil)))
	})
	if err != nil {
		t.Fatalf("a schema whose anonymous types are all well-formed was rejected: %v", err)
	}
}

// TestWalkEntersAnOwnedBaseWithoutChargingIt pins the ONE owning slot this walk
// enters without charging, and both halves of that split.
//
// The src-expredef clause 1.1 ORIGINAL a redefining complex type owns is seated
// by a {base type definition} rather than by a declaration's {type definition},
// and giving it a §3.4.6 verdict is #584's change; the GAP marker on
// checkComplexDerivations names it. The DECLARATIONS inside its content model
// own anonymous types by the ordinary §3.3.2.1 route, so those are charged, and
// entering the slot is what reaches them: a redefined original is in no
// {type definitions} slice either.
func TestWalkEntersAnOwnedBaseWithoutChargingIt(t *testing.T) {
	t.Run("a type owned by a declaration inside the original is charged", func(t *testing.T) {
		local := dOwnInline(t, uq("child"), cwDuplicateAttributes(t), uLocalScope(t))
		expectRule(t, cwFinalize(t, func(b *SchemaBuilder) {
			b.AddType(cwRedefinePair(t, uGroup(t, CompositorSequence, uOne(t, ResolvedTerm{Term: local}))))
		}), ruleCTPropsCorrect)
	})
	t.Run("the original's own content model is charged nothing", func(t *testing.T) {
		// A cos-nonambig violation of the ORIGINAL itself: two same-named element
		// particles in one <choice>. It is charged to no complex type, because the
		// original is the one component the descent passes through without
		// charging. An attribute-side fixture cannot show this — §3.4.2.4 clause 3
		// unions the base's {attribute uses} into R's, so a duplicate seated in the
		// original is charged at R in its own right.
		err := cwFinalize(t, func(b *SchemaBuilder) {
			b.AddType(cwRedefinePair(t, uGroup(t, CompositorChoice,
				uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
				uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("T"))}),
			)))
		})
		if err != nil {
			t.Fatalf("the owned base was charged a verdict of its own, which is #584's to charge: %v", err)
		}
	})
}

// cwAnonymousExtension builds an ANONYMOUS complex type EXTENDING base, the one
// shape the helpers here cannot otherwise reach: dType hard-codes restriction.
// It is the cheapest fixture whose rejection is written by complexextension.go
// rather than complexderivation.go.
func cwAnonymousExtension(t *testing.T, base QName) ComplexType {
	t.Helper()
	ct, err := NewAnonymousComplexType(xsderr.Loc{}, ElementDeclarationContext{Component: NewComponentID()},
		base, nil, DerivationExtension, false, nil, nil, nil, EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewAnonymousComplexType: %v", err)
	}
	return ct
}

// TestWalkRejectionNamesTheAnonymousType pins STYLE E1 for the messages this
// widening newly reaches. A type a DECLARATION owns has the zero QName, which
// String()s to the EMPTY string, so a message printing its {name} leaves a hole
// exactly where the reader looks for the construct — "complex type  extends base
// with a mixed {content type}". complexTypeOwner (componentwalk.go) is the one
// renderer that answers for both cases, and every message in
// complexderivation.go and complexextension.go goes through it.
func TestWalkRejectionNamesTheAnonymousType(t *testing.T) {
	t.Run("complexderivation.go, ct-props-correct clause 4", func(t *testing.T) {
		cwExpectNamed(t, cwFinalize(t, func(b *SchemaBuilder) {
			b.AddElement(dOwnInline(t, uq("g"), cwDuplicateAttributes(t), NewGlobalScope()))
		}), ruleCTPropsCorrect)
	})
	t.Run("complexextension.go, cos-ct-extends clause 1.1", func(t *testing.T) {
		cwExpectNamed(t, cwFinalize(t, func(b *SchemaBuilder) {
			b.AddType(xFinal(t, uq("shut"), anyTypeName, []DerivationMethod{DerivationExtension}))
			b.AddElement(dOwnInline(t, uq("g"), cwAnonymousExtension(t, uq("shut")), NewGlobalScope()))
		}), ruleCosCTExtends)
	})
}

// cwExpectNamed asserts err carries rule and describes the type it is charged to
// rather than leaving the hole a zero QName renders as.
func cwExpectNamed(t *testing.T, err error, rule xsderr.Rule) {
	t.Helper()
	expectRule(t, err, rule)
	if !strings.Contains(err.Error(), "anonymous complex type") {
		t.Fatalf("message %q does not name the anonymous type it is charged to, so the reader cannot tell which construct was meant", err)
	}
}
