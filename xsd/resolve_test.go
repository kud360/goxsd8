package xsd_test

import (
	"fmt"
	"strings"
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

// modelGroupDefAt builds a named model group definition, positioned at loc, over
// a sequence of the given particles.
func modelGroupDefAt(t *testing.T, loc xsderr.Loc, name xsd.QName, particles ...xsd.Particle) xsd.ModelGroupDefinition {
	t.Helper()
	d, err := xsd.NewModelGroupDefinition(loc, name, seqGroup(t, particles...), nil)
	if err != nil {
		t.Fatalf("NewModelGroupDefinition: %v", err)
	}
	return d
}

// modelGroupDef is modelGroupDefAt for a test that does not exercise positions.
func modelGroupDef(t *testing.T, name xsd.QName, particles ...xsd.Particle) xsd.ModelGroupDefinition {
	t.Helper()
	return modelGroupDefAt(t, xsderr.Loc{}, name, particles...)
}

// elementContentCTAt builds a named element-only complex type, positioned at loc,
// whose single particle carries term.
func elementContentCTAt(t *testing.T, loc xsderr.Loc, name xsd.QName, term xsd.TermOrRef) xsd.ComplexType {
	t.Helper()
	ct, err := xsd.NewComplexType(loc, name, xsd.QName{}, nil, xsd.DerivationRestriction, false,
		nil, nil, nil, xsd.ElementContent{Particle: termParticle(t, term)}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType: %v", err)
	}
	return ct
}

// elementContentCT is elementContentCTAt for a test that does not exercise
// positions.
func elementContentCT(t *testing.T, name xsd.QName, term xsd.TermOrRef) xsd.ComplexType {
	t.Helper()
	return elementContentCTAt(t, xsderr.Loc{}, name, term)
}

// baseCTAt builds an empty-content complex type, positioned at loc, whose {base
// type definition} is the by-name reference base.
func baseCTAt(t *testing.T, loc xsderr.Loc, name, base xsd.QName) xsd.ComplexType {
	t.Helper()
	ct, err := xsd.NewComplexType(loc, name, base, nil, xsd.DerivationRestriction, false,
		nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType: %v", err)
	}
	return ct
}

// elementTypedAt builds a global element declaration, positioned at loc, whose
// {type definition} is a by-name reference to typeName, or ABSENT (a nil slot)
// when typeName is zero — a zero-named xsd.TypeDefinitionRef is rejected at
// construction.
func elementTypedAt(t *testing.T, loc xsderr.Loc, name, typeName xsd.QName) xsd.ElementDeclaration {
	t.Helper()
	var typeDef xsd.TypeDefinitionOrRef
	if typeName != (xsd.QName{}) {
		typeDef = xsd.TypeDefinitionRef{Name: typeName}
	}
	e, err := xsd.NewElementDeclaration(loc, name, typeDef, nil, xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	return e
}

// elementTyped is elementTypedAt for a test that does not exercise positions.
func elementTyped(t *testing.T, name, typeName xsd.QName) xsd.ElementDeclaration {
	t.Helper()
	return elementTypedAt(t, xsderr.Loc{}, name, typeName)
}

// elementAffiliatedAt builds a global element declaration, positioned at loc,
// whose {substitution group affiliations} is the single member aff.
func elementAffiliatedAt(t *testing.T, loc xsderr.Loc, name, aff xsd.QName) xsd.ElementDeclaration {
	t.Helper()
	e, err := xsd.NewElementDeclaration(loc, name, nil, nil, xsd.NewGlobalScope(), nil, false, nil,
		[]xsd.QName{aff}, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	return e
}

// keyOrRefFields builds an identity constraint of the given category with
// fieldCount distinct {fields}; refer, when non-empty, is the {referenced key}
// (required for keyref).
func keyOrRefFields(t *testing.T, name xsd.QName, category xsd.IdentityConstraintCategory, refer xsd.QName, fieldCount int) xsd.IdentityConstraint {
	t.Helper()
	return keyOrRefFieldsAt(t, xsderr.Loc{}, name, category, refer, fieldCount)
}

// keyOrRefFieldsAt is keyOrRefFields positioned at loc.
func keyOrRefFieldsAt(t *testing.T, loc xsderr.Loc, name xsd.QName, category xsd.IdentityConstraintCategory, refer xsd.QName, fieldCount int) xsd.IdentityConstraint {
	t.Helper()
	sel := xsd.NewXPathExpression(".", nil, nil, nil)
	fields := make([]xsd.XPathExpression, 0, fieldCount)
	for i := range fieldCount {
		fields = append(fields, xsd.NewXPathExpression(fmt.Sprintf("@x%d", i), nil, nil, nil))
	}
	var referPtr *xsd.QName
	if category == xsd.IdentityConstraintKeyref {
		referPtr = &refer
	}
	c, err := xsd.NewIdentityConstraint(loc, name, category, sel, fields, referPtr, nil)
	if err != nil {
		t.Fatalf("NewIdentityConstraint: %v", err)
	}
	return c
}

// keyOrRef builds a single-{field} identity constraint of the given category.
func keyOrRef(t *testing.T, name xsd.QName, category xsd.IdentityConstraintCategory, refer xsd.QName) xsd.IdentityConstraint {
	t.Helper()
	return keyOrRefFields(t, name, category, refer, 1)
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

// TestResolveAnonymousComplexTypeDanglingBase pins the diagnostic Phase A emits
// for an ANONYMOUS complex type's dangling {base type definition}: the type has
// no {name}, whose String is "", so the message must describe what it is instead
// of leaving a hole.
//
// This is the SchemaBuilder's own defence, and the only path that reaches it: a
// producer-mapped type never gets here, because parser/produce_complex.go
// resolves every base= itself — on the restriction alternant too, since #346's
// §3.4.2.1 clause 1 {assertions} fold reads the base component on both — and
// charges the same src-resolve clause 1.1 first, positioned. Two entry points on
// one rule for the two construction paths, the shape buildComplexType's
// ct-props-correct clause 3 rejection already has.
func TestResolveAnonymousComplexTypeDanglingBase(t *testing.T) {
	id := xsd.NewComponentID()
	ct, err := xsd.NewAnonymousComplexType(xsderr.Loc{}, xsd.ElementDeclarationContext{Component: id},
		qn("nope"), nil, xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewAnonymousComplexType: %v", err)
	}
	e, err := xsd.NewElementDeclarationOwningType(xsderr.Loc{}, id, qn("doc"), ct,
		nil, xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclarationOwningType: %v", err)
	}
	b := xsd.NewSchemaBuilder()
	b.AddElement(e)
	_, err = b.Finalize()
	assertRule(t, err, "src-resolve")
	if !strings.Contains(err.Error(), "anonymous complex type {base type definition}") {
		t.Fatalf("error = %v, want the anonymous owner phrase", err)
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
	use, err := xsd.NewAttributeUse(xsderr.Loc{}, false, xsd.AttributeDeclarationRef{Name: qn("nope")}, nil, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeUse: %v", err)
	}
	ct, err := xsd.NewComplexType(xsderr.Loc{}, qn("ct"), xsd.QName{}, nil, xsd.DerivationRestriction, false,
		[]xsd.AttributeUse{use}, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
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

func TestResolveKeyrefFieldCardinalityMismatch(t *testing.T) {
	// src-resolve clause 1.7 and c-props-correct clause 1 both pass — the target
	// exists and is a key — but the {fields} cardinalities differ, which
	// c-props-correct clause 2 forbids. Checked in both directions so a check
	// written with the wrong comparison would still be caught.
	for _, tc := range []struct {
		name                 string
		keyFields, refFields int
	}{
		{"keyref wider than key", 1, 2},
		{"keyref narrower than key", 3, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := xsd.NewSchemaBuilder()
			b.AddIdentityConstraint(keyOrRefFields(t, qn("k"), xsd.IdentityConstraintKey, xsd.QName{}, tc.keyFields))
			b.AddIdentityConstraint(keyOrRefFields(t, qn("kr"), xsd.IdentityConstraintKeyref, qn("k"), tc.refFields))
			_, err := b.Finalize()
			if err == nil {
				t.Fatal("Finalize(keyref {fields} cardinality mismatch) succeeded, want c-props-correct error")
			}
			assertRule(t, err, "c-props-correct")
			// Both clauses charge c-props-correct, so the rule alone does not say
			// which one fired; the message must name clause 2.
			if !strings.Contains(err.Error(), "clause 2") {
				t.Errorf("Finalize error = %q, want it to cite c-props-correct clause 2", err)
			}
		})
	}
}

func TestResolveKeyrefFieldCardinalityMatch(t *testing.T) {
	// The same shape with equal cardinality satisfies clause 2 and finalizes.
	b := xsd.NewSchemaBuilder()
	b.AddIdentityConstraint(keyOrRefFields(t, qn("k"), xsd.IdentityConstraintKey, xsd.QName{}, 3))
	b.AddIdentityConstraint(keyOrRefFields(t, qn("kr"), xsd.IdentityConstraintKeyref, qn("k"), 3))
	if _, err := b.Finalize(); err != nil {
		t.Fatalf("Finalize(keyref {fields} cardinality match): %v", err)
	}
}

func TestResolveKeyrefReachedByBothWalks(t *testing.T) {
	// resolveKeyref is reached from two places: resolveReferences' direct walk over
	// the schema-level {identity-constraint definitions} (§3.17.2 sources those
	// from the constraints "anywhere within the [[children]]", not from top-level
	// ones alone), and resolveElementDecl's walk over an element's nested ones. An
	// IDC that sits in both — nested under element e AND a member of the
	// schema-level property, as a <key>/<keyref> declared inside an <element> is —
	// is therefore visited twice. That must be idempotent: two clean passes, not a
	// duplicate-name or double-registration failure.
	k := keyOrRefFields(t, qn("k"), xsd.IdentityConstraintKey, xsd.QName{}, 2)
	kr := keyOrRefFields(t, qn("kr"), xsd.IdentityConstraintKeyref, qn("k"), 2)
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, qn("e"), nil, nil, xsd.NewGlobalScope(), nil, false,
		[]xsd.IdentityConstraint{k, kr}, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	if got := len(e.IdentityConstraints()); got != 2 {
		t.Fatalf("element {identity-constraint definitions} count = %d, want 2", got)
	}
	b := xsd.NewSchemaBuilder()
	b.AddElement(e)
	b.AddIdentityConstraint(k)
	b.AddIdentityConstraint(kr)
	if _, err := b.Finalize(); err != nil {
		t.Fatalf("Finalize(keyref reached by both walks): %v", err)
	}

	// And the element walk enforces clause 2 too: widening only the nested keyref
	// (leaving the top-level pair matched) must still be rejected.
	wide := keyOrRefFields(t, qn("kr"), xsd.IdentityConstraintKeyref, qn("k"), 5)
	eWide, err := xsd.NewElementDeclaration(xsderr.Loc{}, qn("e"), nil, nil, xsd.NewGlobalScope(), nil, false,
		[]xsd.IdentityConstraint{wide}, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration (wide): %v", err)
	}
	b2 := xsd.NewSchemaBuilder()
	b2.AddElement(eWide)
	b2.AddIdentityConstraint(k)
	_, err = b2.Finalize()
	if err == nil {
		t.Fatal("Finalize(nested keyref cardinality mismatch) succeeded, want c-props-correct error")
	}
	assertRule(t, err, "c-props-correct")
}

func TestResolveSelfCircularComplexBase(t *testing.T) {
	// A complex type whose {base type definition} is itself (and is not
	// xs:anyType) is a forbidden derivation cycle.
	ct, err := xsd.NewComplexType(xsderr.Loc{}, qn("T"), qn("T"), nil, xsd.DerivationRestriction, false,
		nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
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
		nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType A: %v", err)
	}
	bt, err := xsd.NewComplexType(xsderr.Loc{}, qn("B"), qn("A"), nil, xsd.DerivationRestriction, false,
		nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
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
		nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
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
	b := xsd.NewSchemaBuilder()
	b.AddElement(elementAffiliatedAt(t, xsderr.Loc{}, qn("A"), qn("B")))
	b.AddElement(elementAffiliatedAt(t, xsderr.Loc{}, qn("B"), qn("A")))
	_, err := b.Finalize()
	if err == nil {
		t.Fatal("Finalize(A<->B substitution cycle) succeeded, want e-props-correct error")
	}
	assertRule(t, err, "e-props-correct")
}

func TestResolveValidGraph(t *testing.T) {
	// A fully-resolvable, acyclic interlinked graph must finalize cleanly:
	//   - simple type st
	//   - base complex type, derived complex type extending it
	//   - element e typed by st
	//   - group g1 referencing group g2 (acyclic)
	//   - keyref kr referring to key k
	base, err := xsd.NewComplexType(xsderr.Loc{}, qn("base"), xsd.QName{}, nil, xsd.DerivationRestriction, false,
		nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType base: %v", err)
	}
	derived, err := xsd.NewComplexType(xsderr.Loc{}, qn("derived"), qn("base"), nil, xsd.DerivationExtension, false,
		nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
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

// resolveLoc builds a distinctive non-zero position, one per line number, so an
// assertion can pin WHICH component a rejection was charged to rather than only
// that some position reached the error.
func resolveLoc(line int) xsderr.Loc {
	return xsderr.Loc{URI: "resolve.xsd", Line: line, Col: 3}
}

// assertLoc checks that err is charged exactly want. It compares the WHOLE Loc
// rather than merely rejecting the zero one: a change that threads some other
// component's position — the unreachable target of a dangling reference, or the
// far side of a cycle — must fail here too, not slip through on non-zeroness.
func assertLoc(t *testing.T, err error, want xsderr.Loc) {
	t.Helper()
	got, ok := xsderr.LocOf(err)
	if !ok {
		t.Fatalf("error %v is not an *xsderr.Error, so carries no Loc", err)
	}
	if got == (xsderr.Loc{}) {
		t.Fatalf("error %v is charged the zero Loc (position unknown), want %s", err, want)
	}
	if got != want {
		t.Errorf("Loc = %s, want %s", got, want)
	}
}

// TestResolveRejectionsCiteTheOffendingComponent pins the position of every
// Loc-bearing rejection the resolution pass can raise — all ten sites, across
// src-resolve (§3.17.6.2) clauses 1.1, 1.2, 1.3, 1.5 and 1.7, c-props-correct
// (§3.11.6.1) clauses 1 and 2, ct-props-correct (§3.4.6.1) clause 3,
// mg-props-correct (§3.8.6.1) clause 2 and e-props-correct (§3.3.6.1) clause 5 —
// so no site can drift back to the zero "position unknown" Loc unnoticed.
//
// The table is per-SITE, not per-rule. A shared rule ID is no evidence that a
// sibling site is guarded: sites charged the same rule reach their position by
// different routes, and a case covering one of them would leave the others free
// to regress silently.
//
// src-resolve accordingly gets five cases, because its reference sites do not all
// take their position the same way. Three of them — a particle {term} <element
// ref>, a particle {term} <group ref>, and an attribute use <attribute ref> — sit
// on a Particle and an AttributeUse, neither of which RETAINS a position (xsd
// doc.go, STYLE T5), so all three are charged the enclosing complex type's
// instead. Those three cases build the Particle and the AttributeUse at the ZERO
// Loc on purpose: if the substitution were ever dropped in favour of the
// ref-bearing component's own position, the rejection would go back to citing
// nothing and the case would fail.
//
// c-props-correct gets two cases for the same per-site reason: resolveKeyref's
// category check (clause 1) returns before its cardinality check (clause 2) can
// run, so only a keyref whose target is the wrong CATEGORY reaches the former and
// only one whose target matches in category reaches the latter.
func TestResolveRejectionsCiteTheOffendingComponent(t *testing.T) {
	for _, tc := range []struct {
		name  string
		rule  xsderr.Rule
		build func(t *testing.T) (*xsd.SchemaBuilder, xsderr.Loc)
	}{
		{
			name: "src-resolve dangling @type cites the element declaration",
			rule: "src-resolve",
			build: func(t *testing.T) (*xsd.SchemaBuilder, xsderr.Loc) {
				b := xsd.NewSchemaBuilder()
				b.AddElement(elementTypedAt(t, resolveLoc(11), qn("e"), qn("nope")))
				return b, resolveLoc(11)
			},
		},
		{
			name: "src-resolve dangling element ref cites the enclosing complex type",
			rule: "src-resolve",
			build: func(t *testing.T) (*xsd.SchemaBuilder, xsderr.Loc) {
				b := xsd.NewSchemaBuilder()
				b.AddType(elementContentCTAt(t, resolveLoc(21), qn("ct"), xsd.ElementDeclarationRef{Name: qn("nope")}))
				return b, resolveLoc(21)
			},
		},
		{
			name: "src-resolve dangling attribute ref cites the enclosing complex type",
			rule: "src-resolve",
			build: func(t *testing.T) (*xsd.SchemaBuilder, xsderr.Loc) {
				use, err := xsd.NewAttributeUse(xsderr.Loc{}, false, xsd.AttributeDeclarationRef{Name: qn("nope")}, nil, false, nil)
				if err != nil {
					t.Fatalf("NewAttributeUse: %v", err)
				}
				ct, err := xsd.NewComplexType(resolveLoc(31), qn("ct"), xsd.QName{}, nil, xsd.DerivationRestriction, false,
					[]xsd.AttributeUse{use}, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
				if err != nil {
					t.Fatalf("NewComplexType: %v", err)
				}
				b := xsd.NewSchemaBuilder()
				b.AddType(ct)
				return b, resolveLoc(31)
			},
		},
		{
			name: "src-resolve dangling group ref cites the enclosing complex type",
			rule: "src-resolve",
			build: func(t *testing.T) (*xsd.SchemaBuilder, xsderr.Loc) {
				b := xsd.NewSchemaBuilder()
				b.AddType(elementContentCTAt(t, resolveLoc(81), qn("ct"), xsd.ModelGroupRef{Name: qn("nope")}))
				return b, resolveLoc(81)
			},
		},
		{
			name: "src-resolve dangling keyref cites the keyref",
			rule: "src-resolve",
			build: func(t *testing.T) (*xsd.SchemaBuilder, xsderr.Loc) {
				// The target does not exist to have a position, so the referring
				// keyref is the only component the rejection can name.
				b := xsd.NewSchemaBuilder()
				b.AddIdentityConstraint(keyOrRefFieldsAt(t, resolveLoc(91), qn("kr"), xsd.IdentityConstraintKeyref, qn("nope"), 1))
				return b, resolveLoc(91)
			},
		},
		{
			name: "c-props-correct wrong category cites the referring keyref",
			rule: "c-props-correct",
			build: func(t *testing.T) (*xsd.SchemaBuilder, xsderr.Loc) {
				// Two mutually-referring keyrefs: kr1 is visited first in document
				// order and its {referenced key} is itself a keyref, so clause 1
				// rejects there. Equal {fields} counts keep clause 2 out of it.
				b := xsd.NewSchemaBuilder()
				b.AddIdentityConstraint(keyOrRefFieldsAt(t, resolveLoc(101), qn("kr1"), xsd.IdentityConstraintKeyref, qn("kr2"), 1))
				b.AddIdentityConstraint(keyOrRefFieldsAt(t, resolveLoc(102), qn("kr2"), xsd.IdentityConstraintKeyref, qn("kr1"), 1))
				return b, resolveLoc(101)
			},
		},
		{
			name: "c-props-correct cardinality mismatch cites the keyref",
			rule: "c-props-correct",
			build: func(t *testing.T) (*xsd.SchemaBuilder, xsderr.Loc) {
				b := xsd.NewSchemaBuilder()
				b.AddIdentityConstraint(keyOrRefFieldsAt(t, resolveLoc(41), qn("kr"), xsd.IdentityConstraintKeyref, qn("k"), 2))
				b.AddIdentityConstraint(keyOrRefFieldsAt(t, resolveLoc(42), qn("k"), xsd.IdentityConstraintKey, xsd.QName{}, 1))
				return b, resolveLoc(41)
			},
		},
		{
			name: "ct-props-correct base cycle cites the named complex type",
			rule: "ct-props-correct",
			build: func(t *testing.T) (*xsd.SchemaBuilder, xsderr.Loc) {
				// A -> B -> A. The walk starts at document-order root A and detects the
				// cycle on re-reaching A, so A is both the name printed and the position
				// charged.
				b := xsd.NewSchemaBuilder()
				b.AddType(baseCTAt(t, resolveLoc(51), qn("A"), qn("B")))
				b.AddType(baseCTAt(t, resolveLoc(52), qn("B"), qn("A")))
				return b, resolveLoc(51)
			},
		},
		{
			name: "mg-props-correct group cycle cites the model group definition",
			rule: "mg-props-correct",
			build: func(t *testing.T) (*xsd.SchemaBuilder, xsderr.Loc) {
				b := xsd.NewSchemaBuilder()
				b.AddModelGroup(modelGroupDefAt(t, resolveLoc(61), qn("A"), termParticle(t, xsd.ModelGroupRef{Name: qn("B")})))
				b.AddModelGroup(modelGroupDefAt(t, resolveLoc(62), qn("B"), termParticle(t, xsd.ModelGroupRef{Name: qn("A")})))
				return b, resolveLoc(61)
			},
		},
		{
			name: "e-props-correct substitution cycle cites the element declaration",
			rule: "e-props-correct",
			build: func(t *testing.T) (*xsd.SchemaBuilder, xsderr.Loc) {
				b := xsd.NewSchemaBuilder()
				b.AddElement(elementAffiliatedAt(t, resolveLoc(71), qn("A"), qn("B")))
				b.AddElement(elementAffiliatedAt(t, resolveLoc(72), qn("B"), qn("A")))
				return b, resolveLoc(71)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, want := tc.build(t)
			_, err := b.Finalize()
			if err == nil {
				t.Fatalf("Finalize succeeded, want %s error", tc.rule)
			}
			assertRule(t, err, tc.rule)
			assertLoc(t, err, want)
		})
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
