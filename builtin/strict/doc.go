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
//   - duration — the (·months·, ·seconds·) two-property model (§3.3.6.1),
//     an integer month count and an arbitrary-precision decimal second
//     count sharing one sign, with the four-reference-dateTime partial
//     order.
//   - float/double — XSD-exact semantics (signed zeros, INF, NaN
//     identity for enumeration).
//   - string family, anyURI, QName/NOTATION (context-resolved),
//     hexBinary/base64Binary (octet values).
//
// Every mapping's Parse/Canonical follows the corresponding hfn function
// definition in the local Datatypes spec, cited at the implementation
// site.
//
// # Current coverage
//
// [New] returns a backend covering the primitive cohort so far — xs:decimal,
// xs:boolean, xs:string, xs:anyURI, xs:float, xs:double, xs:hexBinary,
// xs:base64Binary, xs:duration and xs:dateTime — with spec-exact
// parse, canonical and comparison. The remaining representations above
// (precisionDecimal, the rest of the date/time family, the rest of the string
// family, QName/NOTATION) and the
// value.Emitter fast path remain future milestones. The cohort is certified
// by value/backendtest.Run: each type's value carries exactly the capability
// interfaces its applicable facets require (cos-applicable-facets), documented
// per type on [New].
//
// # Facet pipeline
//
// The backend-generic facet pipeline (whiteSpace normalization, pattern, lexical
// mapping, value facets; Datatypes §4.3.6 and §4.3.1–4.3.12) is owned by the
// shared value package — value.ValidateLexical — not this backend, since none of
// it is strict-specific (issue #87). strict supplies only the per-type lexical
// mappings that pipeline drives: decimal, boolean, float, double and anyURI fix
// whiteSpace=collapse and string is whiteSpace=preserve, carried as each
// primitive's own whiteSpace Constraining Facet (§3.16.7.4) so
// value.ValidateLexical resolves the mode off EffectiveFacets (§3.16.6.4
// overlay), never from a side table.
package strict
