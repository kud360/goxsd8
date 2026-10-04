// Package xmltree is a streaming, position-tracking XML reader: the
// origin of every xsderr.Loc in the module.
//
// It is independent of the rest of the module (leaf besides xsderr and the
// internal/ leaves internal/xmldecl, internal/xmlenc, internal/xmlname,
// internal/xmlchar and internal/xmltok) and used for both schema documents
// (parser) and XML instances (validate/xmlsrc).
//
// # Contract (implemented in M2)
//
//   - Streaming with bounded memory: wraps the io.Reader, never
//     io.ReadAll (STYLE P4). Line/column mapping uses an offset index
//     over newline positions (sort-searched on demand), not retained
//     document content.
//   - Namespace-scoped: prefixes resolve against in-scope bindings at
//     each node; unbound prefixes are reported as errors with location,
//     never passed through as if they were namespaces. A start tag's
//     whole set of PREFIXED in-scope bindings is enumerable in a
//     deterministic order (StartElement.InScopePrefixes), for consumers
//     that must carry a namespace context forward rather than resolve one
//     name; the default namespace stays a separate LookupPrefix("") fact.
//   - Every node (element, attribute, character data) answers Loc()
//     (URI, line, column) and, for character content, the byte offset —
//     decode errors downstream cite it.
//   - Nodes are immutable once produced: private fields, getter methods
//     (STYLE T1).
//   - Byte-order marks are honoured per XML 1.0 §4.3.3: a UTF-16 mark
//     (FE FF, FF FE) selects a streaming transcode to UTF-8, a UTF-8
//     mark is dropped as the encoding signature it is (both
//     internal/xmlenc, shared with the conformance harness's raw
//     re-reads), and an encoding declaration that disagrees with the
//     mark is that section's fatal error, reported as
//     RuleXMLWellFormed. Locations are offsets into the decoded UTF-8
//     stream, not into the source bytes.
//   - GAP(xml): UTF-16 without a mark, declared only by encoding=, is
//     not decoded — it fails well-formedness rather than being read.
//     Tracked by #361.
//   - Element, attribute and PI target names and entity references, in
//     the document and in an entity's replacement text, are checked
//     against XML 1.0 5th edition's [5] Name (internal/xmltok); a name
//     outside it is RuleXMLWellFormed. So is a DOCTYPE whose document type
//     name is missing or outside [5] (XML 1.0 [28] doctypedecl, checked
//     against internal/xmlname at the directive).
//   - A document whose XML declaration specifies a 1.x version number
//     other than 1.0 is read as a 1.0 document (XML 1.0 §2.8 Note), by a
//     same-length rewrite of that number (internal/xmldecl), so locations
//     are the source's own. Only the label is admitted: a 1.1-only
//     feature, such as a C0 character reference, is read as XML 1.0 reads
//     it. The one exception is prefix undeclaration: xmlns:p="" undeclares
//     p in a document labelled version="1.1" (Namespaces in XML 1.1), and
//     is rejected as RuleXMLWellFormed in any other document, which is XML
//     1.0 (nsc-NoPrefixUndecl). The default declaration xmlns="" is legal
//     in both.
//   - After the document element only comments, processing instructions
//     and white space may appear (XML 1.0 [1] document, [27] Misc):
//     a character-data run there holding anything but white space is
//     rejected as RuleXMLWellFormed at the run's start, the character
//     after the preceding markup.
//   - The DOCTYPE's internal subset is read for its general entity
//     declarations, the first declaration of a name binding (XML 1.0 §4.2).
//     Its unparsed entities (<!ENTITY name SYSTEM|PUBLIC ... NDATA
//     notation>) are the document's [unparsed entities] property, answered
//     by Reader.HasUnparsedEntity and final once the document element's
//     start tag is read; the subset itself surfaces as no Node, a prolog
//     fact not being part of the element/character-data stream. Internal
//     parameter entities are expanded, bounded in depth and size. Rejected
//     as RuleXMLWellFormed at the directive, in the subset or in an
//     expanded parameter entity's replacement text: text between
//     declarations that is neither S nor a PEReference — a '%' run that is
//     no PEReference, or a '<!' opening no markup declaration, among it
//     ([28b] intSubset, [28a] DeclSep, [69] PEReference, WFC: PE Between
//     Declarations); a comment holding "--" ([15]); a processing
//     instruction whose target is no Name or is "xml" in any case ([16],
//     [17]); a notation declaration that is no [82] NotationDecl; a '<'
//     outside a markup declaration's literals ([45], [52], [70], [82]); a
//     parameter-entity reference inside a markup declaration, an entity
//     value literal included (WFC: PEs in Internal Subset), and any other
//     '%' in an entity value literal ([9]); an <!ENTITY> that is no [70]
//     EntityDecl — no S after its keyword, a declared name, general or
//     parameter, that is no Name ([71], [72], [5]), a missing definition,
//     one that is neither one EntityValue nor an ExternalID ([73], [74],
//     [75], [11], [12], [13]), an NDataDecl in a parameter entity's PEDef
//     ([74]) or anything but one after a general entity's ExternalID
//     ([76]), and an entity value literal whose '&' opens no Reference
//     ([66]–[68]) or whose character reference names no Char (WFC: Legal
//     Character), referenced or not; replacement text ending inside a
//     comment, processing instruction or markup declaration; and a '<' in
//     the DOCTYPE header, a subset no ']' closes, or text other than S
//     between that ']' and '>' ([28] doctypedecl). A processing instruction
//     between the subset's declarations runs to the "?>" that alone closes
//     it ([16] PI), whatever '>', '<' or quote it holds (internal/xmltok),
//     and one no "?>" closes fails as the decoder's syntax error at the end
//     of input. The external subset is never read, by design (#1668), nor
//     is an external parameter entity; after a parameter-entity reference
//     that is not read, unless standalone="yes", the rest of the internal
//     subset is checked for well-formedness alone, binding no parameter
//     entity, declaring no general entity and expanding no reference (XML
//     1.0 §5.1). An entity declared only where the reader did not read is
//     not a member, and Reader.AllDeclarationsProcessed, the [all
//     declarations processed] property, then reports false.
//   - GAP(xml): <!ELEMENT> and <!ATTLIST> bodies ([45]–[60]) are checked
//     only for a parameter-entity reference, a '<' and their closing '>'.
//     Tracked by #2225.
//   - A reference to an internal general entity is replaced by its
//     replacement text (XML 1.0 §4.4.2, §4.4.5): in content, parsed as
//     content in the scope in force at the reference, its nodes located at
//     the reference; in an attribute value, normalized per §3.3.3 together
//     with the rest of that value. Nested references expand at inclusion
//     (§4.5). A recursive reference, a '<' in replacement text an attribute
//     value includes, and replacement text that is not balanced content are
//     RuleXMLWellFormed faults. A reference past the reader's bound on
//     nesting depth or on replacement text included per document is refused,
//     as is a reference to an entity that is not internal or whose
//     declaration the reader did not read, wrapping a cause: the reader does
//     not decide whether the document is well-formed.
//   - Every error the reader returns but io.EOF is a RuleXMLWellFormed
//     *xsderr.Error, and whether it wraps a cause says what it decides. One wrapping no
//     cause is a charge the reader makes itself and a definite fault whatever
//     the document's DTD declares — an entity it includes is one whose
//     internal-subset declaration it read, which binds before the external
//     subset's (XML 1.0 §2.8, §4.2) — except the unbound-prefix charge: an
//     <!ATTLIST can default the namespace declaration that binds the prefix
//     (§3.3.2), a non-validating processor must supply that default from the
//     internal subset (§5.1), and the reader applies no attribute default.
//     One wrapping a cause — an encoding/xml syntax error, among them its
//     refusal of an encoding declaration the reader does not decode, an I/O
//     fault, or one of the refusals above — may be a limit of this reader.
//
// Fuzz targets guard the reader against panics on malformed input
// (PRINCIPLES 24); malformed XML is an error value, never a crash.
package xmltree
