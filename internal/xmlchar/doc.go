// Package xmlchar holds the XML 1.0 [2] Char production (docs/specs/md/xml.md
// §2.2) once for every library package that reads it: parser/xmltree's
// character-reference check in entity replacement text and its check of the
// source of comments, processing instructions and directives, and
// internal/xmltok's check of character data, CDATA sections and attribute
// values.
//
// # Contract
//
//	func IsChar(r rune) bool
//
// IsChar reports whether r is a [2] Char: #x9, #xA, #xD, or a code point in
// [#x20-#xD7FF], [#xE000-#xFFFD] or [#x10000-#x10FFFF]. A negative rune or one
// above #x10FFFF is not a Char.
//
// It depends on nothing but the standard library.
package xmlchar
