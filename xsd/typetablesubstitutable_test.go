package xsd

import (
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests are package-internal because checkTypeTableSubstitutability is an
// unexported finalize step: the verdict is read off SchemaBuilder.Finalize, the
// only way a *Schema comes into existence, so every case below is the whole rule
// end to end.
//
// They reuse the builders three other files already own (STYLE T4): sq, sgType,
// sgSimple and sgSimpleContentType from substitutiongroup_test.go, eTypeTable
// and eGlobalWithTable from elementconsistent_test.go, uOne/uGroup/uLocalScope
// and expectRule from particleattribution_test.go. Only what none of them can
// express is added — a table whose entries are chosen per test, and a
// declaration carrying a {disallowed substitutions} alongside one.

// ttTable builds a {type table} whose {alternatives} name altTypes in order and
// whose {default type definition} names defaultType. eTypeTable fixes the table
// at exactly one alternative; clause 7 quantifies over all of them, so a test
// that puts the offending entry second needs this.
func ttTable(t *testing.T, defaultType QName, altTypes ...QName) *TypeTable {
	t.Helper()
	alts := make([]TypeAlternative, 0, len(altTypes))
	for _, at := range altTypes {
		test := NewXPathExpression("@kind = 'x'", nil, nil, nil)
		alts = append(alts, NewTypeAlternative(&test, at, nil))
	}
	tt, err := NewTypeTable(xsderr.Loc{}, alts, NewTypeAlternative(nil, defaultType, nil))
	if err != nil {
		t.Fatalf("NewTypeTable: %v", err)
	}
	return &tt
}

// ttElement builds a TOP-LEVEL declaration typed typeName, carrying tt and the
// given {disallowed substitutions}. eGlobalWithTable carries neither that
// property nor a non-zero Loc, and widening it would touch every call site for
// something only this file reads.
func ttElement(t *testing.T, name, typeName QName, tt *TypeTable, disallowed ...DerivationMethod) ElementDeclaration {
	t.Helper()
	e, err := NewElementDeclaration(ttLoc, name, TypeDefinitionRef{Name: typeName}, tt, NewGlobalScope(), nil, false, nil,
		nil, nil, false, disallowed, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration(%s): %v", name, err)
	}
	return e
}

// ttLoc is a distinguishable non-zero position, so a rejection can be shown to
// carry the OFFENDING declaration's own Loc rather than a zero one.
var ttLoc = xsderr.Loc{URI: "typetable.xsd", Line: 11, Col: 5}

// ttErrorType builds ·xs:error· (§3.16.7.3) exactly as builtin.buildErrorType
// does — a UNION over an empty {member type definitions}, based on the
// xs:anySimpleType anchor, {final} all four keywords. It is built here rather
// than imported because package xsd cannot depend on builtin (ARCHITECTURE), and
// because clause 7.2 identifies the component by expanded name, which is the
// property under test.
func ttErrorType(t *testing.T) *SimpleType {
	t.Helper()
	final := []DerivationMethod{DerivationExtension, DerivationRestriction, DerivationList, DerivationUnion}
	st, err := NewSimpleType(xsderr.Loc{}, errorTypeName, UnionDerivation{}, OwnedSimpleType{Definition: anySimpleType}, nil, final)
	if err != nil {
		t.Fatalf("NewSimpleType(xs:error): %v", err)
	}
	return st
}

// ttFinalize builds and finalizes, returning the verdict instead of failing on
// it: these tests assert both polarities, so the error is the subject.
func ttFinalize(t *testing.T, add func(*SchemaBuilder)) error {
	t.Helper()
	b := NewSchemaBuilder()
	add(b)
	_, err := b.Finalize()
	return err
}

// TestEPropsCorrectClause7RejectsUnrelatedAlternative is the rule's core: an
// alternative whose type does not ·derive· from the declaration's own {type
// definition} at all is rejected (§3.3.6.1 clause 7). This is the direction that
// was accepted before #823 — nothing consulted clause 7, so the rejection was
// UNMADE.
func TestEPropsCorrectClause7RejectsUnrelatedAlternative(t *testing.T) {
	err := ttFinalize(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("Base"), QName{}, DerivationRestriction))
		b.AddType(sgType(t, sq("Other"), QName{}, DerivationRestriction))
		b.AddElement(ttElement(t, sq("e"), sq("Base"), ttTable(t, sq("Base"), sq("Other"))))
	})
	expectRule(t, err, ruleEPropsCorrect)
	if !strings.Contains(err.Error(), "{alternatives}[0]") {
		t.Errorf("message does not name the offending slot: %v", err)
	}
	var xe *xsderr.Error
	if !errors.As(err, &xe) || xe.Loc != ttLoc {
		t.Errorf("rejection is not charged to the declaration's own Loc: %v", err)
	}
}

// TestEPropsCorrectClause7AcceptsRestrictionOfDeclaredType is clause 7.1
// satisfied by the ordinary case: the alternative's type restricts the
// declaration's own, so it is ·validly substitutable· for it.
func TestEPropsCorrectClause7AcceptsRestrictionOfDeclaredType(t *testing.T) {
	err := ttFinalize(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("Base"), QName{}, DerivationRestriction))
		b.AddType(sgType(t, sq("Sub"), sq("Base"), DerivationRestriction))
		b.AddElement(ttElement(t, sq("e"), sq("Base"), ttTable(t, sq("Base"), sq("Sub"))))
	})
	if err != nil {
		t.Fatalf("Finalize rejected a valid type table: %v", err)
	}
}

// TestEPropsCorrectClause7QuantifiesOverEveryAlternative pins the "for each"
// quantifier: a table whose FIRST alternative is fine and whose second is not is
// still rejected, and the message names the second.
func TestEPropsCorrectClause7QuantifiesOverEveryAlternative(t *testing.T) {
	err := ttFinalize(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("Base"), QName{}, DerivationRestriction))
		b.AddType(sgType(t, sq("Sub"), sq("Base"), DerivationRestriction))
		b.AddType(sgType(t, sq("Other"), QName{}, DerivationRestriction))
		b.AddElement(ttElement(t, sq("e"), sq("Base"), ttTable(t, sq("Base"), sq("Sub"), sq("Other"))))
	})
	expectRule(t, err, ruleEPropsCorrect)
	if !strings.Contains(err.Error(), "{alternatives}[1]") {
		t.Errorf("message does not name the second alternative: %v", err)
	}
}

// TestEPropsCorrectClause7ChargesTheDefaultTypeDefinition pins the clause's
// second half — "and also for E.{type table}.{default type definition}.{type
// definition}" — which a charge quantified over {alternatives} alone would miss.
func TestEPropsCorrectClause7ChargesTheDefaultTypeDefinition(t *testing.T) {
	err := ttFinalize(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("Base"), QName{}, DerivationRestriction))
		b.AddType(sgType(t, sq("Sub"), sq("Base"), DerivationRestriction))
		b.AddType(sgType(t, sq("Other"), QName{}, DerivationRestriction))
		b.AddElement(ttElement(t, sq("e"), sq("Base"), ttTable(t, sq("Other"), sq("Sub"))))
	})
	expectRule(t, err, ruleEPropsCorrect)
	if !strings.Contains(err.Error(), "{default type definition}") {
		t.Errorf("message does not name the default type definition: %v", err)
	}
}

// TestEPropsCorrectClause7AcceptsXSErrorAlternative ARMS clause 7.2: an
// alternative naming ·xs:error· satisfies clause 7 without reaching 7.1, which
// it would fail — xs:error is a simple type and the declaration's own {type
// definition} here is complex, so validlyDerived answers false for it. The pair
// with the test below is what makes this one able to fail: swap xs:error for any
// other simple type and the schema is rejected.
func TestEPropsCorrectClause7AcceptsXSErrorAlternative(t *testing.T) {
	err := ttFinalize(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("Base"), QName{}, DerivationRestriction))
		b.AddType(ttErrorType(t))
		b.AddElement(ttElement(t, sq("e"), sq("Base"), ttTable(t, sq("Base"), errorTypeName)))
	})
	if err != nil {
		t.Fatalf("Finalize rejected an alternative naming ·xs:error·, which clause 7.2 exempts: %v", err)
	}
}

// TestEPropsCorrectClause7RejectsANonErrorSimpleAlternative is the control for
// the test above: a simple type that is NOT xs:error, in the same slot against
// the same complex declared type, is rejected. Without it, clause 7.2's
// exemption could be a blanket "any simple alternative passes" and no test would
// notice.
func TestEPropsCorrectClause7RejectsANonErrorSimpleAlternative(t *testing.T) {
	err := ttFinalize(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("Base"), QName{}, DerivationRestriction))
		b.AddType(ttAtomic(t, sq("S")))
		b.AddElement(ttElement(t, sq("e"), sq("Base"), ttTable(t, sq("Base"), sq("S"))))
	})
	expectRule(t, err, ruleEPropsCorrect)
}

// TestEPropsCorrectClause7AcceptsAUnionMemberAlternative is the settled
// union-base reading of cos-st-derived-ok (§3.16.6.3) clause 2.2.4, and the
// shape conditional type assignment normally takes: the declaration's own {type
// definition} is a union and the alternative names one of its MEMBERS, which
// 2.2.4.2 admits by recursing into B's transitive membership even though neither
// type is ·derived· from the other.
func TestEPropsCorrectClause7AcceptsAUnionMemberAlternative(t *testing.T) {
	err := ttFinalize(t, func(b *SchemaBuilder) {
		b.AddType(ttAtomic(t, sq("M1")))
		b.AddType(ttAtomic(t, sq("M2")))
		b.AddType(ttUnion(t, sq("U"), sq("M1"), sq("M2")))
		b.AddElement(ttElement(t, sq("e"), sq("U"), ttTable(t, sq("U"), sq("M2"))))
	})
	if err != nil {
		t.Fatalf("Finalize rejected an alternative naming a member of the declared union type (cos-st-derived-ok clause 2.2.4): %v", err)
	}
}

// TestEPropsCorrectClause7RejectsANonMemberAlternative is the control for the
// union reading: a simple type that is NOT in the declared union's transitive
// membership, and derives from neither it nor any member, is rejected. Without
// it, clause 2.2.4 could be reading the union as "admits everything".
func TestEPropsCorrectClause7RejectsANonMemberAlternative(t *testing.T) {
	err := ttFinalize(t, func(b *SchemaBuilder) {
		b.AddType(ttAtomic(t, sq("M1")))
		b.AddType(ttAtomic(t, sq("M2")))
		b.AddType(ttAtomic(t, sq("Outsider")))
		b.AddType(ttUnion(t, sq("U"), sq("M1"), sq("M2")))
		b.AddElement(ttElement(t, sq("e"), sq("U"), ttTable(t, sq("U"), sq("Outsider"))))
	})
	expectRule(t, err, ruleEPropsCorrect)
}

// ttAtomic builds a top-level ATOMIC simple type — a primitive, which is the
// shortest well-formed atomic component this package can mint without a facet
// pipeline. sgSimple cannot serve: it restricts its argument, and a restriction
// of xs:anySimpleType has an ·absent· {variety}, which st-props-correct clause 1
// rejects.
func ttAtomic(t *testing.T, name QName) *SimpleType {
	t.Helper()
	st, err := newCheckedPrimitiveType(xsderr.Loc{}, name, nil, nil)
	if err != nil {
		t.Fatalf("newCheckedPrimitiveType(%s): %v", name, err)
	}
	return st
}

// ttUnion builds a top-level union simple type whose members are BY NAME, the
// only encoding under which a member is also an indexable top-level type an
// <alternative> can name.
func ttUnion(t *testing.T, name QName, members ...QName) *SimpleType {
	t.Helper()
	slots := make([]SimpleTypeOrRef, 0, len(members))
	for _, m := range members {
		slots = append(slots, SimpleTypeRef{Name: m})
	}
	st, err := NewSimpleType(xsderr.Loc{}, name, UnionDerivation{Members: slots},
		OwnedSimpleType{Definition: anySimpleType}, nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(union %s): %v", name, err)
	}
	return st
}

// TestEPropsCorrectClause7ReadsDisallowedSubstitutions pins the "subject to the
// blocking keywords of E.{disallowed substitutions}" half: the same alternative
// that passes with an empty set is rejected once the declaration blocks
// restriction. A charge that threaded no blocking set would accept both.
func TestEPropsCorrectClause7ReadsDisallowedSubstitutions(t *testing.T) {
	err := ttFinalize(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("Base"), QName{}, DerivationRestriction))
		b.AddType(sgType(t, sq("Sub"), sq("Base"), DerivationRestriction))
		b.AddElement(ttElement(t, sq("e"), sq("Base"), ttTable(t, sq("Base"), sq("Sub")), DerivationRestriction))
	})
	expectRule(t, err, ruleEPropsCorrect)
}

// TestEPropsCorrectClause7ReachesLocalDeclarations pins the quantifier: a LOCAL
// element declaration carries a {type table} by the same §3.3.2.1 mapping rule a
// top-level one does, so a charge walking s.elements alone would accept this
// schema. The offending declaration sits inside a complex type's content model.
func TestEPropsCorrectClause7ReachesLocalDeclarations(t *testing.T) {
	err := ttFinalize(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("Base"), QName{}, DerivationRestriction))
		b.AddType(sgType(t, sq("Other"), QName{}, DerivationRestriction))
		local, lerr := NewElementDeclaration(ttLoc, sq("inner"), TypeDefinitionRef{Name: sq("Base")},
			ttTable(t, sq("Base"), sq("Other")), uLocalScope(t), nil, false, nil, nil, nil, false, nil, nil)
		if lerr != nil {
			t.Fatalf("NewElementDeclaration(inner): %v", lerr)
		}
		b.AddType(uCT(t, sq("ct"), uOne(t, ResolvedTerm{Term: local})))
	})
	expectRule(t, err, ruleEPropsCorrect)
}

// TestEPropsCorrectClause7SkipsAnAbsentDeclaredType pins the fail-open skip:
// clause 7.1 predicates over E.{type definition}, so a declaration with an
// ·absent· one (§5.3) has nothing for T to be substitutable FOR and is accepted
// rather than rejected on a component that is not there.
func TestEPropsCorrectClause7SkipsAnAbsentDeclaredType(t *testing.T) {
	err := ttFinalize(t, func(b *SchemaBuilder) {
		b.AddType(sgType(t, sq("Other"), QName{}, DerivationRestriction))
		e, eerr := NewElementDeclaration(ttLoc, sq("e"), nil, ttTable(t, sq("Other"), sq("Other")),
			NewGlobalScope(), nil, false, nil, nil, nil, false, nil, nil)
		if eerr != nil {
			t.Fatalf("NewElementDeclaration(e): %v", eerr)
		}
		b.AddElement(e)
	})
	if err != nil {
		t.Fatalf("Finalize rejected a declaration with an ·absent· {type definition}: %v", err)
	}
}
