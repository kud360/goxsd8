// Package xmltree is a streaming, position-tracking XML reader: the
// origin of every xsderr.Loc in the module.
//
// It is independent of the rest of the module (leaf besides xsderr) and
// used for both schema documents (parser) and XML instances
// (validate/xmlsrc).
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
//     mark is dropped as the encoding signature it is, and an encoding
//     declaration that disagrees with the mark is that section's fatal
//     error, reported as RuleXMLWellFormed. Locations are offsets into
//     the decoded UTF-8 stream, not into the source bytes.
//     GAP(xml): UTF-16 without a mark, declared only by encoding=, is
//     not decoded — it fails well-formedness rather than being read.
//     Tracked by #361.
//   - The DOCTYPE's internal subset is read for its unparsed entity
//     declarations (<!ENTITY name SYSTEM|PUBLIC ... NDATA notation>), the
//     document's [unparsed entities] property, answered by
//     Reader.HasUnparsedEntity and final once the document element's start
//     tag is read. It surfaces as no Node: a prolog fact is not part of the
//     element/character-data stream. Internal parameter entities are
//     expanded, bounded in depth and size. The external subset is never
//     read, by design (#1668), nor is an external parameter entity; a
//     declaration after a parameter-entity reference that is not read is not
//     processed unless standalone="yes" (XML 1.0 §5.1). An entity declared
//     only where the reader did not read is not a member, and
//     Reader.AllDeclarationsProcessed, the [all declarations processed]
//     property, then reports false.
//
// Fuzz targets guard the reader against panics on malformed input
// (PRINCIPLES 24); malformed XML is an error value, never a crash.
package xmltree
