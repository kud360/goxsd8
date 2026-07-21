package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

const rns = "urn:resolve"

func qn(local string) xsd.QName { return xsd.QName{Space: rns, Local: local} }

// occurs11 builds the {1,1} occurrence range used throughout these tests.
func occurs11(t *testing.T) xsd.Occurs {
	t.Helper()
	o, err := xsd.NewOccurs(xsderr.Loc{}, 1, 1)
	if err != nil {
		t.Fatalf("NewOccurs: %v", err)
	}
	return o
}

// termParticle wraps a TermOrRef in a {1,1} particle.
func termParticle(t *testing.T, term xsd.TermOrRef) xsd.Particle {
	t.Helper()
	p, err := xsd.NewParticle(xsderr.Loc{}, occurs11(t), term, nil)
	if err != nil {
		t.Fatalf("NewParticle: %v", err)
	}
	return p
}

// seqGroup builds a sequence model group over the given particles.
func seqGroup(t *testing.T, particles ...xsd.Particle) xsd.ModelGroup {
	t.Helper()
	g, err := xsd.NewModelGroup(xsderr.Loc{}, xsd.CompositorSequence, particles, nil)
	if err != nil {
		t.Fatalf("NewModelGroup: %v", err)
	}
	return g
}

// modelGroupDef builds a named model group definition over a sequence of the
// given particles.
func modelGroupDef(t *testing.T, name xsd.QName, particles ...xsd.Particle) xsd.ModelGroupDefinition {
	t.Helper()
	d, err := xsd.NewModelGroupDefinition(xsderr.Loc{}, name, seqGroup(t, particles...), nil)
	if err != nil {
		t.Fatalf("NewModelGroupDefinition: %v", err)
	}
	return d
}

// elementContentCT builds a named element-only complex type whose single
// particle carries term.
func elementContentCT(t *testing.T, name xsd.QName, term xsd.TermOrRef) xsd.ComplexType {
	t.Helper()
	ct, err := xsd.NewComplexType(xsderr.Loc{}, name, xsd.QName{}, nil, xsd.DerivationRestriction, false,
		nil, nil, xsd.ElementContent{Particle: termParticle(t, term)}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType: %v", err)
	}
	return ct
}

// elementTyped builds a global element declaration whose {type definition} is
// typeName (may be zero for absent).
func elementTyped(t *testing.T, name, typeName xsd.QName) xsd.ElementDeclaration {
	t.Helper()
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, name, typeName, nil, xsd.ScopeGlobal, nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	return e
}

// keyOrRef builds an identity constraint of the given category; refer, when
// non-empty, is the {referenced key} (required for keyref).
func keyOrRef(t *testing.T, name xsd.QName, category xsd.IdentityConstraintCategory, refer xsd.QName) xsd.IdentityConstraint {
	t.Helper()
	sel := xsd.NewXPathExpression(".", nil, nil, nil)
	field := xsd.NewXPathExpression("@x", nil, nil, nil)
	var referPtr *xsd.QName
	if category == xsd.IdentityConstraintKeyref {
		referPtr = &refer
	}
	c, err := xsd.NewIdentityConstraint(xsderr.Loc{}, name, category, sel, []xsd.XPathExpression{field}, referPtr, nil)
	if err != nil {
		t.Fatalf("NewIdentityConstraint: %v", err)
	}
	return c
}

func TestResolveDanglingType(t *testing.T) {
	// An element's @type names a type that is not in the schema.
	b := xsd.NewSchemaBuilder()
	b.AddElement(elementTyped(t, qn("e"), qn("nope")))
	if _, err := b.Finalize(); err == nil {
		t.Fatal("Finalize(dangling @type) succeeded, want src-resolve error")
	} else {
		assertRule(t, err, "src-resolve")
	}
}

func TestResolveDanglingElementRef(t *testing.T) {
	// A complex type's particle is an <element ref> to a missing element.
	b := xsd.NewSchemaBuilder()
	b.AddType(elementContentCT(t, qn("ct"), xsd.ElementDeclarationRef{Name: qn("nope")}))
	if _, err := b.Finalize(); err == nil {
		t.Fatal("Finalize(dangling element ref) succeeded, want src-resolve error")
	} else {
		assertRule(t, err, "src-resolve")
	}
}

func TestResolveDanglingAttributeRef(t *testing.T) {
	// A complex type's attribute use is an <attribute ref> to a missing attribute.
	use, err := xsd.NewAttributeUse(xsderr.Loc{}, false, xsd.AttributeDeclarationRef{Name: qn("nope")}, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeUse: %v", err)
	}
	ct, err := xsd.NewComplexType(xsderr.Loc{}, qn("ct"), xsd.QName{}, nil, xsd.DerivationRestriction, false,
		[]xsd.AttributeUse{use}, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType: %v", err)
	}
	b := xsd.NewSchemaBuilder()
	b.AddType(ct)
	if _, err := b.Finalize(); err == nil {
		t.Fatal("Finalize(dangling attribute ref) succeeded, want src-resolve error")
	} else {
		assertRule(t, err, "src-resolve")
	}
}

func TestResolveDanglingGroupRef(t *testing.T) {
	// A model group definition's particle is a <group ref> to a missing group.
	b := xsd.NewSchemaBuilder()
	b.AddModelGroup(modelGroupDef(t, qn("g"), termParticle(t, xsd.ModelGroupRef{Name: qn("nope")})))
	if _, err := b.Finalize(); err == nil {
		t.Fatal("Finalize(dangling group ref) succeeded, want src-resolve error")
	} else {
		assertRule(t, err, "src-resolve")
	}
}

func TestResolveDanglingKeyref(t *testing.T) {
	// A top-level keyref refers to an identity constraint that is not present.
	b := xsd.NewSchemaBuilder()
	b.AddIdentityConstraint(keyOrRef(t, qn("kr"), xsd.IdentityConstraintKeyref, qn("nope")))
	if _, err := b.Finalize(); err == nil {
		t.Fatal("Finalize(dangling keyref) succeeded, want src-resolve error")
	} else {
		assertRule(t, err, "src-resolve")
	}
}

func TestResolveWrongKind(t *testing.T) {
	// An element's @type names a real ELEMENT declaration, not a type. The
	// kind-specific lookup misses the type table, so this is the same src-resolve
	// failure as a dangling reference.
	b := xsd.NewSchemaBuilder()
	b.AddElement(elementTyped(t, qn("target"), xsd.QName{}))
	b.AddElement(elementTyped(t, qn("e"), qn("target"))) // @type = target, an element
	if _, err := b.Finalize(); err == nil {
		t.Fatal("Finalize(@type names an element) succeeded, want src-resolve error")
	} else {
		assertRule(t, err, "src-resolve")
	}
}

func TestResolveKeyrefToKeyref(t *testing.T) {
	// A keyref whose {referenced key} resolves to another keyref: src-resolve
	// passes (both are IDCs) but c-props-correct clause 1 rejects the category.
	b := xsd.NewSchemaBuilder()
	b.AddIdentityConstraint(keyOrRef(t, qn("x"), xsd.IdentityConstraintKeyref, qn("y")))
	b.AddIdentityConstraint(keyOrRef(t, qn("y"), xsd.IdentityConstraintKeyref, qn("z")))
	b.AddIdentityConstraint(keyOrRef(t, qn("z"), xsd.IdentityConstraintKey, xsd.QName{}))
	if _, err := b.Finalize(); err == nil {
		t.Fatal("Finalize(keyref -> keyref) succeeded, want c-props-correct error")
	} else {
		assertRule(t, err, "c-props-correct")
	}
}

func TestResolveSelfCircularComplexBase(t *testing.T) {
	// A complex type whose {base type definition} is itself (and is not
	// xs:anyType) is a forbidden derivation cycle.
	ct, err := xsd.NewComplexType(xsderr.Loc{}, qn("T"), qn("T"), nil, xsd.DerivationRestriction, false,
		nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType: %v", err)
	}
	b := xsd.NewSchemaBuilder()
	b.AddType(ct)
	if _, err := b.Finalize(); err == nil {
		t.Fatal("Finalize(self-based complex type) succeeded, want ct-props-correct error")
	} else {
		assertRule(t, err, "ct-props-correct")
	}
}

func TestResolveMutualCircularComplexBase(t *testing.T) {
	// A -> B -> A base chain across two named types.
	a, err := xsd.NewComplexType(xsderr.Loc{}, qn("A"), qn("B"), nil, xsd.DerivationRestriction, false,
		nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType A: %v", err)
	}
	bt, err := xsd.NewComplexType(xsderr.Loc{}, qn("B"), qn("A"), nil, xsd.DerivationRestriction, false,
		nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType B: %v", err)
	}
	b := xsd.NewSchemaBuilder()
	b.AddType(a)
	b.AddType(bt)
	if _, err := b.Finalize(); err == nil {
		t.Fatal("Finalize(A<->B base cycle) succeeded, want ct-props-correct error")
	} else {
		assertRule(t, err, "ct-props-correct")
	}
}

func TestResolveAnyTypeSelfBaseAccepted(t *testing.T) {
	// xs:anyType is the one complex type permitted to be its own base (§3.4.7).
	anyType, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Space: xsd.XMLSchemaNS, Local: "anyType"},
		xsd.QName{Space: xsd.XMLSchemaNS, Local: "anyType"}, nil, xsd.DerivationRestriction, false,
		nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType anyType: %v", err)
	}
	b := xsd.NewSchemaBuilder()
	b.AddType(anyType)
	if _, err := b.Finalize(); err != nil {
		t.Fatalf("Finalize(xs:anyType self-base): %v", err)
	}
}

func TestResolveCircularModelGroups(t *testing.T) {
	// Group A references B, group B references A.
	b := xsd.NewSchemaBuilder()
	b.AddModelGroup(modelGroupDef(t, qn("A"), termParticle(t, xsd.ModelGroupRef{Name: qn("B")})))
	b.AddModelGroup(modelGroupDef(t, qn("B"), termParticle(t, xsd.ModelGroupRef{Name: qn("A")})))
	if _, err := b.Finalize(); err == nil {
		t.Fatal("Finalize(A<->B group cycle) succeeded, want mg-props-correct error")
	} else {
		assertRule(t, err, "mg-props-correct")
	}
}

func TestResolveCircularSubstitutionGroups(t *testing.T) {
	// Element A affiliates to B, element B affiliates to A.
	ea, err := xsd.NewElementDeclaration(xsderr.Loc{}, qn("A"), xsd.QName{}, nil, xsd.ScopeGlobal, nil, false, nil,
		[]xsd.QName{qn("B")}, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration A: %v", err)
	}
	eb, err := xsd.NewElementDeclaration(xsderr.Loc{}, qn("B"), xsd.QName{}, nil, xsd.ScopeGlobal, nil, false, nil,
		[]xsd.QName{qn("A")}, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration B: %v", err)
	}
	b := xsd.NewSchemaBuilder()
	b.AddElement(ea)
	b.AddElement(eb)
	if _, err := b.Finalize(); err == nil {
		t.Fatal("Finalize(A<->B substitution cycle) succeeded, want e-props-correct error")
	} else {
		assertRule(t, err, "e-props-correct")
	}
}

func TestResolveValidGraph(t *testing.T) {
	// A fully-resolvable, acyclic interlinked graph must finalize cleanly:
	//   - simple type st
	//   - base complex type, derived complex type extending it
	//   - element e typed by st
	//   - group g1 referencing group g2 (acyclic)
	//   - keyref kr referring to key k
	base, err := xsd.NewComplexType(xsderr.Loc{}, qn("base"), xsd.QName{}, nil, xsd.DerivationRestriction, false,
		nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType base: %v", err)
	}
	derived, err := xsd.NewComplexType(xsderr.Loc{}, qn("derived"), qn("base"), nil, xsd.DerivationExtension, false,
		nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType derived: %v", err)
	}

	b := xsd.NewSchemaBuilder()
	b.AddType(simpleTypeNamed(t, qn("st")))
	b.AddType(base)
	b.AddType(derived)
	b.AddElement(elementTyped(t, qn("e"), qn("st")))
	b.AddModelGroup(modelGroupDef(t, qn("g1"), termParticle(t, xsd.ModelGroupRef{Name: qn("g2")})))
	b.AddModelGroup(modelGroupDef(t, qn("g2"))) // empty sequence, no refs
	b.AddIdentityConstraint(keyOrRef(t, qn("k"), xsd.IdentityConstraintKey, xsd.QName{}))
	b.AddIdentityConstraint(keyOrRef(t, qn("kr"), xsd.IdentityConstraintKeyref, qn("k")))

	if _, err := b.Finalize(); err != nil {
		t.Fatalf("Finalize(valid interlinked graph): %v", err)
	}
}

func TestNewParticleRejectsResolvedTermNilInner(t *testing.T) {
	// ResolvedTerm{Term: nil} is a representable absent {term} that the outer
	// nil-TermOrRef check misses; NewParticle must reject it.
	_, err := xsd.NewParticle(xsderr.Loc{}, occurs11(t), xsd.ResolvedTerm{Term: nil}, nil)
	if err == nil {
		t.Fatal("NewParticle(ResolvedTerm{Term: nil}) succeeded, want p-props-correct error")
	}
	assertRule(t, err, "p-props-correct")
}
