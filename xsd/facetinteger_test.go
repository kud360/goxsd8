package xsd

import (
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// huge, hugeLess and hugeMore are nonNegativeInteger {value}s past math.MaxInt:
// in the unbounded value space (Datatypes §3.4.20), and distinct from each other,
// so a reader that saturated them at math.MaxInt would compare them equal.
const (
	huge     = "99999999999999999999"
	hugeLess = "99999999999999999998"
	hugeMore = "100000000000000000000"
)

// TestIntegerFacetValuePastMaxInt pins countValue and scaleValue on {value}s
// past the host int (#1779). Every {value} in the accepting rows is in its value
// space; each such row was charged "is not a nonNegativeInteger" / "is not an
// integer" while the readers used strconv.Atoi. The rows headed "two distinct
// literals" compare two literals past math.MaxInt (or math.MinInt), which
// saturating both would compare equal, so they fail under saturation as well as
// under Atoi. msg, when set, is the charge's whole opening up to the last literal
// it names: it fails if a literal is printed as a parsed or saturated int, or if
// two operands trade places.
func TestIntegerFacetValuePastMaxInt(t *testing.T) {
	str := mustPrim(t, "string")
	dec := mustPrim(t, "decimal")
	pdec := mustPrim(t, "precisionDecimal")
	f := func(kind FacetKind, value string) Facet { return NewFacet(kind, []string{value}, false) }
	fixed := func(kind FacetKind, value string) Facet { return NewFacet(kind, []string{value}, true) }

	cases := []struct {
		name string
		prim *SimpleType
		base []Facet // nil: own restricts prim directly
		own  []Facet
		rule xsderr.Rule // "" means accepted
		msg  string
	}{
		// --- countValue: accepted in the value space ---
		{name: "length huge on string", prim: str, own: []Facet{f(FacetLength, huge)}},
		{name: "length huge restated", prim: str, base: []Facet{f(FacetLength, huge)}, own: []Facet{f(FacetLength, huge)}},
		{name: "length huge respelled +00", prim: str, base: []Facet{f(FacetLength, huge)}, own: []Facet{f(FacetLength, "+00"+huge)}},
		{name: "maxLength huge narrowed to 5", prim: str, base: []Facet{f(FacetMaxLength, huge)}, own: []Facet{f(FacetMaxLength, "5")}},
		{name: "minLength 5 raised to huge", prim: str, base: []Facet{f(FacetMinLength, "5")}, own: []Facet{f(FacetMinLength, huge)}},
		{name: "totalDigits huge narrowed to 5", prim: dec, base: []Facet{f(FacetTotalDigits, huge)}, own: []Facet{f(FacetTotalDigits, "5")}},
		{name: "fractionDigits huge narrowed to 5", prim: dec, base: []Facet{f(FacetFractionDigits, huge)}, own: []Facet{f(FacetFractionDigits, "5")}},
		{name: "maxLength narrowed between huge literals", prim: str, base: []Facet{f(FacetMaxLength, huge)}, own: []Facet{f(FacetMaxLength, hugeLess)}},
		{name: "length 5 within inherited maxLength huge", prim: str, base: []Facet{f(FacetMaxLength, huge)}, own: []Facet{f(FacetLength, "5")}},

		// --- countValue: a real widening, one operand past math.MaxInt ---
		{name: "maxLength 5 widened to +009", prim: str, base: []Facet{f(FacetMaxLength, "5")}, own: []Facet{f(FacetMaxLength, "+009")},
			rule: ruleMaxLengthValidRestriction,
			msg:  "simple type restriction's own maxLength {value} +009 is greater than the {base type definition}'s effective maxLength {value} 5"},
		{name: "maxLength 5 widened to huge", prim: str, base: []Facet{f(FacetMaxLength, "5")}, own: []Facet{f(FacetMaxLength, huge)},
			rule: ruleMaxLengthValidRestriction,
			msg:  "simple type restriction's own maxLength {value} " + huge + " is greater than the {base type definition}'s effective maxLength {value} 5"},
		{name: "fractionDigits huge above totalDigits 5", prim: dec, own: []Facet{f(FacetFractionDigits, huge), f(FacetTotalDigits, "5")},
			rule: ruleFractionDigitsLETotalDigits,
			msg:  "simple type {facets} has fractionDigits {value} " + huge + " greater than totalDigits {value} 5"},

		// --- countValue: two distinct literals past math.MaxInt ---
		{name: "maxLength widened between huge literals", prim: str, base: []Facet{f(FacetMaxLength, hugeLess)}, own: []Facet{f(FacetMaxLength, huge)},
			rule: ruleMaxLengthValidRestriction,
			msg:  "simple type restriction's own maxLength {value} " + huge + " is greater than the {base type definition}'s effective maxLength {value} " + hugeLess},
		{name: "minLength lowered between huge literals", prim: str, base: []Facet{f(FacetMinLength, huge)}, own: []Facet{f(FacetMinLength, hugeLess)},
			rule: ruleMinLengthValidRestriction,
			msg:  "simple type restriction's own minLength {value} " + hugeLess + " is less than the {base type definition}'s effective minLength {value} " + huge},
		{name: "length changed between huge literals", prim: str, base: []Facet{f(FacetLength, huge)}, own: []Facet{f(FacetLength, hugeLess)},
			rule: ruleLengthValidRestriction,
			msg:  "simple type restriction's own length {value} " + hugeLess + " does not equal the {base type definition}'s effective length {value} " + huge},
		{name: "totalDigits widened between huge literals", prim: dec, base: []Facet{f(FacetTotalDigits, huge)}, own: []Facet{f(FacetTotalDigits, hugeMore)},
			rule: ruleTotalDigitsValidRestriction,
			msg:  "simple type restriction's own totalDigits {value} " + hugeMore + " is greater than the {base type definition}'s effective totalDigits {value} " + huge},
		{name: "minLength above maxLength, both huge", prim: str, own: []Facet{f(FacetMinLength, huge), f(FacetMaxLength, hugeLess)},
			rule: ruleMinLengthLEMaxLength,
			msg:  "simple type {facets} has minLength {value} " + huge + " greater than maxLength {value} " + hugeLess},
		{name: "minLength above length, both huge", prim: str, own: []Facet{f(FacetLength, hugeLess), f(FacetMinLength, huge)},
			rule: ruleLengthMinLengthMaxLength,
			msg:  "simple type {facets} has minLength {value} " + huge + " greater than length {value} " + hugeLess},
		{name: "length above maxLength, both huge", prim: str, base: []Facet{f(FacetLength, huge)}, own: []Facet{f(FacetMaxLength, hugeLess)},
			rule: ruleLengthMinLengthMaxLength,
			msg:  "simple type {facets} has length {value} " + huge + " greater than maxLength {value} " + hugeLess},
		// length-minLength-maxLength clause 2.2: the base's maxLength differs
		// from the one specified beside length, so no step holds that {value}
		// without length.
		{name: "maxLength beside length differs from base's, both huge", prim: str, base: []Facet{f(FacetMaxLength, huge)}, own: []Facet{f(FacetLength, "5"), f(FacetMaxLength, hugeLess)},
			rule: ruleLengthMinLengthMaxLength,
			msg:  "simple type {facets} has length alongside maxLength {value} " + hugeLess + ","},

		// --- scaleValue: accepted in the value space ---
		{name: "maxScale huge narrowed to 5", prim: pdec, base: []Facet{f(FacetMaxScale, huge)}, own: []Facet{f(FacetMaxScale, "5")}},
		{name: "minScale -huge raised to -5", prim: pdec, base: []Facet{f(FacetMinScale, "-"+huge)}, own: []Facet{f(FacetMinScale, "-5")}},
		{name: "fixed maxScale huge respelled +0", prim: pdec, base: []Facet{fixed(FacetMaxScale, huge)}, own: []Facet{fixed(FacetMaxScale, "+0"+huge)}},

		// --- scaleValue: one operand past math.MaxInt ---
		{name: "maxScale 5 widened to huge", prim: pdec, base: []Facet{f(FacetMaxScale, "5")}, own: []Facet{f(FacetMaxScale, huge)},
			rule: ruleMaxScaleValidRestriction,
			msg:  "simple type restriction's own maxScale {value} " + huge + " relaxes the {base type definition}'s effective maxScale {value} 5"},

		// --- scaleValue: two distinct literals past math.MaxInt / math.MinInt ---
		{name: "maxScale widened between huge literals", prim: pdec, base: []Facet{f(FacetMaxScale, hugeLess)}, own: []Facet{f(FacetMaxScale, huge)},
			rule: ruleMaxScaleValidRestriction,
			msg:  "simple type restriction's own maxScale {value} " + huge + " relaxes the {base type definition}'s effective maxScale {value} " + hugeLess},
		{name: "minScale lowered between -huge literals", prim: pdec, base: []Facet{f(FacetMinScale, "-"+hugeLess)}, own: []Facet{f(FacetMinScale, "-"+huge)},
			rule: ruleMinScaleValidRestriction,
			msg:  "simple type restriction's own minScale {value} -" + huge + " relaxes the {base type definition}'s effective minScale {value} -" + hugeLess},
		{name: "fixed maxScale changed between huge literals", prim: pdec, base: []Facet{fixed(FacetMaxScale, huge)}, own: []Facet{fixed(FacetMaxScale, hugeLess)},
			rule: ruleMaxScaleFixed,
			msg:  "simple type restriction sets maxScale {value} " + hugeLess + " but the {base type definition}'s effective maxScale is {fixed} at " + huge},
		{name: "fixed minScale changed between -huge literals", prim: pdec, base: []Facet{fixed(FacetMinScale, "-"+huge)}, own: []Facet{fixed(FacetMinScale, "-"+hugeLess)},
			rule: ruleMinScaleFixed,
			msg:  "simple type restriction sets minScale {value} -" + hugeLess + " but the {base type definition}'s effective minScale is {fixed} at -" + huge},
		{name: "minScale above maxScale, both huge", prim: pdec, own: []Facet{f(FacetMinScale, huge), f(FacetMaxScale, hugeLess)},
			rule: ruleMinScaleLEMaxScale,
			msg:  "simple type {facets} has minScale {value} " + huge + " greater than maxScale {value} " + hugeLess},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			base := c.prim
			if c.base != nil {
				var err error
				base, err = newCheckedSimpleType(xsderr.Loc{}, QName{Space: "urn:test", Local: "base"},
					RestrictionDerivation{}, c.prim, c.base, nil)
				if err != nil {
					t.Fatalf("build base: %v", err)
				}
			}
			_, err := newCheckedSimpleType(xsderr.Loc{}, QName{Space: "urn:test", Local: "derived"},
				RestrictionDerivation{}, base, c.own, nil)
			if c.rule == "" {
				if err != nil {
					t.Fatalf("restriction rejected: %v", err)
				}
				return
			}
			wantRule(t, err, c.rule)
			var xe *xsderr.Error
			if !errors.As(err, &xe) || !strings.HasPrefix(xe.Msg, c.msg) {
				t.Errorf("message = %q, want it to open %q", err, c.msg)
			}
		})
	}
}
