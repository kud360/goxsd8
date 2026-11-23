// Package xmlchar holds the XML 1.0 [2] Char production (docs/specs/md/xml.md
// §2.2) once for every library package that reads it: parser/xmltree's
// character-reference check in entity replacement text and its check of the
// source of comments, processing instructions and directives, and
// internal/xmltok's check of character data, CDATA sections and attribute
// values, codec's check of each string value and each xs:QName or xs:NOTATION
// namespace name Append writes, builtin/strict's check of each xs:string and
// xs:anyURI lexical its Parse maps, and validate/jsonsrc's check of each
// string token its walk reads.
//
// # Contract
//
//	func IsChar(r rune) bool
//	func IndexNonChar(s string) int
//
// IsChar reports whether r is a [2] Char: #x9, #xA, #xD, or a code point in
// [#x20-#xD7FF], [#xE000-#xFFFD] or [#x10000-#x10FFFF]. A negative rune or one
// above #x10FFFF is not a Char.
//
// IndexNonChar decodes s rune by rune and returns the byte offset where the
// first fault begins, or -1 when s is UTF-8 holding only Chars. A fault is a
// byte that is not valid UTF-8 or a rune that is not a Char. A literal U+FFFD
// is a Char and is not reported. A caller telling the two faults apart decodes
// at the offset: utf8.RuneError of size 1 is a bad byte.
//
// It depends on nothing but the standard library.
package xmlchar
