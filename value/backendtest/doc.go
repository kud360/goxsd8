// Package backendtest is the conformance kit for value backends: run it
// against any value.Backend — the two shipped ones or your own — to
// verify it implements the value contracts correctly.
//
// # Contract (implemented in M3)
//
//	func Run(t *testing.T, b value.Backend, opts ...Option)
//
// Run drives, per builtin primitive the backend covers:
//
//   - lexical → value → canonical round-trips over spec-derived vectors
//     (valid lexicals map; invalid lexicals error; canonical output is
//     the spec's canonical mapping of the value);
//   - order and identity cases, including the partial-order edges
//     (Incomparable across timezone-less/timezone-aware, NaN
//     incomparable in order yet identical to itself);
//   - capability coverage: the value types produced implement every
//     capability interface the primitive's applicable facets require
//     (value.Ordered for bounded types, value.DigitCounted for digit
//     facets, value.Scaled for precisionDecimal, value.Lengthed for
//     length-faceted spaces, …);
//   - primitive coverage: every builtin primitive is either mapped or
//     explicitly declared absent via Option (absent primitives are then
//     expected to be supplied by composition with value.Override);
//   - widest-space discipline for DERIVED mappings: for every derived
//     builtin the backend chooses to map, vectors verify that inherited
//     enumeration/bound facets still evaluate through the governing
//     ancestor's wider space (a boundary lexical the base space orders
//     correctly must not be misjudged by the narrow representation),
//     and that a wide-valid lexical the narrow representation cannot
//     hold surfaces as a mapping error, never as a validity verdict.
//
// Vectors are spec-derived — generated from examples and function
// definitions in docs/specs/md (PRINCIPLES 26) — and identical for every
// backend: passing Run is what makes a third-party backend first-class.
package backendtest
