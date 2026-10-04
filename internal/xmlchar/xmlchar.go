package xmlchar

// IsChar reports whether r is an XML 1.0 [2] Char.
func IsChar(r rune) bool {
	return r == 0x9 ||
		r == 0xA ||
		r == 0xD ||
		r >= 0x20 && r <= 0xD7FF ||
		r >= 0xE000 && r <= 0xFFFD ||
		r >= 0x10000 && r <= 0x10FFFF
}
