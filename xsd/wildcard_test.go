package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// mustWildcard fails the test if construction errors; construction-rejection
// cases use NewWildcard directly.
func mustWildcard(t *testing.T, nc xsd.NamespaceConstraint, pc xsd.ProcessContents, anns []xsd.Annotation) xsd.Wildcard {
	t.Helper()
	w, err := xsd.NewWildcard(xsderr.Loc{}, nc, pc, anns)
	if err != nil {
		t.Fatalf("NewWildcard(%s, %s) unexpected error: %v", nc.Variety(), pc, err)
	}
	return w
}

func TestNewWildcardValid(t *testing.T) {
	target := xsd.NamespaceName("http://example.com/t")
	varieties := []struct {
		name string
		c    xsd.NamespaceConstraint
	}{
		{"any", mustConstraint(t, xsd.NamespaceConstraintAny, nil, nil)},
		{"enumeration", mustConstraint(t, xsd.NamespaceConstraintEnumeration, []xsd.Namespace{target}, nil)},
		{"not", mustConstraint(t, xsd.NamespaceConstraintNot, []xsd.Namespace{target}, nil)},
	}
	processes := []struct {
		name string
		p    xsd.ProcessContents
	}{
		{"skip", xsd.ProcessSkip},
		{"strict", xsd.ProcessStrict},
		{"lax", xsd.ProcessLax},
	}
	for _, v := range varieties {
		for _, p := range processes {
			t.Run(v.name+"/"+p.name, func(t *testing.T) {
				w := mustWildcard(t, v.c, p.p, nil)
				if w.ProcessContents() != p.p {
					t.Errorf("ProcessContents() = %s, want %s", w.ProcessContents(), p.p)
				}
			})
		}
	}
}

func TestNewWildcardRejectsInvalidProcessContents(t *testing.T) {
	any := mustConstraint(t, xsd.NamespaceConstraintAny, nil, nil)
	_, err := xsd.NewWildcard(xsderr.Loc{}, any, xsd.ProcessContents(0), nil)
	if err == nil {
		t.Fatal("NewWildcard accepted a zero ProcessContents, want w-props-correct error")
	}
	assertRule(t, err, "w-props-correct")
}

func TestNewWildcardRejectsZeroNamespaceConstraint(t *testing.T) {
	// A zero NamespaceConstraint{} was never built through
	// NewNamespaceConstraint; its {variety} is the invalid zero, which
	// NewWildcard must reject to keep an illegal Wildcard unrepresentable.
	_, err := xsd.NewWildcard(xsderr.Loc{}, xsd.NamespaceConstraint{}, xsd.ProcessStrict, nil)
	if err == nil {
		t.Fatal("NewWildcard accepted a zero NamespaceConstraint, want w-props-correct error")
	}
	assertRule(t, err, "w-props-correct")
}

func TestWildcardProcessContentsRoundTrip(t *testing.T) {
	any := mustConstraint(t, xsd.NamespaceConstraintAny, nil, nil)
	for _, p := range []xsd.ProcessContents{xsd.ProcessSkip, xsd.ProcessStrict, xsd.ProcessLax} {
		w := mustWildcard(t, any, p, nil)
		if got := w.ProcessContents(); got != p {
			t.Errorf("ProcessContents() = %s, want %s", got, p)
		}
	}
}

func TestWildcardAnnotationsRoundTrip(t *testing.T) {
	any := mustConstraint(t, xsd.NamespaceConstraintAny, nil, nil)
	anns := []xsd.Annotation{
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "first")}, nil),
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "second")}, nil),
	}
	w := mustWildcard(t, any, xsd.ProcessStrict, anns)

	got := w.Annotations()
	if len(got) != 2 {
		t.Fatalf("Annotations() len = %d, want 2", len(got))
	}
	if docs := got[0].Documentation(); len(docs) != 1 || docs[0].Content() != "first" {
		t.Errorf("Annotations()[0] documentation = %+v, want content %q", docs, "first")
	}

	// Defensive copy: mutating the caller's input after construction must not
	// be observable through the Wildcard.
	anns[0] = xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "tampered")}, nil)
	if docs := w.Annotations()[0].Documentation(); docs[0].Content() != "first" {
		t.Errorf("Wildcard aliased the constructor slice: got content %q, want %q", docs[0].Content(), "first")
	}
}

func TestWildcardAnnotationsNilWhenEmpty(t *testing.T) {
	any := mustConstraint(t, xsd.NamespaceConstraintAny, nil, nil)
	w := mustWildcard(t, any, xsd.ProcessStrict, nil)
	if got := w.Annotations(); got != nil {
		t.Errorf("Annotations() = %v, want nil for empty {annotations}", got)
	}
}

func TestWildcardAllowsNameDelegates(t *testing.T) {
	// ##other in a schema whose targetNamespace is "http://example.com/t" maps
	// (§3.10.2.2) to not { absent, target }, mirroring the worked example in
	// NamespaceConstraint.AllowsNamespace's doc comment.
	target := xsd.NamespaceName("http://example.com/t")
	nc := mustConstraint(t, xsd.NamespaceConstraintNot, []xsd.Namespace{{}, target}, nil)
	w := mustWildcard(t, nc, xsd.ProcessStrict, nil)

	cases := []xsd.QName{
		{Space: "http://example.com/t", Local: "x"},   // target rejected
		{Space: "http://other.example/u", Local: "x"}, // third admitted
		{Local: "x"}, // unqualified (absent) rejected
	}
	sawTrue, sawFalse := false, false
	for _, name := range cases {
		want := nc.AllowsName(name)
		if got := w.AllowsName(name); got != want {
			t.Errorf("Wildcard.AllowsName(%s) = %v, want %v (must agree with NamespaceConstraint.AllowsName)", name, got, want)
		}
		if want {
			sawTrue = true
		}
		if !want {
			sawFalse = true
		}
	}
	if !sawTrue || !sawFalse {
		t.Fatalf("test did not exercise both admit and reject cases (sawTrue=%v sawFalse=%v)", sawTrue, sawFalse)
	}
}

func TestWildcardAllowsNameRespectsDisallowedNames(t *testing.T) {
	// enumeration over the target namespace, with one literal name disallowed:
	// the wildcard must reject that exact QName even though its namespace is
	// otherwise admitted (cvc-wildcard-name clause 2).
	target := xsd.NamespaceName("http://example.com/t")
	disallowed := xsd.QName{Space: "http://example.com/t", Local: "secret"}
	permitted := xsd.QName{Space: "http://example.com/t", Local: "public"}
	nc := mustConstraint(t, xsd.NamespaceConstraintEnumeration, []xsd.Namespace{target}, []xsd.QName{disallowed})
	w := mustWildcard(t, nc, xsd.ProcessLax, nil)

	if w.AllowsName(disallowed) {
		t.Errorf("AllowsName(%s) = true, want false (literal disallowed-name member)", disallowed)
	}
	if !w.AllowsName(permitted) {
		t.Errorf("AllowsName(%s) = false, want true (namespace allowed, not disallowed)", permitted)
	}
	// Delegation must match the underlying constraint on both.
	if w.AllowsName(disallowed) != nc.AllowsName(disallowed) || w.AllowsName(permitted) != nc.AllowsName(permitted) {
		t.Errorf("Wildcard.AllowsName disagrees with NamespaceConstraint.AllowsName")
	}
}
