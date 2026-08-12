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
