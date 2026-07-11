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
			vc := xsd.NewValueConstraint(tt.kind, tt.lexicalForm)
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
}
