package strict

import (
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// New returns the spec-exact value.Backend for the primitive cohort so far:
// xs:decimal, xs:boolean, xs:string, xs:anyURI, xs:float, xs:double, xs:hexBinary,
// xs:base64Binary and xs:duration. Each type's value
// space is represented with full fidelity to the Datatypes spec (§3.3.3,
// §3.3.2, §3.3.1, §3.3.17, §3.3.4, §3.3.5, §3.3.15, §3.3.16, §3.3.6); a [value.Mapping.Parse]
// rejects any lexical
// outside the type's lexical space as an *xsderr.Error with rule
// "cvc-datatype-valid" (§4.1.4), never a false validity verdict.
//
// The value each mapping's Parse produces satisfies exactly the capability
// interfaces its applicable facets require (cos-applicable-facets), so a
// consumer discovers what a value can do by capability assertion, never a type
// switch (STYLE T2). The concrete value types are unexported:
//
//   - xs:decimal — [value.Ordered] (total order, §3.3.3.2), [value.Eq],
//     [value.DigitCounted] (totalDigits/fractionDigits) and [value.Canonical].
//     It is deliberately NOT [value.Scaled]: decimal collapses precision, so
//     1.0 and 1.00 are the same value.
//   - xs:boolean — [value.Eq], [value.Identical] and [value.Canonical]. It is
//     deliberately NOT [value.Ordered] (ordered=false, §3.3.2.3).
//   - xs:string — [value.Eq], [value.Lengthed] (character count) and
//     [value.Canonical]. It is deliberately NOT [value.Ordered]
//     (ordered=false, §3.3.1.3).
//   - xs:anyURI — [value.Eq], [value.Lengthed] (character count) and
//     [value.Canonical]. It is deliberately NOT [value.Ordered]
//     (ordered=false, §3.3.17.3).
//   - xs:float, xs:double — [value.Ordered] (PARTIAL order, §3.3.4.3/§3.3.5.3:
//     NaN is incomparable with every value, itself included), [value.Eq],
//     [value.Identical] and [value.Canonical]. Equality and identity genuinely
//     disagree here (NaN ≠ NaN but is identical to itself; +0 = −0 but is not
//     identical to −0), so both capabilities are implemented independently.
//     Deliberately NOT [value.Lengthed]/[value.DigitCounted]/[value.Scaled].
//   - xs:hexBinary, xs:base64Binary — [value.Eq], [value.Lengthed] (octet count,
//     §4.3.1.3 clause 1.2 — measured in octets of binary data, never lexical
//     characters) and [value.Canonical]. They are deliberately NOT [value.Ordered]
//     (ordered=false, §3.3.15/§3.3.16), so no bound facet applies to them.
//   - xs:duration — [value.Ordered] (PARTIAL order, §3.3.6.1: the
//     four-reference-dateTime algorithm, so e.g. P1M and P30D are incomparable),
//     [value.Eq] and [value.Identical] (both structural over the (·months·,
//     ·seconds·) tuple) and [value.Canonical]. Deliberately NOT [value.Lengthed]/
//     [value.DigitCounted]/[value.Scaled]/[value.TimezoneAware]: no
//     duration-applicable facet needs them (cos-applicable-facets §4.1.5).
func New() value.Backend { return backend{} }

// backend is the spec-exact primitive-cohort mapping. It carries no state: the
// mappings are pure functions, so a value receiver with an empty struct keeps
// New a plain constructor (STYLE T1) with nothing to misconfigure.
type backend struct{}

// Mapping dispatches on the builtin QName. It is a switch, not a map ranged
// into output (STYLE D2/D3), and covers only the cohort types; ok is false for
// every other name, including non-XML-Schema-namespace names.
func (backend) Mapping(typ xsd.QName) (value.Mapping, bool) {
	if typ.Space != xsd.XMLSchemaNS {
		return value.Mapping{}, false
	}
	switch typ.Local {
	case "decimal":
		return value.Mapping{Parse: parseDecimal, Canonical: canonicalDecimal}, true
	case "boolean":
		return value.Mapping{Parse: parseBoolean, Canonical: canonicalBoolean}, true
	case "string":
		return value.Mapping{Parse: parseString, Canonical: canonicalString}, true
	case "anyURI":
		return value.Mapping{Parse: parseAnyURI, Canonical: canonicalAnyURI}, true
	case "float":
		return value.Mapping{Parse: parseFloat, Canonical: canonicalFloat}, true
	case "double":
		return value.Mapping{Parse: parseDouble, Canonical: canonicalDouble}, true
	case "hexBinary":
		return value.Mapping{Parse: parseHexBinary, Canonical: canonicalHexBinary}, true
	case "base64Binary":
		return value.Mapping{Parse: parseBase64Binary, Canonical: canonicalBase64Binary}, true
	case "duration":
		return value.Mapping{Parse: parseDuration, Canonical: canonicalDuration}, true
	}
	return value.Mapping{}, false
}
