package xsd_test

import (
	"errors"
	"strings"
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
	ct, err := xsd.NewComplexType(xsderr.Loc{}, name, xsd.QName{}, nil, xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType(%v): %v", name, err)
	}
	return ct
}

// elementNamed builds a global ElementDeclaration with an unknown source
// position for schema symbol-table tests.
func elementNamed(t *testing.T, name xsd.QName) xsd.ElementDeclaration {
	t.Helper()
	return elementNamedAt(t, xsderr.Loc{}, name)
}

// elementNamedAt builds a global ElementDeclaration carrying loc as its source
// position, for the tests that assert on the position a rejection cites.
func elementNamedAt(t *testing.T, loc xsderr.Loc, name xsd.QName) xsd.ElementDeclaration {
	t.Helper()
	e, err := xsd.NewElementDeclaration(loc, name, nil, nil, xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration(%v): %v", name, err)
	}
	return e
}

// attributeNamed builds a global AttributeDeclaration for schema symbol-table
// tests.
func attributeNamed(t *testing.T, name xsd.QName) xsd.AttributeDeclaration {
	t.Helper()
	a, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, name, nil, xsd.ScopeGlobal, nil, false, nil)
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

// TestFinalizeAnonymousTypesDoNotCollide proves two or more type definitions
// with an ABSENT {name} — the zero QName — finalize cleanly instead of being
// charged a sch-props-correct clause 2 duplicate. §3.4.1 and §3.16.1 exempt
// anonymous type definitions ("those with no {name}") from the uniqueness
// requirement clause 2 enforces, so absent-name components are excluded from
// the by-name index entirely rather than all hashing to one key.
func TestFinalizeAnonymousTypesDoNotCollide(t *testing.T) {
	anon := xsd.QName{} // {name} absent (§3.4.2.1/§3.16.2.1)
	cases := []struct {
		label string
		add   func(b *xsd.SchemaBuilder)
	}{
		{"ComplexType", func(b *xsd.SchemaBuilder) { b.AddType(complexTypeNamed(t, anon)) }},
		{"SimpleType", func(b *xsd.SchemaBuilder) { b.AddType(simpleTypeNamed(t, anon)) }},
	}
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			b := xsd.NewSchemaBuilder()
			c.add(b)
			c.add(b)
			c.add(b)
			if _, err := b.Finalize(); err != nil {
				t.Fatalf("Finalize(3 anonymous %s) = %v, want success: anonymous type definitions are exempt from sch-props-correct clause 2", c.label, err)
			}
		})
	}
	t.Run("MixedKinds", func(t *testing.T) {
		// An anonymous simple type and an anonymous complex type share the one
		// {type definitions} bucket, so they would have collided too.
		b := xsd.NewSchemaBuilder()
		b.AddType(simpleTypeNamed(t, anon))
		b.AddType(complexTypeNamed(t, anon))
		if _, err := b.Finalize(); err != nil {
			t.Fatalf("Finalize(anonymous simple + anonymous complex type) = %v, want success", err)
		}
	})
}

// TestSchemaTypeAbsentNameMisses proves the absent {name} is never a lookup key:
// after anonymous type definitions are added, Type(QName{}) is a MISS. An
// anonymous type is not in §3.17.1's {type definitions} symbol table, so it must
// not be reachable by name — a hit here would leak the last-indexed anonymous
// component to any caller that happened to resolve an absent QName.
func TestSchemaTypeAbsentNameMisses(t *testing.T) {
	b := xsd.NewSchemaBuilder()
	b.AddType(complexTypeNamed(t, xsd.QName{}))
	b.AddType(complexTypeNamed(t, xsd.QName{}))
	b.AddType(complexTypeNamed(t, xsd.QName{Space: "urn:ns", Local: "named"}))

	s, err := b.Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if d, ok := s.Type(xsd.QName{}); ok {
		t.Errorf("Type(QName{}) = (%v, true), want miss: an absent {name} is never a key", d)
	}
	if _, ok := s.Type(xsd.QName{Space: "urn:ns", Local: "named"}); !ok {
		t.Error("Type(urn:ns:named) miss, want hit: named components stay indexed")
	}
}

// TestTopLevelComponentsRetainLoc proves every top-level kind retains the
// source position its constructor was handed and reports it through Loc — the
// provenance the sch-props-correct clause 2 rejection cites. NewPrimitiveType
// is covered alongside NewSimpleType: both build a *SimpleType, so both must
// retain loc.
func TestTopLevelComponentsRetainLoc(t *testing.T) {
	name := xsd.QName{Space: "urn:ns", Local: "c"}
	want := xsderr.Loc{URI: "s.xsd", Line: 12, Col: 4}

	primitiveAt := func(t *testing.T, loc xsderr.Loc) *xsd.SimpleType {
		t.Helper()
		st, err := xsd.NewPrimitiveType(loc, name, nil, nil)
		if err != nil {
			t.Fatalf("NewPrimitiveType: %v", err)
		}
		return st
	}
	cases := []struct {
		kind string
		loc  func(t *testing.T, loc xsderr.Loc) xsderr.Loc
	}{
		{"ElementDeclaration", func(t *testing.T, l xsderr.Loc) xsderr.Loc { return elementNamedAt(t, l, name).Loc() }},
		{"AttributeDeclaration", func(t *testing.T, l xsderr.Loc) xsderr.Loc {
			a, err := xsd.NewAttributeDeclaration(l, name, nil, xsd.ScopeGlobal, nil, false, nil)
			if err != nil {
				t.Fatalf("NewAttributeDeclaration: %v", err)
			}
			return a.Loc()
		}},
		{"ComplexType", func(t *testing.T, l xsderr.Loc) xsderr.Loc {
			c, err := xsd.NewComplexType(l, name, xsd.QName{}, nil, xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
			if err != nil {
				t.Fatalf("NewComplexType: %v", err)
			}
			return c.Loc()
		}},
		{"SimpleType", func(t *testing.T, l xsderr.Loc) xsderr.Loc {
			st, err := xsd.NewSimpleType(l, name, nil, xsd.AnySimpleType(), nil, nil)
			if err != nil {
				t.Fatalf("NewSimpleType: %v", err)
			}
			return st.Loc()
		}},
		{"PrimitiveType", func(t *testing.T, l xsderr.Loc) xsderr.Loc { return primitiveAt(t, l).Loc() }},
		{"ModelGroupDefinition", func(t *testing.T, l xsderr.Loc) xsderr.Loc {
			g, err := xsd.NewModelGroup(xsderr.Loc{}, xsd.CompositorSequence, nil, nil)
			if err != nil {
				t.Fatalf("NewModelGroup: %v", err)
			}
			d, err := xsd.NewModelGroupDefinition(l, name, g, nil)
			if err != nil {
				t.Fatalf("NewModelGroupDefinition: %v", err)
			}
			return d.Loc()
		}},
		{"AttributeGroupDefinition", func(t *testing.T, l xsderr.Loc) xsderr.Loc {
			g, err := xsd.NewAttributeGroupDefinition(l, name, nil, nil, nil)
			if err != nil {
				t.Fatalf("NewAttributeGroupDefinition: %v", err)
			}
			return g.Loc()
		}},
		{"Notation", func(t *testing.T, l xsderr.Loc) xsderr.Loc {
			sys := "urn:sys"
			n, err := xsd.NewNotation(l, name, &sys, nil, nil)
			if err != nil {
				t.Fatalf("NewNotation: %v", err)
			}
			return n.Loc()
		}},
		{"IdentityConstraint", func(t *testing.T, l xsderr.Loc) xsderr.Loc {
			sel := xsd.NewXPathExpression(".", nil, nil, nil)
			field := xsd.NewXPathExpression("@x", nil, nil, nil)
			c, err := xsd.NewIdentityConstraint(l, name, xsd.IdentityConstraintUnique, sel, []xsd.XPathExpression{field}, nil, nil)
			if err != nil {
				t.Fatalf("NewIdentityConstraint: %v", err)
			}
			return c.Loc()
		}},
	}
	for _, c := range cases {
		t.Run(c.kind, func(t *testing.T) {
			if got := c.loc(t, want); got != want {
				t.Errorf("%s.Loc() = %v, want the constructor's loc %v", c.kind, got, want)
			}
			// A component built with no parser position reports the zero Loc,
			// which renders as "?" — the documented "unknown" reading.
			if got := c.loc(t, xsderr.Loc{}); got != (xsderr.Loc{}) {
				t.Errorf("%s.Loc() = %v for a zero-loc build, want the zero Loc", c.kind, got)
			}
		})
	}
}

// TestTypeDefinitionSumPromotesLoc proves Loc is reachable through the
// TypeDefinition sum itself — the property that keeps the unified {type
// definitions} bucket off a type switch — for both variants.
func TestTypeDefinitionSumPromotesLoc(t *testing.T) {
	stLoc := xsderr.Loc{URI: "s.xsd", Line: 2, Col: 1}
	ctLoc := xsderr.Loc{URI: "s.xsd", Line: 7, Col: 1}
	st, err := xsd.NewSimpleType(stLoc, xsd.QName{Space: "urn:ns", Local: "st"}, nil, xsd.AnySimpleType(), nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType: %v", err)
	}
	ct, err := xsd.NewComplexType(ctLoc, xsd.QName{Space: "urn:ns", Local: "ct"}, xsd.QName{}, nil, xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType: %v", err)
	}
	for _, c := range []struct {
		label string
		td    xsd.TypeDefinition
		want  xsderr.Loc
	}{
		{"*SimpleType", st, stLoc},
		{"ComplexType", ct, ctLoc},
	} {
		if got := c.td.Loc(); got != c.want {
			t.Errorf("TypeDefinition(%s).Loc() = %v, want %v", c.label, got, c.want)
		}
	}
}

// TestFinalizeDuplicateNameCitesLocs proves the sch-props-correct clause 2
// rejection is charged to the LATER (duplicate) component's own source position
// — not the zero xsderr.Loc — and names the first occurrence's position in the
// message, so a reader is pointed at the line to edit and at the line it
// collides with.
func TestFinalizeDuplicateNameCitesLocs(t *testing.T) {
	dup := xsd.QName{Space: "urn:ns", Local: "e"}
	firstLoc := xsderr.Loc{URI: "s.xsd", Line: 3, Col: 5}
	dupLoc := xsderr.Loc{URI: "s.xsd", Line: 9, Col: 7}

	b := xsd.NewSchemaBuilder()
	b.AddElement(elementNamedAt(t, firstLoc, dup))
	b.AddElement(elementNamedAt(t, dupLoc, dup))

	_, err := b.Finalize()
	if err == nil {
		t.Fatal("Finalize(duplicate element name) succeeded, want sch-props-correct error")
	}
	assertRule(t, err, "sch-props-correct")

	var e *xsderr.Error
	if !errors.As(err, &e) {
		t.Fatalf("error %v is not an *xsderr.Error", err)
	}
	if e.Loc != dupLoc {
		t.Errorf("rejection Loc = %v, want the duplicate component's own position %v", e.Loc, dupLoc)
	}
	if !strings.Contains(e.Msg, firstLoc.String()) {
		t.Errorf("rejection message %q does not name the first occurrence's position %v", e.Msg, firstLoc)
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
// reject a well-formed one). None of these four kinds has an exported by-name
// *Schema lookup accessor, so observability here is: Finalize succeeds
// (TestSchemaEnumeratorsDocumentOrder covers the document-order enumerators).
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

// annotationDoc builds a schema-level annotation carrying one <documentation>
// whose content identifies it: an Annotation has no {name}, so that content is
// how the enumeration tests tell two of them apart.
func annotationDoc(content string) xsd.Annotation {
	return xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, content)}, nil)
}

// enumerationSchema finalizes a Schema holding two components of every §3.17.1
// property, added in the order the document-order enumerators must report.
func enumerationSchema(t *testing.T) *xsd.Schema {
	t.Helper()
	b := xsd.NewSchemaBuilder()
	b.AddType(simpleTypeNamed(t, qn("t1")))
	b.AddType(complexTypeNamed(t, qn("t2")))
	b.AddElement(elementNamed(t, qn("e1")))
	b.AddElement(elementNamed(t, qn("e2")))
	b.AddAttribute(attributeNamed(t, qn("a1")))
	b.AddAttribute(attributeNamed(t, qn("a2")))
	b.AddAttributeGroup(attributeGroupNamed(t, qn("ag1")))
	b.AddAttributeGroup(attributeGroupNamed(t, qn("ag2")))
	b.AddModelGroup(modelGroupNamed(t, qn("mg1")))
	b.AddModelGroup(modelGroupNamed(t, qn("mg2")))
	b.AddNotation(notationNamed(t, qn("n1")))
	b.AddNotation(notationNamed(t, qn("n2")))
	b.AddIdentityConstraint(idcNamed(t, qn("c1")))
	b.AddIdentityConstraint(idcNamed(t, qn("c2")))
	b.AddAnnotation(annotationDoc("first"))
	b.AddAnnotation(annotationDoc("second"))

	s, err := b.Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	return s
}

// enumeratedNames projects one enumerator's result onto the local names it
// reports, so an order assertion reads as a plain string slice.
func enumeratedNames[T any](items []T, name func(T) xsd.QName) []string {
	locals := make([]string, len(items))
	for i, item := range items {
		locals[i] = name(item).Local
	}
	return locals
}

func assertOrder(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %v, want %v", label, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s = %v, want %v", label, got, want)
		}
	}
}

// TestSchemaEnumeratorsDocumentOrder pins the §3.17.1 enumerators' contract:
// each reports its property's components in the order they were added to the
// builder, which is a guarantee even though seven of the eight properties are
// worded as unordered sets.
func TestSchemaEnumeratorsDocumentOrder(t *testing.T) {
	s := enumerationSchema(t)

	assertOrder(t, "Types", enumeratedNames(s.Types(), xsd.TypeDefinition.Name), []string{"t1", "t2"})
	assertOrder(t, "Elements", enumeratedNames(s.Elements(), xsd.ElementDeclaration.Name), []string{"e1", "e2"})
	assertOrder(t, "Attributes", enumeratedNames(s.Attributes(), xsd.AttributeDeclaration.Name), []string{"a1", "a2"})
	assertOrder(t, "AttributeGroups", enumeratedNames(s.AttributeGroups(), xsd.AttributeGroupDefinition.Name), []string{"ag1", "ag2"})
	assertOrder(t, "ModelGroups", enumeratedNames(s.ModelGroups(), xsd.ModelGroupDefinition.Name), []string{"mg1", "mg2"})
	assertOrder(t, "Notations", enumeratedNames(s.Notations(), xsd.Notation.Name), []string{"n1", "n2"})
	assertOrder(t, "IdentityConstraints", enumeratedNames(s.IdentityConstraints(), xsd.IdentityConstraint.Name), []string{"c1", "c2"})

	annotations := s.Annotations()
	if len(annotations) != 2 {
		t.Fatalf("Annotations() length = %d, want 2", len(annotations))
	}
	assertOrder(t, "Annotations",
		[]string{annotations[0].Documentation()[0].Content(), annotations[1].Documentation()[0].Content()},
		[]string{"first", "second"})
}

// assertEnumeratorCopies proves an enumerator hands back a fresh slice: zeroing
// the returned slice's first element must not change what the next call
// reports. Only the SLICE is copied — key reads a shared, immutable component.
func assertEnumeratorCopies[T any](t *testing.T, label string, get func() []T, key func(T) string) {
	t.Helper()
	first := get()
	if len(first) == 0 {
		t.Fatalf("%s returned nothing; the copy check needs at least one component", label)
	}
	want := key(first[0])
	var zero T
	first[0] = zero
	if got := key(get()[0]); got != want {
		t.Errorf("%s: mutating the returned slice changed the Schema's own; got %q, want %q", label, got, want)
	}
}

func TestSchemaEnumeratorsReturnCopies(t *testing.T) {
	s := enumerationSchema(t)

	// The Types/Annotations keys tolerate the zero value this helper writes back,
	// so a broken copy reports a mismatch rather than panicking.
	assertEnumeratorCopies(t, "Types", s.Types, func(d xsd.TypeDefinition) string {
		if d == nil {
			return "<nil>"
		}
		return d.Name().Local
	})
	assertEnumeratorCopies(t, "Elements", s.Elements, func(d xsd.ElementDeclaration) string { return d.Name().Local })
	assertEnumeratorCopies(t, "Attributes", s.Attributes, func(d xsd.AttributeDeclaration) string { return d.Name().Local })
	assertEnumeratorCopies(t, "AttributeGroups", s.AttributeGroups, func(d xsd.AttributeGroupDefinition) string { return d.Name().Local })
	assertEnumeratorCopies(t, "ModelGroups", s.ModelGroups, func(d xsd.ModelGroupDefinition) string { return d.Name().Local })
	assertEnumeratorCopies(t, "Notations", s.Notations, func(d xsd.Notation) string { return d.Name().Local })
	assertEnumeratorCopies(t, "IdentityConstraints", s.IdentityConstraints, func(d xsd.IdentityConstraint) string { return d.Name().Local })
	assertEnumeratorCopies(t, "Annotations", s.Annotations, func(a xsd.Annotation) string {
		docs := a.Documentation()
		if len(docs) == 0 {
			return "<none>"
		}
		return docs[0].Content()
	})
}

// TestSchemaTypesIncludesAnonymous pins the half of Types()'s contract the
// by-name index cannot show: an anonymous top-level type definition ({name}
// absent) is exempt from the §3.17.1 symbol tables (§3.4.1, §3.16.1) and so is
// unreachable through Type, yet it IS part of {type definitions}.
func TestSchemaTypesIncludesAnonymous(t *testing.T) {
	anon, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{}, nil, xsd.AnySimpleType(), nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(anonymous): %v", err)
	}
	b := xsd.NewSchemaBuilder()
	b.AddType(simpleTypeNamed(t, qn("named")))
	b.AddType(anon)
	s, err := b.Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}

	assertOrder(t, "Types", enumeratedNames(s.Types(), xsd.TypeDefinition.Name), []string{"named", ""})
	if s.Types()[1] != xsd.TypeDefinition(anon) {
		t.Error("Types()[1] is not the anonymous *SimpleType that was added; pointer identity must be preserved")
	}
	if d, ok := s.Type(xsd.QName{}); ok {
		t.Errorf("Type(absent name) = (%v, true), want miss: an anonymous type belongs to no symbol table", d)
	}
}

func TestSchemaEnumeratorsEmptyAreNil(t *testing.T) {
	s, err := xsd.NewSchemaBuilder().Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	if got := s.Types(); got != nil {
		t.Errorf("Types() = %v, want nil", got)
	}
	if got := s.Elements(); got != nil {
		t.Errorf("Elements() = %v, want nil", got)
	}
	if got := s.Attributes(); got != nil {
		t.Errorf("Attributes() = %v, want nil", got)
	}
	if got := s.AttributeGroups(); got != nil {
		t.Errorf("AttributeGroups() = %v, want nil", got)
	}
	if got := s.ModelGroups(); got != nil {
		t.Errorf("ModelGroups() = %v, want nil", got)
	}
	if got := s.Notations(); got != nil {
		t.Errorf("Notations() = %v, want nil", got)
	}
	if got := s.IdentityConstraints(); got != nil {
		t.Errorf("IdentityConstraints() = %v, want nil", got)
	}
	if got := s.Annotations(); got != nil {
		t.Errorf("Annotations() = %v, want nil", got)
	}
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
