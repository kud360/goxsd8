package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// localDecl builds a LocalAttributeDeclaration wrapping a global-scope
// declaration with the given name, for use in Attribute Use tests.
func localDecl(t *testing.T, name xsd.QName) xsd.LocalAttributeDeclaration {
	t.Helper()
	d, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, name, xsd.QName{Local: "T"}, xsd.ScopeLocal, nil, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeDeclaration: %v", err)
	}
	return xsd.LocalAttributeDeclaration{Declaration: d}
}

func TestNewAttributeUseValidLocalDeclaration(t *testing.T) {
	decl := localDecl(t, xsd.QName{Space: "urn:ns", Local: "a"})
	u, err := xsd.NewAttributeUse(xsderr.Loc{}, true, decl, true, nil)
	if err != nil {
		t.Fatalf("NewAttributeUse unexpected error: %v", err)
	}
	if !u.Required() {
		t.Error("Required() = false, want true")
	}
	if !u.Inheritable() {
		t.Error("Inheritable() = false, want true")
	}
	got, ok := u.AttributeDeclaration().(xsd.LocalAttributeDeclaration)
	if !ok {
		t.Fatalf("AttributeDeclaration() type = %T, want LocalAttributeDeclaration", u.AttributeDeclaration())
	}
	if got.Declaration.Name() != (xsd.QName{Space: "urn:ns", Local: "a"}) {
		t.Errorf("declaration name = %v, want {urn:ns}a", got.Declaration.Name())
	}
	if u.Annotations() != nil {
		t.Errorf("Annotations() = %v, want nil", u.Annotations())
	}
}

func TestNewAttributeUseValidRef(t *testing.T) {
	ref := xsd.AttributeDeclarationRef{Name: xsd.QName{Space: "urn:ns", Local: "b"}}
	u, err := xsd.NewAttributeUse(xsderr.Loc{}, false, ref, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeUse unexpected error: %v", err)
	}
	if u.Required() {
		t.Error("Required() = true, want false")
	}
	got, ok := u.AttributeDeclaration().(xsd.AttributeDeclarationRef)
	if !ok {
		t.Fatalf("AttributeDeclaration() type = %T, want AttributeDeclarationRef", u.AttributeDeclaration())
	}
	if got.Name != (xsd.QName{Space: "urn:ns", Local: "b"}) {
		t.Errorf("ref name = %v, want {urn:ns}b", got.Name)
	}
}

func TestNewAttributeUseRejectsNilDeclaration(t *testing.T) {
	_, err := xsd.NewAttributeUse(xsderr.Loc{}, false, nil, false, nil)
	if err == nil {
		t.Fatal("NewAttributeUse(nil declaration) succeeded, want au-props-correct error")
	}
	assertRule(t, err, "au-props-correct")
}

func TestAttributeUseAnnotationsRoundTripAndAlias(t *testing.T) {
	anns := []xsd.Annotation{
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "u")}, nil),
	}
	u, err := xsd.NewAttributeUse(xsderr.Loc{}, false, localDecl(t, xsd.QName{Local: "a"}), false, anns)
	if err != nil {
		t.Fatalf("NewAttributeUse: %v", err)
	}
	if got := u.Annotations(); len(got) != 1 || got[0].Documentation()[0].Content() != "u" {
		t.Errorf("Annotations() = %+v, want one with content u", got)
	}
	anns[0] = xsd.NewAnnotation(nil, nil, nil)
	if got := u.Annotations(); got[0].Documentation()[0].Content() != "u" {
		t.Error("AttributeUse aliased the constructor annotations slice")
	}
	first := u.Annotations()
	first[0] = xsd.NewAnnotation(nil, nil, nil)
	if got := u.Annotations(); got[0].Documentation()[0].Content() != "u" {
		t.Error("Annotations() returned an aliased slice")
	}
}
