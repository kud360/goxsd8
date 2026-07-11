package strict

import (
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// New returns the spec-exact value.Backend for the first primitive cohort:
// xs:decimal, xs:boolean and xs:string. Each type's value space is represented
// with full fidelity to the Datatypes spec (§3.3.3, §3.3.2, §3.3.1); a
// [value.Mapping.Parse] rejects any lexical outside the type's lexical space as
// an *xsderr.Error with rule "cvc-datatype-valid" (§4.1.4), never a false
// validity verdict.
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
func New() value.Backend { return backend{} }

// backend is the spec-exact primitive-cohort mapping. It carries no state: the
// mappings are pure functions, so a value receiver with an empty struct keeps
// New a plain constructor (STYLE T1) with nothing to misconfigure.
type backend struct{}

// Mapping dispatches on the builtin QName. It is a switch, not a map ranged
// into output (STYLE D2/D3), and covers only the three cohort types; ok is
// false for every other name, including non-XML-Schema-namespace names.
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
	}
	return value.Mapping{}, false
}
