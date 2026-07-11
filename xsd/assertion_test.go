package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

func TestNewAssertionTestRoundTrip(t *testing.T) {
	test := xsd.NewXPathExpression("@a > 0", []xsd.NamespaceBinding{xsd.NewNamespaceBinding("p", "urn:ns")}, strptr("urn:dflt"), nil)
	a := xsd.NewAssertion(test, nil)

	got := a.Test()
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

func TestNewAssertionAnnotationsRoundTrip(t *testing.T) {
	anns := []xsd.Annotation{
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "first")}, nil),
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "second")}, nil),
	}
	a := xsd.NewAssertion(xsd.NewXPathExpression("t", nil, nil, nil), anns)

	got := a.Annotations()
	if len(got) != 2 {
		t.Fatalf("Annotations() len = %d, want 2", len(got))
	}
	if docs := got[0].Documentation(); len(docs) != 1 || docs[0].Content() != "first" {
		t.Errorf("Annotations()[0] documentation = %+v, want content %q", docs, "first")
	}
}

func TestAssertionAnnotationsNilWhenEmpty(t *testing.T) {
	if got := xsd.NewAssertion(xsd.NewXPathExpression("t", nil, nil, nil), nil).Annotations(); got != nil {
		t.Errorf("Annotations() = %v, want nil for nil input", got)
	}
	if got := xsd.NewAssertion(xsd.NewXPathExpression("t", nil, nil, nil), []xsd.Annotation{}).Annotations(); got != nil {
		t.Errorf("Annotations() = %v, want nil for empty-slice input", got)
	}
}

func TestAssertionDoesNotAliasConstructorAnnotations(t *testing.T) {
	anns := []xsd.Annotation{xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "keep")}, nil)}
	a := xsd.NewAssertion(xsd.NewXPathExpression("t", nil, nil, nil), anns)

	// Mutate the ORIGINAL backing array.
	anns[0] = xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "tampered")}, nil)

	if docs := a.Annotations()[0].Documentation(); docs[0].Content() != "keep" {
		t.Errorf("Assertion aliased the constructor slice: got %q, want %q", docs[0].Content(), "keep")
	}
}

func TestAssertionAnnotationsAccessorDoesNotAlias(t *testing.T) {
	a := xsd.NewAssertion(
		xsd.NewXPathExpression("t", nil, nil, nil),
		[]xsd.Annotation{xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "keep")}, nil)},
	)

	// Mutate the RETURNED slice; a second call must be unaffected.
	a.Annotations()[0] = xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "tampered")}, nil)

	if docs := a.Annotations()[0].Documentation(); docs[0].Content() != "keep" {
		t.Errorf("Annotations() returned an aliased slice: got %q, want %q", docs[0].Content(), "keep")
	}
}
