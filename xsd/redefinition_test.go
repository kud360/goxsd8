package xsd

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// dGroup builds a top-level attribute group definition with the given
// {attribute uses} and {attribute wildcard}.
func dGroup(t *testing.T, name QName, uses []AttributeUse, wildcard *Wildcard) AttributeGroupDefinition {
	t.Helper()
	g, err := NewAttributeGroupDefinition(xsderr.Loc{}, name, uses, wildcard, nil)
	if err != nil {
		t.Fatalf("NewAttributeGroupDefinition(%s): %v", name, err)
	}
	return g
}

// TestSrcRedefineClause722Accepts is the control: a redefinition that genuinely
// restricts — it narrows one attribute's type and drops an OPTIONAL one — is
// accepted, and the original it pairs with enters neither {attribute group
// definitions} nor the by-name index (§4.2.4 clause 4.1.2), so
// AddRedefiningAttributeGroup adds exactly one component.
func TestSrcRedefineClause722Accepts(t *testing.T) {
	b := NewSchemaBuilder()
	b.AddType(dAnyType(t))
	str := dPrimitive(t, uq("str"))
	b.AddType(str)
	b.AddType(dSimple(t, uq("narrow"), str))
	original := dGroup(t, uq("ag"), []AttributeUse{
		dAttr(t, uq("a"), uq("str")),
		dAttr(t, uq("b"), uq("str")),
	}, nil)
	b.AddRedefiningAttributeGroup(dGroup(t, uq("ag"), []AttributeUse{dAttr(t, uq("a"), uq("narrow"))}, nil), original)
	s, err := b.Finalize()
	if err != nil {
		t.Fatalf("a restrictive redefinition was rejected: %v", err)
	}
	if got := len(s.AttributeGroups()); got != 1 {
		t.Fatalf("{attribute group definitions} has %d members, want 1: the original contributes no component", got)
	}
	g := s.AttributeGroups()[0]
	if got := len(g.AttributeUses()); got != 1 {
		t.Fatalf("the redefinition has %d attribute uses, want 1: no inheritance from the redefined group occurs", got)
	}
}

// TestSrcRedefineClause722Rejects covers the three ways clause 7.2.2's c-ran
// clause 3 comparison can fail over attribute groups, each through a different
// arm of the shared check.
func TestSrcRedefineClause722Rejects(t *testing.T) {
	any := uWildcard(t, NamespaceConstraintAny, nil, ProcessLax)
	for _, tc := range []struct {
		name     string
		original AttributeGroupDefinition
		g        AttributeGroupDefinition
	}{
		{"an attribute the original neither declares nor admits",
			dGroup(t, uq("ag"), []AttributeUse{dAttr(t, uq("a"), uq("str"))}, nil),
			dGroup(t, uq("ag"), []AttributeUse{dAttr(t, uq("a"), uq("str")), dAttr(t, uq("extra"), uq("str"))}, nil)},
		{"a type that is not derived from the original's",
			dGroup(t, uq("ag"), []AttributeUse{dAttr(t, uq("a"), uq("narrow"))}, nil),
			dGroup(t, uq("ag"), []AttributeUse{dAttr(t, uq("a"), uq("str"))}, nil)},
		{"an {attribute wildcard} the original has no wildcard for",
			dGroup(t, uq("ag"), nil, nil),
			dGroup(t, uq("ag"), nil, &any)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := NewSchemaBuilder()
			b.AddType(dAnyType(t))
			str := dPrimitive(t, uq("str"))
			b.AddType(str)
			b.AddType(dSimple(t, uq("narrow"), str))
			b.AddRedefiningAttributeGroup(tc.g, tc.original)
			_, err := b.Finalize()
			expectRule(t, err, ruleSrcRedefine)
			if !strings.Contains(err.Error(), "7.2.2") {
				t.Fatalf("message %q does not name src-redefine clause 7.2.2", err)
			}
			if !strings.Contains(err.Error(), "the redefining attribute group") || !strings.Contains(err.Error(), "the original attribute group") {
				t.Fatalf("message %q does not name both sides by their role", err)
			}
		})
	}
}

// TestSrcRedefineClause722NoInheritance pins the Note under clause 7.2.2 — "No
// inheritance from the <redefine>d attribute group occurs" — by the one shape
// the two readings disagree about: the original REQUIRES an attribute the
// redefinition does not mention. Under the Note the redefinition's {attribute
// uses} are its own <attribute> children alone, so the required use is missing
// and cvc-complex-type clause 3 is violated; under an inheriting reading the use
// would be folded in and the schema would pass. The rejection is the assertion.
func TestSrcRedefineClause722NoInheritance(t *testing.T) {
	b := NewSchemaBuilder()
	b.AddType(dAnyType(t))
	b.AddType(dPrimitive(t, uq("str")))
	original := dGroup(t, uq("ag"), []AttributeUse{
		dAttrUse(t, uq("r"), uq("str"), true, nil),
		dAttr(t, uq("a"), uq("str")),
	}, nil)
	b.AddRedefiningAttributeGroup(dGroup(t, uq("ag"), []AttributeUse{dAttr(t, uq("a"), uq("str"))}, nil), original)
	_, err := b.Finalize()
	expectRule(t, err, ruleSrcRedefine)
	if !strings.Contains(err.Error(), "no use for attribute "+uq("r").String()) {
		t.Fatalf("message %q does not name the required attribute the redefinition drops", err)
	}
}

// dMGD builds a top-level model group definition over the given {model group}.
func dMGD(t *testing.T, name QName, g ModelGroup) ModelGroupDefinition {
	t.Helper()
	d, err := NewModelGroupDefinition(xsderr.Loc{}, name, g, nil)
	if err != nil {
		t.Fatalf("NewModelGroupDefinition(%s): %v", name, err)
	}
	return d
}

// mgdRedefines finalizes a schema holding one redefining model group definition
// paired with original, and reports the verdict src-redefine clause 6.2.2
// reaches. The two model groups are the only difference between runs.
func mgdRedefines(t *testing.T, original, redefining ModelGroup) error {
	t.Helper()
	return dFinalize(t, func(b *SchemaBuilder) {
		b.AddRedefiningModelGroup(dMGD(t, uq("g"), redefining), dMGD(t, uq("g"), original))
	})
}

// TestSrcRedefineClause622 pins the clause over the shapes a <group>
// redefinition takes: it accepts a redefinition whose {model group} accepts a
// SUBSET of the element sequences the original accepts, and rejects one that
// accepts a sequence the original does not.
func TestSrcRedefineClause622(t *testing.T) {
	for _, tc := range []struct {
		name       string
		original   ModelGroup
		redefining ModelGroup
		wantSubset bool
	}{
		{"identical",
			uGroup(t, CompositorSequence, cElem(t, "a", 1, 1)),
			uGroup(t, CompositorSequence, cElem(t, "a", 1, 1)), true},
		{"an optional member dropped",
			uGroup(t, CompositorSequence, cElem(t, "a", 1, 1), cElem(t, "b", 0, 1)),
			uGroup(t, CompositorSequence, cElem(t, "a", 1, 1)), true},
		{"one branch of a choice",
			uGroup(t, CompositorChoice, cElem(t, "a", 1, 1), cElem(t, "b", 1, 1)),
			uGroup(t, CompositorSequence, cElem(t, "b", 1, 1)), true},
		{"an occurrence range narrowed inside a wide one",
			uGroup(t, CompositorSequence, cElem(t, "a", 0, 100)),
			uGroup(t, CompositorSequence, cElem(t, "a", 3, 6)), true},
		{"a member the original never admits",
			uGroup(t, CompositorSequence, cElem(t, "a", 1, 1)),
			uGroup(t, CompositorSequence, cElem(t, "a", 1, 1), cElem(t, "c", 1, 1)), false},
		{"an occurrence range widened",
			uGroup(t, CompositorSequence, cElem(t, "a", 1, 3)),
			uGroup(t, CompositorSequence, cElem(t, "a", 1, 5)), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := mgdRedefines(t, tc.original, tc.redefining)
			if tc.wantSubset {
				if err != nil {
					t.Fatalf("a redefinition accepting a subset was rejected: %v", err)
				}
				return
			}
			expectRule(t, err, ruleSrcRedefine)
			if !strings.Contains(err.Error(), "6.2.2") {
				t.Fatalf("message %q does not name src-redefine clause 6.2.2", err)
			}
		})
	}
}

// TestSrcRedefineClause622ContributesOneComponent pins §4.2.4 clause 4.1.2 at the
// builder boundary: the original is in no property and no index, so a pairing
// adds exactly one model group definition and the schema's only {model group
// definitions} member is the REDEFINITION's.
func TestSrcRedefineClause622ContributesOneComponent(t *testing.T) {
	b := NewSchemaBuilder()
	b.AddType(dAnyType(t))
	b.AddType(uNamedType(t, uq("T")))
	b.AddRedefiningModelGroup(
		dMGD(t, uq("g"), uGroup(t, CompositorSequence, cElem(t, "a", 1, 1))),
		dMGD(t, uq("g"), uGroup(t, CompositorSequence, cElem(t, "a", 1, 1), cElem(t, "b", 0, 1))))
	s, err := b.Finalize()
	if err != nil {
		t.Fatalf("a restrictive redefinition was rejected: %v", err)
	}
	if got := len(s.ModelGroups()); got != 1 {
		t.Fatalf("{model group definitions} has %d members, want 1: the original contributes no component", got)
	}
	if got := len(s.ModelGroups()[0].ModelGroup().Particles()); got != 1 {
		t.Fatalf("the schema's {urn:x}g has %d particles, want the redefinition's 1", got)
	}
}

// TestSrcRedefineClause622ChargesLanguageContainmentAlone pins the scope
// decision (contentRestrictionScope). A redefinition that retypes an element to
// a type NOT derived from the original's accepts exactly the same element
// sequences, so clause 6.2.2 — which states containment and invokes
// cos-content-act-restrict for nothing — accepts it, while clause 2 of that
// constraint (ctr-child-type-subsumption) rejects it. The same pair under
// derivation-ok-restriction clause 2.4.2 must still be rejected, or the
// narrowing has leaked into the caller that does charge clause 2.
func TestSrcRedefineClause622ChargesLanguageContainmentAlone(t *testing.T) {
	original := uGroup(t, CompositorSequence, uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("str"))}))
	retyped := uGroup(t, CompositorSequence, uOne(t, ResolvedTerm{Term: uLocal(t, uq("a"), uq("other"))}))
	if err := mgdRedefines(t, original, retyped); err != nil {
		t.Fatalf("clause 6.2.2 rejected a redefinition accepting the same sequences: %v", err)
	}
	expectRule(t, cRestricts(t, original, retyped), ruleDerivationOKRestriction)
}

// TestAddRedefiningModelGroupPanicsOnAbsentName pins both guards: a redefinition
// pairs two TOP-LEVEL definitions, so an absent {name} on either side is a
// producer bug rather than a schema-validity condition.
func TestAddRedefiningModelGroupPanicsOnAbsentName(t *testing.T) {
	named := dMGD(t, uq("g"), uGroup(t, CompositorSequence, cElem(t, "a", 1, 1)))
	for _, tc := range []struct {
		name string
		d    ModelGroupDefinition
		orig ModelGroupDefinition
	}{
		{"the redefinition", ModelGroupDefinition{}, named},
		{"the original", named, ModelGroupDefinition{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("AddRedefiningModelGroup accepted an absent {name} on %s", tc.name)
				}
			}()
			NewSchemaBuilder().AddRedefiningModelGroup(tc.d, tc.orig)
		})
	}
}

// TestAddRedefiningAttributeGroupPanicsOnAbsentName pins both guards: a
// redefinition pairs two TOP-LEVEL definitions, so an absent {name} on either
// side is a producer bug rather than a schema-validity condition.
func TestAddRedefiningAttributeGroupPanicsOnAbsentName(t *testing.T) {
	named := dGroup(t, uq("ag"), nil, nil)
	for _, tc := range []struct {
		name string
		g    AttributeGroupDefinition
		orig AttributeGroupDefinition
	}{
		{"the redefinition", AttributeGroupDefinition{}, named},
		{"the original", named, AttributeGroupDefinition{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("AddRedefiningAttributeGroup accepted an absent {name} on %s", tc.name)
				}
			}()
			NewSchemaBuilder().AddRedefiningAttributeGroup(tc.g, tc.orig)
		})
	}
}
