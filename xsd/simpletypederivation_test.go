package xsd

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests pin checkSimpleTypeDerivations' SIX-SLOT descent inventory
// (resolve.go) from inside the package, where a stub capability can stand in for
// the real cos-st-restricts charge: package xsd is a pure leaf and cannot reach
// the generated per-primitive applicability table, so the rule ID and message a
// real checker produces are pinned from package builtin instead
// (builtin/restrictionchecker_test.go). What is pinned HERE is exactly what this
// package owns: which simple types the walk reaches.

// stubRestrictionChecker rejects one designated *SimpleType and records every
// type it was asked about, so a test can pin BOTH that a slot is descended and
// that nothing else changed the verdict.
type stubRestrictionChecker struct {
	reject *SimpleType
	seen   []*SimpleType
}

func (c *stubRestrictionChecker) CheckRestriction(t *SimpleType) error {
	c.seen = append(c.seen, t)
	if c.reject != nil && t == c.reject {
		return xsderr.New(xsderr.Rule("cos-st-restricts"), t.Loc(),
			"stub restriction checker rejects the target simple type")
	}
	return nil
}

// sawTarget reports whether the walk put the designated type to the checker.
func (c *stubRestrictionChecker) sawTarget() bool {
	for _, t := range c.seen {
		if t == c.reject {
			return true
		}
	}
	return false
}

// TestCheckSimpleTypeDerivationsDescendsEverySlot is the inventory test: for each
// of the six slots a *SimpleType can occupy, a schema is assembled in which the
// TARGET type occupies that slot and nothing else, and finalize must reject it.
//
// Slots 2 through 6 hold an ANONYMOUS target — its {name} is the zero QName, so
// no §3.17.1 symbol table can hold it and Schema.Type can never return it. That
// is what makes each of those cases fail against an s.types-only walk: the type
// is in no index, so an index-only walk never sees it and the schema is accepted.
// Every case asserts the anonymity, so the test cannot silently degrade into six
// copies of slot 1.
func TestCheckSimpleTypeDerivationsDescendsEverySlot(t *testing.T) {
	for _, tc := range []struct {
		slot      string
		anonymous bool
		build     func(t *testing.T, b *SchemaBuilder, target *SimpleType)
	}{
		{
			slot:      "1: s.types root",
			anonymous: false,
			build: func(t *testing.T, b *SchemaBuilder, target *SimpleType) {
				b.AddType(target)
			},
		},
		{
			slot:      "2: the {base type definition} hop",
			anonymous: true,
			build: func(t *testing.T, b *SchemaBuilder, target *SimpleType) {
				b.AddType(dSimple(t, uq("derived"), target))
			},
		},
		{
			slot:      "3: SimpleContent {simple type definition}",
			anonymous: true,
			build: func(t *testing.T, b *SchemaBuilder, target *SimpleType) {
				b.AddType(dType(t, uq("ct"), anyTypeName, SimpleContent{SimpleType: target}, nil, nil))
			},
		},
		{
			slot:      "4: ListDerivation.Item",
			anonymous: true,
			build: func(t *testing.T, b *SchemaBuilder, target *SimpleType) {
				b.AddType(dList(t, uq("list"), target))
			},
		},
		{
			slot:      "5: UnionDerivation.Members",
			anonymous: true,
			build: func(t *testing.T, b *SchemaBuilder, target *SimpleType) {
				b.AddType(dUnion(t, uq("union"), target))
			},
		},
		{
			slot:      "6: an element declaration's inline {type definition}",
			anonymous: true,
			build: func(t *testing.T, b *SchemaBuilder, target *SimpleType) {
				b.AddElement(dInlineElement(t, uq("e"), target))
			},
		},
	} {
		t.Run(tc.slot, func(t *testing.T) {
			prim := dPrimitive(t, uq("prim"))
			name := QName{}
			if !tc.anonymous {
				name = uq("target")
			}
			target, err := NewSimpleType(xsderr.Loc{}, name, RestrictionDerivation{}, prim, nil, nil)
			if err != nil {
				t.Fatalf("NewSimpleType(target): %v", err)
			}
			if tc.anonymous && target.Name() != (QName{}) {
				t.Fatal("the target must be anonymous, or the case degenerates into slot 1")
			}

			// The control run proves the schema is otherwise valid, so the
			// rejection below comes from the target and not from the fixture.
			control := &stubRestrictionChecker{}
			b := NewSchemaBuilder()
			b.AddType(dAnyType(t))
			b.AddType(prim)
			tc.build(t, b, target)
			s, err := b.FinalizeWith(undecidedValueSpace{}, control)
			if err != nil {
				t.Fatalf("the fixture must finalize when nothing is rejected: %v", err)
			}
			if tc.anonymous {
				if _, ok := s.Type(QName{}); ok {
					t.Fatal("an anonymous type must not be reachable by name")
				}
			}

			checker := &stubRestrictionChecker{reject: target}
			b = NewSchemaBuilder()
			b.AddType(dAnyType(t))
			b.AddType(prim)
			tc.build(t, b, target)
			if _, err := b.FinalizeWith(undecidedValueSpace{}, checker); err == nil {
				t.Fatalf("the target in slot %q was never put to the checker (seen %d types)", tc.slot, len(checker.seen))
			}
			if !checker.sawTarget() {
				t.Fatal("finalize rejected without the target reaching the checker")
			}
		})
	}
}

// TestCheckSimpleTypeDerivationsChecksBaseBeforeDerived pins the bottom-up order
// checkSimpleTypeDerivations' doc promises: a base is charged before every type
// deriving from it, so a fault in a base is reported against the base rather than
// against everything below it.
func TestCheckSimpleTypeDerivationsChecksBaseBeforeDerived(t *testing.T) {
	prim := dPrimitive(t, uq("prim"))
	mid := dSimple(t, uq("mid"), prim)
	leaf := dSimple(t, uq("leaf"), mid)

	checker := &stubRestrictionChecker{}
	b := NewSchemaBuilder()
	b.AddType(dAnyType(t))
	b.AddType(leaf) // the DERIVED type is added first, so document order cannot supply the answer
	if _, err := b.FinalizeWith(undecidedValueSpace{}, checker); err != nil {
		t.Fatalf("FinalizeWith: %v", err)
	}
	if got := indexOfType(checker.seen, mid); got < 0 || got > indexOfType(checker.seen, leaf) {
		t.Fatalf("the base was charged at position %d and the derived type at %d; the base must come first",
			got, indexOfType(checker.seen, leaf))
	}
	if indexOfType(checker.seen, prim) > indexOfType(checker.seen, mid) {
		t.Fatal("the primitive at the foot of the chain must be charged before the type restricting it")
	}
}

// TestFinalizeInstallsUndecidedRestrictionChecker pins that plain Finalize
// installs the undecided checker rather than nil: a schema whose simple types
// would be rejected by an installed checker is ACCEPTED with none, and no nil
// dereference occurs.
func TestFinalizeInstallsUndecidedRestrictionChecker(t *testing.T) {
	prim := dPrimitive(t, uq("prim"))
	build := func(b *SchemaBuilder) {
		b.AddType(dAnyType(t))
		b.AddType(prim)
		b.AddType(dSimple(t, uq("derived"), prim))
	}
	b := NewSchemaBuilder()
	build(b)
	if _, err := b.Finalize(); err != nil {
		t.Fatalf("Finalize must charge no cos-st-restricts facet-value clause at all: %v", err)
	}
	b = NewSchemaBuilder()
	build(b)
	if _, err := b.FinalizeWith(undecidedValueSpace{}, &stubRestrictionChecker{reject: prim}); err == nil {
		t.Fatal("FinalizeWith a rejecting checker must reject the same schema")
	}
}

// indexOfType reports where want first appears in seen, or -1.
func indexOfType(seen []*SimpleType, want *SimpleType) int {
	for i, t := range seen {
		if t == want {
			return i
		}
	}
	return -1
}

// dList builds a named list simple type over item, the ·constructed· shape
// cos-st-restricts clause 2.2.1 gives a <list>: the {base type definition} is
// xs:anySimpleType and the ListDerivation mints the {item type definition}. The
// fixed collapse whiteSpace facet is the one clause 2.2.1.2 requires of a list
// constructed directly from xs:anySimpleType (§4.3.6.1).
func dList(t *testing.T, name QName, item *SimpleType) *SimpleType {
	t.Helper()
	st, err := NewSimpleType(xsderr.Loc{}, name, ListDerivation{Item: item}, anySimpleType,
		[]Facet{NewFacet(FacetWhiteSpace, []string{"collapse"}, true)}, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(list %s): %v", name, err)
	}
	return st
}

// dUnion builds a named union simple type over members, the ·constructed· shape
// cos-st-restricts clause 3.2.1 gives a <union>.
func dUnion(t *testing.T, name QName, members ...*SimpleType) *SimpleType {
	t.Helper()
	st, err := NewSimpleType(xsderr.Loc{}, name, UnionDerivation{Members: members}, anySimpleType, nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(union %s): %v", name, err)
	}
	return st
}

// dInlineElement builds a global element declaration whose {type definition} is
// the given simple type written out in place — the InlineTypeDefinition arm that
// is inventory slot 6.
func dInlineElement(t *testing.T, name QName, inline *SimpleType) ElementDeclaration {
	t.Helper()
	e, err := NewElementDeclaration(xsderr.Loc{}, name, InlineTypeDefinition{Definition: inline},
		nil, NewGlobalScope(), nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration(%s): %v", name, err)
	}
	return e
}
