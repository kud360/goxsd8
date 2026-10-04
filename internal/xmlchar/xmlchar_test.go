package xmlchar

import "testing"

// TestIsChar probes XML 1.0 [2] Char on each side of every alternative's
// bounds.
func TestIsChar(t *testing.T) {
	for _, tc := range []struct {
		r    rune
		want bool
	}{
		{-1, false},
		{0x0, false},
		{0x8, false},
		{0x9, true},
		{0xA, true},
		{0xB, false},
		{0xC, false},
		{0xD, true},
		{0xE, false},
		{0x1F, false},
		{0x20, true},
		{0xD7FF, true},
		{0xD800, false},
		{0xDFFF, false},
		{0xE000, true},
		{0xFFFD, true},
		{0xFFFE, false},
		{0xFFFF, false},
		{0x10000, true},
		{0x10FFFF, true},
		{0x110000, false},
	} {
		if got := IsChar(tc.r); got != tc.want {
			t.Errorf("IsChar(%#x) = %v, want %v", tc.r, got, tc.want)
		}
	}
}
