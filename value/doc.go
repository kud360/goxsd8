// Package value defines the value-space contracts: how typed values are
// represented, compared, and produced from lexical forms — without fixing
// any particular Go representation.
//
// # Values are open
//
// [Value] is an alias for any: deliberately not a sealed interface
// (PRINCIPLES 2), so users bring their own backends and value types. What a
// value can do is discovered through small capability interfaces (STYLE T2),
// never a type switch over concrete types: [Eq] and [Ordered] for comparison
// (XSD value spaces are partially ordered — see [Ordering]), [Identical] for
// the identity relation enumeration matching needs, [Lengthed], [DigitCounted],
// [Scaled], [TimezoneAware] for the facets that measure a value, and
// [Canonical] for value → canonical lexical.
//
// # Backends
//
// A [Backend] supplies lexical↔value mappings for builtin types. It MUST cover
// the primitives (directly or via composition); it MAY also map derived
// builtins to give them their own, typically narrower, representation — derived
// types without their own mapping inherit the nearest mapped ancestor's (see
// package builtin). QName and NOTATION need in-scope namespace bindings at
// parse time, so a mapping's Parse takes a [Context] (PRINCIPLES 19).
// [Override] composes backends per type — partial's mappings where defined,
// base otherwise — so a program can back only xs:decimal with a money type and
// keep the rest.
//
// # The widest-space rule (facet checks under derived mappings)
//
// A derived type's own [Mapping] governs the VALUE the application receives —
// never the space in which inherited facet checks run. A derived representation
// is allowed to be narrower than its base's value space; using it for
// base-chain semantics would corrupt them (overflow where the base space has
// none, collapsed precision, different ordering). So:
//
//   - enumeration and bound facets, wherever they sit on the derivation
//     chain, are compared in the value space of the type that DECLARES the
//     facet, parsed by that type's governing mapping (its own, or its nearest
//     mapped ancestor's — ultimately the primitive's, which is always the
//     widest);
//   - schema-build restriction checks (a derived facet must narrow the
//     base's) always run in the base's space;
//   - only after the wider-space checks pass is the derived mapping's Parse
//     used to produce the application-facing value; a lexical the checks
//     accept but the narrow representation cannot hold is a mapping error on
//     the derived type, reported as such — never a false validity verdict.
//
// Comparison and facet capabilities are NOT backend methods; they live on the
// values a Mapping produces. A backend's values must implement the capabilities
// its types' applicable facets require ([Ordered] for bounded types,
// [DigitCounted] for digit facets, [Scaled] for precisionDecimal, …) —
// value/backendtest verifies this mechanically.
//
// # The facet pipeline
//
// Validation of a literal is a fixed stage sequence (ARCHITECTURE.md):
// whiteSpace → pattern facets → lexical mapping → value facets → assertions.
// [LexicalFacet] is a stage that checks the normalized lexical form;
// [ValueFacet] is a stage that checks the parsed value. Every stage failure is
// an *xsderr.Error carrying the facet's rule ID and the pipeline stage that
// rejected.
//
// # Codegen seam
//
// A backend MAY, in a later milestone, implement an Emitter (API frozen in M9;
// not yet declared here) to contribute specialized decode/encode Go source at
// codegen time — parsing straight from the reader's byte window into the target
// field, facet checks inlined, no boxed Value. A backend without an Emitter
// falls back to the runtime Mapping path for its types. Both paths must produce
// identical values and identical error rule IDs; codec's differential tests
// enforce that.
package value
