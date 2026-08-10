package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

func TestNewValueConstraintRoundTrip(t *testing.T) {
	tests := []struct {
		name        string
		kind        xsd.ValueConstraintKind
		lexicalForm string
	}{
		{"default", xsd.ValueDefault, "42"},
		{"fixed", xsd.ValueFixed, "urn:x"},
		{"default empty lexical form", xsd.ValueDefault, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vc := xsd.NewValueConstraint(tt.kind, tt.lexicalForm, nil, nil)
			if got := vc.Kind(); got != tt.kind {
				t.Errorf("Kind() = %v, want %v", got, tt.kind)
			}
			if got := vc.LexicalForm(); got != tt.lexicalForm {
				t.Errorf("LexicalForm() = %q, want %q", got, tt.lexicalForm)
			}
		})
	}
}

// The zero ValueConstraint is inspectable but never meaningful: its {variety}
// is the invalid zero ValueConstraintKind, so a consumer can detect it rather
// than mistaking it for a real default/fixed constraint.
func TestZeroValueConstraintNotMeaningful(t *testing.T) {
	var zero xsd.ValueConstraint

	if got := zero.Kind(); got == xsd.ValueDefault || got == xsd.ValueFixed {
		t.Errorf("zero Kind() = %v, want an invalid (non-default, non-fixed) kind", got)
	}
	if got := zero.LexicalForm(); got != "" {
		t.Errorf("zero LexicalForm() = %q, want empty string", got)
	}
	if got := zero.NamespaceBindings(); got != nil {
		t.Errorf("zero NamespaceBindings() = %v, want nil", got)
	}
	if ns, ok := zero.DefaultNamespace(); ok {
		t.Errorf("zero DefaultNamespace() = (%q, %t), want absent", ns, ok)
	}
}

// The namespace context round-trips in document order, and an absent {default
// namespace} stays distinguishable from one declared empty — the split
// §3.3.18's default-namespace rule depends on, since "" is a legal anyURI and
// cannot double as an absence sentinel.
func TestValueConstraintNamespaceContextRoundTrip(t *testing.T) {
	bindings := []xsd.NamespaceBinding{
		xsd.NewNamespaceBinding("b", "urn:b"),
		xsd.NewNamespaceBinding("a", "urn:a"),
	}
	empty := ""
	vc := xsd.NewValueConstraint(xsd.ValueFixed, "a:x", bindings, &empty)

	got := vc.NamespaceBindings()
	if len(got) != 2 || got[0] != bindings[0] || got[1] != bindings[1] {
		t.Errorf("NamespaceBindings() = %v, want %v in document order", got, bindings)
	}
	if ns, ok := vc.DefaultNamespace(); !ok || ns != "" {
		t.Errorf("DefaultNamespace() = (%q, %t), want (\"\", true) — declared empty, not absent", ns, ok)
	}
	if ns, ok := xsd.NewValueConstraint(xsd.ValueFixed, "x", nil, nil).DefaultNamespace(); ok {
		t.Errorf("DefaultNamespace() with no default in scope = (%q, %t), want absent", ns, ok)
	}
}

// Neither the caller's backing array nor the returned slice aliases the stored
// bindings: a ValueConstraint is immutable after construction, and the context
// it carries decides a QName {value}, so a mutation reaching it would silently
// change what a fixed value means.
func TestValueConstraintDoesNotAliasBindings(t *testing.T) {
	bindings := []xsd.NamespaceBinding{xsd.NewNamespaceBinding("p", "urn:a")}
	vc := xsd.NewValueConstraint(xsd.ValueFixed, "p:x", bindings, nil)

	bindings[0] = xsd.NewNamespaceBinding("p", "urn:hijacked")
	if got := vc.NamespaceBindings()[0].Namespace(); got != "urn:a" {
		t.Errorf("after mutating the caller's array, Namespace() = %q, want %q", got, "urn:a")
	}
	out := vc.NamespaceBindings()
	out[0] = xsd.NewNamespaceBinding("p", "urn:hijacked")
	if got := vc.NamespaceBindings()[0].Namespace(); got != "urn:a" {
		t.Errorf("after mutating a returned slice, Namespace() = %q, want %q", got, "urn:a")
	}
}
