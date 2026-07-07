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
//     never passed through as if they were namespaces.
//   - Every node (element, attribute, character data) answers Loc()
//     (URI, line, column) and, for character content, the byte offset —
//     decode errors downstream cite it.
//   - Nodes are immutable once produced: private fields, getter methods
//     (STYLE T1).
//
// Fuzz targets guard the reader against panics on malformed input
// (PRINCIPLES 24); malformed XML is an error value, never a crash.
package xmltree
