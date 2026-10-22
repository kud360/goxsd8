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
			name:   "ancestor: an anonymous type joins its named ancestor's path",
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
