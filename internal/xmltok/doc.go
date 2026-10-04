// Package xmltok is encoding/xml's strict tokenizer with XML 1.0 Fifth
// Edition names: element, attribute, processing-instruction target and
// entity-reference names are checked against [5] Name over [4] NameStartChar
// and [4a] NameChar (docs/specs/md/xml.md §2.3) as internal/xmlname holds
// them, at the name sites [40] STag, [41] Attribute, [42] ETag, [44]
// EmptyElemTag, [17] PITarget and [68] EntityRef. encoding/xml checks the same
// sites against XML 1.0 Fourth Edition's Appendix B tables, which reject
// 5e names such as one carrying U+0133, and no wrapper can lift them: its
// RawToken fails the same way.
//
// # Contract
//
//	func NewDecoder(r io.Reader) *Decoder
//	func (d *Decoder) Token() (xml.Token, error)
//	func (d *Decoder) RawToken() (xml.Token, error)
//	func (d *Decoder) InputOffset() int64
//	func (d *Decoder) Skip() error
//	Decoder.Entity        map[string]string
//	Decoder.CharsetReader func(charset string, input io.Reader) (io.Reader, error)
//
// Each method behaves as encoding/xml's Decoder method of the same name with
// Strict left true: the tokens are encoding/xml's own types (xml.StartElement,
// xml.CharData, …), a malformed document fails with an *xml.SyntaxError
// carrying encoding/xml's message and line, and InputOffset reads the same
// byte offset after every token. A name outside 5e [5] Name is the
// syntax error "invalid XML name: <name>" at the offset after the name, the
// [xml-wf] fault encoding/xml raises for a name outside its tables. The
// non-strict mode (Strict, AutoClose, HTMLEntity), DefaultSpace, InputPos,
// NewTokenDecoder and the Unmarshal machinery are not carried.
//
// Two departures besides the name tables. An end of input is recognised with
// errors.Is, so an input or CharsetReader error that wraps io.EOF reads as end
// of input where encoding/xml compares it by identity. And a processing
// instruction at a directive's top level, outside every markup declaration in
// it, is read through the "?>" that alone closes it ([16] PI) and kept whole in
// the xml.Directive: a '>', '<' or quote inside it neither ends the directive
// nor nests nor opens a literal, where encoding/xml ends a DOCTYPE at such a
// '>' and reads its internal subset on as top-level tokens, or runs on to the
// end of input from such a quote.
//
// Its consumers are parser/xmltree's Reader and entity replacement-text
// reader, and the conformance harness's raw re-reads (subtreeroot.go's
// rawDecoder, and the instance hint reader through recordingDecoder).
//
// It is a fork of Go 1.26's encoding/xml/xml.go and read.go's Skip, under
// the BSD license in this directory's LICENSE file. It depends on the standard
// library, internal/xmlname and internal/xmlchar (XML 1.0 [2] Char) only.
package xmltok
