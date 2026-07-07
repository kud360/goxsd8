// Package codec is the dataset serializer/deserializer: schema-directed
// decode of instance data into Go values and canonical encode back out —
// built for minimal allocation.
//
// # Two decode paths, one semantics
//
//   - Runtime path (always available): the facet pipeline +
//     value.Mapping, driven by the compiled schema. General, reflective,
//     allocation-tolerant. Hot-path APIs follow the appender convention
//     (AppendCanonical(dst []byte, v) []byte; ParseBytes([]byte)) so
//     even this path is allocation-frugal.
//   - Generated fast path: code emitted at codegen time via
//     value.Emitter — no intermediate strings, no boxed values, facet
//     checks inlined, zero-allocation scalar decode on the native
//     backend.
//
// Both paths implement the SAME pipeline stages with the SAME spec rule
// IDs. Differential tests feed identical input to both and require
// identical values and identical error rule IDs; testing.AllocsPerRun
// benchmarks pin the fast path's budget. A fast path that disagrees with
// the runtime path is wrong by definition.
//
// # Debuggability
//
// Every decode error carries the pipeline stage that rejected
// (whitespace / pattern / lexical-map / facet / assertion), the type
// QName, the offending input fragment, and the instance Loc + byte
// offset. GOXSD_DEBUG=codec traces stage transitions per value through
// the injected slog logger. Generated fast paths preserve all of it and
// map back to their emitting backend and schema construct via header
// comments.
package codec
