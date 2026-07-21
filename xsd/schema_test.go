package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// simpleTypeNamed builds a named *SimpleType (base xs:anySimpleType) for schema
// symbol-table tests.
func simpleTypeNamed(t *testing.T, name xsd.QName) *xsd.SimpleType {
	t.Helper()
	st, err := xsd.NewSimpleType(xsderr.Loc{}, name, nil, xsd.AnySimpleType(), nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(%v): %v", name, err)
	}
	return st
}

// complexTypeNamed builds a named empty-content ComplexType for schema
// symbol-table tests.
func complexTypeNamed(t *testing.T, name xsd.QName) xsd.ComplexType {
	t.Helper()
	ct, err := xsd.NewComplexType(xsderr.Loc{}, name, xsd.QName{}, nil, xsd.DerivationRestriction, false, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType(%v): %v", name, err)
	}
	return ct
}

// elementNamed builds a global ElementDeclaration for schema symbol-table tests.
func elementNamed(t *testing.T, name xsd.QName) xsd.ElementDeclaration {
	t.Helper()
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, name, xsd.QName{}, nil, xsd.ScopeGlobal, nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration(%v): %v", name, err)
	}
	return e
}

// attributeNamed builds a global AttributeDeclaration for schema symbol-table
// tests.
func attributeNamed(t *testing.T, name xsd.QName) xsd.AttributeDeclaration {
	t.Helper()
	a, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, name, xsd.QName{}, xsd.ScopeGlobal, nil, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeDeclaration(%v): %v", name, err)
	}
	return a
}

// idcNamed builds a keyref-free unique IdentityConstraint for schema
// symbol-table tests.
func idcNamed(t *testing.T, name xsd.QName) xsd.IdentityConstraint {
	t.Helper()
	sel := xsd.NewXPathExpression(".", nil, nil, nil)
	field := xsd.NewXPathExpression("@x", nil, nil, nil)
	c, err := xsd.NewIdentityConstraint(xsderr.Loc{}, name, xsd.IdentityConstraintUnique, sel, []xsd.XPathExpression{field}, nil, nil)
	if err != nil {
		t.Fatalf("NewIdentityConstraint(%v): %v", name, err)
	}
	return c
}

func TestSchemaFinalizeAndQueryHits(t *testing.T) {
	ns := "urn:ns"
	stName := xsd.QName{Space: ns, Local: "st"}
	ctName := xsd.QName{Space: ns, Local: "ct"}
	elName := xsd.QName{Space: ns, Local: "el"}
	atName := xsd.QName{Space: ns, Local: "at"}

	b := xsd.NewSchemaBuilder()
	b.AddType(simpleTypeNamed(t, stName))
	b.AddType(complexTypeNamed(t, ctName))
	b.AddElement(elementNamed(t, elName))
	b.AddAttribute(attributeNamed(t, atName))

	s, err := b.Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}

	if _, ok := s.Type(stName); !ok {
		t.Errorf("Type(%v) miss, want hit", stName)
	}
	if _, ok := s.Type(ctName); !ok {
		t.Errorf("Type(%v) miss, want hit", ctName)
	}
	if _, ok := s.Element(elName); !ok {
		t.Errorf("Element(%v) miss, want hit", elName)
	}
	if _, ok := s.Attribute(atName); !ok {
		t.Errorf("Attribute(%v) miss, want hit", atName)
	}
}

func TestSchemaQueryMisses(t *testing.T) {
	// An empty finalized schema: every lookup is a miss returning the zero
	// component and false.
	s, err := xsd.NewSchemaBuilder().Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	absent := xsd.QName{Space: "urn:ns", Local: "nope"}
	if d, ok := s.Type(absent); ok {
		t.Errorf("Type(%v) = (%v, true), want miss", absent, d)
	}
	if d, ok := s.Element(absent); ok {
		t.Errorf("Element(%v) = (%v, true), want miss", absent, d)
	}
	if d, ok := s.Attribute(absent); ok {
		t.Errorf("Attribute(%v) = (%v, true), want miss", absent, d)
	}
}

func TestSchemaTypeSumAcceptsBothKinds(t *testing.T) {
	ns := "urn:ns"
	stName := xsd.QName{Space: ns, Local: "st"}
	ctName := xsd.QName{Space: ns, Local: "ct"}

	b := xsd.NewSchemaBuilder()
	st := simpleTypeNamed(t, stName)
	b.AddType(st)                          // *SimpleType satisfies TypeDefinition by pointer
	b.AddType(complexTypeNamed(t, ctName)) // ComplexType satisfies it by value

	s, err := b.Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}

	got, ok := s.Type(stName)
	if !ok {
		t.Fatalf("Type(%v) miss, want hit", stName)
	}
	gotST, ok := got.(*xsd.SimpleType)
	if !ok {
		t.Fatalf("Type(%v) concrete = %T, want *xsd.SimpleType", stName, got)
	}
	if gotST != st {
		t.Error("Type(st) returned a different *SimpleType pointer; identity must be preserved (not deep-copied)")
	}

	got, ok = s.Type(ctName)
	if !ok {
		t.Fatalf("Type(%v) miss, want hit", ctName)
	}
	if _, ok := got.(xsd.ComplexType); !ok {
		t.Errorf("Type(%v) concrete = %T, want xsd.ComplexType (by value)", ctName, got)
	}
}

func TestFinalizeRejectsDuplicateTypeName(t *testing.T) {
	// A simple type and a complex type sharing an expanded name are the same
	// {type definitions} kind (§3.17.6.2 clause 1.1 unifies them into one
	// bucket), so this is the sch-props-correct clause 2 collision.
	dup := xsd.QName{Space: "urn:ns", Local: "T"}
	b := xsd.NewSchemaBuilder()
	b.AddType(simpleTypeNamed(t, dup))
	b.AddType(complexTypeNamed(t, dup))
	if _, err := b.Finalize(); err == nil {
		t.Fatal("Finalize(duplicate type name) succeeded, want sch-props-correct error")
	} else {
		assertRule(t, err, "sch-props-correct")
	}
}

func TestFinalizeRejectsDuplicateElementName(t *testing.T) {
	dup := xsd.QName{Space: "urn:ns", Local: "e"}
	b := xsd.NewSchemaBuilder()
	b.AddElement(elementNamed(t, dup))
	b.AddElement(elementNamed(t, dup))
	if _, err := b.Finalize(); err == nil {
		t.Fatal("Finalize(duplicate element name) succeeded, want sch-props-correct error")
	} else {
		assertRule(t, err, "sch-props-correct")
	}
}

func TestFinalizeRejectsDuplicateAttributeName(t *testing.T) {
	dup := xsd.QName{Space: "urn:ns", Local: "a"}
	b := xsd.NewSchemaBuilder()
	b.AddAttribute(attributeNamed(t, dup))
	b.AddAttribute(attributeNamed(t, dup))
	if _, err := b.Finalize(); err == nil {
		t.Fatal("Finalize(duplicate attribute name) succeeded, want sch-props-correct error")
	} else {
		assertRule(t, err, "sch-props-correct")
	}
}

func TestFinalizeRejectsDuplicateIdentityConstraintName(t *testing.T) {
	dup := xsd.QName{Space: "urn:ns", Local: "idc"}
	b := xsd.NewSchemaBuilder()
	b.AddIdentityConstraint(idcNamed(t, dup))
	b.AddIdentityConstraint(idcNamed(t, dup))
	if _, err := b.Finalize(); err == nil {
		t.Fatal("Finalize(duplicate IDC name) succeeded, want sch-props-correct error")
	} else {
		assertRule(t, err, "sch-props-correct")
	}
}

func TestFinalizeDistinctKindsShareNameOK(t *testing.T) {
	// sch-props-correct clause 2 is per-kind: an element and an attribute (and a
	// type) may all share one expanded name without collision.
	name := xsd.QName{Space: "urn:ns", Local: "shared"}
	b := xsd.NewSchemaBuilder()
	b.AddType(complexTypeNamed(t, name))
	b.AddElement(elementNamed(t, name))
	b.AddAttribute(attributeNamed(t, name))
	if _, err := b.Finalize(); err != nil {
		t.Fatalf("Finalize(distinct kinds share name): %v", err)
	}
}

func TestFinalizeDecouplesBuilderFromSchema(t *testing.T) {
	// The builder must remain independently usable after Finalize: adding more
	// components to it does not mutate an already-returned Schema (fresh backing
	// arrays, indexes built at Finalize).
	ns := "urn:ns"
	first := xsd.QName{Space: ns, Local: "first"}
	second := xsd.QName{Space: ns, Local: "second"}

	b := xsd.NewSchemaBuilder()
	b.AddElement(elementNamed(t, first))
	s1, err := b.Finalize()
	if err != nil {
		t.Fatalf("Finalize #1: %v", err)
	}

	b.AddElement(elementNamed(t, second))
	if _, ok := s1.Element(second); ok {
		t.Error("s1.Element(second) hit; the first Schema must not see components added after its Finalize")
	}

	s2, err := b.Finalize()
	if err != nil {
		t.Fatalf("Finalize #2: %v", err)
	}
	if _, ok := s2.Element(first); !ok {
		t.Error("s2.Element(first) miss; the second Schema must carry all accumulated components")
	}
	if _, ok := s2.Element(second); !ok {
		t.Error("s2.Element(second) miss; the second Schema must carry the newly added component")
	}
}

// attributeGroupNamed builds a minimal attribute group definition for the Add*
// wrapper tests.
func attributeGroupNamed(t *testing.T, name xsd.QName) xsd.AttributeGroupDefinition {
	t.Helper()
	g, err := xsd.NewAttributeGroupDefinition(xsderr.Loc{}, name, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewAttributeGroupDefinition(%v): %v", name, err)
	}
	return g
}

// modelGroupNamed builds a minimal (empty-sequence) model group definition for
// the Add* wrapper tests.
func modelGroupNamed(t *testing.T, name xsd.QName) xsd.ModelGroupDefinition {
	t.Helper()
	g, err := xsd.NewModelGroup(xsderr.Loc{}, xsd.CompositorSequence, nil, nil)
	if err != nil {
		t.Fatalf("NewModelGroup: %v", err)
	}
	d, err := xsd.NewModelGroupDefinition(xsderr.Loc{}, name, g, nil)
	if err != nil {
		t.Fatalf("NewModelGroupDefinition(%v): %v", name, err)
	}
	return d
}

// notationNamed builds a minimal notation declaration for the Add* wrapper tests.
func notationNamed(t *testing.T, name xsd.QName) xsd.Notation {
	t.Helper()
	sys := "urn:sys"
	n, err := xsd.NewNotation(xsderr.Loc{}, name, &sys, nil, nil)
	if err != nil {
		t.Fatalf("NewNotation(%v): %v", name, err)
	}
	return n
}

// TestAddWrappersAcceptedByFinalize exercises the four append-only builder
// wrappers (AddAttributeGroup/AddModelGroup/AddNotation/AddAnnotation): each
// component a wrapper adds must survive Finalize (the resolution pass must not
// reject a well-formed one). None of these four kinds has an exported *Schema
// lookup accessor yet, so observability is: Finalize succeeds.
func TestAddWrappersAcceptedByFinalize(t *testing.T) {
	name := xsd.QName{Space: "urn:ns", Local: "w"}
	cases := []struct {
		label string
		add   func(b *xsd.SchemaBuilder)
	}{
		{"AddAttributeGroup", func(b *xsd.SchemaBuilder) { b.AddAttributeGroup(attributeGroupNamed(t, name)) }},
		{"AddModelGroup", func(b *xsd.SchemaBuilder) { b.AddModelGroup(modelGroupNamed(t, name)) }},
		{"AddNotation", func(b *xsd.SchemaBuilder) { b.AddNotation(notationNamed(t, name)) }},
		{"AddAnnotation", func(b *xsd.SchemaBuilder) { b.AddAnnotation(xsd.NewAnnotation(nil, nil, nil)) }},
	}
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			b := xsd.NewSchemaBuilder()
			c.add(b)
			if _, err := b.Finalize(); err != nil {
				t.Fatalf("Finalize after %s: %v", c.label, err)
			}
		})
	}
}

// TestAddModelGroupObservableViaResolution proves an added model group
// definition is actually indexed (not silently dropped): a <group ref> to it
// resolves, so Finalize succeeds; without AddModelGroup wiring the ref would
// dangle and be rejected.
func TestAddModelGroupObservableViaResolution(t *testing.T) {
	target := xsd.QName{Space: "urn:ns", Local: "target"}
	refParticle, err := xsd.NewParticle(xsderr.Loc{}, mustOccurs11(t), xsd.ModelGroupRef{Name: target}, nil)
	if err != nil {
		t.Fatalf("NewParticle: %v", err)
	}
	refGroup, err := xsd.NewModelGroup(xsderr.Loc{}, xsd.CompositorSequence, []xsd.Particle{refParticle}, nil)
	if err != nil {
		t.Fatalf("NewModelGroup: %v", err)
	}
	referrer, err := xsd.NewModelGroupDefinition(xsderr.Loc{}, xsd.QName{Space: "urn:ns", Local: "referrer"}, refGroup, nil)
	if err != nil {
		t.Fatalf("NewModelGroupDefinition: %v", err)
	}

	b := xsd.NewSchemaBuilder()
	b.AddModelGroup(modelGroupNamed(t, target))
	b.AddModelGroup(referrer)
	if _, err := b.Finalize(); err != nil {
		t.Fatalf("Finalize(group ref to an added model group): %v", err)
	}
}

// TestAddWrapperDuplicateRejected proves the appended components are indexed by
// expanded name: two attribute groups, two model groups, or two notations
// sharing a name collide under sch-props-correct clause 2.
func TestAddWrapperDuplicateRejected(t *testing.T) {
	dup := xsd.QName{Space: "urn:ns", Local: "dup"}
	cases := []struct {
		label string
		add   func(b *xsd.SchemaBuilder)
	}{
		{"AddAttributeGroup", func(b *xsd.SchemaBuilder) { b.AddAttributeGroup(attributeGroupNamed(t, dup)) }},
		{"AddModelGroup", func(b *xsd.SchemaBuilder) { b.AddModelGroup(modelGroupNamed(t, dup)) }},
		{"AddNotation", func(b *xsd.SchemaBuilder) { b.AddNotation(notationNamed(t, dup)) }},
	}
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			b := xsd.NewSchemaBuilder()
			c.add(b)
			c.add(b)
			if _, err := b.Finalize(); err == nil {
				t.Fatalf("Finalize(duplicate %s name) succeeded, want sch-props-correct error", c.label)
			} else {
				assertRule(t, err, "sch-props-correct")
			}
		})
	}
}

// mustOccurs11 builds the {1,1} occurrence range for schema_test helpers.
func mustOccurs11(t *testing.T) xsd.Occurs {
	t.Helper()
	o, err := xsd.NewOccurs(xsderr.Loc{}, 1, 1)
	if err != nil {
		t.Fatalf("NewOccurs: %v", err)
	}
	return o
}

func TestAddTypeNilInterfacePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("AddType(nil) did not panic")
		}
	}()
	xsd.NewSchemaBuilder().AddType(nil)
}

func TestAddTypeNilSimpleTypePanics(t *testing.T) {
	// A non-nil TypeDefinition interface wrapping a nil *SimpleType is still a
	// nil type definition and must panic.
	defer func() {
		if recover() == nil {
			t.Error("AddType((*SimpleType)(nil)) did not panic")
		}
	}()
	var st *xsd.SimpleType
	xsd.NewSchemaBuilder().AddType(st)
}
