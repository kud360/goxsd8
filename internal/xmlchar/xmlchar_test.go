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

// TestIndexNonChar pins the offset of the first byte that starts no UTF-8
// encoding of a [2] Char, and -1 for strings of Chars only.
func TestIndexNonChar(t *testing.T) {
	for _, tc := range []struct {
		name string
		s    string
		want int
	}{
		{"empty", "", -1},
		{"TAB LF CR", "a\t\n\rb", -1},
		{"supplementary plane", "x\U0001D11Ey", -1},
		{"literal U+FFFD", "a�b", -1},
		{"U+0000", "a\x00b", 1},
		{"invalid UTF-8", "a\xffb", 1},
		{"U+FFFE after a two-byte character", "é￾", 2},
		{"surrogate bytes", "ab\xed\xa0\x80", 2},
	} {
		if got := IndexNonChar(tc.s); got != tc.want {
			t.Errorf("%s: IndexNonChar(%q) = %d, want %d", tc.name, tc.s, got, tc.want)
		}
	}
}
