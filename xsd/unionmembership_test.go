package xsd

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests are about the BY-NAME arm of {member type definitions}, which is
// why they resolve against a real finalized Schema rather than against
// simpletypefixture_test.go's noSchema, for the reason simpletyperef_test.go's
// header gives: a SimpleTypeRef is exactly the arm a resolves-nothing resolver
// cannot follow, and cos-st-restricts clause 3.3 is exactly the rule that arm
// makes checkable.

// unionType builds a named union whose members are by-name SimpleTypeRefs, over
// the base slot the caller hands it. A produced union always takes
// xs:anySimpleType there (§3.16.2.1 map.std.common case 2); a by-name base is
// what only a programmatic caller can build, and is what clause 3.3's second
// conjunct is about.
func unionType(t *testing.T, local string, base SimpleTypeOrRef, members ...QName) *SimpleType {
	t.Helper()
	slots := make([]SimpleTypeOrRef, 0, len(members))
	for _, m := range members {
		slots = append(slots, SimpleTypeRef{Name: m})
	}
	st, err := NewSimpleType(xsderr.Loc{}, QName{Local: local}, UnionDerivation{Members: slots}, base, nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(union %s): %v", local, err)
	}
	return st
}

// TestUnionMembershipCycleRejected is cos-st-restricts clause 3.3's first
// conjunct: a by-name membership makes "D is a member of its own transitive
// membership" representable — it was not while every member had to pre-exist the
// union holding it — so checkUnionMembershipAcyclic exists to charge it, and the
// unguarded membership walks in derivation.go rest on that charge.
func TestUnionMembershipCycleRejected(t *testing.T) {
	for _, tc := range []struct {
		name  string
		types []*SimpleType
	}{
		{"a union naming itself", []*SimpleType{
			unionType(t, "U", ownedBase(anySimpleType), QName{Local: "U"}),
		}},
		{"two unions naming each other", []*SimpleType{
			unionType(t, "U", ownedBase(anySimpleType), QName{Local: "V"}),
			unionType(t, "V", ownedBase(anySimpleType), QName{Local: "U"}),
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := NewSchemaBuilder()
			for _, st := range tc.types {
				b.AddType(st)
			}
			_, err := b.Finalize()
			if err == nil {
				t.Fatal("Finalize(a union membership cycle) = nil, want a clause 3.3 rejection")
			}
			if r, _ := xsderr.RuleOf(err); r != ruleCosSTRestricts {
				t.Fatalf("charged %s, want %s", r, ruleCosSTRestricts)
			}
			if !strings.Contains(err.Error(), "cos-st-restricts clause 3.3") {
				t.Fatalf("message does not name clause 3.3, the clause this check owns: %v", err)
			}
		})
	}
}

// TestUnionMembershipDerivedMemberRejected is clause 3.3's SECOND conjunct, and
// the only test that isolates it: D's member M is ·derived· from D — its base
// chain reaches D — while the membership graph itself is a plain tree, so a
// check that asked only "is D in its own transitive membership" would accept it.
// The shape needs a UnionDerivation whose base is not xs:anySimpleType, which
// only a programmatic caller can build: the parser's union arm always mints
// xs:anySimpleType there, and a restriction OF a union inherits its membership
// rather than declaring one, which closes a first-conjunct cycle instead.
func TestUnionMembershipDerivedMemberRejected(t *testing.T) {
	prim, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Space: XMLSchemaNS, Local: "string"}, nil, nil)
	if err != nil {
		t.Fatalf("NewPrimitiveType: %v", err)
	}
	b := NewSchemaBuilder()
	// D's transitive membership is {M, X}; M's base chain is M, then D. Every
	// other property is well formed, so the rejection can only be Phase B's.
	b.AddType(unionType(t, "D", ownedBase(anySimpleType), QName{Local: "M"}))
	b.AddType(unionType(t, "M", SimpleTypeRef{Name: QName{Local: "D"}}, QName{Local: "X"}))
	b.AddType(refType(t, "X", prim.Name()))
	b.AddType(prim)

	_, err = b.Finalize()
	if err == nil {
		t.Fatal("Finalize(a member derived from the union holding it) = nil, want a clause 3.3 rejection")
	}
	if r, _ := xsderr.RuleOf(err); r != ruleCosSTRestricts {
		t.Fatalf("charged %s, want %s", r, ruleCosSTRestricts)
	}
	if !strings.Contains(err.Error(), "cos-st-restricts clause 3.3") {
		t.Fatalf("message does not name clause 3.3: %v", err)
	}
	if !strings.Contains(err.Error(), "derived from") {
		t.Fatalf("message does not say which conjunct failed (the derived-from one): %v", err)
	}
}

// TestUnionMembershipDiamondAccepted guards the other polarity: one type reached
// twice through different member paths is not a cycle, and the walk must not
// report one just because a node is visited more than once.
func TestUnionMembershipDiamondAccepted(t *testing.T) {
	prim, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Space: XMLSchemaNS, Local: "string"}, nil, nil)
	if err != nil {
		t.Fatalf("NewPrimitiveType: %v", err)
	}
	b := NewSchemaBuilder()
	b.AddType(unionType(t, "Top", ownedBase(anySimpleType), QName{Local: "L"}, QName{Local: "R"}))
	b.AddType(unionType(t, "L", ownedBase(anySimpleType), QName{Local: "Leaf"}))
	b.AddType(unionType(t, "R", ownedBase(anySimpleType), QName{Local: "Leaf"}))
	b.AddType(refType(t, "Leaf", prim.Name()))
	b.AddType(prim)

	if _, err := b.Finalize(); err != nil {
		t.Fatalf("Finalize(a diamond membership) = %v, want acceptance", err)
	}
}
