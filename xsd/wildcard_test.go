package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// mustWildcard fails the test if construction errors; construction-rejection
// cases use NewWildcard directly.
func mustWildcard(t *testing.T, nc xsd.NamespaceConstraint, pc xsd.ProcessContents) xsd.Wildcard {
	t.Helper()
	w, err := xsd.NewWildcard(xsderr.Loc{}, nc, pc)
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
				w := mustWildcard(t, v.c, p.p)
				if w.ProcessContents() != p.p {
					t.Errorf("ProcessContents() = %s, want %s", w.ProcessContents(), p.p)
				}
			})
		}
	}
}

func TestNewWildcardRejectsInvalidProcessContents(t *testing.T) {
	any := mustConstraint(t, xsd.NamespaceConstraintAny, nil, nil)
	_, err := xsd.NewWildcard(xsderr.Loc{}, any, xsd.ProcessContents(0))
	if err == nil {
		t.Fatal("NewWildcard accepted a zero ProcessContents, want w-props-correct error")
	}
	assertRule(t, err, "w-props-correct")
}

func TestNewWildcardRejectsZeroNamespaceConstraint(t *testing.T) {
	// A zero NamespaceConstraint{} was never built through
	// NewNamespaceConstraint; its {variety} is the invalid zero, which
	// NewWildcard must reject to keep an illegal Wildcard unrepresentable.
	_, err := xsd.NewWildcard(xsderr.Loc{}, xsd.NamespaceConstraint{}, xsd.ProcessStrict)
	if err == nil {
		t.Fatal("NewWildcard accepted a zero NamespaceConstraint, want w-props-correct error")
	}
	assertRule(t, err, "w-props-correct")
}

func TestWildcardProcessContentsRoundTrip(t *testing.T) {
	any := mustConstraint(t, xsd.NamespaceConstraintAny, nil, nil)
	for _, p := range []xsd.ProcessContents{xsd.ProcessSkip, xsd.ProcessStrict, xsd.ProcessLax} {
		w := mustWildcard(t, any, p)
		if got := w.ProcessContents(); got != p {
			t.Errorf("ProcessContents() = %s, want %s", got, p)
		}
	}
}

// The exported path from a Wildcard to cvc-wildcard-name (§3.10.4.2) is
// NamespaceConstraint.AllowsName through the {namespace constraint} accessor:
// the accessor hands back the constraint the wildcard was built with, so the
// two decide one name alike. cvc-wildcard entire — clause 1 and the
// defined/sibling keyword clauses together — is Schema's, not a bare
// Wildcard's (wildcardadmit.go).
func TestWildcardNamespaceConstraintDecidesTheName(t *testing.T) {
	// ##other in a schema whose targetNamespace is "http://example.com/t" maps
	// (§3.10.2.2) to not { absent, target }, mirroring the worked example in
	// NamespaceConstraint.AllowsNamespace's doc comment.
	target := xsd.NamespaceName("http://example.com/t")
	nc := mustConstraint(t, xsd.NamespaceConstraintNot, []xsd.Namespace{{}, target}, nil)
	w := mustWildcard(t, nc, xsd.ProcessStrict)

	cases := []xsd.QName{
		{Space: "http://example.com/t", Local: "x"},   // target rejected
		{Space: "http://other.example/u", Local: "x"}, // third admitted
		{Local: "x"}, // unqualified (absent) rejected
	}
	sawTrue, sawFalse := false, false
	for _, name := range cases {
		want := nc.AllowsName(name)
		if got := w.NamespaceConstraint().AllowsName(name); got != want {
			t.Errorf("w.NamespaceConstraint().AllowsName(%s) = %v, want %v (the accessor must hand back the constraint the wildcard carries)", name, got, want)
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

func TestWildcardNamespaceConstraintRespectsDisallowedNames(t *testing.T) {
	// enumeration over the target namespace, with one literal name disallowed:
	// the wildcard must reject that exact QName even though its namespace is
	// otherwise admitted (cvc-wildcard-name clause 2).
	target := xsd.NamespaceName("http://example.com/t")
	disallowed := xsd.QName{Space: "http://example.com/t", Local: "secret"}
	permitted := xsd.QName{Space: "http://example.com/t", Local: "public"}
	nc := mustConstraint(t, xsd.NamespaceConstraintEnumeration, []xsd.Namespace{target}, []xsd.QName{disallowed})
	w := mustWildcard(t, nc, xsd.ProcessLax)

	c := w.NamespaceConstraint()
	if c.AllowsName(disallowed) {
		t.Errorf("AllowsName(%s) = true, want false (literal disallowed-name member)", disallowed)
	}
	if !c.AllowsName(permitted) {
		t.Errorf("AllowsName(%s) = false, want true (namespace allowed, not disallowed)", permitted)
	}
	// The accessor must hand back the constraint the wildcard carries.
	if c.AllowsName(disallowed) != nc.AllowsName(disallowed) || c.AllowsName(permitted) != nc.AllowsName(permitted) {
		t.Errorf("w.NamespaceConstraint() disagrees with the constraint the wildcard was built with")
	}
}

// keywordWildcard builds an ##any wildcard whose {namespace constraint} carries
// the given {disallowed names} keywords.
func keywordWildcard(t *testing.T, keywords ...xsd.DisallowedNameKeyword) xsd.Wildcard {
	t.Helper()
	nc, err := xsd.NewNamespaceConstraint(xsderr.Loc{}, xsd.NamespaceConstraintAny, nil, nil, keywords)
	if err != nil {
		t.Fatalf("NewNamespaceConstraint: %v", err)
	}
	return mustWildcard(t, nc, xsd.ProcessStrict)
}

// TestWPropsCorrectClause5 pins w-props-correct (§3.10.6.1) clause 5 at the two
// tableau slots that identify a wildcard as an ATTRIBUTE wildcard, and pins that
// the check does NOT fire where sibling is legitimate.
func TestWPropsCorrectClause5(t *testing.T) {
	sibling := keywordWildcard(t, xsd.DisallowedNameSibling)
	defined := keywordWildcard(t, xsd.DisallowedNameDefined)

	newCT := func(w *xsd.Wildcard) error {
		_, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Space: "urn:t", Local: "ct"}, xsd.QName{}, nil,
			xsd.DerivationRestriction, false, nil, nil, w, xsd.EmptyContent{}, nil, nil)
		return err
	}
	newAG := func(w *xsd.Wildcard) error {
		_, err := xsd.NewAttributeGroupDefinition(xsderr.Loc{}, xsd.QName{Space: "urn:t", Local: "ag"}, nil, w)
		return err
	}

	for _, c := range []struct {
		name string
		ctor func(*xsd.Wildcard) error
	}{
		{"complex-type-attribute-wildcard", newCT},
		{"attribute-group-attribute-wildcard", newAG},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := c.ctor(&sibling)
			if err == nil {
				t.Fatal("an attribute wildcard carrying sibling was accepted, want w-props-correct clause 5 rejection")
			}
			if got, ok := xsderr.RuleOf(err); !ok || got != xsderr.Rule("w-props-correct") {
				t.Errorf("RuleOf = (%q, %v), want (%q, true)", got, ok, "w-props-correct")
			}
			// defined is legal on an attribute wildcard (cvc-wildcard clause 2.2),
			// and so is an absent {attribute wildcard}.
			if err := c.ctor(&defined); err != nil {
				t.Errorf("an attribute wildcard carrying defined was rejected: %v", err)
			}
			if err := c.ctor(nil); err != nil {
				t.Errorf("an absent {attribute wildcard} was rejected: %v", err)
			}
		})
	}

	// An open-content {wildcard} is an ELEMENT wildcard: sibling is legitimate
	// there and clause 5 must not fire (cvc-wildcard clause 3.2).
	if _, err := xsd.NewOpenContent(xsderr.Loc{}, xsd.OpenContentInterleave, sibling); err != nil {
		t.Errorf("NewOpenContent rejected an element wildcard carrying sibling: %v", err)
	}
	// An element wildcard reached as a particle {term} likewise carries it.
	occ, err := xsd.NewOccurs(xsderr.Loc{}, 1, 1)
	if err != nil {
		t.Fatalf("NewOccurs: %v", err)
	}
	if _, err := xsd.NewParticle(xsderr.Loc{}, occ, xsd.ResolvedTerm{Term: sibling}); err != nil {
		t.Errorf("NewParticle rejected an element wildcard carrying sibling: %v", err)
	}
}
