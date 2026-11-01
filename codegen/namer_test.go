package codegen

import (
	"slices"
	"testing"
)

// TestResolve pins resolve's rules ("# Naming"), one row each: each row's
// claims, in order, are one package's.
func TestResolve(t *testing.T) {
	d, g := declaredStep, generatedSteps
	fixed := func(words ...string) typeClaim { return fixedClaim(words) }
	path := func(steps ...nameStep) typeClaim { return pathClaim(steps) }
	extend := func(of int, words ...string) typeClaim { return extendClaim{of: of, words: words} }
	tests := []struct {
		name   string
		claims []typeClaim
		want   []string
	}{
		{
			name:   "unique suffix: an anonymous type drops the steps its suffix makes needless",
			claims: []typeClaim{fixed("purchaseOrder"), path(d("purchaseOrder"), d("shipTo")), path(d("purchaseOrder"), d("shipTo"), d("address"))},
			want:   []string{"PurchaseOrder", "ShipTo", "Address"},
		},
		{
			name: "shared suffix: three paths end in address, so each keeps the step before it",
			claims: []typeClaim{fixed("order"), path(d("order"), d("address")), path(d("order"), d("shipTo")),
				path(d("order"), d("shipTo"), d("address")), path(d("order"), d("billTo")), path(d("order"), d("billTo"), d("address"))},
			want: []string{"Order", "OrderAddress", "ShipTo", "ShipToAddress", "BillTo", "BillToAddress"},
		},
		{
			name:   "anchor: a generated step starts no suffix",
			claims: []typeClaim{fixed("row"), path(d("row"), g("sequence")[0]), path(d("ct"), g("alternative2Type")[0], d("note"))},
			want:   []string{"Row", "RowSequence", "Note"},
		},
		{
			name:   "fixed rendering: a suffix rendering a fixed claim's name is not unique",
			claims: []typeClaim{fixed("ab_c"), fixed("c"), path(d("x"), d("ab"), d("c"))},
			want:   []string{"AbC", "C", "XAbC"},
		},
		{
			name:   "no unique suffix: identical paths keep their whole path and take the numeric suffix",
			claims: []typeClaim{path(d("ct"), d("x")), path(d("ct"), d("x"))},
			want:   []string{"CtX", "CtX2"},
		},
		{
			name:   "extension: it extends the final name, and its full path blocks a suffix",
			claims: []typeClaim{fixed("ct"), path(d("ct"), g("choice")[0]), extend(1, "t"), path(d("ct"), d("t")), path(d("ct"), d("line")), extend(4, "alternative2")},
			want:   []string{"Ct", "CtChoice", "CtChoiceT", "CtT", "Line", "LineAlternative2"},
		},
		{
			name:   "residual collision: shortened candidates that render alike take the numeric suffix in claim order",
			claims: []typeClaim{path(d("a"), d("xY")), path(d("b"), d("x"), g("y")[0])},
			want:   []string{"XY", "XY2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolve(tt.claims); !slices.Equal(got, tt.want) {
				t.Errorf("resolve = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestReplayNamer pins that the second walk's namer errs on a claim that
// differs from the first walk's record, in kind, words, steps or extended
// name, or that the record does not hold, that it keeps the first such error
// and names no later claim, and that the first walk's errs on extending a
// name it never returned. Each "kind" row but the first replays a claim
// whose words or steps equal the zero value of the recorded claim's arm, so
// only the arm tells them apart.
func TestReplayNamer(t *testing.T) {
	t.Run("unclaimed extension", func(t *testing.T) {
		s := newSurveyNamer()
		s.fixed("order")
		s.extend("Line", "a")
		if s.err == nil {
			t.Error("no error")
		}
	})
	record := func() *replayNamer {
		s := newSurveyNamer()
		s.fixed("order")
		s.path([]nameStep{declaredStep("order"), declaredStep("line")})
		s.extend("OrderLine", "a")
		return &replayNamer{claims: s.claims, names: resolve(s.claims)}
	}
	t.Run("same claims", func(t *testing.T) {
		r := record()
		got := []string{r.fixed("order"), r.path([]nameStep{declaredStep("order"), declaredStep("line")}), r.extend("Line", "a")}
		if r.err != nil || !slices.Equal(got, []string{"Order", "Line", "LineA"}) {
			t.Errorf("got %q, err %v; want [Order Line LineA], no error", got, r.err)
		}
	})
	for _, tt := range []struct {
		name   string
		replay func(r *replayNamer)
	}{
		{"kind", func(r *replayNamer) { r.path([]nameStep{declaredStep("order")}) }},
		{"kind fixed", func(r *replayNamer) { r.fixed("order"); r.fixed() }},
		{"kind path", func(r *replayNamer) { r.path(nil) }},
		{"kind extend", func(r *replayNamer) { r.extend("Order") }},
		{"steps", func(r *replayNamer) { r.fixed("other") }},
		{"declared", func(r *replayNamer) {
			r.fixed("order")
			r.path([]nameStep{declaredStep("order"), generatedSteps("line")[0]})
		}},
		{"extended", func(r *replayNamer) {
			r.fixed("order")
			r.path([]nameStep{declaredStep("order"), declaredStep("line")})
			r.extend("Order", "a")
		}},
		{"extending words", func(r *replayNamer) {
			r.fixed("order")
			r.path([]nameStep{declaredStep("order"), declaredStep("line")})
			r.extend("Line", "b")
		}},
		{"exhausted", func(r *replayNamer) {
			r.fixed("order")
			r.path([]nameStep{declaredStep("order"), declaredStep("line")})
			r.extend("Line", "a")
			r.fixed("order")
		}},
		{"extra", func(r *replayNamer) { r.fixed("order"); r.fixed("order"); r.fixed("order"); r.fixed("order") }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := record()
			tt.replay(r)
			if r.err == nil {
				t.Error("no error")
			}
		})
	}
	t.Run("first error kept", func(t *testing.T) {
		r := record()
		r.fixed("other")
		if got := r.path([]nameStep{declaredStep("order"), declaredStep("line")}); got != "" {
			t.Errorf("claim after an error = %q, want \"\"", got)
		}
		if r.err == nil || r.err.Error() != "type claim 0 differs from the first walk's" {
			t.Errorf("err = %v, want type claim 0's", r.err)
		}
	})
}
