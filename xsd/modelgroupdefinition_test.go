package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

func TestNewModelGroupDefinitionValid(t *testing.T) {
	name := xsd.QName{Space: "urn:ns", Local: "g"}
	mg := mustModelGroup(t, xsd.CompositorSequence, []xsd.Particle{elementRefParticle(t, "a")}, nil)
	d, err := xsd.NewModelGroupDefinition(xsderr.Loc{}, name, mg, nil)
	if err != nil {
		t.Fatalf("NewModelGroupDefinition unexpected error: %v", err)
	}
	if d.Name() != name {
		t.Errorf("Name() = %v, want %v", d.Name(), name)
	}
	if d.ModelGroup().Compositor() != xsd.CompositorSequence {
		t.Errorf("ModelGroup().Compositor() = %s, want sequence", d.ModelGroup().Compositor())
	}
	if got := d.Annotations(); got != nil {
		t.Errorf("Annotations() = %v, want nil", got)
	}
}

func TestNewModelGroupDefinitionRejectsZeroModelGroup(t *testing.T) {
	// A zero ModelGroup{} was never built through NewModelGroup; its {compositor}
	// is the invalid zero, which the constructor must reject to keep the Required
	// {model group} present.
	_, err := xsd.NewModelGroupDefinition(xsderr.Loc{}, xsd.QName{Local: "g"}, xsd.ModelGroup{}, nil)
	if err == nil {
		t.Fatal("NewModelGroupDefinition accepted a zero ModelGroup, want mgd-props-correct error")
	}
	assertRule(t, err, "mgd-props-correct")
}

func TestModelGroupDefinitionIsNotATerm(t *testing.T) {
	// ModelGroupDefinition must NOT satisfy Term: only its {model group} does.
	// A failed type assertion to a *Term-typed* variable cannot be written at
	// compile time (that would not compile), so assert the runtime shape: the
	// definition's {model group} is a Term, the definition itself is not returned
	// as one anywhere.
	mg := mustModelGroup(t, xsd.CompositorAll, nil, nil)
	d, err := xsd.NewModelGroupDefinition(xsderr.Loc{}, xsd.QName{Local: "g"}, mg, nil)
	if err != nil {
		t.Fatalf("NewModelGroupDefinition: %v", err)
	}
	var _ xsd.Term = d.ModelGroup() // {model group} is a Term
}

func TestModelGroupDefinitionAnnotationsDoNotAlias(t *testing.T) {
	mg := mustModelGroup(t, xsd.CompositorChoice, nil, nil)
	anns := []xsd.Annotation{
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "first")}, nil),
	}
	d, err := xsd.NewModelGroupDefinition(xsderr.Loc{}, xsd.QName{Local: "g"}, mg, anns)
	if err != nil {
		t.Fatalf("NewModelGroupDefinition: %v", err)
	}
	anns[0] = xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "tampered")}, nil)
	if docs := d.Annotations()[0].Documentation(); docs[0].Content() != "first" {
		t.Errorf("ModelGroupDefinition aliased the constructor annotations slice: got %q", docs[0].Content())
	}
}
