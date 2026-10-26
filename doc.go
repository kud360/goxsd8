// Package goxsd8 is an XSD 1.1 processor for Go: a schema parser, an
// instance validator, an XPath 2.0 engine, and a code generator emitting Go
// type declarations.
//
// The module is organized as a strict dependency DAG (see
// docs/ARCHITECTURE.md). Start here:
//
//   - parser — compile schema documents into the component model
//   - xsd — the immutable component model, with query and walk APIs
//   - validate — assess instance documents (XML and JSON adapters; BER,
//     M11, is not yet implemented)
//   - value, builtin — the value-space contracts and two backends
//     (spec-exact strict, shipped; Go-friendly native, M12, not yet
//     implemented); bring your own backend via value.Backend and prove it
//     with value/backendtest
//   - codegen, codec — schema-directed Go code generation and dataset
//     serialization (codegen's first M9 slice ships; codec, M10, is not yet
//     implemented)
//   - cmd/goxsd8 — the command-line interface
//
// This root package holds no code; it exists to document the module and
// to host repo-wide go:generate directives.
package goxsd8
