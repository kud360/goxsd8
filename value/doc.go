// Package value defines the value-space contracts: how typed values are
// represented, compared, and produced from lexical forms — without fixing
// any particular Go representation.
//
// # Values are open
//
//	type Value = any
//
// Deliberately not a sealed interface (PRINCIPLES 2): users bring their
// own backends and value types. What a value can do is discovered through
// small capability interfaces (STYLE T2), never a type switch over
// concrete types:
//
//	type Ordering int  // Less, Equal, Greater, Incomparable
//
//	type Eq interface{ Eq(other Value) bool }
//	type Ordered interface { Eq; Cmp(other Value) Ordering }
//	    XSD value spaces are PARTIALLY ordered: cross-space comparisons
//	    and timezone-less vs timezone-aware date/times are Incomparable.
//
//	type Identical interface{ Identical(other Value) bool }
//	    Identity is distinct from order equality (PRINCIPLES 18):
//	    enumeration matching needs NaN identical to NaN even though the
//	    order comparator treats NaN as incomparable. Types without it
//	    fall back to order equality.
//
//	type Lengthed interface{ Len() int }                       // length facets
//	type DigitCounted interface{ TotalDigits() int; FractionDigits() int }
//	type Scaled interface{ Scale() (int, bool) }
//	    precisionDecimal keeps its lexical scale as part of the value
//	    (3, 3.0, 3.00 are distinct, numerically equal values); specials
//	    (NaN, ±INF) report no scale.
//	type TimezoneAware interface{ HasTimezone() bool }         // explicitTimezone
//	type Canonical interface{ Canonical() string }             // value → canonical lexical
//
// # Backends
//
// A backend supplies the lexical↔value mappings for the builtin
// PRIMITIVES only — derived builtins are data (see package builtin):
//
//	type Context interface{ LookupNamespace(prefix string) (string, bool) }
//	    QName and NOTATION need in-scope namespace bindings at parse time
//	    (PRINCIPLES 19).
//
//	type Mapping struct {
//	    Parse     func(lexical string, ctx Context) (Value, error)
//	    Canonical func(v Value) (string, error)
//	}
//
//	type Backend interface{ Mapping(primitive xsd.QName) (Mapping, bool) }
//
//	func Override(base, partial Backend) Backend
//	    Per-type composition: partial's mappings where defined, base
//	    otherwise — back only xs:decimal with a money type and keep the
//	    rest.
//
// Comparison and facet capabilities are NOT backend methods; they live on
// the values a Mapping produces. A backend's values must implement the
// capabilities its types' applicable facets require (Ordered for bounded
// types, DigitCounted for digit facets, Scaled for precisionDecimal, …) —
// value/backendtest verifies this mechanically.
//
// # The facet pipeline
//
// Validation of a literal is a fixed stage sequence (ARCHITECTURE.md):
// whiteSpace → pattern facets → lexical mapping → value facets →
// assertions. The stage contracts:
//
//	type LexicalFacet interface{ CheckLexical(normalized string) error }
//	type ValueFacet interface{ CheckValue(v Value) error }
//
// Every stage failure is an *xsderr.Error carrying the facet's rule ID
// and the pipeline stage that rejected.
//
// # Codegen seam
//
//	type Emitter interface { ... }   // API frozen in M9
//
// A backend MAY implement Emitter to contribute specialized decode/encode
// Go source at codegen time — parsing straight from the reader's byte
// window into the target field, facet checks inlined, no boxed Value. A
// backend without an Emitter falls back to the runtime Mapping path for
// its types. Both paths must produce identical values and identical error
// rule IDs; codec's differential tests enforce that.
package value
