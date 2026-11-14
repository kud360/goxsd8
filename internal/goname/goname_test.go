package goname

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
			n := NewNamer()
			var got []string
			for _, c := range tt.claims {
				got = append(got, n.Claim(c...))
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("claims %q = %q, want %q", tt.claims, got, tt.want)
			}
		})
	}
}

// TestRank pins Rank's rows: the candidate itself, a suffix from 2, and the
// names it is no rank of.
func TestRank(t *testing.T) {
	tests := []struct {
		name string
		rank int
		ok   bool
	}{
		{"RootX", 1, true},
		{"RootX2", 2, true},
		{"RootX10", 10, true},
		{"RootX1", 0, false},
		{"RootX0", 0, false},
		{"RootX02", 0, false},
		{"RootX+2", 0, false},
		{"RootX2a", 0, false},
		{"RootXa", 0, false},
		{"RootY", 0, false},
		{"Root", 0, false},
		{"RootX99999999999999999999", 0, false},
	}
	for _, tt := range tests {
		rank, ok := Rank("RootX", tt.name)
		if rank != tt.rank || ok != tt.ok {
			t.Errorf("Rank(RootX, %s) = %d, %t, want %d, %t", tt.name, rank, ok, tt.rank, tt.ok)
		}
	}
}

// TestRankFollowsClaimOrder pins that k claims of one candidate on one Namer
// rank strictly increasing in claim order, whichever of its suffixed names
// were claimed before them.
func TestRankFollowsClaimOrder(t *testing.T) {
	for _, holes := range [][]string{nil, {"B"}, {"B2"}, {"B", "B3"}, {"B2", "B4", "B5"}, {"B22"}} {
		n := NewNamer()
		for _, h := range holes {
			n.ClaimCandidate(h)
		}
		last := 0
		for k := range 6 {
			name := n.ClaimCandidate("B")
			rank, ok := Rank("B", name)
			if !ok || rank <= last {
				t.Fatalf("holes %q: claim %d is %s, rank %d, %t, after rank %d", holes, k+1, name, rank, ok, last)
			}
			last = rank
		}
	}
}

// TestExtend pins that Extend renders of's Identifier and then each word's.
func TestExtend(t *testing.T) {
	if got := Extend("RootChoice", "a_a"); got != "RootChoiceAA" {
		t.Errorf("Extend(RootChoice, a_a) = %s, want RootChoiceAA", got)
	}
	if got := Extend("HSubstitutionGroup", "c", "d"); got != "HSubstitutionGroupCD" {
		t.Errorf("Extend(HSubstitutionGroup, c, d) = %s, want HSubstitutionGroupCD", got)
	}
}
