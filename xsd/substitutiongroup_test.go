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
func sgType(t *testing.T, name, base QName, method DerivationMethod, prohibited ...DerivationMethod) ComplexType {
	t.Helper()
	ct, err := NewComplexType(xsderr.Loc{}, name, base, nil, method, false,
		nil, nil, EmptyContent{}, prohibited, nil, nil)
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
			anon := InlineTypeDefinition{Definition: sgType(t, QName{}, sq("H"), DerivationExtension)}
			b.AddElement(sgElement(t, sq("head"), sgRef(sq("H")), nil))
			b.AddElement(sgElement(t, sq("member"), anon, nil, sq("head")))
		})
	}
	expectMembership(t, member(DerivationExtension), sq("member"), sq("head"), false)
	expectMembership(t, member(), sq("member"), sq("head"), true)
}

// TestSubstitutionGroupClause23NoDerivation pins the reading recorded on
// derivationAdmitsSubstitution: clause 2.3 is a BLOCKING clause, so a member
// whose {type definition} does not ·reach· head's at all involves no {derivation
// method}s and is blocked by nothing. Requiring the derivation is
// e-props-correct clause 4's job, not this rule's.
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
