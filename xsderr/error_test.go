package xsderr

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorRendering(t *testing.T) {
	err := New("cvc-elt.1", Loc{URI: "doc.xsd", Line: 12, Col: 4}, "element %q not resolvable", "foo")
	got := err.Error()
	want := `doc.xsd:12:4: [cvc-elt.1] element "foo" not resolvable`
	if got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestErrorRenderingUnknownLoc(t *testing.T) {
	err := New("cvc-elt.1", Loc{}, "boom")
	want := "?: [cvc-elt.1] boom"
	if got := err.Error(); got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestWrapReachThrough(t *testing.T) {
	sentinel := errors.New("underlying cause")
	wrapped := Wrap("src-resolve", Loc{URI: "a.xsd", Line: 1, Col: 1}, sentinel)

	if !errors.Is(wrapped, sentinel) {
		t.Fatalf("errors.Is(wrapped, sentinel) = false, want true")
	}
	// Wrap preserves the wrapped message verbatim rather than duplicating it.
	if wrapped.Msg != sentinel.Error() {
		t.Fatalf("Msg = %q, want %q", wrapped.Msg, sentinel.Error())
	}
	want := "a.xsd:1:1: [src-resolve] underlying cause"
	if got := wrapped.Error(); got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestWrapNil(t *testing.T) {
	if got := Wrap("cvc-elt.1", Loc{}, nil); got != nil {
		t.Fatalf("Wrap(nil) = %v, want nil", got)
	}
}

func TestRuleOfAndLocOf(t *testing.T) {
	loc := Loc{URI: "b.xsd", Line: 7, Col: 2}
	err := New("cos-st-restricts", loc, "bad restriction")

	// Narrowing through an intermediate fmt-wrapped error still reaches the *Error.
	chained := fmt.Errorf("context: %w", err)

	rule, ok := RuleOf(chained)
	if !ok || rule != "cos-st-restricts" {
		t.Fatalf("RuleOf = (%q, %v), want (cos-st-restricts, true)", rule, ok)
	}

	gotLoc, ok := LocOf(chained)
	if !ok || gotLoc != loc {
		t.Fatalf("LocOf = (%v, %v), want (%v, true)", gotLoc, ok, loc)
	}
}

func TestRuleOfLocOfNotAnError(t *testing.T) {
	plain := errors.New("plain error, not an *Error")

	if rule, ok := RuleOf(plain); ok {
		t.Fatalf("RuleOf(plain) = (%q, true), want (_, false)", rule)
	}
	if loc, ok := LocOf(plain); ok {
		t.Fatalf("LocOf(plain) = (%v, true), want (_, false)", loc)
	}
}

func TestIsValidRule(t *testing.T) {
	// cos-st-restricts is a real anchor confirmed present in the generated
	// catalog (and a doc.go example Rule).
	if !IsValidRule("cos-st-restricts") {
		t.Fatalf("IsValidRule(%q) = false, want true", "cos-st-restricts")
	}
	if IsValidRule("totally-made-up-rule") {
		t.Fatalf("IsValidRule(%q) = true, want false", "totally-made-up-rule")
	}
}

// TestIsValidRuleDerivationOkRestriction guards the doc.go contract example:
// derivation-ok-restriction is an irregular rule ID that the extraction must
// capture, or the package's own documentation would be false.
func TestIsValidRuleDerivationOkRestriction(t *testing.T) {
	if !IsValidRule("derivation-ok-restriction") {
		t.Fatalf("IsValidRule(%q) = false, want true", "derivation-ok-restriction")
	}
}
