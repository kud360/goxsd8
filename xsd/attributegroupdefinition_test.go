package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// useWithLocalName builds an Attribute Use whose {attribute declaration} is a
// local declaration with the given expanded name.
func useWithLocalName(t *testing.T, name xsd.QName) xsd.AttributeUse {
	t.Helper()
	u, err := xsd.NewAttributeUse(xsderr.Loc{}, false, localDecl(t, name), nil, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeUse: %v", err)
	}
	return u
}

// useWithRefName builds an Attribute Use whose {attribute declaration} is a
// deferred reference with the given expanded name.
func useWithRefName(t *testing.T, name xsd.QName) xsd.AttributeUse {
	t.Helper()
	u, err := xsd.NewAttributeUse(xsderr.Loc{}, false, xsd.AttributeDeclarationRef{Name: name}, nil, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeUse: %v", err)
	}
	return u
}

func TestNewAttributeGroupDefinitionValid(t *testing.T) {
	name := xsd.QName{Space: "urn:ns", Local: "g"}
	uses := []xsd.AttributeUse{
		useWithLocalName(t, xsd.QName{Space: "urn:ns", Local: "a"}),
		useWithRefName(t, xsd.QName{Space: "urn:ns", Local: "b"}),
	}
	g, err := xsd.NewAttributeGroupDefinition(xsderr.Loc{}, name, uses, nil, nil)
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
	if got := g.Annotations(); got != nil {
		t.Errorf("Annotations() = %v, want nil", got)
	}
}

func TestNewAttributeGroupDefinitionEmptyUsesYieldsNil(t *testing.T) {
	g, err := xsd.NewAttributeGroupDefinition(xsderr.Loc{}, xsd.QName{Local: "g"}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewAttributeGroupDefinition: %v", err)
	}
	if got := g.AttributeUses(); got != nil {
		t.Errorf("AttributeUses() = %v, want nil for empty set", got)
	}
}

func TestNewAttributeGroupDefinitionRejectsDuplicateExpandedNameLocalLocal(t *testing.T) {
	dup := xsd.QName{Space: "urn:ns", Local: "a"}
	uses := []xsd.AttributeUse{
		useWithLocalName(t, dup),
		useWithLocalName(t, dup),
	}
	_, err := xsd.NewAttributeGroupDefinition(xsderr.Loc{}, xsd.QName{Local: "g"}, uses, nil, nil)
	if err == nil {
		t.Fatal("NewAttributeGroupDefinition(duplicate names) succeeded, want ag-props-correct error")
	}
	assertRule(t, err, "ag-props-correct")
}

func TestNewAttributeGroupDefinitionRejectsDuplicateExpandedNameLocalRef(t *testing.T) {
	// The duplicate is detected across the two sum variants: a local declaration
	// and a ref that share an expanded name.
	dup := xsd.QName{Space: "urn:ns", Local: "a"}
	uses := []xsd.AttributeUse{
		useWithLocalName(t, dup),
		useWithRefName(t, dup),
	}
	_, err := xsd.NewAttributeGroupDefinition(xsderr.Loc{}, xsd.QName{Local: "g"}, uses, nil, nil)
	if err == nil {
		t.Fatal("NewAttributeGroupDefinition(duplicate across variants) succeeded, want ag-props-correct error")
	}
	assertRule(t, err, "ag-props-correct")
}

func TestNewAttributeGroupDefinitionDistinctNamespacesNotDuplicate(t *testing.T) {
	// Same local name in different namespaces are distinct expanded names.
	uses := []xsd.AttributeUse{
		useWithLocalName(t, xsd.QName{Space: "urn:x", Local: "a"}),
		useWithLocalName(t, xsd.QName{Space: "urn:y", Local: "a"}),
	}
	if _, err := xsd.NewAttributeGroupDefinition(xsderr.Loc{}, xsd.QName{Local: "g"}, uses, nil, nil); err != nil {
		t.Fatalf("NewAttributeGroupDefinition(distinct namespaces) error: %v", err)
	}
}

func TestNewAttributeGroupDefinitionWildcardPresent(t *testing.T) {
	nc, err := xsd.NewNamespaceConstraint(xsderr.Loc{}, xsd.NamespaceConstraintAny, nil, nil)
	if err != nil {
		t.Fatalf("NewNamespaceConstraint: %v", err)
	}
	w, err := xsd.NewWildcard(xsderr.Loc{}, nc, xsd.ProcessLax, nil)
	if err != nil {
		t.Fatalf("NewWildcard: %v", err)
	}
	g, err := xsd.NewAttributeGroupDefinition(xsderr.Loc{}, xsd.QName{Local: "g"}, nil, &w, nil)
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
	uses := []xsd.AttributeUse{useWithLocalName(t, xsd.QName{Local: "a"})}
	g, err := xsd.NewAttributeGroupDefinition(xsderr.Loc{}, xsd.QName{Local: "g"}, uses, nil, nil)
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
	uses[0] = useWithLocalName(t, xsd.QName{Local: "tampered"})
	if g.AttributeUses()[0].AttributeDeclaration().(xsd.LocalAttributeDeclaration).Declaration.Name() != name {
		t.Error("AttributeGroupDefinition aliased the constructor uses slice")
	}
}
