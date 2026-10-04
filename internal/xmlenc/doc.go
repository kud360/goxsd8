// Package xmlenc reads a document entity's byte-order mark and hands back its
// body as UTF-8 (XML 1.0 5e §4.3.3; the marks are enumerated in Appendix F.1):
// a UTF-16 mark, of either byte order, selects a streaming transcode, and a
// UTF-8 mark is dropped as the encoding signature it is. Ill-formed UTF-16 — an
// unpaired surrogate, a trailing partial code unit — is a read error, never a
// U+FFFD substitution.
//
// # Contract
//
//	type Mark int
//	func Decode(r io.Reader) (io.Reader, Mark)
//	func (Mark) AgreesWith(name string) bool
//	func (Mark) CharsetReader(name string, input io.Reader) (io.Reader, error)
//	func (Mark) String() string
//
// Every reader that hands a document to internal/xmltok takes its mark
// detection and transcoder from here: parser/xmltree's Reader, and the
// conformance harness's raw re-reads of the same documents (its rawDecoder),
// which must read a UTF-16 document exactly when xmltree reads it. The Mark is
// the evidence an encoding declaration must agree with: AgreesWith is §4.3.3's
// name table, and CharsetReader reports a disagreement through the decoder.
//
// GAP(xml): only a mark is taken as evidence of an entity's encoding. A
// BOM-less UTF-16 entity, or one declaring any other EncName, is not decoded.
// Tracked by #361.
//
// It depends on nothing but the standard library.
package xmlenc
