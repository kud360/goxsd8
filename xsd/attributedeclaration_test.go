package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

func TestNewAttributeDeclarationValidGlobalNoValueConstraint(t *testing.T) {
	name := xsd.QName{Space: "urn:ns", Local: "lang"}
	typ := xsd.QName{Space: "urn:t", Local: "LangType"}
	a, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, name, xsd.TypeDefinitionRef{Name: typ}, xsd.ScopeGlobal, nil, false, nil)
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
	vc := xsd.NewValueConstraint(xsd.ValueFixed, "en")
	a, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, xsd.QName{Local: "a"}, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, xsd.ScopeLocal, &vc, true, nil)
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
			_, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, tc.qname, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, xsd.ScopeGlobal, nil, false, nil)
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

func TestNewAttributeDeclarationRejectsUnknownScope(t *testing.T) {
	_, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, xsd.QName{Local: "a"}, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, xsd.ScopeVariety(0), nil, false, nil)
	if err == nil {
		t.Fatal("NewAttributeDeclaration(scope=0) succeeded, want a-props-correct error")
	}
	assertRule(t, err, "a-props-correct")
}

func TestNewAttributeDeclarationRejectsUnknownValueConstraintKind(t *testing.T) {
	// A zero ValueConstraint carries the invalid zero ValueConstraintKind.
	bad := xsd.ValueConstraint{}
	_, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, xsd.QName{Local: "a"}, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, xsd.ScopeGlobal, &bad, false, nil)
	if err == nil {
		t.Fatal("NewAttributeDeclaration(zero value constraint) succeeded, want a-props-correct error")
	}
	assertRule(t, err, "a-props-correct")
}

func TestAttributeDeclarationAnnotationsRoundTripAndAlias(t *testing.T) {
	anns := []xsd.Annotation{
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "doc")}, nil),
	}
	a, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, xsd.QName{Local: "a"}, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, xsd.ScopeGlobal, nil, false, anns)
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
