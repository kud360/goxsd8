// Package xsd is the XSD 1.1 component model: the immutable, in-memory
// representation of a compiled schema set.
//
// It is a pure leaf — it imports only xsderr and the stdlib. Parsing
// lives in package parser; values in package value; this package is the
// shape those operate on.
//
// # Components
//
// The model covers the component kinds of Structures §2.2: simple and
// complex type definitions, element and attribute declarations, attribute
// and model groups, particles, wildcards, identity constraints, type
// alternatives (CTA), assertions, notations, and annotations.
//
// Design rules (see docs/STYLE.md, docs/PRINCIPLES.md):
//
//   - Immutable after construction. Construction happens in phases
//     (parse → resolve → finalize) so no traversal ever needs a cycle
//     check (D4); spec-forbidden circularities are rejected at finalize
//     with their named src-/cos- rule.
//
//   - Every child collection is a slice in document order. Maps exist
//     only as internal lookup indexes and never determine order (D2).
//
//   - One fact, one encoding (D3): nothing derivable is stored. There is
//     no Primitive bool — a type that defines its own fundamental facets
//     is a primitive, answered by IsPrimitive(); effective facets are
//     computed on demand by merging the base chain, never cached.
//
//   - Closed sets (variety, derivation method, process-contents, use)
//     are typed constants with unexported tags, never strings (T1).
//
// Names are expanded QNames:
//
//	type QName struct{ Space, Local string }
//
// The zero value means absent/anonymous; String() renders Clark notation
// ("{ns}local").
//
// # Query API
//
// Direct lookups over a compiled schema set — element, attribute, and
// type definitions by QName — exposed as minimal capability views (T3):
// a consumer that needs only element lookup receives an interface with
// exactly that method, not the whole schema. Views are read-only windows
// onto the compiled set; they never copy it.
//
// # Walk API
//
// Traversal of a type's effective content model. The reusable core is an
// algebra — type-derivation validity, substitution-group acceptance,
// wildcard admission (one canonical implementation, context supplied by
// the caller), attribute-use lookup — with two drivers built on it:
//
//   - the push driver: an exhaustive, schema-only Walker that visits
//     every particle reachable through sequences, choices, all-groups,
//     and named group references, in document order (codegen's driver);
//   - the pull driver: an instance-guided Matcher that advances the
//     content model one child at a time, greedily and deterministically
//     (validation's driver).
//
// Substitution groups are not expanded at walk time — instance-time
// concern. Recursive named-group references terminate by construction in
// the finalized model; the Walker needs no visited set beyond the
// path-scoped guard the spec's one legal nesting requires.
package xsd
