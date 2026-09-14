package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

func TestNewAssertionTestRoundTrip(t *testing.T) {
	test := xsd.NewXPathExpression("@a > 0", []xsd.NamespaceBinding{xsd.NewNamespaceBinding("p", "urn:ns")}, strptr("urn:dflt"), nil)
	a := xsd.NewAssertion(test)

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
