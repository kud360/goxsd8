package xsd

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests are package-internal: inSubstitutionGroupOf is unexported (STYLE
// T5) machinery the §3.8.6 and §3.10.4 constraints read, so membership is
// asserted directly on a finalized *Schema rather than through the error one of
// those constraints would raise.

const sgns = "urn:sg"

func sq(local string) QName { return QName{Space: sgns, Local: local} }

// sgType builds a complex type with an explicit {base type definition},
// {derivation method} and {prohibited substitutions}. Its {content type} is
// EmptyContent throughout: clause 2.3 reads neither the content model nor the
// attribute uses, so an empty one keeps every derivation in these tests valid.
// An absent name selects NewAnonymousComplexType, whose §3.4.1 tableau makes
// {context} Required; the caller that passes one wraps the result in an
// InlineTypeDefinition on an element declaration, so ElementDeclarationContext
// is the honest arm.
func sgType(t *testing.T, name, base QName, method DerivationMethod, prohibited ...DerivationMethod) ComplexType {
	t.Helper()
	if name.Local == "" {
		ct, err := NewAnonymousComplexType(xsderr.Loc{}, ElementDeclarationContext{Component: NewComponentID()},
			base, nil, method, false, nil, nil, nil, EmptyContent{}, prohibited, nil, nil)
		if err != nil {
			t.Fatalf("NewAnonymousComplexType: %v", err)
		}
		return ct
	}
	ct, err := NewComplexType(xsderr.Loc{}, name, base, nil, method, false,
		nil, nil, nil, EmptyContent{}, prohibited, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType(%s): %v", name, err)
	}
	return ct
}

// sgSimple builds a named atomic simple type restricting base — a ·derivation·
// chain link that carries neither {derivation method} nor {prohibited
// substitutions}, both being Complex Type Definition properties (§3.4.1).
func sgSimple(t *testing.T, name QName, base *SimpleType) *SimpleType {
	t.Helper()
	st, err := NewSimpleType(xsderr.Loc{}, name, base.Variety(), base, nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(%s): %v", name, err)
	}
	return st
}

// sgSimpleContentType builds a complex type extending the simple type base — the
// <simpleContent> shape, and the only way a simple type gets onto the {base type
// definition} chain of a complex one.
func sgSimpleContentType(t *testing.T, name QName, base *SimpleType) ComplexType {
	t.Helper()
	ct, err := NewComplexType(xsderr.Loc{}, name, base.Name(), nil, DerivationExtension, false,
		nil, nil, nil, SimpleContent{SimpleType: base}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType(%s): %v", name, err)
	}
	return ct
}

// sgElement builds a TOP-LEVEL element declaration with the given {type
// definition} slot, {disallowed substitutions} and {substitution group
// affiliations}.
func sgElement(t *testing.T, name QName, typeDef TypeDefinitionOrRef, disallowed []DerivationMethod, affiliations ...QName) ElementDeclaration {
	t.Helper()
	e, err := NewElementDeclaration(xsderr.Loc{}, name, typeDef, nil, NewGlobalScope(), nil, false, nil,
		affiliations, nil, false, disallowed, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration(%s): %v", name, err)
	}
	return e
}

// sgSchema finalizes a schema built by add. It carries no content model at all:
// these tests read the membership relation itself, not a constraint that
// consumes it.
func sgSchema(t *testing.T, add func(*SchemaBuilder)) *Schema {
	t.Helper()
	b := NewSchemaBuilder()
	add(b)
	s, err := b.Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	return s
}

// sgRef is the by-name {type definition} slot.
func sgRef(name QName) TypeDefinitionOrRef { return TypeDefinitionRef{Name: name} }

// expectMembership asserts the verdict of inSubstitutionGroupOf.
func expectMembership(t *testing.T, s *Schema, member, head QName, want bool) {
	t.Helper()
	if got := s.inSubstitutionGroupOf(member, head); got != want {
		t.Fatalf("inSubstitutionGroupOf(%s, %s) = %v, want %v", member, head, got, want)
	}
}

// TestSubstitutionGroupReflexive pins cos-equiv-derived-ok-rec clause 1: HEAD is
// in its own ·substitution group· (§3.3.6.4, key-eq), whatever its {disallowed
// substitutions} says — clause 1 and clause 2 are alternatives, not conjuncts.
func TestSubstitutionGroupReflexive(t *testing.T) {
	s := sgSchema(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("H"), QName{}, DerivationRestriction))
		b.AddElement(sgElement(t, sq("head"), sgRef(sq("H")), []DerivationMethod{DerivationSubstitution}))
	})
	expectMembership(t, s, sq("head"), sq("head"), true)
}

// TestSubstitutionGroupClause21 pins clause 2.1: head's {disallowed
// substitutions} containing substitution refuses every member.
func TestSubstitutionGroupClause21(t *testing.T) {
	s := sgSchema(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("H"), QName{}, DerivationRestriction))
		b.AddElement(sgElement(t, sq("head"), sgRef(sq("H")), []DerivationMethod{DerivationSubstitution}))
		b.AddElement(sgElement(t, sq("member"), sgRef(sq("H")), nil, sq("head")))
	})
	expectMembership(t, s, sq("member"), sq("head"), false)
}

// TestSubstitutionGroupClause22 pins clause 2.2 in both directions: the
// affiliation chain is followed transitively, and a declaration off the chain is
// not a member.
func TestSubstitutionGroupClause22(t *testing.T) {
	s := sgSchema(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("H"), QName{}, DerivationRestriction))
		b.AddElement(sgElement(t, sq("head"), sgRef(sq("H")), nil))
		b.AddElement(sgElement(t, sq("mid"), sgRef(sq("H")), nil, sq("head")))
		b.AddElement(sgElement(t, sq("member"), sgRef(sq("H")), nil, sq("mid")))
		b.AddElement(sgElement(t, sq("stranger"), sgRef(sq("H")), nil))
	})
	expectMembership(t, s, sq("member"), sq("head"), true)
	expectMembership(t, s, sq("stranger"), sq("head"), false)
}

// TestSubstitutionGroupClause23HeadDisallowedSubstitutions pins union member (1)
// of clause 2.3: head's own {disallowed substitutions} blocks the {derivation
// method}s the member's type reaches head's type by. The control differs only in
// the {derivation method} of the member's type, so the blocking is attributable
// to the intersection and not to the chain.
func TestSubstitutionGroupClause23HeadDisallowedSubstitutions(t *testing.T) {
	member := func(method DerivationMethod) *Schema {
		return sgSchema(t, func(b *SchemaBuilder) {
			b.AddType(sgType(t, sq("H"), QName{}, DerivationRestriction))
			b.AddType(sgType(t, sq("M"), sq("H"), method))
			b.AddElement(sgElement(t, sq("head"), sgRef(sq("H")), []DerivationMethod{DerivationExtension}))
			b.AddElement(sgElement(t, sq("member"), sgRef(sq("M")), nil, sq("head")))
		})
	}
	expectMembership(t, member(DerivationExtension), sq("member"), sq("head"), false)
	expectMembership(t, member(DerivationRestriction), sq("member"), sq("head"), true)
}

// TestSubstitutionGroupClause23HeadTypeProhibitedSubstitutions pins union member
// (2): H.{type definition}.{prohibited substitutions}, read off the endpoint of
// the ·derivation· and not off any type the walk passes through. Head's own
// {disallowed substitutions} is empty here, so only union member (2) can block.
func TestSubstitutionGroupClause23HeadTypeProhibitedSubstitutions(t *testing.T) {
	member := func(prohibited ...DerivationMethod) *Schema {
		return sgSchema(t, func(b *SchemaBuilder) {
			b.AddType(sgType(t, sq("H"), QName{}, DerivationRestriction, prohibited...))
			b.AddType(sgType(t, sq("M"), sq("H"), DerivationExtension))
			b.AddElement(sgElement(t, sq("head"), sgRef(sq("H")), nil))
			b.AddElement(sgElement(t, sq("member"), sgRef(sq("M")), nil, sq("head")))
		})
	}
	expectMembership(t, member(DerivationExtension), sq("member"), sq("head"), false)
	expectMembership(t, member(DerivationRestriction), sq("member"), sq("head"), true)
	expectMembership(t, member(), sq("member"), sq("head"), true)
}

// TestSubstitutionGroupClause23IntermediateProhibitedSubstitutions pins union
// member (3) INDEPENDENTLY of the other two: head's {disallowed substitutions}
// and head's own {type definition}.{prohibited substitutions} are both empty, and
// the only thing blocking is the {prohibited substitutions} of the type strictly
// between the two endpoints of the ·derivation·.
//
// It also pins that the blocking is not per-step: the extension step is BELOW the
// intermediate type that prohibits extension, so a walk that intersected each
// step's {derivation method} against the union accumulated so far would miss it.
func TestSubstitutionGroupClause23IntermediateProhibitedSubstitutions(t *testing.T) {
	member := func(prohibited ...DerivationMethod) *Schema {
		return sgSchema(t, func(b *SchemaBuilder) {
			b.AddType(sgType(t, sq("H"), QName{}, DerivationRestriction))
			b.AddType(sgType(t, sq("Mid"), sq("H"), DerivationRestriction, prohibited...))
			b.AddType(sgType(t, sq("M"), sq("Mid"), DerivationExtension))
			b.AddElement(sgElement(t, sq("head"), sgRef(sq("H")), nil))
			b.AddElement(sgElement(t, sq("member"), sgRef(sq("M")), nil, sq("head")))
		})
	}
	expectMembership(t, member(DerivationExtension), sq("member"), sq("head"), false)
	expectMembership(t, member(), sq("member"), sq("head"), true)
}

// TestSubstitutionGroupClause23MemberTypeProhibitedSubstitutionsIgnored pins the
// other edge of "intermediate": M.{type definition} is an ENDPOINT of the
// ·derivation·, not an intermediate, so its own {prohibited substitutions} is not
// in the blocking union even when it names the method M was derived by.
func TestSubstitutionGroupClause23MemberTypeProhibitedSubstitutionsIgnored(t *testing.T) {
	s := sgSchema(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("H"), QName{}, DerivationRestriction))
		b.AddType(sgType(t, sq("M"), sq("H"), DerivationExtension, DerivationExtension))
		b.AddElement(sgElement(t, sq("head"), sgRef(sq("H")), nil))
		b.AddElement(sgElement(t, sq("member"), sgRef(sq("M")), nil, sq("head")))
	})
	expectMembership(t, s, sq("member"), sq("head"), true)
}

// TestSubstitutionGroupClause23AnonymousMemberType pins that an ANONYMOUS {type
// definition} participates in the ·derivation· walk as its own first step: the
// member's inline type reaches head's type by extension, which head's {type
// definition} prohibits. A walk that skipped an unnamed type — or looked it up by
// its zero QName — would find no derivation method and wrongly admit the member.
func TestSubstitutionGroupClause23AnonymousMemberType(t *testing.T) {
	member := func(prohibited ...DerivationMethod) *Schema {
		return sgSchema(t, func(b *SchemaBuilder) {
			b.AddType(sgType(t, sq("H"), QName{}, DerivationRestriction, prohibited...))
			anon := sgType(t, QName{}, sq("H"), DerivationExtension)
			b.AddElement(sgElement(t, sq("head"), sgRef(sq("H")), nil))
			b.AddElement(dOwnInline(t, sq("member"), anon, NewGlobalScope(), sq("head")))
		})
	}
	expectMembership(t, member(DerivationExtension), sq("member"), sq("head"), false)
	expectMembership(t, member(), sq("member"), sq("head"), true)
}

// TestSubstitutionGroupClause23SimpleTypeOnDerivationChain pins that the
// ·derivation· walk runs THROUGH a simple type instead of stopping at it: M's
// {type definition} is a complex type extending the simple type B, and H's is the
// simple type A that B restricts, so key-derived's chain (§2.2.1.1, "D can reach
// B by following its base type definition chain") reaches H past one simple link.
// The extension step below that link is genuinely involved in the ·derivation·
// and head's {disallowed substitutions} blocks it (union member (1)).
//
// A walk that abandoned at the first non-complex type would discard that
// extension step and admit the member — a false membership, which under
// cos-nonambig false-REJECTS a schema the spec accepts. The control differs only
// in which method head disallows, so the verdict is attributable to the
// intersection and not to the chain having been walked at all.
func TestSubstitutionGroupClause23SimpleTypeOnDerivationChain(t *testing.T) {
	member := func(disallowed DerivationMethod) *Schema {
		return sgSchema(t, func(b *SchemaBuilder) {
			head := dPrimitive(t, sq("A"))
			mid := sgSimple(t, sq("B"), head)
			b.AddType(head)
			b.AddType(mid)
			b.AddType(sgSimpleContentType(t, sq("C"), mid))
			b.AddElement(sgElement(t, sq("head"), sgRef(sq("A")), []DerivationMethod{disallowed}))
			b.AddElement(sgElement(t, sq("member"), sgRef(sq("C")), nil, sq("head")))
		})
	}
	expectMembership(t, member(DerivationExtension), sq("member"), sq("head"), false)
	expectMembership(t, member(DerivationRestriction), sq("member"), sq("head"), true)
}

// TestSubstitutionGroupClause23SimpleTypeChainNotReachingHead pins the other exit
// from the simple arm: the chain tops out at xs:anySimpleType (whose {base type
// definition} is absent) without reaching H.{type definition}, so no ·derivation·
// exists, no {derivation method} is involved, and clause 2.3 blocks nothing —
// the same vacuous reading TestSubstitutionGroupClause23NoDerivation records.
func TestSubstitutionGroupClause23SimpleTypeChainNotReachingHead(t *testing.T) {
	s := sgSchema(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("H"), QName{}, DerivationRestriction, DerivationExtension))
		unrelated := dPrimitive(t, sq("Unrelated"))
		b.AddType(unrelated)
		b.AddType(sgSimpleContentType(t, sq("C"), unrelated))
		b.AddElement(sgElement(t, sq("head"), sgRef(sq("H")), []DerivationMethod{DerivationExtension}))
		b.AddElement(sgElement(t, sq("member"), sgRef(sq("C")), nil, sq("head")))
	})
	expectMembership(t, s, sq("member"), sq("head"), true)
}

// TestSubstitutionGroupClause23NoDerivation pins the reading recorded on
// derivationAdmitsSubstitution: clause 2.3 is a BLOCKING clause, so a member
// whose {type definition} does not ·reach· head's at all involves no {derivation
// method}s and is blocked by nothing. Requiring the derivation is
// e-props-correct clause 4's job (c-vs-sg), not this rule's — and clause 4 is
// unimplemented here, so this pins silence, not delegation (see the GAP on
// derivationAdmitsSubstitution).
func TestSubstitutionGroupClause23NoDerivation(t *testing.T) {
	s := sgSchema(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("H"), QName{}, DerivationRestriction, DerivationExtension))
		b.AddType(sgType(t, sq("Unrelated"), QName{}, DerivationRestriction))
		b.AddType(sgType(t, sq("M"), sq("Unrelated"), DerivationExtension))
		b.AddElement(sgElement(t, sq("head"), sgRef(sq("H")), []DerivationMethod{DerivationExtension}))
		b.AddElement(sgElement(t, sq("member"), sgRef(sq("M")), nil, sq("head")))
	})
	expectMembership(t, s, sq("member"), sq("head"), true)
}

// TestSubstitutionGroupClause23SameTypeDefinition pins the zero-step case: member
// and head share one {type definition}, so no {derivation method} is involved and
// no blocking keyword can bite.
func TestSubstitutionGroupClause23SameTypeDefinition(t *testing.T) {
	s := sgSchema(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("H"), QName{}, DerivationRestriction, DerivationExtension, DerivationRestriction))
		b.AddElement(sgElement(t, sq("head"), sgRef(sq("H")), []DerivationMethod{DerivationExtension, DerivationRestriction}))
		b.AddElement(sgElement(t, sq("member"), sgRef(sq("H")), nil, sq("head")))
	})
	expectMembership(t, s, sq("member"), sq("head"), true)
}

// TestSubstitutionGroupHeadNotTopLevel pins that a name outside {element
// declarations} heads no ·substitution group·: §3.3.6.4 defines one for each
// declaration in the {element declarations} of a schema, which §3.17.1 restricts
// to top-level, so a local or absent head answers false.
func TestSubstitutionGroupHeadNotTopLevel(t *testing.T) {
	s := sgSchema(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("H"), QName{}, DerivationRestriction))
		b.AddElement(sgElement(t, sq("member"), sgRef(sq("H")), nil))
	})
	expectMembership(t, s, sq("member"), sq("absent"), false)
}
