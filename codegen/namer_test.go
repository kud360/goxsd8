package codegen

import (
	"slices"
	"testing"
)

// TestNamer pins the namer's rules, one row each. Each row runs its claims in
// order on one fresh namer.
func TestNamer(t *testing.T) {
	tests := []struct {
		name   string
		claims [][]string
		want   []string
	}{
		{
			name:   "path: a path's local names join in order, before resolve shortens it",
			claims: [][]string{{"purchaseOrder", "shipTo"}, {"purchaseOrder", "shipTo", "address"}},
			want:   []string{"PurchaseOrderShipTo", "PurchaseOrderShipToAddress"},
		},
		{
			name:   "keyword: an XSD name spelling a Go keyword is upper-cased out of it",
			claims: [][]string{{"type"}, {"func"}},
			want:   []string{"Type", "Func"},
		},
		{
			name:   "case fold: the later name in document order takes a suffix",
			claims: [][]string{{"a"}, {"A"}, {"a"}},
			want:   []string{"A", "A2", "A3"},
		},
		{
			name:   "suffix: a suffixed candidate already claimed moves on",
			claims: [][]string{{"x"}, {"X"}, {"x2"}},
			want:   []string{"X", "X2", "X22"},
		},
		{
			name:   "Go-illegal characters break words",
			claims: [][]string{{"item-list"}, {"a.b_c"}, {"x·y"}},
			want:   []string{"ItemList", "ABC", "XY"},
		},
		{
			name:   "no upper-case first letter: prefixed with X",
			claims: [][]string{{"_1st"}, {"中"}},
			want:   []string{"X1st", "X中"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := newNamer()
			var got []string
			for _, c := range tt.claims {
				got = append(got, n.claim(c...))
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("claims %q = %q, want %q", tt.claims, got, tt.want)
			}
		})
	}
}

// TestResolve pins resolve's rules ("# Naming"), one row each: each row's
// claims, in order, are one package's.
func TestResolve(t *testing.T) {
	d, g := declaredStep, generatedSteps
	fixed := func(words ...string) typeClaim { return typeClaim{kind: fixedClaim, steps: g(words...)} }
	path := func(steps ...nameStep) typeClaim { return typeClaim{kind: pathClaim, steps: steps} }
	extend := func(of int, words ...string) typeClaim {
		return typeClaim{kind: extendClaim, steps: g(words...), of: of}
	}
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
// differs from the first walk's record, in kind, steps or extended name, or
// that the record does not hold.
func TestReplayNamer(t *testing.T) {
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
}
