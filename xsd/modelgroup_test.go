package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// mustModelGroup fails the test if construction errors; construction-rejection
// cases use NewModelGroup directly.
func mustModelGroup(t *testing.T, c xsd.Compositor, ps []xsd.Particle, anns []xsd.Annotation) xsd.ModelGroup {
	t.Helper()
	g, err := xsd.NewModelGroup(xsderr.Loc{}, c, ps, anns)
	if err != nil {
		t.Fatalf("NewModelGroup(%s) unexpected error: %v", c, err)
	}
	return g
}

// elementParticle builds a particle whose {term} is an inline element
// declaration reference, for populating model groups.
func elementRefParticle(t *testing.T, local string) xsd.Particle {
	t.Helper()
	p, err := xsd.NewParticle(xsderr.Loc{}, xsd.Occurs{}, xsd.ElementDeclarationRef{Name: xsd.QName{Local: local}}, nil)
	if err != nil {
		t.Fatalf("NewParticle: %v", err)
	}
	return p
}

func TestNewModelGroupValidCompositors(t *testing.T) {
	for _, c := range []xsd.Compositor{xsd.CompositorAll, xsd.CompositorChoice, xsd.CompositorSequence} {
		t.Run(c.String(), func(t *testing.T) {
			g := mustModelGroup(t, c, nil, nil)
			if g.Compositor() != c {
				t.Errorf("Compositor() = %s, want %s", g.Compositor(), c)
			}
			if got := g.Particles(); got != nil {
				t.Errorf("Particles() = %v, want nil for empty {particles}", got)
			}
		})
	}
}

func TestNewModelGroupRejectsZeroCompositor(t *testing.T) {
	_, err := xsd.NewModelGroup(xsderr.Loc{}, xsd.Compositor(0), nil, nil)
	if err == nil {
		t.Fatal("NewModelGroup accepted a zero Compositor, want mg-props-correct error")
	}
	assertRule(t, err, "mg-props-correct")
}

func TestNewModelGroupRejectsUnknownCompositor(t *testing.T) {
	_, err := xsd.NewModelGroup(xsderr.Loc{}, xsd.Compositor(99), nil, nil)
	if err == nil {
		t.Fatal("NewModelGroup accepted an out-of-range Compositor, want mg-props-correct error")
	}
	assertRule(t, err, "mg-props-correct")
}

func TestNewModelGroupPreservesParticleOrder(t *testing.T) {
	ps := []xsd.Particle{
		elementRefParticle(t, "a"),
		elementRefParticle(t, "b"),
		elementRefParticle(t, "c"),
	}
	g := mustModelGroup(t, xsd.CompositorSequence, ps, nil)
	got := g.Particles()
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("Particles() len = %d, want %d", len(got), len(want))
	}
	for i, name := range want {
		if ref := got[i].Term().(xsd.ElementDeclarationRef); ref.Name.Local != name {
			t.Errorf("Particles()[%d] term name = %q, want %q", i, ref.Name.Local, name)
		}
	}
}

func TestModelGroupParticlesAccessorDoesNotAlias(t *testing.T) {
	ps := []xsd.Particle{elementRefParticle(t, "a")}
	g := mustModelGroup(t, xsd.CompositorChoice, ps, nil)

	// The accessor returns a copy.
	first := g.Particles()
	first[0] = elementRefParticle(t, "tampered")
	if got := g.Particles()[0].Term().(xsd.ElementDeclarationRef).Name.Local; got != "a" {
		t.Errorf("Particles() returned an aliased slice: got %q", got)
	}
	// The constructor does not alias the caller's backing array.
	ps[0] = elementRefParticle(t, "tampered")
	if got := g.Particles()[0].Term().(xsd.ElementDeclarationRef).Name.Local; got != "a" {
		t.Errorf("ModelGroup aliased the constructor particles slice: got %q", got)
	}
}

func TestModelGroupIsATerm(t *testing.T) {
	// Compile-time-style assertion that ModelGroup satisfies Term, checked at
	// runtime via a ResolvedTerm slot.
	g := mustModelGroup(t, xsd.CompositorAll, nil, nil)
	var tr xsd.Term = g
	if _, ok := tr.(xsd.ModelGroup); !ok {
		t.Fatal("ModelGroup does not satisfy Term")
	}
}

func TestModelGroupAnnotationsNilWhenEmpty(t *testing.T) {
	g := mustModelGroup(t, xsd.CompositorSequence, nil, nil)
	if got := g.Annotations(); got != nil {
		t.Errorf("Annotations() = %v, want nil for empty {annotations}", got)
	}
}
