package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// adLocalScope is a local attribute {scope} whose {parent} names a containing
// complex type called container, for the tests that only need a non-global
// declaration.
func adLocalScope(t *testing.T) xsd.AttributeScope {
	t.Helper()
	s, err := xsd.NewAttributeLocalScope(xsderr.Loc{}, xsd.AttributeComplexTypeScopeParent{Name: xsd.QName{Local: "container"}})
	if err != nil {
		t.Fatalf("NewAttributeLocalScope: %v", err)
	}
	return s
}

func TestNewAttributeDeclarationValidGlobalNoValueConstraint(t *testing.T) {
	name := xsd.QName{Space: "urn:ns", Local: "lang"}
	typ := xsd.QName{Space: "urn:t", Local: "LangType"}
	a, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, name, xsd.TypeDefinitionRef{Name: typ}, xsd.NewAttributeGlobalScope(), nil, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeDeclaration unexpected error: %v", err)
	}
	if a.Name() != name {
		t.Errorf("Name() = %v, want %v", a.Name(), name)
	}
	if got := a.TypeDefinition(); got != (xsd.TypeDefinitionOrRef)(xsd.TypeDefinitionRef{Name: typ}) {
		t.Errorf("TypeDefinition() = %v, want a TypeDefinitionRef naming %v", got, typ)
	}
	if a.ScopeVariety() != xsd.ScopeGlobal {
		t.Errorf("ScopeVariety() = %v, want global", a.ScopeVariety())
	}
	if a.Inheritable() {
		t.Error("Inheritable() = true, want false")
	}
	if _, ok := a.ValueConstraint(); ok {
		t.Error("ValueConstraint() ok = true, want false for absent value constraint")
	}
	if got := a.Annotations(); got != nil {
		t.Errorf("Annotations() = %v, want nil", got)
	}
}

func TestNewAttributeDeclarationValueConstraintAndInheritablePresent(t *testing.T) {
	vc := xsd.NewValueConstraint(xsd.ValueFixed, "en", nil, nil)
	a, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, xsd.QName{Local: "a"}, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, adLocalScope(t), &vc, true, nil)
	if err != nil {
		t.Fatalf("NewAttributeDeclaration: %v", err)
	}
	if a.ScopeVariety() != xsd.ScopeLocal {
		t.Errorf("ScopeVariety() = %v, want local", a.ScopeVariety())
	}
	if !a.Inheritable() {
		t.Error("Inheritable() = false, want true")
	}
	gotVC, ok := a.ValueConstraint()
	if !ok {
		t.Fatal("ValueConstraint() ok = false, want true")
	}
	if gotVC.Kind() != xsd.ValueFixed || gotVC.LexicalForm() != "en" {
		t.Errorf("ValueConstraint() = (%v, %q), want (fixed, en)", gotVC.Kind(), gotVC.LexicalForm())
	}
}

// TestNewAttributeDeclarationRejectsAbsentName exercises a-props-correct clause
// 1 for the {name} slot: the §3.2.1 tableau types it as a Required xs:NCName, and
// NCName's value space excludes the empty string, so a QName with an empty Local
// is not a legal {name} — with or without a namespace name. A present local name
// in no namespace stays legal (a zero Space is a present name, not an absent one).
func TestNewAttributeDeclarationRejectsAbsentName(t *testing.T) {
	tests := []struct {
		name    string
		qname   xsd.QName
		wantErr bool
	}{
		{"zero QName", xsd.QName{}, true},
		{"namespace with empty local", xsd.QName{Space: "urn:ns"}, true},
		{"no-namespace present local", xsd.QName{Local: "a"}, false},
		{"namespaced present local", xsd.QName{Space: "urn:ns", Local: "a"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, tc.qname, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, xsd.NewAttributeGlobalScope(), nil, false, nil)
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("NewAttributeDeclaration(%v) unexpected error: %v", tc.qname, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("NewAttributeDeclaration(%v) succeeded, want a-props-correct clause 1 error", tc.qname)
			}
			assertRule(t, err, "a-props-correct")
		})
	}
}

// TestNewAttributeGlobalScope pins §3.2.2.1 dcl.att.global: {variety} global,
// {parent} ·absent·.
func TestNewAttributeGlobalScope(t *testing.T) {
	s := xsd.NewAttributeGlobalScope()
	if s.Variety() != xsd.ScopeGlobal {
		t.Errorf("Variety() = %v, want global", s.Variety())
	}
	parent, ok := s.Parent()
	if ok {
		t.Errorf("Parent() ok = true, want false for a global scope (got %#v)", parent)
	}
	if parent != nil {
		t.Errorf("Parent() = %#v, want nil for a global scope", parent)
	}
}

// TestNewAttributeLocalScopeCarriesParent pins §3.2.2.2 dcl.att.local for every
// target kind: {variety} is local (derived from {parent}'s presence, never
// stored) and {parent} reads back as the very variant it was built from — a
// Complex Type Definition for an <attribute> under a <complexType>, an Attribute
// Group Definition for one within an <attributeGroup>.
func TestNewAttributeLocalScopeCarriesParent(t *testing.T) {
	ct := xsd.QName{Space: "urn:ns", Local: "AddressType"}
	agd := xsd.QName{Space: "urn:ns", Local: "addressAttrs"}
	for _, tc := range []struct {
		name string
		want xsd.AttributeScopeParent
	}{
		{"complex type definition", xsd.AttributeComplexTypeScopeParent{Name: ct}},
		{"attribute group definition", xsd.AttributeGroupScopeParent{Name: agd}},
		{"anonymous complex type definition", xsd.AttributeAnonymousComplexTypeScopeParent{Owner: xsd.NewComponentID()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := xsd.NewAttributeLocalScope(xsderr.Loc{}, tc.want)
			if err != nil {
				t.Fatalf("NewAttributeLocalScope(%#v): %v", tc.want, err)
			}
			if s.Variety() != xsd.ScopeLocal {
				t.Errorf("Variety() = %v, want local", s.Variety())
			}
			got, ok := s.Parent()
			if !ok {
				t.Fatal("Parent() ok = false, want true for a local scope")
			}
			if got != tc.want {
				t.Errorf("Parent() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

// TestNewAttributeLocalScopeRejectsUnusableParent pins the two states
// NewAttributeLocalScope refuses (a-props-correct clause 1): an absent {parent},
// which the §3.2.1 tableau requires to be present when {variety} is local, and a
// variant that identifies nothing — an absent name on either by-NAME arm, or an
// unminted identity on the anonymous arm, neither of which could ever be
// followed. The rule charged is a-props-correct, never the element side's
// e-props-correct.
func TestNewAttributeLocalScopeRejectsUnusableParent(t *testing.T) {
	for _, tc := range []struct {
		name   string
		parent xsd.AttributeScopeParent
	}{
		{"absent parent", nil},
		{"unnamed complex type", xsd.AttributeComplexTypeScopeParent{}},
		{"unnamed attribute group", xsd.AttributeGroupScopeParent{}},
		{"unminted anonymous complex type", xsd.AttributeAnonymousComplexTypeScopeParent{}},
		{"namespace but no local name", xsd.AttributeComplexTypeScopeParent{Name: xsd.QName{Space: "urn:ns"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := xsd.NewAttributeLocalScope(xsderr.Loc{}, tc.parent)
			if err == nil {
				t.Fatalf("NewAttributeLocalScope(%#v) succeeded, want an a-props-correct clause 1 rejection", tc.parent)
			}
			assertRule(t, err, "a-props-correct")
		})
	}
}

// TestAttributeDeclarationScopeRoundTrip is the containment round trip: a local
// attribute declaration nested in a named container names that container back,
// and the name is the one the container itself reports, so a consumer can go from
// the declaration to its {scope}.{parent} component with a schema lookup. The
// global declaration alongside it carries no {parent} at all (§3.2.2.1).
func TestAttributeDeclarationScopeRoundTrip(t *testing.T) {
	container, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Space: "urn:ns", Local: "AddressType"},
		xsd.QName{Local: "anyType"}, nil, xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType: %v", err)
	}
	groupDef, err := xsd.NewAttributeGroupDefinition(xsderr.Loc{}, xsd.QName{Space: "urn:ns", Local: "addressAttrs"}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewAttributeGroupDefinition: %v", err)
	}

	for _, tc := range []struct {
		name   string
		parent xsd.AttributeScopeParent
		want   xsd.QName
	}{
		{"in a complex type", xsd.AttributeComplexTypeScopeParent{Name: container.Name()}, container.Name()},
		{"in an attribute group definition", xsd.AttributeGroupScopeParent{Name: groupDef.Name()}, groupDef.Name()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scope, err := xsd.NewAttributeLocalScope(xsderr.Loc{}, tc.parent)
			if err != nil {
				t.Fatalf("NewAttributeLocalScope: %v", err)
			}
			a, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, xsd.QName{Space: "urn:ns", Local: "country"},
				xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, scope, nil, false, nil)
			if err != nil {
				t.Fatalf("NewAttributeDeclaration: %v", err)
			}
			if a.ScopeVariety() != xsd.ScopeLocal {
				t.Errorf("ScopeVariety() = %v, want local", a.ScopeVariety())
			}
			got, ok := a.Scope().Parent()
			if !ok {
				t.Fatal("Scope().Parent() ok = false, want true for a local declaration")
			}
			if got != tc.parent {
				t.Errorf("Scope().Parent() = %#v, want %#v", got, tc.parent)
			}
			var name xsd.QName
			switch p := got.(type) {
			case xsd.AttributeComplexTypeScopeParent:
				name = p.Name
			case xsd.AttributeGroupScopeParent:
				name = p.Name
			default:
				t.Fatalf("Scope().Parent() = %T, want one of the two by-name variants", got)
			}
			if name != tc.want {
				t.Errorf("{scope}.{parent} names %v, but the container reports %v", name, tc.want)
			}
		})
	}

	global, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, xsd.QName{Space: "urn:ns", Local: "top"},
		xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, xsd.NewAttributeGlobalScope(), nil, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeDeclaration(global): %v", err)
	}
	if global.ScopeVariety() != xsd.ScopeGlobal {
		t.Errorf("global ScopeVariety() = %v, want global", global.ScopeVariety())
	}
	if parent, ok := global.Scope().Parent(); ok {
		t.Errorf("global Scope().Parent() = %#v, want absent (§3.2.2.1)", parent)
	}
}

func TestNewAttributeDeclarationRejectsUnknownValueConstraintKind(t *testing.T) {
	// A zero ValueConstraint carries the invalid zero ValueConstraintKind.
	bad := xsd.ValueConstraint{}
	_, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, xsd.QName{Local: "a"}, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, xsd.NewAttributeGlobalScope(), &bad, false, nil)
	if err == nil {
		t.Fatal("NewAttributeDeclaration(zero value constraint) succeeded, want a-props-correct error")
	}
	assertRule(t, err, "a-props-correct")
}

func TestAttributeDeclarationAnnotationsRoundTripAndAlias(t *testing.T) {
	anns := []xsd.Annotation{
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "doc")}, nil),
	}
	a, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, xsd.QName{Local: "a"}, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, xsd.NewAttributeGlobalScope(), nil, false, anns)
	if err != nil {
		t.Fatalf("NewAttributeDeclaration: %v", err)
	}
	if got := a.Annotations(); len(got) != 1 || got[0].Documentation()[0].Content() != "doc" {
		t.Errorf("Annotations() = %+v, want one with content doc", got)
	}
	// The constructor must not alias the caller's backing array.
	anns[0] = xsd.NewAnnotation(nil, nil, nil)
	if got := a.Annotations(); got[0].Documentation()[0].Content() != "doc" {
		t.Error("AttributeDeclaration aliased the constructor annotations slice")
	}
	// The accessor must not alias the stored slice.
	first := a.Annotations()
	first[0] = xsd.NewAnnotation(nil, nil, nil)
	if got := a.Annotations(); got[0].Documentation()[0].Content() != "doc" {
		t.Error("Annotations() returned an aliased slice")
	}
}
