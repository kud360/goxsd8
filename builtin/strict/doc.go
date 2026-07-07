// Package strict is the spec-exact value backend: value spaces
// represented with full fidelity to the Datatypes spec, at the cost of
// heavier representations.
//
// # Representations (implemented across M3+)
//
//   - decimal, integer family — arbitrary precision (math/big), with
//     preserved digit counts for totalDigits/fractionDigits.
//   - precisionDecimal — (coefficient, scale, sign) kept VERBATIM: 3,
//     3.0, 3.00 are distinct values that compare numerically equal;
//     TotalDigits counts trailing zeros; NaN and ±INF supported with
//     exact-case lexicals (INF, +INF, -INF, NaN — no float-style
//     aliases). Order treats NaN as incomparable; Identical makes NaN
//     equal to itself (PRINCIPLES 18).
//   - date/time family — the seven-property model (year, month, day,
//     hour, minute, second, timezoneOffset), NOT time.Time: proleptic
//     calendar math, optional timezone, partial order across
//     timezone-less and timezone-aware values.
//   - duration — six-component model with the four-reference-dateTime
//     partial order.
//   - float/double — XSD-exact semantics (signed zeros, INF, NaN
//     identity for enumeration).
//   - string family, anyURI, QName/NOTATION (context-resolved),
//     hexBinary/base64Binary (octet values).
//
// Every mapping's Parse/Canonical follows the corresponding hfn function
// definition in the local Datatypes spec, cited at the implementation
// site.
//
// The package passes value/backendtest with full primitive coverage and
// implements value.Emitter (M9+) for generated fast paths.
package strict
