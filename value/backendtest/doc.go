// Package backendtest is the conformance kit for value backends: run it
// against any value.Backend — the two shipped ones or your own — to
// verify it implements the value contracts correctly.
//
// # Contract (core round-trip + capability checks implemented; derived-mapping discipline pending)
//
//	func Run(t *testing.T, b value.Backend, opts ...Option)
//
// Run drives, per builtin type the backend covers:
//
//   - lexical → value → canonical round-trips over spec-derived vectors
//     (valid lexicals map; invalid lexicals error; canonical output is
//     the spec's canonical mapping of the value);
//   - capability coverage: the value each mapping produces implements
//     every capability interface the type's applicable facets require
//     (value.Ordered for bounded types, value.DigitCounted for digit
//     facets, value.Lengthed for length-faceted spaces, value.Scaled for
//     scale facets, value.Eq for enumeration) — a missing capability or a
//     facet the kit cannot classify is a loud failure. The applicable-facet
//     lists are spec-derived (cos-applicable-facets); the facet→capability
//     classification is a fixed, spec-cited table in this package.
//
// Two contract areas named in the value package doc are not yet exercised,
// because they need inputs no current backend can supply:
//
//   - order and identity cases (the partial-order edges — Incomparable
//     across timezone-less/timezone-aware, NaN incomparable in order yet
//     identical to itself) await the date/time and float families;
//   - the widest-space discipline for DERIVED mappings awaits the
//     derived-type facet model (#33): the narrowReject slot and the
//     inherited-facet vectors are exercised once a derived mapping lands.
//
// Coverage checking is likewise partial: Run checks only the types it has
// vectors for. Full primitive coverage (every builtin primitive mapped or
// declared Absent via Option, absent ones supplied by value.Override) arrives
// with the first backend that maps them all.
//
// Vectors are spec-derived — generated from examples and function
// definitions in docs/specs/md (PRINCIPLES 26) — and identical for every
// backend: passing Run is what makes a third-party backend first-class.
package backendtest
