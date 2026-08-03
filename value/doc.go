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
// The list and union varieties recurse rather than adding stages. A list runs
// this whole pipeline against the ITEM TYPE per whitespace-delimited item — the
// item type's own facets included, which is what makes a list of a derived type
// (a list of xs:byte, whose mapping is xs:decimal's) reject an out-of-range item
// — and then its own facets over the resulting sequence. A union has no
// whiteSpace facet at all: it
// hands the RAW literal to its member types in order, and the first one that is
// itself datatype-valid supplies both the value and the whiteSpace normalization
// its own pattern facet then matches against — so a union's value is always some
// member's value, never a wrapper of its own.
//
// A facet's OWN {value} goes through the same whiteSpace normalization before it
// is parsed, once at construction: a facet's {value} property is "a value from
// the value space of the {base type definition}" (§4.3.7.1 f-mai-value, §4.3.5.1),
// and reaching that value space runs the base's lexical mapping, whose first
// stage is its whiteSpace normalization (key-vv §3.1.3, key-nv §3.1.4). So
// `<maxInclusive value=" 9 "/>` on a collapse-normalized base denotes exactly
// what the untrailed spelling denotes, on both the validation and the
// restriction-checking path.
//
// # Construction-time facet checks
//
//	func CheckFacetRestriction(b Backend, t *xsd.SimpleType) error
//
// [CheckFacetRestriction] is the once-per-type counterpart of that pipeline: it
// charges the value-space Schema Component Constraints relating a simple type's
// own bound and enumeration facets to its {base type definition}
// (maxInclusive/maxExclusive/minInclusive/minExclusive valid restriction
// §4.3.7.4–§4.3.10.4, enumeration valid restriction §4.3.5.5) — the half of
// cos-st-restricts that package xsd, a pure leaf with no value spaces, cannot
// reach. Call it through builtin.CheckSimpleTypeRestriction, which charges facet
// APPLICABILITY first and then delegates here; a type built through the xsd
// constructors alone gets neither.
//
// # Value-constraint validity and comparison (the xsd.ValueSpace seam)
//
//	func NewValueSpace(b Backend) xsd.ValueSpace
//
// [NewValueSpace] is what lets package xsd — a pure leaf that cannot import this
// one — decide the Structures constraints that reach into a value space. Two
// COMPARE two Value Constraints' {value}s: au-props-correct (§3.5.6) clause 3
// under the identity relation (§2.2.1), loc-testSubP (§3.4.6.4) clauses 4.2 and
// 5.2.2 under the equal-or-identical union (§2.2.2). It maps each side's {lexical
// form} through the governing mapping of the type that constrains it and compares
// the values with the [Identical]/[Eq] capabilities they carry.
//
// The other two VALIDATE one Value Constraint against one type: a-props-correct
// (§3.2.6.1) clause 2 and au-props-correct clause 2, both charging Simple Default
// Valid (§3.2.6.2), which is Datatype Valid (§4.1.4) and so [ValidateLexical]'s
// job. Being one-sided it needs no shared mapping, so it decides the list and
// union varieties the comparisons refuse.
//
// It answers "undecided" — never a verdict — for everything it cannot decide: an
// ungoverned type (the ·special· xs:anySimpleType and xs:anyAtomicType included,
// for which Datatype Valid is unconditionally true), an unmappable lexical, the
// context-dependent QName and NOTATION spaces, a construction-stage failure in
// the type's own facets, the list and union varieties on the comparisons, and any
// pair whose two types resolve to DIFFERENT governing mappings (the widest-space
// rule above is what makes a general/specific pair on one base chain comparable
// at all, and what makes anything else incommensurable). Undecided always accepts
// on the xsd side, so this seam can only narrow what a schema set admits.
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
