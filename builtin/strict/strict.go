package strict

import (
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// New returns the spec-exact value.Backend for the primitive cohort so far:
// xs:decimal, xs:precisionDecimal,
// xs:boolean, xs:string, xs:anyURI, xs:float, xs:double, xs:hexBinary,
// xs:base64Binary, xs:duration, xs:dateTime, the six remaining
// seven-property date/time siblings xs:time, xs:date, xs:gYearMonth, xs:gYear,
// xs:gMonthDay, xs:gDay and xs:gMonth, and xs:QName and xs:NOTATION. Each type's
// value
// space is represented with full fidelity to the Datatypes spec (§3.3.3,
// §3.3.2, §3.3.1, §3.3.17, §3.3.4, §3.3.5, §3.3.15, §3.3.16, §3.3.6, §3.3.7,
// §3.3.8–§3.3.14, §3.3.18, §3.3.19); a [value.Mapping.Parse]
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
//   - xs:precisionDecimal — the (·numericalValue·, ·scale·, ·sign·) triple
//     (xsd-precisionDecimal §3.1). [value.Ordered] (PARTIAL order, §3.1: −INF <
//     numeric < +INF, NaN incomparable with everything including itself),
//     [value.Eq] (scale-blind: 1.0 = 1.00), [value.Identical] (scale-SENSITIVE:
//     3, 3.0, 3.00 are distinct and +0 ≠ −0, yet NaN ≡ NaN — PRINCIPLES 18),
//     [value.Scaled] (·scale· kept verbatim), [value.DigitCounted] (totalDigits
//     counts the coefficient's digits, trailing zeros included; fractionDigits is
//     inert — not an applicable facet, §3.3) and [value.Canonical]. Deliberately
//     NOT [value.Lengthed]/[value.TimezoneAware]. The maxScale/minScale facets are
//     applicable per spec but not yet enforced (see the GAP marker in
//     precisiondecimal.go).
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
//   - xs:dateTime — the seven-property temporal model (§3.3.7.1/§D.2.1).
//     [value.Ordered] (PARTIAL order, §D.2.1: instants compare over
//     ·timeOnTimeline·, but a timezone-less value is Incomparable to a
//     timezone-aware one whose ±14h-imputed instants straddle it),
//     [value.Eq] and [value.Identical] (which genuinely diverge: a
//     timezone-shifted pair denoting the same instant is Eq but not Identical,
//     since Identical compares the stored ·timezoneOffset· exactly),
//     [value.Canonical] and [value.TimezoneAware] (the explicitTimezone facet,
//     §4.3.14, reads HasTimezone). Deliberately NOT [value.Lengthed]/
//     [value.DigitCounted]/[value.Scaled].
//   - xs:time, xs:date, xs:gYearMonth, xs:gYear, xs:gMonthDay, xs:gDay, xs:gMonth
//     — the same seven-property model as xs:dateTime (§3.3.8–§3.3.14/§D.2.1),
//     each a lexical "projection" that forces a per-type subset of the seven
//     properties ·absent· (e.g. time drops year/month/day, gYear keeps only
//     year). Same capability set and rationale as xs:dateTime: [value.Ordered]
//     (PARTIAL order over ·timeOnTimeline·, ordered=partial for all seven),
//     [value.Eq] and [value.Identical] (Eq/Identical diverge on a
//     timezone-shifted pair as for dateTime), [value.Canonical] and
//     [value.TimezoneAware]. Deliberately NOT [value.Lengthed]/
//     [value.DigitCounted]/[value.Scaled] — no applicable facet needs them
//     (cos-applicable-facets §4.1.5: no length family).
//   - xs:QName, xs:NOTATION — the {namespace name, local part} tuple
//     (§3.3.18/§3.3.19). Their lexical mapping is CONTEXT-DEPENDENT: Parse
//     resolves the prefix (empty prefix = default namespace) against the
//     [value.Context]'s in-scope namespace bindings, rejecting an unresolvable
//     prefix or malformed grammar as cvc-datatype-valid (§4.1.4). [value.Eq]
//     (value-space tuple equality; a QName never equals a same-tuple NOTATION,
//     they are distinct value types) and [value.Lengthed] (rune count of the
//     local part — length is applicable but deprecated, and §4.3.1.3 clause 1.3
//     makes any value length-facet-valid, so the count never gates validity).
//     Deliberately NOT [value.Ordered] (ordered=false, §3.3.18/§3.3.19) and,
//     uniquely in this cohort, their [value.Mapping.Canonical] is nil: the spec
//     defines NO canonical representation for either (their lexical forms vary
//     with context).
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
	case "precisionDecimal":
		return value.Mapping{Parse: parsePrecisionDecimal, Canonical: canonicalPrecisionDecimal}, true
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
	case "dateTime":
		return value.Mapping{Parse: parseDateTime, Canonical: canonicalDateTime}, true
	case "time":
		return value.Mapping{Parse: parseTime, Canonical: canonicalTime}, true
	case "date":
		return value.Mapping{Parse: parseDate, Canonical: canonicalDate}, true
	case "gYearMonth":
		return value.Mapping{Parse: parseGYearMonth, Canonical: canonicalGYearMonth}, true
	case "gYear":
		return value.Mapping{Parse: parseGYear, Canonical: canonicalGYear}, true
	case "gMonthDay":
		return value.Mapping{Parse: parseGMonthDay, Canonical: canonicalGMonthDay}, true
	case "gDay":
		return value.Mapping{Parse: parseGDay, Canonical: canonicalGDay}, true
	case "gMonth":
		return value.Mapping{Parse: parseGMonth, Canonical: canonicalGMonth}, true
	case "QName":
		// No canonical mapping: the spec defines none for QName (§3.3.18), so
		// Canonical is nil (value.Mapping documents nil Canonical as legitimate).
		return value.Mapping{Parse: parseQName, Canonical: nil}, true
	case "NOTATION":
		// No canonical mapping either (§3.3.19), so Canonical is nil.
		return value.Mapping{Parse: parseNOTATION, Canonical: nil}, true
	}
	return value.Mapping{}, false
}
