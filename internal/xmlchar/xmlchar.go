package xmlchar

import "unicode/utf8"

// IsChar reports whether r is an XML 1.0 [2] Char.
func IsChar(r rune) bool {
	return r == 0x9 ||
		r == 0xA ||
		r == 0xD ||
		r >= 0x20 && r <= 0xD7FF ||
		r >= 0xE000 && r <= 0xFFFD ||
		r >= 0x10000 && r <= 0x10FFFF
}

// IndexNonChar finds the first non-Char in s, as the package doc's
// "# Contract" states.
func IndexNonChar(s string) int {
	for i, r := range s {
		if r == utf8.RuneError {
			if _, size := utf8.DecodeRuneInString(s[i:]); size <= 1 {
				return i
			}
		}
		if !IsChar(r) {
			return i
		}
	}
	return -1
}
