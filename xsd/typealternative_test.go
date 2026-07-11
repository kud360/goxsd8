package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

func TestNewTypeAlternativeTestPresent(t *testing.T) {
	test := xsd.NewXPathExpression("@a > 0", []xsd.NamespaceBinding{xsd.NewNamespaceBinding("p", "urn:ns")}, strptr("urn:dflt"), nil)
	ta := xsd.NewTypeAlternative(&test, xsd.QName{Space: "urn:t", Local: "Even"}, nil)

	got, ok := ta.Test()
	if !ok {
		t.Fatal("Test() ok = false, want true for a present {test}")
	}
	if got.Expression() != "@a > 0" {
		t.Errorf("Test().Expression() = %q, want %q", got.Expression(), "@a > 0")
	}
	binds := got.NamespaceBindings()
	if len(binds) != 1 || binds[0].Prefix() != "p" || binds[0].Namespace() != "urn:ns" {
		t.Errorf("Test().NamespaceBindings() = %+v, want [p=urn:ns]", binds)
	}
	if ns, ok := got.DefaultNamespace(); !ok || ns != "urn:dflt" {
		t.Errorf("Test().DefaultNamespace() = (%q, %v), want (%q, true)", ns, ok, "urn:dflt")
	}
}

func TestNewTypeAlternativeTestAbsent(t *testing.T) {
	ta := xsd.NewTypeAlternative(nil, xsd.QName{Space: "urn:t", Local: "Default"}, nil)

	if got, ok := ta.Test(); ok {
		t.Errorf("Test() = (%+v, true), want (_, false) for the absent \"otherwise\" alternative", got)
	}
}

func TestTypeAlternativeTypeDefinitionNameRoundTrip(t *testing.T) {
	name := xsd.QName{Space: "urn:t", Local: "Even"}
	ta := xsd.NewTypeAlternative(nil, name, nil)

	if got := ta.TypeDefinitionName(); got != name {
		t.Errorf("TypeDefinitionName() = %+v, want %+v", got, name)
	}
}

func TestNewTypeAlternativeAnnotationsRoundTrip(t *testing.T) {
	anns := []xsd.Annotation{
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "first")}, nil),
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "second")}, nil),
	}
	ta := xsd.NewTypeAlternative(nil, xsd.QName{Local: "T"}, anns)

	got := ta.Annotations()
	if len(got) != 2 {
		t.Fatalf("Annotations() len = %d, want 2", len(got))
	}
	if docs := got[0].Documentation(); len(docs) != 1 || docs[0].Content() != "first" {
		t.Errorf("Annotations()[0] documentation = %+v, want content %q", docs, "first")
	}
}

func TestTypeAlternativeAnnotationsNilWhenEmpty(t *testing.T) {
	if got := xsd.NewTypeAlternative(nil, xsd.QName{Local: "T"}, nil).Annotations(); got != nil {
		t.Errorf("Annotations() = %v, want nil for nil input", got)
	}
	if got := xsd.NewTypeAlternative(nil, xsd.QName{Local: "T"}, []xsd.Annotation{}).Annotations(); got != nil {
		t.Errorf("Annotations() = %v, want nil for empty-slice input", got)
	}
}

func TestTypeAlternativeDoesNotAliasConstructorAnnotations(t *testing.T) {
	anns := []xsd.Annotation{xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "keep")}, nil)}
	ta := xsd.NewTypeAlternative(nil, xsd.QName{Local: "T"}, anns)

	// Mutate the ORIGINAL backing array.
	anns[0] = xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "tampered")}, nil)

	if docs := ta.Annotations()[0].Documentation(); docs[0].Content() != "keep" {
		t.Errorf("TypeAlternative aliased the constructor slice: got %q, want %q", docs[0].Content(), "keep")
	}
}

func TestTypeAlternativeAnnotationsAccessorDoesNotAlias(t *testing.T) {
	ta := xsd.NewTypeAlternative(
		nil,
		xsd.QName{Local: "T"},
		[]xsd.Annotation{xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "keep")}, nil)},
	)

	// Mutate the RETURNED slice; a second call must be unaffected.
	ta.Annotations()[0] = xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "tampered")}, nil)

	if docs := ta.Annotations()[0].Documentation(); docs[0].Content() != "keep" {
		t.Errorf("Annotations() returned an aliased slice: got %q, want %q", docs[0].Content(), "keep")
	}
}
