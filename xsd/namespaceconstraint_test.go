package xsd_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

func TestNamespaceConstraintVarietyString(t *testing.T) {
	cases := []struct {
		v    xsd.NamespaceConstraintVariety
		want string
	}{
		{xsd.NamespaceConstraintAny, "any"},
		{xsd.NamespaceConstraintEnumeration, "enumeration"},
		{xsd.NamespaceConstraintNot, "not"},
		{0, "NamespaceConstraintVariety(0)"},
		{99, "NamespaceConstraintVariety(99)"},
	}
	for _, c := range cases {
		if got := c.v.String(); got != c.want {
			t.Errorf("NamespaceConstraintVariety(%d).String() = %q, want %q", uint8(c.v), got, c.want)
		}
	}
}

func TestNamespaceNameNormalizesEmptyToAbsent(t *testing.T) {
	if got := xsd.NamespaceName(""); got != (xsd.Namespace{}) {
		t.Errorf("NamespaceName(%q) = %+v, want the zero (absent) Namespace", "", got)
	}
	if !xsd.NamespaceName("").IsAbsent() {
		t.Errorf("NamespaceName(%q).IsAbsent() = false, want true", "")
	}
	if _, ok := xsd.NamespaceName("").URI(); ok {
		t.Errorf("NamespaceName(%q).URI() ok = true, want false (absent)", "")
	}
}

func TestNamespacePresent(t *testing.T) {
	n := xsd.NamespaceName("http://example.com/t")
	if n.IsAbsent() {
		t.Fatalf("present namespace reported IsAbsent")
	}
	uri, ok := n.URI()
	if !ok || uri != "http://example.com/t" {
		t.Fatalf("URI() = (%q, %v), want (%q, true)", uri, ok, "http://example.com/t")
	}
	// A zero Namespace is absent and distinct from a present name.
	if (xsd.Namespace{}) == n {
		t.Fatalf("present namespace equals the absent zero value")
	}
}

// mustConstraint fails the test if construction errors; construction-rejection
// cases use NewNamespaceConstraint directly.
func mustConstraint(t *testing.T, v xsd.NamespaceConstraintVariety, ns []xsd.Namespace, dn []xsd.QName) xsd.NamespaceConstraint {
	t.Helper()
	c, err := xsd.NewNamespaceConstraint(xsderr.Loc{}, v, ns, dn)
	if err != nil {
		t.Fatalf("NewNamespaceConstraint(%s, %+v, %+v) unexpected error: %v", v, ns, dn, err)
	}
	return c
}

func TestAllowsNamespace(t *testing.T) {
	absent := xsd.Namespace{}
	target := xsd.NamespaceName("http://example.com/t")
	other := xsd.NamespaceName("http://other.example/u")

	cases := []struct {
		name string
		c    xsd.NamespaceConstraint
		v    xsd.Namespace
		want bool
	}{
		{"any-admits-present", mustConstraint(t, xsd.NamespaceConstraintAny, nil, nil), target, true},
		{"any-admits-absent", mustConstraint(t, xsd.NamespaceConstraintAny, nil, nil), absent, true},

		{"enum-admits-member", mustConstraint(t, xsd.NamespaceConstraintEnumeration, []xsd.Namespace{target}, nil), target, true},
		{"enum-rejects-nonmember", mustConstraint(t, xsd.NamespaceConstraintEnumeration, []xsd.Namespace{target}, nil), other, false},
		{"enum-admits-absent-member", mustConstraint(t, xsd.NamespaceConstraintEnumeration, []xsd.Namespace{absent}, nil), absent, true},
		{"enum-rejects-absent-nonmember", mustConstraint(t, xsd.NamespaceConstraintEnumeration, []xsd.Namespace{target}, nil), absent, false},

		// ##other with target namespace: not { absent, target }.
		{"not-rejects-target", mustConstraint(t, xsd.NamespaceConstraintNot, []xsd.Namespace{absent, target}, nil), target, false},
		{"not-rejects-absent", mustConstraint(t, xsd.NamespaceConstraintNot, []xsd.Namespace{absent, target}, nil), absent, false},
		{"not-admits-third", mustConstraint(t, xsd.NamespaceConstraintNot, []xsd.Namespace{absent, target}, nil), other, true},
		// notNamespace="##local": not { absent } — absent rejected, every real ns admitted.
		{"not-local-rejects-absent", mustConstraint(t, xsd.NamespaceConstraintNot, []xsd.Namespace{absent}, nil), absent, false},
		{"not-local-admits-target", mustConstraint(t, xsd.NamespaceConstraintNot, []xsd.Namespace{absent}, nil), target, true},
	}
	for _, c := range cases {
		if got := c.c.AllowsNamespace(c.v); got != c.want {
			t.Errorf("%s: AllowsNamespace(%+v) = %v, want %v", c.name, c.v, got, c.want)
		}
	}
}

func TestAllowsNameDisallowedSubtraction(t *testing.T) {
	target := xsd.NamespaceName("http://example.com/t")
	// enumeration over the target namespace, with one literal name disallowed.
	disallowed := xsd.QName{Space: "http://example.com/t", Local: "secret"}
	permitted := xsd.QName{Space: "http://example.com/t", Local: "public"}
	c := mustConstraint(t, xsd.NamespaceConstraintEnumeration, []xsd.Namespace{target}, []xsd.QName{disallowed})

	if c.AllowsName(disallowed) {
		t.Errorf("AllowsName(%s) = true, want false (literal disallowed-name member)", disallowed)
	}
	if !c.AllowsName(permitted) {
		t.Errorf("AllowsName(%s) = false, want true (namespace allowed, not disallowed)", permitted)
	}
	// A name whose namespace is not admitted fails on clause 1 regardless of
	// disallowed-name membership.
	outside := xsd.QName{Space: "http://other.example/u", Local: "x"}
	if c.AllowsName(outside) {
		t.Errorf("AllowsName(%s) = true, want false (namespace not allowed)", outside)
	}
}

func TestAllowsNameAbsentBridge(t *testing.T) {
	// enumeration over ·absent· admits a present no-namespace QName (Space=="")
	// because AllowsName bridges Space=="" to the absent namespace name.
	c := mustConstraint(t, xsd.NamespaceConstraintEnumeration, []xsd.Namespace{{}}, nil)
	unqualified := xsd.QName{Local: "x"}
	if !c.AllowsName(unqualified) {
		t.Errorf("AllowsName(%s) = false, want true (Space=='' bridges to absent, an enum member)", unqualified)
	}
	qualified := xsd.QName{Space: "http://example.com/t", Local: "x"}
	if c.AllowsName(qualified) {
		t.Errorf("AllowsName(%s) = true, want false (namespaced name not an enum member)", qualified)
	}
}

func TestNewNamespaceConstraintRejects(t *testing.T) {
	cases := []struct {
		name    string
		variety xsd.NamespaceConstraintVariety
		ns      []xsd.Namespace
		dn      []xsd.QName
	}{
		{"invalid-variety-zero", 0, nil, nil},
		{"invalid-variety-oob", 99, nil, nil},
		{"not-empty-namespaces", xsd.NamespaceConstraintNot, nil, nil},
		{"any-nonempty-namespaces", xsd.NamespaceConstraintAny, []xsd.Namespace{xsd.NamespaceName("http://example.com/t")}, nil},
		{
			// clause 4: disallowed QName whose namespace is not admitted by an
			// enumeration over a different namespace.
			"disallowed-namespace-not-allowed",
			xsd.NamespaceConstraintEnumeration,
			[]xsd.Namespace{xsd.NamespaceName("http://example.com/t")},
			[]xsd.QName{{Space: "http://other.example/u", Local: "x"}},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := xsd.NewNamespaceConstraint(xsderr.Loc{}, c.variety, c.ns, c.dn)
			if err == nil {
				t.Fatalf("NewNamespaceConstraint accepted an illegal record, want w-props-correct error")
			}
			var e *xsderr.Error
			if !errors.As(err, &e) {
				t.Fatalf("error %v is not an *xsderr.Error", err)
			}
			if got, ok := xsderr.RuleOf(err); !ok || got != xsderr.Rule("w-props-correct") {
				t.Errorf("RuleOf = (%q, %v), want (%q, true)", got, ok, "w-props-correct")
			}
		})
	}
}

func TestNewNamespaceConstraintAccepts(t *testing.T) {
	target := xsd.NamespaceName("http://example.com/t")
	// clause 4 satisfied: the disallowed QName's namespace is a member of the
	// enumeration.
	c := mustConstraint(t, xsd.NamespaceConstraintEnumeration,
		[]xsd.Namespace{target},
		[]xsd.QName{{Space: "http://example.com/t", Local: "secret"}})
	if c.Variety() != xsd.NamespaceConstraintEnumeration {
		t.Errorf("Variety() = %s, want enumeration", c.Variety())
	}
}

func TestNamespaceConstraintDedupAndOrder(t *testing.T) {
	a := xsd.NamespaceName("http://a.example")
	b := xsd.NamespaceName("http://b.example")
	// Duplicates and order: {a, b, a} dedups to [a, b] in first-occurrence order.
	c := mustConstraint(t, xsd.NamespaceConstraintEnumeration, []xsd.Namespace{a, b, a}, nil)
	got := c.Namespaces()
	want := []xsd.Namespace{a, b}
	if len(got) != len(want) {
		t.Fatalf("Namespaces() = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Namespaces()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestNamespacesAccessorIsDefensiveCopy(t *testing.T) {
	target := xsd.NamespaceName("http://example.com/t")
	c := mustConstraint(t, xsd.NamespaceConstraintEnumeration, []xsd.Namespace{target}, nil)
	got := c.Namespaces()
	got[0] = xsd.NamespaceName("http://mutated.example")
	if again := c.Namespaces(); again[0] != target {
		t.Errorf("mutating the accessor result leaked into the constraint: %+v", again[0])
	}
}

func TestNewNamespaceConstraintCopiesInput(t *testing.T) {
	ns := []xsd.Namespace{xsd.NamespaceName("http://a.example")}
	c := mustConstraint(t, xsd.NamespaceConstraintEnumeration, ns, nil)
	ns[0] = xsd.NamespaceName("http://mutated.example")
	if got := c.Namespaces(); got[0] != xsd.NamespaceName("http://a.example") {
		t.Errorf("constructor aliased caller's slice: %+v", got[0])
	}
}

func ExampleNamespaceConstraint_AllowsNamespace() {
	// ##other in a schema whose targetNamespace is "http://example.com/t" maps
	// (§3.10.2.2) to not { absent, target }.
	target := xsd.NamespaceName("http://example.com/t")
	c, err := xsd.NewNamespaceConstraint(xsderr.Loc{}, xsd.NamespaceConstraintNot,
		[]xsd.Namespace{{}, target}, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(c.AllowsNamespace(target))                                      // target rejected
	fmt.Println(c.AllowsNamespace(xsd.Namespace{}))                             // absent rejected
	fmt.Println(c.AllowsNamespace(xsd.NamespaceName("http://other.example/u"))) // third admitted
	// Output:
	// false
	// false
	// true
}
