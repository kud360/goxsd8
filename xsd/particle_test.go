package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

func TestNewParticleValidWithResolvedTerm(t *testing.T) {
	w := mustWildcard(t, mustConstraint(t, xsd.NamespaceConstraintAny, nil, nil), xsd.ProcessStrict, nil)
	occ, err := xsd.NewOccurs(xsderr.Loc{}, 1, 5)
	if err != nil {
		t.Fatalf("NewOccurs: %v", err)
	}
	p, err := xsd.NewParticle(xsderr.Loc{}, occ, xsd.ResolvedTerm{Term: w}, nil)
	if err != nil {
		t.Fatalf("NewParticle unexpected error: %v", err)
	}
	if p.Occurs() != occ {
		t.Errorf("Occurs() = %v, want %v", p.Occurs(), occ)
	}
	rt, ok := p.Term().(xsd.ResolvedTerm)
	if !ok {
		t.Fatalf("Term() type = %T, want ResolvedTerm", p.Term())
	}
	if _, ok := rt.Term.(xsd.Wildcard); !ok {
		t.Errorf("ResolvedTerm.Term type = %T, want Wildcard", rt.Term)
	}
}

func TestNewParticleValidWithRefTerms(t *testing.T) {
	cases := []struct {
		name string
		term xsd.TermOrRef
	}{
		{"element-ref", xsd.ElementDeclarationRef{Name: xsd.QName{Space: "urn:ns", Local: "e"}}},
		{"group-ref", xsd.ModelGroupRef{Name: xsd.QName{Space: "urn:ns", Local: "g"}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, err := xsd.NewParticle(xsderr.Loc{}, xsd.Occurs{}, c.term, nil)
			if err != nil {
				t.Fatalf("NewParticle unexpected error: %v", err)
			}
			if p.Term() != c.term {
				t.Errorf("Term() = %v, want %v", p.Term(), c.term)
			}
		})
	}
}

func TestNewParticleRejectsAbsentTerm(t *testing.T) {
	_, err := xsd.NewParticle(xsderr.Loc{}, xsd.Occurs{}, nil, nil)
	if err == nil {
		t.Fatal("NewParticle accepted a nil {term}, want p-props-correct error")
	}
	assertRule(t, err, "p-props-correct")
}

// TestNewParticleRejectsAbsentRefName proves the representation invariant (STYLE
// T1) on the two ref {term} variants: ElementDeclarationRef and ModelGroupRef map
// only the ref-present branches (§3.3.2.4, §3.8.2), so their Name is always a
// present QName. An empty local part — with or without a namespace name — is
// rejected at construction rather than deferred, so finalize's "a zero QName is
// absent" skip (resolve.go) can never swallow an unresolvable {term}.
func TestNewParticleRejectsAbsentRefName(t *testing.T) {
	tests := []struct {
		name    string
		term    xsd.TermOrRef
		wantErr bool
	}{
		{"element-ref zero QName", xsd.ElementDeclarationRef{}, true},
		{"element-ref namespace with empty local", xsd.ElementDeclarationRef{Name: xsd.QName{Space: "urn:ns"}}, true},
		{"element-ref present local", xsd.ElementDeclarationRef{Name: xsd.QName{Local: "e"}}, false},
		{"group-ref zero QName", xsd.ModelGroupRef{}, true},
		{"group-ref namespace with empty local", xsd.ModelGroupRef{Name: xsd.QName{Space: "urn:ns"}}, true},
		{"group-ref present local", xsd.ModelGroupRef{Name: xsd.QName{Local: "g"}}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := xsd.NewParticle(xsderr.Loc{}, xsd.Occurs{}, tc.term, nil)
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("NewParticle(%v) unexpected error: %v", tc.term, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("NewParticle(%v) succeeded, want an absent-ref-name rejection", tc.term)
			}
			assertRule(t, err, "p-props-correct")
		})
	}
}

func TestNewParticleAcceptsVacuousOccurs(t *testing.T) {
	// Occurs{0,0} is a legal vacuous range at the component level (occurs.go);
	// Particle must not reject it — the min=max=0 "no component" rule is a
	// producer concern, not a Particle-constructor invariant.
	p, err := xsd.NewParticle(xsderr.Loc{}, xsd.Occurs{}, xsd.ElementDeclarationRef{Name: xsd.QName{Local: "e"}}, nil)
	if err != nil {
		t.Fatalf("NewParticle rejected Occurs{0,0}: %v", err)
	}
	if got := p.Occurs().Min(); got != 0 {
		t.Errorf("Occurs().Min() = %d, want 0", got)
	}
	if max, ok := p.Occurs().Max(); !ok || max != 0 {
		t.Errorf("Occurs().Max() = (%d, %v), want (0, true)", max, ok)
	}
}

func TestParticleAnnotationsDoNotAlias(t *testing.T) {
	anns := []xsd.Annotation{
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "first")}, nil),
	}
	p, err := xsd.NewParticle(xsderr.Loc{}, xsd.Occurs{}, xsd.ElementDeclarationRef{Name: xsd.QName{Local: "e"}}, anns)
	if err != nil {
		t.Fatalf("NewParticle: %v", err)
	}
	anns[0] = xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "tampered")}, nil)
	if docs := p.Annotations()[0].Documentation(); docs[0].Content() != "first" {
		t.Errorf("Particle aliased the constructor annotations slice: got %q", docs[0].Content())
	}
}
