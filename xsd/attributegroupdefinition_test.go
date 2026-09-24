package xsd_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// useWithLocalName builds an Attribute Use whose {attribute declaration} is a
// local declaration with the given expanded name.
func useWithLocalName(t *testing.T, name xsd.QName) xsd.AttributeUse {
	t.Helper()
	u, err := xsd.NewAttributeUse(xsderr.Loc{}, false, localDecl(t, name), nil, false)
	if err != nil {
		t.Fatalf("NewAttributeUse: %v", err)
	}
	return u
}

// useWithRefName builds an Attribute Use whose {attribute declaration} is a
// deferred reference with the given expanded name.
func useWithRefName(t *testing.T, name xsd.QName) xsd.AttributeUse {
	t.Helper()
	u, err := xsd.NewAttributeUse(xsderr.Loc{}, false, xsd.AttributeDeclarationRef{Name: name}, nil, false)
	if err != nil {
		t.Fatalf("NewAttributeUse: %v", err)
	}
	return u
}

// resolvedUses lifts attribute uses into attribute content, one
// ResolvedAttributeUse each: the content of a container with no
// <attributeGroup ref>.
func resolvedUses(uses ...xsd.AttributeUse) []xsd.AttributeUseOrGroupRef {
	content := make([]xsd.AttributeUseOrGroupRef, len(uses))
	for i, u := range uses {
		content[i] = xsd.ResolvedAttributeUse{Use: u}
	}
	return content
}

func TestNewAttributeGroupDefinitionValid(t *testing.T) {
	name := xsd.QName{Space: "urn:ns", Local: "g"}
	content := resolvedUses(
		useWithLocalName(t, xsd.QName{Space: "urn:ns", Local: "a"}),
		useWithRefName(t, xsd.QName{Space: "urn:ns", Local: "b"}),
	)
	g, err := xsd.NewAttributeGroupDefinition(xsderr.Loc{}, name, content, nil)
	if err != nil {
		t.Fatalf("NewAttributeGroupDefinition unexpected error: %v", err)
	}
	if g.Name() != name {
		t.Errorf("Name() = %v, want %v", g.Name(), name)
	}
	if got := g.AttributeUses(); len(got) != 2 {
		t.Errorf("AttributeUses() len = %d, want 2", len(got))
	}
	if _, ok := g.AttributeWildcard(); ok {
		t.Error("AttributeWildcard() ok = true, want false for absent wildcard")
	}
}

func TestNewAttributeGroupDefinitionEmptyUsesYieldsNil(t *testing.T) {
	g, err := xsd.NewAttributeGroupDefinition(xsderr.Loc{}, xsd.QName{Local: "g"}, nil, nil)
	if err != nil {
		t.Fatalf("NewAttributeGroupDefinition: %v", err)
	}
	if got := g.AttributeUses(); got != nil {
		t.Errorf("AttributeUses() = %v, want nil for empty set", got)
	}
}

// TestNewAttributeGroupDefinitionSeedsOnlyDirectUses pins the pre-finalize
// reading AttributeUses documents: before Finalize folds the referenced groups
// in, the property holds the ResolvedAttributeUse members alone, in order, and
// an <attributeGroup ref> contributes nothing yet.
func TestNewAttributeGroupDefinitionSeedsOnlyDirectUses(t *testing.T) {
	a, b := xsd.QName{Local: "a"}, xsd.QName{Local: "b"}
	g, err := xsd.NewAttributeGroupDefinition(xsderr.Loc{}, xsd.QName{Local: "g"}, []xsd.AttributeUseOrGroupRef{
		xsd.ResolvedAttributeUse{Use: useWithRefName(t, a)},
		xsd.AttributeGroupRef{Name: xsd.QName{Local: "other"}},
		xsd.ResolvedAttributeUse{Use: useWithRefName(t, b)},
	}, nil)
	if err != nil {
		t.Fatalf("NewAttributeGroupDefinition: %v", err)
	}
	got := g.AttributeUses()
	if len(got) != 2 || got[0].DeclarationName() != a || got[1].DeclarationName() != b {
		t.Fatalf("AttributeUses() = %v, want the two direct uses [a b] in order", got)
	}
}

// TestNewAttributeGroupDefinitionDefersDuplicateCheck pins where
// ag-props-correct clause 2 moved: the constructor ACCEPTS two members sharing
// an expanded name, because the referenced groups' members are not known until
// Finalize folds them in, and Finalize charges the clause over the folded
// property (see the internal attributegroupfold tests for that half).
func TestNewAttributeGroupDefinitionDefersDuplicateCheck(t *testing.T) {
	dup := xsd.QName{Space: "urn:ns", Local: "a"}
	content := resolvedUses(useWithLocalName(t, dup), useWithRefName(t, dup))
	if _, err := xsd.NewAttributeGroupDefinition(xsderr.Loc{}, xsd.QName{Local: "g"}, content, nil); err != nil {
		t.Fatalf("NewAttributeGroupDefinition(duplicate names) rejected at construction: %v; ag-props-correct clause 2 belongs to Finalize", err)
	}
}

// TestNewAttributeGroupDefinitionRejectsAbsentName exercises ag-props-correct
// clause 1 for the {name} slot: the §3.6.1 tableau types it as a Required
// xs:NCName, and NCName's value space excludes the empty string, so a QName with
// an empty Local is not a legal {name} — with or without a namespace name. A
// present local name in no namespace stays legal (a zero Space is a present name,
// not an absent one).
func TestNewAttributeGroupDefinitionRejectsAbsentName(t *testing.T) {
	tests := []struct {
		name    string
		qname   xsd.QName
		wantErr bool
	}{
		{"zero QName", xsd.QName{}, true},
		{"namespace with empty local", xsd.QName{Space: "urn:ns"}, true},
		{"no-namespace present local", xsd.QName{Local: "g"}, false},
		{"namespaced present local", xsd.QName{Space: "urn:ns", Local: "g"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := xsd.NewAttributeGroupDefinition(xsderr.Loc{}, tc.qname, nil, nil)
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("NewAttributeGroupDefinition(%v) unexpected error: %v", tc.qname, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("NewAttributeGroupDefinition(%v) succeeded, want ag-props-correct clause 1 error", tc.qname)
			}
			assertRule(t, err, "ag-props-correct")
		})
	}
}

// TestAttributeContentRejectsIllegalMembers pins checkAttributeContent through
// every constructor that takes attribute content: each of the four member states
// no arm describes is rejected, charged to RuleComponentInvariant, and the
// message names the offending index and the arm. The complex-type constructors
// share one core, so NewComplexType and NewAnonymousComplexType stand for all
// four; the attribute group constructor calls the helper itself.
func TestAttributeContentRejectsIllegalMembers(t *testing.T) {
	constructors := []struct {
		name  string
		build func(content []xsd.AttributeUseOrGroupRef) error
	}{
		{"NewAttributeGroupDefinition", func(content []xsd.AttributeUseOrGroupRef) error {
			_, err := xsd.NewAttributeGroupDefinition(xsderr.Loc{}, xsd.QName{Local: "g"}, content, nil)
			return err
		}},
		{"NewComplexType", func(content []xsd.AttributeUseOrGroupRef) error {
			_, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "ct"}, xsd.QName{}, nil, xsd.DerivationRestriction, false,
				content, nil, nil, xsd.EmptyContent{}, nil, nil)
			return err
		}},
		{"NewAnonymousComplexType", func(content []xsd.AttributeUseOrGroupRef) error {
			_, err := xsd.NewAnonymousComplexType(xsderr.Loc{}, xsd.ElementDeclarationContext{Component: xsd.NewComponentID()}, xsd.QName{}, nil, xsd.DerivationRestriction, false,
				content, nil, nil, xsd.EmptyContent{}, nil, nil)
			return err
		}},
	}
	members := []struct {
		name   string
		member xsd.AttributeUseOrGroupRef
		msg    string
	}{
		{"nil member", nil, "is nil"},
		{"zero AttributeUse", xsd.ResolvedAttributeUse{}, "wrapping the zero AttributeUse"},
		{"empty ref local part", xsd.AttributeGroupRef{Name: xsd.QName{Space: "urn:ns"}}, "carrying an empty reference"},
		{"zero owned definition", xsd.ResolvedAttributeGroup{}, "with an absent {name}"},
	}
	for _, c := range constructors {
		for _, m := range members {
			t.Run(c.name+"/"+m.name, func(t *testing.T) {
				ok := xsd.ResolvedAttributeUse{Use: useWithRefName(t, xsd.QName{Local: "ok"})}
				err := c.build([]xsd.AttributeUseOrGroupRef{ok, m.member})
				if err == nil {
					t.Fatalf("%s accepted attribute content holding a %s", c.name, m.name)
				}
				assertRule(t, err, xsderr.RuleComponentInvariant)
				if !strings.Contains(err.Error(), "attribute content[1] ") || !strings.Contains(err.Error(), m.msg) {
					t.Fatalf("message %q does not name member [1] and %q", err, m.msg)
				}
			})
		}
	}
}

// TestAttributeContentAcceptsEveryArm is the control for the rejections above:
// one legal member of each arm, a present owned definition included, builds.
func TestAttributeContentAcceptsEveryArm(t *testing.T) {
	original, err := xsd.NewAttributeGroupDefinition(xsderr.Loc{}, xsd.QName{Local: "g"},
		resolvedUses(useWithRefName(t, xsd.QName{Local: "o"})), nil)
	if err != nil {
		t.Fatalf("NewAttributeGroupDefinition(original): %v", err)
	}
	content := []xsd.AttributeUseOrGroupRef{
		xsd.ResolvedAttributeUse{Use: useWithRefName(t, xsd.QName{Local: "a"})},
		xsd.AttributeGroupRef{Name: xsd.QName{Local: "h"}},
		xsd.ResolvedAttributeGroup{Definition: original},
	}
	if _, err := xsd.NewAttributeGroupDefinition(xsderr.Loc{}, xsd.QName{Local: "g"}, content, nil); err != nil {
		t.Fatalf("NewAttributeGroupDefinition rejected one legal member of each arm: %v", err)
	}
	if _, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "ct"}, xsd.QName{}, nil, xsd.DerivationRestriction, false,
		content, nil, nil, xsd.EmptyContent{}, nil, nil); err != nil {
		t.Fatalf("NewComplexType rejected one legal member of each arm: %v", err)
	}
}

func TestNewAttributeGroupDefinitionWildcardPresent(t *testing.T) {
	nc, err := xsd.NewNamespaceConstraint(xsderr.Loc{}, xsd.NamespaceConstraintAny, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewNamespaceConstraint: %v", err)
	}
	w, err := xsd.NewWildcard(xsderr.Loc{}, nc, xsd.ProcessLax)
	if err != nil {
		t.Fatalf("NewWildcard: %v", err)
	}
	g, err := xsd.NewAttributeGroupDefinition(xsderr.Loc{}, xsd.QName{Local: "g"}, nil, &w)
	if err != nil {
		t.Fatalf("NewAttributeGroupDefinition: %v", err)
	}
	gotW, ok := g.AttributeWildcard()
	if !ok {
		t.Fatal("AttributeWildcard() ok = false, want true")
	}
	if gotW.ProcessContents() != xsd.ProcessLax {
		t.Errorf("wildcard {process contents} = %v, want lax", gotW.ProcessContents())
	}
}

func TestAttributeGroupDefinitionUsesAccessorDoesNotAlias(t *testing.T) {
	content := resolvedUses(useWithLocalName(t, xsd.QName{Local: "a"}))
	g, err := xsd.NewAttributeGroupDefinition(xsderr.Loc{}, xsd.QName{Local: "g"}, content, nil)
	if err != nil {
		t.Fatalf("NewAttributeGroupDefinition: %v", err)
	}
	// The accessor returns a copy.
	first := g.AttributeUses()
	first[0] = useWithLocalName(t, xsd.QName{Local: "tampered"})
	second := g.AttributeUses()
	name := xsd.QName{Local: "a"}
	if second[0].AttributeDeclaration().(xsd.LocalAttributeDeclaration).Declaration.Name() != name {
		t.Error("AttributeUses() returned an aliased slice")
	}
	// The constructor does not alias the caller's backing array.
	content[0] = xsd.ResolvedAttributeUse{Use: useWithLocalName(t, xsd.QName{Local: "tampered"})}
	if g.AttributeUses()[0].AttributeDeclaration().(xsd.LocalAttributeDeclaration).Declaration.Name() != name {
		t.Error("AttributeGroupDefinition aliased the constructor content slice")
	}
}
