package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

func TestNewNamespaceBinding(t *testing.T) {
	b := xsd.NewNamespaceBinding("p", "urn:ns")
	if b.Prefix() != "p" {
		t.Errorf("Prefix() = %q, want %q", b.Prefix(), "p")
	}
	if b.Namespace() != "urn:ns" {
		t.Errorf("Namespace() = %q, want %q", b.Namespace(), "urn:ns")
	}
}

func TestNewXPathExpressionExpressionRoundTrip(t *testing.T) {
	x := xsd.NewXPathExpression("@a > 0", nil, nil, nil)
	if x.Expression() != "@a > 0" {
		t.Errorf("Expression() = %q, want %q", x.Expression(), "@a > 0")
	}
}

func TestNewXPathExpressionBindingsRoundTrip(t *testing.T) {
	bindings := []xsd.NamespaceBinding{
		xsd.NewNamespaceBinding("p0", "urn:ns0"),
		xsd.NewNamespaceBinding("p1", "urn:ns1"),
	}
	x := xsd.NewXPathExpression("t", bindings, nil, nil)

	got := x.NamespaceBindings()
	if len(got) != 2 || got[0].Prefix() != "p0" || got[1].Prefix() != "p1" {
		t.Fatalf("NamespaceBindings() = %+v, want document order p0, p1", got)
	}
}

func TestXPathExpressionBindingsNilWhenEmpty(t *testing.T) {
	if got := xsd.NewXPathExpression("t", nil, nil, nil).NamespaceBindings(); got != nil {
		t.Errorf("NamespaceBindings() = %v, want nil for nil input", got)
	}
	if got := xsd.NewXPathExpression("t", []xsd.NamespaceBinding{}, nil, nil).NamespaceBindings(); got != nil {
		t.Errorf("NamespaceBindings() = %v, want nil for empty-slice input", got)
	}
}

func TestXPathExpressionDoesNotAliasConstructorBindings(t *testing.T) {
	bindings := []xsd.NamespaceBinding{xsd.NewNamespaceBinding("keep", "urn:keep")}
	x := xsd.NewXPathExpression("t", bindings, nil, nil)

	// Mutate the ORIGINAL backing array.
	bindings[0] = xsd.NewNamespaceBinding("tampered", "urn:tampered")

	if got := x.NamespaceBindings()[0].Prefix(); got != "keep" {
		t.Errorf("XPathExpression aliased the constructor slice: got %q, want %q", got, "keep")
	}
}

func TestXPathExpressionBindingsAccessorDoesNotAlias(t *testing.T) {
	x := xsd.NewXPathExpression("t", []xsd.NamespaceBinding{xsd.NewNamespaceBinding("keep", "urn:keep")}, nil, nil)

	// Mutate the RETURNED slice; a second call must be unaffected.
	x.NamespaceBindings()[0] = xsd.NewNamespaceBinding("tampered", "urn:tampered")

	if got := x.NamespaceBindings()[0].Prefix(); got != "keep" {
		t.Errorf("NamespaceBindings() returned an aliased slice: got %q, want %q", got, "keep")
	}
}

func TestXPathExpressionDefaultNamespaceAndBaseURI(t *testing.T) {
	t.Run("both-present", func(t *testing.T) {
		x := xsd.NewXPathExpression("t", nil, strptr("urn:dflt"), strptr("urn:base"))
		if got, ok := x.DefaultNamespace(); !ok || got != "urn:dflt" {
			t.Errorf("DefaultNamespace() = (%q, %v), want (%q, true)", got, ok, "urn:dflt")
		}
		if got, ok := x.BaseURI(); !ok || got != "urn:base" {
			t.Errorf("BaseURI() = (%q, %v), want (%q, true)", got, ok, "urn:base")
		}
	})
	t.Run("both-absent", func(t *testing.T) {
		x := xsd.NewXPathExpression("t", nil, nil, nil)
		if got, ok := x.DefaultNamespace(); ok {
			t.Errorf("DefaultNamespace() = (%q, true), want (_, false)", got)
		}
		if got, ok := x.BaseURI(); ok {
			t.Errorf("BaseURI() = (%q, true), want (_, false)", got)
		}
	})
	t.Run("empty-string-is-present", func(t *testing.T) {
		// "" is a legal anyURI: a pointer to "" is PRESENT, distinct from a nil
		// pointer's absence.
		x := xsd.NewXPathExpression("t", nil, strptr(""), strptr(""))
		if got, ok := x.DefaultNamespace(); !ok || got != "" {
			t.Errorf("DefaultNamespace() = (%q, %v), want (\"\", true)", got, ok)
		}
		if got, ok := x.BaseURI(); !ok || got != "" {
			t.Errorf("BaseURI() = (%q, %v), want (\"\", true)", got, ok)
		}
	})
}
