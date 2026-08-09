package xsd

import (
	"errors"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests are package-internal because checkSubstitutionGroupTypes is an
// unexported finalize step: the verdict is read off SchemaBuilder.Finalize, which
// is the only way a *Schema comes into existence, so every case below is the
// whole rule end to end rather than a call to the checker.
//
// They share substitutiongroup_test.go's builders — sq, sgType, sgSimple,
// sgSchema, sgRef (STYLE T4) — and add only what those cannot express: a
// declaration carrying a {substitution group exclusions}, which is the property
// clause 4 reads and which no earlier test needed.

// cvsLoc is a distinguishable non-zero position, so a rejection can be shown to
// carry the OFFENDING declaration's own Loc rather than a zero one.
var cvsLoc = xsderr.Loc{URI: "cvs.xsd", Line: 7, Col: 3}

// cvsElement builds a top-level element declaration with a {substitution group
// exclusions} and a real Loc. sgElement carries neither, and widening it would
// touch every one of its call sites for a property only this file reads.
func cvsElement(t *testing.T, name QName, typeDef TypeDefinitionOrRef, exclusions []DerivationMethod, affiliations ...QName) ElementDeclaration {
	t.Helper()
	e, err := NewElementDeclaration(cvsLoc, name, typeDef, nil, NewGlobalScope(), nil, false, nil,
		affiliations, exclusions, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration(%s): %v", name, err)
	}
	return e
}

// cvsFinalize builds and finalizes, returning the verdict instead of failing on
// it: these tests assert both polarities, so the error is the subject.
func cvsFinalize(t *testing.T, add func(*SchemaBuilder)) error {
	t.Helper()
	b := NewSchemaBuilder()
	add(b)
	_, err := b.Finalize()
	return err
}

// TestEPropsCorrectClause4RejectsUnrelatedType pins the rule's core: a member
// whose {type definition} does not ·derive· from its head's at all is rejected
// (§3.3.6.1 clause 4, c-vs-sg). This is the direction that was accepted before
// #395 — cos-equiv-derived-ok-rec clause 2.3 blocks nothing when the ·derivation·
// does not exist, and nothing else required it to.
func TestEPropsCorrectClause4RejectsUnrelatedType(t *testing.T) {
	err := cvsFinalize(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("H"), QName{}, DerivationRestriction))
		b.AddType(sgType(t, sq("Unrelated"), QName{}, DerivationRestriction))
		b.AddType(sgType(t, sq("M"), sq("Unrelated"), DerivationRestriction))
		b.AddElement(cvsElement(t, sq("head"), sgRef(sq("H")), nil))
		b.AddElement(cvsElement(t, sq("member"), sgRef(sq("M")), nil, sq("head")))
	})
	expectRule(t, err, ruleEPropsCorrect)
	var xe *xsderr.Error
	if !errors.As(err, &xe) {
		t.Fatalf("expected an *xsderr.Error, got %T", err)
	}
	if xe.Loc != cvsLoc {
		t.Fatalf("rejection carries Loc %s, want the declaration's own %s", xe.Loc, cvsLoc)
	}
}

// TestEPropsCorrectClause4AcceptsDerivedType is the control for the case above,
// differing ONLY in what M's {base type definition} is: a member whose type
// really does derive from the head's stands. Without it the rejection above
// would be satisfiable by a checker that rejected every affiliation.
func TestEPropsCorrectClause4AcceptsDerivedType(t *testing.T) {
	err := cvsFinalize(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("H"), QName{}, DerivationRestriction))
		b.AddType(sgType(t, sq("M"), sq("H"), DerivationRestriction))
		b.AddElement(cvsElement(t, sq("head"), sgRef(sq("H")), nil))
		b.AddElement(cvsElement(t, sq("member"), sgRef(sq("M")), nil, sq("head")))
	})
	if err != nil {
		t.Fatalf("a member derived from its head's type was rejected: %v", err)
	}
}

// TestEPropsCorrectClause4BlockingKeywords pins the blocking-keyword half
// INDEPENDENTLY of the chain-existence half: the type pair is identical in all
// three schemas — M extends H, so the ·derivation· always exists and always
// consists of exactly the extension step — and only the {substitution group
// exclusions} moves.
//
// The three cases also pin WHOSE property is read. It is the HEAD's, per "subject
// to the blocking keywords in M.{substitution group exclusions}" where M is the
// affiliation member: the head carrying extension rejects, the MEMBER carrying
// the very same keyword does not, and neither declaration's {disallowed
// substitutions} is consulted at all (that property feeds
// cos-equiv-derived-ok-rec clause 2.1, a different rule).
func TestEPropsCorrectClause4BlockingKeywords(t *testing.T) {
	schema := func(headExclusions, memberExclusions []DerivationMethod) error {
		return cvsFinalize(t, func(b *SchemaBuilder) {
			b.AddType(sgType(t, sq("H"), QName{}, DerivationRestriction))
			b.AddType(sgType(t, sq("M"), sq("H"), DerivationExtension))
			b.AddElement(cvsElement(t, sq("head"), sgRef(sq("H")), headExclusions))
			b.AddElement(cvsElement(t, sq("member"), sgRef(sq("M")), memberExclusions, sq("head")))
		})
	}
	if err := schema(nil, nil); err != nil {
		t.Fatalf("an unblocked extension was rejected: %v", err)
	}
	expectRule(t, schema([]DerivationMethod{DerivationExtension}, nil), ruleEPropsCorrect)
	if err := schema(nil, []DerivationMethod{DerivationExtension}); err != nil {
		t.Fatalf("the MEMBER's own {substitution group exclusions} blocked its own substitution, but clause 4 reads the head's: %v", err)
	}
}

// TestEPropsCorrectClause4RestrictionKeyword pins the other member of the
// two-keyword set, so a checker that only ever consulted extension is caught: the
// same shape under restriction, blocked and unblocked.
func TestEPropsCorrectClause4RestrictionKeyword(t *testing.T) {
	schema := func(exclusions []DerivationMethod) error {
		return cvsFinalize(t, func(b *SchemaBuilder) {
			b.AddType(sgType(t, sq("H"), QName{}, DerivationRestriction))
			b.AddType(sgType(t, sq("M"), sq("H"), DerivationRestriction))
			b.AddElement(cvsElement(t, sq("head"), sgRef(sq("H")), exclusions))
			b.AddElement(cvsElement(t, sq("member"), sgRef(sq("M")), nil, sq("head")))
		})
	}
	if err := schema(nil); err != nil {
		t.Fatalf("an unblocked restriction was rejected: %v", err)
	}
	expectRule(t, schema([]DerivationMethod{DerivationRestriction}), ruleEPropsCorrect)
}

// TestEPropsCorrectClause4DirectMembersOnly pins that clause 4 quantifies over a
// declaration's OWN {substitution group affiliations} and does not walk the
// chain: "for each member M of E.{substitution group affiliations}".
//
// member is affiliated to mid and mid to head. Every DIRECT edge passes — M
// extends X under mid's empty exclusions, X restricts H under head's
// {extension} — so the schema stands. A checker that followed the chain to head
// would test M against H subject to head's {extension} and REJECT, because the
// extension step from M is then inside the ·derivation· under test. The
// transitive reading belongs to cos-equiv-derived-ok-rec clause 2.2, which is a
// membership question and not a schema component constraint.
func TestEPropsCorrectClause4DirectMembersOnly(t *testing.T) {
	err := cvsFinalize(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("H"), QName{}, DerivationRestriction))
		b.AddType(sgType(t, sq("X"), sq("H"), DerivationRestriction))
		b.AddType(sgType(t, sq("M"), sq("X"), DerivationExtension))
		b.AddElement(cvsElement(t, sq("head"), sgRef(sq("H")), []DerivationMethod{DerivationExtension}))
		b.AddElement(cvsElement(t, sq("mid"), sgRef(sq("X")), nil, sq("head")))
		b.AddElement(cvsElement(t, sq("member"), sgRef(sq("M")), nil, sq("mid")))
	})
	if err != nil {
		t.Fatalf("clause 4 walked the affiliation chain instead of the direct members: %v", err)
	}
}

// TestEPropsCorrectClause4EveryMemberChecked pins the other edge of "for each
// member": {substitution group affiliations} is a LIST in XSD 1.1, and a
// declaration valid against its first head and invalid against its second is
// still invalid. It is the shape of W3C ibmData S2_2_2/s2_2_2si02.
func TestEPropsCorrectClause4EveryMemberChecked(t *testing.T) {
	err := cvsFinalize(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("H"), QName{}, DerivationRestriction))
		b.AddType(sgType(t, sq("Unrelated"), QName{}, DerivationRestriction))
		b.AddType(sgType(t, sq("M"), sq("H"), DerivationRestriction))
		b.AddElement(cvsElement(t, sq("first"), sgRef(sq("H")), nil))
		b.AddElement(cvsElement(t, sq("second"), sgRef(sq("Unrelated")), nil))
		b.AddElement(cvsElement(t, sq("member"), sgRef(sq("M")), nil, sq("first"), sq("second")))
	})
	expectRule(t, err, ruleEPropsCorrect)
}

// TestEPropsCorrectClause4DanglingAffiliationNotCharged pins the §5.3 skip: an
// affiliation naming no declaration is an ·absent· member, which resolve.go's
// Phase A deliberately does not reject (resolveElementDecl, and W3C
// saxonData/Missing missing002), so clause 4 has no M to read a {type
// definition} from and must not charge one either. Without the skip the schema
// below would be rejected twice over — once here and once by a Phase A that does
// not fire.
func TestEPropsCorrectClause4DanglingAffiliationNotCharged(t *testing.T) {
	err := cvsFinalize(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("M"), QName{}, DerivationRestriction))
		b.AddElement(cvsElement(t, sq("member"), sgRef(sq("M")), nil, sq("nosuchhead")))
	})
	if err != nil {
		t.Fatalf("a dangling affiliation was charged, but §5.3 makes it an ·absent· member: %v", err)
	}
}

// TestEPropsCorrectClause4AbsentTypeSkipped pins the second skip, in both
// directions: with no {type definition} on one end of the edge there is nothing
// to be ·validly substitutable· for, or with. The two declarations' types are
// otherwise unrelated, so a checker that did not skip would reject each schema.
func TestEPropsCorrectClause4AbsentTypeSkipped(t *testing.T) {
	schema := func(headType, memberType TypeDefinitionOrRef) error {
		return cvsFinalize(t, func(b *SchemaBuilder) {
			b.AddType(sgType(t, sq("H"), QName{}, DerivationRestriction))
			b.AddType(sgType(t, sq("M"), QName{}, DerivationRestriction))
			b.AddElement(cvsElement(t, sq("head"), headType, nil))
			b.AddElement(cvsElement(t, sq("member"), memberType, nil, sq("head")))
		})
	}
	if err := schema(nil, sgRef(sq("M"))); err != nil {
		t.Fatalf("a head with an absent {type definition} was charged: %v", err)
	}
	if err := schema(sgRef(sq("H")), nil); err != nil {
		t.Fatalf("a member with an absent {type definition} was charged: %v", err)
	}
	expectRule(t, schema(sgRef(sq("H")), sgRef(sq("M"))), ruleEPropsCorrect)
}

// TestEPropsCorrectClause4SimpleTypes pins the third case of ·validly
// substitutable· (key-val-sub-type: "S is a simple type definition and S is
// validly ·derived· from T"), so the rule is not silently complex-only: a member
// typed by a simple type restricting the head's stands, and one typed by an
// unrelated simple type does not.
func TestEPropsCorrectClause4SimpleTypes(t *testing.T) {
	schema := func(memberBase func(head *SimpleType) *SimpleType) error {
		return cvsFinalize(t, func(b *SchemaBuilder) {
			head := sgSimple(t, sq("A"), AnyAtomicType())
			b.AddType(head)
			b.AddType(memberBase(head))
			b.AddElement(cvsElement(t, sq("head"), sgRef(sq("A")), nil))
			b.AddElement(cvsElement(t, sq("member"), sgRef(sq("B")), nil, sq("head")))
		})
	}
	if err := schema(func(head *SimpleType) *SimpleType { return sgSimple(t, sq("B"), head) }); err != nil {
		t.Fatalf("a simple type restricting the head's was rejected: %v", err)
	}
	expectRule(t, schema(func(*SimpleType) *SimpleType { return sgSimple(t, sq("B"), AnyAtomicType()) }), ruleEPropsCorrect)
}

// TestEPropsCorrectClause4IgnoresHeadTypeProhibitedSubstitutions pins the one
// term clause 4 does NOT read: the head TYPE's {prohibited substitutions}.
// Composing the clause's own words with key-val-sub-type's complex/complex arm
// would union that property into the blocking set and reject the first schema
// below; §3.3.3 ("An empty {substitution group exclusions} allows a declaration
// to be named in the {substitution group affiliations} of other element
// declarations having … some type ·derived· therefrom"), §3.4.1's list of what
// {prohibited substitutions} governs, and W3C sunData ElemDecl
// disallowedSubst00503m3/m4/m5 all say it stands. See
// checkElementSubstitutableForHeads for the full reading.
//
// The second schema is the control that keeps this from being satisfied by a
// checker that stopped reading blocking keywords altogether: the SAME derivation
// step, blocked by the head's {substitution group exclusions} instead, is still
// rejected. The third pins that the property genuinely still bites where it
// belongs — cos-equiv-derived-ok-rec clause 2.3 keeps the member out of the
// head's ·substitution group· in exactly the schema clause 4 accepts.
func TestEPropsCorrectClause4IgnoresHeadTypeProhibitedSubstitutions(t *testing.T) {
	build := func(b *SchemaBuilder, exclusions []DerivationMethod) {
		b.AddType(sgType(t, sq("H"), QName{}, DerivationRestriction, DerivationExtension))
		b.AddType(sgType(t, sq("M"), sq("H"), DerivationExtension))
		b.AddElement(cvsElement(t, sq("head"), sgRef(sq("H")), exclusions))
		b.AddElement(cvsElement(t, sq("member"), sgRef(sq("M")), nil, sq("head")))
	}
	if err := cvsFinalize(t, func(b *SchemaBuilder) { build(b, nil) }); err != nil {
		t.Fatalf("the head TYPE blocks extension but the head DECLARATION excludes nothing, so clause 4 must accept: %v", err)
	}
	expectRule(t, cvsFinalize(t, func(b *SchemaBuilder) {
		build(b, []DerivationMethod{DerivationExtension})
	}), ruleEPropsCorrect)

	s := sgSchema(t, func(b *SchemaBuilder) { build(b, nil) })
	if s.inSubstitutionGroupOf(sq("member"), sq("head")) {
		t.Fatal("clause 4 accepted the affiliation, but the head type's {prohibited substitutions} must still keep the member out of its ·substitution group· (cos-equiv-derived-ok-rec clause 2.3)")
	}
}
