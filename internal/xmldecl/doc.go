// Package xmldecl presents a document whose XML declaration labels it 1.x as
// the XML 1.0 document an XML 1.0 processor processes it as (XML 1.0 5e §2.8
// Note: "When an XML 1.0 processor encounters a document that specifies a 1.x
// version number other than '1.0', it will process it as a 1.0 document").
//
// # Contract
//
//	func As10(r io.Reader) *Reader
//	func (x *Reader) Version() string
//
// internal/xmltok, as encoding/xml, rejects every version number but "1.0",
// so each reader that hands a document to it wraps the stream in As10 first,
// once it has dropped a leading byte-order mark: parser/xmltree's Reader, and
// the conformance harness's raw re-reads of the same documents (its
// rawDecoder), which must admit exactly the label xmltree admits. The rewrite
// is byte-for-byte the same length, so every offset, line and column after the
// declaration is the source's own.
//
// Only the label is admitted. A 1.1 feature the document then uses — a C0
// character reference, NEL or U+2028 as a line end — is a non-1.0 feature, and
// §2.8 accepts a 1.x document only "provided they do not use any non-1.0
// features"; it stays whatever the 1.0 reader makes of it. Version reports the
// label the source carried, for a reader that decides a version-dependent
// constraint itself: parser/xmltree admits a prefix undeclaration
// (xmlns:p="") only in a document labelled 1.1 (nsc-NoPrefixUndecl).
//
// It depends on nothing but the standard library.
package xmldecl
