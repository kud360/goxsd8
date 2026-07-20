package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

func TestNewAttributeDeclarationValidGlobalNoValueConstraint(t *testing.T) {
	name := xsd.QName{Space: "urn:ns", Local: "lang"}
	typ := xsd.QName{Space: "urn:t", Local: "LangType"}
	a, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, name, typ, xsd.ScopeGlobal, nil, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeDeclaration unexpected error: %v", err)
	}
	if a.Name() != name {
		t.Errorf("Name() = %v, want %v", a.Name(), name)
	}
	if a.TypeDefinitionName() != typ {
		t.Errorf("TypeDefinitionName() = %v, want %v", a.TypeDefinitionName(), typ)
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
	vc := xsd.NewValueConstraint(xsd.ValueFixed, "en")
	a, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, xsd.QName{Local: "a"}, xsd.QName{Local: "T"}, xsd.ScopeLocal, &vc, true, nil)
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

func TestNewAttributeDeclarationRejectsUnknownScope(t *testing.T) {
	_, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, xsd.QName{Local: "a"}, xsd.QName{Local: "T"}, xsd.ScopeVariety(0), nil, false, nil)
	if err == nil {
		t.Fatal("NewAttributeDeclaration(scope=0) succeeded, want a-props-correct error")
	}
	assertRule(t, err, "a-props-correct")
}

func TestNewAttributeDeclarationRejectsUnknownValueConstraintKind(t *testing.T) {
	// A zero ValueConstraint carries the invalid zero ValueConstraintKind.
	bad := xsd.ValueConstraint{}
	_, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, xsd.QName{Local: "a"}, xsd.QName{Local: "T"}, xsd.ScopeGlobal, &bad, false, nil)
	if err == nil {
		t.Fatal("NewAttributeDeclaration(zero value constraint) succeeded, want a-props-correct error")
	}
	assertRule(t, err, "a-props-correct")
}

func TestAttributeDeclarationAnnotationsRoundTripAndAlias(t *testing.T) {
	anns := []xsd.Annotation{
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "doc")}, nil),
	}
	a, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, xsd.QName{Local: "a"}, xsd.QName{Local: "T"}, xsd.ScopeGlobal, nil, false, anns)
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
